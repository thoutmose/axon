// Package habits implements habits designed with cues, stacks, and two-minute
// versions, daily check-ins on local calendar days, and recovery after a miss
// instead of punishment (EP-12).
package habits

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Type is the graph entity type of a habit.
const Type = "habit"

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "habits", Export: export})
}

// Errors callers translate to HTTP responses.
var ErrStale = errors.New("stale version")

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

var statuses = []string{"active", "paused", "archived"}

// Habit is the API representation; State is computed for "today".
type Habit struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"` // the response: what you do
	Cue        string    `json:"cue"`
	Craving    string    `json:"craving"`
	Reward     string    `json:"reward"`
	StackAfter string    `json:"stack_after"`
	TwoMinute  string    `json:"two_minute"`
	Days       []int     `json:"days"`
	Status     string    `json:"status"`
	// Atomic Habits (EP-34). For a habit to break, cue, craving, two_minute,
	// and reward hold the inverted laws: invisible, unattractive, difficult,
	// and unsatisfying.
	Kind            string    `json:"kind"`     // build or break
	Identity        string    `json:"identity"` // "I am a person who…"
	IntentionTime   string    `json:"intention_time"`
	IntentionPlace  string    `json:"intention_place"`
	ContractPartner string    `json:"contract_partner"`
	ContractPledge  string    `json:"contract_pledge"`
	Votes           int       `json:"votes"` // check-ins kept so far: each one is a vote for the identity
	State           State     `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// State is where a habit stands today, in the user's time zone.
type State struct {
	Today          tasks.Date `json:"today"`
	ScheduledToday bool       `json:"scheduled_today"`
	CheckedIn      *string    `json:"checked_in"` // today's check-in status, if any
	Streak         int        `json:"streak"`     // scheduled days kept in a row; rest days neither count nor break it
	Recovery       *Recovery  `json:"recovery"`   // set when the last scheduled day was missed
}

// Recovery is a gentle prompt after a miss: start small again.
type Recovery struct {
	Missed  int    `json:"missed"` // scheduled days missed in a row (capped at 7)
	Message string `json:"message"`
}

// lookback bounds how far state computation reads history: streaks are
// counted up to a year.
const lookback = 366

func scheduled(days []int, d time.Time) bool {
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7
	}
	return slices.Contains(days, wd)
}

// BestRun is the longest run of scheduled days kept, from the habit's
// creation to today: a two-minute version counts, a planned rest day
// neither counts nor breaks it, a missed day ends it (EP-119).
func BestRun(days []int, checkins map[string]string, created, today time.Time) int {
	best, run := 0, 0
	for d := created; !d.After(today); d = d.AddDate(0, 0, 1) {
		if !scheduled(days, d) {
			continue
		}
		switch checkins[d.Format(time.DateOnly)] {
		case "done", "two_minute":
			run++
			best = max(best, run)
		case "skipped":
		default:
			if !d.Equal(today) { // today is not over yet
				run = 0
			}
		}
	}
	return best
}

// Compute derives today's state from check-ins keyed by local date
// (YYYY-MM-DD). Days before the habit existed never count as missed.
// It is pure date arithmetic, so DST changes and travel cannot shift it.
func Compute(days []int, checkins map[string]string, today, created time.Time) State {
	st := State{Today: tasks.Date{Time: today}, ScheduledToday: scheduled(days, today)}
	if s, ok := checkins[today.Format(time.DateOnly)]; ok {
		st.CheckedIn = &s
	}
	kept := func(s string) bool { return s == "done" || s == "two_minute" }

	// Streak: from today if checked in, else from yesterday.
	start := today.AddDate(0, 0, -1)
	if st.CheckedIn != nil {
		start = today
	}
streak:
	for d, i := start, 0; i < lookback && !d.Before(created); d, i = d.AddDate(0, 0, -1), i+1 {
		if !scheduled(days, d) {
			continue
		}
		switch s, ok := checkins[d.Format(time.DateOnly)]; {
		case ok && kept(s):
			st.Streak++
		case ok && s == "skipped": // a planned rest day: no gain, no loss
		default:
			break streak // missed
		}
	}

	// Recovery: count missed scheduled days before today, newest first.
	if st.CheckedIn == nil {
		missed := 0
		var last time.Time
		for d, i := today.AddDate(0, 0, -1), 0; i < lookback && !d.Before(created) && missed < 7; d, i = d.AddDate(0, 0, -1), i+1 {
			if !scheduled(days, d) {
				continue
			}
			if _, ok := checkins[d.Format(time.DateOnly)]; ok {
				break
			}
			if missed == 0 {
				last = d
			}
			missed++
		}
		if missed > 0 {
			st.Recovery = &Recovery{Missed: missed, Message: recoveryMessage(missed, last, today)}
		}
	}
	return st
}

// recoveryMessage never shames: a miss is normal, and the next small step
// is what counts.
func recoveryMessage(missed int, last, today time.Time) string {
	if missed == 1 {
		when := last.Weekday().String()
		if today.Sub(last) == 24*time.Hour {
			when = "yesterday"
		}
		return fmt.Sprintf("You missed %s. That happens. Never miss twice: even the two-minute version counts today.", when)
	}
	return "It's been a few scheduled days. No catching up needed: start again today with the smallest version."
}

const columns = `h.id, e.title, h.cue, h.craving, h.reward, h.stack_after, h.two_minute, h.days, h.status,
	h.kind, h.identity, h.intention_time, h.intention_place, h.contract_partner, h.contract_pledge, e.created_at, h.updated_at`

func scan(row pgx.Row) (Habit, error) {
	var h Habit
	var days []int16
	err := row.Scan(&h.ID, &h.Title, &h.Cue, &h.Craving, &h.Reward, &h.StackAfter, &h.TwoMinute, &days, &h.Status,
		&h.Kind, &h.Identity, &h.IntentionTime, &h.IntentionPlace, &h.ContractPartner, &h.ContractPledge, &h.CreatedAt, &h.UpdatedAt)
	for _, d := range days {
		h.Days = append(h.Days, int(d))
	}
	return h, err
}

// Today is the user's current date.
func Today(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (time.Time, *time.Location, error) {
	return clock.Today(ctx, q, userID, now)
}

// withState fills State for each habit from recent check-ins (one query).
func withState(ctx context.Context, q db.Querier, list []Habit, today time.Time, loc *time.Location) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(list))
	for i, h := range list {
		ids[i] = h.ID
	}
	rows, err := q.Query(ctx, `SELECT habit_id, day, status FROM habit_checkins WHERE habit_id = ANY($1) AND day >= $2`,
		ids, today.AddDate(0, 0, -lookback-1))
	if err != nil {
		return err
	}
	byHabit := map[uuid.UUID]map[string]string{}
	for rows.Next() {
		var id uuid.UUID
		var day time.Time
		var status string
		if err := rows.Scan(&id, &day, &status); err != nil {
			return err
		}
		if byHabit[id] == nil {
			byHabit[id] = map[string]string{}
		}
		byHabit[id][day.Format(time.DateOnly)] = status
	}
	if err := rows.Err(); err != nil {
		return err
	}
	votes := map[uuid.UUID]int{}
	rows, err = q.Query(ctx, `SELECT habit_id, count(*) FROM habit_checkins WHERE habit_id = ANY($1) AND status IN ('done', 'two_minute') GROUP BY habit_id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return err
		}
		votes[id] = n
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range list {
		list[i].Votes = votes[list[i].ID]
		c := list[i].CreatedAt.In(loc)
		created := time.Date(c.Year(), c.Month(), c.Day(), 0, 0, 0, 0, time.UTC)
		list[i].State = Compute(list[i].Days, byHabit[list[i].ID], today, created)
	}
	return nil
}

