package roadmaps

import (
	"slices"
	"testing"
)

func TestCriteriaComeFromTheProofSentence(t *testing.T) {
	cases := []struct {
		description string
		want        []string
	}{
		{"Phases 2 and 3. Proof: the Linux Foundation's Linux kernel development certificate (LFD103) and a patch sent upstream; all MIT 6.1810 xv6 labs pass make grade.",
			[]string{"the Linux Foundation's Linux kernel development certificate (LFD103) and a patch sent upstream", "all MIT 6.1810 xv6 labs pass make grade"}},
		{"Architecture, compilers, and theory (phases 2, 17). Proof: Nand to Tetris 12 projects and the Coursera certificate for part I, CS:APP labs, both Crafting Interpreters interpreters, 18.404 problem sets ≥ 80%.",
			[]string{"Nand to Tetris 12 projects and the Coursera certificate for part I", "CS:APP labs", "both Crafting Interpreters interpreters", "18.404 problem sets ≥ 80%"}},
		{"Phases 1, 4, 10. Proof: LFCS.", []string{"LFCS"}},
		{"Proof: 60 LeetCode (≥ 30 medium, 5 hard), a note.", []string{"60 LeetCode (≥ 30 medium, 5 hard)", "a note"}},
		{"Writes clearly for engineers.", nil},
	}
	for _, c := range cases {
		if got := Criteria(c.description); !slices.Equal(got, c.want) {
			t.Errorf("Criteria(%q) = %q, want %q", c.description, got, c.want)
		}
	}
}

func TestTheMarkFollowsTheSheet(t *testing.T) {
	m := func(ok bool) ValidationItem { return ValidationItem{Text: "x", Kind: "mandatory", OK: ok} }
	b := func(ok bool) ValidationItem { return ValidationItem{Text: "x", Kind: "bonus", OK: ok} }
	cases := []struct {
		items []ValidationItem
		mark  int
		flag  string
	}{
		{[]ValidationItem{m(true), m(false)}, 0, "incomplete"},
		{[]ValidationItem{m(false), b(true)}, 0, "incomplete"},
		{[]ValidationItem{m(true)}, 100, "ok"},
		{[]ValidationItem{m(true), b(false), b(false)}, 100, "ok"},
		{[]ValidationItem{m(true), b(true), b(false), b(false)}, 108, "ok"},
		{[]ValidationItem{m(true), b(true), b(true)}, 125, "outstanding"},
	}
	for _, c := range cases {
		if mark, flag := grade(c.items); mark != c.mark || flag != c.flag {
			t.Errorf("grade(%+v) = %d %s, want %d %s", c.items, mark, flag, c.mark, c.flag)
		}
	}
	if err := checkItems([]ValidationItem{b(true)}, ""); err == nil {
		t.Error("a sheet with no mandatory item was accepted")
	}
	if err := checkItems(nil, ""); err == nil {
		t.Error("an empty sheet was accepted")
	}
	items := []ValidationItem{{Text: "  spaced  ", Kind: ""}}
	if err := checkItems(items, ""); err != nil || items[0].Text != "spaced" || items[0].Kind != "mandatory" {
		t.Errorf("items were not cleaned: %v %+v", err, items)
	}
}
