package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math/rand/v2"
	"slices"
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
	// One month, chosen: a past one holds none of this month's money (EP-138).
	var past struct {
		Months     []struct{ Month string } `json:"months"`
		Categories []struct{}               `json:"categories"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget/report?months=1&month=2020-01", nil), &past)
	if len(past.Months) != 1 || past.Months[0].Month != "2020-01" || len(past.Categories) != 0 {
		t.Fatalf("a past month = %+v", past)
	}
	alice.mustDo(400, "GET", "/v1/budget/report?month=January", nil)

	// Bob reaches none of it.
	bob.mustDo(404, "PATCH", "/v1/budget/accounts/"+checking, []byte(`{"name":"Mine"}`))
	bob.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"`+day+`","kind":"expense","amount_cents":1,"account_id":"`+checking+`"}`))
	if bytes.Contains(bob.mustDo(200, "GET", "/v1/budget/accounts", nil), []byte(checking)) {
		t.Error("bob sees alice's accounts")
	}
	empty := post("/v1/budget/accounts", `{"name":"Unused","type":"asset"}`)
	alice.mustDo(204, "DELETE", "/v1/budget/accounts/"+empty, nil)
}

// movement is what balances needs from a transaction; From is the account
// (income uses it as the destination) and To the transfer's destination.
type movement struct {
	Kind        string
	AmountCents int64
	From, To    *uuid.UUID
}

// balances applies movements to opening balances: income adds to its
// account, an expense subtracts, a transfer moves money between two accounts.
func balances(opening map[uuid.UUID]int64, moves []movement) map[uuid.UUID]int64 {
	out := make(map[uuid.UUID]int64, len(opening))
	for id, c := range opening {
		out[id] = c
	}
	for _, m := range moves {
		switch m.Kind {
		case "income":
			out[*m.From] += m.AmountCents
		case "expense":
			out[*m.From] -= m.AmountCents
		case "transfer":
			out[*m.From] -= m.AmountCents
			out[*m.To] += m.AmountCents
		}
	}
	return out
}

