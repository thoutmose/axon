package budget

import (
	"math/rand/v2"
	"testing"

	"github.com/google/uuid"
)

// movement is what balances needs from a transaction.
type movement struct {
	Kind        string
	AmountCents int64
	From, To    *uuid.UUID // From: account_id (income uses it as the destination); To: to_account_id
}

// balances is the model of balanceSQLOver: income adds to its account, an
// expense subtracts, a transfer moves money between two accounts. The API
// test TestAccountBalancesMatchTheModel checks the SQL against the same rules.
func balances(opening map[uuid.UUID]int64, moves []movement) map[uuid.UUID]int64 {
	out := make(map[uuid.UUID]int64, len(opening))
	for id, c := range opening {
		out[id] = c
	}
	for _, m := range moves {
		if m.From == nil {
			continue
		}
		switch m.Kind {
		case "income":
			out[*m.From] += m.AmountCents
		case "expense":
			out[*m.From] -= m.AmountCents
		case "transfer":
			if m.To != nil {
				out[*m.From] -= m.AmountCents
				out[*m.To] += m.AmountCents
			}
		}
	}
	return out
}

// Double entry: whatever the transactions, transfers never change the
// total, and the total is openings + income − expenses.
func TestBalancesReconcile(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test data, not security
	for run := 0; run < 500; run++ {
		ids := make([]uuid.UUID, 1+r.IntN(5))
		opening := map[uuid.UUID]int64{}
		var want int64
		for i := range ids {
			ids[i] = uuid.New()
			opening[ids[i]] = r.Int64N(2_000_000) - 1_000_000
			want += opening[ids[i]]
		}
		var moves []movement
		for n := r.IntN(60); n > 0; n-- {
			from, to := &ids[r.IntN(len(ids))], &ids[r.IntN(len(ids))]
			amount := 1 + r.Int64N(MaxCents/1000)
			switch kind := []string{"income", "expense", "transfer"}[r.IntN(3)]; kind {
			case "income":
				want += amount
				moves = append(moves, movement{Kind: kind, AmountCents: amount, From: from})
			case "expense":
				want -= amount
				moves = append(moves, movement{Kind: kind, AmountCents: amount, From: from})
			default:
				if *from == *to {
					continue
				}
				moves = append(moves, movement{Kind: kind, AmountCents: amount, From: from, To: to})
			}
		}
		var got int64
		for _, c := range balances(opening, moves) {
			got += c
		}
		if got != want {
			t.Fatalf("run %d: total %d, want %d", run, got, want)
		}
	}
}

func TestParseCents(t *testing.T) {
	for in, want := range map[string]int64{
		"12": 1200, "-12.50": -1250, "1 234,56": 123456, "1.234,56": 123456, "1,234.56": 123456,
		"(12.00)": -1200, "€ 5": 500, "12.50-": -1250, "+3,5": 350, "0.29": 29, "1.005": 100500, "-0,01": -1,
	} {
		got, err := ParseCents(in)
		if err != nil || got != want {
			t.Errorf("ParseCents(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "--", "12345678901"} {
		if _, err := ParseCents(in); err == nil {
			t.Errorf("ParseCents(%q) accepted", in)
		}
	}
}

func TestRuleMatches(t *testing.T) {
	r := Rule{Field: "payee", Operator: "contains", Value: "Carrefour"}
	if !r.Matches("CARREFOUR MARKET 123", "") || r.Matches("Lidl", "carrefour") {
		t.Error("contains on payee")
	}
	r = Rule{Field: "note", Operator: "starts_with", Value: "rent"}
	if !r.Matches("", "Rent October") || r.Matches("", "Car rent") {
		t.Error("starts_with on note")
	}
	r = Rule{Field: "payee", Operator: "equals", Value: "Netflix"}
	if !r.Matches(" netflix ", "") || r.Matches("Netflix.com", "") {
		t.Error("equals")
	}
}

func TestCleanTags(t *testing.T) {
	got, err := cleanTags([]string{"#Food", "food", " travel ", ""})
	if err != nil || len(got) != 2 || got[0] != "food" || got[1] != "travel" {
		t.Fatalf("cleanTags = %v, %v", got, err)
	}
	if _, err := cleanTags([]string{"no spaces allowed"}); err == nil {
		t.Error("accepted a tag with spaces")
	}
}
