package digest

import (
	"testing"
	"time"
)

// Stories (EP-155): items that say the same thing are one story, and a
// story many outlets carry comes first.
func TestStoriesRankByOutlets(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	terms := func(s string) []string { return []string{s + "1", s + "2", s + "3", s + "4", s + "5"} }
	items := []newsItem{
		{title: "Small local story", source: "Le Monde", simhash: 0xF0F0, terms: terms("local"), published: now.Add(-time.Hour)},
		{title: "Summit ends with a deal", source: "BBC", simhash: 0x1111, terms: terms("summit"), published: now.Add(-5 * time.Hour)},
		{title: "Deal reached at the summit", source: "Reuters", simhash: 0x1113, terms: terms("summit"), published: now.Add(-4 * time.Hour)},
		{title: "Summit: a deal", source: "Le Monde", simhash: 0xFFFF_0000_1111, terms: append(terms("summit")[:4], "x"), published: now.Add(-3 * time.Hour)},
		{title: "Same outlet twice", source: "BBC", simhash: 0xF0F0_0000_0000, terms: terms("local"), published: now.Add(-2 * time.Hour)},
	}
	got := stories(items, now)
	if len(got) != 2 {
		t.Fatalf("%d stories", len(got))
	}
	if len(got[0].outlets) != 3 || got[0].items[0].title != "Summit ends with a deal" {
		t.Fatalf("first story = %+v", got[0])
	}
	if len(got[1].outlets) != 2 || got[1].items[0].title != "Same outlet twice" { // the first to cover it
		t.Fatalf("second story = %+v", got[1])
	}
}
