package watch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/l10n"
)

// Handlers serves the tech watch.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
// Reaching outside Axon (following, fetching, importing from FreshRSS) is
// refused to the shared demo.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/watch/topics", h.topics)
	r.Post("/v1/watch/topics", h.saveTopic)
	r.Put("/v1/watch/topics/{id}", h.saveTopic)
	r.Delete("/v1/watch/topics/{id}", h.deleteTopic)
	r.Get("/v1/watch/sources", h.sources)
	r.With(auth.NotDemo).Post("/v1/watch/discover", h.discover)
	r.With(auth.NotDemo).Post("/v1/watch/sources", h.createSource)
	r.Patch("/v1/watch/sources/{id}", h.updateSource)
	r.Delete("/v1/watch/sources/{id}", h.deleteSource)
	r.With(auth.NotDemo).Post("/v1/watch/sources/{id}/fetch", h.fetchNow)
	r.With(auth.NotDemo).Post("/v1/watch/import/freshrss", h.importFreshRSS)
	r.Get("/v1/watch/sources.opml", h.opml)
	r.Get("/v1/watch/items", h.items)
	r.Post("/v1/watch/items/read", h.readAll)
	r.Get("/v1/watch/items/{id}", h.item)
	r.Patch("/v1/watch/items/{id}", h.updateItem)
	r.Post("/v1/watch/items/{id}/keep", h.keep)
	r.With(auth.NotDemo).Post("/v1/watch/items/{id}/transcript", h.transcript)
	r.Get("/v1/watch/stats", h.stats)
	r.Get("/v1/watch/graph", h.graph)
	r.Get("/v1/watch/export", h.export)
	r.Get("/v1/watch/podcast", h.podcast)
}

// SessionRoutes manage share links: they make news public, so they take a
// signed-in session, not an API token, and are audited.
func (h *Handlers) SessionRoutes(r chi.Router) {
	r.Get("/v1/watch/shares", h.shares)
	r.Post("/v1/watch/shares", h.createShare)
	r.Delete("/v1/watch/shares/{id}", h.deleteShare)
}

// PublicRoutes serve share links, authenticated by their token, with a
// budget of their own per address.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.With(auth.RateLimit(2*time.Second, 30, l10n.Mark("Too many requests for shared news. Wait a minute and try again."))).
		Get("/v1/shared/watch/{token}", h.sharedPage)
}

