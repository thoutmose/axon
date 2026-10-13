package journal

import (
	"context"
	"hash/fnv"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/l10n"
)

// What helps you (EP-92): over the last 90 days, which of a day's habits go
// with a better or worse mood, energy, or stress. Rank correlation, a
// permutation test for each pair, and the Benjamini–Hochberg correction
// for testing many pairs; only clear links are shown, each in plain words,
// and always as a link, never as proof of cause. Computed here, no AI.

const (
	helpsWindow = 90
	minDays     = 14   // days with both the factor and the outcome
	minSide     = 5    // days on each side of the split
	maxQ        = 0.1  // false findings accepted among those shown
	minRho      = 0.3  // a link weaker than this is not worth a sentence
	rounds      = 2000 // permutations per pair
)

// Day is one day's outcomes (from the journal) and factors (from the logs).
type Day struct {
	Date                 string
	Mood, Energy, Stress *float64
	Sleep                *float64
	Workout, Focus       float64 // minutes
	Habits, Tasks        float64 // habits kept, tasks finished
	Gratitude            bool
}

// Finding is one clear link between a factor and an outcome.
type Finding struct {
	Factor   string  `json:"factor"`
	Outcome  string  `json:"outcome"`
	Rho      float64 `json:"rho"`
	P        float64 `json:"p"`
	Q        float64 `json:"q"`
	N        int     `json:"n"`
	MeanHigh float64 `json:"mean_with"`
	MeanLow  float64 `json:"mean_without"`
	NHigh    int     `json:"days_with"`
	NLow     int     `json:"days_without"`
	Strength string  `json:"strength"` // moderate or strong
	Sentence string  `json:"sentence"`
}

// Helps is what the last 90 days say.
type Helps struct {
	Days     int       `json:"days"`   // days with something in the journal
	Needed   int       `json:"needed"` // more days before anything can be said
	Tested   int       `json:"tested"`
	Findings []Finding `json:"findings"`
	Method   string    `json:"method"`
}

// factor reads one factor from a day, and says on which side of its split
// the day falls; ok is false when the day does not have it.
type factor struct {
	name  string
	value func(d Day) (v float64, ok bool)
	// split returns the threshold rule for the window's values and how the
	// "with" side reads ("slept 7 h or more").
	split func(values []float64, lang string) (with func(v float64) bool, phrase string)
}

func any0(lang, phrase string) (func(float64) bool, string) {
	return func(v float64) bool { return v > 0 }, l10n.Tr(lang, phrase)
}

// countSplit splits a count at its median: "more than N", or "at least one".
func countSplit(one, more string) func([]float64, string) (func(float64) bool, string) {
	return func(values []float64, lang string) (func(float64) bool, string) {
		s := slices.Clone(values)
		slices.Sort(s)
		m := s[len(s)/2]
		if m == 0 {
			return func(v float64) bool { return v > 0 }, l10n.Tr(lang, one)
		}
		return func(v float64) bool { return v > m }, l10n.Tr(lang, more, int(m))
	}
}

var factors = []factor{
	{"sleep", func(d Day) (float64, bool) {
		if d.Sleep == nil {
			return 0, false
		}
		return *d.Sleep, true
	}, func(_ []float64, lang string) (func(float64) bool, string) {
		// Seven hours: the least adults need, by the AASM and Sleep Research Society.
		return func(v float64) bool { return v >= 7 }, l10n.Tr(lang, "slept 7 h or more")
	}},
	{"training", func(d Day) (float64, bool) { return d.Workout, true },
		func(_ []float64, lang string) (func(float64) bool, string) { return any0(lang, l10n.Mark("trained")) }},
	{"focus", func(d Day) (float64, bool) { return d.Focus, true },
		func(_ []float64, lang string) (func(float64) bool, string) {
			return any0(lang, l10n.Mark("had a focus session"))
		}},
	{"habits", func(d Day) (float64, bool) { return d.Habits, true },
		countSplit(l10n.Mark("kept a habit"), l10n.Mark("kept more than %d habits"))},
	{"tasks", func(d Day) (float64, bool) { return d.Tasks, true },
		countSplit(l10n.Mark("finished a task"), l10n.Mark("finished more than %d tasks"))},
	{"gratitude", func(d Day) (float64, bool) {
		if d.Gratitude {
			return 1, true
		}
		return 0, true
	}, func(_ []float64, lang string) (func(float64) bool, string) {
		return any0(lang, l10n.Mark("wrote what you are grateful for"))
	}},
}

// outcomeWords are the outcomes as they read in a sentence.
var outcomeWords = map[string]string{"mood": l10n.Mark("mood"), "energy": l10n.Mark("energy"), "stress": l10n.Mark("stress")}

var outcomes = []struct {
	name string
	get  func(d Day) *float64
}{
	{"mood", func(d Day) *float64 { return d.Mood }},
	{"energy", func(d Day) *float64 { return d.Energy }},
	{"stress", func(d Day) *float64 { return d.Stress }},
}

func seed(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s)) // a hash never fails to write
	return h.Sum64()
}

