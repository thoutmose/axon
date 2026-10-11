package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func (c *client) putYAML(doc string) (int, []byte) {
	c.e.t.Helper()
	return c.do("PUT", "/v1/me/settings.yaml", []byte(doc), "Content-Type", "application/yaml")
}

func TestSettingsInTheAppAndAsYAML(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	def := alice.mustDo(200, "GET", "/v1/me/settings", nil)
	if !bytes.Contains(def, []byte(`"keyboard":{"shortcuts":true,"vim":false}`)) {
		t.Fatalf("keyboard defaults = %s", def)
	}
	if !bytes.Contains(def, []byte(`"language":"en"`)) || !bytes.Contains(def, []byte(`"theme":"system"`)) || !bytes.Contains(def, []byte(`"monthly_budget_cents":200`)) {
		t.Fatalf("defaults = %s", def)
	}
	doc := `{"language":"fr","appearance":{"theme":"dark","font":"serif","font_size":"large","icons":"symbols","motion":"reduced"},
		"templates":{"note":"# {title}\n\n## Sources\n","project":{"description":"Why, what, done when.","wip_limit":3,"docs_folder":"1-Projects/{slug}"},"resource":{"section":"To read","kind":"soft"}}}`
	alice.mustDo(200, "PUT", "/v1/me/settings", []byte(doc))
	if out := alice.mustDo(200, "PUT", "/v1/me/settings", []byte(strings.Replace(doc, `"templates":`, `"keyboard":{"shortcuts":false,"vim":true},"templates":`, 1))); !bytes.Contains(out, []byte(`"vim":true`)) {
		t.Fatalf("keyboard not saved: %s", out)
	}
	if code, out := alice.do("PUT", "/v1/me/settings", []byte(strings.Replace(doc, `"dark"`, `"neon"`, 1))); code != 422 || !bytes.Contains(out, []byte("appearance.theme")) {
		t.Fatalf("bad theme = %d %s", code, out)
	}

	// A stored key never leaves through the YAML export.
	alice.mustDo(200, "PUT", "/v1/me/agent-provider", []byte(`{"provider":"anthropic","model":"claude-opus-5-5","api_key":"sk-very-secret"}`))
	code, y := alice.do("GET", "/v1/me/settings.yaml", nil)
	if code != 200 || bytes.Contains(y, []byte("sk-very-secret")) || bytes.Contains(y, []byte("api_key:")) ||
		!bytes.Contains(y, []byte("language: fr")) || !bytes.Contains(y, []byte("version: 1")) || !bytes.Contains(y, []byte("model: claude-opus-5-5")) {
		t.Fatalf("yaml export = %d\n%s", code, y)
	}
	// It round-trips without loss.
	before := alice.mustDo(200, "GET", "/v1/me/settings", nil)
	if code, out := alice.putYAML(string(y)); code != 200 {
		t.Fatalf("round trip = %d %s", code, out)
	}
	if after := alice.mustDo(200, "GET", "/v1/me/settings", nil); !bytes.Equal(before, after) {
		t.Fatalf("round trip changed settings:\n%s\n%s", before, after)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/me/agent-provider", nil), []byte(`"has_key":true`)) {
		t.Fatal("re-importing the file dropped the key")
	}

	// Editing the file changes everything in one go, including a new key.
	edited := strings.NewReplacer("language: fr", "language: en", "monthly_budget_cents: 200", "monthly_budget_cents: 500",
		"provider: anthropic", "provider: openai", "model: claude-opus-5-5", "model: gpt-5\n    api_key: sk-new").Replace(string(y))
	if code, out := alice.putYAML(edited); code != 200 || !bytes.Contains(out, []byte(`"language":"en"`)) || !bytes.Contains(out, []byte(`"monthly_budget_cents":500`)) {
		t.Fatalf("edited = %d %s\n%s", code, out, edited)
	}
	if p := alice.mustDo(200, "GET", "/v1/me/agent-provider", nil); !bytes.Contains(p, []byte(`"provider":"openai"`)) || !bytes.Contains(p, []byte(`"has_key":true`)) {
		t.Fatalf("provider = %s", p)
	}

	// Errors point to the line.
	for _, c := range []struct{ doc, want string }{
		{"version: 1\nlanguage: en\nappearance:\n  theme: neon\n", "line 4: appearance.theme"},
		{"version: 1\nlanguage: klingon\n", "line 2: language"},
		{"version: 2\n", "line 1: version must be 1"},
		{"version: 1\ncolour: red\n", "colour"},
		{"version: 1\nlanguage: [\n", "not valid"},
		{"version: 1\nai:\n  provider:\n    provider: skynet\n    model: x\n", "line 2: ai: Choose a provider"},
	} {
		code, out := alice.putYAML(c.doc)
		var p struct {
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(out, &p)
		if code != 422 || !strings.Contains(p.Detail, c.want) {
			t.Errorf("%q = %d %q, want %q", c.doc, code, p.Detail, c.want)
		}
	}

	// Settings are for the person at the browser, not for tokens.
	var tok struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"s","scope":"write","expires_in_days":1}`)), &tok)
	if code, _ := e.client().do("GET", "/v1/me/settings", nil, "Authorization", "Bearer "+tok.Token); code != 403 {
		t.Fatalf("token read settings: %d", code)
	}
}
