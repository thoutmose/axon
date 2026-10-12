// Package reminders sends the morning digest and the evening nudge
// (EP-56): Web Push to the person's devices, and email when the instance
// has a mail server. Each goes out once a day, at the person's local time,
// only when there is something to say.
package reminders

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/learning"
	"github.com/thoutmose/axon/api/internal/push"
)

// Message is one reminder.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"` // a newer reminder of the same kind replaces the older one
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// Morning is the day's digest: tasks, reviews, notes to recall, habits.
// It is empty when there is nothing to do.
func Morning(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Message, error) {
	day, _, err := habits.Today(ctx, q, userID, now)
	if err != nil {
		return Message{}, err
	}
	var today, overdue int
	if err := q.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE t.status NOT IN ('done', 'cancelled', 'someday') AND (t.scheduled_on <= $2 OR t.due_on <= $2)),
		count(*) FILTER (WHERE t.status NOT IN ('done', 'cancelled', 'someday') AND t.due_on < $2)
		FROM tasks t JOIN entities e ON e.id = t.id WHERE e.user_id = $1`, userID, day).Scan(&today, &overdue); err != nil {
		return Message{}, err
	}
	plan, err := learning.Today(ctx, q, userID, now)
	if err != nil {
		return Message{}, err
	}
	hs, err := habits.List(ctx, q, userID, "active", now)
	if err != nil {
		return Message{}, err
	}
	scheduled := 0
	for _, h := range hs {
		if h.State.ScheduledToday && h.State.CheckedIn == nil {
			scheduled++
		}
	}
	var parts []string
	if today > 0 {
		s := plural(today, "task", "tasks")
		if overdue > 0 {
			s += fmt.Sprintf(" (%d overdue)", overdue)
		}
		parts = append(parts, s)
	}
	if plan.Cards.Due > 0 {
		parts = append(parts, fmt.Sprintf("%s to review, about %d min", plural(plan.Cards.Due, "card", "cards"), plan.Cards.Minutes))
	}
	if n := len(plan.Recall.Due); n > 0 {
		parts = append(parts, plural(n, "note", "notes")+" to recall")
	}
	if scheduled > 0 {
		parts = append(parts, plural(scheduled, "habit", "habits"))
	}
	if plan.Test.Due {
		parts = append(parts, "the weekly test")
	}
	if len(parts) == 0 {
		return Message{}, nil
	}
	return Message{Title: "Today in Axon", Body: strings.Join(parts, " · "), URL: "/today", Tag: "morning"}, nil
}

// Evening names the habits scheduled today and not checked in yet, and the
// cards still due. It is empty when the day is done.
func Evening(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Message, error) {
	hs, err := habits.List(ctx, q, userID, "active", now)
	if err != nil {
		return Message{}, err
	}
	var open []string
	for _, h := range hs {
		if h.State.ScheduledToday && h.State.CheckedIn == nil {
			label := h.Title
			if h.TwoMinute != "" {
				label += " (or just: " + h.TwoMinute + ")"
			}
			open = append(open, label)
		}
	}
	due, err := cards.DueCount(ctx, q, userID, now)
	if err != nil {
		return Message{}, err
	}
	var parts []string
	if len(open) > 0 {
		if len(open) > 3 {
			open = append(open[:3], fmt.Sprintf("%d more", len(open)-3))
		}
		parts = append(parts, "Still open today: "+strings.Join(open, ", "))
	}
	if due > 0 {
		parts = append(parts, plural(due, "card", "cards")+" still to review")
	}
	if len(parts) == 0 {
		return Message{}, nil
	}
	url := "/habits"
	if len(open) == 0 {
		url = "/cards/review"
	}
	return Message{Title: "Before the day ends", Body: strings.Join(parts, ". ") + ".", URL: url, Tag: "evening"}, nil
}

// VAPIDKey returns the instance's push signing key, making it on first use.
func VAPIDKey(ctx context.Context, pool *pgxpool.Pool, sealer *auth.Sealer) (*ecdsa.PrivateKey, error) {
	for range 2 {
		var stored []byte
		err := pool.QueryRow(ctx, `SELECT value FROM instance_keys WHERE name = 'vapid'`).Scan(&stored)
		if errors.Is(err, pgx.ErrNoRows) {
			key, err := push.NewKey()
			if err != nil {
				return nil, err
			}
			der, err := x509.MarshalECPrivateKey(key)
			if err != nil {
				return nil, err
			}
			value := append([]byte{0}, der...)
			if sealer != nil {
				value = append([]byte{1}, sealer.Seal(der)...)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO instance_keys (name, value) VALUES ('vapid', $1) ON CONFLICT DO NOTHING`, value); err != nil {
				return nil, err
			}
			continue // read back whichever key won
		}
		if err != nil {
			return nil, err
		}
		der := stored[1:]
		if stored[0] == 1 {
			if sealer == nil {
				return nil, errors.New("the push key is sealed and AXON_ENCRYPTION_KEY is not set")
			}
			if der, err = sealer.Open(der); err != nil {
				return nil, err
			}
		}
		return x509.ParseECPrivateKey(der)
	}
	return nil, errors.New("push key: not stored")
}