// The balances the API shows (computed in SQL) agree with the model above, over random income, expenses, and transfers.
func TestAccountBalancesMatchTheModel(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	r := rand.New(rand.NewPCG(7, 69)) //nolint:gosec // deterministic test data, not security
	ids := make([]uuid.UUID, 3)
	opening := map[uuid.UUID]int64{}
	for i := range ids {
		cents := r.Int64N(200_000) - 50_000
		var a struct {
			ID uuid.UUID `json:"id"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/accounts",
			[]byte(fmt.Sprintf(`{"name":"Account %d","type":"asset","opening_cents":%d}`, i, cents))), &a)
		ids[i], opening[a.ID] = a.ID, cents
	}
	var moves []movement
	for range 60 {
		from, to := &ids[r.IntN(3)], &ids[r.IntN(3)]
		m := movement{Kind: []string{"income", "expense", "transfer"}[r.IntN(3)], AmountCents: 1 + r.Int64N(100_000), From: from}
		body := fmt.Sprintf(`{"occurred_on":"2026-10-01","kind":"%s","amount_cents":%d,"account_id":"%s"`, m.Kind, m.AmountCents, *from)
		if m.Kind == "transfer" {
			if *from == *to {
				continue
			}
			m.To = to
			body += fmt.Sprintf(`,"to_account_id":"%s"`, *to)
		}
		alice.mustDo(201, "POST", "/v1/budget/transactions", []byte(body+"}"))
		moves = append(moves, m)
	}
	var list struct {
		Items []struct {
			ID           uuid.UUID `json:"id"`
			BalanceCents int64     `json:"balance_cents"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget/accounts", nil), &list)
	want := balances(opening, moves)
	if len(list.Items) != len(ids) {
		t.Fatalf("got %d accounts", len(list.Items))
	}
	var total, wantTotal int64
	for _, a := range list.Items {
		if a.BalanceCents != want[a.ID] {
			t.Errorf("account %s: API says %d, the model %d", a.ID, a.BalanceCents, want[a.ID])
		}
		total += a.BalanceCents
	}
	// Double entry: a transfer never changes the total, so the balances add
	// up to the openings plus income minus expenses.
	for _, c := range opening {
		wantTotal += c
	}
	for _, m := range moves {
		switch m.Kind {
		case "income":
			wantTotal += m.AmountCents
		case "expense":
			wantTotal -= m.AmountCents
		}
	}
	if total != wantTotal {
		t.Errorf("balances add up to %d, want openings + income - expenses = %d", total, wantTotal)
	}
}

// Double entry, on the production SQL (balanceSQLOver, through the API):
// whatever the transactions, over many random runs with 1 to 5 accounts,
// transfers never change the total, and the total is openings + income −
// expenses.
func TestBalancesReconcile(t *testing.T) {
	e := newEnv(t, true)
	c := e.client()
	c.register("Alice")
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test data, not security
	for run := range 20 {
		ids := make([]uuid.UUID, 1+r.IntN(5))
		var want int64
		for i := range ids {
			cents := r.Int64N(2_000_000) - 1_000_000
			var a struct {
				ID uuid.UUID `json:"id"`
			}
			_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/budget/accounts",
				[]byte(fmt.Sprintf(`{"name":"Run %d account %d","type":"asset","opening_cents":%d}`, run, i, cents))), &a)
			ids[i] = a.ID
			want += cents
		}
		for n := r.IntN(30); n > 0; n-- {
			from, to := ids[r.IntN(len(ids))], ids[r.IntN(len(ids))]
			amount := 1 + r.Int64N(1_000_000)
			kind := []string{"income", "expense", "transfer"}[r.IntN(3)]
			body := fmt.Sprintf(`{"occurred_on":"2026-10-01","kind":"%s","amount_cents":%d,"account_id":"%s"`, kind, amount, from)
			switch kind {
			case "income":
				want += amount
			case "expense":
				want -= amount
			default:
				if from == to {
					continue
				}
				body += fmt.Sprintf(`,"to_account_id":"%s"`, to)
			}
			c.mustDo(201, "POST", "/v1/budget/transactions", []byte(body+"}"))
		}
		var list struct {
			Items []struct {
				ID           uuid.UUID `json:"id"`
				BalanceCents int64     `json:"balance_cents"`
			} `json:"items"`
		}
		_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/budget/accounts", nil), &list)
		var got int64
		seen := 0
		for _, a := range list.Items {
			if slices.Contains(ids, a.ID) {
				got += a.BalanceCents
				seen++
			}
		}
		if seen != len(ids) {
			t.Fatalf("run %d: got %d of the run's %d accounts", run, seen, len(ids))
		}
		if got != want {
			t.Fatalf("run %d: total %d, want %d", run, got, want)
		}
	}
}

// The Budget page counts each account (EP-175): balances at the end of the
// month viewed, net worth, and the bills still to come this month.
func TestTheBudgetPageKnowsEachAccount(t *testing.T) {
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
	now := time.Now().UTC()
	thisMonth := now.Format("2006-01")
	lastMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0).Format("2006-01")
	tomorrow := now.AddDate(0, 0, 1).Format(time.DateOnly)
	daysLeft := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day() - now.Day()

	checking := post("/v1/budget/accounts", `{"name":"Checking","type":"asset","opening_cents":100000}`)
	card := post("/v1/budget/accounts", `{"name":"Visa","type":"liability","role":"credit_card","opening_cents":-5000}`)
	old := post("/v1/budget/accounts", `{"name":"Old","type":"asset","opening_cents":700}`)
	alice.mustDo(200, "PATCH", "/v1/budget/accounts/"+old, []byte(`{"archived":true}`))
	bob.mustDo(201, "POST", "/v1/budget/accounts", []byte(`{"name":"Bob's","type":"asset","opening_cents":999999}`))
	post("/v1/budget/transactions", `{"occurred_on":"`+lastMonth+`-01","kind":"expense","amount_cents":10000,"account_id":"`+checking+`"}`)
	post("/v1/budget/transactions", `{"occurred_on":"`+thisMonth+`-01","kind":"expense","amount_cents":2500,"account_id":"`+card+`"}`)
	// Internet is due every day from tomorrow, as many times as the month
	// has days left; a transfer is not a bill.
	post("/v1/budget/recurring", `{"name":"Internet","kind":"expense","amount_cents":3000,"account_id":"`+checking+`","rule":"FREQ=DAILY","start_on":"`+tomorrow+`"}`)
	post("/v1/budget/recurring", `{"name":"Pay the card","kind":"transfer","amount_cents":100,"account_id":"`+checking+`","to_account_id":"`+card+`","rule":"FREQ=DAILY","start_on":"`+tomorrow+`"}`)

	type month struct {
		Accounts      []accountJSON `json:"accounts"`
		NetWorthCents int64         `json:"net_worth_cents"`
		Upcoming      []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
			On   string `json:"on"`
		} `json:"upcoming"`
		UpcomingExpenseCents int64 `json:"upcoming_expense_cents"`
	}
	get := func(m string) month {
		var out month
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget?month="+m, nil), &out)
		return out
	}
	check := func(got month, want map[string]int64, netWorth int64) {
		t.Helper()
		if len(got.Accounts) != len(want) {
			t.Fatalf("accounts = %+v, want the %d open ones", got.Accounts, len(want))
		}
		for _, a := range got.Accounts {
			if a.BalanceCents != want[a.ID] {
				t.Fatalf("account %s = %d, want %d", a.ID, a.BalanceCents, want[a.ID])
			}
		}
		if got.NetWorthCents != netWorth {
			t.Fatalf("net worth = %d, want %d", got.NetWorthCents, netWorth)
		}
	}
	cur := get(thisMonth)
	check(cur, map[string]int64{checking: 90000, card: -7500}, 90000-7500+700)
	if len(cur.Upcoming) != daysLeft || cur.UpcomingExpenseCents != int64(daysLeft)*3000 {
		t.Fatalf("upcoming = %+v (%d cents), want Internet %d times", cur.Upcoming, cur.UpcomingExpenseCents, daysLeft)
	}
	for _, u := range cur.Upcoming {
		if u.Name != "Internet" || u.Kind != "expense" {
			t.Fatalf("a transfer is not a bill: %+v", u)
		}
	}
	if daysLeft > 0 && cur.Upcoming[0].On != tomorrow {
		t.Fatalf("first due %s, want %s", cur.Upcoming[0].On, tomorrow)
	}
	// At the end of last month, this month's card expense had not happened,
	// and nothing is "to come" for a month that is over.
	past := get(lastMonth)
	check(past, map[string]int64{checking: 90000, card: -5000}, 90000-5000+700)
	if len(past.Upcoming) != 0 {
		t.Fatalf("a past month has upcoming %+v", past.Upcoming)
	}
}
