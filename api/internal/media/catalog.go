package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Catalogue lookups (EP-17). Open Library needs no key; TMDB is on when the
// operator sets TMDB_API_KEY; video games come from Wikipedia (EP-149),
// which needs none either. Results are cached, requests are paced, and
// covers are copied to Axon so pages never load images from elsewhere.

// Retention: searches for a week; TMDB data for at most six months, as its
// terms require.
const (
	lookupTTL  = 7 * 24 * time.Hour
	tmdbMaxAge = 180 * 24 * time.Hour
	maxBody    = 2 << 20
)

// ErrUnavailable means the catalogue could not be reached or is not set up.
var ErrUnavailable = errors.New("catalogue unavailable")

// Candidate is a catalogue entry a person can add.
type Candidate struct {
	Source   string   `json:"source"`
	SourceID string   `json:"source_id"`
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Year     *int     `json:"year"`
	Creators string   `json:"creators"`
	Genres   []string `json:"genres"`
	Overview string   `json:"overview"`
	Total    *int     `json:"total"`
	CoverKey string   `json:"-"`
	HasCover bool     `json:"has_cover"`
}

// Endpoints are the providers' base URLs.
type Endpoints struct {
	OpenLibrary, OpenLibraryCovers, TMDB, TMDBImages, Wikipedia string
	WikimediaImages                                             []string // where Wikipedia's pictures may come from
}

// DefaultEndpoints are the real providers; tests point them elsewhere.
var DefaultEndpoints = Endpoints{
	OpenLibrary: "https://openlibrary.org", OpenLibraryCovers: "https://covers.openlibrary.org",
	TMDB: "https://api.themoviedb.org/3", TMDBImages: "https://image.tmdb.org/t/p/w342",
	Wikipedia: "https://en.wikipedia.org", WikimediaImages: []string{"https://upload.wikimedia.org/", "https://thumb.wikimedia.org/"},
}

// Catalogs holds the providers and their cache.
type Catalogs struct {
	db        *pgxpool.Pool
	client    *http.Client
	userAgent string
	tmdbKey   string
	Endpoints

	mu   sync.Mutex
	last map[string]time.Time
}

// NewCatalogs returns the providers. contact identifies Axon to Open Library.
func NewCatalogs(pool *pgxpool.Pool, version, contact, tmdbKey string) *Catalogs {
	ua := "Axon/" + version
	if contact != "" {
		ua += " (" + contact + ")"
	}
	return &Catalogs{
		db: pool, client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: httpsOnly}, userAgent: ua, tmdbKey: strings.TrimSpace(tmdbKey),
		Endpoints: DefaultEndpoints, last: map[string]time.Time{},
	}
}

// httpsOnly follows at most three redirects, never off HTTPS (Open Library
// covers redirect to archive.org).
func httpsOnly(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 || (req.URL.Scheme != "https" && via[0].URL.Scheme == "https") {
		return errors.New("redirect refused")
	}
	return nil
}

// TMDBEnabled reports whether films and series can be looked up.
func (c *Catalogs) TMDBEnabled() bool { return c.tmdbKey != "" }

// pace keeps requests to a provider apart: Open Library asks identified
// clients for at most three a second.
func (c *Catalogs) pace(ctx context.Context, source string) error {
	gap := 350 * time.Millisecond
	switch source {
	case "tmdb":
		gap = 100 * time.Millisecond
	case "wikipedia":
		gap = 200 * time.Millisecond
	}
	c.mu.Lock()
	wait := time.Until(c.last[source].Add(gap))
	if wait < 0 {
		wait = 0
	}
	c.last[source] = time.Now().Add(wait)
	c.mu.Unlock()
	select {
	case <-time.After(wait):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Catalogs) get(ctx context.Context, source, rawURL string, out any) error {
	if err := c.pace(ctx, source); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if source == "tmdb" && strings.HasPrefix(c.tmdbKey, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+c.tmdbKey) // a v4 read access token
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err) //nolint:errorlint // the cause is for logs; callers check ErrUnavailable
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s answered %d", ErrUnavailable, source, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out)
}

func (c *Catalogs) tmdbURL(path string, q url.Values) string {
	if q == nil {
		q = url.Values{}
	}
	if !strings.HasPrefix(c.tmdbKey, "eyJ") {
		q.Set("api_key", c.tmdbKey)
	}
	return c.TMDB + path + "?" + q.Encode()
}

