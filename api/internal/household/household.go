// Package household lets people on the same Axon share lists (EP-61). A
// person joins a household with a one-time invite code from a member; the
// household's lists (shopping, chores, anything) and their items are what
// members share, and nothing else: tasks, notes, money, and the rest stay
// personal. Anyone can leave at any time.
package household

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Errors callers translate to HTTP responses.
var (
	ErrNone     = errors.New("not in a household")
	ErrAlready  = errors.New("already in a household")
	ErrNotFound = errors.New("not found")
	ErrNotOwner = errors.New("only the household's owner can do that")
	ErrCode     = errors.New("that invite code is not valid, was used, or has expired")
)

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Member is someone in the household, as other members see them.
type Member struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Owner    bool      `json:"owner"`
	JoinedAt time.Time `json:"joined_at"`
}

// Household is the person's household with its members and lists.
type Household struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Members []Member  `json:"members"`
	Lists   []List    `json:"lists"`
}

// List is a shared list with its items.
type List struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Kind  string    `json:"kind"`
	Open  int       `json:"open"`
	Items []Item    `json:"items"`
}

// Item is one entry of a list.
type Item struct {
	ID        uuid.UUID  `json:"id"`
	ListID    uuid.UUID  `json:"list_id"`
	Title     string     `json:"title"`
	DueOn     *string    `json:"due_on"`
	Assignee  *uuid.UUID `json:"assignee"`
	CreatedBy *uuid.UUID `json:"created_by"`
	DoneBy    *uuid.UUID `json:"done_by"`
	DoneAt    *time.Time `json:"done_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// mine returns the person's household and whether they own it.
func mine(ctx context.Context, q db.Querier, userID uuid.UUID) (uuid.UUID, bool, error) {
	var id uuid.UUID
	var owner bool
	err := q.QueryRow(ctx, `SELECT household_id, owner FROM household_members WHERE user_id = $1`, userID).Scan(&id, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return id, false, ErrNone
	}
	return id, owner, err
}

func cleanName(s string, max int, what string) (string, error) {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" || len([]rune(s)) > max {
		return "", ValidationError{what}
	}
	return s, nil
}

// Get returns the person's household with its members and lists; done
// items are kept for a week, so a mistaken tick can be undone.
func Get(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Household, error) {
	hid, _, err := mine(ctx, q, userID)
	if err != nil {
		return Household{}, err
	}
	h := Household{ID: hid, Members: []Member{}, Lists: []List{}}
	if err := q.QueryRow(ctx, `SELECT name FROM households WHERE id = $1`, hid).Scan(&h.Name); err != nil {
		return h, err
	}
	rows, err := q.Query(ctx, `SELECT m.user_id, u.display_name, m.owner, m.joined_at FROM household_members m JOIN users u ON u.id = m.user_id
		WHERE m.household_id = $1 ORDER BY m.joined_at`, hid)
	if err != nil {
		return h, err
	}
	if h.Members, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Member]); err != nil {
		return h, err
	}
	rows, err = q.Query(ctx, `SELECT id, title, kind FROM household_lists WHERE household_id = $1 ORDER BY created_at`, hid)
	if err != nil {
		return h, err
	}
	lists, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (List, error) {
		var l List
		err := r.Scan(&l.ID, &l.Title, &l.Kind)
		l.Items = []Item{}
		return l, err
	})
	if err != nil {
		return h, err
	}
	rows, err = q.Query(ctx, `SELECT i.id, i.list_id, i.title, to_char(i.due_on, 'YYYY-MM-DD'), i.assignee, i.created_by, i.done_by, i.done_at, i.created_at
		FROM household_items i JOIN household_lists l ON l.id = i.list_id
		WHERE l.household_id = $1 AND (i.done_at IS NULL OR i.done_at > $2)
		ORDER BY i.done_at NULLS FIRST, i.due_on NULLS LAST, i.created_at LIMIT 2000`, hid, now.AddDate(0, 0, -7))
	if err != nil {
		return h, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Item])
	if err != nil {
		return h, err
	}
	byList := map[uuid.UUID]int{}
	for i, l := range lists {
		byList[l.ID] = i
	}
	for _, it := range items {
		l := &lists[byList[it.ListID]]
		l.Items = append(l.Items, it)
		if it.DoneAt == nil {
			l.Open++
		}
	}
	h.Lists = lists
	return h, nil
}

// Create starts a household with the person as its owner.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, name string) error {
	name, err := cleanName(name, 100, "A household's name is 1 to 100 characters.")
	if err != nil {
		return err
	}
	if _, _, err := mine(ctx, tx, userID); err == nil {
		return ErrAlready
	} else if !errors.Is(err, ErrNone) {
		return err
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := tx.Exec(ctx, `INSERT INTO households (id, name) VALUES ($1, $2)`, id, name); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO household_members (household_id, user_id, owner) VALUES ($1, $2, true)`, id, userID)
	return err
}

