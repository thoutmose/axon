package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

type learnCard struct {
	ID          string  `json:"id"`
	Deck        string  `json:"deck"`
	Front       string  `json:"front"`
	Elaboration string  `json:"elaboration"`
	Example     string  `json:"example"`
	Mnemonic    string  `json:"mnemonic"`
	PalaceID    *string `json:"palace_id"`
	Locus       *int    `json:"locus"`
	Leech       bool    `json:"leech"`
	Reps        int     `json:"reps"`
}

func (c *client) dueCards(query string) []learnCard {
	c.e.t.Helper()
	var res struct {
		Items    []learnCard `json:"items"`
		DueTotal int         `json:"due_total"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/flashcards/due?limit=100"+query, nil), &res)
	return res.Items
}

func (c *client) setLearning(newPerDay int, interleave bool) {
	c.e.t.Helper()
	var doc map[string]any
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc["learning"] = map[string]any{"new_per_day": newPerDay, "retention": 0.9, "interleave": interleave, "type_answer": true}
	body, _ := json.Marshal(doc)
	c.mustDo(200, "PUT", "/v1/me/settings", body)
}

// The review engine paces new cards, mixes decks, enriches cards with the
// why, an example, and a memory image, places them in memory palaces, and
// names leeches for deliberate practice.
func TestLearningEngineCards(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	// Elaboration, a concrete example, and a vivid image travel with the card.
	var c learnCard
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Why does spacing beat cramming?","back":"Forgetting a little makes retrieval effortful, and effort strengthens memory.",
		"deck":"Learning","elaboration":"Each retrieval after some forgetting rebuilds the memory more strongly.","example":"Ten minutes a day for a week beats an hour the night before.",
		"mnemonic":"A gardener watering a plant a little every day while a neighbour floods theirs once."}`)), &c)
	if c.Elaboration == "" || c.Example == "" || c.Mnemonic == "" {
		t.Fatalf("card = %+v", c)
	}

	// New cards: 20 a day by default, then what the settings say.
	for i := range 24 {
		deck := "Biology"
		if i%2 == 1 {
			deck = "History"
		}
		alice.mustDo(201, "POST", "/v1/flashcards", fmt.Appendf(nil, `{"front":"Question %d","back":"Answer","deck":"%s"}`, i, deck))
	}
	if got := alice.dueCards(""); len(got) != 20 {
		t.Fatalf("due with 25 new cards = %d, want the 20-a-day allowance", len(got))
	}
	// Interleaving: across every deck, decks take turns.
	got := alice.dueCards("")
	for i := 1; i < 6; i++ {
		if got[i].Deck == got[i-1].Deck {
			t.Fatalf("decks not interleaved: %v", []string{got[0].Deck, got[1].Deck, got[2].Deck, got[3].Deck})
		}
	}
	alice.setLearning(5, false)
	if out := alice.mustDo(200, "GET", "/v1/me/settings", nil); !bytes.Contains(out, []byte(`"learning":{"new_per_day":5,"retention":0.9,"interleave":false,"type_answer":true}`)) {
		t.Fatalf("settings do not show the learning section: %s", out)
	}
	if got := alice.dueCards(""); len(got) != 5 || got[0].Deck != "Learning" || got[1].Front != "Question 0" || got[2].Front != "Question 1" {
		t.Fatalf("due with 5 a day, not mixed = %+v", got)
	}
	// Reviewing new cards uses the allowance up.
	for _, card := range alice.dueCards("")[:3] {
		alice.mustDo(200, "POST", "/v1/flashcards/"+card.ID+"/review", []byte(`{"rating":3,"duration_ms":4000}`))
	}
	// Two new cards are left today; the three just seen come back after their 10-minute step.
	if got := alice.dueCards(""); len(got) != 2 {
		t.Fatalf("after three reviews, due = %d", len(got))
	}

	// Memory palace: places in order, cards placed at them, walked in order.
	var palace struct {
		ID    string   `json:"id"`
		Loci  []string `json:"loci"`
		Cards []struct {
			ID    string `json:"id"`
			Locus int    `json:"locus"`
		} `json:"cards"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/palaces", []byte(`{"title":"My flat","loci":["Front door","Coat hook","Kitchen sink"]}`)), &palace)
	if len(palace.Loci) != 3 {
		t.Fatalf("palace = %+v", palace)
	}
	patch := func(cl *client, path, body string, want int) []byte {
		t.Helper()
		code, out := cl.do("PATCH", path, []byte(body))
		if code != want {
			t.Fatalf("PATCH %s = %d %s", path, code, out)
		}
		return out
	}
	_ = json.Unmarshal(patch(alice, "/v1/flashcards/"+c.ID, `{"palace_id":"`+palace.ID+`","locus":2}`, 200), &c)
	if c.PalaceID == nil || *c.PalaceID != palace.ID || c.Locus == nil || *c.Locus != 2 {
		t.Fatalf("placed card = %+v", c)
	}
	patch(alice, "/v1/flashcards/"+c.ID, `{"palace_id":"`+palace.ID+`","locus":3}`, 422) // no fourth place
	patch(bob, "/v1/flashcards/"+c.ID, `{"palace_id":"","locus":0}`, 404)
	var other learnCard
	_ = json.Unmarshal(bob.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Bob's"}`)), &other)
	patch(bob, "/v1/flashcards/"+other.ID, `{"palace_id":"`+palace.ID+`","locus":0}`, 422) // not bob's palace
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/palaces/"+palace.ID, nil), &palace)
	if len(palace.Cards) != 1 || palace.Cards[0].ID != c.ID || palace.Cards[0].Locus != 2 {
		t.Fatalf("palace cards = %+v", palace.Cards)
	}
	bob.mustDo(404, "GET", "/v1/palaces/"+palace.ID, nil)
	// Removing places takes cards at them out of the palace.
	patch(alice, "/v1/palaces/"+palace.ID, `{"loci":["Front door","Coat hook"]}`, 200)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/"+c.ID, nil), &c)
	if c.PalaceID != nil {
		t.Fatalf("card kept a place that is gone: %+v", c)
	}
	patch(alice, "/v1/flashcards/"+c.ID, `{"palace_id":"`+palace.ID+`","locus":0}`, 200)
	alice.mustDo(200, "GET", "/v1/palaces", nil)
	alice.mustDo(204, "DELETE", "/v1/palaces/"+palace.ID, nil)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/"+c.ID, nil), &c)
	if c.PalaceID != nil {
		t.Fatalf("card still in a deleted palace: %+v", c)
	}

	// Leeches are named and come first in deliberate practice.
	if _, err := pool.Exec(context.Background(), `UPDATE cards SET lapses = 7, reps = 12, state = 2, stability = 30, last_review = now() WHERE id = $1`, c.ID); err != nil {
		t.Fatal(err)
	}
	var leeches struct {
		Items []learnCard `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards?leech=true", nil), &leeches)
	if len(leeches.Items) != 1 || !leeches.Items[0].Leech {
		t.Fatalf("leeches = %+v", leeches.Items)
	}
	var practice struct {
		Items []learnCard `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/practice?limit=5", nil), &practice)
	if len(practice.Items) == 0 || practice.Items[0].ID != c.ID {
		t.Fatalf("practice = %+v", practice.Items)
	}
}

