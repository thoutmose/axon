// Package db owns the database schema and its migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedded embed.FS

// Migrate applies all pending migrations ("up"), rolls back the latest one
// ("down"), or prints the state of every migration ("status") to out.
func Migrate(ctx context.Context, dsn, direction string, out io.Writer) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	p, err := provider(conn)
	if err != nil {
		return err
	}

	switch direction {
	case "up":
		results, err := p.Up(ctx)
		for _, r := range results {
			fmt.Fprintln(out, "applied", r.Source.Path)
		}
		return err
	case "down":
		r, err := p.Down(ctx)
		if r != nil {
			fmt.Fprintln(out, "rolled back", r.Source.Path)
		}
		return err
	case "status":
		statuses, err := p.Status(ctx)
		for _, s := range statuses {
			fmt.Fprintf(out, "%-8s %s\n", s.State, s.Source.Path)
		}
		return err
	default:
		return fmt.Errorf("unknown migrate direction %q (want up, down, or status)", direction)
	}
}

func provider(conn *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectPostgres, conn, migrations)
}

// Latest is the newest migration version this build knows.
func Latest() (int64, error) {
	entries, err := fs.ReadDir(embedded, "migrations")
	if err != nil {
		return 0, err
	}
	var latest int64
	for _, e := range entries {
		digits, _, _ := strings.Cut(e.Name(), "_")
		v, err := strconv.ParseInt(digits, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("migration %s has no version: %w", e.Name(), err)
		}
		latest = max(latest, v)
	}
	return latest, nil
}

// MigrateTo moves the schema to exactly version, up or down.
func MigrateTo(ctx context.Context, dsn string, version int64, out io.Writer) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	p, err := provider(conn)
	if err != nil {
		return err
	}
	current, err := p.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	var results []*goose.MigrationResult
	switch {
	case current < version:
		results, err = p.UpTo(ctx, version)
	case current > version:
		results, err = p.DownTo(ctx, version)
	}
	for _, r := range results {
		fmt.Fprintln(out, r.Direction, r.Source.Path)
	}
	return err
}
