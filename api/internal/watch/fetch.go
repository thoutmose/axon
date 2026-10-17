package watch

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/clip"

	"github.com/thoutmose/axon/api/internal/telemetry"
)

// Metrics for monitoring: fetches by result, news by outcome, fetch time.
var (
	fetches = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "axon_watch_fetches_total", Help: "Tech watch fetches by result (ok, not_modified, error)."}, []string{"result"})
	itemsC  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "axon_watch_items_total", Help: "Tech watch entries by outcome (new, merged, skipped)."}, []string{"outcome"})
	fetchS  = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "axon_watch_fetch_seconds", Help: "Time to fetch and store one source.", Buckets: prometheus.ExponentialBuckets(0.1, 2, 10)})
)

// RegisterMetrics adds the tech watch metrics to the process registry.
func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(fetches, itemsC, fetchS)
}

// Retention is how long news nobody saved, kept, or linked are kept.
const Retention = 180 * 24 * time.Hour

// fullTextPerRun bounds how many article pages one fetch reads.
const fullTextPerRun = 10

// Worker fetches sources when their schedule comes.
type Worker struct {
	DB        *pgxpool.Pool
	Log       *slog.Logger
	Parallel  int
	lastPurge time.Time
}

// Run checks every minute until ctx ends.
func (w *Worker) Run(ctx context.Context) {
	for {
		done := telemetry.Time("watch")
		err := w.Step(ctx, time.Now())
		done(err)
		if err != nil && ctx.Err() == nil {
			w.Log.Error("tech watch step failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

// Step fetches every source that is due at now, a few at a time.
func (w *Worker) Step(ctx context.Context, now time.Time) error {
	rows, err := w.DB.Query(ctx, `SELECT s.id, s.user_id FROM watch_sources s JOIN users u ON u.id = s.user_id
		WHERE s.enabled AND NOT u.demo AND s.next_fetch_at <= $1 ORDER BY s.next_fetch_at LIMIT 100`, now)
	if err != nil {
		return err
	}
	type due struct{ id, user uuid.UUID }
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (due, error) {
		var d due
		err := r.Scan(&d.id, &d.user)
		return d, err
	})
	if err != nil {
		return err
	}
	n := w.Parallel
	if n <= 0 {
		n = 4
	}
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	for _, d := range list {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			if _, err := FetchSource(ctx, w.DB, d.user, d.id, now); err != nil && ctx.Err() == nil {
				w.Log.Warn("tech watch fetch failed", "source", d.id, "err", err)
			}
		}()
	}
	wg.Wait()
	// A few videos get their words (EP-123).
	if err := TranscribePending(ctx, w.DB, WhisperFromEnv(), PendingTranscripts); err != nil {
		return err
	}
	if now.Sub(w.lastPurge) > 24*time.Hour {
		w.lastPurge = now
		if n, err := Purge(ctx, w.DB, now.Add(-Retention)); err != nil {
			return err
		} else if n > 0 {
			w.Log.Info("tech watch purged old news", "items", n)
		}
	}
	return ctx.Err()
}

// FetchSource fetches one source now and stores what is new. A failure is
// recorded on the source and retried sooner than its schedule, with a
// growing pause.
func FetchSource(ctx context.Context, pool *pgxpool.Pool, userID, id uuid.UUID, now time.Time) (Stats, error) {
	start := time.Now()
	defer func() { fetchS.Observe(time.Since(start).Seconds()) }()
	src, err := GetSource(ctx, pool, userID, id)
	if err != nil {
		return Stats{}, err
	}
	next, err := nextRun(ctx, pool, userID, src.Schedule, now)
	if err != nil {
		return Stats{}, err
	}
	fail := func(cause error) (Stats, error) {
		fetches.WithLabelValues("error").Inc()
		backoff := MinInterval << min(src.Failures, 5) // 15 min, 30 min, … 8 h
		retry := now.Add(backoff)
		if retry.After(next) {
			retry = next
		}
		_, err := pool.Exec(ctx, `UPDATE watch_sources SET last_fetched_at = $2, last_error = $3, failures = failures + 1, next_fetch_at = $4 WHERE id = $1`,
			id, now, cut(cause.Error(), 500), retry)
		return Stats{}, errors.Join(cause, err)
	}
	fctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var r response
	var feed Feed
	if o, ok := origins[src.Origin]; ok { // not a feed: its origin reads it (EP-124)
		if feed, err = o.Fetch(fctx, pool, userID, src); err != nil {
			return fail(err)
		}
	} else if r, err = get(fctx, src.URL, src.etag, src.mod); err != nil {
		return fail(err)
	}
	if r.notModified {
		fetches.WithLabelValues("not_modified").Inc()
		_, err := pool.Exec(ctx, `UPDATE watch_sources SET last_fetched_at = $2, last_ok_at = $2, last_error = '', failures = 0, next_fetch_at = $3 WHERE id = $1`, id, now, next)
		return Stats{}, err
	}
	if _, ok := origins[src.Origin]; !ok {
		feed, err = Parse(r.body)
	}
	if err != nil && src.LastOKAt == nil {
		// Imported as a site, a Medium author, a Substack…: find its feed once.
		found, derr := Discover(fctx, src.URL)
		if derr != nil {
			return fail(derr)
		}
		if _, err := pool.Exec(ctx, `UPDATE watch_sources SET url = $2, title = CASE WHEN title = $3 THEN $4 ELSE title END WHERE id = $1`,
			id, cut(found.URL, 2000), hostOf(src.URL), cut(firstOf(found.Feed.Title, src.Title), 200)); err != nil {
			return fail(errors.New("this feed is already followed under another name"))
		}
		feed, err, r.etag, r.mod = found.Feed, nil, "", ""
	}
	if err != nil {
		return fail(errors.New("the address no longer gives a feed"))
	}
	// An import without titles named the source after its host (a YouTube
	// channel, www.youtube.com): the feed's own title replaces it.
	if feed.Title != "" && src.Title == hostOf(src.URL) {
		if _, err := pool.Exec(ctx, `UPDATE watch_sources SET title = $2 WHERE id = $1`, id, cut(feed.Title, 200)); err != nil {
			return fail(err)
		}
	}
	if src.FullText {
		fillFullText(fctx, pool, src.ID, feed.Items)
	}
	var st Stats
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var err error
		if st, err = Ingest(ctx, tx, userID, src, feed.Items, now); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE watch_sources SET etag = $2, last_modified = $3, last_fetched_at = $4, last_ok_at = $4, last_error = '', failures = 0,
			next_fetch_at = $5, site_url = CASE WHEN site_url = '' THEN $6 ELSE site_url END WHERE id = $1`,
			id, cut(r.etag, 500), cut(r.mod, 100), now, next, cut(feed.SiteURL, 2000))
		return err
	})
	if err != nil {
		return fail(err)
	}
	fetches.WithLabelValues("ok").Inc()
	itemsC.WithLabelValues("new").Add(float64(st.New))
	itemsC.WithLabelValues("merged").Add(float64(st.Merged))
	itemsC.WithLabelValues("skipped").Add(float64(st.Skipped))
	return st, nil
}

// fillFullText reads the article pages of new entries whose feed carries
// only an excerpt (Medium and Substack previews, most news sites), a few
// per run, politely one after the other.
func fillFullText(ctx context.Context, pool *pgxpool.Pool, sourceID uuid.UUID, entries []Entry) {
	done := 0
	for i := range entries {
		e := &entries[i]
		if done == fullTextPerRun || ctx.Err() != nil {
			return
		}
		if e.URL == "" || len(strings.Fields(PlainText(e.Content))) >= 150 {
			continue
		}
		var seen bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM watch_seen WHERE source_id = $1 AND guid_hash = $2)`, sourceID, guidHash(*e)).Scan(&seen); err != nil || seen {
			continue
		}
		done++
		page, err := clip.Fetch(ctx, e.URL)
		if err != nil || len(strings.Fields(page.Markdown)) < len(strings.Fields(PlainText(e.Content))) {
			continue
		}
		e.Markdown = page.Markdown
	}
}
