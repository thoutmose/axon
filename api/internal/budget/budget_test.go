package budget

import "testing"

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
