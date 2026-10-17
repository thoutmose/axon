package demo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/budget"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/roadmaps"
)

// Starter gives a real account what the demo is built on, without the
// demo's invented history: the library roadmap started as a plan (its goal,
// milestones, projects, habits, and tasks, as the Start button does), the
// first course's literature note and its flashcards, the learning notes
// linked to it, the universities and schools behind its courses as
// resources linked to it, its funds (empty), the skills assessment, and
// Huberman's protocols as habits. What the account already has, by
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
		if _, err := notes.GetByPath(ctx, tx, user, LiteratureNote); errors.Is(err, graph.ErrNotFound) {
			r, err := roadmaps.Get(ctx, tx, user, roadmap)
			if err != nil {
				return err
			}
			deck, err := seedLiterature(ctx, tx, user, r.Skills, now)
			if err != nil {
				return err
			}
			added = append(added, fmt.Sprintf("note %s with %d flashcards", LiteratureNote, len(deck)))
		} else if err != nil {
			return err
		}
		for i, n := range LearningNotes {
			if _, err := notes.GetByPath(ctx, tx, user, n[0]); !errors.Is(err, graph.ErrNotFound) {
				if err != nil {
					return err
				}
				continue
			}
			note, err := notes.Create(ctx, tx, user, graph.UserActor, n[0], n[1])
			if err != nil {
				return err
			}
			if i == 0 { // the map of how to learn
				r, err := roadmaps.Get(ctx, tx, user, roadmap)
				if err != nil {
					return err
				}
				if _, err := graph.CreateLink(ctx, tx, user, graph.UserActor, note.ID, graph.RelatesTo, roadmap); err != nil {
					return err
				}
				for _, sk := range r.Skills {
					if sk.Title == "Learning how to learn" {
						if _, err := graph.CreateLink(ctx, tx, user, graph.UserActor, sk.ID, graph.RelatesTo, note.ID); err != nil {
							return err
						}
					}
				}
			}
			added = append(added, "note "+n[0])
		}
		for _, f := range Funds {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM budget_pots WHERE user_id = $1 AND name = $2)`, user, *f.Name).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
			today, _, err := clock.Today(ctx, tx, user, now)
			if err != nil {
				return err
			}
			if _, err := budget.CreatePot(ctx, tx, user, f, today); err != nil {
				return err
			}
			added = append(added, "pot "+*f.Name)
		}
		if done, err := roadmaps.ListAssessments(ctx, tx, user); err != nil {
			return err
		} else if len(done) == 0 {
			if _, err := roadmaps.CreateAssessment(ctx, tx, user, graph.UserActor, assessment(), now); err != nil {
				return err
			}
			added = append(added, "skills assessment")
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
