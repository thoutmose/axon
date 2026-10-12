package habits

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// The habits scorecard (Atomic Habits, chapter 4): list what you do in a
// day, in order, and mark each behaviour good (+), neutral (=), or bad (−).
// Awareness comes before change; nothing here is judged or scored by Axon.

// ScorecardItem is one behaviour in the day.
type ScorecardItem struct {
	Behavior string `json:"behavior"`
	Rating   string `json:"rating"` // "+", "=", or "-"
}

// GetScorecard returns the person's scorecard, in the order of their day.
func GetScorecard(ctx context.Context, q db.Querier, userID uuid.UUID) ([]ScorecardItem, error) {
	items := []ScorecardItem{}
	err := q.QueryRow(ctx, `SELECT items FROM habit_scorecards WHERE user_id = $1`, userID).Scan(&items)
	if errors.Is(err, pgx.ErrNoRows) {
		return []ScorecardItem{}, nil
	}
	return items, err
}

// PutScorecard replaces the scorecard.
func PutScorecard(ctx context.Context, q db.Querier, userID uuid.UUID, items []ScorecardItem) error {
	if len(items) > 100 {
		return ValidationError{"A scorecard has up to 100 behaviours."}
	}
	for i := range items {
		items[i].Behavior = strings.TrimSpace(items[i].Behavior)
		switch {
		case items[i].Behavior == "" || len([]rune(items[i].Behavior)) > 200:
			return ValidationError{"Each behaviour is 1 to 200 characters, like “Wake up” or “Check my phone”."}
		case items[i].Rating != "+" && items[i].Rating != "=" && items[i].Rating != "-":
			return ValidationError{"Rate each behaviour + (good), = (neutral), or - (bad)."}
		}
	}
	if items == nil {
		items = []ScorecardItem{}
	}
	_, err := q.Exec(ctx, `INSERT INTO habit_scorecards (user_id, items) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET items = $2, updated_at = now()`, userID, items)
	return err
}

// ExportScorecard is the scorecard part of a personal data export.
func ExportScorecard(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return GetScorecard(ctx, q, userID)
}

func (h *Handlers) getScorecard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := GetScorecard(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) putScorecard(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Items []ScorecardItem `json:"items"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := PutScorecard(r.Context(), h.db, who.UserID, body.Items); err != nil {
		h.fail(w, err)
		return
	}
	h.getScorecard(w, r)
}
