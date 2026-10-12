package db

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestMigrateUpDown needs a disposable PostgreSQL with pgvector, e.g. the one
// from compose.yaml: AXON_TEST_DATABASE_URL=postgres://... go test ./...
func TestMigrateUpDown(t *testing.T) {
	dsn := os.Getenv("AXON_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AXON_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	for _, step := range []string{"up", "down", "up"} {
		if err := Migrate(ctx, dsn, step, io.Discard); err != nil {
			t.Fatalf("migrate %s: %v", step, err)
		}
	}
}

func TestMigrateRejectsUnknownDirection(t *testing.T) {
	err := Migrate(context.Background(), "postgres://unused", "sideways", io.Discard)
	if err == nil {
		t.Fatal("want error for unknown direction")
	}
}

func TestLatestIsTheNewestMigration(t *testing.T) {
	v, err := Latest()
	if err != nil || v < 31 {
		t.Fatalf("Latest() = %d, %v; want at least 31", v, err)
	}
}

// Every foreign key has an index that starts with its columns. Without one,
// deleting a parent row (an erasure, a deleted note, a calendar refresh)
// scans the whole child table, for every user, and so does a lookup by it.
func TestEveryForeignKeyIsIndexed(t *testing.T) {
	dsn := os.Getenv("AXON_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AXON_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := Migrate(ctx, dsn, "up", io.Discard); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	rows, err := conn.Query(ctx, `SELECT c.conrelid::regclass || '(' || (SELECT string_agg(attname, ', ' ORDER BY k.n)
			FROM unnest(c.conkey) WITH ORDINALITY k(attnum, n) JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum) || ')'
		FROM pg_constraint c
		WHERE c.contype = 'f' AND NOT EXISTS (SELECT 1 FROM pg_index i WHERE i.indrelid = c.conrelid
			AND (i.indkey::int2[])[0:cardinality(c.conkey) - 1] @> c.conkey AND (i.indkey::int2[])[0:cardinality(c.conkey) - 1] <@ c.conkey)
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) > 0 {
		t.Errorf("%d foreign keys have no index starting with their columns:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}
