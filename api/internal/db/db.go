// Package db owns the database schema and its migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"

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

	migrations, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, conn, migrations)
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
