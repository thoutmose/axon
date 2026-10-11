package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/server"
)

// previousRelease is the schema version of the release before 1.0 (v0.1.0,
// on main), which production runs.
const previousRelease = 5

// Upgrading from the previous release keeps every row: a database filled
// at v0.1.0's schema is migrated to the latest, passes the integrity check,
// and the person's API token and tasks work as before (EP-21).
func TestUpgradeFromThePreviousReleaseKeepsData(t *testing.T) {
	base := os.Getenv("AXON_TEST_DATABASE_URL")
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = admin.Close(ctx) }()
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS axon_upgrade WITH (FORCE)`); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE axon_upgrade`); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(base)
	u.Path = "/axon_upgrade"
	dsn := u.String()
	if err := db.MigrateTo(ctx, dsn, previousRelease, io.Discard); err != nil {
		t.Fatal(err)
	}
	old, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()

	// What v0.1.0 wrote: an account, a passkey, a session, an API token,
	// tasks (one done, one recurring), a link between them, and events.
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := "axon_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	user, task, done := uuid.New(), uuid.New(), uuid.New()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users (id, display_name) VALUES ($1, 'Ada')`, []any{user}},
		{`INSERT INTO passkeys (id, user_id, credential_id, credential, name) VALUES ($1, $2, '\x0102', '{"ID":"AQI="}', 'Laptop')`, []any{uuid.New(), user}},
		{`INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES ($1, $2, '\x0a0b', now() + interval '1 day')`, []any{uuid.New(), user}},
		{`INSERT INTO api_tokens (id, user_id, name, token_hash, scope, expires_at) VALUES ($1, $2, 'CLI', $3, 'write', now() + interval '30 days')`, []any{uuid.New(), user, sum[:]}},
		{`INSERT INTO entities (id, user_id, type, title) VALUES ($1, $3, 'task', 'Call the plumber'), ($2, $3, 'task', 'Water the plants')`, []any{task, done, user}},
		{`INSERT INTO tasks (id, status, context, due_on, recurrence) VALUES ($1, 'next', '@phone', current_date + 1, NULL)`, []any{task}},
		{`INSERT INTO tasks (id, status, completed_at, recurrence) VALUES ($1, 'done', now(), 'FREQ=WEEKLY;BYDAY=SA')`, []any{done}},
		{`INSERT INTO links (id, user_id, source_id, relation, target_id) VALUES ($1, $2, $3, 'related', $4)`, []any{uuid.New(), user, task, done}},
		{`INSERT INTO events (id, user_id, actor, entity_type, entity_id, kind) VALUES ($1, $2, 'user', 'task', $3, 'created')`, []any{uuid.New(), user, task}},
	} {
		if _, err := old.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seed v0.1.0 data: %v\n%s", err, stmt.sql)
		}
	}
	old.Close()

	if err := db.Migrate(ctx, dsn, "up", io.Discard); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	upgraded, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	// Test-only entity types registered by other tests in this package need their tables here too.
	for _, table := range []string{"test_notes", "test_goals"} {
		if _, err := upgraded.Exec(ctx, `CREATE TABLE `+table+` (id uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE)`); err != nil {
			t.Fatal(err)
		}
	}
	problems, err := graph.CheckIntegrity(ctx, upgraded)
	if err != nil || len(problems) != 0 {
		t.Fatalf("integrity after upgrade: %v %+v", err, problems)
	}
	var counts string
	if err := upgraded.QueryRow(ctx, `SELECT concat_ws('/', (SELECT count(*) FROM users), (SELECT count(*) FROM passkeys), (SELECT count(*) FROM sessions),
		(SELECT count(*) FROM api_tokens), (SELECT count(*) FROM tasks), (SELECT count(*) FROM links), (SELECT count(*) FROM events))`).Scan(&counts); err != nil {
		t.Fatal(err)
	}
	if counts != "1/1/1/1/2/1/1" {
		t.Fatalf("rows after upgrade = %s, want 1/1/1/1/2/1/1", counts)
	}

	// The new API serves the old data to the old token.
	h, err := Handler("upgrade", auth.Config{BaseURL: origin, RPID: "localhost", EncryptionKey: strings.Repeat("ab", 32)}, upgraded,
		slog.New(slog.NewTextHandler(io.Discard, nil)), server.NewMetrics(prometheus.NewRegistry(), "upgrade", upgraded), fake)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/tasks/"+task.String(), nil) //nolint:gosec // the test server started above
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // the test server started above
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var got struct {
		Title   string  `json:"title"`
		Context *string `json:"context"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if resp.StatusCode != 200 || got.Title != "Call the plumber" || got.Context == nil || *got.Context != "@phone" {
		t.Fatalf("GET task after upgrade = %d %+v", resp.StatusCode, got)
	}
}
