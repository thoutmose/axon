package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/server"
)

// These tests exercise the real API against PostgreSQL, e.g.
// AXON_TEST_DATABASE_URL=postgres://axon:local-dev-only@localhost:5432/axon_test?sslmode=disable

const origin = "http://localhost:3000"

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("AXON_TEST_DATABASE_URL")
	if dsn == "" {
		os.Exit(0) // integration tests only; unit tests live next to their packages
	}
	ctx := context.Background()
	if err := db.Migrate(ctx, dsn, "up", io.Discard); err != nil {
		panic(err)
	}
	var err error
	if pool, err = pgxpool.New(ctx, dsn); err != nil {
		panic(err)
	}
	for _, t := range []string{"test_notes", "test_goals"} {
		if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+t+` (id uuid PRIMARY KEY REFERENCES entities(id) ON DELETE CASCADE)`); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	_, _ = pool.Exec(ctx, `DROP TABLE test_notes, test_goals`)
	pool.Close()
	os.Exit(code)
}

type env struct {
	t   *testing.T
	srv *httptest.Server
}

func newEnv(t *testing.T, openRegistration bool) *env {
	t.Helper()
	return newEnvWith(t, auth.Config{OpenRegistration: openRegistration})
}

// newEnvWith starts the API with cfg; the URL, RP ID, and key are filled in.
func newEnvWith(t *testing.T, cfg auth.Config) *env {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `TRUNCATE users, webauthn_ceremonies CASCADE`); err != nil {
		t.Fatal(err)
	}
	cfg.BaseURL, cfg.RPID, cfg.EncryptionKey = origin, "localhost", strings.Repeat("ab", 32)
	h, err := Handler("test", cfg, pool, slog.New(slog.NewTextHandler(io.Discard, nil)), server.NewMetrics(prometheus.NewRegistry(), "test", pool), fake)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv}
}

// client is one browser: its own cookie jar and passkey.
type client struct {
	e    *env
	http *http.Client
	key  *softAuthenticator
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar}, key: &softAuthenticator{t: e.t, origin: origin}}
}

