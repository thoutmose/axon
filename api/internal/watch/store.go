package watch

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/knowledge"
)

// Type is the graph entity type of a news item.
const Type = "feed_item"

// Actor marks what the tech watch does on its own, so activity charts can
// leave it out.
const Actor graph.Actor = "watch"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "watch_items", Export: exportItems})
}

// Errors a person can act on.
var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already followed")
)

// InvalidError says what to correct.
type InvalidError struct{ Msg string }

func (e InvalidError) Error() string { return e.Msg }

func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// --- Topics ---

// ListTopics returns the person's topics.
func ListTopics(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Topic, error) {
	rows, err := q.Query(ctx, `SELECT id::text, name, keywords FROM watch_topics WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Topic])
	if out == nil {
		out = []Topic{}
	}
	return out, err
}

// SaveTopic creates (id empty) or changes a topic, then labels the last
// month's news again with it.
func SaveTopic(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id, name string, keywords []string) (Topic, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return Topic{}, InvalidError{"A topic has a name of 1 to 80 characters."}
	}
	var kws []string
	for _, k := range keywords {
		if k = strings.TrimSpace(k); k != "" && utf8.RuneCountInString(k) <= 80 && !slices.Contains(kws, k) {
			kws = append(kws, k)
		}
	}
	if len(kws) > 50 {
		return Topic{}, InvalidError{"A topic has up to 50 keywords."}
	}
	if kws == nil {
		kws = []string{}
	}
	t := Topic{ID: id, Name: name, Keywords: kws}
	var err error
	if id == "" {
		t.ID = uuid.Must(uuid.NewV7()).String()
		_, err = tx.Exec(ctx, `INSERT INTO watch_topics (id, user_id, name, keywords) VALUES ($1, $2, $3, $4)`, t.ID, userID, name, kws)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE watch_topics SET name = $3, keywords = $4 WHERE id = $1 AND user_id = $2`, id, userID, name, kws)
		if err == nil && tag.RowsAffected() == 0 {
			return Topic{}, ErrNotFound
		}
	}
	if isUnique(err) {
		return Topic{}, InvalidError{"You already have a topic with that name."}
	}
	if err != nil {
		return Topic{}, err
	}
	return t, Relabel(ctx, tx, userID, 30*24*time.Hour)
}

// DeleteTopic removes a topic; news keep their other labels.
func DeleteTopic(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id string) error {
	var name string
	err := tx.QueryRow(ctx, `DELETE FROM watch_topics WHERE id = $1 AND user_id = $2 RETURNING name`, id, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE watch_items SET topic_ids = array_remove(topic_ids, $2::uuid), labels = array_remove(labels, $3)
		WHERE user_id = $1 AND $2::uuid = ANY (topic_ids)`, userID, id, strings.ToLower(name))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE watch_sources SET topic_ids = array_remove(topic_ids, $2::uuid) WHERE user_id = $1`, userID, id)
	return err
}

// Relabel applies the current topics to recent news again, from the terms
// stored with each item.
func Relabel(ctx context.Context, tx pgx.Tx, userID uuid.UUID, within time.Duration) error {
	topics, err := ListTopics(ctx, tx, userID)
	if err != nil {
		return err
	}
	names := make([]string, len(topics))
	for i, t := range topics {
		names[i] = strings.ToLower(t.Name)
	}
	rows, err := tx.Query(ctx, `SELECT id, title, lang, terms, labels FROM watch_items WHERE user_id = $1 AND published_at > $2 LIMIT 5000`,
		userID, time.Now().Add(-within))
	if err != nil {
		return err
	}
	type it struct {
		id            uuid.UUID
		title, lang   string
		terms, labels []string
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (it, error) {
		var x it
		err := r.Scan(&x.id, &x.title, &x.lang, &x.terms, &x.labels)
		return x, err
	})
	if err != nil {
		return err
	}
	for _, x := range list {
		title, _ := knowledge.TokensIn(x.title, x.lang)
		body := map[string]bool{}
		for _, t := range x.terms {
			body[t] = true
		}
		var ids, labels []string
		var score float64
		for _, t := range topics {
			if s := topicScore(t, title, body, x.lang); s > 0 {
				ids, labels, score = append(ids, t.ID), append(labels, strings.ToLower(t.Name)), score+s
			}
		}
		// Keep labels that are not topic names (key phrases, categories, the person's own).
		for _, l := range x.labels {
			if !slices.Contains(names, l) && !slices.Contains(labels, l) {
				labels = append(labels, l)
			}
		}
		if ids == nil {
			ids = []string{}
		}
		if _, err := tx.Exec(ctx, `UPDATE watch_items SET topic_ids = $2::uuid[], labels = $3, score = $4 WHERE id = $1`,
			x.id, ids, uniq(labels, 16), score); err != nil {
			return err
		}
	}
	return nil
}

