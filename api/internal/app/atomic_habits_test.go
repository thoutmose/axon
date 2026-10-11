package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestAtomicHabitsIdentityIntentionsBreakingAndScorecard(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	var read habitJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read 20 pages","two_minute":"Read one page",
		"stack_after":"After I pour my morning coffee","intention_time":"07:15","intention_place":"on the balcony",
		"identity":"I am a reader","craving":"Learn something before work","reward":"Tick the box with a nice pen",
		"contract_partner":"Sam","contract_pledge":"If I skip a week, I buy Sam lunch."}`)), &read)
	out := alice.mustDo(200, "GET", "/v1/habits/"+read.ID, nil)
	for _, want := range []string{`"kind":"build"`, `"identity":"I am a reader"`, `"intention_time":"07:15"`, `"intention_place":"on the balcony"`, `"contract_partner":"Sam"`, `"votes":0`} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("habit lacks %s: %s", want, out)
		}
	}

	// Each kept check-in is a vote for the identity.
	today := time.Now().UTC().Format(time.DateOnly)
	alice.mustDo(200, "PUT", "/v1/habits/"+read.ID+"/checkins/"+today, []byte(`{"status":"two_minute"}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/habits/"+read.ID, nil), []byte(`"votes":1`)) {
		t.Fatal("check-in not counted as a vote")
	}

	// A habit to break holds the inverted laws.
	var phone habitJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Scroll my phone in bed","kind":"break",
		"cue":"Charge the phone in the kitchen","craving":"Notice how tired I am the next day",
		"two_minute":"Log out of social apps every night","reward":"Pay 2 euros into a jar each time"}`)), &phone)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/habits/"+phone.ID, nil), []byte(`"kind":"break"`)) {
		t.Fatal("break habit not stored")
	}

	for _, bad := range []string{`{"kind":"maybe"}`, `{"intention_time":"25:00"}`, `{"intention_time":"7am"}`} {
		if code, _ := alice.do("PATCH", "/v1/habits/"+read.ID, []byte(bad), "If-Match", alice.etag("/v1/habits/"+read.ID)); code != 422 {
			t.Errorf("%s = %d", bad, code)
		}
	}

	// The scorecard: the day's behaviours, rated.
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/habits/scorecard", nil), []byte(`"items":[]`)) {
		t.Fatal("scorecard should start empty")
	}
	card := `{"items":[{"behavior":"Wake up","rating":"="},{"behavior":"Check my phone","rating":"-"},{"behavior":"Make coffee","rating":"+"}]}`
	if out := alice.mustDo(200, "PUT", "/v1/habits/scorecard", []byte(card)); !bytes.Contains(out, []byte(`"Check my phone","rating":"-"`)) {
		t.Fatalf("scorecard = %s", out)
	}
	alice.mustDo(422, "PUT", "/v1/habits/scorecard", []byte(`{"items":[{"behavior":"Nap","rating":"?"}]}`))
	alice.mustDo(422, "PUT", "/v1/habits/scorecard", []byte(`{"items":[{"behavior":" ","rating":"+"}]}`))
}
