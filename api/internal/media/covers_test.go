package media

import "testing"

func TestBestMatchTakesOnlyTheSameWork(t *testing.T) {
	year := func(y int) *int { return &y }
	dune := Candidate{Title: "Dune", Creators: "Frank Herbert", Year: year(1965), HasCover: true, CoverKey: "1"}
	for _, c := range []struct {
		name  string
		item  Item
		cands []Candidate
		want  string // the chosen cover key, "" for none
	}{
		{"same title and author", Item{Title: "Dune", Creators: "Frank Herbert"}, []Candidate{dune}, "1"},
		{"case, accents, punctuation", Item{Title: "dune!", Creators: "Frank HERBERT"}, []Candidate{dune}, "1"},
		{"same title and year, no author known", Item{Title: "Dune", Year: year(1965)}, []Candidate{dune}, "1"},
		{"title only, nothing to contradict", Item{Title: "Dune"}, []Candidate{dune}, "1"},
		{"another author", Item{Title: "Dune", Creators: "Someone Else"}, []Candidate{dune}, ""},
		{"another year", Item{Title: "Dune", Year: year(2021)}, []Candidate{dune}, ""},
		{"another title", Item{Title: "Dune Messiah", Creators: "Frank Herbert"}, []Candidate{dune}, ""},
		{"no cover to take", Item{Title: "Dune"}, []Candidate{{Title: "Dune", Creators: "Frank Herbert"}}, ""},
		{"the first that fits", Item{Title: "Arrival", Creators: "Denis Villeneuve"}, []Candidate{
			{Title: "Arrival", Creators: "Someone", HasCover: true, CoverKey: "x"},
			{Title: "Arrival", Creators: "Denis Villeneuve", HasCover: true, CoverKey: "y"},
		}, "y"},
		{"Le Guin, one of several authors", Item{Title: "The Dispossessed", Creators: "Ursula K. Le Guin"}, []Candidate{
			{Title: "The dispossessed", Creators: "Ursula K. Le Guin, Another Editor", HasCover: true, CoverKey: "z"},
		}, "z"},
	} {
		got := bestMatch(c.item, c.cands)
		key := ""
		if got != nil {
			key = got.CoverKey
		}
		if key != c.want {
			t.Errorf("%s: chose %q, want %q", c.name, key, c.want)
		}
	}
}
