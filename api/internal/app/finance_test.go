package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type accountJSON struct {
	ID           string `json:"id"`
	BalanceCents int64  `json:"balance_cents"`
}

// Firefly-style finance: balances follow double entry through transfers,
// recurring transactions and CSV imports never record anything twice, and
// rules categorise what comes in.
func TestFinanceAccountsRecurringRulesAndImport(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	post := func(path, body string) string {
		var x struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", path, []byte(body)), &x)
		return x.ID
	}
	checking := post("/v1/budget/accounts", `{"name":"Checking","type":"asset","opening_cents":100000}`)
	savings := post("/v1/budget/accounts", `{"name":"Savings","type":"asset","role":"savings"}`)
	card := post("/v1/budget/accounts", `{"name":"Visa","type":"liability","opening_cents":-5000}`)
	alice.mustDo(422, "POST", "/v1/budget/accounts", []byte(`{"name":"Checking","type":"asset"}`))
	alice.mustDo(422, "POST", "/v1/budget/accounts", []byte(`{"name":"X","type":"crypto"}`))
	groceries := post("/v1/budget/categories", `{"name":"Groceries","kind":"expense","monthly_cents":30000}`)
	salary := post("/v1/budget/categories", `{"name":"Salary","kind":"income"}`)

	// A rule categorises and tags anything from the supermarket.
	alice.mustDo(422, "POST", "/v1/budget/rules", []byte(`{"name":"Empty","value":"x"}`)) // sets nothing
	post("/v1/budget/rules", `{"name":"Supermarket","field":"payee","operator":"contains","value":"carrefour","category_id":"`+groceries+`","tag":"food"}`)

	month := time.Now().UTC().Format("2006-01")
	day := month + "-01"
	post("/v1/budget/transactions", `{"occurred_on":"`+day+`","kind":"income","amount_cents":250000,"account_id":"`+checking+`","category_id":"`+salary+`"}`)
	var bought struct {
		CategoryID *string  `json:"category_id"`
		Tags       []string `json:"tags"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"expense","amount_cents":4321,"account_id":"`+card+`","payee":"CARREFOUR City"}`)), &bought)
	if bought.CategoryID == nil || *bought.CategoryID != groceries || len(bought.Tags) != 1 || bought.Tags[0] != "food" {
		t.Fatalf("rule did not categorise: %+v", bought)
	}
	post("/v1/budget/transactions", `{"occurred_on":"`+day+`","kind":"transfer","amount_cents":50000,"account_id":"`+checking+`","to_account_id":"`+savings+`"}`)
	post("/v1/budget/transactions", `{"occurred_on":"`+day+`","kind":"transfer","amount_cents":4321,"account_id":"`+checking+`","to_account_id":"`+card+`"}`) // pay the card
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"transfer","amount_cents":1,"account_id":"`+checking+`","to_account_id":"`+checking+`"}`))
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"transfer","amount_cents":1,"account_id":"`+checking+`","to_account_id":"`+savings+`","category_id":"`+groceries+`"}`))
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"expense","amount_cents":1,"account_id":"`+checking+`","to_account_id":"`+savings+`"}`))

	balances := func() map[string]int64 {
		var list struct {
			Items []accountJSON `json:"items"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget/accounts", nil), &list)
		out := map[string]int64{}
		for _, a := range list.Items {
			out[a.ID] = a.BalanceCents
		}
		return out
	}
	b := balances()
	if b[checking] != 100000+250000-50000-4321 || b[savings] != 50000 || b[card] != -5000-4321+4321 {
		t.Fatalf("balances = %v", b)
	}
	// Double entry: the total is openings + income − expenses; transfers cancel out.
	if b[checking]+b[savings]+b[card] != 100000-5000+250000-4321 {
		t.Fatalf("balances do not reconcile: %v", b)
	}
	// Transfers are not spending: the month counts only income and expenses.
	var m budgetMonthJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget?month="+month, nil), &m)
	if m.IncomeCents != 250000 || m.ExpenseCents != 4321 {
		t.Fatalf("month = %+v", m)
	}
	alice.mustDo(422, "DELETE", "/v1/budget/accounts/"+savings, nil) // used: archive instead
	alice.mustDo(200, "PATCH", "/v1/budget/accounts/"+savings, []byte(`{"archived":true,"name":"Rainy day"}`))
	if got := alice.mustDo(200, "GET", "/v1/budget/transactions?month="+month+"&tag=food", nil); bytes.Count(got, []byte(`"id"`)) != 1 {
		t.Fatalf("tag filter = %s", got)
	}
	alice.mustDo(200, "GET", "/v1/budget/transactions?month="+month+"&account="+checking, nil)

	// Recurring: three months back, monthly; opening the budget records the
	// due ones, and running again records nothing twice.
	start := time.Now().UTC().AddDate(0, -2, 0).Format("2006-01") + "-01"
	rent := post("/v1/budget/recurring", `{"name":"Rent","kind":"expense","amount_cents":90000,"account_id":"`+checking+`","rule":"FREQ=MONTHLY","start_on":"`+start+`","payee":"Landlord"}`)
	alice.mustDo(422, "POST", "/v1/budget/recurring", []byte(`{"name":"Bad","kind":"expense","amount_cents":1,"rule":"FREQ=SECONDLY","start_on":"`+start+`"}`))
	alice.mustDo(200, "GET", "/v1/budget?month="+month, nil)
	var run struct {
		Created int `json:"created"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/budget/recurring/run", nil), &run)
	if run.Created != 0 {
		t.Fatalf("second run created %d", run.Created)
	}
	var count int
	_ = pool.QueryRow(t.Context(), `SELECT count(*) FROM budget_transactions WHERE recurring_id = $1`, rent).Scan(&count)
	if count != 3 {
		t.Fatalf("recurring recorded %d, want 3", count)
	}
	alice.mustDo(200, "GET", "/v1/budget/recurring", nil)
	alice.mustDo(200, "PATCH", "/v1/budget/recurring/"+rent, []byte(`{"amount_cents":95000}`))
	alice.mustDo(204, "DELETE", "/v1/budget/recurring/"+rent, nil)
	_ = pool.QueryRow(t.Context(), `SELECT count(*) FROM budget_transactions WHERE payee = 'Landlord'`).Scan(&count)
	if count != 3 {
		t.Fatal("deleting a series deleted its history")
	}

	// CSV import with ; and decimal commas: rules apply, a re-import skips
	// everything, and identical rows on the same day both count.
	csv := "Date;Amount;Payee\n05/" + month[5:] + "/" + month[:4] + ";-12,50;Carrefour\n05/" + month[5:] + "/" + month[:4] +
		";-12,50;Carrefour\n06/" + month[5:] + "/" + month[:4] + ";1 000,00;Employer\nnot a date;1;x\n"
	spec, _ := json.Marshal(map[string]any{"csv": csv, "account_id": checking, "delimiter": ";", "date_format": "DD/MM/YYYY",
		"has_header": true, "date_column": 0, "amount_column": 1, "payee_column": 2})
	var res struct {
		Created  int      `json:"created"`
		Skipped  int      `json:"skipped"`
		Problems []string `json:"problems"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/budget/import", spec), &res)
	if res.Created != 3 || res.Skipped != 0 || len(res.Problems) != 1 {
		t.Fatalf("import = %+v", res)
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/budget/import", spec), &res)
	if res.Created != 0 || res.Skipped != 3 {
		t.Fatalf("re-import = %+v", res)
	}
	if got := alice.mustDo(200, "GET", "/v1/budget/transactions?month="+month+"&category="+groceries, nil); bytes.Count(got, []byte(`"id"`)) != 3 {
		t.Fatalf("imported rows not categorised: %s", got)
	}

	// Apply rules later to what was recorded before the rule existed.
	post("/v1/budget/transactions", `{"occurred_on":"`+day+`","kind":"expense","amount_cents":999,"payee":"Netflix"}`)
	post("/v1/budget/rules", `{"name":"Streaming","value":"netflix","tag":"subscriptions"}`)
	var applied struct {
		Changed int `json:"changed"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/budget/rules/apply", nil), &applied)
	if applied.Changed != 1 {
		t.Fatalf("apply = %d", applied.Changed)
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/budget/rules/apply", nil), &applied)
	if applied.Changed != 0 {
		t.Fatalf("second apply = %d", applied.Changed)
	}
	var rules struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget/rules", nil), &rules)
	alice.mustDo(200, "PATCH", "/v1/budget/rules/"+rules.Items[0].ID, []byte(`{"active":false}`))
	alice.mustDo(204, "DELETE", "/v1/budget/rules/"+rules.Items[1].ID, nil)

	var rep struct {
		Months        []struct{ Month string } `json:"months"`
		NetWorthCents int64                    `json:"net_worth_cents"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget/report?months=3", nil), &rep)
	b = balances()
	if len(rep.Months) != 3 || rep.Months[2].Month != month || rep.NetWorthCents != b[checking]+b[savings]+b[card] {
		t.Fatalf("report = %+v, balances %v", rep, b)
	}
	alice.mustDo(400, "GET", "/v1/budget/report?months=0", nil)

	// Bob reaches none of it.
	bob.mustDo(404, "PATCH", "/v1/budget/accounts/"+checking, []byte(`{"name":"Mine"}`))
	bob.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"expense","amount_cents":1,"account_id":"`+checking+`"}`))
	if bytes.Contains(bob.mustDo(200, "GET", "/v1/budget/accounts", nil), []byte(checking)) {
		t.Error("bob sees alice's accounts")
	}
	empty := post("/v1/budget/accounts", `{"name":"Unused","type":"asset"}`)
	alice.mustDo(204, "DELETE", "/v1/budget/accounts/"+empty, nil)
}
