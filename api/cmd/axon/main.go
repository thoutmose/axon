// Command axon runs the Axon API server and its maintenance tasks.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // users' time zones, even in images without /usr/share/zoneinfo

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thoutmose/axon/api/internal/app"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/server"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "axon:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "serve":
		return serve()
	case "migrate":
		if len(args) != 2 {
			return errors.New("usage: axon migrate up|down|status")
		}
		return migrate(args[1])
	case "integrity":
		return integrity()
	case "version":
		fmt.Println(version)
		return nil
	default:
		return fmt.Errorf("unknown command %q (want serve, migrate, integrity, or version)", cmd)
	}
}

func serve() error {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn, err := databaseURL()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The pool connects lazily, so the API starts even when the database is
	// down; /readyz reports that state instead.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	defer pool.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registerPoolStats(reg, pool)
	metrics := server.NewMetrics(reg, version, pool)

	// The API and the metrics endpoint listen separately, so /metrics is never
	// reachable through Caddy and the public routes.
	handler, err := app.Handler(version, auth.Config{
		BaseURL:          os.Getenv("AXON_BASE_URL"),
		RPID:             os.Getenv("WEBAUTHN_RP_ID"),
		EncryptionKey:    os.Getenv("AXON_ENCRYPTION_KEY"),
		OpenRegistration: os.Getenv("AXON_REGISTRATION") == "open",
	}, pool, log, metrics)
	if err != nil {
		return err
	}
	api := newHTTPServer(envOr("AXON_ADDR", ":8080"), handler)
	metricsSrv := newHTTPServer(envOr("AXON_METRICS_ADDR", ":9464"), promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	errc := make(chan error, 2)
	for _, srv := range []*http.Server{api, metricsSrv} {
		go func() {
			log.Info("listening", "addr", srv.Addr, "version", version)
			errc <- srv.ListenAndServe()
		}()
	}

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdownCtx)
	return api.Shutdown(shutdownCtx)
}

func newHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func registerPoolStats(reg prometheus.Registerer, pool *pgxpool.Pool) {
	gauge := func(name, help string, value func(*pgxpool.Stat) int32) prometheus.Collector {
		return prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: name, Help: help},
			func() float64 { return float64(value(pool.Stat())) })
	}
	reg.MustRegister(
		gauge("axon_db_pool_connections", "Open connections in the pool.", (*pgxpool.Stat).TotalConns),
		gauge("axon_db_pool_connections_in_use", "Connections currently acquired.", (*pgxpool.Stat).AcquiredConns),
		gauge("axon_db_pool_connections_max", "Maximum pool size.", (*pgxpool.Stat).MaxConns),
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func migrate(direction string) error {
	dsn, err := databaseURL()
	if err != nil {
		return err
	}
	return db.Migrate(context.Background(), dsn, direction, os.Stdout)
}

// integrity reports data the schema cannot rule out by itself; exit status 1 if any.
func integrity() error {
	dsn, err := databaseURL()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	problems, err := graph.CheckIntegrity(ctx, pool)
	if err != nil {
		return err
	}
	for _, p := range problems {
		fmt.Printf("%-40s %d\n", p.Check, p.Count)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d integrity problem(s)", len(problems))
	}
	fmt.Println("integrity ok")
	return nil
}

func databaseURL() (string, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return "", errors.New("DATABASE_URL is not set")
	}
	return dsn, nil
}
