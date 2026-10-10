// Package app composes the API: health routes, accounts, and privacy.
package app

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/privacy"
	"github.com/thoutmose/axon/api/internal/server"
)

// Handler builds the complete API handler.
func Handler(version string, cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger, m *server.Metrics) (http.Handler, error) {
	authn, err := auth.New(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	priv := privacy.New(pool, log, authn.ClearSessionCookie)
	priv.Register("graph", graph.Export)
	return server.New(version, pool, log, m, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(limitBody, authn.SameOrigin)
			authn.Routes(r)
			r.Group(func(r chi.Router) {
				r.Use(authn.Require)
				priv.Routes(r)
			})
		})
	}), nil
}

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxBody)
		next.ServeHTTP(w, r)
	})
}