// UploadRoutes take files (CSV, OPML) larger than JSON bodies.
func (h *Handlers) UploadRoutes(r chi.Router) {
	r.With(auth.NotDemo).Post("/v1/watch/import", h.importFile)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var inv ValidationError
	switch {
	case errors.As(err, &inv):
		httpx.Problem(w, http.StatusUnprocessableEntity, inv.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	case errors.Is(err, ErrExists):
		httpx.Problem(w, http.StatusConflict, "You already follow this feed.")
	default:
		h.log.Error("tech watch request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func idParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	}
	return id, err == nil
}

func (h *Handlers) tx(ctx context.Context, f func(pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, h.db, f)
}

// --- topics ---

func (h *Handlers) topics(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	list, err := ListTopics(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (h *Handlers) saveTopic(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Name     string   `json:"name"`
		Keywords []string `json:"keywords"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	id := chi.URLParam(r, "id")
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			httpx.Problem(w, http.StatusNotFound, "Not found.")
			return
		}
	}
	var t Topic
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		t, err = SaveTopic(r.Context(), tx, who.UserID, id, b.Name, b.Keywords)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, map[bool]int{true: http.StatusCreated, false: http.StatusOK}[id == ""], t)
}

func (h *Handlers) deleteTopic(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.tx(r.Context(), func(tx pgx.Tx) error { return DeleteTopic(r.Context(), tx, who.UserID, id.String()) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- sources ---

func (h *Handlers) sources(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	list, err := ListSources(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

// Preview is a found feed before following it.
type Preview struct {
	URL     string   `json:"url"`
	Title   string   `json:"title"`
	SiteURL string   `json:"site_url"`
	Recent  []string `json:"recent"` // titles of its latest entries
	Entries int      `json:"entries"`
}

func (h *Handlers) found(w http.ResponseWriter, r *http.Request, input string) (Found, bool) {
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	f, err := Discover(ctx, input)
	if err != nil {
		httpx.Problem(w, http.StatusUnprocessableEntity, "No feed found: "+err.Error()+".")
		return f, false
	}
	return f, true
}

func (h *Handlers) discover(w http.ResponseWriter, r *http.Request) {
	var b struct {
		URL string `json:"url"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	f, ok := h.found(w, r, b.URL)
	if !ok {
		return
	}
	p := Preview{URL: f.URL, Title: f.Feed.Title, SiteURL: f.Feed.SiteURL, Entries: len(f.Feed.Items), Recent: []string{}}
	for _, e := range f.Feed.Items[:min(5, len(f.Feed.Items))] {
		p.Recent = append(p.Recent, e.Title)
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handlers) createSource(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var in SourceInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var n int
	if err := h.db.QueryRow(r.Context(), `SELECT count(*) FROM watch_sources WHERE user_id = $1`, who.UserID).Scan(&n); err != nil {
		h.fail(w, err)
		return
	}
	if n >= MaxImport {
		httpx.Problem(w, http.StatusUnprocessableEntity, "You follow the most sources Axon allows.")
		return
	}
	f, ok := h.found(w, r, in.URL)
	if !ok {
		return
	}
	if in.FullText != nil && *in.FullText {
		fillFullText(r.Context(), h.db, uuid.Nil, f.Feed.Items)
	}
	var src Source
	var st Stats
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		src, st, err = CreateSource(r.Context(), tx, who.UserID, in, f, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"source": src, "stats": st})
}

func (h *Handlers) updateSource(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in SourceInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var src Source
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		src, err = UpdateSource(r.Context(), tx, who.UserID, id, in, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, src)
}

func (h *Handlers) deleteSource(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	withItems := r.URL.Query().Get("items") == "delete"
	if err := h.tx(r.Context(), func(tx pgx.Tx) error { return DeleteSource(r.Context(), tx, who.UserID, id, withItems) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) fetchNow(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	st, err := FetchSource(r.Context(), h.db, who.UserID, id, h.now())
	if errors.Is(err, ErrNotFound) {
		h.fail(w, err)
		return
	}
	src, gerr := GetSource(r.Context(), h.db, who.UserID, id)
	if gerr != nil {
		h.fail(w, gerr)
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusOK, map[string]any{"source": src, "stats": st, "error": src.LastError})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"source": src, "stats": st, "error": ""})
}

// --- imports ---

func (h *Handlers) importFile(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The file could not be read.")
		return
	}
	var plan []Planned
	format := r.URL.Query().Get("format")
	switch format {
	case "csv":
		plan, err = ParseCSV(bytes.NewReader(body))
	case "opml":
		plan, err = ParseOPML(bytes.NewReader(body))
	default:
		httpx.Problem(w, http.StatusBadRequest, "format is csv or opml.")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.runImport(w, r, who.UserID, plan, format)
}

func (h *Handlers) importFreshRSS(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		URL      string `json:"url"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()
	plan, err := FreshRSS(ctx, b.URL, b.User, b.Password)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.runImport(w, r, who.UserID, plan, "freshrss")
}

func (h *Handlers) runImport(w http.ResponseWriter, r *http.Request, userID uuid.UUID, plan []Planned, origin string) {
	var res ImportResult
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		res, err = Import(r.Context(), tx, userID, plan, origin, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handlers) opml(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	sources, err := ListSources(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	topics, err := ListTopics(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="axon-tech-watch.opml"`)
	_, _ = w.Write(OPML(sources, topics, "Tech watch — Axon"))
}

// --- items ---

// ParseFilter reads the filter from a query string.
func ParseFilter(r *http.Request) (Filter, error) { return FilterFrom(r.URL.Query()) }

// FilterFrom reads the filter from query values.
func FilterFrom(q url.Values) (Filter, error) {
	f := Filter{Status: q.Get("status"), TopicID: q.Get("topic"), Label: q.Get("label"), Query: q.Get("q"), Outliers: q.Get("outliers") == "1", Kind: q.Get("kind")}
	if f.Kind != "" && !slices.Contains(Kinds, f.Kind) {
		return f, ValidationError{"kind is article, video, or podcast."}
	}
	if f.Status != "" && f.Status != "all" && !strings.Contains(strings.Join(Statuses, " "), f.Status) {
		return f, ValidationError{"status is unread, read, saved, archived, or all."}
	}
	if s := q.Get("source"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return f, ValidationError{"source is an id."}
		}
		f.SourceID = &id
	}
	if f.TopicID != "" {
		if _, err := uuid.Parse(f.TopicID); err != nil {
			return f, ValidationError{"topic is an id."}
		}
	}
	for name, dst := range map[string]**time.Time{"from": &f.From, "to": &f.To, "before": &f.Before} {
		if v := q.Get(name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				if t, err = time.Parse(time.DateOnly, v); err != nil {
					return f, ValidationError{name + " is a date (2026-10-14) or a time (RFC 3339)."}
				}
			}
			*dst = &t
		}
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			return f, ValidationError{"limit is between 1 and 200."}
		}
		f.Limit = n
	}
	return f, nil
}

func (h *Handlers) items(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := ParseFilter(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	list, err := ListItems(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (h *Handlers) item(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	it, err := GetItem(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	topics, err := ListTopics(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	names := map[string]string{}
	for _, t := range topics {
		names[t.ID] = t.Name
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"item": it, "note": FeedNote(it, names)})
}

func (h *Handlers) updateItem(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var b struct {
		Status *string  `json:"status"`
		Labels []string `json:"labels"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := UpdateItem(r.Context(), h.db, who.UserID, id, b.Status, b.Labels); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) readAll(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := ParseFilter(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	n, err := MarkAll(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"marked": n})
}

func (h *Handlers) keep(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var note uuid.UUID
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		note, err = Keep(r.Context(), tx, who.UserID, id)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"note_id": note})
}

// transcript reads a video's words now, rather than at the next fetch (EP-123).
func (h *Handlers) transcript(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	it, err := Transcribe(r.Context(), h.db, who.UserID, id, WhisperFromEnv())
	switch {
	case errors.Is(err, ErrNoTranscript):
		httpx.Problem(w, http.StatusUnprocessableEntity, "This video has no captions, and no Whisper server is set up to transcribe it.")
	case err != nil && !errors.As(err, new(ValidationError)) && !errors.Is(err, ErrNotFound):
		h.log.Warn("transcript failed", "item", id, "err", err)
		httpx.Problem(w, http.StatusBadGateway, "YouTube or the Whisper server did not answer. Try again later.")
	case err != nil:
		h.fail(w, err)
	default:
		httpx.JSON(w, http.StatusOK, it)
	}
}

// --- monitoring ---

// Overview is the state of the tech watch at a glance.
type Overview struct {
	Sources struct {
		Total   int `json:"total"`
		OK      int `json:"ok"`
		Failing int `json:"failing"`
		Waiting int `json:"waiting"`
		Off     int `json:"off"`
	} `json:"sources"`
	Items struct {
		Total    int `json:"total"`
		Unread   int `json:"unread"`
		Saved    int `json:"saved"`
		Today    int `json:"today"`
		Week     int `json:"week"`
		Outliers int `json:"outliers"`
		Merged   int `json:"merged"` // stories carried by more than one source, stored once
	} `json:"items"`
	Daily  []DayCount   `json:"daily"` // news stored per day, last 30 days
	Labels []LabelCount `json:"labels"`
	Next   *time.Time   `json:"next_fetch_at"`
}

// DayCount is news stored on a day.
type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// LabelCount is how many recent items carry a label.
type LabelCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func (h *Handlers) stats(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	o, err := Summarize(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// Summarize computes the overview.
func Summarize(ctx context.Context, q *pgxpool.Pool, userID uuid.UUID, now time.Time) (Overview, error) {
	var o Overview
	sources, err := ListSources(ctx, q, userID)
	if err != nil {
		return o, err
	}
	for _, s := range sources {
		o.Sources.Total++
		switch s.Health {
		case "ok":
			o.Sources.OK++
		case "failing":
			o.Sources.Failing++
		case "waiting":
			o.Sources.Waiting++
		default:
			o.Sources.Off++
		}
		if s.Enabled && (o.Next == nil || s.NextFetchAt.Before(*o.Next)) {
			t := s.NextFetchAt
			o.Next = &t
		}
	}
	err = q.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE status = 'unread' AND outlier = ''), count(*) FILTER (WHERE status = 'saved'),
		count(*) FILTER (WHERE fetched_at > $2::timestamptz - interval '1 day'), count(*) FILTER (WHERE fetched_at > $2::timestamptz - interval '7 days'),
		count(*) FILTER (WHERE outlier <> ''), count(*) FILTER (WHERE cardinality(also_in) > 0)
		FROM watch_items WHERE user_id = $1`, userID, now).Scan(&o.Items.Total, &o.Items.Unread, &o.Items.Saved, &o.Items.Today, &o.Items.Week, &o.Items.Outliers, &o.Items.Merged)
	if err != nil {
		return o, err
	}
	rows, err := q.Query(ctx, `SELECT to_char(d, 'YYYY-MM-DD'), (SELECT count(*) FROM watch_items i WHERE i.user_id = $1 AND i.fetched_at >= d AND i.fetched_at < d + interval '1 day')
		FROM generate_series(date_trunc('day', $2::timestamptz) - interval '29 days', date_trunc('day', $2::timestamptz), interval '1 day') d ORDER BY d`, userID, now)
	if err != nil {
		return o, err
	}
	if o.Daily, err = pgx.CollectRows(rows, pgx.RowToStructByPos[DayCount]); err != nil {
		return o, err
	}
	rows, err = q.Query(ctx, `SELECT l, count(*) FROM watch_items, unnest(labels) l WHERE user_id = $1 AND fetched_at > $2::timestamptz - interval '30 days'
		GROUP BY l ORDER BY count(*) DESC, l LIMIT 20`, userID, now)
	if err != nil {
		return o, err
	}
	o.Labels, err = pgx.CollectRows(rows, pgx.RowToStructByPos[LabelCount])
	if o.Labels == nil {
		o.Labels = []LabelCount{}
	}
	return o, err
}

// graph serves the knowledge graph of the news: the last 30 days unless
// from and to say otherwise.
func (h *Handlers) graph(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := ParseFilter(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	gf := GraphFilter{To: h.now(), TopicID: f.TopicID, SourceID: f.SourceID, Fringe: r.URL.Query().Get("fringe") == "1"}
	if f.To != nil {
		gf.To = *f.To
	}
	gf.From = gf.To.AddDate(0, 0, -30)
	if f.From != nil {
		gf.From = *f.From
	}
	if !gf.From.Before(gf.To) || gf.To.Sub(gf.From) > 366*24*time.Hour {
		h.fail(w, ValidationError{"from is before to, at most a year apart."})
		return
	}
	if v := r.URL.Query().Get("min"); v != "" {
		if gf.MinNPMI, err = strconv.ParseFloat(v, 64); err != nil || gf.MinNPMI <= 0 || gf.MinNPMI > 1 {
			h.fail(w, ValidationError{"min is above 0 and at most 1."})
			return
		}
	}
	g, err := BuildGraph(r.Context(), h.db, who.UserID, gf)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, g)
}

// podcast serves the episode script of the period, shaped by the person's
// podcast settings unless the query says otherwise. The podcast is off
// until they turn it on.
func (h *Handlers) podcast(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var o PodcastOptions
	var on bool
	// The settings package depends on agents, which reads the watch: the
	// podcast's own key is read here.
	err := h.db.QueryRow(r.Context(), `SELECT coalesce((doc->'podcast'->>'enabled')::boolean, false), coalesce(doc->'podcast'->>'format', ''),
		coalesce(doc->'podcast'->>'lang', ''), coalesce((doc->'podcast'->>'minutes')::int, 0), coalesce((doc->'podcast'->>'days')::int, 0)
		FROM user_settings WHERE user_id = $1`, who.UserID).Scan(&on, &o.Format, &o.Lang, &o.Minutes, &o.Days)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.fail(w, err)
		return
	}
	if !on {
		httpx.Problem(w, http.StatusConflict, "The podcast is off. Turn it on in Preferences → Tech watch podcast.")
		return
	}
	q := r.URL.Query()
	for name, dst := range map[string]*string{"format": &o.Format, "lang": &o.Lang, "topic": &o.TopicID} {
		if v := q.Get(name); v != "" {
			*dst = v
		}
	}
	for name, dst := range map[string]*int{"minutes": &o.Minutes, "days": &o.Days} {
		if v := q.Get(name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				h.fail(w, ValidationError{name + " is a number."})
				return
			}
			*dst = n
		}
	}
	if o.TopicID != "" {
		if _, err := uuid.Parse(o.TopicID); err != nil {
			h.fail(w, ValidationError{"topic is an id."})
			return
		}
	}
	loc, err := userLocation(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	ep, err := BuildEpisode(r.Context(), h.db, who.UserID, o, h.now(), loc)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ep)
}