// Free recall of whole notes, cards drafted from a note's markup, practice
// tests, and the day's plan.
func TestLearningEngineNotes(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	content := `---
type: permanent
---
# Retrieval practice

Retrieval practice means recalling information from memory instead of rereading it.
Each successful retrieval strengthens long-term memory and slows forgetting.
Spaced repetition schedules retrieval just before forgetting, which makes learning durable.
Interleaving different topics forces discrimination between problem types.

What is the testing effect? :: Remembering better what you were tested on than what you reread.
Q: Who described the forgetting curve?
A: Hermann Ebbinghaus
A memory palace uses ==familiar places== to hold images.
**Elaboration**: explaining why and how an idea works.
`
	var note struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", fmt.Appendf(nil, `{"path":"Zettel/Retrieval practice.md","content":%q}`, content)), &note)

	// Recall on a blank page: some ideas back, some missed, and a next date.
	var res struct {
		Score float64 `json:"score"`
		Ideas []struct {
			Text     string `json:"text"`
			Recalled bool   `json:"recalled"`
		} `json:"ideas"`
		Interval int `json:"interval_days"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/notes/"+note.ID+"/recall", []byte(`{"text":"Retrieval practice: recalling from memory rather than rereading strengthens long-term memory."}`)), &res)
	if res.Score <= 0 || res.Score >= 1 || len(res.Ideas) < 4 || res.Interval != 1 {
		t.Fatalf("recall = %+v", res)
	}
	recalled, missed := 0, 0
	for _, i := range res.Ideas {
		if i.Recalled {
			recalled++
		} else {
			missed++
		}
	}
	if recalled == 0 || missed == 0 {
		t.Fatalf("ideas = %+v", res.Ideas)
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/notes/"+note.ID+"/recall", []byte(fmt.Sprintf(`{"text":%q}`, content))), &res)
	if res.Score < 0.8 || res.Interval != 2 && res.Interval != 3 {
		t.Fatalf("full recall = %+v", res)
	}
	if out := alice.mustDo(200, "GET", "/v1/notes/"+note.ID+"/recall", nil); bytes.Count(out, []byte(`"score"`)) != 2 {
		t.Fatalf("history = %s", out)
	}
	bob.mustDo(404, "POST", "/v1/notes/"+note.ID+"/recall", []byte(`{"text":"x"}`))
	var fleeting struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"0-Inbox/idea.md","content":"---\ntype: fleeting\n---\n# Idea\n\nSomething to think about later."}`)), &fleeting)
	alice.mustDo(422, "POST", "/v1/notes/"+fleeting.ID+"/recall", []byte(`{"text":"something"}`))

	// Cards drafted from the note's markup.
	var drafts struct {
		Items []struct {
			Front, Back, Kind string
			Exists            bool
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+note.ID+"/card-drafts", nil), &drafts)
	kinds := map[string]string{}
	for _, d := range drafts.Items {
		kinds[d.Kind] = d.Front + " → " + d.Back
	}
	if kinds["qa"] == "" || kinds["cloze"] != "A memory palace uses […] to hold images. → familiar places" || kinds["definition"] != "What is Elaboration? → explaining why and how an idea works." {
		t.Fatalf("drafts = %+v", drafts.Items)
	}
	alice.mustDo(201, "POST", "/v1/flashcards", fmt.Appendf(nil, `{"front":"Who described the forgetting curve?","back":"Hermann Ebbinghaus","source_id":"%s"}`, note.ID))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/"+note.ID+"/card-drafts", nil), &drafts)
	made := 0
	for _, d := range drafts.Items {
		if d.Exists {
			made++
		}
	}
	if made != 1 {
		t.Fatalf("drafts after adding one = %+v", drafts.Items)
	}

	// The plan: cards due, a first test offered once there are cards, the streak.
	for i := range 5 {
		alice.mustDo(201, "POST", "/v1/flashcards", fmt.Appendf(nil, `{"front":"Plan %d","back":"x"}`, i))
	}
	var plan struct {
		Cards struct {
			Due     int `json:"due"`
			New     int `json:"new"`
			Minutes int `json:"minutes"`
		} `json:"cards"`
		Test struct {
			Due bool `json:"due"`
		} `json:"test"`
		Fleeting   int `json:"fleeting"`
		StreakDays int `json:"streak_days"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/learning/plan", nil), &plan)
	if plan.Cards.Due != 6 || plan.Cards.New != 6 || plan.Cards.Minutes < 1 || !plan.Test.Due || plan.StreakDays != 1 {
		t.Fatalf("plan = %+v", plan)
	}
	alice.mustDo(204, "POST", "/v1/learning/tests", []byte(`{"deck":"","questions":10,"correct":7}`))
	alice.mustDo(422, "POST", "/v1/learning/tests", []byte(`{"questions":3,"correct":4}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/learning/plan", nil), &plan)
	if plan.Test.Due {
		t.Fatal("a test is still due right after one")
	}
	if out := bob.mustDo(200, "GET", "/v1/learning/plan", nil); !bytes.Contains(out, []byte(`"due":0`)) {
		t.Fatalf("bob's plan = %s", out)
	}
	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, _ := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			if !bytes.Contains(b, []byte(`"practice_tests"`)) || !bytes.Contains(b, []byte(`"note_recalls"`)) {
				t.Fatal("export lacks the learning records")
			}
		}
	}
}
