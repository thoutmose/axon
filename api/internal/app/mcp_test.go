package app

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
)

// Claude Code on Axon (EP-169): an MCP client with a personal access token
// lists the agents' tools and prompts, reads framed as data, and writes
// only proposals; nothing without the AI agents consent.
func TestClaudeCodeUsesTheAgentsToolsThroughMCP(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.createTask(`{"title":"Call the plumber","status":"next"}`)
	mint := func(scope string) string {
		var res struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"claude","scope":"`+scope+`","expires_in_days":30}`)), &res)
		return res.Token
	}
	tok := mint("write")
	claude := e.client() // no cookies, no Origin
	rpc := func(id int, method, params string) map[string]any {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"` + method + `","params":` + params + `}`
		code, out := claude.do("POST", "/v1/mcp", []byte(body), "Authorization", "Bearer "+tok, "Origin", "")
		if code != 200 {
			t.Fatalf("%s = %d: %s", method, code, out)
		}
		var m map[string]any
		if err := json.Unmarshal(out, &m); err != nil {
			t.Fatalf("%s: %v: %s", method, err, out)
		}
		return m
	}

	if m := rpc(1, "tools/list", `{}`); m["error"] == nil {
		t.Fatalf("tools without the AI agents consent: %v", m)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))

	init := rpc(1, "initialize", `{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-code","version":"2"}}`)
	if res, _ := init["result"].(map[string]any); res["protocolVersion"] != "2025-06-18" {
		t.Fatalf("initialize = %v", init)
	}
	if code, _ := claude.do("POST", "/v1/mcp", []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`), "Authorization", "Bearer "+tok, "Origin", ""); code != 202 {
		t.Fatalf("notification = %d", code)
	}

	list, _ := json.Marshal(rpc(2, "tools/list", `{}`))
	for _, want := range []string{`"name":"list_tasks"`, `"name":"create_task"`, `"readOnlyHint":true`} {
		if !bytes.Contains(list, []byte(want)) {
			t.Errorf("tools lack %s", want)
		}
	}
	// Jarvis's tool would call the server's provider; health tools need their consents.
	for _, absent := range []string{`"name":"ask_agent"`, `"name":"food_log"`} {
		if bytes.Contains(list, []byte(absent)) {
			t.Errorf("tools offer %s", absent)
		}
	}

	read, _ := json.Marshal(rpc(3, "tools/call", `{"name":"list_tasks","arguments":{}}`))
	if !bytes.Contains(read, []byte("Call the plumber")) || !bytes.Contains(read, []byte("\\u003cdata\\u003e") /* json.Marshal escapes < */) {
		t.Fatalf("read = %s", read)
	}
	write, _ := json.Marshal(rpc(4, "tools/call", `{"name":"create_task","arguments":{"title":"Book the boiler service","reason":"asked in Claude Code"}}`))
	if !bytes.Contains(write, []byte("waiting for the person's approval")) {
		t.Fatalf("write = %s", write)
	}
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/tasks?limit=100", nil), []byte("Book the boiler service")) {
		t.Fatal("a write changed data before approval")
	}
	var props struct {
		Items []struct {
			Agent  string `json:"agent"`
			Status string `json:"status"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/proposals?status=pending", nil), &props)
	if len(props.Items) != 1 || props.Items[0].Agent != "task-manager" {
		t.Fatalf("proposals = %+v", props.Items)
	}
	if runs := alice.mustDo(200, "GET", "/v1/agent-runs", nil); !bytes.Contains(runs, []byte("claude-code")) {
		t.Errorf("runs = %s", runs)
	}

	prompts, _ := json.Marshal(rpc(5, "prompts/list", `{}`))
	if !bytes.Contains(prompts, []byte(`"name":"task-manager"`)) || bytes.Contains(prompts, []byte(`"name":"jarvis"`)) {
		t.Errorf("prompts = %s", prompts)
	}
	got, _ := json.Marshal(rpc(6, "prompts/get", `{"name":"task-manager","arguments":{"request":"What is next?"}}`))
	if !bytes.Contains(got, []byte("What is next?")) || !bytes.Contains(got, []byte("Today is")) {
		t.Errorf("prompt = %.300s", got)
	}
	if m := rpc(7, "nope", `{}`); m["error"] == nil {
		t.Errorf("unknown method = %v", m)
	}

	// A read token cannot post, so it cannot speak MCP.
	if code, _ := claude.do("POST", "/v1/mcp", []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), "Authorization", "Bearer "+mint("read"), "Origin", ""); code != 403 {
		t.Errorf("read token = %d", code)
	}
}
