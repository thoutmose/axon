package instance_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/demo"
	"github.com/thoutmose/axon/api/internal/instance"
)

// Test keys, built so they are obviously not real ones.
var (
	key      = strings.Repeat("0123456789abcdef", 4)
	otherKey = strings.Repeat("fedcba9876543210", 4)
)

// fresh creates an empty database next to the test one and returns its DSN.
func fresh(t *testing.T, name string) string {
	t.Helper()
	base := os.Getenv("AXON_TEST_DATABASE_URL")
	if base == "" {
		t.Skip("AXON_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = admin.Close(ctx) }()
	ident := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+ident+" WITH (FORCE)"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+ident); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

func open(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func load(t *testing.T, dsn string, archive []byte, hexKey string) error {
	t.Helper()
	latest, err := db.Latest()
	if err != nil {
		t.Fatal(err)
	}
	_, err = instance.Import(context.Background(), open(t, dsn), bytes.NewReader(archive), hexKey, latest,
		func(ctx context.Context, v int64) error { return db.MigrateTo(ctx, dsn, v, io.Discard) })
	return err
}

// fingerprints hashes every table's rows, independent of physical order.
func fingerprints(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(ctx, "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'goose_db_version'")
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, n := range names {
		var sum string
		q := "SELECT count(*) || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '') FROM " + pgx.Identifier{n}.Sanitize() + " t"
		if err := pool.QueryRow(ctx, q).Scan(&sum); err != nil {
			t.Fatal(err)
		}
		out[n] = sum
	}
	return out
}

func users(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRestoreDrill(t *testing.T) {
	ctx := context.Background()
	srcDSN := fresh(t, "axon_instance_src")
	if err := db.Migrate(ctx, srcDSN, "up", io.Discard); err != nil {
		t.Fatal(err)
	}
	src := open(t, srcDSN)
	if _, err := demo.Reset(ctx, src, time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	m, err := instance.Export(ctx, src, &buf, "test", key)
	if err != nil {
		t.Fatal(err)
	}
	latest, _ := db.Latest()
	if m.SchemaVersion != latest || m.KeyFingerprint != instance.KeyFingerprint(key) || len(m.Tables) < 20 {
		t.Fatalf("manifest: schema %d (want %d), %d tables, key %q", m.SchemaVersion, latest, len(m.Tables), m.KeyFingerprint)
	}
	archive := buf.Bytes()

	// A fresh installation that already ran its migrations takes the archive
	// and ends up with exactly the same rows in every table.
	dstDSN := fresh(t, "axon_instance_dst")
	if err := db.Migrate(ctx, dstDSN, "up", io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := load(t, dstDSN, archive, key); err != nil {
		t.Fatal(err)
	}
	want, got := fingerprints(t, src), fingerprints(t, open(t, dstDSN))
	if len(want) != len(got) {
		t.Fatalf("%d tables after import, want %d", len(got), len(want))
	}
	for table, sum := range want {
		if got[table] != sum {
			t.Errorf("table %s: got %s, want %s", table, got[table], sum)
		}
	}
	if users(t, open(t, dstDSN)) == 0 {
		t.Fatal("no accounts after import")
	}

	// Never over existing data.
	if err := load(t, dstDSN, archive, key); err == nil || !strings.Contains(err.Error(), "fresh installation") {
		t.Fatalf("second import: %v, want a refusal", err)
	}

	// A different key would strand the sealed secrets: refused before anything changes.
	emptyDSN := fresh(t, "axon_instance_empty")
	if err := load(t, emptyDSN, archive, otherKey); err == nil || !strings.Contains(err.Error(), "AXON_ENCRYPTION_KEY") {
		t.Fatalf("other key: %v, want a refusal", err)
	}

	// A damaged table is caught by its checksum and nothing is kept. The
	// database starts with no schema at all, so this also migrates from zero.
	if err := load(t, emptyDSN, damage(t, archive), key); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Fatalf("damaged archive: %v, want a refusal", err)
	}
	if n := users(t, open(t, emptyDSN)); n != 0 {
		t.Fatalf("%d accounts kept from a damaged archive", n)
	}
}

// damage rewrites the archive with one character changed in the users table.
func damage(t *testing.T, archive []byte) []byte {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var out bytes.Buffer
	gw := gzip.NewWriter(&out)
	tw := tar.NewWriter(gw)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "tables/users.copy" {
			body = bytes.Replace(body, []byte("a"), []byte("b"), 1)
		}
		hdr.Size = int64(len(body))
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestKeyFingerprint(t *testing.T) {
	if instance.KeyFingerprint("") != "" {
		t.Fatal("no key, no fingerprint")
	}
	a, b := instance.KeyFingerprint(key), instance.KeyFingerprint(strings.ToUpper(key))
	if a != b || len(a) != 16 || strings.Contains(key, a) || a == instance.KeyFingerprint(otherKey) {
		t.Fatalf("fingerprints %q %q", a, b)
	}
}

// An archive from an older Axon goes into a newer installation: the schema is
// moved back to the archive's version for the copy, then migrated forward.
func TestImportFromAnOlderSchema(t *testing.T) {
	ctx := context.Background()
	srcDSN := fresh(t, "axon_instance_old")
	if err := db.MigrateTo(ctx, srcDSN, 29, io.Discard); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := instance.Export(ctx, open(t, srcDSN), &buf, "old", ""); err != nil {
		t.Fatal(err)
	}
	dstDSN := fresh(t, "axon_instance_new")
	if err := db.Migrate(ctx, dstDSN, "up", io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := load(t, dstDSN, buf.Bytes(), key); err != nil {
		t.Fatal(err)
	}
	var v int64
	if err := open(t, dstDSN).QueryRow(ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&v); err != nil || v != 29 {
		t.Fatalf("schema after import = %d, %v; want 29", v, err)
	}
	if err := db.Migrate(ctx, dstDSN, "up", io.Discard); err != nil {
		t.Fatal(err)
	}
}
