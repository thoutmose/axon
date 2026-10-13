package journal

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

func f(v float64) *float64 { return &v }

// synthetic days: mood follows sleep, energy is noise; workouts every third day.
func synthetic(n int) []Day {
	r := rand.New(rand.NewPCG(7, 7)) //nolint:gosec // reproducible test data
	var days []Day
	for i := range n {
		sleep := 5.5 + float64(r.IntN(7))/2 // 5.5 to 8.5 h
		mood := 2.0
		if sleep >= 7 {
			mood = 4
		}
		if r.IntN(5) == 0 { // a few days go against the pattern
			mood = 3
		}
		day := Day{Date: fmt.Sprintf("2026-07-%02d", i%28+1), Mood: f(mood), Energy: f(float64(1 + r.IntN(5))), Sleep: f(sleep)}
		if i%3 == 0 {
			day.Workout = 30
		}
		days = append(days, day)
	}
	return days
}

func TestHelpsFindsAPlantedLinkAndNotNoise(t *testing.T) {
	h := analyse(synthetic(40), "en")
	if h.Needed != 0 || h.Tested == 0 {
		t.Fatalf("helps = %+v", h)
	}
	var sleepMood *Finding
	for i, fd := range h.Findings {
		if fd.Outcome == "energy" {
			t.Errorf("a link with noise was reported: %+v", fd)
		}
		if fd.Factor == "sleep" && fd.Outcome == "mood" {
			sleepMood = &h.Findings[i]
		}
	}
	if sleepMood == nil {
		t.Fatalf("the sleep and mood link was missed: %+v", h.Findings)
	}
	if sleepMood.Rho < 0.5 || sleepMood.Q > 0.1 || sleepMood.MeanHigh <= sleepMood.MeanLow || sleepMood.NHigh+sleepMood.NLow != 40 {
		t.Fatalf("finding = %+v", sleepMood)
	}
	if !strings.Contains(sleepMood.Sentence, "slept 7 h or more") || !strings.Contains(sleepMood.Sentence, "out of 5") {
		t.Fatalf("sentence = %q", sleepMood.Sentence)
	}
}

func TestHelpsWaitsForEnoughDays(t *testing.T) {
	h := analyse(synthetic(9), "en")
	if h.Needed != 5 || len(h.Findings) != 0 {
		t.Fatalf("helps = %+v", h)
	}
}

func TestHelpsSpeaksFrench(t *testing.T) {
	h := analyse(synthetic(40), "fr")
	if len(h.Findings) == 0 || !strings.Contains(h.Findings[0].Sentence, "sur 5") || !strings.Contains(h.Method, "Spearman") {
		t.Fatalf("fr = %+v", h)
	}
}
