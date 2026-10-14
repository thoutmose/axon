package watch

import (
	"strings"
	"testing"
)

func TestAnalyse(t *testing.T) {
	topics := []Topic{
		{ID: "db", Name: "Databases", Keywords: []string{"Postgres", "SQL", "vacuum"}},
		{ID: "ml", Name: "Machine learning", Keywords: []string{"neural network", "LLM"}},
	}
	e := Entry{
		Title:      "Postgres 18 makes vacuum faster",
		URL:        "https://blog.example/pg18",
		Categories: []string{"Release"},
		Content: `<p>PostgreSQL 18 ships a new <a href="/docs">vacuum</a> strategy that reduces bloat on large tables.</p>
<p>Benchmarks on busy clusters show lower write amplification, faster index cleanup, and shorter maintenance windows for operators.
The release also improves logical replication and the query planner for partitioned tables.</p>`,
	}
	a := Analyse(e, topics)
	if a.Lang != "en" || a.Words < 30 || a.Outlier != "" || a.SimHash == 0 || len(a.Terms) < 20 {
		t.Fatalf("analysis = %+v", a)
	}
	if len(a.Topics) != 1 || a.Topics[0] != "db" || a.Score < 3 {
		t.Fatalf("topics = %v score %v", a.Topics, a.Score)
	}
	if a.Labels[0] != "databases" || !containsStr(a.Labels, "release") || len(a.Keywords) == 0 {
		t.Fatalf("labels = %v keywords = %v", a.Labels, a.Keywords)
	}
	if !strings.Contains(a.Markdown, "[vacuum](https://blog.example/docs)") || !strings.HasPrefix(a.Summary, "PostgreSQL 18 ships") {
		t.Fatalf("markdown = %q summary = %q", a.Markdown, a.Summary)
	}
	// A single passing mention in the text is not enough for a topic.
	other := Analyse(Entry{Title: "A neural approach to cooking", Content: "<p>" + strings.Repeat("Recipes and kitchens and flavours. ", 10) + "We mention SQL once.</p>"}, topics)
	if len(other.Topics) != 0 {
		t.Fatalf("a passing mention labelled it: %v", other.Topics)
	}
	// Outliers: too short, mostly links, shouting.
	for want, e := range map[string]Entry{
		"too short":      {Title: "Hi", Summary: "Short."},
		"mostly links":   {Title: "Links", Content: strings.Repeat(`<a href="https://a.example">https://a.example</a> `, 20) + "<p>" + strings.Repeat("word ", 30) + "</p>"},
		"shouting title": {Title: "YOU WON'T BELIEVE THIS", Content: "<p>" + strings.Repeat("word ", 40) + "</p>"},
	} {
		if got := Analyse(e, nil).Outlier; got != want {
			t.Errorf("%s: outlier = %q", want, got)
		}
	}
	// French text is read as French.
	if fr := Analyse(Entry{Title: "Les bases de données", Content: "<p>" + strings.Repeat("La base de données est rapide et les requêtes sont simples pour les équipes. ", 5) + "</p>"}, nil); fr.Lang != "fr" {
		t.Fatalf("lang = %s", fr.Lang)
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