// Get returns one habit with today's state.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID, now time.Time) (Habit, error) {
	h, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM habits h JOIN entities e ON e.id = h.id WHERE h.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return h, graph.ErrNotFound
	}
	if err != nil {
		return h, err
	}
	today, loc, err := Today(ctx, q, userID, now)
	if err != nil {
		return h, err
	}
	list := []Habit{h}
	err = withState(ctx, q, list, today, loc)
	return list[0], err
}

// List returns the user's habits with a status, oldest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, status string, now time.Time) ([]Habit, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM habits h JOIN entities e ON e.id = h.id
		WHERE e.user_id = $1 AND h.status = $2 ORDER BY e.created_at LIMIT 200`, userID, status)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Habit, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Habit{}
	}
	today, loc, err := Today(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	return list, withState(ctx, q, list, today, loc)
}

// Fields are the editable parts of a habit; nil means "not provided".
type Fields struct {
	Title, Cue, Craving, Reward, StackAfter, TwoMinute, Status *string
	Days                                                       []int
	Kind, Identity, IntentionTime, IntentionPlace              *string
	ContractPartner, ContractPledge                            *string
}

var hhmm = regexp.MustCompile(`^(([01][0-9]|2[0-3]):[0-5][0-9])?$`)

func (f *Fields) validate() error {
	if f.Title != nil {
		*f.Title = strings.TrimSpace(*f.Title)
		if *f.Title == "" || len([]rune(*f.Title)) > 500 {
			return ValidationError{"Describe the habit in 1 to 500 characters, like “Read one page”."}
		}
	}
	for _, s := range []*string{f.Cue, f.Craving, f.Reward, f.StackAfter, f.TwoMinute} {
		if s != nil {
			*s = strings.TrimSpace(*s)
			if len([]rune(*s)) > 500 {
				return ValidationError{"Cue, craving, reward, stack, and two-minute version are limited to 500 characters each."}
			}
		}
	}
	if f.Status != nil && !slices.Contains(statuses, *f.Status) {
		return ValidationError{"Status must be active, paused, or archived."}
	}
	for _, s := range []*string{f.Kind, f.Identity, f.IntentionTime, f.IntentionPlace, f.ContractPartner, f.ContractPledge} {
		if s != nil {
			*s = strings.TrimSpace(*s)
		}
	}
	switch {
	case f.Kind != nil && *f.Kind != "build" && *f.Kind != "break":
		return ValidationError{"Kind is build (a habit to start) or break (a habit to stop)."}
	case f.IntentionTime != nil && !hhmm.MatchString(*f.IntentionTime):
		return ValidationError{"The time is HH:MM, like 07:30, or empty."}
	case f.Identity != nil && len([]rune(*f.Identity)) > 500, f.IntentionPlace != nil && len([]rune(*f.IntentionPlace)) > 500:
		return ValidationError{"Identity and place are limited to 500 characters each."}
	case f.ContractPartner != nil && len([]rune(*f.ContractPartner)) > 200:
		return ValidationError{"The partner's name is limited to 200 characters."}
	case f.ContractPledge != nil && len([]rune(*f.ContractPledge)) > 2000:
		return ValidationError{"The pledge is limited to 2,000 characters."}
	}
	if f.Days != nil {
		seen := map[int]bool{}
		for _, d := range f.Days {
			if d < 1 || d > 7 || seen[d] {
				return ValidationError{"Days are distinct ISO weekdays from 1 (Monday) to 7 (Sunday)."}
			}
			seen[d] = true
		}
		if len(f.Days) == 0 {
			return ValidationError{"Choose at least one day."}
		}
		slices.Sort(f.Days)
	}
	return nil
}

func or(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}

// Create adds a habit (every day unless days are given).
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields, now time.Time) (Habit, error) {
	if f.Title == nil {
		return Habit{}, ValidationError{"A habit needs a title."}
	}
	if err := f.validate(); err != nil {
		return Habit{}, err
	}
	if f.Days == nil {
		f.Days = []int{1, 2, 3, 4, 5, 6, 7}
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, *f.Title, or(f.Cue, "")+" "+or(f.StackAfter, ""))
	if err != nil {
		return Habit{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO habits (id, cue, craving, reward, stack_after, two_minute, days, status,
		kind, identity, intention_time, intention_place, contract_partner, contract_pledge)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`, id, or(f.Cue, ""), or(f.Craving, ""), or(f.Reward, ""),
		or(f.StackAfter, ""), or(f.TwoMinute, ""), f.Days, or(f.Status, "active"),
		or(f.Kind, "build"), or(f.Identity, ""), or(f.IntentionTime, ""), or(f.IntentionPlace, ""), or(f.ContractPartner, ""), or(f.ContractPledge, ""))
	if err != nil {
		return Habit{}, err
	}
	return Get(ctx, tx, userID, id, now)
}

