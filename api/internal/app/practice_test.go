package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type problemJSON struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Platform string   `json:"platform"`
	Topics   []string `json:"topics"`
	Solved   bool     `json:"solved"`
	DueOn    *string  `json:"due_on"`
	Attempts []struct {
		On      string `json:"on"`
		Outcome string `json:"outcome"`
	} `json:"attempts"`
}

// The practice log (EP-103): problems with their attempts, re-solved at
// growing intervals, counted by difficulty and topic, never someone else's.
func TestPracticeLog(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	today := time.Now().UTC()
	day := func(back int) string { return today.AddDate(0, 0, -back).Format(time.DateOnly) }

	var p problemJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/practice", []byte(`{"title":"Two Sum","url":"https://leetcode.com/problems/two-sum/",
		"platform":"leetcode","difficulty":"easy","topics":["Arrays","hash table"],"language":"python","outcome":"with_help","minutes":25,"on":"`+day(10)+`"}`)), &p)
	if p.Solved || p.DueOn == nil || *p.DueOn != day(7) || p.Topics[0] != "arrays" {
		t.Fatalf("first attempt = %+v", p)
	}
	// The same link again is another attempt, not another problem.
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/practice", []byte(`{"title":"Two Sum","url":"https://leetcode.com/problems/two-sum/",
		"platform":"leetcode","difficulty":"easy","outcome":"solved","on":"`+day(6)+`"}`)), &p)
	if len(p.Attempts) != 2 || !p.Solved || p.DueOn == nil || *p.DueOn != day(-1) {
		t.Fatalf("second attempt = %+v", p)
	}
	var hard problemJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/practice", []byte(`{"title":"Course Schedule","url":"https://leetcode.com/problems/course-schedule/",
		"platform":"leetcode","difficulty":"medium","topics":["graphs","topological sort"],"outcome":"not_solved","on":"`+day(2)+`"}`)), &hard)
	alice.mustDo(201, "POST", "/v1/practice/"+hard.ID+"/attempts", []byte(`{"outcome":"solved","minutes":40,"on":"`+day(1)+`"}`))
	alice.mustDo(201, "POST", "/v1/practice", []byte(`{"title":"Bandit level 5","platform":"overthewire","difficulty":"easy","topics":["bash"],"outcome":"solved"}`))
	for _, bad := range []string{
		`{"title":"","platform":"leetcode","difficulty":"easy","outcome":"solved"}`,
		`{"title":"x","platform":"myspace","difficulty":"easy","outcome":"solved"}`,
		`{"title":"x","platform":"leetcode","difficulty":"brutal","outcome":"solved"}`,
		`{"title":"x","platform":"leetcode","difficulty":"easy","outcome":"nearly"}`,
		`{"title":"x","platform":"leetcode","difficulty":"easy","outcome":"solved","on":"2999-01-01"}`,
		`{"title":"x","url":"javascript:alert(1)","platform":"leetcode","difficulty":"easy","outcome":"solved"}`,
		`{"title":"x","platform":"leetcode","difficulty":"easy","outcome":"solved","minutes":0}`,
	} {
		alice.mustDo(422, "POST", "/v1/practice", []byte(bad))
	}

	// Due today: Two Sum (re-solve after an unaided solve is a week out), not the others.
	var list struct {
		Items []problemJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/practice?due=true", nil), &list)
	if len(list.Items) != 0 {
		t.Fatalf("due = %+v", list.Items)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/practice", nil), &list)
	if len(list.Items) != 3 {
		t.Fatalf("all = %d", len(list.Items))
	}

	var stats struct {
		Problems     int            `json:"problems"`
		Solved       int            `json:"solved"`
		ByDifficulty map[string]int `json:"by_difficulty"`
		Topics       []struct {
			Topic  string `json:"topic"`
			Solved int    `json:"solved"`
			Total  int    `json:"total"`
		} `json:"topics"`
		Weeks []struct {
			Week   string `json:"week"`
			Solved int    `json:"solved"`
		} `json:"weeks"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/practice/stats", nil), &stats)
	if stats.Problems != 3 || stats.Solved != 3 || stats.ByDifficulty["easy"] != 2 || stats.ByDifficulty["medium"] != 1 || len(stats.Weeks) != 12 {
		t.Fatalf("stats = %+v", stats)
	}

	bob.mustDo(404, "POST", "/v1/practice/"+hard.ID+"/attempts", []byte(`{"outcome":"solved"}`))
	bob.mustDo(404, "DELETE", "/v1/practice/"+hard.ID, nil)
	if out := bob.mustDo(200, "GET", "/v1/practice", nil); bytes.Contains(out, []byte("Course Schedule")) {
		t.Fatal("bob sees alice's problems")
	}
	alice.mustDo(204, "DELETE", "/v1/practice/"+hard.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/practice/"+hard.ID, nil)
}
