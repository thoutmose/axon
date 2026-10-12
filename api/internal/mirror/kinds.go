package mirror

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/media"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Actor marks changes the mirror makes, so it never reacts to its own writes.
const Actor graph.Actor = "vault"

const footer = "Kept in step with Axon: edit the title, the front matter, or the sections; delete the file to delete it."

// rendered is an item ready to write: its page, folder under the mirror
// root, and whether it belongs in Archive/.
type rendered struct {
	page     page
	folder   string
	archived bool
}

// kind knows how one item type maps to a note and back.
type kind struct {
	entity string   // graph entity type
	types  []string // front-matter types that create it from a new note
	// ownNotes: the "Notes" section is the item's own text field (tasks);
	// otherwise it is free text kept in vault_mirror.extra.
	ownNotes bool
	render   func(ctx context.Context, q db.Querier, user, id uuid.UUID, now time.Time) (rendered, error)
	apply    func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, inArchive bool, now time.Time) error
	create   func(ctx context.Context, tx pgx.Tx, user uuid.UUID, typ string, p parsed, inArchive bool, now time.Time) (uuid.UUID, error)
	remove   func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, now time.Time) error
}

var kinds = map[string]kind{}

func register(k kind) { kinds[k.entity] = k }

func kindForType(typ string) (kind, bool) {
	for _, k := range kinds {
		if slices.Contains(k.types, typ) {
			return k, true
		}
	}
	return kind{}, false
}

// --- helpers to read fields ---

func str(p parsed, key string) *string {
	if v, ok := p.fm[key]; ok {
		return &v
	}
	return nil
}

func sectionText(p parsed, heading string) *string {
	if v, ok := p.sections[heading]; ok {
		return &v
	}
	return nil
}

func title(p parsed) *string {
	if p.title == "" {
		return nil
	}
	return &p.title
}

// date reads "YYYY-MM-DD"; set reports the key was there, clear that it was empty.
func date(p parsed, key string) (t *time.Time, clear bool, err error) {
	v, ok := p.fm[key]
	if !ok {
		return nil, false, nil
	}
	if strings.TrimSpace(v) == "" {
		return nil, true, nil
	}
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(v))
	if err != nil {
		return nil, false, fmt.Errorf("%s: dates are YYYY-MM-DD", key)
	}
	return &d, false, nil
}

func number(p parsed, key string) (*int, error) {
	v, ok := p.fm[key]
	if !ok {
		return nil, nil
	}
	if strings.TrimSpace(v) == "" {
		zero := 0
		return &zero, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return nil, fmt.Errorf("%s is a whole number", key)
	}
	return &n, nil
}

func ymd(d *tasks.Date) string {
	if d == nil {
		return ""
	}
	return d.Format(time.DateOnly)
}