// Update applies f if the habit's version still equals version.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, version time.Time, f Fields, now time.Time) (Habit, error) {
	if err := f.validate(); err != nil {
		return Habit{}, err
	}
	cur, err := Get(ctx, tx, userID, id, now)
	if err != nil {
		return Habit{}, err
	}
	if !cur.UpdatedAt.Equal(version) {
		return Habit{}, ErrStale
	}
	next := cur
	var changed []string
	set := func(name string, p *string, dst *string) {
		if p != nil && *p != *dst {
			*dst = *p
			changed = append(changed, name)
		}
	}
	set("title", f.Title, &next.Title)
	set("cue", f.Cue, &next.Cue)
	set("craving", f.Craving, &next.Craving)
	set("reward", f.Reward, &next.Reward)
	set("stack_after", f.StackAfter, &next.StackAfter)
	set("two_minute", f.TwoMinute, &next.TwoMinute)
	set("status", f.Status, &next.Status)
	set("kind", f.Kind, &next.Kind)
	set("identity", f.Identity, &next.Identity)
	set("intention_time", f.IntentionTime, &next.IntentionTime)
	set("intention_place", f.IntentionPlace, &next.IntentionPlace)
	set("contract_partner", f.ContractPartner, &next.ContractPartner)
	set("contract_pledge", f.ContractPledge, &next.ContractPledge)
	if f.Days != nil && !slices.Equal(f.Days, cur.Days) {
		next.Days = f.Days
		changed = append(changed, "days")
	}
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Cue+" "+next.StackAfter, changed); err != nil {
		return Habit{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE habits SET cue = $2, craving = $3, reward = $4, stack_after = $5, two_minute = $6,
		days = $7, status = $8, kind = $9, identity = $10, intention_time = $11, intention_place = $12,
		contract_partner = $13, contract_pledge = $14, updated_at = clock_timestamp() WHERE id = $1`,
		id, next.Cue, next.Craving, next.Reward, next.StackAfter, next.TwoMinute, next.Days, next.Status,
		next.Kind, next.Identity, next.IntentionTime, next.IntentionPlace, next.ContractPartner, next.ContractPledge)
	if err != nil {
		return Habit{}, err
	}
	return Get(ctx, tx, userID, id, now)
}

// Delete removes a habit, its check-ins, and its links.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, now time.Time) error {
	if _, err := Get(ctx, tx, userID, id, now); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// Checkin is one day's record.
type Checkin struct {
	Day       tasks.Date `json:"day"`
	Status    string     `json:"status"`
	Note      string     `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
}

// CheckIn records a day: today or up to 7 days back, never the future.
func CheckIn(ctx context.Context, q db.Querier, userID, id uuid.UUID, day time.Time, status, note string, now time.Time) (Checkin, error) {
	if !slices.Contains([]string{"done", "two_minute", "skipped"}, status) {
		return Checkin{}, ValidationError{"Status is done, two_minute, or skipped."}
	}
	if len(note) > 1000 {
		return Checkin{}, ValidationError{"Notes are limited to 1,000 characters."}
	}
	if _, err := Get(ctx, q, userID, id, now); err != nil {
		return Checkin{}, err
	}
	today, _, err := Today(ctx, q, userID, now)
	if err != nil {
		return Checkin{}, err
	}
	if day.After(today) || day.Before(today.AddDate(0, 0, -7)) {
		return Checkin{}, ValidationError{"Check in for today or one of the last 7 days."}
	}
	c := Checkin{Day: tasks.Date{Time: day}, Status: status, Note: note}
	err = q.QueryRow(ctx, `INSERT INTO habit_checkins (habit_id, day, status, note) VALUES ($1, $2, $3, $4)
		ON CONFLICT (habit_id, day) DO UPDATE SET status = $3, note = $4 RETURNING created_at`, id, day, status, note).Scan(&c.CreatedAt)
	if err == nil && OnCheckin != nil {
		OnCheckin(ctx, q, userID, id)
	}
	return c, err
}

// Uncheck removes a day's check-in.
func Uncheck(ctx context.Context, q db.Querier, userID, id uuid.UUID, day time.Time, now time.Time) error {
	if _, err := Get(ctx, q, userID, id, now); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `DELETE FROM habit_checkins WHERE habit_id = $1 AND day = $2`, id, day)
	if err == nil && OnCheckin != nil {
		OnCheckin(ctx, q, userID, id)
	}
	return err
}

