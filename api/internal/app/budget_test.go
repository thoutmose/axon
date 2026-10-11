package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type budgetMonthJSON struct {
	Currency      string `json:"currency"`
	IncomeCents   int64  `json:"income_cents"`
	ExpenseCents  int64  `json:"expense_cents"`
	BudgetedCents int64  `json:"budgeted_cents"`
	UncatIncome   int64  `json:"uncategorised_income_cents"`
	UncatExpense  int64  `json:"uncategorised_expense_cents"`
	Lines         []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		ActualCents int64  `json:"actual_cents"`
	} `json:"lines"`
}

// The budget adds up exactly: category lines plus uncategorised amounts are
// the month's totals, amounts are integer cents, months do not leak into
// each other, and another person sees none of it.
func TestBudgetReconcilesAndIsPrivate(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	cat := func(body string) string {
		var c struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/categories", []byte(body)), &c)
		return c.ID
	}
	salary := cat(`{"name":"Salary","kind":"income"}`)
	food := cat(`{"name":"Groceries","kind":"expense","monthly_cents":40000}`)
	rent := cat(`{"name":"Rent","kind":"expense","monthly_cents":95000}`)
	alice.mustDo(422, "POST", "/v1/budget/categories", []byte(`{"name":"Rent","kind":"expense"}`)) // duplicate
	alice.mustDo(422, "POST", "/v1/budget/categories", []byte(`{"name":"x","kind":"gift"}`))
	alice.mustDo(422, "POST", "/v1/budget/categories", []byte(`{"name":"x","kind":"expense","monthly_cents":-1}`))

	tx := func(body string) string {
		var tr struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/budget/transactions", []byte(body)), &tr)
		return tr.ID
	}
	tx(`{"occurred_on":"2026-10-01","kind":"income","amount_cents":250000,"category_id":"` + salary + `","payee":"Employer"}`)
	tx(`{"occurred_on":"2026-10-02","kind":"expense","amount_cents":95000,"category_id":"` + rent + `"}`)
	tx(`{"occurred_on":"2026-10-05","kind":"expense","amount_cents":1999,"category_id":"` + food + `"}`)
	tx(`{"occurred_on":"2026-10-06","kind":"expense","amount_cents":1}`) // uncategorised
	coffee := tx(`{"occurred_on":"2026-10-31","kind":"expense","amount_cents":350,"category_id":"` + food + `"}`)
	tx(`{"occurred_on":"2026-11-01","kind":"expense","amount_cents":7777,"category_id":"` + food + `"}`) // next month
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"2026-10-01","kind":"expense","amount_cents":0}`))
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"2026-10-01","kind":"expense","amount_cents":1,"category_id":"`+salary+`"}`))
	alice.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"2026-10-01","kind":"expense","amount_cents":1.5}`))
	alice.mustDo(422, "GET", "/v1/budget?month=2026-13", nil)

	var m budgetMonthJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget?month=2026-10", nil), &m)
	if m.Currency != "EUR" || m.IncomeCents != 250000 || m.ExpenseCents != 95000+1999+1+350 || m.BudgetedCents != 135000 || m.UncatExpense != 1 {
		t.Fatalf("month = %+v", m)
	}
	var income, expense int64
	actual := map[string]int64{}
	for _, l := range m.Lines {
		actual[l.Name] = l.ActualCents
		if l.Kind == "income" {
			income += l.ActualCents
		} else {
			expense += l.ActualCents
		}
	}
	if income+m.UncatIncome != m.IncomeCents || expense+m.UncatExpense != m.ExpenseCents || actual["Groceries"] != 2349 {
		t.Fatalf("lines do not add up: %v", actual)
	}

	// Edits and deletes keep the books consistent.
	alice.mustDo(200, "PATCH", "/v1/budget/transactions/"+coffee, []byte(`{"amount_cents":400,"payee":"Café"}`))
	alice.mustDo(200, "PATCH", "/v1/budget/categories/"+food, []byte(`{"monthly_cents":null,"name":"Food"}`))
	alice.mustDo(422, "PATCH", "/v1/budget/categories/"+food, []byte(`{"kind":"income"}`))
	alice.mustDo(204, "DELETE", "/v1/budget/categories/"+rent, nil) // its transaction becomes uncategorised
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/budget?month=2026-10", nil), &m)
	if m.ExpenseCents != 95000+1999+1+400 || m.UncatExpense != 95001 || m.BudgetedCents != 0 {
		t.Fatalf("after edits = %+v", m)
	}
	if got := alice.mustDo(200, "GET", "/v1/budget/transactions?month=2026-10&category="+food, nil); bytes.Count(got, []byte(`"id"`)) != 2 {
		t.Fatalf("food transactions = %s", got)
	}
	alice.mustDo(200, "GET", "/v1/budget/categories", nil)
	alice.mustDo(200, "PUT", "/v1/budget/currency", []byte(`{"currency":"usd"}`))
	alice.mustDo(422, "PUT", "/v1/budget/currency", []byte(`{"currency":"dollars"}`))

	// Bob sees nothing and can change nothing.
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/budget?month=2026-10", nil), &m)
	if m.IncomeCents != 0 || len(m.Lines) != 0 || m.Currency != "EUR" {
		t.Fatalf("bob sees %+v", m)
	}
	bob.mustDo(404, "PATCH", "/v1/budget/transactions/"+coffee, []byte(`{"amount_cents":1}`))
	bob.mustDo(404, "DELETE", "/v1/budget/categories/"+food, nil)
	bob.mustDo(422, "POST", "/v1/budget/transactions", []byte(`{"occurred_on":"2026-10-01","kind":"expense","amount_cents":5,"category_id":"`+food+`"}`))
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/search?q=Caf%C3%A9", nil), []byte("Café")) {
		t.Error("finance data reached search")
	}
	alice.mustDo(204, "DELETE", "/v1/budget/transactions/"+coffee, nil)
	alice.mustDo(404, "DELETE", "/v1/budget/transactions/"+coffee, nil)
}