// Search looks a title up: books in Open Library, films and series in TMDB.
func (c *Catalogs) Search(ctx context.Context, kind, query string) ([]Candidate, error) {
	query = strings.Join(strings.Fields(query), " ")
	if query == "" || len([]rune(query)) > 200 {
		return nil, ValidationError{"Search for 1 to 200 characters."}
	}
	source := "openlibrary"
	switch kind {
	case "book":
	case "film", "series":
		if !c.TMDBEnabled() {
			return nil, fmt.Errorf("%w: films and series need a TMDB key on this server", ErrUnavailable)
		}
		source = "tmdb"
	case "game":
		source = "wikipedia"
	default:
		return nil, ValidationError{"Kind is film, series, book, or game."}
	}
	key := kind + ":" + strings.ToLower(query)
	var cached []Candidate
	var raw []byte
	err := c.db.QueryRow(ctx, `SELECT result FROM media_lookups WHERE source = $1 AND query = $2 AND fetched_at > now() - $3::interval`,
		source, key, fmt.Sprintf("%d seconds", int(lookupTTL.Seconds()))).Scan(&raw)
	if err == nil && json.Unmarshal(raw, &cached) == nil {
		return cached, nil
	}
	var found []Candidate
	switch source {
	case "openlibrary":
		found, err = c.searchOpenLibrary(ctx, query)
	case "wikipedia":
		found, err = c.searchWikipedia(ctx, query)
	default:
		found, err = c.searchTMDB(ctx, kind, query)
	}
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(found)
	_, err = c.db.Exec(ctx, `INSERT INTO media_lookups (source, query, result) VALUES ($1, $2, $3)
		ON CONFLICT (source, query) DO UPDATE SET result = EXCLUDED.result, fetched_at = now()`, source, key, b)
	return found, err
}

func (c *Catalogs) searchOpenLibrary(ctx context.Context, query string) ([]Candidate, error) {
	var res struct {
		Docs []struct {
			Key      string   `json:"key"`
			Title    string   `json:"title"`
			Authors  []string `json:"author_name"`
			Year     int      `json:"first_publish_year"`
			Cover    int      `json:"cover_i"`
			Pages    int      `json:"number_of_pages_median"`
			Subjects []string `json:"subject"`
		} `json:"docs"`
	}
	q := url.Values{"q": {query}, "limit": {"10"}, "fields": {"key,title,author_name,first_publish_year,cover_i,number_of_pages_median,subject"}}
	if err := c.get(ctx, "openlibrary", c.OpenLibrary+"/search.json?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	out := []Candidate{}
	for _, d := range res.Docs {
		if d.Title == "" || !strings.HasPrefix(d.Key, "/works/") {
			continue
		}
		cand := Candidate{Source: "openlibrary", SourceID: strings.TrimPrefix(d.Key, "/works/"), Kind: "book", Title: truncate(d.Title, 500),
			Creators: truncate(strings.Join(first(d.Authors, 3), ", "), 500), Genres: cleanGenres(first(d.Subjects, 5))}
		if d.Year >= 1000 && d.Year <= 3000 {
			cand.Year = &d.Year
		}
		if d.Pages > 0 && d.Pages <= 100000 {
			cand.Total = &d.Pages
		}
		if d.Cover > 0 {
			cand.CoverKey, cand.HasCover = strconv.Itoa(d.Cover), true // by cover ID, which Open Library does not rate-limit
		}
		out = append(out, cand)
	}
	return out, nil
}

type tmdbResult struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Name     string `json:"name"`
	Release  string `json:"release_date"`
	FirstAir string `json:"first_air_date"`
	Overview string `json:"overview"`
	Poster   string `json:"poster_path"`
}

func (r tmdbResult) candidate(kind string) Candidate {
	cand := Candidate{Source: "tmdb", SourceID: strconv.Itoa(r.ID), Kind: kind, Title: truncate(r.Title+r.Name, 500),
		Overview: truncate(r.Overview, 4000), Genres: []string{}}
	if y, err := strconv.Atoi(firstN(r.Release+r.FirstAir, 4)); err == nil && y >= 1000 && y <= 3000 {
		cand.Year = &y
	}
	if strings.HasPrefix(r.Poster, "/") && len(r.Poster) < 100 {
		cand.CoverKey, cand.HasCover = r.Poster, true
	}
	return cand
}

