// Package telemetry measures what each part of Axon costs (EP-157): the
// background workers' runs, time, and failures, and the storage each
// feature holds, for the dashboards on srv-monitoring. Requests are
// measured by the server package, by route and by feature.
package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/db"
)

var (
	workerRuns = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "axon_worker_runs_total",
		Help: "Runs of each background worker, by result (ok or error).",
	}, []string{"worker", "result"})
	workerSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "axon_worker_run_seconds",
		Help:    "How long each background worker's runs take.",
		Buckets: []float64{.01, .05, .1, .5, 1, 5, 15, 60, 300},
	}, []string{"worker"})
)

// Time starts measuring a worker's run; call the result with its error.
func Time(worker string) func(error) {
	start := time.Now()
	return func(err error) {
		workerSeconds.WithLabelValues(worker).Observe(time.Since(start).Seconds())
		result := "ok"
		if err != nil {
			result = "error"
		}
		workerRuns.WithLabelValues(worker, result).Inc()
	}
}

// features are the tables each feature keeps its data in.
var features = map[string][]string{
	"notes":     {"notes", "note_files", "note_embeddings", "note_tombstones", "note_file_tombstones", "note_recalls", "memory_palaces"},
	"vault":     {"vault_mirror", "vault_mirror_cursor", "vault_mirror_queue", "vault_mirror_users", "vault_sources"},
	"tasks":     {"tasks", "projects", "project_cards", "checkins", "areas", "resources"},
	"goals":     {"goals", "milestones", "decisions"},
	"habits":    {"habits", "habit_checkins", "habit_scorecards", "streak_freezes"},
	"learning":  {"roadmaps", "skills", "cards", "card_reviews", "assessments", "practice_problems", "practice_attempts", "practice_tests", "certifications"},
	"watch":     {"watch_items", "watch_sources", "watch_topics", "watch_seen", "watch_shares", "digest_items"},
	"media":     {"media_items", "media_covers", "media_lookups"},
	"health":    {"health_profiles", "food_entries", "food_lookups", "foods", "water_entries", "caffeine_entries", "workouts", "workout_sets"},
	"budget":    {"budget_accounts", "budget_categories", "budget_transactions", "budget_pots", "budget_pot_moves", "budget_profiles", "budget_recurring", "budget_rules"},
	"journal":   {"journal_entries"},
	"household": {"households", "household_members", "household_lists", "household_items", "household_invites"},
	"agents":    {"agent_runs", "agent_proposals", "agent_memories", "agent_models", "agent_providers", "agent_settings"},
	"graph":     {"entities", "links", "events", "trash", "audit_events"},
	"focus":     {"focus_sessions"},
	"calendars": {"calendar_events", "calendar_feeds", "calendar_sources"},
	"accounts":  {"users", "sessions", "passkeys", "api_tokens", "user_settings", "consents", "push_subscriptions", "notify_channels"},
}

// storage reports each feature's tables' size, and the database's, at most
// once a minute: sizes change slowly, and the query reads the catalogue.
type storage struct {
	db      db.Querier
	bytes   *prometheus.Desc
	total   *prometheus.Desc
	mu      sync.Mutex
	at      time.Time
	cached  map[string]float64
	dbBytes float64
}

func (s *storage) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.bytes
	ch <- s.total
}

func (s *storage) Collect(ch chan<- prometheus.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.at) > time.Minute {
		s.refresh()
	}
	for f, b := range s.cached {
		ch <- prometheus.MustNewConstMetric(s.bytes, prometheus.GaugeValue, b, f)
	}
	ch <- prometheus.MustNewConstMetric(s.total, prometheus.GaugeValue, s.dbBytes)
}

func (s *storage) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out := map[string]float64{}
	for f, tables := range features {
		var b int64
		// to_regclass is null for a table this version does not have.
		if err := s.db.QueryRow(ctx, `SELECT coalesce(sum(pg_total_relation_size(to_regclass(t))), 0)::bigint
			FROM unnest($1::text[]) t WHERE to_regclass(t) IS NOT NULL`, tables).Scan(&b); err != nil {
			return
		}
		out[f] = float64(b)
	}
	var total int64
	if err := s.db.QueryRow(ctx, `SELECT pg_database_size(current_database())`).Scan(&total); err != nil {
		return
	}
	s.cached, s.dbBytes, s.at = out, float64(total), time.Now()
}

// Register adds the workers' and storage metrics to reg.
func Register(reg prometheus.Registerer, q db.Querier) {
	reg.MustRegister(workerRuns, workerSeconds, &storage{
		db:    q,
		bytes: prometheus.NewDesc("axon_storage_bytes", "Bytes each feature's tables hold, indexes and toast included.", []string{"feature"}, nil),
		total: prometheus.NewDesc("axon_database_bytes", "Bytes of the whole database.", nil, nil),
	})
}
