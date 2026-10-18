// Package server wires the HTTP routes of the Axon API.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Pinger reports whether a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// New returns the API handler. mount adds the application routes.
func New(version string, db Pinger, log *slog.Logger, m *Metrics, mount ...func(chi.Router)) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(observe(m, log)) // outside Recoverer, so panics are recorded as 500s
	r.Use(middleware.Recoverer)

	// Liveness: the process is up. Never depends on other services.
	healthz := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}

	// Readiness: the API can serve requests that need the database.
	readyz := func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			log.Warn("readiness check failed", "dependency", "postgres", "err", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
	// HEAD too: uptime probes often use it.
	for path, h := range map[string]http.HandlerFunc{"/healthz": healthz, "/readyz": readyz} {
		r.Get(path, h)
		r.Head(path, h)
	}

	for _, f := range mount {
		f(r)
	}
	return r
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
