// Command axon runs the Axon API server and its maintenance tasks.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // users' time zones, even in images without /usr/share/zoneinfo

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thoutmose/axon/api/internal/agents"
	"github.com/thoutmose/axon/api/internal/app"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notesync"
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
	case "notes-sync":
		return notesSync(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	default:
		return fmt.Errorf("unknown command %q (want serve, migrate, integrity, notes-sync, or version)", cmd)
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
	}, pool, log, metrics, provider(log))
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

// provider returns the agent model provider, or nil when no key is set:
// Axon then runs with agents unavailable and everything else unchanged.
func provider(log *slog.Logger) agents.Provider {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		log.Info("ANTHROPIC_API_KEY not set; agents are unavailable")
		return nil
	}
	return agents.NewAnthropic(key)
}

// notesSync keeps a Markdown vault and Axon's notes in step (see internal/notesync).
func notesSync(args []string) error {
	fs := flag.NewFlagSet("notes-sync", flag.ContinueOnError)
	dir := fs.String("dir", "", "vault folder")
	url := fs.String("url", "", "Axon base URL, e.g. https://srv-prod.tail781d53.ts.net")
	tokenFile := fs.String("token-file", "", "file holding a write-scope access token")
	statePath := fs.String("state", "", "sync state file (default <dir>/../.axon-sync-<folder>.json)")
	twoWay := fs.Bool("two-way", false, "also copy Axon's changes and deletions to the files (default: import only)")
	useGit := fs.Bool("git", false, "git pull before and commit + push after")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *url == "" || *tokenFile == "" {
		return errors.New("usage: axon notes-sync -dir DIR -url URL -token-file FILE [-two-way] [-git]")
	}
	token, err := os.ReadFile(*tokenFile)
	if err != nil {
		return err
	}
	if *statePath == "" {
		abs, err := filepath.Abs(*dir)
		if err != nil {
			return err
		}
		*statePath = filepath.Join(filepath.Dir(abs), ".axon-sync-"+filepath.Base(abs)+".json")
	}
	git := func(args ...string) error {
		cmd := exec.Command("git", append([]string{"-C", *dir}, args...)...) //nolint:gosec // fixed git subcommands; dir is the operator's flag
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		return cmd.Run()
	}
	if *useGit {
		if err := git("pull", "--rebase", "--autostash", "--quiet"); err != nil {
			return fmt.Errorf("git pull: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rep, err := notesync.Run(ctx, notesync.Config{
		Dir: *dir, URL: *url, Token: strings.TrimSpace(string(token)), StatePath: *statePath, TwoWay: *twoWay,
		HTTP: &http.Client{Timeout: time.Minute},
	})
	if err != nil {
		return err
	}
	fmt.Printf("pushed %d, pulled %d, deleted %d, unchanged %d, conflicts %d\n",
		rep.Pushed, rep.Pulled, rep.Deleted, rep.Unchanged, len(rep.Conflicts))
	for _, c := range rep.Conflicts {
		fmt.Printf("conflict: %s (Axon's version is next to it as *%s)\n", c, notesync.ConflictSuffix)
	}
	if *useGit && (rep.Pulled > 0 || rep.Deleted > 0 || len(rep.Conflicts) > 0) {
		if err := git("add", "-A"); err != nil {
			return err
		}
		if git("diff", "--cached", "--quiet") != nil {
			if err := git("commit", "--quiet", "-m", "axon: sync notes"); err != nil {
				return err
			}
			if err := git("push", "--quiet"); err != nil {
				return fmt.Errorf("git push: %w", err)
			}
		}
	}
	return nil
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
