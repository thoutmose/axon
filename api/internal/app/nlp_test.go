package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// Literature and permanent notes get topics, a concept graph whose edges
// say why, and per-note analysis with reasons; other notes stay out of the
// corpus, and another person sees nothing.
func TestNoteAnalysis(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	note := func(path, content string) string {
		var n noteJSON
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"`+path+`","content":`+quote(content)+`}`)), &n)
		return n.ID
	}
	spaced := note("Zettel/permanent/spaced.md", "---\ntype: permanent\n---\n# Spaced repetition\n\nSpaced repetition and retrieval practice fight the forgetting curve. Reviews come just before forgetting.\n")
	note("Zettel/permanent/retrieval.md", "---\ntype: permanent\n---\n# Retrieval practice\n\nRetrieval practice with spaced repetition beats rereading and flattens the forgetting curve.\n")
	note("Zettel/literature/stick.md", "---\ntype: literature\n---\n# Make It Stick\n\nThe book shows that retrieval practice and spaced repetition beat cramming. A remark on the forgetting curve.\n")
	note("Zettel/permanent/pods.md", "---\ntype: permanent\n---\n# Kubernetes pods\n\nKubernetes schedules pods on nodes. The control plane watches pods and nodes.\n")
	note("Zettel/literature/k8s.md", "---\ntype: literature\n---\n# Kubernetes in Action\n\nA Kubernetes service routes traffic to pods on nodes behind the control plane.\n")
	fleeting := note("0-Inbox/idea.md", "# Idea\n\nTry spaced repetition for Kubernetes commands.\n")
	fr := note("Zettel/permanent/fr.md", "---\ntype: permanent\n---\n# Les cartes\n\nLes étudiants révisent leurs cartes avec la répétition espacée pour mieux retenir.\n")

	var a struct {
		Notes     int            `json:"notes"`
		Languages map[string]int `json:"languages"`
		Topics    []struct {
			Label string `json:"label"`
			Notes []struct {
				ID string `json:"id"`
			} `json:"notes"`
		} `json:"topics"`
		Concepts []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"concepts"`
		Edges []struct {
			Relation string `json:"relation"`
			Reason   string `json:"reason"`
		} `json:"edges"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/analysis", nil), &a)
	if a.Notes != 6 || a.Languages["fr"] != 1 || a.Languages["en"] != 5 {
		t.Fatalf("corpus: %d notes, languages %v (the fleeting note must stay out)", a.Notes, a.Languages)
	}
	if len(a.Topics) < 2 || len(a.Topics[0].Notes) != 3 {
		t.Fatalf("topics = %+v", a.Topics)
	}
	texts := map[string]bool{}
	for _, c := range a.Concepts {
		texts[c.Text] = true
	}
	if !texts["spaced repetition"] || !texts["retrieval practice"] || !texts["control plane"] {
		t.Fatalf("concepts = %v", texts)
	}
	if len(a.Edges) == 0 || !strings.HasPrefix(a.Edges[0].Reason, "Together in") {
		t.Fatalf("edges = %+v", a.Edges)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/analysis?kind=literature", nil), &a)
	if a.Notes != 2 {
		t.Fatalf("literature only: %d", a.Notes)
	}
	alice.mustDo(400, "GET", "/v1/notes/analysis?kind=fleeting", nil)

	var n struct {
		Readability struct {
			Language string `json:"language"`
			Words    int    `json:"words"`
		} `json:"readability"`
		Phrases []struct {
			Text string `json:"text"`
		} `json:"phrases"`
		Topic *struct {
			Label string `json:"label"`
		} `json:"topic"`
		Related []struct {
			Title   string   `json:"title"`
			Reasons []string `json:"reasons"`
		} `json:"related"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+spaced+"/analysis", nil), &n)
	if n.Readability.Language != "en" || n.Readability.Words == 0 || len(n.Phrases) == 0 || n.Topic == nil || len(n.Related) == 0 {
		t.Fatalf("note analysis = %+v", n)
	}
	if !strings.HasPrefix(n.Related[0].Reasons[0], "Shares") {
		t.Fatalf("related = %+v", n.Related)
	}
	// A fleeting note is analysed against the corpus without joining it.
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+fleeting+"/analysis", nil), &n)
	if n.Topic != nil || len(n.Related) == 0 {
		t.Fatalf("fleeting analysis = %+v", n)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+fr+"/analysis", nil), &n)
	if n.Readability.Language != "fr" {
		t.Fatalf("french = %+v", n.Readability)
	}

	bob.mustDo(404, "GET", "/v1/notes/"+spaced+"/analysis", nil)
	alice.mustDo(404, "GET", "/v1/notes/not-a-uuid/analysis", nil)
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/notes/analysis", nil), &a)
	if a.Notes != 0 || len(a.Concepts) != 0 {
		t.Fatalf("bob sees %d notes", a.Notes)
	}
}
