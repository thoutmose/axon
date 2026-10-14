package replies

import (
	"reflect"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	for in, want := range map[string]Command{
		"done 2":                             {Verb: Done, Numbers: []int{2}},
		"Done 1, 3":                          {Verb: Done, Numbers: []int{1, 3}},
		"fait 1-3":                           {Verb: Done, Numbers: []int{1, 2, 3}},
		"✅ 4":                                {Verb: Done, Numbers: []int{4}},
		"2 done":                             {Verb: Done, Numbers: []int{2}},
		"2":                                  {Verb: Done, Numbers: []int{2}},
		"1 and 2":                            {Verb: Done, Numbers: []int{1, 2}},
		"/done@AxonBot 5":                    {Verb: Done, Numbers: []int{5}},
		"done passport":                      {Verb: Done, Text: "passport"},
		"I finished the passport":            {Verb: Done, Text: "I finished the passport"},
		"j'ai terminé le rapport":            {Verb: Done, Text: "j'ai terminé le rapport"},
		"undo":                               {Verb: Undo},
		"Annule":                             {Verb: Undo},
		"add buy milk":                       {Verb: Add, Text: "buy milk"},
		"add buy 2 apples":                   {Verb: Add, Text: "buy 2 apples"},
		"add call the bank tomorrow":         {Verb: Add, Text: "call the bank", When: &When{Days: 1}},
		"ajoute appeler Paul demain à 18h30": {Verb: Add, Text: "appeler Paul", When: &When{Days: 1, At: "18:30"}},
		"add dentist at 3pm":                 {Verb: Add, Text: "dentist", When: &When{At: "15:00"}},
		"add pay rent 2026-11-01":            {Verb: Add, Text: "pay rent", When: &When{Date: "2026-11-01"}},
		"add review in 3 days":               {Verb: Add, Text: "review", When: &When{Days: 3}},
		"+ water the plants":                 {Verb: Add, Text: "water the plants"},
		"note the meeting moved":             {Verb: Note, Text: "the meeting moved"},
		"snooze 3":                           {Verb: Snooze, Numbers: []int{3}, When: &When{Days: 1}},
		"reporte 2 lundi":                    {Verb: Snooze, Numbers: []int{2}, When: &When{Days: -10 - int(time.Monday)}},
		"later 1 2 next week":                {Verb: Snooze, Numbers: []int{1, 2}, When: &When{Days: -1}},
		"list":                               {Verb: List},
		"aujourd'hui":                        {Verb: List},
		"help":                               {Verb: Help},
		"?":                                  {Verb: Help},
		"/start":                             {Verb: Help},
		"/start ABCD-EFGH":                   {Verb: Link, Text: "ABCD-EFGH"},
		"link abcd efgh":                     {Verb: Link, Text: "abcd efgh"},
		"thanks":                             {Verb: Unknown, Text: "thanks"},
		"done":                               {Verb: Help},
	} {
		if got := Parse(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	wed := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC) // a Wednesday
	for _, c := range []struct {
		w    When
		want string
	}{
		{When{Days: 1}, "2026-10-15"},
		{When{Days: -1}, "2026-10-19"},                        // next week: Monday
		{When{Days: -10 - int(time.Monday)}, "2026-10-19"},    // monday
		{When{Days: -10 - int(time.Wednesday)}, "2026-10-21"}, // wednesday: next week's
		{When{Days: -10 - int(time.Friday)}, "2026-10-16"},
		{When{Date: "2026-12-25"}, "2026-12-25"},
	} {
		if got, _ := c.w.Resolve(wed); got.Format(time.DateOnly) != c.want {
			t.Errorf("%+v → %s, want %s", c.w, got.Format(time.DateOnly), c.want)
		}
	}
}