// Rename changes the household's name; any member can.
func Rename(ctx context.Context, q db.Querier, userID uuid.UUID, name string) error {
	name, err := cleanName(name, 100, "A household's name is 1 to 100 characters.")
	if err != nil {
		return err
	}
	hid, _, err := mine(ctx, q, userID)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `UPDATE households SET name = $2 WHERE id = $1`, hid, name)
	return err
}

var codeAlphabet = base32.NewEncoding("ABCDEFGHJKLMNPQRSTUVWXYZ23456789").WithPadding(base32.NoPadding)

func hash(code string) []byte {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))))
	return sum[:]
}

// Invite makes a one-time code, valid for a week, shown only now.
func Invite(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (string, time.Time, error) {
	hid, _, err := mine(ctx, q, userID)
	if err != nil {
		return "", time.Time{}, err
	}
	raw := make([]byte, 10)
	_, _ = rand.Read(raw)
	code := codeAlphabet.EncodeToString(raw) // 16 characters
	code = code[:4] + "-" + code[4:8] + "-" + code[8:12] + "-" + code[12:16]
	expires := now.Add(7 * 24 * time.Hour)
	_, err = q.Exec(ctx, `INSERT INTO household_invites (code_hash, household_id, created_by, expires_at) VALUES ($1, $2, $3, $4)`,
		hash(code), hid, userID, expires)
	return code, expires, err
}

// Join adds the person to the household of an invite code.
func Join(ctx context.Context, tx pgx.Tx, userID uuid.UUID, code string, now time.Time) error {
	if _, _, err := mine(ctx, tx, userID); err == nil {
		return ErrAlready
	} else if !errors.Is(err, ErrNone) {
		return err
	}
	var hid uuid.UUID
	err := tx.QueryRow(ctx, `UPDATE household_invites SET used_at = $2 WHERE code_hash = $1 AND used_at IS NULL AND expires_at > $2
		RETURNING household_id`, hash(code), now).Scan(&hid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCode
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO household_members (household_id, user_id) VALUES ($1, $2)`, hid, userID)
	return err
}

// Leave takes the person out. The last one out deletes the household;
// when the owner leaves, the longest-standing member becomes the owner.
func Leave(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error {
	hid, owner, err := mine(ctx, tx, userID)
	if err != nil {
		return err
	}
	return removeMember(ctx, tx, hid, userID, owner)
}

func removeMember(ctx context.Context, tx pgx.Tx, hid, userID uuid.UUID, owner bool) error {
	if _, err := tx.Exec(ctx, `DELETE FROM household_members WHERE household_id = $1 AND user_id = $2`, hid, userID); err != nil {
		return err
	}
	// What they were asked to do is no longer theirs.
	if _, err := tx.Exec(ctx, `UPDATE household_items i SET assignee = NULL FROM household_lists l
		WHERE l.id = i.list_id AND l.household_id = $1 AND i.assignee = $2`, hid, userID); err != nil {
		return err
	}
	var left int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM household_members WHERE household_id = $1`, hid).Scan(&left); err != nil {
		return err
	}
	if left == 0 {
		_, err := tx.Exec(ctx, `DELETE FROM households WHERE id = $1`, hid)
		return err
	}
	if owner {
		_, err := tx.Exec(ctx, `UPDATE household_members SET owner = true WHERE household_id = $1 AND user_id =
			(SELECT user_id FROM household_members WHERE household_id = $1 ORDER BY joined_at LIMIT 1)`, hid)
		return err
	}
	return nil
}

// Remove takes another member out; only the owner can.
func Remove(ctx context.Context, tx pgx.Tx, userID, member uuid.UUID) error {
	hid, owner, err := mine(ctx, tx, userID)
	if err != nil {
		return err
	}
	if !owner {
		return ErrNotOwner
	}
	if member == userID {
		return removeMember(ctx, tx, hid, userID, true)
	}
	var in bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM household_members WHERE household_id = $1 AND user_id = $2)`, hid, member).Scan(&in); err != nil {
		return err
	}
	if !in {
		return ErrNotFound
	}
	return removeMember(ctx, tx, hid, member, false)
}
