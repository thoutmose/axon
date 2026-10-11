package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type potJSON struct {
	ID           string `json:"id"`
	SavedCents   int64  `json:"saved_cents"`
	MonthlyCents *int64 `json:"monthly_cents"`
	Archived     bool   `json:"archived"`
	Moves        []struct {
		ID          string `json:"id"`
		AmountCents int64  `json:"amount_cents"`
	} `json:"moves"`
}

// A savings pot (tirelire) adds up its moves exactly, never goes below
// zero, says what to save each month, and is private.
func TestSavingsPots(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	var p potJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/pots", []byte(`{"name":"New bike","target_cents":120000,"target_date":"2099-12-31"}`)), &p)
	if p.SavedCents != 0 || p.MonthlyCents == nil || *p.MonthlyCents <= 0 {
		t.Fatalf("new pot = %+v", p)
	}
	alice.mustDo(422, "POST", "/v1/budget/pots", []byte(`{"name":"x","target_cents":0}`))
	alice.mustDo(422, "POST", "/v1/budget/pots", []byte(`{"target_cents":100}`))

	for _, cents := range []string{"2999", "1", "-1000"} {
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/pots/"+p.ID+"/moves", []byte(`{"amount_cents":`+cents+`,"note":"x"}`)), &p)
	}
	if p.SavedCents != 2000 || len(p.Moves) != 3 {
		t.Fatalf("after moves = %+v", p)
	}
	alice.mustDo(422, "POST", "/v1/budget/pots/"+p.ID+"/moves", []byte(`{"amount_cents":-2001}`)) // below zero
	alice.mustDo(422, "POST", "/v1/budget/pots/"+p.ID+"/moves", []byte(`{"amount_cents":0}`))
	// Removing the first deposit would leave the pot below zero; removing the withdrawal is fine.
	var deposit, withdrawal string
	for _, m := range p.Moves {
		switch m.AmountCents {
		case 2999:
			deposit = m.ID
		case -1000:
			withdrawal = m.ID
		}
	}
	alice.mustDo(422, "DELETE", "/v1/budget/pots/"+p.ID+"/moves/"+deposit, nil)
	_ = json.Unmarshal(alice.mustDo(200, "DELETE", "/v1/budget/pots/"+p.ID+"/moves/"+withdrawal, nil), &p)
	if p.SavedCents != 3000 {
		t.Fatalf("saved = %d", p.SavedCents)
	}

	_ = json.Unmarshal(alice.mustDo(200, "PATCH", "/v1/budget/pots/"+p.ID, []byte(`{"target_cents":3000,"target_date":""}`)), &p)
	if p.MonthlyCents != nil {
		t.Fatalf("reached pot still asks for %d a month", *p.MonthlyCents)
	}
	alice.mustDo(200, "PATCH", "/v1/budget/pots/"+p.ID, []byte(`{"archived":true}`))

	bob.mustDo(404, "PATCH", "/v1/budget/pots/"+p.ID, []byte(`{"name":"Mine"}`))
	bob.mustDo(404, "POST", "/v1/budget/pots/"+p.ID+"/moves", []byte(`{"amount_cents":100}`))
	bob.mustDo(404, "DELETE", "/v1/budget/pots/"+p.ID+"/moves/"+deposit, nil)
	bob.mustDo(404, "DELETE", "/v1/budget/pots/"+p.ID, nil)
	if out := bob.mustDo(200, "GET", "/v1/budget/pots", nil); bytes.Contains(out, []byte("New bike")) {
		t.Fatal("bob sees alice's pot")
	}
	if out := alice.mustDo(200, "GET", "/v1/budget/pots", nil); !bytes.Contains(out, []byte(`"archived":true`)) {
		t.Fatalf("pots = %s", out)
	}
	alice.mustDo(204, "DELETE", "/v1/budget/pots/"+p.ID, nil)
}
