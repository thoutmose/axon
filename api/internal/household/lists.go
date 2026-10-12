package household

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

var kinds = map[string]bool{"shopping": true, "chores": true, "other": true}

// CreateList adds a list to the person's household.
func CreateList(ctx context.Context, q db.Querier, userID uuid.UUID, title, kind string) (uuid.UUID, error) {
	hid, _, err := mine(ctx, q, userID)
	if err != nil {
		return uuid.Nil, err
	}
	title, err = cleanName(title, 100, "A list's name is 1 to 100 characters.")
	if err != nil {
		return uuid.Nil, err
	}
	if kind == "" {
		kind = "other"
	}
	if !kinds[kind] {
		return uuid.Nil, ValidationError{"A list is for shopping, chores, or other."}
	}
	var n int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM household_lists WHERE household_id = $1`, hid).Scan(&n); err != nil {
		return uuid.Nil, err
	}
	if n >= 50 {
		return uuid.Nil, ValidationError{"A household has up to 50 lists."}
	}
	id := uuid.Must(uuid.NewV7())
	_, err = q.Exec(ctx, `INSERT INTO household_lists (id, household_id, title, kind) VALUES ($1, $2, $3, $4)`, id, hid, title, kind)
	return id, err
}

// list checks that a list belongs to the person's household.
func list(ctx context.Context, q db.Querier, userID, listID uuid.UUID) error {
	var ok bool
	err := q.QueryRow(ctx, `SELECT true FROM household_lists l JOIN household_members m ON m.household_id = l.household_id
		WHERE l.id = $1 AND m.user_id = $2`, listID, userID).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// RenameList renames a list.
func RenameList(ctx context.Context, q db.Querier, userID, listID uuid.UUID, title string) error {
	if err := list(ctx, q, userID, listID); err != nil {
		return err
	}
	title, err := cleanName(title, 100, "A list's name is 1 to 100 characters.")
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `UPDATE household_lists SET title = $2 WHERE id = $1`, listID, title)
	return err
}

// DeleteList removes a list and its items.
func DeleteList(ctx context.Context, q db.Querier, userID, listID uuid.UUID) error {
	if err := list(ctx, q, userID, listID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `DELETE FROM household_lists WHERE id = $1`, listID)
	return err
}

// ItemFields are the editable parts of an item. Assignee "" clears it;
// DueOn "" clears it.
type ItemFields struct {
	Title    *string `json:"title"`
	DueOn    *string `json:"due_on"`
	Assignee *string `json:"assignee"`
	Done     *bool   `json:"done"`
}

func (f ItemFields) check(ctx context.Context, q db.Querier, userID uuid.UUID) (*string, *time.Time, *uuid.UUID, error) {
	var title *string
	if f.Title != nil {
		t, err := cleanName(*f.Title, 300, "An item is 1 to 300 characters.")
		if err != nil {
			return nil, nil, nil, err
		}
		title = &t
	}
	var due *time.Time
	if f.DueOn != nil && *f.DueOn != "" {
		d, err := time.Parse(time.DateOnly, *f.DueOn)
		if err != nil {
			return nil, nil, nil, ValidationError{"A due date looks like 2026-10-14."}
		}
		due = &d
	}
	var assignee *uuid.UUID
	if f.Assignee != nil && *f.Assignee != "" {
		a, err := uuid.Parse(*f.Assignee)
		if err != nil {
			return nil, nil, nil, ValidationError{"Assign items to a member of the household."}
		}
		var same bool
		if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM household_members a JOIN household_members b ON a.household_id = b.household_id
			WHERE a.user_id = $1 AND b.user_id = $2)`, userID, a).Scan(&same); err != nil {
			return nil, nil, nil, err
		}
		if !same {
			return nil, nil, nil, ValidationError{"Assign items to a member of the household."}
		}
		assignee = &a
	}
	return title, due, assignee, nil
}

