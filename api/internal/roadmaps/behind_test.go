package roadmaps

import (
	"testing"
	"time"
)

func TestBehind(t *testing.T) {
	today := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	ms := func(done bool) *PlanItemRef { return &PlanItemRef{Done: done} }
	late := PlanPhaseView{State: "past", End: "2026-10-10", Milestone: ms(false)}
	later := PlanPhaseView{State: "past", End: "2026-10-15", Milestone: ms(false)}
	cases := []struct {
		name   string
		phases []PlanPhaseView
		want   int
	}{
		{"past phase, milestone not reached", []PlanPhaseView{late}, 10},
		{"past phase, milestone reached", []PlanPhaseView{{State: "past", End: "2026-10-10", Milestone: ms(true)}}, 0},
		{"no milestone", []PlanPhaseView{{State: "past", End: "2026-10-10"}}, 0},
		{"current phase is not late", []PlanPhaseView{{State: "current", End: "2026-10-30", Milestone: ms(false)}}, 0},
		{"earliest late phase wins", []PlanPhaseView{late, later}, 10},
	}
	for _, c := range cases {
		if got := behind(c.phases, today); got != c.want {
			t.Errorf("%s: behind = %d, want %d", c.name, got, c.want)
		}
	}
}
