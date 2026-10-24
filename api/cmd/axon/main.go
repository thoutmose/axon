// Command axon runs the Axon API server and its maintenance tasks.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/demo"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/instance"
	"github.com/thoutmose/axon/api/internal/media"
	"github.com/thoutmose/axon/api/internal/mirror"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/notesync"
	"github.com/thoutmose/axon/api/internal/server"
	"github.com/thoutmose/axon/api/internal/telemetry"
	"github.com/thoutmose/axon/api/internal/watch"
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
	case "demo":
		switch {
		case len(args) == 2 && args[1] == "reset":
			return demoReset()
		case len(args) == 4 && args[1] == "starter" && args[2] == "--user":
			return demoStarter(args[3])
		}
		return errors.New("usage: axon demo reset | axon demo starter --user <id or display name>")
	case "instance":
		return instanceCmd(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	default:
		return fmt.Errorf("unknown command %q (want serve, migrate, integrity, notes-sync, demo, instance, or version)", cmd)
	}
}

func serve() error {
	log := slog.New(clientGone{slog.NewJSONHandler(os.Stdout, nil)})

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
	watch.RegisterMetrics(reg)
	telemetry.Register(reg, pool) // workers and storage by feature (EP-157)

	// The API and the metrics endpoint listen separately, so /metrics is never
	// reachable through Caddy and the public routes.
	providers, err := auth.ProvidersFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	cfg := auth.Config{
		BaseURL:          os.Getenv("AXON_BASE_URL"),
		RPID:             os.Getenv("WEBAUTHN_RP_ID"),
		EncryptionKey:    os.Getenv("AXON_ENCRYPTION_KEY"),
		OpenRegistration: os.Getenv("AXON_REGISTRATION") == "open",
		Demo:             demoEnabled(),
		Providers:        providers,
	}
	ai := provider(log)
	handler, err := app.Handler(version, cfg, pool, log, metrics, ai)
	if err != nil {
		return err
	}
	workers, err := backgroundWorkers(cfg, pool, log, ai)
	if err != nil {
		return err
	}
	go func() {
		if !waitForSchema(ctx, pool, log) {
			return
		}
		for _, w := range workers {
			go w.run(ctx)
		}
	}()
	watch.RegisterSourceAge(reg, pool)
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

// worker is one background loop, started once the database is migrated.
type worker struct {
	name string
	run  func(context.Context)
}

// backgroundWorkers lists every loop the server runs. Each one skips the
// shared demo account in its own query (NOT u.demo), so none is left out
// when the demo is on: that once stopped the tech watch in production.
func backgroundWorkers(cfg auth.Config, pool *pgxpool.Pool, log *slog.Logger, ai agents.Provider) ([]worker, error) {
	reminders, err := app.Reminders(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	cals, err := app.Calendars(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	index, err := app.Semantic(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	listeners, err := app.Replies(cfg, pool, log)
	if err != nil {
		return nil, err
	}
	summaries, err := app.VideoSummaries(cfg, pool, log, ai)
	if err != nil {
		return nil, err
	}
	var list []worker
	if cfg.Demo {
		list = append(list, worker{"demo", func(ctx context.Context) { keepDemo(ctx, pool, log) }})
	}
	return append(list,
		worker{"maintenance", func(ctx context.Context) { pruneMedia(ctx, pool, log) }},
		worker{"vault-mirror", mirror.New(pool, log).Run},
		worker{"reminders", reminders.Run},
		worker{"calendars", cals.Run},
		worker{"semantic-index", index.Run},
		worker{"replies", listeners.Run},
		worker{"watch", (&watch.Worker{DB: pool, Log: log}).Run},
		worker{"video-summaries", summaries.RunVideoSummaries},
		worker{"watch-knowledge", summaries.RunWatchKnowledge},
	), nil
}

// waitForSchema waits until `axon migrate up` has brought the database to
// this build's latest migration: on a fresh install the workers would fail
// on missing tables, and maintenance would wait a day for its next round.
func waitForSchema(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) bool {
	latest, err := db.Latest()
	if err != nil {
		log.Error("cannot read the migrations; workers not started", "err", err)
		return false
	}
	for waiting := false; ; waiting = true {
		var v int64
		err := pool.QueryRow(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&v)
		if err == nil && v >= latest {
			if waiting {
				log.Info("database migrated; starting workers", "version", v)
			}
			return true
		}
		if !waiting {
			log.Info("waiting for `axon migrate up` before starting workers", "want", latest)
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(5 * time.Second):
		}
	}
}

// clientGone logs as INFO an error that only says the request was cancelled:
// a closed tab is not a failure of Axon, and would page whoever watches ERRORs.
type clientGone struct{ slog.Handler }

func (h clientGone) Handle(ctx context.Context, r slog.Record) error {
	if r.Level == slog.LevelError {
		r.Attrs(func(a slog.Attr) bool {
			if err, ok := a.Value.Any().(error); ok && errors.Is(err, context.Canceled) {
				r.Level = slog.LevelInfo
				return false
			}
			return true
		})
	}
	return h.Handler.Handle(ctx, r)
}

func (h clientGone) WithAttrs(as []slog.Attr) slog.Handler {
	return clientGone{h.Handler.WithAttrs(as)}
}
func (h clientGone) WithGroup(name string) slog.Handler { return clientGone{h.Handler.WithGroup(name)} }

// demoEnabled reports whether this server offers the demo account (AXON_DEMO=1).
func demoEnabled() bool { return os.Getenv("AXON_DEMO") == "1" }

// keepDemo creates the demo account once the database is migrated, then
// resets it every night after 03:00 UTC, so visitors' changes never pile up.
// pruneMedia drops expired catalogue data once a day (TMDB allows six months),
// purges the trash, and gives older card reviews their prediction (EP-77).
func pruneMedia(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) {
	for {
		done := telemetry.Time("maintenance")
		err := media.Prune(ctx, pool)
		done(err)
		if err != nil && ctx.Err() == nil {
			log.Error("media prune failed", "err", err)
		}
		if err := graph.PurgeTrash(ctx, pool, time.Now()); err != nil && ctx.Err() == nil {
			log.Error("trash purge failed", "err", err)
		}
		if err := cards.ReplayPredictions(ctx, pool); err != nil && ctx.Err() == nil {
			log.Error("review prediction replay failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(24 * time.Hour):
		}
	}
}

func keepDemo(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) {
	last := time.Now().UTC()
	for {
		exists, err := demo.Exists(ctx, pool)
		now := time.Now().UTC()
		nightly := time.Date(last.Year(), last.Month(), last.Day(), 3, 0, 0, 0, time.UTC)
		if !nightly.After(last) {
			nightly = nightly.AddDate(0, 0, 1)
		}
		if err == nil && (!exists || now.After(nightly)) {
			if _, err := demo.Reset(ctx, pool, now); err != nil {
				log.Error("demo reset failed", "err", err)
			} else {
				log.Info("demo account reset")
				last = now
				if n, err := demo.Enrich(ctx, pool); err != nil {
					log.Warn("demo transcripts failed", "read", n, "err", err)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

// demoStarter gives a real account the demo's roadmap and what surrounds
// it, without its invented history (demo.Starter).
func demoStarter(who string) error {
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
	added, err := demo.Starter(ctx, pool, who, time.Now())
	if err != nil {
		return err
	}
	if len(added) == 0 {
		fmt.Println("nothing to add: the account already has it all")
	}
	for _, a := range added {
		fmt.Println("added", a)
	}
	return nil
}

func demoReset() error {
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
	if _, err := demo.Reset(ctx, pool, time.Now()); err != nil {
		return err
	}
	n, err := demo.Enrich(ctx, pool)
	fmt.Printf("demo account reset; %d video transcripts read\n", n)
	if err != nil {
		fmt.Println("transcripts:", err)
	}
	return nil
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
	ctx := context.Background()
	if err := db.Migrate(ctx, dsn, direction, os.Stdout); err != nil || direction != "up" {
		return err
	}
	// Data that needs Go to compute, for rows written before it existed.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	n, err := notes.BackfillTags(ctx, pool)
	if n > 0 {
		fmt.Printf("parsed tags for %d notes\n", n)
	}
	return err
}

// provider returns the server's default agent provider, or nil: people can
// still configure their own, and everything else works without one.
func provider(log *slog.Logger) agents.Provider {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return agents.NewAnthropic(key)
	}
	if os.Getenv("AXON_ANTHROPIC_OAUTH") == "1" { // a profile from "ant auth login", no key stored
		return agents.NewAnthropicFromEnvironment()
	}
	log.Info("no server AI provider; people can still set up their own under Agents")
	return nil
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
	watch := fs.Bool("watch", false, "keep running: sync whenever the vault or Axon changes (implies -two-way)")
	interval := fs.Duration("interval", 2*time.Second, "with -watch: how often to check for changes")
	pairCode := fs.String("pair", "", "while -token-file does not exist: pair with this one-time code from Axon and save the token there")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" || *url == "" || *tokenFile == "" {
		return errors.New("usage: axon notes-sync -dir DIR -url URL -token-file FILE [-pair CODE] [-two-way] [-git | -watch [-interval 2s]]")
	}
	if *watch && *useGit {
		return errors.New("-watch keeps files in step continuously; commit them with your own schedule instead of -git")
	}
	if _, err := os.Stat(*tokenFile); errors.Is(err, os.ErrNotExist) && *pairCode != "" {
		if err := os.MkdirAll(*dir, 0o700); err != nil {
			return err
		}
		cfg := notesync.Config{Dir: *dir, URL: *url, HTTP: &http.Client{Timeout: time.Minute}}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		p, err := notesync.Pair(ctx, &cfg, *pairCode, *tokenFile)
		cancel()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "paired: token saved to %s (expires %s)\n", *tokenFile, p.ExpiresAt)
		if len(p.Folders) > 0 {
			fmt.Fprintf(os.Stderr, "the vault was empty: created %d folders; the first sync writes Home.md, the templates, and your notes\n", len(p.Folders))
		}
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
	if *watch {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		log := slog.New(slog.NewTextHandler(os.Stderr, nil))
		log.Info("watching", "dir", *dir, "interval", interval.String())
		err := notesync.Watch(ctx, notesync.Config{
			Dir: *dir, URL: *url, Token: strings.TrimSpace(string(token)), StatePath: *statePath, TwoWay: true,
			HTTP: &http.Client{Timeout: time.Minute},
		}, *interval, log, func(r notesync.Report) {
			log.Info("synced", "pushed", r.Pushed, "pulled", r.Pulled, "deleted", r.Deleted, "conflicts", len(r.Conflicts))
		})
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cfg := notesync.Config{
		Dir: *dir, URL: *url, Token: strings.TrimSpace(string(token)), StatePath: *statePath, TwoWay: *twoWay,
		HTTP: &http.Client{Timeout: time.Minute},
	}
	rep, err := notesync.Run(ctx, cfg)
	if herr := notesync.Heartbeat(ctx, cfg, rep, err); herr != nil {
		fmt.Fprintln(os.Stderr, "could not report to Axon:", herr)
	}
	if err != nil {
		return err
	}
	fmt.Printf("pushed %d, pulled %d, deleted %d, unchanged %d, conflicts %d\n",
		rep.Pushed, rep.Pulled, rep.Deleted, rep.Unchanged, len(rep.Conflicts))
	for _, c := range rep.Conflicts {
		fmt.Printf("conflict: %s (Axon's version is next to it, named *.axon-conflict.*)\n", c)
	}
	for _, s := range rep.Skipped {
		fmt.Printf("skipped: %s (not a picture or PDF of the kind its name says, or too large)\n", s)
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

// instanceCmd moves a whole installation: "export [FILE]" writes the archive
// (to stdout without FILE), "import [FILE]" loads one into a fresh
// installation (from stdin without FILE) and then applies newer migrations.
func instanceCmd(args []string) error {
	if len(args) < 1 || len(args) > 2 || (args[0] != "export" && args[0] != "import") {
		return errors.New("usage: axon instance export|import [FILE]")
	}
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
	key := os.Getenv("AXON_ENCRYPTION_KEY")
	path := ""
	if len(args) == 2 && args[1] != "-" {
		path = args[1]
	}

	if args[0] == "export" {
		out := io.Writer(os.Stdout)
		if path != "" {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // an operator-chosen path
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()
			out = f
		}
		m, err := instance.Export(ctx, pool, out, version, key)
		if err != nil {
			return err
		}
		var rows int64
		for _, t := range m.Tables {
			rows += t.Rows
		}
		fmt.Fprintf(os.Stderr, "exported %d tables, %d rows, schema %d\n", len(m.Tables), rows, m.SchemaVersion)
		return nil
	}

	in := io.Reader(os.Stdin)
	if path != "" {
		f, err := os.Open(path) //nolint:gosec // an operator-chosen path
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		in = f
	}
	latest, err := db.Latest()
	if err != nil {
		return err
	}
	m, err := instance.Import(ctx, pool, in, key, latest, func(ctx context.Context, v int64) error {
		return db.MigrateTo(ctx, dsn, v, os.Stderr)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "imported %d tables from Axon %s (schema %d)\n", len(m.Tables), m.AxonVersion, m.SchemaVersion)
	return migrate("up")
}
