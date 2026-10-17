// Package digest builds what a person reads each morning, each week, and
// each month (EP-107): what to do today, what comes next, what to learn and
// recall, how the period went, and one line to carry, chosen for the day
// and for what the person works on. A digest is data; renderers turn it
// into an email, a chat message, or a notification. Items a person can act
// on are numbered, so "done 2" in a reply means the same everywhere (EP-108).
package digest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/household"
	"github.com/thoutmose/axon/api/internal/learning"
)

// Kinds of digest.
const (
	Daily   = "daily"
	Weekly  = "weekly"
	Monthly = "monthly"
	News    = "news" // the day's general news, apart from the tech watch (EP-155)
)

// Item is one line. N numbers the lines a reply can act on.
type Item struct {
	N      int       `json:"n,omitempty"`
	ID     uuid.UUID `json:"id"`
	Type   string    `json:"type"` // task, habit, milestone, goal, card, note, review, household
	Text   string    `json:"text"`
	Detail string    `json:"detail,omitempty"`
	URL    string    `json:"url"`
	Late   bool      `json:"late,omitempty"`
}

// Section groups items under a heading.
type Section struct {
	Key   string `json:"key"`
	Emoji string `json:"emoji"`
	Title string `json:"title"`
	Items []Item `json:"items"`
	More  int    `json:"more,omitempty"` // items left out to keep it short
}

