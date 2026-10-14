// Package app composes the API: health routes, accounts, and privacy.
package app

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/openapi"

	"github.com/thoutmose/axon/api/internal/agents"
	"github.com/thoutmose/axon/api/internal/ankisync"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/avatars"
	"github.com/thoutmose/axon/api/internal/budget"
	"github.com/thoutmose/axon/api/internal/calendars"
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/clip"
	"github.com/thoutmose/axon/api/internal/dashboard"
	"github.com/thoutmose/axon/api/internal/focus"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/health"
	"github.com/thoutmose/axon/api/internal/household"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/idem"
	"github.com/thoutmose/axon/api/internal/imports"
	"github.com/thoutmose/axon/api/internal/insights"
	"github.com/thoutmose/axon/api/internal/journal"
	"github.com/thoutmose/axon/api/internal/knowledge"
	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/learning"
	"github.com/thoutmose/axon/api/internal/media"
	_ "github.com/thoutmose/axon/api/internal/mirror" // registers the vault mirror hooks (EP-48)
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/practice"
	"github.com/thoutmose/axon/api/internal/privacy"
	"github.com/thoutmose/axon/api/internal/progress"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/reminders"
	"github.com/thoutmose/axon/api/internal/repos"
	"github.com/thoutmose/axon/api/internal/roadmaps"
	"github.com/thoutmose/axon/api/internal/semantic"
	"github.com/thoutmose/axon/api/internal/server"
	"github.com/thoutmose/axon/api/internal/settings"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Handler builds the complete API handler.
