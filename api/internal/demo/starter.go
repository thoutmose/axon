package demo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/roadmaps"
)

// Starter gives a real account what the demo is built on, without the
// demo's invented history: the library roadmap started as a plan (its goal,
// milestones, projects, habits, and tasks, as the Start button does), the
// universities and schools behind its courses as resources linked to it,
// and Huberman's protocols as habits. What the account already has, by
// title, is left alone, so it can run twice. It returns what it added.
func Starter(ctx context.Context, pool *pgxpool.Pool, who string, now time.Time) ([]string, error) {
	var added []string
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		user, err := findUser(ctx, tx, who)
		if err != nil {
			return err
		}
		has := func(typ, title string) (uuid.UUID, bool) {
			var id uuid.UUID
			err := tx.QueryRow(ctx, `SELECT id FROM entities WHERE user_id = $1 AND type = $2 AND title = $3 LIMIT 1`, user, typ, title).Scan(&id)
			return id, err == nil
		}

		d := roadmaps.Library()[0]
		roadmap, ok := has("roadmap", d.Title)
		if !ok {
			today, _, err := clock.Today(ctx, tx, user, now)
			if err != nil {
				return err
			}
			plan, err := roadmaps.StartPlan(ctx, tx, user, graph.UserActor, d, d.Plan.Shift(today), now)
			if err != nil {
				return fmt.Errorf("start the roadmap: %w", err)
			}
			roadmap = plan.Roadmap.ID
			added = append(added, fmt.Sprintf("roadmap %q with its plan (%d habits, %d tasks, %d projects)", d.Title, len(plan.Habits), len(plan.Tasks), len(plan.Projects)))
		}
		for _, r := range Schools {
			if _, ok := has("resource", r[0]); ok {
				continue
			}
			res, err := para.Resources.Create(ctx, tx, user, graph.UserActor, para.Fields{Title: ptr(r[0]), URL: ptr(r[1]), Description: ptr(r[2])})
			if err != nil {
				return err
			}
			if _, err := graph.CreateLink(ctx, tx, user, graph.UserActor, res.ID, graph.RelatesTo, roadmap); err != nil {
				return err
			}
			added = append(added, "resource "+r[0])
		}
		for _, p := range Protocols {
			if _, ok := has("habit", p[0]); ok {
				continue
			}
			if _, err := habits.Create(ctx, tx, user, graph.UserActor, habits.Fields{Title: ptr(p[0]), StackAfter: ptr(p[1]), TwoMinute: ptr(p[2]),
				Identity: ptr(p[3]), IntentionTime: ptr(p[4]), IntentionPlace: ptr(p[5])}, now); err != nil {
				return err
			}
			added = append(added, "habit "+p[0])
		}
		return nil
	})
	return added, err
}

// findUser is the account named who (its id or display name), never the demo.
func findUser(ctx context.Context, tx pgx.Tx, who string) (uuid.UUID, error) {
	rows, err := tx.Query(ctx, `SELECT id FROM users WHERE NOT demo AND (id::text = $1 OR display_name = $1)`, who)
	if err != nil {
		return uuid.Nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return uuid.Nil, err
	}
	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		return uuid.Nil, fmt.Errorf("no account %q (give its id or display name; the demo is never one)", who)
	default:
		return uuid.Nil, errors.New("several accounts have that name: give the id")
	}
}