// OnCheckin, when set, hears about every check-in change (the vault mirror
// refreshes the habit's note). Its errors never fail the check-in.
var OnCheckin func(ctx context.Context, q db.Querier, userID, habitID uuid.UUID)

// History returns check-ins between from and to, newest first.
func History(ctx context.Context, q db.Querier, userID, id uuid.UUID, from, to time.Time, now time.Time) ([]Checkin, error) {
	if _, err := Get(ctx, q, userID, id, now); err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT day, status, note, created_at FROM habit_checkins WHERE habit_id = $1 AND day BETWEEN $2 AND $3 ORDER BY day DESC`, id, from, to)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Checkin, error) {
		var c Checkin
		err := r.Scan(&c.Day.Time, &c.Status, &c.Note, &c.CreatedAt)
		return c, err
	})
	if out == nil {
		out = []Checkin{}
	}
	return out, err
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM habits h JOIN entities e ON e.id = h.id WHERE e.user_id = $1 ORDER BY h.id`, userID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Habit, error) { return scan(r) })
	if err != nil {
		return nil, err
	}
	type row struct {
		Habit
		State    *struct{} `json:"state,omitempty"`
		Checkins []Checkin `json:"checkins"`
	}
	out := []row{}
	for _, h := range list {
		rows, err := q.Query(ctx, `SELECT day, status, note, created_at FROM habit_checkins WHERE habit_id = $1 ORDER BY day`, h.ID)
		if err != nil {
			return nil, err
		}
		cs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Checkin, error) {
			var c Checkin
			err := r.Scan(&c.Day.Time, &c.Status, &c.Note, &c.CreatedAt)
			return c, err
		})
		if err != nil {
			return nil, err
		}
		out = append(out, row{Habit: h, Checkins: cs})
	}
	return out, nil
}
