// Package practice keeps a log of practice problems (EP-103): algorithm
// problems, wargame levels, security labs, competitions. Each attempt is
// logged as solved unaided, solved with help, or not solved, and a problem
// comes back to re-solve at growing intervals, the way spaced retrieval
// works for cards: retrieving a solution, not recognising it, is the skill.
package practice

import (
	"math"
	"time"
)

// Attempt is one try at a problem.
type Attempt struct {
	On      time.Time
	Outcome string // solved, with_help, not_solved
}

const (
	minGap  = 7  // days after an unaided solve
	maxGap  = 90 // past this, the problem is known
	growth  = 2.5
	helpGap = 3
	missGap = 1
)

// Next is the day to re-solve a problem after its attempts (oldest first),
// or nil when it needs no more practice.
func Next(difficulty string, attempts []Attempt) *time.Time {
	if len(attempts) == 0 {
		return nil
	}
	last := attempts[len(attempts)-1]
	at := func(days int) *time.Time { t := last.On.AddDate(0, 0, days); return &t }
	switch last.Outcome {
	case "not_solved":
		return at(missGap)
	case "with_help":
		return at(helpGap)
	}
	// Unaided: the gap grows from the previous unaided solve in this run.
	if len(attempts) == 1 || attempts[len(attempts)-2].Outcome != "solved" {
		if len(attempts) == 1 && difficulty == "easy" {
			return nil
		}
		return at(minGap)
	}
	prev := attempts[len(attempts)-2].On
	gap := int(math.Round(math.Round(last.On.Sub(prev).Hours()/24) * growth))
	if gap < minGap {
		gap = minGap
	}
	if gap > maxGap {
		return nil
	}
	return at(gap)
}
