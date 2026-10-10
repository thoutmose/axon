package goals

import "testing"

func TestCompute(t *testing.T) {
	cases := []struct {
		name        string
		m           *Measure
		done, total int
		method      string
		percent     int
		summary     string
	}{
		{"measure halfway", &Measure{Unit: "km", Start: 0, Target: 10, Current: 5}, 0, 0, "measure", 50, "5 of 10 km (from 0)"},
		{"measure going down", &Measure{Unit: "kg", Start: 90, Target: 80, Current: 87.5}, 0, 0, "measure", 25, "87.5 of 80 kg (from 90)"},
		{"measure past target is capped", &Measure{Unit: "books", Start: 0, Target: 12, Current: 15}, 1, 3, "measure", 100, "15 of 12 books (from 0)"},
		{"measure below start is zero", &Measure{Unit: "km", Start: 5, Target: 10, Current: 2}, 0, 0, "measure", 0, "2 of 10 km (from 5)"},
		{"items round down", nil, 2, 3, "items", 66, "2 of 3 milestones and linked tasks done"},
		{"items all done", nil, 4, 4, "items", 100, "4 of 4 milestones and linked tasks done"},
	}
	for _, c := range cases {
		p := compute(c.m, c.done, c.total)
		if p.Method != c.method || p.Percent == nil || *p.Percent != c.percent || p.Summary != c.summary {
			t.Errorf("%s: got %s %v %q", c.name, p.Method, deref(p.Percent), p.Summary)
		}
	}
	if p := compute(nil, 0, 0); p.Method != "none" || p.Percent != nil {
		t.Errorf("nothing to measure: %+v", p)
	}
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