func ptrDay(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

// archiveStatus decides the status after an edit: moving the file into or
// out of Archive/ archives or restores the item; otherwise the front matter wins.
func archiveStatus(p parsed, current string, inArchive bool) *string {
	wasArchived := current == "archived"
	if inArchive != wasArchived {
		s := "active"
		if inArchive {
			s = "archived"
		} else if fm := p.fm["status"]; fm != "" && fm != "archived" {
			s = fm
		}
		return &s
	}
	return str(p, "status")
}

func base(typ string, id uuid.UUID) []kv {
	return []kv{{"type", typ}, {"axon_id", id.String()}}
}

func init() {
	register(kind{entity: "task", types: []string{"task"}, ownNotes: true,
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
			t, err := tasks.Get(ctx, q, user, id)
			if err != nil {
				return rendered{}, err
			}
			fm := append(base("task", id),
				kv{"status", t.Status}, kv{"context", yamlValue(deref(t.Context))}, kv{"scheduled", ymd(t.ScheduledOn)},
				kv{"due", ymd(t.DueOn)}, kv{"recurrence", yamlValue(deref(t.Recurrence))},
				kv{"completed", ptrDay(t.CompletedAt)}, kv{"created", t.CreatedAt.Format(time.DateOnly)})
			return rendered{folder: "Tasks", page: page{fm: fm, title: t.Title, sections: []section{{heading: "Notes", body: t.Notes}}, footer: footer}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, _ bool, _ time.Time) error {
			cur, err := tasks.Get(ctx, tx, user, id)
			if err != nil {
				return err
			}
			f, err := taskFields(p)
			if err != nil {
				return err
			}
			_, err = tasks.Update(ctx, tx, user, id, Actor, cur.UpdatedAt, f)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, _ bool, _ time.Time) (uuid.UUID, error) {
			f, err := taskFields(p)
			if err != nil {
				return uuid.Nil, err
			}
			t, err := tasks.Create(ctx, tx, user, Actor, f)
			return t.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
			return tasks.Delete(ctx, tx, user, id, Actor)
		},
	})

	register(kind{entity: "goal", types: []string{"goal"},
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
			g, err := goals.Get(ctx, q, user, id)
			if err != nil {
				return rendered{}, err
			}
			progress := "none yet"
			if g.Progress.Percent != nil {
				progress = fmt.Sprintf("%d %% (%d of %d)", *g.Progress.Percent, g.Progress.Done, g.Progress.Total)
			}
			fm := append(base("goal", id), kv{"status", g.Status}, kv{"target", ymd(g.TargetDate)}, kv{"progress", yamlValue(progress)})
			if m := g.Measure; m != nil {
				fm = append(fm, kv{"measure", yamlValue(fmt.Sprintf("%g / %g %s", m.Current, m.Target, m.Unit))})
			}
			var ms []string
			for _, m := range g.Milestones {
				box := " "
				if m.DoneAt != nil {
					box = "x"
				}
				line := fmt.Sprintf("- [%s] [[%s]]", box, fileName(m.Title))
				if m.DueOn != nil {
					line += " · due " + ymd(m.DueOn)
				}
				ms = append(ms, line)
			}
			return rendered{folder: "Goals", archived: g.Status == "archived", page: page{fm: fm, title: g.Title, footer: footer, sections: []section{
				{heading: "Outcome", body: g.Outcome}, {heading: "Milestones", body: strings.Join(ms, "\n"), readOnly: true}}}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, inArchive bool, _ time.Time) error {
			cur, err := goals.Get(ctx, tx, user, id)
			if err != nil {
				return err
			}
			f := goals.Fields{Title: title(p), Outcome: sectionText(p, "Outcome"), Status: archiveStatus(p, cur.Status, inArchive)}
			if f.TargetDate, f.ClearTargetDate, err = date(p, "target"); err != nil {
				return err
			}
			_, err = goals.Update(ctx, tx, user, id, Actor, cur.UpdatedAt, f)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, inArchive bool, _ time.Time) (uuid.UUID, error) {
			f := goals.Fields{Title: title(p), Outcome: sectionText(p, "Outcome"), Status: archiveStatus(p, "active", inArchive)}
			var err error
			if f.TargetDate, _, err = date(p, "target"); err != nil {
				return uuid.Nil, err
			}
			g, err := goals.Create(ctx, tx, user, Actor, f)
			return g.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
			return goals.Delete(ctx, tx, user, id, Actor)
		},
	})

	register(kind{entity: "milestone", types: []string{"milestone"},
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
			m, err := goals.GetMilestone(ctx, q, user, id)
			if err != nil {
				return rendered{}, err
			}
			g, err := goals.Get(ctx, q, user, m.GoalID)
			if err != nil {
				return rendered{}, err
			}
			fm := append(base("milestone", id), kv{"goal", yamlValue("[[" + fileName(g.Title) + "]]")}, kv{"due", ymd(m.DueOn)},
				kv{"done", strconv.FormatBool(m.DoneAt != nil)})
			return rendered{folder: "Milestones", archived: g.Status == "archived", page: page{fm: fm, title: m.Title, footer: footer}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, _ bool, _ time.Time) error {
			cur, err := goals.GetMilestone(ctx, tx, user, id)
			if err != nil {
				return err
			}
			f := goals.MilestoneFields{Title: title(p)}
			if f.DueOn, f.ClearDue, err = date(p, "due"); err != nil {
				return err
			}
			if v, ok := p.fm["done"]; ok {
				done := v == "true" || v == "yes" || v == "x"
				f.Done = &done
			}
			_, err = goals.UpdateMilestone(ctx, tx, user, id, Actor, cur.UpdatedAt, f)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, _ bool, _ time.Time) (uuid.UUID, error) {
			name := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(p.fm["goal"]), "[["), "]]")
			var goalID uuid.UUID
			if err := tx.QueryRow(ctx, `SELECT g.id FROM goals g JOIN entities e ON e.id = g.id WHERE e.user_id = $1 AND lower(e.title) = lower($2)
				ORDER BY e.created_at LIMIT 1`, user, name).Scan(&goalID); err != nil {
				return uuid.Nil, fmt.Errorf("a milestone needs goal: [[Goal title]] naming one of your goals")
			}
			f := goals.MilestoneFields{Title: title(p)}
			var err error
			if f.DueOn, _, err = date(p, "due"); err != nil {
				return uuid.Nil, err
			}
			m, err := goals.AddMilestone(ctx, tx, user, goalID, Actor, f)
			return m.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
			return goals.DeleteMilestone(ctx, tx, user, id, Actor)
		},
	})

	register(kind{entity: "project", types: []string{"project"},
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
			pr, err := projects.Get(ctx, q, user, id)
			if err != nil {
				return rendered{}, err
			}
			wip := ""
			if pr.WIPLimit != nil {
				wip = strconv.Itoa(*pr.WIPLimit)
			}
			fm := append(base("project", id), kv{"status", pr.Status}, kv{"wip_limit", wip},
				kv{"cards", yamlValue(fmt.Sprintf("%d of %d done", pr.Done, pr.Total))}, kv{"docs", yamlValue(pr.DocsFolder)})
			return rendered{folder: "Projects", archived: pr.Status == "archived", page: page{fm: fm, title: pr.Title, footer: footer,
				sections: []section{{heading: "Description", body: pr.Description}}}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, inArchive bool, _ time.Time) error {
			cur, err := projects.Get(ctx, tx, user, id)
			if err != nil {
				return err
			}
			f := projects.Fields{Title: title(p), Description: sectionText(p, "Description"), Status: archiveStatus(p, cur.Status, inArchive)}
			n, err := number(p, "wip_limit")
			if err != nil {
				return err
			}
			if n != nil && *n == 0 {
				f.ClearWIPLimit = true
			} else {
				f.WIPLimit = n
			}
			_, err = projects.Update(ctx, tx, user, id, Actor, cur.UpdatedAt, f)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, inArchive bool, _ time.Time) (uuid.UUID, error) {
			pr, err := projects.Create(ctx, tx, user, Actor, projects.Fields{Title: title(p), Description: sectionText(p, "Description"), Status: archiveStatus(p, "active", inArchive)})
			return pr.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
			return projects.Delete(ctx, tx, user, id, Actor)
		},
	})

	for _, pk := range []struct {
		k      para.Kind
		folder string
	}{{para.Areas, "Areas"}, {para.Resources, "Resources"}} {
		pk := pk
		register(kind{entity: pk.k.Type, types: []string{pk.k.Type},
			render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
				it, err := pk.k.Get(ctx, q, user, id)
				if err != nil {
					return rendered{}, err
				}
				fm := append(base(pk.k.Type, id), kv{"status", it.Status}, kv{"url", yamlValue(deref(it.URL))})
				return rendered{folder: pk.folder, archived: it.Status == "archived", page: page{fm: fm, title: it.Title, footer: footer,
					sections: []section{{heading: "Description", body: it.Description}}}}, nil
			},
			apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, inArchive bool, _ time.Time) error {
				cur, err := pk.k.Get(ctx, tx, user, id)
				if err != nil {
					return err
				}
				_, err = pk.k.Update(ctx, tx, user, id, Actor, cur.UpdatedAt, para.Fields{Title: title(p), Description: sectionText(p, "Description"),
					URL: str(p, "url"), Status: archiveStatus(p, cur.Status, inArchive)})
				return err
			},
			create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, inArchive bool, _ time.Time) (uuid.UUID, error) {
				it, err := pk.k.Create(ctx, tx, user, Actor, para.Fields{Title: title(p), Description: sectionText(p, "Description"), URL: str(p, "url"),
					Status: archiveStatus(p, "active", inArchive)})
				return it.ID, err
			},
			remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
				return pk.k.Delete(ctx, tx, user, id, Actor)
			},
		})
	}

	register(kind{entity: "habit", types: []string{"habit"},
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, now time.Time) (rendered, error) {
			h, err := habits.Get(ctx, q, user, id, now)
			if err != nil {
				return rendered{}, err
			}
			days := make([]string, len(h.Days))
			for i, d := range h.Days {
				days[i] = strconv.Itoa(d)
			}
			fm := append(base("habit", id), kv{"kind", h.Kind}, kv{"status", h.Status}, kv{"days", "[" + strings.Join(days, ", ") + "]"},
				kv{"intention_time", yamlValue(h.IntentionTime)}, kv{"intention_place", yamlValue(h.IntentionPlace)},
				kv{"stack_after", yamlValue(h.StackAfter)}, kv{"contract_partner", yamlValue(h.ContractPartner)},
				kv{"votes", strconv.Itoa(h.Votes)}, kv{"streak", strconv.Itoa(h.State.Streak)})
			hist, err := habits.History(ctx, q, user, id, h.State.Today.AddDate(0, 0, -13), h.State.Today.Time, now)
			if err != nil {
				return rendered{}, err
			}
			var lines []string
			for _, c := range hist {
				lines = append(lines, "- "+c.Day.Format("Mon 2 Jan")+": "+strings.ReplaceAll(c.Status, "_", "-"))
			}
			return rendered{folder: "Habits", archived: h.Status == "archived", page: page{fm: fm, title: h.Title, footer: footer, sections: []section{
				{heading: "Identity", body: h.Identity},
				{heading: "Make it obvious (cue)", body: h.Cue},
				{heading: "Make it attractive (craving)", body: h.Craving},
				{heading: "Make it easy (two-minute version)", body: h.TwoMinute},
				{heading: "Make it satisfying (reward)", body: h.Reward},
				{heading: "Habit contract", body: h.ContractPledge},
				{heading: "Recent check-ins", body: strings.Join(lines, "\n"), readOnly: true},
			}}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, inArchive bool, now time.Time) error {
			cur, err := habits.Get(ctx, tx, user, id, now)
			if err != nil {
				return err
			}
			f, err := habitFields(p)
			if err != nil {
				return err
			}
			f.Status = archiveStatus(p, cur.Status, inArchive)
			_, err = habits.Update(ctx, tx, user, id, Actor, cur.UpdatedAt, f, now)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, _ string, p parsed, inArchive bool, now time.Time) (uuid.UUID, error) {
			f, err := habitFields(p)
			if err != nil {
				return uuid.Nil, err
			}
			f.Status = archiveStatus(p, "active", inArchive)
			h, err := habits.Create(ctx, tx, user, Actor, f, now)
			return h.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, now time.Time) error {
			return habits.Delete(ctx, tx, user, id, Actor, now)
		},
	})

	register(kind{entity: media.Type, types: []string{"film", "series", "book"},
		render: func(ctx context.Context, q db.Querier, user, id uuid.UUID, _ time.Time) (rendered, error) {
			it, err := media.Get(ctx, q, user, id)
			if err != nil {
				return rendered{}, err
			}
			rating, year, total := "", "", ""
			if it.Rating != nil {
				rating = strconv.Itoa(*it.Rating)
			}
			if it.Year != nil {
				year = strconv.Itoa(*it.Year)
			}
			if it.Total != nil {
				total = strconv.Itoa(*it.Total)
			}
			fm := append(base(it.Kind, id), kv{"status", it.Status}, kv{"rating", rating}, kv{"progress", strconv.Itoa(it.Progress)},
				kv{"total", total}, kv{"year", year}, kv{"creators", yamlValue(it.Creators)}, kv{"genres", list(it.Genres)},
				kv{"started", yamlValue(deref(it.StartedOn))}, kv{"finished", yamlValue(deref(it.FinishedOn))}, kv{"source", it.Source})
			secs := []section{}
			if it.Overview != "" {
				secs = append(secs, section{heading: "Overview", body: it.Overview, readOnly: true})
			}
			secs = append(secs, section{heading: "Review", body: it.Review})
			folder := map[string]string{"film": "Media/Films", "series": "Media/Series", "book": "Media/Books"}[it.Kind]
			return rendered{folder: folder, page: page{fm: fm, title: it.Title, footer: footer, sections: secs}}, nil
		},
		apply: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, p parsed, _ bool, now time.Time) error {
			f, err := mediaFields(p)
			if err != nil {
				return err
			}
			_, err = media.Update(ctx, tx, user, id, Actor, f, now)
			return err
		},
		create: func(ctx context.Context, tx pgx.Tx, user uuid.UUID, typ string, p parsed, _ bool, now time.Time) (uuid.UUID, error) {
			f, err := mediaFields(p)
			if err != nil {
				return uuid.Nil, err
			}
			f.Kind = &typ
			it, err := media.Create(ctx, tx, user, Actor, f, nil, now)
			return it.ID, err
		},
		remove: func(ctx context.Context, tx pgx.Tx, user, id uuid.UUID, _ time.Time) error {
			return graph.DeleteEntity(ctx, tx, user, id, Actor)
		},
	})
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func taskFields(p parsed) (tasks.Fields, error) {
	f := tasks.Fields{Title: title(p), Notes: sectionText(p, "Notes"), Status: str(p, "status")}
	if v, ok := p.fm["context"]; ok {
		if v == "" {
			f.ClearContext = true
		} else {
			f.Context = &v
		}
	}
	if v, ok := p.fm["recurrence"]; ok {
		if v == "" {
			f.ClearRecurrence = true
		} else {
			f.Recurrence = &v
		}
	}
	var err error
	if f.ScheduledOn, f.ClearScheduled, err = date(p, "scheduled"); err != nil {
		return f, err
	}
	if f.DueOn, f.ClearDue, err = date(p, "due"); err != nil {
		return f, err
	}
	return f, nil
}

