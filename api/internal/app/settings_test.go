package app

import (
	"bytes"
	"context"
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
	doc := `{"language":"fr","appearance":{"theme":"dark","font":"serif","font_size":"large","icons":"symbols","motion":"reduced","spaces":"off"},
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

// Today's sections are chosen and ordered from a known list, each once.
func TestTodaySections(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	put := func(sections string, want int) []byte {
		t.Helper()
		var doc map[string]any
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
		delete(doc, "ai")
		var secs []string
		_ = json.Unmarshal([]byte(sections), &secs)
		doc["today"] = map[string]any{"sections": secs}
		body, _ := json.Marshal(doc)
		code, out := alice.do("PUT", "/v1/me/settings", body)
		if code != want {
			t.Fatalf("today %s = %d %s", sections, code, out)
		}
		return out
	}
	if out := put(`["learning","kpis","tasks"]`, 200); !bytes.Contains(out, []byte(`"today":{"sections":["learning","kpis","tasks"]}`)) {
		t.Fatalf("saved = %s", out)
	}
	put(`["tasks","tasks"]`, 422)
	put(`["weather"]`, 422)
}

// The menu shows the pages a person uses (EP-94): any page but Today can
// be hidden, each once, and a new account has none hidden and has not chosen.
func TestMenuPages(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var doc map[string]any
	out := alice.mustDo(200, "GET", "/v1/me/settings", nil)
	if !bytes.Contains(out, []byte(`"menu":{"hidden":[],"chosen":false}`)) {
		t.Fatalf("default menu: %s", out)
	}
	_ = json.Unmarshal(out, &doc)
	delete(doc, "ai")
	put := func(menu string, want int) []byte {
		t.Helper()
		var m map[string]any
		_ = json.Unmarshal([]byte(menu), &m)
		doc["menu"] = m
		body, _ := json.Marshal(doc)
		code, out := alice.do("PUT", "/v1/me/settings", body)
		if code != want {
			t.Fatalf("menu %s = %d %s", menu, code, out)
		}
		return out
	}
	if out := put(`{"hidden":["budget","media"],"chosen":true}`, 200); !bytes.Contains(out, []byte(`"menu":{"hidden":["budget","media"],"chosen":true}`)) {
		t.Fatalf("saved = %s", out)
	}
	put(`{"hidden":["budget","budget"],"chosen":true}`, 422)
	put(`{"hidden":["today"],"chosen":true}`, 422)
	put(`{"hidden":["weather"],"chosen":true}`, 422)
	if y := string(alice.mustDo(200, "GET", "/v1/me/settings.yaml", nil)); !strings.Contains(y, "menu:\n    hidden:\n        - budget\n        - media\n    chosen: true") && !strings.Contains(y, "menu:\n  hidden:\n    - budget") {
		t.Fatalf("yaml = %s", y)
	}
}

// Today's sections are never null: a document stored that way (as the
// staging demo's was) reads back with the default sections, so Today never
// fails on them. (The API refuses null in a request.)
func TestTodaySectionsAreNeverNull(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	defaults := `"today":{"sections":["kpis","tasks","milestones","goals","notes","overview"]}`
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	body, _ := json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)
	if _, err := pool.Exec(context.Background(), `UPDATE user_settings SET doc = jsonb_set(doc, '{today}', '{"sections": null}')`); err != nil {
		t.Fatal(err)
	}
	if out := alice.mustDo(200, "GET", "/v1/me/settings", nil); !bytes.Contains(out, []byte(defaults)) {
		t.Fatalf("stored with null sections: %s", out)
	}
	// An empty list is a choice: Today shows only what is always there.
	doc["today"] = map[string]any{"sections": []string{}}
	body, _ = json.Marshal(doc)
	if out := alice.mustDo(200, "PUT", "/v1/me/settings", body); !bytes.Contains(out, []byte(`"today":{"sections":[]}`)) {
		t.Fatalf("saved with no sections: %s", out)
	}
}

// The API answers in the language the request prefers (EP-87).
func TestTheAPISpeaksFrenchWhenAsked(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	missing := "/v1/goals/00000000-0000-7000-8000-000000000000"
	if code, out := alice.do("GET", missing, nil, "Accept-Language", "fr-FR,fr;q=0.9"); code != 404 || !strings.Contains(string(out), "Objectif ou jalon introuvable.") {
		t.Fatalf("fr problem = %d %s", code, out)
	}
	if code, out := alice.do("GET", missing, nil, "Accept-Language", "en-GB"); code != 404 || !strings.Contains(string(out), "No such goal or milestone.") {
		t.Fatalf("en problem = %d %s", code, out)
	}
	if code, out := alice.do("GET", "/v1/insights", nil, "Accept-Language", "fr"); code != 200 || !strings.Contains(string(out), "Tâches terminées") {
		t.Fatalf("fr insights = %d %s", code, out)
	}
	if code, out := alice.do("GET", "/v1/me/consents", nil, "Accept-Language", "fr"); code != 200 || !strings.Contains(string(out), "Rechercher films, séries et livres") {
		t.Fatalf("fr consents = %d %s", code, out)
	}
}

// The theme can follow the sun, and the home picture shows a landscape the
// person chooses (EP-96).
func TestTheThemeFollowsTheSunOverAChosenLandscape(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	put := func(theme, scenery string, want int) []byte {
		t.Helper()
		a := doc["appearance"].(map[string]any)
		a["theme"], a["scenery"] = theme, scenery
		body, _ := json.Marshal(doc)
		code, out := alice.do("PUT", "/v1/me/settings", body)
		if code != want {
			t.Fatalf("%s over %s = %d %s", theme, scenery, code, out)
		}
		return out
	}
	if out := put("sun", "taj-mahal", 200); !bytes.Contains(out, []byte(`"theme":"sun"`)) || !bytes.Contains(out, []byte(`"scenery":"taj-mahal"`)) {
		t.Fatalf("saved = %s", out)
	}
	put("sun", "atlantis", 422)
	put("moon", "forest", 422)
}
