package db

import (
	"context"
	"io"
	"os"
	"testing"
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
