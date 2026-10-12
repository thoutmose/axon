package fsrs

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// TestMatchesReference replays the reference implementation's reviews
// (py-fsrs 4.1.2, fuzzing off; see testdata) and compares every step.
func TestMatchesReference(t *testing.T) {
	b, err := os.ReadFile("testdata/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Parameters []float64 `json:"parameters"`
		Cases      [][]struct {
			Rating     Rating    `json:"rating"`
			At         time.Time `json:"at"`
			State      State     `json:"state"`
			Step       *int      `json:"step"`
			Stability  float64   `json:"stability"`
			Difficulty float64   `json:"difficulty"`
			Due        time.Time `json:"due"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	s := Default()
	for i, p := range fx.Parameters {
		if p != s.W[i] {
			t.Fatalf("parameter %d differs: %v vs %v", i, p, s.W[i])
		}
	}
	reviews := 0
	for n, steps := range fx.Cases {
		c := New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		for i, want := range steps {
			c = s.Review(c, want.Rating, want.At)
			reviews++
			switch {
			case c.State != want.State,
				(c.Step == nil) != (want.Step == nil),
				c.Step != nil && *c.Step != *want.Step,
				math.Abs(*c.Stability-want.Stability) > 1e-9,
				math.Abs(*c.Difficulty-want.Difficulty) > 1e-9,
				!c.Due.Equal(want.Due):
				t.Fatalf("case %d step %d (rating %d at %s): got state %d step %v S %.6f D %.6f due %s; want state %d step %v S %.6f D %.6f due %s",
					n, i, want.Rating, want.At, c.State, c.Step, *c.Stability, *c.Difficulty, c.Due, want.State, want.Step, want.Stability, want.Difficulty, want.Due)
			}
		}
	}
	if reviews != 480 {
		t.Fatalf("replayed %d reviews", reviews)
	}
}

func TestRetrievabilityFallsOverTime(t *testing.T) {
	s := Default()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := s.Review(New(now), Easy, now)
	r1 := s.Retrievability(c, now.AddDate(0, 0, 1))
	r30 := s.Retrievability(c, now.AddDate(0, 0, 30))
	if r1 <= r30 || r1 > 1 || r30 <= 0 {
		t.Fatalf("retrievability %f then %f", r1, r30)
	}
}
