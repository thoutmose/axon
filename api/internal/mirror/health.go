package mirror

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/health"
)

// Health notes are written one way, from Axon to the vault, and only when
// the person turned them on with the health consent given. Edits in the
// vault are not read back.

const healthFooter = "Written from your health log in Axon; edit it there, changes here are not read back."

func num(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}

func cell(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "|", "/"), "\n", " ") }

// renderFoodDay returns the day's food log, or ok=false when the day is empty.
func renderFoodDay(ctx context.Context, q db.Querier, user uuid.UUID, day string) (rendered, bool, error) {
	d, err := health.FoodDay(ctx, q, user, day)
	if err != nil || len(d.Entries) == 0 {
		return rendered{}, false, err
	}
	rows := []string{"| Meal | Food | Grams | kcal | Protein (g) | Carbs (g) | Fat (g) |", "|---|---|---|---|---|---|---|"}
	for _, e := range d.Entries {
		rows = append(rows, fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |", e.Meal, cell(e.Name), num(e.Grams), num(e.Kcal), num(e.Protein), num(e.Carbs), num(e.Fat)))
	}
	t := d.Totals
	totals := fmt.Sprintf("- Energy: %g kcal\n- Protein: %g g\n- Carbohydrates: %g g\n- Fat: %g g", t.Kcal, t.Protein, t.Carbs, t.Fat)
	return rendered{folder: "Health/Food", page: page{fm: []kv{{"type", "food-log"}, {"date", day}}, title: "Food " + day, footer: healthFooter,
		sections: []section{{heading: "Meals", body: strings.Join(rows, "\n"), readOnly: true}, {heading: "Totals", body: totals, readOnly: true}}}}, true, nil
}

// renderWorkout returns one workout, or ok=false when it is gone.
func renderWorkout(ctx context.Context, q db.Querier, user, id uuid.UUID) (rendered, bool, error) {
	w, err := health.GetWorkout(ctx, q, user, id)
	if errors.Is(err, health.ErrNotFound) {
		return rendered{}, false, nil
	}
	if err != nil {
		return rendered{}, false, err
	}
	minutes, effort := "", ""
	if w.Minutes != nil {
		minutes = strconv.Itoa(*w.Minutes)
	}
	if w.Effort != nil {
		effort = strconv.Itoa(*w.Effort)
	}
	rows := []string{"| Exercise | Reps | Weight (kg) | Distance (km) | Time |", "|---|---|---|---|---|"}
	for _, s := range w.Sets {
		reps, secs := "", ""
		if s.Reps != nil {
			reps = strconv.Itoa(*s.Reps)
		}
		if s.Seconds != nil {
			secs = (time.Duration(*s.Seconds) * time.Second).String()
		}
		rows = append(rows, fmt.Sprintf("| %s | %s | %s | %s | %s |", cell(s.Exercise), reps, num(s.WeightKg), num(s.DistanceKm), secs))
	}
	secs := []section{}
	if len(w.Sets) > 0 {
		secs = append(secs, section{heading: "Sets", body: strings.Join(rows, "\n"), readOnly: true})
	}
	if w.Notes != "" {
		secs = append(secs, section{heading: "Notes", body: w.Notes, readOnly: true})
	}
	return rendered{folder: "Health/Training", page: page{fm: []kv{{"type", "workout"}, {"date", w.DoneOn}, {"kind", w.Kind},
		{"minutes", minutes}, {"effort", effort}}, title: w.DoneOn + " " + w.Title, footer: healthFooter, sections: secs}}, true, nil
}
