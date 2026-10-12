package app

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// queryCounter counts the queries the test pool runs.
type queryCounter struct{ n atomic.Int64 }

var queries queryCounter

func (c *queryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.n.Add(1)
	return ctx
}

func (c *queryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// Lists cost the same number of queries for 2 items as for 12: no list runs
// a query per item (an "N+1"), which would slow every page as data grows.
func TestListsDoNotQueryPerItem(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	lists := []struct {
		list   string
		create func(i int) (string, string)
	}{
		{"/v1/tasks?view=all", func(i int) (string, string) {
			return "/v1/tasks", fmt.Sprintf(`{"title":"Task %d","status":"next","due_on":"2026-10-2%d"}`, i, i%10)
		}},
		{"/v1/notes?limit=100", func(i int) (string, string) {
			return "/v1/notes", fmt.Sprintf(`{"path":"Notes/n%d.md","content":"# Note %d\n\nSee [[Note %d]]. #tag%d"}`, i, i, i+1, i%3)
		}},
		{"/v1/habits", func(i int) (string, string) { return "/v1/habits", fmt.Sprintf(`{"title":"Habit %d"}`, i) }},
		{"/v1/goals", func(i int) (string, string) { return "/v1/goals", fmt.Sprintf(`{"title":"Goal %d"}`, i) }},
		{"/v1/projects", func(i int) (string, string) { return "/v1/projects", fmt.Sprintf(`{"title":"Project %d"}`, i) }},
		{"/v1/areas", func(i int) (string, string) { return "/v1/areas", fmt.Sprintf(`{"title":"Area %d"}`, i) }},
		{"/v1/media", func(i int) (string, string) {
			return "/v1/media", fmt.Sprintf(`{"title":"Film %d","kind":"film","status":"done","rating":%d}`, i, 1+i%10)
		}},
		{"/v1/dashboard", func(i int) (string, string) {
			return "/v1/tasks", fmt.Sprintf(`{"title":"Due %d","status":"next","due_on":"%s"}`, i, time.Now().UTC().Format(time.DateOnly))
		}},
		{"/v1/calendar?from=2026-10-01&to=2026-10-31", func(i int) (string, string) {
			return "/v1/tasks", fmt.Sprintf(`{"title":"Dated %d","status":"next","scheduled_on":"2026-10-1%d"}`, i, i%10)
		}},
		{"/v1/palaces", func(i int) (string, string) {
			return "/v1/palaces", fmt.Sprintf(`{"title":"Palace %d","loci":["Door","Hook"]}`, i)
		}},
		{"/v1/budget/accounts", func(i int) (string, string) {
			return "/v1/budget/accounts", fmt.Sprintf(`{"name":"Account %d","type":"asset","opening_cents":%d}`, i, i*100)
		}},
		{"/v1/budget/transactions?month=2026-10", func(i int) (string, string) {
			return "/v1/budget/transactions", fmt.Sprintf(`{"occurred_on":"2026-10-0%d","kind":"expense","amount_cents":%d,"payee":"Shop %d"}`, 1+i%9, 100+i, i)
		}},
		{"/v1/flashcards", func(i int) (string, string) {
			return "/v1/flashcards", fmt.Sprintf(`{"front":"Question %d?","back":"Answer %d","deck":"Deck %d"}`, i, i, i%2)
		}},
	}
	for _, l := range lists {
		add := func(from, to int) {
			for i := from; i < to; i++ {
				path, body := l.create(i)
				alice.mustDo(201, "POST", path, []byte(body))
			}
		}
		count := func() int64 {
			before := queries.n.Load()
			alice.mustDo(200, "GET", l.list, nil)
			return queries.n.Load() - before
		}
		add(0, 2)
		few := count()
		add(2, 12)
		if many := count(); many != few {
			t.Errorf("GET %s: %d queries for 2 items, %d for 12", l.list, few, many)
		}
	}
}
