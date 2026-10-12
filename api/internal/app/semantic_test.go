package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/semantic"
)

// fakeEmbeddings embeds by concept: words of the same meaning share a
// dimension, so "automobile" is close to "car" and far from "bread".
func fakeEmbeddings(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	concepts := map[string]int{"car": 0, "cars": 0, "automobile": 0, "vehicle": 0, "engine": 0,
		"bread": 1, "baking": 1, "flour": 1, "dough": 1, "sleep": 2, "rest": 2, "night": 2}
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer sk-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var in struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		type item struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		}
		out := []item{}
		for i, text := range in.Input {
			v := make([]float32, 4)
			v[3] = 0.1
			for _, w := range strings.Fields(strings.ToLower(text)) {
				if c, ok := concepts[strings.Trim(w, ".,!?#")]; ok {
					v[c]++
				}
			}
			out = append(out, item{i, v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
	}))
	t.Cleanup(srv.Close)
	client := semantic.Client
	semantic.Client = srv.Client()
	t.Cleanup(func() { semantic.Client = client })
	return srv, &calls
}

// Search by meaning: off until a service and the consent are given, then
// notes are found by what they mean; withdrawing the consent deletes the vectors.
func TestSearchByMeaning(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	srv, calls := fakeEmbeddings(t)
	for _, n := range []string{
		`{"path":"Zettel/Vehicles.md","content":"# Vehicles\n\nAn automobile needs an engine."}`,
		`{"path":"Zettel/Sourdough.md","content":"# Sourdough\n\nBaking bread from flour and dough."}`,
		`{"path":"Zettel/Rest.md","content":"# Rest\n\nSleep at night."}`,
	} {
		alice.mustDo(201, "POST", "/v1/notes", []byte(n))
	}
	bob.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Bob/Cars.md","content":"# Cars\n\nCars and engines."}`))

	alice.mustDo(409, "GET", "/v1/notes/semantic?q=car", nil)
	alice.mustDo(422, "PUT", "/v1/me/embeddings", []byte(`{"provider":"local"}`))
	alice.mustDo(422, "PUT", "/v1/me/embeddings", []byte(`{"provider":"openai"}`)) // no key
	body := `{"provider":"local","base_url":"` + srv.URL + `","model":"test-embed","api_key":"sk-test"}`
	if out := alice.mustDo(200, "PUT", "/v1/me/embeddings", []byte(body)); strings.Contains(string(out), "sk-test") {
		t.Fatalf("the key came back: %s", out)
	}
	alice.mustDo(403, "GET", "/v1/notes/semantic?q=car", nil) // no consent yet

	ix, err := Semantic(auth.Config{EncryptionKey: strings.Repeat("ab", 32)}, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.Step(context.Background()); err != nil || calls.Load() != 0 {
		t.Fatalf("indexed without consent: %v, %d calls", err, calls.Load())
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_embeddings", []byte(`{"granted":true}`))
	if err := ix.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	var st struct{ Indexed, Notes int }
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/embeddings", nil), &st)
	if st.Indexed != 3 || st.Notes != 3 {
		t.Fatalf("status = %+v", st)
	}
	before := calls.Load()
	if err := ix.Step(context.Background()); err != nil || calls.Load() != before {
		t.Fatal("unchanged notes were embedded again")
	}

	var hits struct {
		Items []struct {
			Title string  `json:"title"`
			Score float64 `json:"score"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/semantic?q="+url.QueryEscape("my car"), nil), &hits)
	if len(hits.Items) != 3 || hits.Items[0].Title != "Vehicles" || hits.Items[0].Score < 0.9 {
		t.Fatalf("by meaning = %+v", hits.Items)
	}
	var sourdough struct {
		ID string `json:"id"`
	}
	var list struct {
		Items []struct {
			ID, Title string
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?limit=100", nil), &list)
	for _, n := range list.Items {
		if n.Title == "Sourdough" {
			sourdough.ID = n.ID
		}
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+sourdough.ID+"/similar", nil), &hits)
	if len(hits.Items) != 2 || hits.Items[0].Title == "Sourdough" {
		t.Fatalf("similar = %+v", hits.Items)
	}
	bob.mustDo(409, "GET", "/v1/notes/semantic?q=car", nil) // bob has not set it up, and never sees alice's notes

	// Withdrawing the consent deletes the vectors; turning it off forgets the service.
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_embeddings", []byte(`{"granted":false}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/embeddings", nil), &st)
	if st.Indexed != 0 {
		t.Fatalf("vectors kept after withdrawal: %+v", st)
	}
	alice.mustDo(204, "DELETE", "/v1/me/embeddings", nil)
	alice.mustDo(409, "GET", "/v1/notes/semantic?q=car", nil)
}
