// Package demo creates and resets the shared demo account (EP-28): one user
// with realistic sample data in every area, so people can try Axon without
// a passkey and without seeing anyone's real data.
package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/budget"
	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/fsrs"
	"github.com/thoutmose/axon/api/internal/goals"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/habits"
	"github.com/thoutmose/axon/api/internal/images"
	"github.com/thoutmose/axon/api/internal/learning"
	"github.com/thoutmose/axon/api/internal/media"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/para"
	"github.com/thoutmose/axon/api/internal/projects"
	"github.com/thoutmose/axon/api/internal/roadmaps"
	"github.com/thoutmose/axon/api/internal/settings"
	"github.com/thoutmose/axon/api/internal/tasks"
)

// Name is the demo account's display name.
const Name = "Demo"

// Exists reports whether the demo account has been created.
func Exists(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE demo)`).Scan(&ok)
	return ok, err
}

// Reset deletes the demo account with everything it owns (every table
// cascades from users) and creates it again with fresh sample data. It runs
// in one transaction, so the demo is never half-built.
func Reset(ctx context.Context, pool *pgxpool.Pool, now time.Time) (uuid.UUID, error) {
	id := uuid.Must(uuid.NewV7())
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM users WHERE demo`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO users (id, display_name, timezone, demo) VALUES ($1, $2, 'Europe/Paris', true)`, id, Name); err != nil {
			return err
		}
		return seed(ctx, tx, id, now)
	})
	return id, err
}

func ptr[T any](v T) *T { return &v }

type seeder struct {
	ctx   context.Context
	tx    pgx.Tx
	user  uuid.UUID
	today time.Time
	err   error
}

func (s *seeder) day(offset int) *time.Time { return ptr(s.today.AddDate(0, 0, offset)) }

func (s *seeder) link(from uuid.UUID, relation string, to uuid.UUID) {
	if s.err == nil {
		_, s.err = graph.CreateLink(s.ctx, s.tx, s.user, graph.UserActor, from, relation, to)
	}
}

func (s *seeder) task(title, status string, context *string, due *time.Time, notes string) uuid.UUID {
	if s.err != nil {
		return uuid.Nil
	}
	t, err := tasks.Create(s.ctx, s.tx, s.user, graph.UserActor, tasks.Fields{Title: &title, Status: &status, Context: context, DueOn: due, Notes: &notes})
	s.err = err
	return t.ID
}

func (s *seeder) note(path, content string) uuid.UUID {
	if s.err != nil {
		return uuid.Nil
	}
	n, err := notes.Create(s.ctx, s.tx, s.user, graph.UserActor, path, content)
	s.err = err
	return n.ID
}

func seed(ctx context.Context, tx pgx.Tx, user uuid.UUID, now time.Time) error {
	loc, _ := time.LoadLocation("Europe/Paris")
	y, m, d := now.In(loc).Date()
	s := &seeder{ctx: ctx, tx: tx, user: user, today: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
	phone, computer, home := ptr("@phone"), ptr("@computer"), ptr("@home")

	// Tasks across GTD lists.
	call := s.task("Call the dentist to move Thursday's appointment", "next", phone, s.day(1), "")
	review := s.task("Weekly review", "next", home, s.day(0), "Inbox to zero, look at every project, pick next actions.")
	s.task("Renew passport", "next", computer, s.day(9), "Photos are in the drawer.")
	s.task("Reply to Sam about the hiking weekend", "inbox", nil, nil, "")
	s.task("Ideas for Lea's birthday", "inbox", nil, nil, "")
	s.task("Wait for the plumber's quote", "waiting", phone, s.day(3), "")
	s.task("Learn to bake sourdough", "someday", nil, nil, "")
	s.task("Back up the photo library", "done", computer, nil, "")
	if s.err == nil {
		_, s.err = tx.Exec(ctx, `UPDATE tasks SET recurrence = 'FREQ=WEEKLY;BYDAY=SU' WHERE id = $1`, review)
	}

	// A project on a board.
	var project projects.Project
	if s.err == nil {
		project, s.err = projects.Create(ctx, tx, user, graph.UserActor, projects.Fields{
			Title: ptr("Personal website"), Description: ptr("A portfolio with a blog, live by the end of the month."),
			WIPLimit: ptr(2), DocsFolder: ptr("1-Projects/personal-website"),
		})
	}
	boardCards := []struct{ title, column string }{
		{"Pick a static site generator", "done"}, {"Write the About page", "doing"}, {"Design the home page", "doing"},
		{"Set up deployment", "todo"}, {"Write the first blog post", "todo"}, {"Add an RSS feed", "backlog"}, {"Dark mode", "backlog"},
	}
	var cardIDs []uuid.UUID
	for _, c := range boardCards {
		if s.err != nil {
			break
		}
		var card projects.Card
		card, s.err = projects.AddCard(ctx, tx, user, project.ID, graph.UserActor, c.title, projects.CardInput{Column: ptr(c.column)})
		cardIDs = append(cardIDs, card.TaskID)
	}

	// Notes: PARA folders, Zettelkasten with wikilinks, a daily note.
	moc := s.note("3-Resources/Learning/Learning MOC.md", "---\ntags: [moc, learning]\n---\n# Learning MOC\n\nA map of how I learn.\n\n- [[Spaced repetition]]\n- [[Retrieval practice]]\n- [[The Feynman technique]]\n")
	spaced := s.note("4-Zettelkasten/Spaced repetition.md", "---\ntype: permanent\ntags: [learning, memory]\n---\n# Spaced repetition\n\nReviewing just before forgetting makes memories last longer. Intervals grow with each successful review.\n\n> [!tip] Pair it with [[Retrieval practice]]: test yourself instead of rereading.\n\nSource: Ebbinghaus' forgetting curve.\n")
	s.note("4-Zettelkasten/Retrieval practice.md", "---\ntype: permanent\ntags: [learning]\n---\n# Retrieval practice\n\nPulling information out of memory strengthens it more than putting it in again. See [[Spaced repetition]].\n")
	s.note("4-Zettelkasten/The Feynman technique.md", "---\ntype: permanent\ntags: [learning, writing]\n---\n# The Feynman technique\n\nExplain it to a twelve-year-old; the gaps you find are what to study next. Linked to [[Learning MOC]].\n")
	// A small Zettelkasten in three themes, so the knowledge graph has communities.
	for _, z := range [][2]string{
		{"4-Zettelkasten/Literature/Make It Stick.md", "---\ntype: literature\ntags: [learning]\n---\n# Make It Stick\n\nBrown, Roediger, and McDaniel show that retrieval practice, spaced repetition, and interleaving beat rereading. Effortful recall feels slower but memory lasts longer; the forgetting curve flattens with each review.\n"},
		{"4-Zettelkasten/Interleaving.md", "---\ntype: permanent\ntags: [learning]\n---\n# Interleaving\n\nMixing problem types during practice forces you to choose a strategy, which improves recall and transfer more than blocked practice. Works best with spaced repetition.\n"},
		{"4-Zettelkasten/Literature/Atomic Habits.md", "---\ntype: literature\ntags: [habits]\n---\n# Atomic Habits\n\nJames Clear: make habits obvious, attractive, easy, and satisfying. Habit stacking ties a new habit to a current one; identity-based habits start from who you want to become.\n"},
		{"4-Zettelkasten/Habit stacking.md", "---\ntype: permanent\ntags: [habits]\n---\n# Habit stacking\n\nAfter I pour my coffee, I will read one page. A reliable cue from a current habit makes the new habit obvious and easy to start.\n"},
		{"4-Zettelkasten/Identity-based habits.md", "---\ntype: permanent\ntags: [habits]\n---\n# Identity-based habits\n\nEvery check-in is a vote for the person you want to become. Habits last when they express an identity, not a goal; James Clear calls outcomes lagging measures.\n"},
		{"4-Zettelkasten/Literature/Designing Data-Intensive Applications.md", "---\ntype: literature\ntags: [systems]\n---\n# Designing Data-Intensive Applications\n\nMartin Kleppmann on replication, partitioning, transactions, and consensus. Leader-based replication trades replication lag for simplicity; consensus algorithms like Raft keep replicas agreeing.\n"},
		{"4-Zettelkasten/Replication lag.md", "---\ntype: permanent\ntags: [systems]\n---\n# Replication lag\n\nAsynchronous replication lets followers fall behind the leader. Read-your-writes and monotonic reads hide replication lag from users.\n"},
		{"4-Zettelkasten/Consensus with Raft.md", "---\ntype: permanent\ntags: [systems]\n---\n# Consensus with Raft\n\nRaft elects a leader and replicates a log; a write commits once a majority of replicas store it. Consensus survives the failure of a minority of nodes. See [[Replication lag]].\n"},
	} {
		s.note(z[0], z[1])
	}
	site := s.note("1-Projects/personal-website/Spec.md", "---\ntags: [project, web]\n---\n# Website spec\n\n| Page | Status |\n|---|---|\n| Home | ==in design== |\n| About | drafting |\n| Blog | planned |\n\n- [ ] Choose fonts\n- [x] Pick a generator\n")
	s.note("2-Areas/Health/Running log.md", "---\ntags: [health, running]\n---\n# Running log\n\n- 5 km in 29:40, felt easy\n- Intervals: 6 x 400 m\n")
	s.note("Daily/"+s.today.Format(time.DateOnly)+".md", "# "+s.today.Format("Monday 2 January 2006")+"\n\n## Focus\n- Finish the About page\n\n## Notes\n- Read about [[Spaced repetition]] on the train.\n")
	s.note("0-Inbox/Book ideas.md", "# Book ideas\n\n- *Atomic Habits*, James Clear\n- *How to Take Smart Notes*, Sönke Ahrens\n")

	// Goals with milestones, connected to their work.
	var health, career goals.Goal
	if s.err == nil {
		health, s.err = goals.Create(ctx, tx, user, graph.UserActor, goals.Fields{Title: ptr("Run a 10 km race"), Outcome: ptr("Finish the city 10 km in under an hour."), TargetDate: s.day(75)})
	}
	if s.err == nil {
		career, s.err = goals.Create(ctx, tx, user, graph.UserActor, goals.Fields{Title: ptr("Become a backend engineer"), Outcome: ptr("Ship a production service and pass a system design interview."), TargetDate: s.day(240)})
	}
	for i, ms := range []struct {
		goal  uuid.UUID
		title string
		done  bool
	}{{health.ID, "Run 5 km without stopping", true}, {health.ID, "Run 8 km", false}, {health.ID, "Race day", false},
		{career.ID, "Finish the Python roadmap", false}, {career.ID, "Read Designing Data-Intensive Applications", false}} {
		if s.err == nil {
			_, s.err = goals.AddMilestone(ctx, tx, user, ms.goal, graph.UserActor, goals.MilestoneFields{Title: ptr(ms.title), Done: ptr(ms.done), DueOn: s.day(20 * (i%3 + 1))})
		}
	}

	// PARA areas and resources, with the goals filed under their areas.
	for _, a := range []struct {
		title, standard string
		goal            uuid.UUID
	}{{"Health", "Sleep 7–8 h, move every day, eat mostly plants.", health.ID}, {"Career", "Learn every week and ship work I am proud of.", career.ID}} {
		var area para.Item
		if s.err == nil {
			area, s.err = para.Areas.Create(ctx, tx, user, graph.UserActor, para.Fields{Title: ptr(a.title), Description: ptr(a.standard)})
		}
		if s.err == nil {
			_, s.err = graph.CreateLink(ctx, tx, user, graph.UserActor, a.goal, para.InArea, area.ID)
		}
	}
	for _, r := range [][2]string{{"Recipes", ""}, {"Running", "https://www.runnersworld.com/"}} {
		if s.err == nil {
			_, s.err = para.Resources.Create(ctx, tx, user, graph.UserActor, para.Fields{Title: ptr(r[0]), URL: ptr(r[1])})
		}
	}

	// Films, series, and books: finished favourites make the recommendations explain themselves.
	for _, m := range []struct {
		title, kind, status, creators string
		genres                        []string
		rating, progress, total       int
	}{
		{"Arrival", "film", "done", "Denis Villeneuve", []string{"Science fiction"}, 9, 0, 0},
		{"Dune: Part One", "film", "want", "Denis Villeneuve", []string{"Science fiction"}, 0, 0, 0},
		{"Severance", "series", "doing", "Dan Erickson", []string{"Drama", "Mystery"}, 0, 4, 19},
		{"The Left Hand of Darkness", "book", "done", "Ursula K. Le Guin", []string{"Science fiction"}, 10, 0, 0},
		{"The Dispossessed", "book", "want", "Ursula K. Le Guin", []string{"Science fiction"}, 0, 0, 0},
		{"Designing Data-Intensive Applications", "book", "doing", "Martin Kleppmann", []string{"Software"}, 0, 210, 611},
	} {
		if s.err != nil {
			break
		}
		f := media.Fields{Title: ptr(m.title), Kind: ptr(m.kind), Status: ptr(m.status), Creators: ptr(m.creators), Genres: &m.genres, Progress: ptr(m.progress)}
		if m.rating > 0 {
			f.Rating = ptr(m.rating)
		}
		if m.total > 0 {
			f.Total = ptr(m.total)
		}
		_, s.err = media.Create(ctx, tx, user, graph.UserActor, f, nil, s.today)
	}

	// A month of budget: categories with budgets, income, and spending.
	cats := map[string]uuid.UUID{}
	for _, c := range []struct {
		name, kind string
		budget     int64
	}{{"Salary", "income", 0}, {"Rent", "expense", 95000}, {"Groceries", "expense", 40000}, {"Transport", "expense", 8000}, {"Eating out", "expense", 15000}} {
		if s.err != nil {
			break
		}
		f := budget.CategoryFields{Name: ptr(c.name), Kind: ptr(c.kind)}
		if c.budget > 0 {
			f.MonthlyCents = &c.budget
		}
		var cat budget.Category
		cat, s.err = budget.CreateCategory(ctx, tx, user, f)
		cats[c.name] = cat.ID
	}
	var checking, savings budget.Account
	if s.err == nil {
		checking, s.err = budget.CreateAccount(ctx, tx, user, budget.AccountFields{Name: ptr("Checking"), Type: ptr("asset"), OpeningCents: ptr(int64(180000))})
	}
	if s.err == nil {
		savings, s.err = budget.CreateAccount(ctx, tx, user, budget.AccountFields{Name: ptr("Savings"), Type: ptr("asset"), Role: ptr("savings"), OpeningCents: ptr(int64(500000))})
	}
	if s.err == nil {
		_, s.err = budget.CreateAccount(ctx, tx, user, budget.AccountFields{Name: ptr("Visa"), Type: ptr("liability"), OpeningCents: ptr(int64(-24000))})
	}
	if s.err == nil {
		var pot budget.Pot
		pot, s.err = budget.CreatePot(ctx, tx, user, budget.PotFields{Name: ptr("Summer holiday"), TargetCents: ptr(int64(150000)),
			TargetDate: ptr(s.today.AddDate(0, 8, 0).Format(time.DateOnly))}, s.today)
		for _, c := range []int64{20000, 15000} {
			if s.err == nil {
				_, s.err = budget.MovePot(ctx, tx, user, pot.ID, budget.PotMove{AmountCents: c}, s.today)
			}
		}
	}
	if s.err == nil {
		groceries := cats["Groceries"]
		_, s.err = budget.CreateRule(ctx, tx, user, budget.RuleFields{Name: ptr("Supermarkets"), Value: ptr("market"), CategoryID: &groceries, Tag: ptr("food")})
	}
	first := time.Date(s.today.Year(), s.today.Month(), 1, 0, 0, 0, 0, time.UTC)
	for _, t := range []struct {
		day       int
		kind, cat string
		cents     int64
		payee     string
	}{{0, "income", "Salary", 260000, "Employer"}, {1, "expense", "Rent", 95000, "Landlord"}, {2, "expense", "Groceries", 6420, "Market"},
		{4, "expense", "Transport", 7500, "Monthly pass"}, {6, "expense", "Eating out", 2850, "Pizzeria"}, {8, "expense", "Groceries", 4315, "Supermarket"}} {
		on := first.AddDate(0, 0, t.day)
		if s.err != nil || on.After(s.today) {
			continue
		}
		id := cats[t.cat]
		_, s.err = budget.CreateTransaction(ctx, tx, user, budget.TransactionFields{OccurredOn: ptr(on.Format(time.DateOnly)), Kind: ptr(t.kind),
			AmountCents: &t.cents, CategoryID: &id, AccountID: &checking.ID, Payee: ptr(t.payee)})
	}
	if s.err == nil { // pay yourself first: a monthly transfer to savings
		_, s.err = budget.CreateRecurring(ctx, tx, user, budget.RecurringFields{Name: ptr("Savings"), Rule: ptr("FREQ=MONTHLY;BYMONTHDAY=2"),
			Start: ptr(first.AddDate(0, 0, 1).Format(time.DateOnly)), TransactionFields: budget.TransactionFields{Kind: ptr("transfer"),
				AmountCents: ptr(int64(30000)), AccountID: &checking.ID, ToAccountID: &savings.ID}})
	}

	// Habits with a month of history.
	now = s.today.Add(12 * time.Hour)
	type habitSeed struct {
		title, stack, twoMin, identity, at, where string
		every                                     int // checked in every n days
	}
	var habitIDs []uuid.UUID
	for _, h := range []habitSeed{
		{"Read 20 pages", "After I pour my morning coffee", "Read one page", "I am a reader", "07:15", "on the balcony", 1},
		{"Run", "After I get home from work", "Put on running shoes", "I am a runner", "18:30", "in the park", 2},
		{"Meditate", "After I brush my teeth at night", "One mindful breath", "I am a calm person", "", "", 1},
	} {
		if s.err != nil {
			break
		}
		var habit habits.Habit
		habit, s.err = habits.Create(ctx, tx, user, graph.UserActor, habits.Fields{Title: ptr(h.title), StackAfter: ptr(h.stack), TwoMinute: ptr(h.twoMin),
			Identity: ptr(h.identity), IntentionTime: ptr(h.at), IntentionPlace: ptr(h.where)}, now)
		habitIDs = append(habitIDs, habit.ID)
		for back := 30; back >= 1 && s.err == nil; back-- {
			if back%h.every != 0 || back%9 == 0 { // a few missed days, as in real life
				continue
			}
			status := "done"
			if back%5 == 0 {
				status = "two_minute"
			}
			_, s.err = tx.Exec(ctx, `INSERT INTO habit_checkins (habit_id, day, status) VALUES ($1, $2, $3)`, habit.ID, s.day(-back), status)
		}
	}

	// A habit to break, with the four laws inverted, and the habits scorecard.
	if s.err == nil {
		_, s.err = habits.Create(ctx, tx, user, graph.UserActor, habits.Fields{Title: ptr("Scroll my phone in bed"), Kind: ptr("break"),
			Cue: ptr("Charge the phone in the kitchen"), Craving: ptr("I sleep worse and wake up tired"),
			TwoMinute: ptr("Log out of social apps every night"), Reward: ptr("Two euros into the jar each time"),
			Identity: ptr("I am a person who sleeps well")}, now)
	}
	if s.err == nil {
		s.err = habits.PutScorecard(ctx, tx, user, []habits.ScorecardItem{
			{Behavior: "Wake up", Rating: "="}, {Behavior: "Check my phone", Rating: "-"}, {Behavior: "Make coffee", Rating: "="},
			{Behavior: "Read on the balcony", Rating: "+"}, {Behavior: "Commute by bike", Rating: "+"}, {Behavior: "Snack at 4 pm", Rating: "-"},
		})
	}

	// Roadmaps from the library, with progress.
	var python, design roadmaps.Roadmap
	if s.err == nil {
		python, s.err = roadmaps.Import(ctx, tx, user, graph.UserActor, "python")
	}
	if s.err == nil {
		design, s.err = roadmaps.Import(ctx, tx, user, graph.UserActor, "system-design")
	}
	for i, sk := range python.Skills {
		if s.err != nil || i > 4 {
			break
		}
		status := "done"
		if i >= 3 {
			status = "learning"
		}
		_, s.err = roadmaps.UpdateSkill(ctx, tx, user, sk.ID, graph.UserActor, roadmaps.SkillFields{Status: &status})
	}
	var soft roadmaps.Roadmap
	if s.err == nil {
		soft, s.err = roadmaps.Import(ctx, tx, user, graph.UserActor, "soft-skills")
	}
	for i, sk := range soft.Skills {
		if s.err != nil || i > 2 {
			break
		}
		status := map[int]string{0: "done", 1: "learning", 2: "learning"}[i]
		_, s.err = roadmaps.UpdateSkill(ctx, tx, user, sk.ID, graph.UserActor, roadmaps.SkillFields{Status: &status})
	}
	if s.err == nil && len(soft.Skills) > 2 {
		s.link(soft.Skills[2].ID, goals.Supports, career.ID)
		s.link(soft.Skills[0].ID, graph.RelatesTo, moc)
	}
	if s.err == nil && len(python.Skills) > 1 {
		s.link(python.Skills[1].ID, graph.RelatesTo, project.ID)
	}
	if s.err == nil && len(design.Skills) > 0 {
		_, s.err = roadmaps.UpdateSkill(ctx, tx, user, design.Skills[0].ID, graph.UserActor, roadmaps.SkillFields{Status: ptr("learning")})
	}

	// Flashcards from the learning notes, a few already reviewed; the first
	// with the why, an example, a memory image, and a drawing (EP-52).
	var demoCards []cards.Card
	for i, qa := range [][2]string{
		{"Why does spaced repetition work?", "Reviewing just before you would forget makes the memory last longer."},
		{"What is retrieval practice?", "Pulling information out of memory, for example by testing yourself."},
		{"Why test yourself instead of rereading?", "Retrieval strengthens memory more than putting it in again."},
		{"What is the Feynman technique?", "Explain it simply; the gaps you find are what to study next."},
		{"What does FSRS predict?", "The chance you still remember a card, to schedule the next review."},
	} {
		if s.err != nil {
			break
		}
		f := cards.Fields{Front: ptr(qa[0]), Back: ptr(qa[1]), Deck: ptr("Learning science")}
		if i == 0 {
			f.Elaboration = ptr("Each retrieval after some forgetting takes effort, and that effort rebuilds the memory more strongly than an easy one.")
			f.Example = ptr("Ten minutes a day for a week beats an hour the night before the exam.")
			f.Mnemonic = ptr("A gardener watering a plant a little every day while the neighbour floods theirs once a month and watches it wilt.")
		}
		var c cards.Card
		c, s.err = cards.Create(ctx, tx, user, graph.UserActor, f, &spaced, now)
		if s.err == nil && i < 2 {
			_, s.err = cards.Review(ctx, tx, user, c.ID, fsrs.Easy, nil, now.Add(-72*time.Hour))
		}
		demoCards = append(demoCards, c)
	}
	if s.err == nil && len(demoCards) == 5 {
		var drawing []byte
		if drawing, s.err = images.SanitizeSVG(forgettingCurve); s.err == nil {
			s.err = cards.SetImage(ctx, tx, user, demoCards[0].ID, drawing, "image/svg+xml")
		}
	}
	// A memory palace with three cards placed along the walk.
	if s.err == nil && len(demoCards) == 5 {
		var palace learning.Palace
		palace, s.err = learning.CreatePalace(ctx, tx, user, graph.UserActor, learning.PalaceFields{
			Title: ptr("My flat"), Description: ptr("From the front door to the balcony."),
			Loci: &[]string{"Front door", "Coat hook", "Kitchen sink", "Sofa", "Balcony"},
		})
		for i, c := range demoCards[1:4] {
			if s.err != nil {
				break
			}
			locus := i * 2
			_, s.err = cards.Update(ctx, tx, user, c.ID, graph.UserActor, cards.Fields{Palace: ptr(palace.ID.String()), Locus: &locus,
				Mnemonic: ptr([]string{"A quiz show host bursting out of the front door, demanding answers.", "Your coat answering exam questions out loud.", "A man scrubbing a blackboard clean in the sink, but the chalk keeps coming back."}[i])})
		}
	}
	// The spacing note was recalled three days ago, and is due again today; the last mixed test was eight days ago.
	if s.err == nil {
		_, s.err = tx.Exec(ctx, `INSERT INTO note_recalls (id, user_id, note_id, score, recalled_at, due_on) VALUES ($1, $2, $3, 0.6, $4, $5)`,
			uuid.Must(uuid.NewV7()), user, spaced, now.AddDate(0, 0, -3), now.Format(time.DateOnly))
	}
	if s.err == nil {
		_, s.err = tx.Exec(ctx, `INSERT INTO practice_tests (id, user_id, deck, questions, correct, taken_at) VALUES ($1, $2, '', 5, 4, $3)`,
			uuid.Must(uuid.NewV7()), user, now.AddDate(0, 0, -8))
	}

	// Progress rewards on, with a freeze in stock, so the demo shows them.
	if s.err == nil {
		doc := settings.Defaults()
		doc.Gamification.Enabled = true
		s.err = settings.Put(ctx, tx, user, doc)
	}
	if s.err == nil {
		_, s.err = tx.Exec(ctx, `INSERT INTO streak_freezes (id, user_id, bought_on) VALUES ($1, $2, $3)`, uuid.Must(uuid.NewV7()), user, s.today)
	}

	// A skills assessment, as taken before starting the roadmaps.
	if s.err == nil {
		_, s.err = roadmaps.CreateAssessment(ctx, tx, user, graph.UserActor, roadmaps.AssessmentInput{
			Profile: roadmaps.Profile{Role: "Support engineer", Experience: "3 years of scripting and SQL",
				Goals: "Become a backend engineer within a year.", HoursPerWeek: 6, Formats: []string{"book", "practice"}},
			Interests: []string{"python", "system-design", "soft-skills"},
			Ratings:   map[string]int{"python": 3, "system-design": 1, "soft-skills": 3},
			Answers:   map[string]int{},
		}, now)
	}

	// Everything connects.
	if s.err == nil {
		s.link(project.ID, goals.Supports, career.ID)
		s.link(python.ID, goals.Supports, career.ID)
		s.link(design.ID, goals.Supports, career.ID)
		s.link(habitIDs[1], goals.Supports, health.ID)
		s.link(habitIDs[0], goals.Supports, career.ID)
		s.link(site, graph.RelatesTo, project.ID)
		s.link(moc, graph.RelatesTo, python.ID)
		s.link(spaced, graph.RelatesTo, habitIDs[0])
		s.link(call, graph.RelatesTo, health.ID)
		s.link(cardIDs[1], graph.RelatesTo, site)
	}
	if s.err != nil {
		return fmt.Errorf("demo seed: %w", s.err)
	}
	return seedProposals(ctx, tx, user, review)
}

// seedProposals adds an agent run with pending proposals, so the approval
// queue can be tried without an AI provider.
func seedProposals(ctx context.Context, tx pgx.Tx, user, task uuid.UUID) error {
	run := uuid.Must(uuid.NewV7())
	if _, err := tx.Exec(ctx, `INSERT INTO agent_runs (id, user_id, agent, skill_version, model, status, tools_called, proposals, finished_at)
		VALUES ($1, $2, 'task-manager', '1.0.0', 'demo/scripted', 'completed', ARRAY['list_tasks', 'create_task:proposed', 'update_task:proposed'], 2, now())`, run, user); err != nil {
		return err
	}
	var version time.Time
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM tasks WHERE id = $1`, task).Scan(&version); err != nil {
		return err
	}
	for _, p := range []struct{ tool, input, summary, reason string }{
		{"create_task", `{"title":"Book a table for Lea's birthday","status":"next","context":"@phone","reason":"Her birthday is in your inbox."}`,
			"Create “Book a table for Lea's birthday” in Next (@phone)", "Her birthday is in your inbox."},
		{"update_task", fmt.Sprintf(`{"id":%q,"context":"@computer","reason":"You do your review at the desk.","version":%q}`, task, version.Format(time.RFC3339Nano)),
			"Set the context of “Weekly review” to @computer", "You do your review at the desk."},
	} {
		if _, err := tx.Exec(ctx, `INSERT INTO agent_proposals (id, user_id, run_id, agent, tool, input, summary, reason)
			VALUES ($1, $2, $3, 'task-manager', $4, $5, $6, $7)`, uuid.Must(uuid.NewV7()), user, run, p.tool, p.input, p.summary, p.reason); err != nil {
			return err
		}
	}
	return nil
}

