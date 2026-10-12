package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Stand-ins for the domain types later Epics add.
func init() {
	export := func(context.Context, db.Querier, uuid.UUID) (any, error) { return []string{}, nil }
	graph.RegisterType(graph.Type{Name: "test_note", Table: "test_notes", Export: export})
	graph.RegisterType(graph.Type{Name: "test_goal", Table: "test_goals", Export: export})
	graph.RegisterRelation(graph.Relation{Name: "test_supports", Sources: []string{"test_note"}, Targets: []string{"test_goal"}})
}

func userID(t *testing.T, c *client) uuid.UUID {
	t.Helper()
	var me struct {
		ID uuid.UUID `json:"id"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me", nil), &me)
	return me.ID
}

func entity(t *testing.T, owner uuid.UUID, typ, title string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pgx.BeginFunc(context.Background(), pool, func(tx pgx.Tx) error {
		var err error
		if id, err = graph.CreateEntity(context.Background(), tx, owner, graph.UserActor, typ, title, ""); err != nil {
			return err
		}
		_, err = tx.Exec(context.Background(), `INSERT INTO `+map[string]string{"test_note": "test_notes", "test_goal": "test_goals"}[typ]+` (id) VALUES ($1)`, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestGraphEndpoints(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	u := userID(t, alice)
	note := entity(t, u, "test_note", "Mnemonic techniques")
	goal := entity(t, u, "test_goal", "Pass JLPT N4")

	alice.mustDo(200, "GET", "/v1/relations", nil)
	alice.mustDo(200, "GET", "/v1/entities/"+note.String(), nil)
	alice.mustDo(404, "GET", "/v1/entities/"+uuid.Must(uuid.NewV7()).String(), nil)

	link := []byte(`{"source_id":"` + note.String() + `","relation":"test_supports","target_id":"` + goal.String() + `"}`)
	var created struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/links", link), &created)
	alice.mustDo(409, "POST", "/v1/links", link)
	alice.mustDo(422, "POST", "/v1/links", []byte(`{"source_id":"`+goal.String()+`","relation":"test_supports","target_id":"`+note.String()+`"}`))

	back := alice.mustDo(200, "GET", "/v1/entities/"+goal.String()+"/links?direction=incoming&limit=1", nil)
	if !bytes.Contains(back, []byte(note.String())) || !bytes.Contains(back, []byte(`"next_cursor":"`)) {
		t.Fatalf("backlinks = %s", back)
	}
	alice.mustDo(400, "GET", "/v1/entities/"+goal.String()+"/links?limit=1000", nil)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/search?q=mnemonic", nil), []byte("Mnemonic techniques")) {
		t.Fatal("search missed the note")
	}
	alice.mustDo(204, "DELETE", "/v1/links/"+created.ID, nil)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+note.String()+"/timeline", nil), []byte(`"unlinked"`)) {
		t.Fatal("timeline lacks the unlink")
	}
}

func TestIdempotencyKeyReplaysAndRejectsReuse(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	u := userID(t, alice)
	a, b, c := entity(t, u, "test_note", "A"), entity(t, u, "test_note", "B"), entity(t, u, "test_note", "C")
	body := []byte(`{"source_id":"` + a.String() + `","relation":"relates_to","target_id":"` + b.String() + `"}`)

	code1, first := alice.do("POST", "/v1/links", body, "Idempotency-Key", "k1")
	code2, again := alice.do("POST", "/v1/links", body, "Idempotency-Key", "k1")
	if code1 != 201 || code2 != 201 || !bytes.Equal(first, again) {
		t.Fatalf("replay: %d %s / %d %s", code1, first, code2, again)
	}
	other := []byte(`{"source_id":"` + a.String() + `","relation":"relates_to","target_id":"` + c.String() + `"}`)
	if code, _ := alice.do("POST", "/v1/links", other, "Idempotency-Key", "k1"); code != 422 {
		t.Fatalf("key reuse with a different body = %d, want 422", code)
	}
}