func habitFields(p parsed) (habits.Fields, error) {
	f := habits.Fields{Title: title(p), Kind: str(p, "kind"), IntentionTime: str(p, "intention_time"), IntentionPlace: str(p, "intention_place"),
		StackAfter: str(p, "stack_after"), ContractPartner: str(p, "contract_partner"),
		Identity: sectionText(p, "Identity"), Cue: sectionText(p, "Make it obvious (cue)"), Craving: sectionText(p, "Make it attractive (craving)"),
		TwoMinute: sectionText(p, "Make it easy (two-minute version)"), Reward: sectionText(p, "Make it satisfying (reward)"),
		ContractPledge: sectionText(p, "Habit contract")}
	if v, ok := p.fm["days"]; ok {
		f.Days = []int{}
		for _, d := range unlist(v) {
			n, err := strconv.Atoi(d)
			if err != nil {
				return f, errors.New("days are ISO weekdays, like [1, 2, 3, 4, 5]")
			}
			f.Days = append(f.Days, n)
		}
	}
	if f.Kind != nil && *f.Kind == "" {
		f.Kind = nil
	}
	return f, nil
}

func mediaFields(p parsed) (media.Fields, error) {
	f := media.Fields{Title: title(p), Status: str(p, "status"), Creators: str(p, "creators"), Review: sectionText(p, "Review"),
		StartedOn: str(p, "started"), FinishedOn: str(p, "finished")}
	if t, ok := p.fm["type"]; ok && slices.Contains(media.Kinds, t) {
		f.Kind = &t
	}
	for key, dst := range map[string]**int{"rating": &f.Rating, "progress": &f.Progress, "total": &f.Total, "year": &f.Year} {
		n, err := number(p, key)
		if err != nil {
			return f, err
		}
		*dst = n
	}
	if v, ok := p.fm["genres"]; ok {
		g := unlist(v)
		f.Genres = &g
	}
	return f, nil
}
