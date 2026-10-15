package app

import (
	"encoding/json"
	"testing"
)

// The collection (EP-119): cards raised from bronze to gold by what the
// person does, only when progress rewards are on.
func TestCollection(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(409, "GET", "/v1/progress/collection", nil)
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc["gamification"] = map[string]any{"enabled": true}
	body, _ := json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)

	var h struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read 20 pages","two_minute":"Read one page"}`)), &h)
	for back := 6; back >= 0; back-- {
		alice.mustDo(200, "PUT", "/v1/habits/"+h.ID+"/checkins/"+day(-back), []byte(`{"status":"done"}`))
	}
	alice.mustDo(201, "POST", "/v1/practice", []byte(`{"title":"Median of Two Sorted Arrays","platform":"leetcode","difficulty":"hard","outcome":"solved"}`))

	var c struct {
		Cards []struct {
			ID    string  `json:"id"`
			Title string  `json:"title"`
			Tier  string  `json:"tier"`
			Value float64 `json:"value"`
			Next  *struct {
				Tier   string  `json:"tier"`
				Target float64 `json:"target"`
			} `json:"next"`
		} `json:"cards"`
		Earned int `json:"earned"`
		Total  int `json:"total"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/progress/collection", nil), &c)
	byID := map[string]int{}
	for i, k := range c.Cards {
		byID[k.ID] = i
	}
	habit := c.Cards[byID["habit:"+h.ID]]
	if habit.Title != "Read 20 pages" || habit.Tier != "bronze" || habit.Value != 7 || habit.Next == nil || habit.Next.Target != 21 {
		t.Fatalf("habit card = %+v", habit)
	}
	if hard := c.Cards[byID["practice:hard"]]; hard.Tier != "bronze" || hard.Next.Tier != "silver" {
		t.Fatalf("hard card = %+v", hard)
	}
	if easy := c.Cards[byID["practice:easy"]]; easy.Tier != "" || easy.Next.Target != 10 {
		t.Fatalf("easy card = %+v", easy)
	}
	if c.Earned != 2 || c.Total != 3*len(c.Cards) {
		t.Fatalf("earned %d of %d over %d cards", c.Earned, c.Total, len(c.Cards))
	}
}
