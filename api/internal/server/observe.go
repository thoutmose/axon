package server

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds the API's Prometheus instruments.
type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewMetrics registers the API's metrics on reg. axon_db_up pings the
// database on every scrape, so alerts see readiness without probing HTTP.
func NewMetrics(reg prometheus.Registerer, version string, db Pinger) *Metrics {
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "axon_http_requests_total",
			Help: "HTTP requests by method, route pattern, and status code.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "axon_http_request_duration_seconds",
			Help:    "HTTP request duration by method and route pattern.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"method", "route"}),
	}
	buildInfo := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "axon_build_info",
		Help:        "Always 1; the version label identifies the running build.",
		ConstLabels: prometheus.Labels{"version": version},
	})
	buildInfo.Set(1)
	dbUp := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "axon_db_up",
		Help: "1 if PostgreSQL answered a ping during this scrape, else 0.",
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if db.Ping(ctx) != nil {
			return 0
		}
		return 1
	})
	reg.MustRegister(m.requests, m.duration, buildInfo, dbUp)
	return m
}

// observe records metrics and one structured log line per request. It logs
// only method, route pattern, status, and timing: never the raw path, query
// string, headers, or body, which may carry personal data or tokens.
func observe(m *Metrics, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			elapsed := time.Since(start)

			m.requests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
			m.duration.WithLabelValues(r.Method, route).Observe(elapsed.Seconds())

			if route == "/healthz" || route == "/readyz" {
				return // probed constantly; metrics are enough
			}
			log.Info("request",
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", elapsed.Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
