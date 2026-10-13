package agents

import (
	"strings"
	"testing"
)

func TestEstimateIsReadAndBounded(t *testing.T) {
	text := "Here you go:\n```json\n" + `{"items":[
		{"name":"Spaghetti bolognese","grams":350.26,"kcal":520,"protein_g":24,"carbs_g":68,"fat_g":16},
		{"name":"","grams":50,"kcal":10},
		{"name":"Water","grams":250,"kcal":0,"protein_g":0,"carbs_g":0,"fat_g":0},
		{"name":"Mystery","grams":-3,"kcal":100},
		{"name":"Cake","grams":120,"kcal":99999,"protein_g":5,"carbs_g":50,"fat_g":20}
	],"note":"Portion sizes are estimates."}` + "\n```"
	e, err := parseEstimate(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Items) != 2 || e.Items[0].Name != "Spaghetti bolognese" || e.Items[0].Grams != 350.3 || e.Items[1].Name != "Water" {
		t.Fatalf("items = %+v", e.Items)
	}
	if e.Note != "Portion sizes are estimates." {
		t.Fatalf("note = %q", e.Note)
	}
}

func TestEstimateWithoutFoodOrJSON(t *testing.T) {
	if e, err := parseEstimate(`{"items":[],"note":"This is a photo of a cat."}`); err != nil || len(e.Items) != 0 || e.Items == nil {
		t.Fatalf("no food = %+v, %v", e, err)
	}
	if _, err := parseEstimate("I cannot help with that."); err == nil {
		t.Fatal("prose accepted")
	}
	long := `{"items":[` + strings.Repeat(`{"name":"Pea","grams":1,"kcal":1},`, 20) + `{"name":"Pea","grams":1,"kcal":1}]}`
	if e, _ := parseEstimate(long); len(e.Items) != maxEstimateItems {
		t.Fatalf("%d items", len(e.Items))
	}
}
