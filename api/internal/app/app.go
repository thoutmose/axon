// Package app composes the API: health routes, accounts, and privacy.
package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/openapi"

	"github.com/thoutmose/axon/api/internal/agents"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/dashboard"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/idem"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/privacy"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/repos"
	"github.com/thoutmose/axon/api/internal/server"
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
	dashboardAPI := dashboard.NewHandlers(pool, log)
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
	return server.New(version, pool, log, m, func(r chi.Router) {
		r.Get("/v1/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write(openapi.Spec)
		})
		r.Group(func(r chi.Router) {
			r.Use(limitBody(50<<20), authn.SameOrigin, authn.Require, idem.Middleware(pool, log))
			notesAPI.Routes(r)
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
				reposAPI.Routes(r)
			})
			r.Group(func(r chi.Router) {
				r.Use(authn.Require, idem.Middleware(pool, log))
				graphAPI.Routes(r)
				tasksAPI.Routes(r)
				goalsAPI.Routes(r)
				habitsAPI.Routes(r)
				projectsAPI.Routes(r)
				dashboardAPI.Routes(r)
				agentsAPI.Routes(r)
			})
		})
	}), nil
}

func limitBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}