func TestETagPreventsLostUpdates(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	req := func(etag string) (int, string) {
		code, body := alice.do("PATCH", "/v1/me", []byte(`{"display_name":"Alice B."}`), "If-Match", etag)
		return code, string(body)
	}
	if code, _ := req(""); code != 428 {
		t.Fatalf("PATCH without If-Match = %d, want 428", code)
	}
	etag := alice.etag("/v1/me")
	if code, body := req(etag); code != 204 {
		t.Fatalf("PATCH with current ETag = %d %s", code, body)
	}
	if code, _ := req(etag); code != 412 {
		t.Fatalf("PATCH with stale ETag = %d, want 412", code)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me", nil), []byte("Alice B.")) {
		t.Fatal("rename not applied")
	}
}

func TestPersonalAccessTokens(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	u := userID(t, alice)
	a, b := entity(t, u, "test_note", "A"), entity(t, u, "test_note", "B")

	mint := func(scope string) (string, string) {
		var res struct {
			Token string `json:"token"`
			Info  struct {
				ID string `json:"id"`
			} `json:"info"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"sync","scope":"`+scope+`","expires_in_days":30}`)), &res)
		return res.Token, res.Info.ID
	}
	readTok, readID := mint("read")
	writeTok, _ := mint("write")
	alice.mustDo(200, "GET", "/v1/me/tokens", nil)

	script := e.client() // no cookies, no Origin
	call := func(tok, method, path string, body []byte) int {
		code, _ := script.do(method, path, body, "Authorization", "Bearer "+tok, "Origin", "")
		return code
	}
	link := []byte(`{"source_id":"` + a.String() + `","relation":"relates_to","target_id":"` + b.String() + `"}`)
	if call(readTok, "GET", "/v1/entities/"+a.String(), nil) != 200 {
		t.Fatal("read token cannot read")
	}
	if call(readTok, "POST", "/v1/links", link) != 403 {
		t.Fatal("read token could write")
	}
	if call(writeTok, "POST", "/v1/links", link) != 201 {
		t.Fatal("write token cannot write")
	}
	for _, path := range []string{"/v1/me", "/v1/me/tokens", "/v1/me/export"} {
		if code := call(writeTok, "GET", path, nil); code != 403 {
			t.Fatalf("token reached %s: %d", path, code)
		}
	}
	if call("axon_forged", "GET", "/v1/entities/"+a.String(), nil) != 401 {
		t.Fatal("forged token accepted")
	}
	alice.mustDo(204, "DELETE", "/v1/me/tokens/"+readID, nil)
	if call(readTok, "GET", "/v1/entities/"+a.String(), nil) != 401 {
		t.Fatal("revoked token still works")
	}
}

func TestSpecIsServed(t *testing.T) {
	e := newEnv(t, false)
	if !bytes.Contains(e.client().mustDo(200, "GET", "/v1/openapi.yaml", nil), []byte("openapi: 3.1.0")) {
		t.Fatal("spec not served")
	}
	e.client().mustDo(200, "GET", "/v1/auth/status", nil)
}

func TestAddAndRemovePasskeyAndReissueCodes(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	first := alice.register("Alice")

	phone := &softAuthenticator{t: t, origin: origin}
	opts := alice.mustDo(200, "POST", "/v1/me/passkeys/begin", []byte(`{"name":"Phone"}`))
	alice.mustDo(201, "POST", "/v1/me/passkeys/finish", phone.create(opts))
	keys := ids(t, alice.mustDo(200, "GET", "/v1/me/passkeys", nil))
	if len(keys) != 2 {
		t.Fatalf("passkeys = %d, want 2", len(keys))
	}
	alice.mustDo(204, "DELETE", "/v1/me/passkeys/"+keys[0], nil)

	// The phone's passkey now signs in on its own.
	other := e.client()
	other.key = phone
	other.signIn()

	var res struct {
		Codes []string `json:"recovery_codes"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/me/recovery-codes", nil), &res)
	if len(res.Codes) != 10 || res.Codes[0] == first[0] {
		t.Fatal("recovery codes not reissued")
	}
	e.client().mustDo(401, "POST", "/v1/auth/recovery", []byte(`{"code":"`+first[0]+`"}`))
}
