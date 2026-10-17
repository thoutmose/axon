package media

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func TestRankExplainsEverySuggestion(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	item := func(title, status, creators string, genres []string, rating int, age time.Duration) Item {
		it := Item{ID: uuid.New(), Title: title, Status: status, Creators: creators, Genres: genres, CreatedAt: now.Add(-age), UpdatedAt: now.Add(-age)}
		if rating > 0 {
			it.Rating = ptr(rating)
		}
		return it
	}
	paused := item("Half read", "doing", "", nil, 0, 30*24*time.Hour)
	paused.Total, paused.Progress = ptr(300), 120
	items := []Item{
		item("Left Hand of Darkness", "done", "Ursula K. Le Guin", []string{"Science fiction"}, 9, 0),
		item("Meh", "done", "Someone", []string{"Romance"}, 5, 0),
		item("The Dispossessed", "want", "Ursula K. Le Guin", []string{"science fiction"}, 0, time.Hour),
		item("Old want", "want", "", nil, 0, 400*24*time.Hour),
		item("New want", "want", "", nil, 0, time.Hour),
		item("Romance novel", "want", "", []string{"Romance"}, 0, time.Hour),
		item("Gave up", "dropped", "Ursula K. Le Guin", nil, 0, time.Hour),
		paused,
	}
	got := rank(items, now, 10, "en")
	titles := make([]string, len(got))
	for i, r := range got {
		titles[i] = r.Item.Title
		if len(r.Reasons) == 0 {
			t.Errorf("%s has no reason", r.Item.Title)
		}
	}
	want := "The Dispossessed,Half read,Old want,New want,Romance novel"
	if strings.Join(titles, ",") != want {
		t.Fatalf("order = %v, want %s", titles, want)
	}
	if r := strings.Join(got[0].Reasons, " "); !strings.Contains(r, "By Ursula K. Le Guin, like Left Hand of Darkness, which you rated 9/10.") ||
		!strings.Contains(r, "science fiction, like Left Hand of Darkness, which you loved.") {
		t.Fatalf("reasons = %q", r)
	}
	if r := got[1].Reasons[0]; r != "You started it and are at 120 of 300; untouched for 30 days." {
		t.Fatalf("paused reason = %q", r)
	}
	if len(rank(items, now, 2, "en")) != 2 {
		t.Fatal("limit ignored")
	}
}

func TestApplyFillsDatesAndChecksBounds(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	it := Item{Status: "want", Total: ptr(10)}
	if err := apply(&it, Fields{Status: ptr("done")}, now); err != nil {
		t.Fatal(err)
	}
	if it.StartedOn == nil || *it.FinishedOn != "2026-10-11" || it.Progress != 10 {
		t.Fatalf("done: %+v", it)
	}
	for _, f := range []Fields{
		{Rating: ptr(11)}, {Progress: ptr(11)}, {Kind: ptr("podcast")}, {Year: ptr(99)},
		{StartedOn: ptr("yesterday")}, {Genres: ptr(strings.Split("a,b,c,d,e,f,g,h,i,j,k,l,m", ","))},
	} {
		c := it
		if err := apply(&c, f, now); err == nil {
			t.Errorf("%+v accepted", f)
		}
	}
	c := it
	if err := apply(&c, Fields{Rating: ptr(0), Total: ptr(0), Genres: ptr([]string{" Drama ", "drama", ""})}, now); err != nil || c.Rating != nil || c.Total != nil || len(c.Genres) != 1 {
		t.Fatalf("clearing: %+v, %v", c, err)
	}
}

func TestCatalogueIDsAreStrict(t *testing.T) {
	for id, ok := range map[string]bool{"OL45883W": true, "OL1W": true, "OL45883M": false, "../x": false, "OLW": false, "OL12W/../": false} {
		if validID(id, "OLW") != ok {
			t.Errorf("validID(%q) = %v", id, !ok)
		}
	}
	if validNumber("12a") || validNumber("") || !validNumber("603") {
		t.Error("validNumber")
	}
}
