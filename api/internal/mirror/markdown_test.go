package mirror

import (
	"reflect"
	"testing"
)

func TestPageRoundTrip(t *testing.T) {
	p := page{
		fm:       []kv{{"type", "task"}, {"context", yamlValue("@phone")}, {"title", yamlValue("Note: with colon")}, {"empty", ""}, {"genres", list([]string{"Science fiction", "Drama"})}},
		title:    "Call the plumber",
		sections: []section{{heading: "Notes", body: "After 5 pm.\n\nTwo lines."}, {heading: "Recent check-ins", body: "", readOnly: true}},
		footer:   footer,
	}
	got := parse(p.markdown())
	if got.title != "Call the plumber" || got.fm["context"] != "@phone" || got.fm["title"] != "Note: with colon" || got.fm["empty"] != "" {
		t.Fatalf("parsed = %+v", got)
	}
	if got.sections["Notes"] != "After 5 pm.\n\nTwo lines." || got.sections["Recent check-ins"] != "" {
		t.Fatalf("sections = %q", got.sections)
	}
	if !reflect.DeepEqual(unlist(got.fm["genres"]), []string{"Science fiction", "Drama"}) {
		t.Fatalf("genres = %q", got.fm["genres"])
	}
}

func TestFileName(t *testing.T) {
	for in, want := range map[string]string{"Raft & Paxos: a comparison?": "Raft & Paxos a comparison", "  ": "Untitled", "a/b\\c": "a b c", ".hidden": "Untitled .hidden"} {
		if got := fileName(in); got != want {
			t.Errorf("fileName(%q) = %q, want %q", in, got, want)
		}
	}
}