// AddItem adds an item to a list.
func AddItem(ctx context.Context, q db.Querier, userID, listID uuid.UUID, f ItemFields) (uuid.UUID, error) {
	if err := list(ctx, q, userID, listID); err != nil {
		return uuid.Nil, err
	}
	if f.Title == nil {
		return uuid.Nil, ValidationError{"An item is 1 to 300 characters."}
	}
	title, due, assignee, err := f.check(ctx, q, userID)
	if err != nil {
		return uuid.Nil, err
	}
	var n int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM household_items WHERE list_id = $1 AND done_at IS NULL`, listID).Scan(&n); err != nil {
		return uuid.Nil, err
	}
	if n >= 500 {
		return uuid.Nil, ValidationError{"A list has up to 500 open items."}
	}
	id := uuid.Must(uuid.NewV7())
	_, err = q.Exec(ctx, `INSERT INTO household_items (id, list_id, title, due_on, assignee, created_by) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, listID, *title, due, assignee, userID)
	return id, err
}

func item(ctx context.Context, q db.Querier, userID, itemID uuid.UUID) error {
	var ok bool
	err := q.QueryRow(ctx, `SELECT true FROM household_items i JOIN household_lists l ON l.id = i.list_id
		JOIN household_members m ON m.household_id = l.household_id WHERE i.id = $1 AND m.user_id = $2`, itemID, userID).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// UpdateItem changes, ticks, or unticks an item.
func UpdateItem(ctx context.Context, q db.Querier, userID, itemID uuid.UUID, f ItemFields, now time.Time) error {
	if err := item(ctx, q, userID, itemID); err != nil {
		return err
	}
	title, due, assignee, err := f.check(ctx, q, userID)
	if err != nil {
		return err
	}
	if title != nil {
		if _, err := q.Exec(ctx, `UPDATE household_items SET title = $2, updated_at = now() WHERE id = $1`, itemID, *title); err != nil {
			return err
		}
	}
	if f.DueOn != nil {
		if _, err := q.Exec(ctx, `UPDATE household_items SET due_on = $2, updated_at = now() WHERE id = $1`, itemID, due); err != nil {
			return err
		}
	}
	if f.Assignee != nil {
		if _, err := q.Exec(ctx, `UPDATE household_items SET assignee = $2, updated_at = now() WHERE id = $1`, itemID, assignee); err != nil {
			return err
		}
	}
	if f.Done != nil {
		if *f.Done {
			_, err = q.Exec(ctx, `UPDATE household_items SET done_at = coalesce(done_at, $2), done_by = coalesce(done_by, $3), updated_at = now() WHERE id = $1`, itemID, now, userID)
		} else {
			_, err = q.Exec(ctx, `UPDATE household_items SET done_at = NULL, done_by = NULL, updated_at = now() WHERE id = $1`, itemID)
		}
	}
	return err
}

// DeleteItem removes an item.
func DeleteItem(ctx context.Context, q db.Querier, userID, itemID uuid.UUID) error {
	if err := item(ctx, q, userID, itemID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `DELETE FROM household_items WHERE id = $1`, itemID)
	return err
}

// MineToday counts open items assigned to the person, due by day, for Today.
func MineToday(ctx context.Context, q db.Querier, userID uuid.UUID, day time.Time) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT count(*) FROM household_items WHERE assignee = $1 AND done_at IS NULL AND (due_on IS NULL OR due_on <= $2)`, userID, day).Scan(&n)
	return n, err
}

// Export is what a personal data export holds: the household's name, and
// the lists and the items the person added or was given; other members'
// names stay out.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	hid, _, err := mine(ctx, q, userID)
	if errors.Is(err, ErrNone) {
		return nil, nil //nolint:nilnil // no household: nothing to export
	}
	if err != nil {
		return nil, err
	}
	var name string
	if err := q.QueryRow(ctx, `SELECT name FROM households WHERE id = $1`, hid).Scan(&name); err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT l.title, i.title, to_char(i.due_on, 'YYYY-MM-DD'), i.done_at, i.created_at FROM household_items i
		JOIN household_lists l ON l.id = i.list_id WHERE l.household_id = $1 AND (i.created_by = $2 OR i.assignee = $2) ORDER BY i.created_at`, hid, userID)
	if err != nil {
		return nil, err
	}
	type row struct {
		List      string     `json:"list"`
		Title     string     `json:"title"`
		DueOn     *string    `json:"due_on"`
		DoneAt    *time.Time `json:"done_at"`
		CreatedAt time.Time  `json:"created_at"`
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
	return map[string]any{"household": name, "items": items}, err
}
