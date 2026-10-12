package agents

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/health"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// Health tools for the coach agents (EP-44). They read the person's logs
// only while both health_data and health_ai are granted, checked on every
// call, not just when the run starts.

func healthAllowed(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	for _, p := range []string{health.Consent, "health_ai"} {
		ok, err := privacy.Granted(ctx, q, userID, p)
		if err != nil {
			return err
		}
		if !ok {
			return toolErrorf("the person has not allowed their health logs to be shared with you; say so and offer general guidance only")
		}
	}
	return nil
}

func days(raw json.RawMessage, max int) (int, error) {
	var in struct {
		Days int `json:"days"`
	}
	if err := decode(raw, &in); err != nil {
		return 0, err
	}
	if in.Days == 0 {
		in.Days = 14
	}
	if in.Days < 1 || in.Days > max {
		return 0, toolErrorf("days is 1 to %d", max)
	}
	return in.Days, nil
}

func init() {
	registry["training_log"] = Tool{
		Name:        "training_log",
		Description: "The person's workouts over the last days (default 14, up to 90): date, kind, title, minutes, effort (RPE 1-10), and sets.",
		Properties:  map[string]any{"days": map[string]any{"type": "integer", "minimum": 1, "maximum": 90}},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			n, err := days(raw, 90)
			if err != nil {
				return nil, err
			}
			if err := healthAllowed(ctx, q, userID); err != nil {
				return nil, err
			}
			today := time.Now().UTC()
			return health.Workouts(ctx, q, userID, today.AddDate(0, 0, -n+1).Format(time.DateOnly), today.Format(time.DateOnly))
		},
	}
	registry["food_log"] = Tool{
		Name:        "food_log",
		Description: "The person's food log over the last days (default 7, up to 30): each day's entries (meal, name, grams, kcal, macros) and totals.",
		Properties:  map[string]any{"days": map[string]any{"type": "integer", "minimum": 1, "maximum": 30}},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			n, err := days(raw, 30)
			if err != nil {
				return nil, err
			}
			if err := healthAllowed(ctx, q, userID); err != nil {
				return nil, err
			}
			today := time.Now().UTC()
			out := []health.Day{}
			for i := n - 1; i >= 0; i-- {
				d, err := health.FoodDay(ctx, q, userID, today.AddDate(0, 0, -i).Format(time.DateOnly))
				if err != nil {
					return nil, err
				}
				if len(d.Entries) > 0 {
					out = append(out, d)
				}
			}
			return out, nil
		},
	}
}