func mean(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// analyse works out the findings from the window's days.
func analyse(days []Day, lang string) Helps {
	h := Helps{Findings: []Finding{}, Method: l10n.Tr(lang, "Spearman rank correlation between each of these habits and how the day felt, over the last 90 days, with a permutation test (2,000 shuffles) for each pair and the Benjamini–Hochberg correction, since many pairs are tested. Shown only when the link is clear: q ≤ 0.1, a correlation of at least 0.3, at least 14 days, and 5 days on each side. A link is not proof of cause: something else may explain both.")}
	for _, d := range days {
		if d.Mood != nil || d.Energy != nil || d.Stress != nil {
			h.Days++
		}
	}
	if h.Days < minDays {
		h.Needed = minDays - h.Days
		return h
	}
	var found []Finding
	var ps []float64
	for _, fc := range factors {
		for _, oc := range outcomes {
			var x, y []float64
			for _, d := range days {
				v, ok := fc.value(d)
				o := oc.get(d)
				if ok && o != nil {
					x, y = append(x, v), append(y, *o)
				}
			}
			if len(x) < minDays {
				continue
			}
			with, phrase := fc.split(x, lang)
			var hi, lo []float64
			for i := range x {
				if with(x[i]) {
					hi = append(hi, y[i])
				} else {
					lo = append(lo, y[i])
				}
			}
			rho := spearman(x, y)
			if len(hi) < minSide || len(lo) < minSide || math.IsNaN(rho) {
				continue
			}
			p := permutationP(x, y, rounds, seed(fc.name+"/"+oc.name))
			ps = append(ps, p)
			mh, ml := mean(hi), mean(lo)
			found = append(found, Finding{Factor: fc.name, Outcome: oc.name, Rho: math.Round(rho*100) / 100, P: p, N: len(x),
				MeanHigh: math.Round(mh*10) / 10, MeanLow: math.Round(ml*10) / 10, NHigh: len(hi), NLow: len(lo),
				Sentence: l10n.Tr(lang, "On days you %s, your %s averaged %s out of 5; on the other days, %s (%d and %d days).",
					phrase, l10n.Tr(lang, outcomeWords[oc.name]), l10n.Num(lang, mh, 1), l10n.Num(lang, ml, 1), len(hi), len(lo))})
		}
	}
	h.Tested = len(found)
	q := benjaminiHochberg(ps)
	for i := range found {
		found[i].Q = q[i]
		if q[i] <= maxQ && math.Abs(found[i].Rho) >= minRho {
			found[i].Strength = "moderate"
			if math.Abs(found[i].Rho) >= 0.5 {
				found[i].Strength = "strong"
			}
			h.Findings = append(h.Findings, found[i])
		}
	}
	sort.SliceStable(h.Findings, func(a, b int) bool { return math.Abs(h.Findings[a].Rho) > math.Abs(h.Findings[b].Rho) })
	if len(h.Findings) > 6 {
		h.Findings = h.Findings[:6]
	}
	return h
}

// Window reads the last 90 days ending today, in the person's time zone.
func Window(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time, tz string) ([]Day, error) {
	rows, err := q.Query(ctx, `WITH d AS (SELECT generate_series($2::date - 89, $2::date, interval '1 day')::date AS day)
		SELECT d.day::text, j.mood::float8, j.energy::float8, j.stress::float8, j.sleep_hours::float8,
			coalesce(cardinality(j.gratitude), 0) > 0,
			(SELECT coalesce(sum(w.minutes), 0)::float8 FROM workouts w WHERE w.user_id = $1 AND w.done_on = d.day),
			(SELECT coalesce(sum(f.worked_minutes), 0)::float8 FROM focus_sessions f JOIN entities e ON e.id = f.id
				WHERE e.user_id = $1 AND (f.started_at AT TIME ZONE $3)::date = d.day),
			(SELECT count(*)::float8 FROM habit_checkins c JOIN entities e ON e.id = c.habit_id
				WHERE e.user_id = $1 AND c.day = d.day AND c.status IN ('done', 'two_minute')),
			(SELECT count(*)::float8 FROM tasks t JOIN entities e ON e.id = t.id
				WHERE e.user_id = $1 AND (t.completed_at AT TIME ZONE $3)::date = d.day)
		FROM d LEFT JOIN journal_entries j ON j.user_id = $1 AND j.day = d.day ORDER BY d.day`,
		userID, today.Format(time.DateOnly), tz)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Day, error) {
		var d Day
		err := r.Scan(&d.Date, &d.Mood, &d.Energy, &d.Stress, &d.Sleep, &d.Gratitude, &d.Workout, &d.Focus, &d.Habits, &d.Tasks)
		return d, err
	})
}

// WhatHelps is the analysis of the person's last 90 days.
func WhatHelps(ctx context.Context, q db.Querier, userID uuid.UUID, today time.Time, tz, lang string) (Helps, error) {
	days, err := Window(ctx, q, userID, today, tz)
	if err != nil {
		return Helps{}, err
	}
	return analyse(days, lang), nil
}