// --- Sources ---

// Source is a followed feed, with how it is doing.
type Source struct {
	ID            uuid.UUID  `json:"id"`
	URL           string     `json:"url"`
	SiteURL       string     `json:"site_url"`
	Title         string     `json:"title"`
	TopicIDs      []string   `json:"topic_ids"`
	Schedule      string     `json:"schedule"`
	Enabled       bool       `json:"enabled"`
	FullText      bool       `json:"full_text"`
	Origin        string     `json:"origin"`
	NextFetchAt   time.Time  `json:"next_fetch_at"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	LastOKAt      *time.Time `json:"last_ok_at"`
	LastError     string     `json:"last_error"`
	Failures      int        `json:"failures"`
	CreatedAt     time.Time  `json:"created_at"`
	Items         int        `json:"items"`      // stored news from it
	ItemsWeek     int        `json:"items_week"` // in the last 7 days
	Merged        int        `json:"merged"`     // stories it also carried, stored once from another source
	Health        string     `json:"health"`     // ok, failing, waiting, off
	etag, mod     string
}

const sourceColumns = `s.id, s.url, s.site_url, s.title, s.topic_ids::text[], s.schedule, s.enabled, s.full_text, s.origin, s.next_fetch_at,
	s.last_fetched_at, s.last_ok_at, s.last_error, s.failures, s.created_at,
	(SELECT count(*) FROM watch_items i WHERE i.source_id = s.id),
	(SELECT count(*) FROM watch_items i WHERE i.source_id = s.id AND i.fetched_at > now() - interval '7 days'),
	(SELECT count(*) FROM watch_items i WHERE i.user_id = s.user_id AND s.title = ANY (i.also_in)),
	s.etag, s.last_modified`

func scanSource(r pgx.Row) (Source, error) {
	var s Source
	err := r.Scan(&s.ID, &s.URL, &s.SiteURL, &s.Title, &s.TopicIDs, &s.Schedule, &s.Enabled, &s.FullText, &s.Origin, &s.NextFetchAt,
		&s.LastFetchedAt, &s.LastOKAt, &s.LastError, &s.Failures, &s.CreatedAt, &s.Items, &s.ItemsWeek, &s.Merged, &s.etag, &s.mod)
	switch {
	case !s.Enabled:
		s.Health = "off"
	case s.LastFetchedAt == nil:
		s.Health = "waiting"
	case s.Failures > 0:
		s.Health = "failing"
	default:
		s.Health = "ok"
	}
	return s, err
}

// ListSources returns the person's sources with their health.
func ListSources(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Source, error) {
	rows, err := q.Query(ctx, `SELECT `+sourceColumns+` FROM watch_sources s WHERE s.user_id = $1 ORDER BY s.title`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Source, error) { return scanSource(r) })
	if out == nil {
		out = []Source{}
	}
	return out, err
}

// GetSource returns one source.
func GetSource(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Source, error) {
	s, err := scanSource(q.QueryRow(ctx, `SELECT `+sourceColumns+` FROM watch_sources s WHERE s.id = $1 AND s.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

// SourceInput creates or changes a source.
type SourceInput struct {
	URL      string   `json:"url"` // to create: a feed, a site, a Medium author, a Substack…
	Title    *string  `json:"title"`
	TopicIDs []string `json:"topic_ids"`
	Schedule *string  `json:"schedule"`
	Enabled  *bool    `json:"enabled"`
	FullText *bool    `json:"full_text"`
	Origin   string   `json:"-"`
}

func (in SourceInput) check(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if in.Schedule != nil {
		if _, err := ParseSchedule(*in.Schedule); err != nil {
			return InvalidError{"Schedule: " + err.Error() + "."}
		}
	}
	if in.Title != nil && (strings.TrimSpace(*in.Title) == "" || utf8.RuneCountInString(*in.Title) > 200) {
		return InvalidError{"A source has a name of 1 to 200 characters."}
	}
	if len(in.TopicIDs) > 0 {
		var n int
		if err := q.QueryRow(ctx, `SELECT count(*) FROM watch_topics WHERE user_id = $1 AND id::text = ANY ($2)`, userID, in.TopicIDs).Scan(&n); err != nil {
			return err
		}
		if n != len(in.TopicIDs) {
			return InvalidError{"One of the topics does not exist."}
		}
	}
	return nil
}

// CreateSource stores a found feed and its current news. The next fetch
// follows the schedule, in the person's time zone.
func CreateSource(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in SourceInput, found Found, now time.Time) (Source, Stats, error) {
	if err := in.check(ctx, tx, userID); err != nil {
		return Source{}, Stats{}, err
	}
	title := cut(firstOf(found.Feed.Title, hostOf(found.URL)), 200)
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
	}
	sched := DefaultSchedule
	if in.Schedule != nil {
		sched = *in.Schedule
	}
	topics := in.TopicIDs
	if topics == nil {
		topics = []string{}
	}
	origin := firstOf(in.Origin, "manual")
	next, err := nextRun(ctx, tx, userID, sched, now)
	if err != nil {
		return Source{}, Stats{}, err
	}
	id := uuid.Must(uuid.NewV7())
	_, err = tx.Exec(ctx, `INSERT INTO watch_sources (id, user_id, url, site_url, title, topic_ids, schedule, enabled, full_text, origin, next_fetch_at, last_fetched_at, last_ok_at)
		VALUES ($1, $2, $3, $4, $5, $6::uuid[], $7, $8, $9, $10, $11, $12, $12)`,
		id, userID, cut(found.URL, 2000), cut(found.Feed.SiteURL, 2000), title, topics, sched, in.Enabled == nil || *in.Enabled, in.FullText != nil && *in.FullText, origin, next, now)
	if isUnique(err) {
		return Source{}, Stats{}, ErrExists
	}
	if err != nil {
		return Source{}, Stats{}, err
	}
	src, err := GetSource(ctx, tx, userID, id)
	if err != nil {
		return src, Stats{}, err
	}
	stats, err := Ingest(ctx, tx, userID, src, found.Feed.Items, now)
	if err != nil {
		return src, stats, err
	}
	src, err = GetSource(ctx, tx, userID, id)
	return src, stats, err
}

