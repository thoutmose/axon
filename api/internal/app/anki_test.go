package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeBridge stands in for the Anki bridge and records what it received.
type fakeBridge struct {
	mu    sync.Mutex
	calls []map[string]string
	fail  bool
}

func (f *fakeBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	f.calls = append(f.calls, body)
	fail := f.fail
	f.mu.Unlock()
	if fail {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"401 Unauthorized"}`))
		return
	}
	_, _ = w.Write([]byte(`{"added":1,"updated":0,"unchanged":0}`))
}

func (f *fakeBridge) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestAnkiSyncSendsCardsAndRunsAfterApprovedTeacherCards(t *testing.T) {
	bridge := &fakeBridge{}
	srv := httptest.NewServer(bridge)
	defer srv.Close()
	t.Setenv("AXON_ANKI_BRIDGE_URL", srv.URL)
	t.Setenv("ANKI_SYNC_URL", "http://srv-services:8080/")
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	if out := alice.mustDo(200, "GET", "/v1/me/anki", nil); !bytes.Contains(out, []byte(`"available":true`)) || !bytes.Contains(out, []byte(`"account":null`)) {
		t.Fatalf("anki = %s", out)
	}
	alice.mustDo(409, "POST", "/v1/me/anki/sync", nil)
	alice.mustDo(422, "PUT", "/v1/me/anki", []byte(`{"endpoint":"ftp://x","username":"a","password":"p"}`))
	alice.mustDo(422, "PUT", "/v1/me/anki", []byte(`{"endpoint":"http://srv-services:8080/","username":"alice"}`)) // no password yet
	out := alice.mustDo(200, "PUT", "/v1/me/anki", []byte(`{"endpoint":"http://srv-services:8080/","username":"alice","password":"s3cret-anki"}`))
	if bytes.Contains(out, []byte("s3cret")) {
		t.Fatal("the password came back")
	}
	// Changing settings without a password keeps the stored one.
	alice.mustDo(200, "PUT", "/v1/me/anki", []byte(`{"endpoint":"http://srv-services:8080/","username":"alice","auto_sync":true}`))

	alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Capital of France?","back":"Paris"}`))
	res := alice.mustDo(200, "POST", "/v1/me/anki/sync", nil)
	if !bytes.Contains(res, []byte(`"added":1`)) || bridge.count() != 1 {
		t.Fatalf("sync = %s, bridge calls %d", res, bridge.count())
	}
	call := bridge.calls[0]
	pkg, _ := base64.StdEncoding.DecodeString(call["apkg"])
	if call["username"] != "alice" || call["password"] != "s3cret-anki" || call["endpoint"] != "http://srv-services:8080/" || !bytes.HasPrefix(pkg, []byte("PK")) {
		t.Fatalf("bridge got %v (package %d bytes)", map[string]string{"username": call["username"], "endpoint": call["endpoint"]}, len(pkg))
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/anki", nil), []byte(`"last_status":"ok"`)) {
		t.Fatal("status not recorded")
	}
	// Failures are recorded for the person and returned as 502.
	bridge.mu.Lock()
	bridge.fail = true
	bridge.mu.Unlock()
	if code, body := alice.do("POST", "/v1/me/anki/sync", nil); code != 502 || !bytes.Contains(body, []byte("401 Unauthorized")) {
		t.Fatalf("failed sync = %d %s", code, body)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/anki", nil), []byte(`"last_status":"error"`)) {
		t.Fatal("error not recorded")
	}
	bridge.mu.Lock()
	bridge.fail = false
	bridge.mu.Unlock()

	// Approving the Teacher's cards syncs automatically.
	alice.enableAgents()
	var note struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/permanent/rome.md","content":"---\ntype: permanent\n---\n# Rome"}`)), &note)
	fake.load(toolUse("propose_cards", `{"cards":[{"front":"Capital of Italy?","back":"Rome","note_id":"`+note.ID+`"}],"reason":"x"}`), finalText("Proposed."))
	r := alice.runAgent("teacher", "Cards please")
	before := bridge.count()
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	deadline := time.Now().Add(5 * time.Second)
	for bridge.count() == before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if bridge.count() != before+1 {
		t.Fatal("approving Teacher cards did not sync")
	}

	alice.mustDo(204, "DELETE", "/v1/me/anki", nil)
	alice.mustDo(409, "POST", "/v1/me/anki/sync", nil)
}