func (c *Catalogs) searchTMDB(ctx context.Context, kind, query string) ([]Candidate, error) {
	path := "/search/movie"
	if kind == "series" {
		path = "/search/tv"
	}
	var res struct {
		Results []tmdbResult `json:"results"`
	}
	if err := c.get(ctx, "tmdb", c.tmdbURL(path, url.Values{"query": {query}, "include_adult": {"false"}}), &res); err != nil {
		return nil, err
	}
	out := []Candidate{}
	for _, r := range first(res.Results, 10) {
		if r.ID > 0 && r.Title+r.Name != "" {
			out = append(out, r.candidate(kind))
		}
	}
	return out, nil
}

// Details returns one catalogue entry with what a search leaves out: genres,
// creators, and length.
func (c *Catalogs) Details(ctx context.Context, source, kind, id string) (Candidate, error) {
	switch {
	case source == "openlibrary" && kind == "book" && validID(id, "OLW"):
		return c.openLibraryWork(ctx, id)
	case source == "tmdb" && (kind == "film" || kind == "series") && validNumber(id):
		if !c.TMDBEnabled() {
			return Candidate{}, fmt.Errorf("%w: films and series need a TMDB key on this server", ErrUnavailable)
		}
		return c.tmdbDetails(ctx, kind, id)
	case source == "wikipedia" && kind == "game" && validPage(id):
		return c.wikipediaGame(ctx, id)
	}
	return Candidate{}, ValidationError{"That catalogue entry does not exist."}
}

func (c *Catalogs) openLibraryWork(ctx context.Context, id string) (Candidate, error) {
	// The search index has authors, year, pages, and cover in one request.
	var res struct {
		Docs []struct {
			Title    string   `json:"title"`
			Authors  []string `json:"author_name"`
			Year     int      `json:"first_publish_year"`
			Cover    int      `json:"cover_i"`
			Pages    int      `json:"number_of_pages_median"`
			Subjects []string `json:"subject"`
		} `json:"docs"`
	}
	q := url.Values{"q": {"key:/works/" + id}, "limit": {"1"}, "fields": {"title,author_name,first_publish_year,cover_i,number_of_pages_median,subject"}}
	if err := c.get(ctx, "openlibrary", c.OpenLibrary+"/search.json?"+q.Encode(), &res); err != nil {
		return Candidate{}, err
	}
	if len(res.Docs) == 0 || res.Docs[0].Title == "" {
		return Candidate{}, ValidationError{"That catalogue entry does not exist."}
	}
	d := res.Docs[0]
	cand := Candidate{Source: "openlibrary", SourceID: id, Kind: "book", Title: truncate(d.Title, 500),
		Creators: truncate(strings.Join(first(d.Authors, 3), ", "), 500), Genres: cleanGenres(first(d.Subjects, 5))}
	if d.Year >= 1000 && d.Year <= 3000 {
		cand.Year = &d.Year
	}
	if d.Pages > 0 && d.Pages <= 100000 {
		cand.Total = &d.Pages
	}
	if d.Cover > 0 {
		cand.CoverKey, cand.HasCover = strconv.Itoa(d.Cover), true
	}
	return cand, nil
}

func (c *Catalogs) tmdbDetails(ctx context.Context, kind, id string) (Candidate, error) {
	path, q := "/movie/"+id, url.Values{"append_to_response": {"credits"}}
	if kind == "series" {
		path, q = "/tv/"+id, nil
	}
	var res struct {
		tmdbResult
		Genres []struct {
			Name string `json:"name"`
		} `json:"genres"`
		Runtime   int `json:"runtime"`
		Episodes  int `json:"number_of_episodes"`
		CreatedBy []struct {
			Name string `json:"name"`
		} `json:"created_by"`
		Credits struct {
			Crew []struct {
				Name string `json:"name"`
				Job  string `json:"job"`
			} `json:"crew"`
		} `json:"credits"`
	}
	if err := c.get(ctx, "tmdb", c.tmdbURL(path, q), &res); err != nil {
		return Candidate{}, err
	}
	if res.ID == 0 {
		return Candidate{}, ValidationError{"That catalogue entry does not exist."}
	}
	cand := res.candidate(kind)
	var genres, people []string
	for _, g := range res.Genres {
		genres = append(genres, g.Name)
	}
	cand.Genres = cleanGenres(first(genres, 5))
	for _, p := range res.CreatedBy {
		people = append(people, p.Name)
	}
	for _, p := range res.Credits.Crew {
		if p.Job == "Director" {
			people = append(people, p.Name)
		}
	}
	cand.Creators = truncate(strings.Join(first(people, 3), ", "), 500)
	if kind == "series" && res.Episodes > 0 && res.Episodes <= 100000 {
		cand.Total = &res.Episodes // progress counts episodes
	}
	return cand, nil
}

