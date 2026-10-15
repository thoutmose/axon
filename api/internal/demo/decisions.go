package demo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/decisions"
)

// demoDecision is a decision of the past year: made some days ago, to be
// reviewed after some more; happened is nil while it waits.
type demoDecision struct {
	ago, after         int
	title, choice, exp string
	p                  float64
	happened           *bool
	outcome, lessons   string
}

var came, missed = ptr(true), ptr(false)

// The demo's decision journal (EP-114): a year of choices about the
// roadmap, the side projects, and running, reviewed, a little too sure of
// itself; one due today, two waiting.
var demoDecisions = []demoDecision{
	{340, 180, "PostgreSQL rather than MongoDB for Axon", "PostgreSQL with goose migrations", "No painful schema change in six months", 0.85, came, "Migrations stayed small and reversible.", "Relational first; JSON columns where the shape is open."},
	{320, 30, "A 90-minute study block before work", "Study from 6:30, phone in another room", "20 hours of study a week for a month", 0.7, missed, "About 14 hours a week: two late nights a week broke it.", "Plan for the bad days, not the good ones."},
	{300, 56, "Rust before Kubernetes", "The Rust book, then Rustlings", "Finish the Rust book in eight weeks", 0.6, missed, "Chapter 16 after eight weeks; ownership took longer.", "Double the estimate for a new paradigm."},
	{280, 90, "Self-host on Proxmox rather than a VPS", "Three VMs on the home server", "Under two hours of upkeep a month", 0.7, came, "About an hour a month, once the backups were automated.", ""},
	{250, 45, "Apply to three backend roles", "Applications with the Axon repository as a portfolio", "At least one technical interview", 0.75, came, "Two interviews, one take-home.", "A real project says more than a CV."},
	{220, 30, "Run three times a week", "Couch to 5K, after work in the park", "Run 5 km without stopping by the end of the month", 0.8, came, "5 km on day 27.", ""},
	{200, 30, "Write the budget module test-first", "Tests before each handler", "No bug reported in the budget for a month", 0.6, came, "None; two edge cases were caught by the tests.", "TDD pays most where money is involved."},
	{170, 21, "Move deploys to GitOps", "A deploy script that only takes green CI", "Deploy in under 10 minutes", 0.9, came, "About 6 minutes.", ""},
	{150, 40, "Skip the conference to finish the project", "Stay home and ship", "Ship the project on time", 0.8, missed, "Two weeks late: the scope grew.", "Cut scope before cutting learning."},
	{120, 75, "Take the AWS Solutions Architect exam in the autumn", "Six weeks of practice tests", "Pass on the first try", 0.65, came, "Passed with 812.", ""},
	{100, 90, "Read one paper a week", "Papers We Love, on Sunday mornings", "Twelve papers in a quarter", 0.7, missed, "Seven papers.", "Pick the papers a month ahead."},
	{80, 60, "SQLite for the side project", "One file, Litestream backups", "No need to move to a server database in a year", 0.8, came, "Still fine at 2,000 users.", ""},
	{35, 34, "Move the home server's containers to Podman", "Rootless Podman with Quadlet units", "Every service runs rootless within a month", 0.7, nil, "", ""},
	{10, 50, "Take the CKA before the end of the year", "Killer.sh practice and the Kubernetes docs", "Pass on the first try", 0.6, nil, "", ""},
	{3, 90, "Learn Go generics before Rust async", "Rewrite the watch's helpers with generics", "Use generics in three packages by the review", 0.75, nil, "", ""},
}

func seedDecisions(ctx context.Context, tx pgx.Tx, user uuid.UUID, today time.Time) error {
	for _, d := range demoDecisions {
		decided := today.AddDate(0, 0, -d.ago)
		in := decisions.Input{Title: d.title, Choice: d.choice, Expected: d.exp, Probability: ptr(d.p),
			DecidedOn: decided.Format(time.DateOnly), ReviewOn: decided.AddDate(0, 0, d.after).Format(time.DateOnly)}
		made, err := decisions.Create(ctx, tx, user, in, today)
		if err != nil {
			return err
		}
		if d.happened != nil {
			review := decisions.Review{Happened: d.happened, Outcome: d.outcome, Lessons: d.lessons}
			if _, err := decisions.Resolve(ctx, tx, user, made.ID, review, decided.AddDate(0, 0, d.after)); err != nil {
				return err
			}
		}
	}
	return nil
}
