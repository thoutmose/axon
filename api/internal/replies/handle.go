package replies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/digest"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/knowledge"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/notify"
	"github.com/thoutmose/axon/api/internal/settings"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Inbound is one message a bot received.
type Inbound struct {
	Channels []uuid.UUID // the channels using this bot
	Kind     string      // slack, discord, telegram
	From     string      // the sender's id in that app
	FromName string
	Text     string
}

// Handler acts on messages.
type Handler struct {
	DB   *pgxpool.Pool
	Base string // AXON_BASE_URL, for links in lists
	Now  func() time.Time
}

// action is what a command changed, kept for "undo".
type action struct {
	Type   string     `json:"type"` // task_done, habit_done, milestone_done, task_added, note_added, task_moved
	ID     uuid.UUID  `json:"id"`
	Status string     `json:"status,omitempty"` // the task's status before
	Day    string     `json:"day,omitempty"`    // the habit's day, or the task's scheduled day before ("" = none)
	Title  string     `json:"title"`
	At     *time.Time `json:"at,omitempty"`
}

type person struct {
	channel uuid.UUID
	user    uuid.UUID
	lang    string
	tr      map[string]string
	today   time.Time
	now     time.Time
}

func (p person) t(s string, args ...any) string {
	if v, ok := p.tr[s]; ok {
		s = v
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Handle answers one message. Only the chat linked to a channel is
// obeyed; anyone else is told how to link their own.
func (h Handler) Handle(ctx context.Context, in Inbound) (string, error) {
	cmd := Parse(in.Text)
	now := h.Now()
	if cmd.Verb == Link {
		return h.link(ctx, in, cmd.Text, now)
	}
	var p person
	err := h.DB.QueryRow(ctx, `SELECT id, user_id FROM notify_channels WHERE id = ANY ($1) AND peer = $2 AND peer <> '' AND enabled LIMIT 1`,
		in.Channels, in.From).Scan(&p.channel, &p.user)
	if errors.Is(err, pgx.ErrNoRows) {
		return en["This bot answers one person. To use it with your Axon, open Preferences → Chat apps and send the code shown there: link ABCD-EFGH"], nil
	}
	if err != nil {
		return "", err
	}
	doc, err := settings.Get(ctx, h.DB, p.user)
	if err != nil {
		return "", err
	}
	p.lang, p.tr, p.now = doc.Language, words[doc.Language], now
	if p.tr == nil {
		p.tr, p.lang = en, "en"
	}
	if p.today, _, err = habits.Today(ctx, h.DB, p.user, now); err != nil {
		return "", err
	}
	switch cmd.Verb {
	case Done:
		return h.done(ctx, p, cmd)
	case Undo:
		return h.undo(ctx, p)
	case Add:
		return h.add(ctx, p, cmd, in.Kind)
	case Note:
		return h.note(ctx, p, cmd.Text, in.Kind)
	case Snooze:
		return h.snooze(ctx, p, cmd)
	case List:
		return h.list(ctx, p)
	case Help:
		return p.t("help"), nil
	}
	if len(strings.Fields(cmd.Text)) >= 3 {
		return h.note(ctx, p, cmd.Text, in.Kind)
	}
	return p.t("Not sure what you mean. Say help to see what I understand."), nil
}

func (h Handler) link(ctx context.Context, in Inbound, code string, now time.Time) (string, error) {
	var user uuid.UUID
	err := h.DB.QueryRow(ctx, `UPDATE notify_channels SET peer = $3, peer_name = $4, link_hash = NULL, link_expires_at = NULL
		WHERE id = ANY ($1) AND link_hash = $2 AND link_expires_at > $5 RETURNING user_id`,
		in.Channels, notify.HashCode(code), in.From, truncate(in.FromName, 100), now).Scan(&user)
	if errors.Is(err, pgx.ErrNoRows) {
		return en["That code is wrong or has expired. Make a new one in Axon: Preferences → Chat apps."], nil
	}
	if err != nil {
		return "", err
	}
	doc, err := settings.Get(ctx, h.DB, user)
	if err != nil {
		return "", err
	}
	tr := words[doc.Language]
	if tr == nil {
		tr = en
	}
	return tr["Linked. Your digests arrive here, and I do what you answer."] + "\n\n" + tr["help"], nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// remember stores what a command did, for undo.
func (h Handler) remember(ctx context.Context, q pgx.Tx, p person, acts []action) error {
	b, _ := json.Marshal(acts)
	_, err := q.Exec(ctx, `UPDATE notify_channels SET last_action = $2 WHERE id = $1`, p.channel, b)
	return err
}

type numbered struct {
	n     int
	id    uuid.UUID
	typ   string
	title string
}

func (h Handler) byNumbers(ctx context.Context, p person, ns []int) ([]numbered, []int, error) {
	rows, err := h.DB.Query(ctx, `SELECT n, entity_id, type, title FROM digest_items WHERE user_id = $1 AND n = ANY ($2) ORDER BY n`, p.user, ns)
	if err != nil {
		return nil, nil, err
	}
	found, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (numbered, error) {
		var x numbered
		err := r.Scan(&x.n, &x.id, &x.typ, &x.title)
		return x, err
	})
	if err != nil {
		return nil, nil, err
	}
	var missing []int
	for _, n := range ns {
		if !slices.ContainsFunc(found, func(x numbered) bool { return x.n == n }) {
			missing = append(missing, n)
		}
	}
	return found, missing, nil
}

func (h Handler) done(ctx context.Context, p person, cmd Command) (string, error) {
	var targets []numbered
	var missing []int
	if len(cmd.Numbers) > 0 {
		var err error
		if targets, missing, err = h.byNumbers(ctx, p, cmd.Numbers); err != nil {
			return "", err
		}
	} else {
		matches, err := h.match(ctx, p, cmd.Text)
		if err != nil {
			return "", err
		}
		switch {
		case len(matches) == 0:
			return p.t("I found nothing open like “%s”. Say list to see today’s items with their numbers.", cmd.Text), nil
		case len(matches) > 1:
			return h.ask(ctx, p, matches)
		}
		targets = matches
	}
	var lines []string
	var acts []action
	err := pgx.BeginFunc(ctx, h.DB, func(tx pgx.Tx) error {
		for _, x := range targets {
			act, line, err := h.complete(ctx, tx, p, x)
			if err != nil {
				return err
			}
			lines = append(lines, line)
			if act != nil {
				acts = append(acts, *act)
			}
		}
		if len(acts) == 0 {
			return nil
		}
		return h.remember(ctx, tx, p, acts)
	})
	if err != nil {
		return "", err
	}
	for _, n := range missing {
		lines = append(lines, p.t("No item %d in your latest list: say list to see it again.", n))
	}
	if len(acts) > 0 {
		lines = append(lines, p.t(cheer(p, len(acts))))
	}
	return strings.Join(lines, "\n"), nil
}

// cheer is a short word of encouragement, different as the day goes on.
func cheer(p person, n int) string {
	options := []string{"Nice work.", "One less thing on your mind.", "Keep going.", "Well done."}
	if n > 1 {
		options = []string{"That is a productive moment.", "Great progress.", "Well done."}
	}
	return options[(p.now.Hour()+p.now.Minute())%len(options)]
}

func (h Handler) complete(ctx context.Context, tx pgx.Tx, p person, x numbered) (*action, string, error) {
	switch x.typ {
	case "task":
		t, err := tasks.Get(ctx, tx, p.user, x.id)
		if errors.Is(err, graph.ErrNotFound) {
			return nil, p.t("“%s” is gone from Axon.", x.title), nil
		}
		if err != nil {
			return nil, "", err
		}
		if t.Status == "done" {
			return nil, p.t("“%s” was already done.", t.Title), nil
		}
		done := "done"
		if _, err := tasks.Update(ctx, tx, p.user, x.id, graph.UserActor, t.UpdatedAt, tasks.Fields{Status: &done}); err != nil {
			return nil, "", err
		}
		line := "✅ " + t.Title
		if t.Recurrence != nil {
			line += " " + p.t("(the next one is planned)")
		}
		return &action{Type: "task_done", ID: x.id, Status: t.Status, Title: t.Title}, line, nil
	case "habit":
		hb, err := habits.Get(ctx, tx, p.user, x.id, p.now)
		if errors.Is(err, graph.ErrNotFound) {
			return nil, p.t("“%s” is gone from Axon.", x.title), nil
		}
		if err != nil {
			return nil, "", err
		}
		if _, err := habits.CheckIn(ctx, tx, p.user, x.id, p.today, "done", "", p.now); err != nil {
			return nil, "", err
		}
		return &action{Type: "habit_done", ID: x.id, Day: p.today.Format(time.DateOnly), Title: hb.Title}, "🌱 " + hb.Title, nil
	case "milestone":
		m, err := goals.GetMilestone(ctx, tx, p.user, x.id)
		if errors.Is(err, graph.ErrNotFound) {
			return nil, p.t("“%s” is gone from Axon.", x.title), nil
		}
		if err != nil {
			return nil, "", err
		}
		if m.DoneAt != nil {
			return nil, p.t("“%s” was already done.", m.Title), nil
		}
		yes := true
		if _, err := goals.UpdateMilestone(ctx, tx, p.user, x.id, graph.UserActor, m.UpdatedAt, goals.MilestoneFields{Done: &yes}); err != nil {
			return nil, "", err
		}
		return &action{Type: "milestone_done", ID: x.id, Title: m.Title}, "🏁 " + m.Title, nil
	}
	return nil, "", fmt.Errorf("unknown item type %q", x.typ)
}

// match finds open tasks, today's habits, and open milestones whose title
// is closest to what the person wrote. One clear winner is returned alone;
// close ones are returned to ask which.
func (h Handler) match(ctx context.Context, p person, text string) ([]numbered, error) {
	query := map[string]bool{}
	stems, _ := knowledge.TokensIn(text, p.lang)
	for _, s := range stems {
		if !slices.Contains(completionStems(p.lang), s) && !filler[s] {
			query[s] = true
		}
	}
	if len(query) == 0 {
		return nil, nil
	}
	rows, err := h.DB.Query(ctx, `SELECT t.id, 'task', e.title, coalesce(least(t.scheduled_on, t.due_on) <= $2::date, false) FROM tasks t JOIN entities e ON e.id = t.id
		WHERE e.user_id = $1 AND t.status NOT IN ('done', 'cancelled')
		UNION ALL
		SELECT m.id, 'milestone', e.title, coalesce(m.due_on <= $2::date + 14, false) FROM milestones m JOIN entities e ON e.id = m.id WHERE e.user_id = $1 AND m.done_at IS NULL
		LIMIT 1000`, p.user, p.today)
	if err != nil {
		return nil, err
	}
	type cand struct {
		numbered
		soon  bool
		score float64
	}
	cands, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (cand, error) {
		var c cand
		err := r.Scan(&c.id, &c.typ, &c.title, &c.soon)
		return c, err
	})
	if err != nil {
		return nil, err
	}
	hs, err := habits.List(ctx, h.DB, p.user, "active", p.now)
	if err != nil {
		return nil, err
	}
	for _, hb := range hs {
		if hb.State.CheckedIn == nil {
			cands = append(cands, cand{numbered: numbered{id: hb.ID, typ: "habit", title: hb.Title}, soon: hb.State.ScheduledToday})
		}
	}
	for i := range cands {
		ts, _ := knowledge.TokensIn(cands[i].title, p.lang)
		title := map[string]bool{}
		for _, s := range ts {
			title[s] = true
		}
		if len(title) == 0 {
			continue
		}
		common := 0
		for s := range query {
			if title[s] {
				common++
			}
		}
		if common == 0 {
			continue
		}
		cands[i].score = (float64(common)/float64(len(query)) + float64(common)/float64(len(title))) / 2
		if cands[i].soon {
			cands[i].score += 0.05
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	var near []numbered
	for _, c := range cands {
		if c.score < 0.34 || len(near) == 5 {
			break
		}
		if len(near) > 0 && c.score < cands[0].score-0.2 {
			break
		}
		near = append(near, c.numbered)
	}
	if len(near) > 1 && cands[0].score >= 0.5 && cands[0].score-cands[1].score >= 0.2 {
		near = near[:1]
	}
	return near, nil
}

func completionStems(lang string) []string {
	var out []string
	for _, w := range completion {
		out = append(out, knowledge.StemIn(w, lang))
	}
	return out
}

// ask lists close matches with numbers, so the next answer can pick one.
func (h Handler) ask(ctx context.Context, p person, matches []numbered) (string, error) {
	items := make([]digest.Item, len(matches))
	lines := []string{p.t("Which one?")}
	for i, m := range matches {
		items[i] = digest.Item{N: i + 1, ID: m.id, Type: m.typ, Text: m.title}
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, m.title))
	}
	if err := digest.Remember(ctx, h.DB, p.user, items); err != nil {
		return "", err
	}
	lines = append(lines, p.t("Answer done and the number."))
	return strings.Join(lines, "\n"), nil
}

func (h Handler) add(ctx context.Context, p person, cmd Command, kind string) (string, error) {
	title := strings.TrimSpace(cmd.Text)
	if title == "" {
		return p.t("help"), nil
	}
	f := tasks.Fields{Title: &title}
	status := "inbox"
	detail := ""
	if cmd.When != nil {
		day, at := cmd.When.Resolve(p.today)
		status = "next"
		f.ScheduledOn = &day
		detail = day.Format(time.DateOnly)
		if at != "" {
			loc, err := clock.Zone(ctx, h.DB, p.user)
			if err != nil {
				return "", err
			}
			remind, err := time.ParseInLocation("2006-01-02 15:04", day.Format(time.DateOnly)+" "+at, loc)
			if err == nil {
				if remind.Before(p.now) && cmd.When.Days == 0 && cmd.When.Date == "" {
					remind = remind.AddDate(0, 0, 1) // a time already gone today means tomorrow
					day = day.AddDate(0, 0, 1)
					f.ScheduledOn = &day
				}
				f.RemindAt = &remind
				detail = remind.Format("2006-01-02 15:04") + " 🔔"
			}
		}
	}
	f.Status = &status
	var t tasks.Task
	err := pgx.BeginFunc(ctx, h.DB, func(tx pgx.Tx) error {
		var err error
		if t, err = tasks.Create(ctx, tx, p.user, graph.UserActor, f); err != nil {
			return err
		}
		return h.remember(ctx, tx, p, []action{{Type: "task_added", ID: t.ID, Title: t.Title}})
	})
	var ve tasks.ValidationError
	if errors.As(err, &ve) {
		return ve.Msg, nil
	}
	if err != nil {
		return "", err
	}
	line := "➕ " + t.Title
	if detail != "" {
		line += " — " + detail
	} else {
		line += " — " + p.t("in your inbox")
	}
	return line, nil
}

func (h Handler) note(ctx context.Context, p person, text, kind string) (string, error) {
	title := truncate(strings.Join(strings.Fields(text), " "), 60)
	when := p.now.In(time.UTC)
	path := fmt.Sprintf("0-Inbox/%s-%s.md", when.Format("200601021504"), notes.Slug(title))
	content := fmt.Sprintf("---\ntype: fleeting\ncreated: %s\nsource: %s\n---\n# %s\n\n%s\n", p.today.Format(time.DateOnly), kind, title, strings.TrimSpace(text))
	var n notes.Note
	err := pgx.BeginFunc(ctx, h.DB, func(tx pgx.Tx) error {
		var err error
		for i := 2; ; i++ {
			// Look before creating: a failed insert would end the transaction.
			if _, err := notes.GetByPath(ctx, tx, p.user, path); errors.Is(err, graph.ErrNotFound) {
				break
			} else if err != nil {
				return err
			}
			path = fmt.Sprintf("0-Inbox/%s-%s-%d.md", when.Format("200601021504"), notes.Slug(title), i)
		}
		if n, err = notes.Create(ctx, tx, p.user, graph.UserActor, path, content); err != nil {
			return err
		}
		return h.remember(ctx, tx, p, []action{{Type: "note_added", ID: n.ID, Title: title}})
	})
	if err != nil {
		return "", err
	}
	return "📝 " + p.t("Saved to your inbox: “%s”", title), nil
}

func (h Handler) snooze(ctx context.Context, p person, cmd Command) (string, error) {
	if len(cmd.Numbers) == 0 {
		return p.t("Say which item to move, for example: snooze 2 tomorrow."), nil
	}
	targets, missing, err := h.byNumbers(ctx, p, cmd.Numbers)
	if err != nil {
		return "", err
	}
	day, _ := cmd.When.Resolve(p.today)
	var lines []string
	var acts []action
	err = pgx.BeginFunc(ctx, h.DB, func(tx pgx.Tx) error {
		for _, x := range targets {
			if x.typ != "task" {
				lines = append(lines, p.t("Only tasks can be moved; “%s” stays.", x.title))
				continue
			}
			t, err := tasks.Get(ctx, tx, p.user, x.id)
			if errors.Is(err, graph.ErrNotFound) {
				lines = append(lines, p.t("“%s” is gone from Axon.", x.title))
				continue
			}
			if err != nil {
				return err
			}
			prev := ""
			if t.ScheduledOn != nil {
				prev = t.ScheduledOn.Format(time.DateOnly)
			}
			if _, err := tasks.Update(ctx, tx, p.user, x.id, graph.UserActor, t.UpdatedAt, tasks.Fields{ScheduledOn: &day}); err != nil {
				return err
			}
			acts = append(acts, action{Type: "task_moved", ID: x.id, Day: prev, Title: t.Title})
			lines = append(lines, "⏭️ "+t.Title+" → "+day.Format(time.DateOnly))
		}
		if len(acts) == 0 {
			return nil
		}
		return h.remember(ctx, tx, p, acts)
	})
	if err != nil {
		return "", err
	}
	for _, n := range missing {
		lines = append(lines, p.t("No item %d in your latest list: say list to see it again.", n))
	}
	return strings.Join(lines, "\n"), nil
}

func (h Handler) undo(ctx context.Context, p person) (string, error) {
	var raw []byte
	if err := h.DB.QueryRow(ctx, `SELECT last_action FROM notify_channels WHERE id = $1`, p.channel).Scan(&raw); err != nil {
		return "", err
	}
	var acts []action
	_ = json.Unmarshal(raw, &acts)
	if len(acts) == 0 {
		return p.t("Nothing to undo."), nil
	}
	var lines []string
	err := pgx.BeginFunc(ctx, h.DB, func(tx pgx.Tx) error {
		for i := len(acts) - 1; i >= 0; i-- {
			a := acts[i]
			var err error
			switch a.Type {
			case "task_done":
				var t tasks.Task
				if t, err = tasks.Get(ctx, tx, p.user, a.ID); err == nil {
					status := a.Status
					_, err = tasks.Update(ctx, tx, p.user, a.ID, graph.UserActor, t.UpdatedAt, tasks.Fields{Status: &status})
				}
			case "habit_done":
				day, _ := time.Parse(time.DateOnly, a.Day)
				err = habits.Uncheck(ctx, tx, p.user, a.ID, day, p.now)
			case "milestone_done":
				var m goals.Milestone
				if m, err = goals.GetMilestone(ctx, tx, p.user, a.ID); err == nil {
					no := false
					_, err = goals.UpdateMilestone(ctx, tx, p.user, a.ID, graph.UserActor, m.UpdatedAt, goals.MilestoneFields{Done: &no})
				}
			case "task_added":
				err = tasks.Delete(ctx, tx, p.user, a.ID, graph.UserActor)
			case "note_added":
				err = notes.Delete(ctx, tx, p.user, a.ID, graph.UserActor)
			case "task_moved":
				var t tasks.Task
				if t, err = tasks.Get(ctx, tx, p.user, a.ID); err == nil {
					f := tasks.Fields{ClearScheduled: a.Day == ""}
					if d, perr := time.Parse(time.DateOnly, a.Day); perr == nil {
						f.ScheduledOn = &d
					}
					_, err = tasks.Update(ctx, tx, p.user, a.ID, graph.UserActor, t.UpdatedAt, f)
				}
			}
			if errors.Is(err, graph.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			lines = append(lines, "↩️ "+a.Title)
		}
		_, err := tx.Exec(ctx, `UPDATE notify_channels SET last_action = '[]' WHERE id = $1`, p.channel)
		return err
	})
	if err != nil {
		return "", err
	}
	return p.t("Undone:") + "\n" + strings.Join(lines, "\n"), nil
}

// list sends today's items again, numbered.
func (h Handler) list(ctx context.Context, p person) (string, error) {
	d, err := digest.Build(ctx, h.DB, p.user, digest.Daily, p.lang, p.now)
	if err != nil {
		return "", err
	}
	if err := digest.Remember(ctx, h.DB, p.user, d.Actionable()); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(d.Summary + "\n")
	for _, s := range d.Sections {
		fmt.Fprintf(&b, "\n%s %s\n", s.Emoji, s.Title)
		for _, it := range s.Items {
			if it.N > 0 {
				fmt.Fprintf(&b, "%d. %s", it.N, it.Text)
			} else {
				b.WriteString("• " + it.Text)
			}
			if it.Detail != "" {
				b.WriteString(" — " + it.Detail)
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String()), nil
}
