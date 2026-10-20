package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/health"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/textx"
)

// Meal planning for the Chef & Nutritionist (EP-90): the agent reads the
// person's daily needs and proposes a plan, which becomes a note in
// "Meal plans/" once they approve it.

// MealPlanFolder is where approved meal plans are kept.
const MealPlanFolder = "Meal plans"

type mealPlanInput struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
	Reason   string `json:"reason"`
}

func (in *mealPlanInput) check() error {
	in.Title, in.Markdown = strings.TrimSpace(in.Title), strings.TrimSpace(in.Markdown)
	switch {
	case in.Title == "" || len([]rune(in.Title)) > 120:
		return toolErrorf("title is 1 to 120 characters")
	case in.Markdown == "" || len(in.Markdown) > 40_000:
		return toolErrorf("markdown is 1 to 40,000 characters")
	}
	return nil
}

// mealPlanNote is the note an approved plan becomes.
func mealPlanNote(in mealPlanInput, day time.Time) (path, content string) {
	stem := notes.Slug(in.Title)
	path = fmt.Sprintf("%s/%s-%s.md", MealPlanFolder, day.Format(time.DateOnly), stem)
	content = fmt.Sprintf("---\ntype: meal-plan\ncreated: %s\n---\n# %s\n\n%s\n\n"+
		"> Planned by the Chef & Nutritionist agent. Amounts, nutrients, and prices are estimates; check labels for allergens. "+
		"For a medical condition, pregnancy, or an eating disorder, ask a registered dietitian.\n",
		day.Format(time.DateOnly), in.Title, in.Markdown)
	return path, content
}

func init() {
	registry["daily_needs"] = Tool{
		Name: "daily_needs",
		Description: "The person's daily reference intakes (energy, protein, fat, carbohydrates, fibre, water) for their goal (maintain, lose gently, or gain), " +
			"worked out and bounded by Axon from their profile, never under their basal rate or 1,200 kcal; or why there are none (no profile yet, under 18, or pregnancy: then use no numbers).",
		Properties: map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, _ json.RawMessage) (any, error) {
			if err := healthAllowed(ctx, q, userID); err != nil {
				return nil, err
			}
			p, err := health.GetProfile(ctx, q, userID)
			if err != nil {
				return nil, err
			}
			if p == nil {
				return map[string]any{"available": false, "reason": "no_profile"}, nil
			}
			return health.Calc(*p, time.Now().Year(), "en"), nil
		},
	}
	registry["draft_meal_plan"] = Tool{
		Name: "draft_meal_plan",
		Description: "Propose a meal plan, saved as a note in \"" + MealPlanFolder + "/\" once the person approves it. markdown holds " +
			"each day and meal with its recipe, then the shopping list as checkboxes and the estimated cost. Nothing is saved until the person approves.",
		Properties: map[string]any{
			"title":    map[string]any{"type": "string", "maxLength": 120, "description": "For example \"Week of 14 October, 2 people\"."},
			"markdown": map[string]any{"type": "string", "maxLength": 40000},
			"reason":   reason,
		},
		Required: []string{"title", "markdown", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in mealPlanInput
			if err := decode(raw, &in); err != nil {
				return "", err
			}
			if err := in.check(); err != nil {
				return "", err
			}
			return fmt.Sprintf("Save the meal plan “%s” as a note in %s/: %s", in.Title, MealPlanFolder, textx.Cut(in.Reason, 200)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in mealPlanInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if err := in.check(); err != nil {
				return err
			}
			path, content := mealPlanNote(in, time.Now())
			// A second plan with the same title on the same day gets -2, -3…
			// (checked first: a failed insert would abort the transaction).
			base := strings.TrimSuffix(path, ".md")
			for i := 2; ; i++ {
				var taken bool
				if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM notes WHERE user_id = $1 AND path = $2)`, userID, path).Scan(&taken); err != nil {
					return err
				}
				if !taken {
					break
				}
				path = fmt.Sprintf("%s-%d.md", base, i)
			}
			_, err := notes.Create(ctx, tx, userID, actor, path, content)
			return err
		},
	}
}
