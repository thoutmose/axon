package media

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
)

// A habit linked to a book or a series, with a number in its name ("Read 20
// pages", "Watch 1 episode"), moves the item's progress by that number each
// day it is done, and back when undone, never past the total (EP-137).
var amount = regexp.MustCompile(`\d+`)

func init() { habits.OnDone = advance }

func advance(ctx context.Context, q db.Querier, userID, habitID uuid.UUID, title string, delta int, now time.Time) {
	n, err := strconv.Atoi(amount.FindString(title))
	if err != nil || n <= 0 {
		return
	}
	b, ok := q.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := Advance(ctx, tx, userID, habitID, delta*n, now); err == nil {
		_ = tx.Commit(ctx)
	}
}

// Advance moves the books and series linked to an entity by by (negative
// goes back), between 0 and their total; a book wanted becomes in progress.
func Advance(ctx context.Context, tx pgx.Tx, userID, entityID uuid.UUID, by int, now time.Time) error {
	rows, err := tx.Query(ctx, `SELECT m.id FROM media_items m JOIN links l
		ON (l.source_id = $2 AND l.target_id = m.id) OR (l.target_id = $2 AND l.source_id = m.id)
		WHERE l.user_id = $1 AND m.kind IN ('book', 'series')`, userID, entityID)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return err
	}
	for _, id := range ids {
		it, err := Get(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		p := max(0, it.Progress+by)
		if it.Total != nil {
			p = min(p, *it.Total)
		}
		f := Fields{Progress: &p}
		if by > 0 && it.Status == "want" {
			doing := "doing"
			f.Status = &doing
		}
		if _, err := Update(ctx, tx, userID, id, graph.UserActor, f, now); err != nil {
			return err
		}
	}
	return nil
}