// Cover copies a candidate's cover into Axon once and returns its id. A
// missing or unusable cover is not an error: the item just has none.
func (c *Catalogs) Cover(ctx context.Context, cand Candidate) *uuid.UUID {
	if !cand.HasCover || cand.CoverKey == "" {
		return nil
	}
	var id uuid.UUID
	err := c.db.QueryRow(ctx, `SELECT id FROM media_covers WHERE source = $1 AND source_key = $2`, cand.Source, cand.CoverKey).Scan(&id)
	if err == nil {
		return &id
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	var src string
	switch cand.Source {
	case "openlibrary":
		if !validNumber(cand.CoverKey) {
			return nil
		}
		src = c.OpenLibraryCovers + "/b/id/" + cand.CoverKey + "-M.jpg?default=false"
	case "tmdb":
		src = c.TMDBImages + cand.CoverKey
	case "wikipedia":
		src = cand.CoverKey // a whole address on Wikimedia's image servers, checked by coverKey
	default:
		return nil
	}
	data, ctype, err := c.fetchImage(ctx, cand.Source, src)
	if err != nil {
		return nil
	}
	id = uuid.Must(uuid.NewV7())
	if err := c.db.QueryRow(ctx, `INSERT INTO media_covers (id, source, source_key, content_type, data) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source, source_key) DO UPDATE SET fetched_at = media_covers.fetched_at RETURNING id`,
		id, cand.Source, cand.CoverKey, ctype, data).Scan(&id); err != nil {
		return nil
	}
	return &id
}

func (c *Catalogs) fetchImage(ctx context.Context, source, src string) ([]byte, string, error) {
	if err := c.pace(ctx, source); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("cover answered %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(data) > maxBody {
		return nil, "", errors.New("cover too large")
	}
	// Trust the bytes, not the header.
	ctype := http.DetectContentType(data)
	switch ctype {
	case "image/jpeg", "image/png", "image/webp":
		return data, ctype, nil
	}
	return nil, "", fmt.Errorf("cover is %s", ctype)
}

// CoverImage returns a stored cover.
func CoverImage(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) ([]byte, string, error) {
	var data []byte
	var ctype string
	err := pool.QueryRow(ctx, `SELECT data, content_type FROM media_covers WHERE id = $1`, id).Scan(&data, &ctype)
	return data, ctype, err
}

// Prune drops expired searches, and TMDB covers and descriptions older than
// TMDB allows; the person's own fields (title, rating, review) stay.
func Prune(ctx context.Context, pool *pgxpool.Pool) error {
	age := fmt.Sprintf("%d seconds", int(tmdbMaxAge.Seconds()))
	for _, stmt := range []string{
		`DELETE FROM media_lookups WHERE fetched_at < now() - interval '7 days'`,
		`DELETE FROM media_covers WHERE source = 'tmdb' AND fetched_at < now() - $1::interval`,
		`UPDATE media_items SET overview = '', fetched_at = NULL WHERE source = 'tmdb' AND fetched_at < now() - $1::interval`,
	} {
		var err error
		if strings.Contains(stmt, "$1") {
			_, err = pool.Exec(ctx, stmt, age)
		} else {
			_, err = pool.Exec(ctx, stmt)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func first[T any](s []T, n int) []T {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func firstN(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func validNumber(s string) bool {
	if s == "" || len(s) > 12 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// validID accepts Open Library ids such as OL45883W.
func validID(s, kind string) bool {
	return strings.HasPrefix(s, "OL") && strings.HasSuffix(s, kind[2:]) && validNumber(s[2:len(s)-1])
}