// forgettingCurve is the demo card's drawing: memory fading, and lifted
// again by each spaced review.
const forgettingCurve = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 260" role="img" aria-label="Memory fades, and each spaced review lifts it higher and makes it fade more slowly">
<rect width="400" height="260" fill="#f4f1ea"/>
<line x1="40" y1="220" x2="380" y2="220" stroke="#555" stroke-width="2"/>
<line x1="40" y1="220" x2="40" y2="30" stroke="#555" stroke-width="2"/>
<path d="M40 40 C70 120 90 150 120 160" fill="none" stroke="#c0392b" stroke-width="3"/>
<path d="M120 40 C160 90 190 110 220 115" fill="none" stroke="#d35400" stroke-width="3"/>
<path d="M220 40 C270 65 310 75 370 80" fill="none" stroke="#27ae60" stroke-width="3"/>
<circle cx="120" cy="40" r="5" fill="#2c3e50"/><circle cx="220" cy="40" r="5" fill="#2c3e50"/>
<text x="210" y="250" text-anchor="middle" font-size="14" fill="#333">time</text>
<text x="20" y="130" text-anchor="middle" font-size="14" fill="#333" transform="rotate(-90 20 130)">memory</text>
<text x="128" y="32" font-size="12" fill="#2c3e50">review</text><text x="228" y="32" font-size="12" fill="#2c3e50">review</text>
</svg>`
