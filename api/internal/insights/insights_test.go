package insights

import (
	"testing"
	"time"
)

func TestWeeks(t *testing.T) {
	sun := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC) // a Sunday
	if m := Monday(sun); m.Format(time.DateOnly) != "2026-10-05" || Label(m) != "2026-W41" {
		t.Fatalf("Monday = %v, label %s", m, Label(m))
	}
	for s, want := range map[string]string{"2026-W41": "2026-10-05", "2026-W01": "2025-12-29", "2020-W53": "2020-12-28"} {
		m, ok := ParseWeek(s)
		if !ok || m.Format(time.DateOnly) != want {
			t.Errorf("ParseWeek(%s) = %v, %v; want %s", s, m, ok, want)
		}
	}
	for _, bad := range []string{"2026-W54", "2025-W53", "x", "2026-41"} {
		if _, ok := ParseWeek(bad); ok {
			t.Errorf("ParseWeek(%q) accepted", bad)
		}
	}
}

func TestTrendSentences(t *testing.T) {
	pts := func(vals ...float64) []Point {
		out := make([]Point, len(vals))
		for i, v := range vals {
			out[i] = Point{Value: v}
		}
		return out
	}
	if tr := trend(pts(2, 2, 2, 2, 3, 3, 3, 3), "count", "en"); tr.Direction != "up" || tr.Change != 50 || tr.Sentence != "Up +50 % on the four weeks before (2.0 → 3.0 a week)." {
		t.Fatalf("up = %+v", tr)
	}
	if tr := trend(pts(4, 4, 4, 4, 4, 4, 4, 3.8), "count", "en"); tr.Direction != "flat" {
		t.Fatalf("flat = %+v", tr)
	}
	if tr := trend(pts(0, 0, 0, 0, 0, 0, 1, 2), "count", "en"); tr.Direction != "new" {
		t.Fatalf("new = %+v", tr)
	}
	if tr := trend(pts(0, 0, 0, 0), "count", "en"); tr.Direction != "flat" || tr.Sentence != "Nothing in the last eight weeks." {
		t.Fatalf("empty = %+v", tr)
	}
	for _, m := range Metrics {
		if m.Formula == "" || m.Source == "" || m.weekly == "" || m.items == "" {
			t.Errorf("%s is not fully documented", m.ID)
		}
	}
}

func TestTrendSentencesInFrench(t *testing.T) {
	pts := []Point{{Value: 2}, {Value: 2}, {Value: 2}, {Value: 2}, {Value: 3}, {Value: 3}, {Value: 3}, {Value: 3}}
	want := "En hausse (+50 %) par rapport aux quatre semaines précédentes (2,0 → 3,0 par semaine)."
	if tr := trend(pts, "count", "fr"); tr.Sentence != want {
		t.Fatalf("fr = %q", tr.Sentence)
	}
}