func hostOf(raw string) string {
	if i := strings.Index(raw, "://"); i >= 0 {
		raw = raw[i+3:]
	}
	h, _, _ := strings.Cut(raw, "/")
	return strings.TrimPrefix(h, "www.")
}

// nextRun is the schedule's next time in the person's time zone.
func nextRun(ctx context.Context, q db.Querier, userID uuid.UUID, sched string, now time.Time) (time.Time, error) {
	s, err := ParseSchedule(sched)
	if err != nil {
		s, _ = ParseSchedule(DefaultSchedule)
	}
	loc, err := userLocation(ctx, q, userID)
	if err != nil {
		return time.Time{}, err
	}
	return s.Next(now, loc).UTC(), nil
}

// userLocation is the person's time zone, UTC if it is unknown.
func userLocation(ctx context.Context, q db.Querier, userID uuid.UUID) (*time.Location, error) {
	var tz string
	if err := q.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC, nil
	}
	return loc, nil
}

// UpdateSource changes a source's name, topics, schedule, or switches.
func UpdateSource(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, in SourceInput, now time.Time) (Source, error) {
	cur, err := GetSource(ctx, tx, userID, id)
	if err != nil {
		return cur, err
	}
	if err := in.check(ctx, tx, userID); err != nil {
		return cur, err
	}
	if in.Title != nil {
		cur.Title = strings.TrimSpace(*in.Title)
	}
	if in.TopicIDs != nil {
		cur.TopicIDs = in.TopicIDs
	}
	next := cur.NextFetchAt
	if in.Schedule != nil && *in.Schedule != cur.Schedule {
		cur.Schedule = *in.Schedule
		if next, err = nextRun(ctx, tx, userID, cur.Schedule, now); err != nil {
			return cur, err
		}
	}
	if in.Enabled != nil {
		cur.Enabled = *in.Enabled
	}
	if in.FullText != nil {
		cur.FullText = *in.FullText
	}
	_, err = tx.Exec(ctx, `UPDATE watch_sources SET title = $3, topic_ids = $4::uuid[], schedule = $5, enabled = $6, full_text = $7, next_fetch_at = $8
		WHERE id = $1 AND user_id = $2`, id, userID, cur.Title, cur.TopicIDs, cur.Schedule, cur.Enabled, cur.FullText, next)
	if err != nil {
		return cur, err
	}
	return GetSource(ctx, tx, userID, id)
}

