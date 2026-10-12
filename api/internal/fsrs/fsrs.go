// Package fsrs schedules flashcard reviews with FSRS-5, the Free Spaced
// Repetition Scheduler used by Anki (EP-16). It is a straight port of the
// reference implementation, py-fsrs 4.1.2, with fuzzing off so results are
// reproducible; testdata holds that implementation's output for 480 reviews.
package fsrs

import (
	"math"
	"time"
)

// State is a card's learning state.
type State int

// Learning states, as in the reference implementation.
const (
	Learning   State = 1
	Review     State = 2
	Relearning State = 3
)

// Rating is how well the person recalled the card.
type Rating int

// Ratings: Again (forgot), Hard, Good, Easy.
const (
	Again Rating = 1
	Hard  Rating = 2
	Good  Rating = 3
	Easy  Rating = 4
)

const decay = -0.5

var factor = math.Pow(0.9, 1/decay) - 1

// DefaultParameters are FSRS-5's 19 default weights.
var DefaultParameters = [19]float64{
	0.40255, 1.18385, 3.173, 15.69105, 7.1949, 0.5345, 1.4604, 0.0046, 1.54575, 0.1192,
	1.01925, 1.9395, 0.11, 0.29605, 2.2698, 0.2315, 2.9898, 0.51655, 0.6621,
}

// Scheduler holds the model and the learning steps.
type Scheduler struct {
	W                [19]float64
	DesiredRetention float64
	LearningSteps    []time.Duration
	RelearningSteps  []time.Duration
	MaximumInterval  int // days
}

// Default returns the reference defaults: 90% retention, 1 and 10 minute
// learning steps, a 10 minute relearning step, and intervals up to 100 years.
func Default() Scheduler {
	return Scheduler{
		W:                DefaultParameters,
		DesiredRetention: 0.9,
		LearningSteps:    []time.Duration{time.Minute, 10 * time.Minute},
		RelearningSteps:  []time.Duration{10 * time.Minute},
		MaximumInterval:  36500,
	}
}

// Card is the scheduling state of one card.
type Card struct {
	State      State      `json:"state"`
	Step       *int       `json:"step"`
	Stability  *float64   `json:"stability"`
	Difficulty *float64   `json:"difficulty"`
	Due        time.Time  `json:"due"`
	LastReview *time.Time `json:"last_review"`
}

// New returns a new card, due now.
func New(now time.Time) Card {
	zero := 0
	return Card{State: Learning, Step: &zero, Due: now}
}

// Retrievability is the predicted chance of recalling the card at t.
func (s Scheduler) Retrievability(c Card, t time.Time) float64 {
	if c.LastReview == nil || c.Stability == nil {
		return 0
	}
	elapsed := math.Max(0, math.Floor(t.Sub(*c.LastReview).Hours()/24))
	return math.Pow(1+factor*elapsed/(*c.Stability), decay)
}

func clamp(x, lo, hi float64) float64 { return math.Min(math.Max(x, lo), hi) }

func (s Scheduler) initialStability(r Rating) float64 { return math.Max(s.W[r-1], 0.1) }

func (s Scheduler) initialDifficulty(r Rating) float64 {
	return clamp(s.W[4]-math.Exp(s.W[5]*float64(r-1))+1, 1, 10)
}

func (s Scheduler) nextInterval(stability float64) int {
	i := int(math.Round(stability / factor * (math.Pow(s.DesiredRetention, 1/decay) - 1)))
	return min(max(i, 1), s.MaximumInterval)
}

func (s Scheduler) shortTermStability(stability float64, r Rating) float64 {
	return stability * math.Exp(s.W[17]*(float64(r)-3+s.W[18]))
}

func (s Scheduler) nextDifficulty(d float64, r Rating) float64 {
	delta := -(s.W[6] * float64(r-3))
	damped := d + (10-d)*delta/9
	return clamp(s.W[7]*s.initialDifficulty(Easy)+(1-s.W[7])*damped, 1, 10)
}

func (s Scheduler) nextStability(d, st, ret float64, r Rating) float64 {
	if r == Again {
		long := s.W[11] * math.Pow(d, -s.W[12]) * (math.Pow(st+1, s.W[13]) - 1) * math.Exp((1-ret)*s.W[14])
		short := st / math.Exp(s.W[17]*s.W[18])
		return math.Min(long, short)
	}
	hard, easy := 1.0, 1.0
	if r == Hard {
		hard = s.W[15]
	}
	if r == Easy {
		easy = s.W[16]
	}
	return st * (1 + math.Exp(s.W[8])*(11-d)*math.Pow(st, -s.W[9])*(math.Exp((1-ret)*s.W[10])-1)*hard*easy)
}

// Review returns the card after a review with rating r at time t.
func (s Scheduler) Review(c Card, r Rating, t time.Time) Card {
	var sinceDays *int
	if c.LastReview != nil {
		d := int(t.Sub(*c.LastReview).Hours() / 24)
		sinceDays = &d
	}
	stab, diff := 0.0, 0.0
	switch {
	case c.State == Learning && c.Stability == nil && c.Difficulty == nil:
		stab, diff = s.initialStability(r), s.initialDifficulty(r)
	case sinceDays != nil && *sinceDays < 1:
		stab, diff = s.shortTermStability(*c.Stability, r), s.nextDifficulty(*c.Difficulty, r)
	default:
		stab = s.nextStability(*c.Difficulty, *c.Stability, s.Retrievability(c, t), r)
		diff = s.nextDifficulty(*c.Difficulty, r)
	}
	c.Stability, c.Difficulty = &stab, &diff

	toReview := func() time.Duration {
		c.State, c.Step = Review, nil
		return time.Duration(s.nextInterval(stab)) * 24 * time.Hour
	}
	steps := func(list []time.Duration) time.Duration {
		step := *c.Step
		if len(list) == 0 || step > len(list) {
			return toReview()
		}
		switch r {
		case Again:
			zero := 0
			c.Step = &zero
			return list[0]
		case Hard:
			switch {
			case step == 0 && len(list) == 1:
				return list[0] * 3 / 2
			case step == 0:
				return (list[0] + list[1]) / 2
			default:
				return list[step]
			}
		case Good:
			if step+1 == len(list) {
				return toReview()
			}
			next := step + 1
			c.Step = &next
			return list[next]
		default: // Easy
			return toReview()
		}
	}

	var interval time.Duration
	switch c.State {
	case Learning:
		interval = steps(s.LearningSteps)
	case Review:
		if r == Again && len(s.RelearningSteps) > 0 {
			zero := 0
			c.State, c.Step = Relearning, &zero
			interval = s.RelearningSteps[0]
		} else {
			interval = time.Duration(s.nextInterval(stab)) * 24 * time.Hour
		}
	case Relearning:
		interval = steps(s.RelearningSteps)
	}
	c.Due = t.Add(interval)
	c.LastReview = &t
	return c
}