func (c *client) do(method, path string, body []byte, headers ...string) (int, []byte) {
	c.e.t.Helper()
	req, _ := http.NewRequest(method, c.e.srv.URL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	for i := 0; i+1 < len(headers); i += 2 {
		if headers[i+1] == "" {
			req.Header.Del(headers[i])
		} else {
			req.Header.Set(headers[i], headers[i+1])
		}
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	out, _ := io.ReadAll(res.Body)
	checkContract(c.e.t, req, body, res, out)
	return res.StatusCode, out
}

func (c *client) mustDo(want int, method, path string, body []byte) []byte {
	c.e.t.Helper()
	code, out := c.do(method, path, body)
	if code != want {
		c.e.t.Fatalf("%s %s = %d, want %d: %s", method, path, code, want, out)
	}
	return out
}

func (c *client) etag(path string) string {
	c.e.t.Helper()
	req, _ := http.NewRequest("GET", c.e.srv.URL+path, nil)
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.Header.Get("ETag")
}

func (c *client) register(name string) []string {
	c.e.t.Helper()
	opts := c.mustDo(200, "POST", "/v1/auth/register/begin", []byte(`{"display_name":"`+name+`"}`))
	out := c.mustDo(201, "POST", "/v1/auth/register/finish", c.key.create(opts))
	var res struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	_ = json.Unmarshal(out, &res)
	return res.RecoveryCodes
}

func (c *client) signIn() {
	c.e.t.Helper()
	opts := c.mustDo(200, "POST", "/v1/auth/login/begin", nil)
	c.mustDo(200, "POST", "/v1/auth/login/finish", c.key.get(opts))
}

func (c *client) signOut() { c.mustDo(204, "POST", "/v1/auth/logout", nil) }

func ids(t *testing.T, body []byte) []string {
	t.Helper()
	var res struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(res.Items))
	for i, it := range res.Items {
		out[i] = it.ID
	}
	return out
}

func TestRegisterSignInAndFirstUserOnly(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	codes := alice.register("Alice")
	if len(codes) != 10 {
		t.Fatalf("got %d recovery codes, want 10", len(codes))
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me", nil), []byte(`"display_name":"Alice"`)) {
		t.Fatal("/v1/me does not show the new account")
	}

	alice.signOut()
	alice.mustDo(401, "GET", "/v1/me", nil)
	alice.signIn()
	alice.mustDo(200, "GET", "/v1/me", nil)

	// Registration closes after the first account.
	if code, _ := e.client().do("POST", "/v1/auth/register/begin", []byte(`{"display_name":"Mallory"}`)); code != 403 {
		t.Fatalf("second registration = %d, want 403", code)
	}
}

func TestPasskeyFromAnotherBrowserIsRejected(t *testing.T) {
	e := newEnv(t, false)
	e.client().register("Alice")
	stranger := e.client()
	stranger.key.create([]byte(`{"publicKey":{"challenge":"x","rp":{"id":"localhost"},"user":{"id":"AAAAAAAAAAAAAAAAAAAAAA"}}}`))
	opts := stranger.mustDo(200, "POST", "/v1/auth/login/begin", nil)
	stranger.mustDo(401, "POST", "/v1/auth/login/finish", stranger.key.get(opts))
}

func TestRecoveryCodesWorkOnceAndRequireTOTPWhenEnabled(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	codes := alice.register("Alice")

	other := e.client()
	other.mustDo(200, "POST", "/v1/auth/recovery", []byte(`{"code":"`+strings.ToUpper(codes[0])+`"}`))
	other.mustDo(200, "GET", "/v1/me", nil)
	e.client().mustDo(401, "POST", "/v1/auth/recovery", []byte(`{"code":"`+codes[0]+`"}`))

	// Turn on TOTP.
	var setup struct {
		Secret string `json:"secret"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/me/totp/begin", nil), &setup)
	alice.mustDo(422, "POST", "/v1/me/totp/confirm", []byte(`{"code":"000000"}`))
	now, _ := totp.GenerateCode(setup.Secret, time.Now())
	alice.mustDo(204, "POST", "/v1/me/totp/confirm", []byte(`{"code":"`+now+`"}`))

	e.client().mustDo(401, "POST", "/v1/auth/recovery", []byte(`{"code":"`+codes[1]+`"}`))
	// The code used to turn TOTP on cannot be replayed.
	e.client().mustDo(401, "POST", "/v1/auth/recovery", []byte(`{"code":"`+codes[1]+`","totp":"`+now+`"}`))
	next, _ := totp.GenerateCode(setup.Secret, time.Now().Add(30*time.Second))
	e.client().mustDo(200, "POST", "/v1/auth/recovery", []byte(`{"code":"`+codes[1]+`","totp":"`+next+`"}`))

	// Turning it off needs an unused code too. Rewind the recorded step
	// instead of waiting 30 seconds for the next one.
	alice.mustDo(422, "POST", "/v1/me/totp/disable", []byte(`{"code":"`+next+`"}`))
	if _, err := pool.Exec(context.Background(), `UPDATE users SET totp_last_step = 0`); err != nil {
		t.Fatal(err)
	}
	alice.mustDo(204, "POST", "/v1/me/totp/disable", []byte(`{"code":"`+next+`"}`))
}

func TestCrossSiteRequestsAreRejected(t *testing.T) {
	e := newEnv(t, false)
	c := e.client()
	if code, _ := c.do("POST", "/v1/auth/register/begin", []byte(`{"display_name":"A"}`), "Origin", "https://evil.example"); code != 403 {
		t.Fatalf("foreign origin = %d, want 403", code)
	}
	if code, _ := c.do("POST", "/v1/auth/register/begin", []byte(`{"display_name":"A"}`), "Origin", ""); code != 403 {
		t.Fatalf("missing origin = %d, want 403", code)
	}
	if code, _ := c.do("POST", "/v1/auth/register/begin", []byte(`{"display_name":"A"}`), "Origin", "", "Sec-Fetch-Site", "same-origin"); code != 200 {
		t.Fatalf("same-origin without Origin = %d, want 200", code)
	}
}

// Every owned resource must be invisible and immutable to other users.
func TestUsersCannotSeeOrChangeEachOthersData(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	aliceSessions := ids(t, alice.mustDo(200, "GET", "/v1/me/sessions", nil))
	alicePasskeys := ids(t, alice.mustDo(200, "GET", "/v1/me/passkeys", nil))

	for _, path := range []string{"/v1/me/sessions", "/v1/me/passkeys", "/v1/me/audit"} {
		for _, id := range ids(t, bob.mustDo(200, "GET", path, nil)) {
			for _, mine := range append(aliceSessions, alicePasskeys...) {
				if id == mine {
					t.Fatalf("Bob sees Alice's %s %s", path, id)
				}
			}
		}
	}
	bob.mustDo(404, "DELETE", "/v1/me/sessions/"+aliceSessions[0], nil)
	bob.mustDo(404, "DELETE", "/v1/me/passkeys/"+alicePasskeys[0], nil)
	alice.mustDo(200, "GET", "/v1/me", nil) // Alice is still signed in
}

func TestSignOutEverywhere(t *testing.T) {
	e := newEnv(t, false)
	laptop := e.client()
	laptop.register("Alice")
	phone := &client{e: e, key: laptop.key}
	jar, _ := cookiejar.New(nil)
	phone.http = &http.Client{Jar: jar}
	phone.signIn()

	phone.mustDo(204, "POST", "/v1/me/logout-all", nil)
	laptop.mustDo(401, "GET", "/v1/me", nil)
	phone.mustDo(401, "GET", "/v1/me", nil)
}

func TestLastPasskeyCannotBeRemoved(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	id := ids(t, alice.mustDo(200, "GET", "/v1/me/passkeys", nil))[0]
	alice.mustDo(409, "DELETE", "/v1/me/passkeys/"+id, nil)
}

func TestConsentIsOffByDefaultAndAudited(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/me/consents", nil), []byte(`"granted":true`)) {
		t.Fatal("a consent is on by default")
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))
	alice.mustDo(404, "PUT", "/v1/me/consents/sell_my_data", []byte(`{"granted":true}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/audit", nil), []byte(`"consent.changed"`)) {
		t.Fatal("consent change not audited")
	}
}

func TestExportIsCompleteAndErasureLeavesNothing(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))

	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			r, _ := f.Open()
			_ = json.NewDecoder(r).Decode(&data)
		}
	}
	for _, section := range []string{"account", "passkeys", "sessions", "consents", "audit_events", "agents"} {
		if len(data[section]) == 0 {
			t.Fatalf("export lacks %q", section)
		}
	}
	if bytes.Contains(data["passkeys"], []byte("publicKey")) {
		t.Fatal("export leaks credential key material")
	}

	var userID string
	_ = pool.QueryRow(context.Background(), `SELECT id::text FROM users`).Scan(&userID)
	alice.mustDo(400, "DELETE", "/v1/me", []byte(`{"confirm":"yes"}`))
	alice.mustDo(204, "DELETE", "/v1/me", []byte(`{"confirm":"DELETE"}`))
	alice.mustDo(401, "GET", "/v1/me", nil)

	// Verify, not assume: no table may still hold a row for the user.
	rows, err := pool.Query(context.Background(),
		`SELECT table_name, column_name FROM information_schema.columns
		 WHERE table_schema = 'public' AND column_name IN ('user_id', 'id') AND data_type = 'uuid'`)
	if err != nil {
		t.Fatal(err)
	}
	type col struct{ table, column string }
	var cols []col
	for rows.Next() {
		var c col
		_ = rows.Scan(&c.table, &c.column)
		cols = append(cols, c)
	}
	for _, c := range cols {
		var n int
		q := `SELECT count(*) FROM "` + c.table + `" WHERE "` + c.column + `"::text = $1`
		if err := pool.QueryRow(context.Background(), q, userID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s.%s still has %d row(s) for the erased user", c.table, c.column, n)
		}
	}
}

func TestAuditEventsAreAppendOnly(t *testing.T) {
	e := newEnv(t, false)
	e.client().register("Alice")
	if _, err := pool.Exec(context.Background(), `UPDATE audit_events SET action = 'forged'`); err == nil {
		t.Fatal("audit_events accepted an UPDATE")
	}
}

func TestSignInIsRateLimited(t *testing.T) {
	e := newEnv(t, false)
	c := e.client()
	limited := false
	for range 15 {
		if code, _ := c.do("POST", "/v1/auth/login/begin", nil); code == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("15 rapid sign-in attempts were all accepted")
	}
}
