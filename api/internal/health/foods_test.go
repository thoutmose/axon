package health

import (
	"encoding/json"
	"testing"
)

func TestOpenFoodFactsParsing(t *testing.T) {
	var p offProduct
	_ = json.Unmarshal([]byte(`{"code":"123","product_name":" Oat drink ","brands":"Oatly, Other","serving_quantity":"250",
		"nutriments":{"energy-kcal_100g":"46","proteins_100g":1,"carbohydrates_100g":6.7,"fat_100g":1.5,"fiber_100g":5000}}`), &p)
	f, ok := p.food()
	if !ok || f.Name != "Oat drink" || f.Brand != "Oatly" || *f.Kcal != 46 || *f.ServingG != 250 || f.Fiber != nil {
		t.Fatalf("food = %+v, %v", f, ok)
	}
	p = offProduct{}
	_ = json.Unmarshal([]byte(`{"code":"9","product_name":"x","brands":["A","B"],"nutriments":{}}`), &p)
	if _, ok := p.food(); ok {
		t.Fatal("a food without energy was kept")
	}
	if brandOf(json.RawMessage(`["Chobani"]`)) != "Chobani" || brandOf(nil) != "" {
		t.Fatal("brandOf")
	}
}