// provider is the AI model provider for agents, or nil to run without them.
func Handler(version string, cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger, m *server.Metrics, provider agents.Provider) (http.Handler, error) {
	authn, err := auth.New(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	priv := privacy.New(pool, log, authn.ClearSessionCookie)
	priv.Register("graph", graph.Export)
	graphAPI := graph.NewHandlers(pool, log)
	tasksAPI := tasks.NewHandlers(pool, log)
	notesAPI := notes.NewHandlers(pool, log)
	goalsAPI := goals.NewHandlers(pool, log)
	habitsAPI := habits.NewHandlers(pool, log)
	projectsAPI := projects.NewHandlers(pool, log)
	roadmapsAPI := roadmaps.NewHandlers(pool, log)
	paraAPI := para.NewHandlers(pool, log)
	budgetAPI := budget.NewHandlers(pool, log)
	knowledgeAPI := knowledge.NewHandlers(pool, log)
	dashboardAPI := dashboard.NewHandlers(pool, log)
	insightsAPI := insights.NewHandlers(pool, log)
	contact := os.Getenv("AXON_CONTACT")
	if contact == "" {
		contact = cfg.BaseURL
	}
	healthAPI := health.NewHandlers(pool, log, health.NewCatalogs(pool, version, contact, os.Getenv("USDA_API_KEY")))
	priv.Register("health", health.Export)
	journalAPI := journal.NewHandlers(pool, log)
	priv.Register("journal", journal.Export)
	mediaAPI := media.NewHandlers(pool, log, media.NewCatalogs(pool, version, contact, os.Getenv("TMDB_API_KEY")))
	var sealer *auth.Sealer
	if cfg.EncryptionKey != "" {
		if sealer, err = auth.NewSealer(cfg.EncryptionKey); err != nil {
			return nil, err
		}
	}
	agentsAPI, err := agents.NewRunner(pool, log, provider, sealer)
	reposAPI := repos.NewHandlers(pool, log, sealer)
	if err != nil {
		return nil, err
	}
	priv.RegisterFiles("notes", notes.Files)
	priv.Register("agents", agents.Export)
	priv.Register("settings", settings.Export)
	priv.Register("habit_scorecard", habits.ExportScorecard)
	priv.Register("budget", budget.Export)
	settingsAPI := settings.NewHandlers(pool, log, agentsAPI)
	avatarsAPI := avatars.NewHandlers(pool, log)
	focusAPI := focus.NewHandlers(pool, log)
	cardsAPI := cards.NewHandlers(pool, log)
	learningAPI := learning.NewHandlers(pool, log)
	practiceAPI := practice.NewHandlers(pool, log)
	priv.Register("practice", practice.Export)
	remindersAPI, err := Reminders(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	calendarsAPI := calendars.NewStore(pool, log, sealer)
	importsAPI := imports.NewHandlers(pool, log)
	householdAPI := household.NewHandlers(pool, log)
	semanticAPI := semantic.NewHandlers(pool, log, sealer)
	priv.Register("household", household.Export)
	priv.Register("trash", graph.TrashExport)
	priv.Register("calendar_sources", calendars.Export)
	priv.Register("learning", learning.Export)
	progressAPI := progress.NewHandlers(pool, log)
	priv.Register("streak_freezes", progress.Export)
	ankiAPI := ankisync.New(pool, log, sealer, os.Getenv("AXON_ANKI_BRIDGE_URL"), os.Getenv("ANKI_SYNC_URL"))
	agentsAPI.OnApproved = ankiAPI.AfterApproved
	priv.Register("anki_account", ankiAPI.Export)
	priv.Register("card_reviews", cards.ExportReviews)
	priv.RegisterFiles("account", avatars.Files)
	// What the API says to people, in their language (EP-87), on every route.
	return l10n.Middleware(server.New(version, pool, log, m, func(r chi.Router) {
		r.Get("/v1/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(openapi.Spec)
		})
		// Picture uploads: larger bodies than JSON, re-encoded by the handlers.
		r.Group(func(r chi.Router) {
			r.Use(limitBody(12<<20), authn.SameOrigin)
			r.With(authn.RequireSession).Group(avatarsAPI.UploadRoutes)
			r.With(authn.Require).Group(cardsAPI.UploadRoutes)
			r.With(authn.Require).Group(agentsAPI.UploadRoutes)
		})
		r.Group(func(r chi.Router) {
			r.Use(limitBody(50<<20), authn.SameOrigin, authn.Require, idem.Middleware(pool, log))
			notesAPI.Routes(r)
			importsAPI.Routes(r)
		})
		r.Group(func(r chi.Router) {
			r.Use(limitBody(httpx.MaxBody), authn.SameOrigin)
			authn.Routes(r)
			dashboardAPI.PublicRoutes(r) // the feed URL's token is its credential
			r.Group(func(r chi.Router) {
				r.Use(authn.RequireSession)
				priv.Routes(r)
				dashboardAPI.FeedRoutes(r)
				agentsAPI.SessionRoutes(r)
				settingsAPI.Routes(r)
				avatarsAPI.Routes(r)
				ankiAPI.Routes(r)
				reposAPI.Routes(r)
			})
			r.Group(func(r chi.Router) {
				r.Use(authn.Require, idem.Middleware(pool, log))
				graphAPI.Routes(r)
				tasksAPI.Routes(r)
				goalsAPI.Routes(r)
				habitsAPI.Routes(r)
				projectsAPI.Routes(r)
				roadmapsAPI.Routes(r)
				focusAPI.Routes(r)
				cardsAPI.Routes(r)
				learningAPI.Routes(r)
				practiceAPI.Routes(r)
				remindersAPI.Routes(r)
				calendarsAPI.Routes(r)
				householdAPI.Routes(r)
				semanticAPI.Routes(r)
				clip.Routes(log)(r)
				progressAPI.Routes(r)
				paraAPI.Routes(r)
				budgetAPI.Routes(r)
				mediaAPI.Routes(r)
				healthAPI.Routes(r)
				journalAPI.Routes(r)
				knowledgeAPI.Routes(r)
				dashboardAPI.Routes(r)
				insightsAPI.Routes(r)
				agentsAPI.Routes(r)
			})
		})
	})), nil
}

// Calendars returns the store that fetches followed calendars (EP-58); the
// process runs it to keep them fresh.
func Calendars(cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger) (*calendars.Store, error) {
	var sealer *auth.Sealer
	if cfg.EncryptionKey != "" {
		var err error
		if sealer, err = auth.NewSealer(cfg.EncryptionKey); err != nil {
			return nil, err
		}
	}
	return calendars.NewStore(pool, log, sealer), nil
}

// Semantic returns the indexer that embeds notes for search by meaning
// (EP-62); the process runs it.
func Semantic(cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger) (*semantic.Indexer, error) {
	var sealer *auth.Sealer
	if cfg.EncryptionKey != "" {
		var err error
		if sealer, err = auth.NewSealer(cfg.EncryptionKey); err != nil {
			return nil, err
		}
	}
	return semantic.NewIndexer(pool, log, sealer), nil
}

// Reminders returns the reminders worker (EP-56); the API serves its
// endpoints, and the process runs one to send reminders on time.
func Reminders(cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger) (*reminders.Worker, error) {
	var sealer *auth.Sealer
	if cfg.EncryptionKey != "" {
		var err error
		if sealer, err = auth.NewSealer(cfg.EncryptionKey); err != nil {
			return nil, err
		}
	}
	mailer, err := reminders.NewMailer(os.Getenv("AXON_SMTP_URL"), os.Getenv("AXON_SMTP_FROM"), cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	contact := os.Getenv("AXON_CONTACT")
	if contact == "" {
		contact = cfg.BaseURL
	}
	return reminders.NewWorker(pool, log, sealer, mailer, contact), nil
}

func limitBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