// Stat is one number about the period that ended.
type Stat struct {
	Emoji string `json:"emoji"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// Digest is everything a channel needs to say it.
type Digest struct {
	Kind     string    `json:"kind"`
	Lang     string    `json:"lang"`
	Title    string    `json:"title"`
	Greeting string    `json:"greeting"`
	Date     string    `json:"date"`  // the local date it is for, YYYY-MM-DD
	Label    string    `json:"label"` // "Wednesday 14 October", "Week of 12 October"
	Summary  string    `json:"summary"`
	Sections []Section `json:"sections"`
	Stats    []Stat    `json:"stats"`
	Quote    Quote     `json:"quote"`
	Theme    string    `json:"theme"`   // the quote's theme, which also tints the picture
	Picture  bool      `json:"picture"` // whether the person wants the picture
	URL      string    `json:"url"`
	Seed     string    `json:"-"` // makes the picture the same on every channel
}

// Empty reports a digest with nothing to do and nothing to report.
func (d Digest) Empty() bool { return len(d.Sections) == 0 && len(d.Stats) == 0 }

// Actionable lists the numbered items, for replies.
func (d Digest) Actionable() []Item {
	var out []Item
	for _, s := range d.Sections {
		for _, it := range s.Items {
			if it.N > 0 {
				out = append(out, it)
			}
		}
	}
	return out
}

// Limits keep a digest readable on a phone.
const (
	maxToday    = 10
	maxUpcoming = 8
)

// Build makes a digest of kind for the person at now.
func Build(ctx context.Context, q db.Querier, userID uuid.UUID, kind, lang string, now time.Time) (Digest, error) {
	today, loc, err := habits.Today(ctx, q, userID, now)
	if err != nil {
		return Digest{}, err
	}
	if lang != "fr" {
		lang = "en"
	}
	tr := words[lang]
	b := &builder{ctx: ctx, q: q, user: userID, lang: lang, tr: tr, today: today, loc: loc, now: now}
	var name string
	if err := q.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, userID).Scan(&name); err != nil {
		return Digest{}, err
	}
	d := Digest{Kind: kind, Lang: lang, Date: today.Format(time.DateOnly), Seed: userID.String() + kind + today.Format(time.DateOnly)}
	first := strings.Fields(name)
	if len(first) > 0 {
		name = first[0]
	}
	hour := now.In(loc).Hour()
	greet := tr["Good morning"]
	switch {
	case hour >= 18:
		greet = tr["Good evening"]
	case hour >= 12:
		greet = tr["Good afternoon"]
	}
	d.Greeting = fmt.Sprintf("%s, %s", greet, name)
	switch kind {
	case News: // its own digest, without the template's sections (EP-155)
		d.Picture = true
		return d, b.headlines(&d)
	case Weekly:
		err = b.weekly(&d)
	case Monthly:
		err = b.monthly(&d)
	default:
		d.Kind = Daily
		err = b.daily(&d)
	}
	if err != nil {
		return Digest{}, err
	}
	tpl, err := b.template()
	if err != nil {
		return Digest{}, err
	}
	order := map[string][]string{Daily: tpl.Daily, Weekly: tpl.Weekly, Monthly: tpl.Monthly}[d.Kind]
	if slices.Contains(order, "watch") {
		days := map[string]int{Daily: 1, Weekly: 7, Monthly: 30}[d.Kind]
		news, err := b.news(now.AddDate(0, 0, -days))
		if err != nil {
			return Digest{}, err
		}
		if len(news) > 0 {
			d.Sections = append(d.Sections, b.section("Tech watch", "📰", news, 0))
		}
	}
	arrange(&d, order)
	d.Picture = tpl.Picture
	if tpl.Quote {
		domains, err := b.domains()
		if err != nil {
			return Digest{}, err
		}
		d.Quote = pick(d.Theme, domains, d.Seed, lang)
	}
	return d, nil
}

// Template is the person's digest template (EP-113), stored with their
// settings; DefaultTemplate mirrors the settings' defaults.
type Template struct {
	Daily   []string `json:"daily"`
	Weekly  []string `json:"weekly"`
	Monthly []string `json:"monthly"`
	Quote   bool     `json:"quote"`
	Picture bool     `json:"picture"`
}

// DefaultTemplate is every section in its usual place, with the quote
// and the picture.
func DefaultTemplate() Template {
	return Template{Daily: []string{"tasks", "habits", "coming", "learn", "watch"}, Weekly: []string{"open", "week", "reflect", "watch"},
		Monthly: []string{"goals", "month", "watch"}, Quote: true, Picture: true}
}

// sectionKeys name the sections in the template.
var sectionKeys = map[string]string{
	"To do today": "tasks", "Habits": "habits", "Coming up": "coming", "Learn and remember": "learn",
	"Still open": "open", "This week": "week", "Reflect": "reflect", "Goals": "goals", "This month": "month", "Tech watch": "watch",
}

// template reads the person's template; the settings package is not
// imported, as it depends on much of Axon.
func (b *builder) template() (Template, error) {
	t := DefaultTemplate()
	var raw []byte
	err := b.q.QueryRow(b.ctx, `SELECT doc->'digest' FROM user_settings WHERE user_id = $1`, b.user).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(raw) == 0) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	return t, json.Unmarshal(raw, &t)
}

// arrange puts the sections in the template's order, leaving out the
// others, and numbers the items again in reading order.
func arrange(d *Digest, order []string) {
	var out []Section
	for _, key := range order {
		for _, s := range d.Sections {
			if sectionKeys[s.Key] == key {
				out = append(out, s)
			}
		}
	}
	d.Sections = out
	renumber(d)
}

// news are the period's top stories of the tech watch: unnumbered, as an
// answer cannot act on them.
func (b *builder) news(since time.Time) ([]Item, error) {
	rows, err := b.q.Query(b.ctx, `SELECT id, title, source_title, cardinality(also_in) FROM watch_items
		WHERE user_id = $1 AND published_at >= $2 AND outlier = '' AND status <> 'archived'
		  AND NOT EXISTS (SELECT 1 FROM watch_sources s WHERE s.id = watch_items.source_id AND s.news)
		ORDER BY score DESC, cardinality(also_in) DESC, published_at DESC LIMIT 5`, b.user, since)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Item, error) {
		var it Item
		var also int
		err := r.Scan(&it.ID, &it.Text, &it.Detail, &also)
		if also > 0 {
			it.Detail += fmt.Sprintf(b.tr[" and %d more"], also)
		}
		it.Type, it.URL = "news", "/watch?item="+it.ID.String()
		return it, err
	})
}

type builder struct {
	ctx   context.Context
	q     db.Querier
	user  uuid.UUID
	lang  string
	tr    map[string]string
	today time.Time
	loc   *time.Location
	now   time.Time
	n     int // last number given
}

func (b *builder) next() int { b.n++; return b.n }

// days says how far a date is from today, in words.
func (b *builder) days(d time.Time) (string, bool) {
	n := int(d.Sub(b.today).Hours() / 24)
	switch {
	case n < -1:
		return fmt.Sprintf(b.tr["%d days late"], -n), true
	case n == -1:
		return b.tr["since yesterday"], true
	case n == 0:
		return b.tr["today"], false
	case n == 1:
		return b.tr["tomorrow"], false
	case n < 7:
		return b.weekday(d), false
	}
	return b.dayMonth(d), false
}

func (b *builder) weekday(d time.Time) string { return b.tr[d.Weekday().String()] }

func (b *builder) dayMonth(d time.Time) string {
	return fmt.Sprintf(b.tr["%d %s"], d.Day(), b.tr[d.Month().String()])
}

func (b *builder) longDate(d time.Time) string {
	return b.weekday(d) + " " + b.dayMonth(d)
}

type openTask struct {
	id        uuid.UUID
	title     string
	due, sch  *time.Time
	doing     bool
	recurring bool
}

// tasks lists open tasks whose day (scheduled or due) is on or before until.
func (b *builder) tasks(from, until time.Time) ([]openTask, error) {
	rows, err := b.q.Query(b.ctx, `SELECT t.id, e.title, t.due_on, t.scheduled_on, t.status = 'doing', t.series_id IS NOT NULL
		FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled', 'someday')
		  AND (least(t.scheduled_on, t.due_on) BETWEEN $2 AND $3 OR (t.due_on IS NOT NULL AND t.due_on BETWEEN $2 AND $3))
		ORDER BY coalesce(t.due_on, t.scheduled_on), t.due_on IS NULL, t.status = 'doing' DESC, e.title
		LIMIT 200`, b.user, from, until)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (openTask, error) {
		var t openTask
		err := r.Scan(&t.id, &t.title, &t.due, &t.sch, &t.doing, &t.recurring)
		return t, err
	})
}

func (b *builder) taskItem(t openTask) Item {
	it := Item{N: b.next(), ID: t.id, Type: "task", Text: t.title, URL: "/tasks?open=" + t.id.String()}
	switch {
	case t.due != nil:
		when, late := b.days(*t.due)
		it.Detail, it.Late = b.tr["due"]+" "+when, late
	case t.sch != nil && t.sch.Before(b.today):
		it.Detail = b.tr["carried over"]
	}
	if t.doing {
		it.Detail = strings.TrimSpace(b.tr["in progress"] + " · " + it.Detail)
		it.Detail = strings.TrimSuffix(it.Detail, " ·")
	}
	return it
}

func (b *builder) section(key, emoji string, items []Item, limit int) Section {
	s := Section{Key: key, Emoji: emoji, Title: b.tr[key], Items: items}
	if limit > 0 && len(items) > limit {
		s.Items, s.More = items[:limit], len(items)-limit
	}
	return s
}

// renumber gives numbers in reading order once sections are cut short.
func renumber(d *Digest) {
	n := 0
	for i := range d.Sections {
		for j := range d.Sections[i].Items {
			if d.Sections[i].Items[j].N > 0 {
				n++
				d.Sections[i].Items[j].N = n
			}
		}
	}
}

func (b *builder) daily(d *Digest) error {
	tr := b.tr
	d.Title = tr["Your day"]
	d.Label = b.longDate(b.today)
	d.URL = "/today"
	list, err := b.tasks(time.Time{}, b.today)
	if err != nil {
		return err
	}
	var today []Item
	overdue := 0
	for _, t := range list {
		it := b.taskItem(t)
		if it.Late {
			overdue++
		}
		today = append(today, it)
	}
	if len(today) > 0 {
		d.Sections = append(d.Sections, b.section("To do today", "✅", today, maxToday))
	}

	hs, err := habits.List(b.ctx, b.q, b.user, "active", b.now)
	if err != nil {
		return err
	}
	var hab []Item
	for _, h := range hs {
		if h.State.ScheduledToday && h.State.CheckedIn == nil {
			it := Item{N: b.next(), ID: h.ID, Type: "habit", Text: h.Title, URL: "/habits"}
			switch {
			case h.Kind == "break": // a habit to break: its check-in means it was avoided
				it.Detail = tr["to avoid today"]
			case h.TwoMinute != "":
				it.Detail = tr["or just"] + " " + h.TwoMinute
			}
			hab = append(hab, it)
		}
	}
	if len(hab) > 0 {
		d.Sections = append(d.Sections, b.section("Habits", "🌱", hab, 0))
	}

	upcoming, err := b.upcoming(b.today.AddDate(0, 0, 1), b.today.AddDate(0, 0, 7), b.today.AddDate(0, 0, 14), b.today.AddDate(0, 0, 30))
	if err != nil {
		return err
	}
	if len(upcoming) > 0 {
		d.Sections = append(d.Sections, b.section("Coming up", "🗓️", upcoming, maxUpcoming))
	}

	learn, err := b.learning()
	if err != nil {
		return err
	}
	if n, err := household.MineToday(b.ctx, b.q, b.user, b.today); err != nil {
		return err
	} else if n > 0 {
		learn = append(learn, Item{Type: "household", Text: fmt.Sprintf(tr["%d household items for you"], n), URL: "/household"})
	}
	if wd := b.today.Weekday(); wd == time.Friday || wd == time.Saturday || wd == time.Sunday {
		done, err := b.reviewDone()
		if err != nil {
			return err
		}
		if !done {
			learn = append(learn, Item{Type: "review", Text: tr["The weekly review, about 20 min"], URL: "/review"})
		}
	}
	if len(learn) > 0 {
		d.Sections = append(d.Sections, b.section("Learn and remember", "🧠", learn, 0))
	}
	renumber(d)

	// The day's theme decides the quote and tints the picture.
	learning := 0
	for _, it := range learn {
		if it.Type == "card" || it.Type == "note" {
			learning++
		}
	}
	switch wd := b.today.Weekday(); {
	case overdue > 0:
		d.Theme = "persistence"
	case wd == time.Monday:
		d.Theme = "start"
	case wd == time.Saturday || wd == time.Sunday:
		d.Theme = "rest"
	case len(hab) > len(today):
		d.Theme = "habits"
	case learning > 0 && len(today) < 3:
		d.Theme = "learning"
	case len(today) > 6:
		d.Theme = "focus"
	case len(today) == 0:
		d.Theme = "simplicity"
	default:
		d.Theme = "focus"
	}
	parts := []string{}
	if n := len(today); n > 0 {
		s := plural(tr, n, "task", "tasks")
		if overdue > 0 {
			s += fmt.Sprintf(tr[" (%d late)"], overdue)
		}
		parts = append(parts, s)
	}
	if n := len(hab); n > 0 {
		parts = append(parts, plural(tr, n, "habit", "habits"))
	}
	for _, it := range learn {
		if it.Type == "card" {
			parts = append(parts, it.Text)
		}
	}
	if len(parts) == 0 {
		d.Summary = tr["Nothing is due today."]
	} else {
		d.Summary = strings.Join(parts, " · ")
	}
	return nil
}

// upcoming lists open tasks due from..taskUntil, milestones due until
// milestoneUntil, and goals whose target date is until goalUntil.
func (b *builder) upcoming(from, taskUntil, milestoneUntil, goalUntil time.Time) ([]Item, error) {
	var out []Item
	rows, err := b.q.Query(b.ctx, `SELECT t.id, e.title, t.due_on, t.scheduled_on, t.status = 'doing', t.series_id IS NOT NULL
		FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled', 'someday') AND t.due_on BETWEEN $2 AND $3
		  AND (t.scheduled_on IS NULL OR t.scheduled_on >= $2)
		ORDER BY t.due_on, e.title LIMIT 50`, b.user, from, taskUntil)
	if err != nil {
		return nil, err
	}
	ts, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (openTask, error) {
		var t openTask
		err := r.Scan(&t.id, &t.title, &t.due, &t.sch, &t.doing, &t.recurring)
		return t, err
	})
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		out = append(out, b.taskItem(t))
	}
	rows, err = b.q.Query(b.ctx, `SELECT m.id, e.title, ge.title, m.due_on FROM milestones m JOIN entities e ON e.id = m.id
		JOIN entities ge ON ge.id = m.goal_id JOIN goals g ON g.id = m.goal_id
		WHERE e.user_id = $1 AND m.done_at IS NULL AND g.status = 'active' AND m.due_on BETWEEN $2 AND $3
		ORDER BY m.due_on, e.title LIMIT 20`, b.user, b.today, milestoneUntil)
	if err != nil {
		return nil, err
	}
	type ms struct {
		id          uuid.UUID
		title, goal string
		due         time.Time
	}
	mss, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (ms, error) {
		var m ms
		err := r.Scan(&m.id, &m.title, &m.goal, &m.due)
		return m, err
	})
	if err != nil {
		return nil, err
	}
	for _, m := range mss {
		when, late := b.days(m.due)
		out = append(out, Item{N: b.next(), ID: m.id, Type: "milestone", Text: m.title, Detail: "🏁 " + m.goal + " · " + when, URL: "/goals", Late: late})
	}
	rows, err = b.q.Query(b.ctx, `SELECT g.id, e.title, g.target_date FROM goals g JOIN entities e ON e.id = g.id
		WHERE e.user_id = $1 AND g.status = 'active' AND g.target_date BETWEEN $2 AND $3 ORDER BY g.target_date LIMIT 5`, b.user, b.today, goalUntil)
	if err != nil {
		return nil, err
	}
	type goal struct {
		id     uuid.UUID
		title  string
		target time.Time
	}
	gs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (goal, error) {
		var g goal
		err := r.Scan(&g.id, &g.title, &g.target)
		return g, err
	})
	if err != nil {
		return nil, err
	}
	for _, g := range gs {
		when, _ := b.days(g.target)
		out = append(out, Item{ID: g.id, Type: "goal", Text: g.title, Detail: "🎯 " + b.tr["target"] + " " + when, URL: "/goals"})
	}
	return out, nil
}

// learning lists cards to review, notes to recall, and the weekly test.
func (b *builder) learning() ([]Item, error) {
	plan, err := learning.Today(b.ctx, b.q, b.user, b.now)
	if err != nil {
		return nil, err
	}
	var out []Item
	if plan.Cards.Due > 0 {
		out = append(out, Item{Type: "card", Text: plural(b.tr, plan.Cards.Due, "card to review", "cards to review"),
			Detail: fmt.Sprintf(b.tr["about %d min"], plan.Cards.Minutes), URL: "/cards/review"})
	}
	for i, n := range plan.Recall.Due {
		if i == 3 {
			out = append(out, Item{Type: "note", Text: fmt.Sprintf(b.tr["and %d more notes to recall"], len(plan.Recall.Due)-3), URL: "/learn"})
			break
		}
		out = append(out, Item{ID: n.ID, Type: "note", Text: n.Title, Detail: b.tr["to recall"], URL: "/learn"})
	}
	if plan.Test.Due {
		out = append(out, Item{Type: "test", Text: b.tr["The weekly test"], URL: "/learn"})
	}
	return out, nil
}

func (b *builder) reviewDone() (bool, error) {
	monday := b.today.AddDate(0, 0, -((int(b.today.Weekday()) + 6) % 7))
	var done bool
	err := b.q.QueryRow(b.ctx, `SELECT EXISTS (SELECT 1 FROM checkins WHERE user_id = $1 AND kind = 'weekly' AND day = $2)`, b.user, monday).Scan(&done)
	return done, err
}

// stats counts what happened from from (inclusive) to until (exclusive), local days.
func (b *builder) stats(from, until time.Time) ([]Stat, int, error) {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, b.loc)
	end := time.Date(until.Year(), until.Month(), until.Day(), 0, 0, 0, 0, b.loc)
	var tasksDone, checkins, reviews, focus, notes, milestones int
	if err := b.q.QueryRow(b.ctx, `SELECT
		(SELECT count(*) FROM tasks t JOIN entities e ON e.id = t.id WHERE e.user_id = $1 AND t.status = 'done' AND t.completed_at >= $2 AND t.completed_at < $3),
		(SELECT count(*) FROM habit_checkins c JOIN entities e ON e.id = c.habit_id WHERE e.user_id = $1 AND c.status IN ('done', 'two_minute') AND c.day >= $4 AND c.day < $5),
		(SELECT count(*) FROM card_reviews r JOIN entities e ON e.id = r.card_id WHERE e.user_id = $1 AND r.reviewed_at >= $2 AND r.reviewed_at < $3),
		(SELECT coalesce(sum(f.worked_minutes), 0) FROM focus_sessions f JOIN entities e ON e.id = f.id WHERE e.user_id = $1 AND f.started_at >= $2 AND f.started_at < $3),
		(SELECT count(*) FROM notes n JOIN entities e ON e.id = n.id WHERE n.user_id = $1 AND e.created_at >= $2 AND e.created_at < $3 AND n.path NOT LIKE 'Axon/%'),
		(SELECT count(*) FROM milestones m JOIN entities e ON e.id = m.id WHERE e.user_id = $1 AND m.done_at >= $2 AND m.done_at < $3)`,
		b.user, start, end, from, until).Scan(&tasksDone, &checkins, &reviews, &focus, &notes, &milestones); err != nil {
		return nil, 0, err
	}
	tr := b.tr
	var out []Stat
	add := func(emoji, label string, v int, value string) {
		if v > 0 {
			out = append(out, Stat{Emoji: emoji, Label: tr[label], Value: value})
		}
	}
	add("✅", "Tasks done", tasksDone, fmt.Sprint(tasksDone))
	add("🏁", "Milestones reached", milestones, fmt.Sprint(milestones))
	add("🌱", "Habit check-ins", checkins, fmt.Sprint(checkins))
	add("🃏", "Cards reviewed", reviews, fmt.Sprint(reviews))
	add("⏱️", "Focused", focus, fmt.Sprintf("%dh%02d", focus/60, focus%60))
	add("📝", "Notes written", notes, fmt.Sprint(notes))
	return out, tasksDone + checkins + reviews + focus + notes + milestones, nil
}

func (b *builder) weekly(d *Digest) error {
	tr := b.tr
	d.Title = tr["Your week"]
	monday := b.today.AddDate(0, 0, -((int(b.today.Weekday()) + 6) % 7))
	d.Label = fmt.Sprintf(tr["Week of %s"], b.dayMonth(monday))
	d.URL = "/review"
	stats, activity, err := b.stats(b.today.AddDate(0, 0, -7), b.today)
	if err != nil {
		return err
	}
	d.Stats = stats
	late, err := b.tasks(time.Time{}, b.today.AddDate(0, 0, -1))
	if err != nil {
		return err
	}
	var carry []Item
	for _, t := range late {
		carry = append(carry, b.taskItem(t))
	}
	if len(carry) > 0 {
		d.Sections = append(d.Sections, b.section("Still open", "📌", carry, maxToday))
	}
	week, err := b.upcoming(b.today, b.today.AddDate(0, 0, 6), b.today.AddDate(0, 0, 13), b.today.AddDate(0, 0, 60))
	if err != nil {
		return err
	}
	if len(week) > 0 {
		d.Sections = append(d.Sections, b.section("This week", "🗓️", week, maxUpcoming+4))
	}
	if done, err := b.reviewDone(); err != nil {
		return err
	} else if !done {
		d.Sections = append(d.Sections, b.section("Reflect", "🪞", []Item{{Type: "review", Text: tr["The weekly review, about 20 min"], URL: "/review"}}, 0))
	}
	renumber(d)
	d.Theme = "reflection"
	if activity > 0 {
		d.Summary = fmt.Sprintf(tr["Last week: %s."], statLine(stats))
	} else {
		d.Theme = "start"
		d.Summary = tr["A new week, a fresh start."]
	}
	return nil
}

func (b *builder) monthly(d *Digest) error {
	tr := b.tr
	d.Title = tr["Your month"]
	first := time.Date(b.today.Year(), b.today.Month(), 1, 0, 0, 0, 0, time.UTC)
	prev := first.AddDate(0, -1, 0)
	d.Label = tr[b.today.Month().String()] + " " + fmt.Sprint(b.today.Year())
	d.URL = "/insights"
	stats, activity, err := b.stats(prev, first)
	if err != nil {
		return err
	}
	d.Stats = stats
	goals, err := b.goals()
	if err != nil {
		return err
	}
	if len(goals) > 0 {
		d.Sections = append(d.Sections, b.section("Goals", "🎯", goals, 0))
	}
	month, err := b.upcoming(b.today, first.AddDate(0, 1, -1), first.AddDate(0, 1, -1), first.AddDate(0, 3, 0))
	if err != nil {
		return err
	}
	// The month ahead: milestones and goals, and the tasks with a deadline.
	if len(month) > 0 {
		d.Sections = append(d.Sections, b.section("This month", "🗓️", month, maxUpcoming+4))
	}
	renumber(d)
	d.Theme = "progress"
	if activity > 0 {
		d.Summary = fmt.Sprintf(tr["In %s: %s."], tr[prev.Month().String()], statLine(stats))
	} else {
		d.Theme = "planning"
		d.Summary = tr["A new month: a good time to choose what matters."]
	}
	return nil
}

// goals lists active goals with how far along they are.
func (b *builder) goals() ([]Item, error) {
	rows, err := b.q.Query(b.ctx, `SELECT g.id, e.title, g.measure_start, g.measure_target, g.measure_current, g.measure_unit,
		(SELECT count(*) FROM milestones m WHERE m.goal_id = g.id), (SELECT count(*) FROM milestones m WHERE m.goal_id = g.id AND m.done_at IS NOT NULL)
		FROM goals g JOIN entities e ON e.id = g.id WHERE e.user_id = $1 AND g.status = 'active' ORDER BY g.target_date NULLS LAST, e.title LIMIT 6`, b.user)
	if err != nil {
		return nil, err
	}
	type goal struct {
		id                    uuid.UUID
		title                 string
		start, target, now    *float64
		unit                  *string
		milestones, reachedMs int
	}
	gs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (goal, error) {
		var g goal
		err := r.Scan(&g.id, &g.title, &g.start, &g.target, &g.now, &g.unit, &g.milestones, &g.reachedMs)
		return g, err
	})
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, g := range gs {
		it := Item{ID: g.id, Type: "goal", Text: g.title, URL: "/goals"}
		switch {
		case g.target != nil && *g.target != *g.start:
			pct := max(0, min(100, int((*g.now-*g.start)/(*g.target-*g.start)*100+0.5)))
			it.Detail = fmt.Sprintf("%s %d%% · %g/%g %s", bar(pct), pct, *g.now, *g.target, *g.unit)
		case g.milestones > 0:
			pct := g.reachedMs * 100 / g.milestones
			it.Detail = fmt.Sprintf(b.tr["%s %d of %d milestones"], bar(pct), g.reachedMs, g.milestones)
		}
		out = append(out, it)
	}
	return out, nil
}

// bar draws progress in ten blocks, readable in any channel.
func bar(pct int) string {
	n := (pct + 5) / 10
	return strings.Repeat("▰", n) + strings.Repeat("▱", 10-n)
}

func statLine(stats []Stat) string {
	parts := make([]string, 0, len(stats))
	for _, s := range stats {
		parts = append(parts, s.Value+" "+strings.ToLower(s.Label))
	}
	return strings.Join(parts, ", ")
}

func plural(tr map[string]string, n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, tr[one])
	}
	return fmt.Sprintf("%d %s", n, tr[many])
}

// domains reads what the person works on from their active goals and areas.
func (b *builder) domains() ([]string, error) {
	rows, err := b.q.Query(b.ctx, `SELECT e.title FROM entities e
		LEFT JOIN goals g ON g.id = e.id LEFT JOIN projects p ON p.id = e.id
		WHERE e.user_id = $1 AND e.type IN ('goal', 'area', 'project') AND coalesce(g.status, p.status, 'active') = 'active'
		ORDER BY e.id DESC LIMIT 50`, b.user)
	if err != nil {
		return nil, err
	}
	titles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	return Domains(titles), nil
}

// Kinds lists the digests in the order they are offered.
var Kinds = []string{Daily, Weekly, Monthly, News}

// Valid reports a known kind.
func Valid(kind string) bool { return slices.Contains(Kinds, kind) }

// Remember stores the numbers of the items a person just received, so an
// answer such as "done 2" finds them (EP-108). The latest list wins.
func Remember(ctx context.Context, q db.Querier, userID uuid.UUID, items []Item) error {
	if _, err := q.Exec(ctx, `DELETE FROM digest_items WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, it := range items {
		if it.N < 1 || it.N > 100 || (it.Type != "task" && it.Type != "habit" && it.Type != "milestone") {
			continue
		}
		title := it.Text
		if len(title) > 500 {
			title = title[:500]
		}
		if _, err := q.Exec(ctx, `INSERT INTO digest_items (user_id, n, entity_id, type, title) VALUES ($1, $2, $3, $4, $5)`,
			userID, it.N, it.ID, it.Type, title); err != nil {
			return err
		}
	}
	return nil
}
