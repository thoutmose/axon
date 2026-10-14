package demo

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/fsrs"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/images"
	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/practice"
	"github.com/thoutmose/axon/api/internal/roadmaps"
)

// The demo's roadmap (EP-95): the Top-5% plan, started on the Monday a
// week or more ago, so the demo always shows it under way: the first
// phase's skills being learnt, the first week's tasks done, the habits
// checked in, and flashcards from the first course with their diagrams.

//go:embed mit6042j/*.svg
var diagrams embed.FS

func seedRoadmap(ctx context.Context, tx pgx.Tx, user uuid.UUID, today, now time.Time) (roadmaps.Started, error) {
	d := roadmaps.Library()[0]
	week := today.AddDate(0, 0, -7)
	start := week.AddDate(0, 0, -((int(week.Weekday()) + 6) % 7)) // that week's Monday
	planStart, _ := time.Parse(time.DateOnly, d.Plan.Start)
	plan, err := roadmaps.StartPlan(ctx, tx, user, graph.UserActor, d, int(start.Sub(planStart).Hours()/24), now)
	if err != nil {
		return plan, err
	}

	learning := "learning"
	for _, sk := range plan.Roadmap.Skills {
		switch sk.Title {
		case "Python", "Data structures and algorithms", "Maths", "Software engineering", "Bash", "Linux", "Consistency", "Learning how to learn":
			if _, err := roadmaps.UpdateSkill(ctx, tx, user, sk.ID, graph.UserActor, roadmaps.SkillFields{Status: &learning}); err != nil {
				return plan, err
			}
		}
	}

	// The habits since the start: one rest day spent on the study block,
	// the review missed twice, the weekend sessions kept.
	days := int(today.Sub(start).Hours() / 24)
	for i := range days {
		on := start.AddDate(0, 0, i)
		for h, keep := range []bool{i != 3, i != 3 && i != 6, on.Weekday() == time.Saturday, on.Weekday() == time.Sunday} {
			if !keep {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO habit_checkins (habit_id, day, status) VALUES ($1, $2, 'done')`, plan.Habits[h], on); err != nil {
				return plan, err
			}
		}
	}

	// The first week's tasks are done but one.
	for i, id := range plan.Tasks {
		if i == 8 {
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE tasks SET status = 'done', completed_at = due_on + time '18:00'
			WHERE id = $1 AND recurrence IS NULL AND due_on < $2`, id, today); err != nil {
			return plan, err
		}
	}
	if err := seedPractice(ctx, tx, user, today, plan.Roadmap.Skills); err != nil {
		return plan, err
	}

	// The first course's literature note, and its flashcards with diagrams.
	note, err := notes.Create(ctx, tx, user, graph.UserActor, "4-Zettelkasten/Literature/MIT 6.042J Mathematics for Computer Science.md", logicNote)
	if err != nil {
		return plan, err
	}
	for _, sk := range plan.Roadmap.Skills {
		if sk.Title == "Maths" {
			if _, err := graph.CreateLink(ctx, tx, user, graph.UserActor, note.ID, graph.RelatesTo, sk.ID); err != nil {
				return plan, err
			}
		}
	}
	deck := "MIT 6.042J: logic"
	for i, c := range logicCards {
		card, err := cards.Create(ctx, tx, user, graph.UserActor, cards.Fields{Front: &c[0], Back: &c[1], Deck: &deck}, &note.ID, now)
		if err != nil {
			return plan, err
		}
		if c[2] != "" {
			raw, err := diagrams.ReadFile("mit6042j/mit6042j-" + c[2] + ".svg")
			if err != nil {
				return plan, err
			}
			svg, err := images.SanitizeSVG(string(raw))
			if err != nil {
				return plan, fmt.Errorf("%s: %w", c[2], err)
			}
			if err := cards.SetImage(ctx, tx, user, card.ID, svg, "image/svg+xml"); err != nil {
				return plan, err
			}
		}
		if i < 6 {
			if _, err := cards.Review(ctx, tx, user, card.ID, fsrs.Good, nil, now.Add(-96*time.Hour)); err != nil {
				return plan, err
			}
		}
	}
	return plan, nil
}

// logicCards are the first flashcards of MIT 6.042J: front, back, and the
// diagram that goes with them.
var logicCards = [][3]string{
	{"What is a mathematical proof of a proposition?", "A chain of logical deductions that leads to the proposition from a base set of axioms.", ""},
	{"What is a proposition?", "A statement that is either true or false (exactly one of the two).", ""},
	{"What does a truth table show?", "The true/false value of a proposition for every possible setting of its variables (2ⁿ rows for n variables).", ""},
	{"When is ¬P true?", "Exactly when P is false.", "not"},
	{"When is P ∧ Q true?", "Only when both P and Q are true.", "and"},
	{"When is P ∨ Q true?", "When at least one of P, Q is true (both true included).", "or"},
	{"P ⊕ Q vs P ∨ Q: on which row do they differ?", "Only on P = T, Q = T: ∨ is true there, ⊕ (exactly one true) is false.", "xor"},
	{"When is P → Q false?", "Only when P is true and Q is false. Otherwise (P false, or Q true) it is true.", "implies"},
	{"Why is P → Q true whenever P is false?", "The implication only promises Q if P holds; when P is false the promise can't be broken, so it is (vacuously) true.", ""},
	{"When is P ↔ Q true?", "When P and Q have the same value: both true or both false (they are logically equivalent).", "iff"},
	{"Which connective doesn't arise in ordinary speech?", "↔ (“if and only if”).", ""},
	{"What is the contrapositive of P → Q, and how do they relate?", "¬Q → ¬P. Same truth table column, so they are logically equivalent.", "contrapositive"},
	{"In P → Q, what are P and Q called?", "P is the hypothesis (if-part), Q the conclusion (then-part); each is a proposition.", ""},
	{"P = “it rains”, Q = “the ground is wet”. Which situation makes P → Q false?", "It rains and the ground is not wet (P true, Q false). A dry day with wet ground does not falsify it.", ""},
	{"What is the contrapositive of “if n² is odd, then n is odd”?", "“If n is even, then n² is even.” It is equivalent, and it's often the easier one to prove.", ""},
}

const logicNote = `---
type: literature
title: MIT 6.042J Mathematics for Computer Science
source_type: course
author: Eric Lehman, F. Thomson Leighton, Albert R. Meyer
url: https://ocw.mit.edu/courses/6-042j-mathematics-for-computer-science-spring-2015/
area: "[[maths]]"
roadmap: "[[top5-engineer]]"
tags: [maths, logic]
---
# MIT 6.042J Mathematics for Computer Science

## Summary (own words, no copy-paste)

A proof is a chain of logical deductions from axioms to a proposition: a statement that is either true or false. Truth tables give a compound proposition's value for every setting of its variables.

## Key ideas

| P | Q | P ∧ Q | P ∨ Q | P ⊕ Q | P → Q | P ↔ Q |
| --- | --- | --- | --- | --- | --- | --- |
| T | T | T | T | F | T | T |
| T | F | F | T | T | F | F |
| F | T | F | T | T | T | F |
| F | F | F | F | F | T | T |

- An implication is false only when its hypothesis is true and its conclusion false.
- ¬Q → ¬P, the contrapositive, has the same truth table as P → Q.

## Concrete examples

- "If it rains, the ground is wet" is broken only by rain on dry ground.
- To prove "if n² is odd, then n is odd", prove "if n is even, then n² is even".

## Permanent notes to extract

- [ ] Implication and its contrapositive
`

// seedPractice gives the demo twelve weeks of practice (EP-103): classic
// LeetCode problems on arrays, two pointers, stacks, lists, trees, and
// graphs, and the first OverTheWire Bandit levels, some solved with help
// and coming back to be re-solved today.
func seedPractice(ctx context.Context, tx pgx.Tx, user uuid.UUID, today time.Time, skills []roadmaps.Skill) error {
	skill := map[string]*uuid.UUID{}
	for _, sk := range skills {
		skill[sk.Title] = &sk.ID
	}
	type try struct {
		back    int
		outcome string
		minutes int
	}
	for _, p := range []struct {
		title, slug, difficulty string
		topics                  []string
		tries                   []try
	}{
		{"Two Sum", "two-sum", "easy", []string{"arrays", "hash table"}, []try{{80, "with_help", 30}, {77, "solved", 12}}},
		{"Contains Duplicate", "contains-duplicate", "easy", []string{"arrays", "hash table"}, []try{{78, "solved", 8}}},
		{"Valid Anagram", "valid-anagram", "easy", []string{"strings", "hash table"}, []try{{75, "solved", 10}}},
		{"Group Anagrams", "group-anagrams", "medium", []string{"strings", "hash table"}, []try{{70, "with_help", 40}, {67, "solved", 22}, {60, "solved", 15}}},
		{"Top K Frequent Elements", "top-k-frequent-elements", "medium", []string{"arrays", "heap"}, []try{{62, "solved", 28}}},
		{"Product of Array Except Self", "product-of-array-except-self", "medium", []string{"arrays", "prefix sum"}, []try{{55, "with_help", 45}, {52, "with_help", 25}, {3, "with_help", 20}}},
		{"Longest Consecutive Sequence", "longest-consecutive-sequence", "medium", []string{"arrays", "hash table"}, []try{{48, "solved", 30}, {41, "solved", 18}}},
		{"Valid Palindrome", "valid-palindrome", "easy", []string{"two pointers", "strings"}, []try{{45, "solved", 9}}},
		{"3Sum", "3sum", "medium", []string{"two pointers", "sorting"}, []try{{40, "not_solved", 50}, {39, "with_help", 35}, {36, "solved", 25}, {29, "solved", 18}}},
		{"Container With Most Water", "container-with-most-water", "medium", []string{"two pointers", "greedy"}, []try{{34, "solved", 20}}},
		{"Best Time to Buy and Sell Stock", "best-time-to-buy-and-sell-stock", "easy", []string{"arrays", "sliding window"}, []try{{30, "solved", 11}}},
		{"Longest Substring Without Repeating Characters", "longest-substring-without-repeating-characters", "medium", []string{"sliding window", "hash table"}, []try{{27, "with_help", 40}, {24, "solved", 20}}},
		{"Valid Parentheses", "valid-parentheses", "easy", []string{"stack", "strings"}, []try{{22, "solved", 10}}},
		{"Min Stack", "min-stack", "medium", []string{"stack", "design"}, []try{{19, "solved", 25}, {12, "solved", 10}}},
		{"Binary Search", "binary-search", "easy", []string{"binary search"}, []try{{16, "solved", 7}}},
		{"Reverse Linked List", "reverse-linked-list", "easy", []string{"linked list"}, []try{{13, "with_help", 20}, {10, "solved", 8}}},
		{"Merge Two Sorted Lists", "merge-two-sorted-lists", "easy", []string{"linked list"}, []try{{11, "solved", 12}}},
		{"Invert Binary Tree", "invert-binary-tree", "easy", []string{"trees", "dfs"}, []try{{8, "solved", 9}}},
		{"Number of Islands", "number-of-islands", "medium", []string{"graphs", "bfs", "dfs"}, []try{{6, "not_solved", 45}, {5, "with_help", 30}, {2, "with_help", 25}}},
		{"Course Schedule", "course-schedule", "medium", []string{"graphs", "topological sort"}, []try{{4, "with_help", 40}, {1, "solved", 26}}},
		{"Climbing Stairs", "climbing-stairs", "easy", []string{"dynamic programming"}, []try{{1, "not_solved", 20}}},
	} {
		in := practice.Input{Title: p.title, URL: "https://leetcode.com/problems/" + p.slug + "/", Platform: "leetcode", Difficulty: p.difficulty,
			Topics: p.topics, Language: "python", SkillID: skill["Data structures and algorithms"]}
		for i, t := range p.tries {
			in.AttemptInput = practice.AttemptInput{Outcome: t.outcome, Minutes: &t.minutes, On: today.AddDate(0, 0, -t.back).Format(time.DateOnly)}
			if _, _, err := practice.Log(ctx, tx, user, in, today); err != nil {
				return fmt.Errorf("practice %s, try %d: %w", p.title, i, err)
			}
		}
	}
	for level := range 9 {
		minutes := 5 + level*3
		outcome := "solved"
		if level == 5 || level == 8 {
			outcome = "with_help"
		}
		in := practice.Input{Title: fmt.Sprintf("Bandit level %d → %d", level, level+1), URL: fmt.Sprintf("https://overthewire.org/wargames/bandit/bandit%d.html", level+1),
			Platform: "overthewire", Difficulty: "easy", Topics: []string{"bash", "linux"}, Language: "bash", SkillID: skill["Bash"],
			AttemptInput: practice.AttemptInput{Outcome: outcome, Minutes: &minutes, On: today.AddDate(0, 0, -(9 - level)).Format(time.DateOnly)}}
		if _, _, err := practice.Log(ctx, tx, user, in, today); err != nil {
			return err
		}
	}
	return nil
}