// DeleteSource stops following a source. Its news stay unless withItems.
func DeleteSource(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, withItems bool) error {
	if withItems {
		if _, err := tx.Exec(ctx, `DELETE FROM entities e USING watch_items i WHERE e.id = i.id AND i.source_id = $1 AND i.user_id = $2
			AND i.status <> 'saved' AND i.note_id IS NULL`, id, userID); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM watch_sources WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// --- Ingest ---

// Stats counts what one fetch brought.
type Stats struct {
	New     int `json:"new"`
	Merged  int `json:"merged"`  // the same story, already stored from another source or link
	Skipped int `json:"skipped"` // already seen from this source, or without a link
}

func guidHash(e Entry) []byte {
	h := sha256.Sum256([]byte(firstOf(e.GUID, e.URL, e.Title)))
	return h[:16]
}

type recent struct {
	id          uuid.UUID
	titleTerms  []string
	terms       []string
	simhash     uint64
	alsoIn      []string
	sourceTitle string
}

// sameStory says whether a new item repeats a stored one: the same words in
// a long text (SimHash within 3 bits), or nearly the same words in a short
// one or in the title (Jaccard).
func sameStory(r recent, a Analysis, titleTerms []string) bool {
	if len(a.Terms) >= 60 && len(r.terms) >= 60 && Distance(r.simhash, a.SimHash) <= NearDuplicate {
		return true
	}
	if len(titleTerms) >= 4 && Jaccard(r.titleTerms, titleTerms) >= 0.85 {
		return true
	}
	return len(a.Terms) >= 8 && Jaccard(r.terms, a.Terms) >= 0.8
}

// Ingest stores a feed's new entries for a source, each story once: an
// entry already seen from this source is skipped; one whose link, or whose
// text, matches a stored item only adds this source to that item.
func Ingest(ctx context.Context, tx pgx.Tx, userID uuid.UUID, src Source, entries []Entry, now time.Time) (Stats, error) {
	var st Stats
	// One ingest per person at a time, so two sources cannot store the same story twice.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('watch:' || $1::text, 0))`, userID); err != nil {
		return st, err
	}
	topics, err := ListTopics(ctx, tx, userID)
	if err != nil {
		return st, err
	}
	var mine []Topic
	for _, t := range topics {
		if len(src.TopicIDs) == 0 || slices.Contains(src.TopicIDs, t.ID) {
			mine = append(mine, t)
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, title, lang, terms, simhash, also_in, source_title FROM watch_items
		WHERE user_id = $1 AND published_at > $2 ORDER BY published_at DESC LIMIT 5000`, userID, now.Add(-10*24*time.Hour))
	if err != nil {
		return st, err
	}
	stored, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (recent, error) {
		var x recent
		var title, lang string
		var sh int64
		err := r.Scan(&x.id, &title, &lang, &x.terms, &sh, &x.alsoIn, &x.sourceTitle)
		x.simhash = uint64(sh) //nolint:gosec // stored as the same 64 bits
		x.titleTerms, _ = knowledge.TokensIn(title, lang)
		return x, err
	})
	if err != nil {
		return st, err
	}
	merge := func(id uuid.UUID) error {
		_, err := tx.Exec(ctx, `UPDATE watch_items SET also_in = array_append(also_in, $2) WHERE id = $1 AND source_title <> $2 AND NOT ($2 = ANY (also_in))`, id, src.Title)
		st.Merged++
		return err
	}
	for _, e := range entries {
		e.URL = strings.TrimSpace(e.URL)
		tag, err := tx.Exec(ctx, `INSERT INTO watch_seen (source_id, guid_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`, src.ID, guidHash(e))
		if err != nil {
			return st, err
		}
		if tag.RowsAffected() == 0 || e.URL == "" || e.Title == "" && e.Summary == "" && e.Content == "" {
			st.Skipped++
			continue
		}
		var existing uuid.UUID
		err = tx.QueryRow(ctx, `SELECT id FROM watch_items WHERE user_id = $1 AND url_hash = $2`, userID, URLHash(e.URL)).Scan(&existing)
		if err == nil {
			if err := merge(existing); err != nil {
				return st, err
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return st, err
		}
		if e.Title == "" {
			e.Title = firstSentences(PlainText(firstOf(e.Summary, e.Content)), 120)
		}
		a := Analyse(e, mine)
		titleTerms, _ := knowledge.TokensIn(e.Title, a.Lang)
		dup := uuid.Nil
		for _, r := range stored {
			if sameStory(r, a, titleTerms) {
				dup = r.id
				break
			}
		}
		if dup != uuid.Nil {
			if err := merge(dup); err != nil {
				return st, err
			}
			continue
		}
		published := e.Published
		if published.IsZero() || published.After(now.Add(time.Hour)) {
			published = now
		}
		id, err := graph.CreateEntity(ctx, tx, userID, Actor, Type, cut(e.Title, 500), cut(a.Summary, 1000))
		if err != nil {
			return st, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO watch_items (id, user_id, source_id, source_title, guid, url, url_hash, title, author, published_at, fetched_at,
			summary, content, image, audio, lang, words, simhash, terms, keywords, labels, categories, topic_ids, score, outlier)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23::uuid[], $24, $25)`,
			id, userID, src.ID, src.Title, cut(e.GUID, 2000), cut(e.URL, 2000), URLHash(e.URL), cut(e.Title, 500), cut(e.Author, 200), published, now,
			cut(a.Summary, 5000), cut(a.Markdown, 200000), cut(e.Image, 2000), cut(e.Audio, 2000), a.Lang, a.Words, int64(a.SimHash), //nolint:gosec // the same 64 bits
			nonNil(a.Terms), nonNil(a.Keywords), nonNil(a.Labels), nonNil(cleanAll(e.Categories)), nonNil(a.Topics), a.Score, a.Outlier)
		if err != nil {
			return st, fmt.Errorf("store %q: %w", e.URL, err)
		}
		stored = append(stored, recent{id: id, titleTerms: titleTerms, terms: a.Terms, simhash: a.SimHash, sourceTitle: src.Title})
		st.New++
	}
	return st, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLSTATE 23505")
}
