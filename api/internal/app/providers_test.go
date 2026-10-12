package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// chatServer is a fake OpenAI-compatible server (like Ollama) replaying replies.
type chatServer struct {
	mu       sync.Mutex
	replies  []string
	requests []map[string]any
	auth     []string
}

func (c *chatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, _ := io.ReadAll(r.Body)
	var req map[string]any
	_ = json.Unmarshal(b, &req)
	c.requests = append(c.requests, req)
	c.auth = append(c.auth, r.Header.Get("Authorization"))
	if r.URL.Path != "/v1/chat/completions" || len(c.replies) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(c.replies[0]))
	c.replies = c.replies[1:]
}

func chatToolCall(name, args string) string {
	b, _ := json.Marshal(args)
	return `{"model":"llama3.1:8b","choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,
		"tool_calls":[{"id":"call_` + name + `","type":"function","function":{"name":"` + name + `","arguments":` + string(b) + `}}]}}],
		"usage":{"prompt_tokens":500,"completion_tokens":40}}`
}

func chatText(text string) string {
	return `{"model":"llama3.1:8b","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"` + text + `"}}],
		"usage":{"prompt_tokens":600,"completion_tokens":20}}`
}

func TestAgentsRunOnAnOpenAICompatibleLocalModel(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()
	task := alice.createTask(`{"title":"Renew passport"}`)

	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/agent-provider", nil), []byte(`"settings":null`)) {
		t.Fatal("expected the server default")
	}
	chat := &chatServer{}
	srv := httptest.NewServer(chat)
	defer srv.Close()

	body := `{"provider":"local","model":"llama3.1:8b","base_url":"` + srv.URL + `/v1","input_price":0,"output_price":0}`
	alice.mustDo(200, "PUT", "/v1/me/agent-provider", []byte(body))
	chat.replies = []string{chatText("OK")}
	if !bytes.Contains(alice.mustDo(200, "POST", "/v1/me/agent-provider/test", nil), []byte(`"ok":true`)) {
		t.Fatal("connection test failed")
	}

	chat.replies = []string{
		chatToolCall("list_tasks", `{"status":"inbox"}`),
		chatToolCall("update_task", `{"id":"`+task.ID+`","status":"next","context":"@errands","reason":"Needs a trip."}`),
		chatText("I proposed one change."),
	}
	before := snapshot(t)
	r := alice.runAgent("task-manager", "Triage")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 || r.Run.CostMicros != 0 || r.Reply == nil {
		t.Fatalf("local run = %+v", r)
	}
	if snapshot(t) != before {
		t.Fatal("a local model changed data without approval")
	}
	var rec struct {
		Run struct {
			Model string `json:"model"`
		} `json:"run"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/agent-runs/"+r.Run.ID, nil), &rec)
	if rec.Run.Model != "local/llama3.1:8b" {
		t.Fatalf("model recorded = %s", rec.Run.Model)
	}
	// The wire format was translated both ways, with no key sent.
	sent, _ := json.Marshal(chat.requests[2])
	for _, want := range []string{`"role":"system"`, `"role":"tool"`, `"tool_call_id":"call_list_tasks"`, `"name":"update_task"`} {
		if !bytes.Contains(sent, []byte(want)) {
			t.Errorf("request lacks %s: %s", want, sent)
		}
	}
	for _, a := range chat.auth {
		if a != "" {
			t.Fatal("an Authorization header was sent to a keyless local server")
		}
	}
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
}

func TestProviderKeysAreSealedAndNeverReturned(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()

	out := alice.mustDo(200, "PUT", "/v1/me/agent-provider", []byte(`{"provider":"mistral","model":"mistral-large-latest","api_key":"sk-secret-123"}`))
	if bytes.Contains(out, []byte("sk-secret")) || !bytes.Contains(out, []byte(`"has_key":true`)) {
		t.Fatalf("response = %s", out)
	}
	var sealed []byte
	if err := pool.QueryRow(context.Background(), `SELECT api_key_sealed FROM agent_providers`).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("sk-secret")) {
		t.Fatal("key stored in plain text")
	}
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/me/export", nil), []byte("sk-secret")) {
		t.Fatal("key in export")
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/agents", nil), []byte(`"provider":"mistral"`)) {
		t.Fatal("status does not name the provider")
	}
	// Omitting the key keeps it; an empty key removes it.
	out = alice.mustDo(200, "PUT", "/v1/me/agent-provider", []byte(`{"provider":"mistral","model":"mistral-small-latest"}`))
	if !bytes.Contains(out, []byte(`"has_key":true`)) {
		t.Fatalf("key dropped: %s", out)
	}
	out = alice.mustDo(200, "PUT", "/v1/me/agent-provider", []byte(`{"provider":"mistral","model":"mistral-small-latest","api_key":""}`))
	if !bytes.Contains(out, []byte(`"has_key":false`)) {
		t.Fatalf("key kept: %s", out)
	}
	// A cloud provider without a key is not usable yet.
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/agents", nil), []byte(`"reason":"not_configured"`)) {
		t.Fatal("keyless mistral should not be available")
	}
	for _, bad := range []string{
		`{"provider":"skynet","model":"x"}`,
		`{"provider":"local","model":"x"}`,
		`{"provider":"local","model":"x","base_url":"file:///etc/passwd"}`,
		`{"provider":"openai","model":"x","base_url":"http://evil.example"}`,
		`{"provider":"openai","model":"has space"}`,
	} {
		alice.mustDo(422, "PUT", "/v1/me/agent-provider", []byte(bad))
	}
	var tok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"s","scope":"write","expires_in_days":1}`)), &tok)
	if code, _ := e.client().do("GET", "/v1/me/agent-provider", nil, "Authorization", "Bearer "+tok.Token); code != 403 {
		t.Fatalf("token read provider settings: %d", code)
	}
	alice.mustDo(204, "DELETE", "/v1/me/agent-provider", nil)
	if !strings.Contains(string(alice.mustDo(200, "GET", "/v1/me/agent-provider", nil)), `"settings":null`) {
		t.Fatal("not reverted to the server default")
	}
}
