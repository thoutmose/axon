package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/health"
)

func useFakeFoods(t *testing.T) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch {
		case r.URL.Path == "/search":
			_, _ = w.Write([]byte(`{"hits":[{"code":"0894700010137","product_name":"Nonfat Greek Yogurt","brands":["Chobani"],"serving_quantity":"170",
				"nutriments":{"energy-kcal_100g":52.9,"proteins_100g":9.41,"carbohydrates_100g":3.53,"fat_100g":0}},
				{"code":"1","product_name":"No energy","nutriments":{}}]}`))
		case r.URL.Path == "/api/v2/product/3017620422003":
			_, _ = w.Write([]byte(`{"status":1,"product":{"code":"3017620422003","product_name":"Nutella","brands":"Nutella, Ferrero",
				"nutriments":{"energy-kcal_100g":539,"proteins_100g":6.3,"carbohydrates_100g":57.5,"fat_100g":30.9},"serving_quantity":15}}`))
		case r.URL.Path == "/fdc/foods/search" && r.URL.Query().Get("api_key") == "usda-key":
			_, _ = w.Write([]byte(`{"foods":[{"fdcId":171705,"description":"Banana, raw","foodNutrients":[
				{"nutrientNumber":"208","unitName":"KCAL","value":89},{"nutrientNumber":"203","unitName":"G","value":1.09},
				{"nutrientNumber":"205","unitName":"G","value":22.8},{"nutrientNumber":"204","unitName":"G","value":0.33}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	saved := health.DefaultEndpoints
	health.DefaultEndpoints = health.Endpoints{OFFSearch: srv.URL, OFF: srv.URL, USDA: srv.URL + "/fdc"}
	t.Cleanup(func() { health.DefaultEndpoints = saved })
	for _, table := range []string{"food_lookups", "foods"} {
		if _, err := pool.Exec(context.Background(), "DELETE FROM "+table); err != nil {
			t.Fatal(err)
		}
	}
	return &calls
}

type foodDayJSON struct {
	Entries []struct {
		ID    string   `json:"id"`
		Name  string   `json:"name"`
		Kcal  *float64 `json:"kcal"`
		Grams *float64 `json:"grams"`
	} `json:"entries"`
	Totals struct {
		Kcal    float64 `json:"kcal"`
		Protein float64 `json:"protein_g"`
	} `json:"totals"`
	Week []struct {
		Date string  `json:"date"`
		Kcal float64 `json:"kcal"`
	} `json:"week"`
}

// Nothing health-related is stored or read without consent; food lookups
// have their own consent; typed-in entries work with every catalogue down;
// withdrawing the consent deletes everything.
func TestHealthNeedsConsentAndLogsFoodAndTraining(t *testing.T) {
	calls := useFakeFoods(t)
	t.Setenv("USDA_API_KEY", "usda-key")
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	if out := alice.mustDo(200, "GET", "/v1/health/status", nil); !bytes.Contains(out, []byte(`"consented":false`)) || !bytes.Contains(out, []byte(`"usda":true`)) {
		t.Fatalf("status = %s", out)
	}
	alice.mustDo(403, "GET", "/v1/health/food", nil)
	alice.mustDo(403, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-11","meal":"lunch","name":"Soup","kcal":200}`))
	alice.mustDo(403, "GET", "/v1/health/workouts", nil)
	alice.mustDo(403, "GET", "/v1/health/foods/search?q=yogurt", nil)
	var stored int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM food_entries`).Scan(&stored)
	if stored != 0 || calls.Load() != 0 {
		t.Fatalf("stored %d entries and made %d lookups without consent", stored, calls.Load())
	}

	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	// Typed in: no catalogue involved.
	alice.mustDo(201, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-11","meal":"lunch","name":"Lentil soup","kcal":320.5,"protein_g":18}`))
	alice.mustDo(422, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-11","meal":"brunch","name":"x"}`))
	alice.mustDo(422, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-11","meal":"lunch","name":"x","kcal":-1}`))
	alice.mustDo(403, "GET", "/v1/health/foods/search?q=yogurt", nil) // lookups have their own consent

	alice.mustDo(204, "PUT", "/v1/me/consents/food_lookups", []byte(`{"granted":true}`))
	var found struct {
		Items []struct {
			ID       string   `json:"id"`
			Name     string   `json:"name"`
			Brand    string   `json:"brand"`
			ServingG *float64 `json:"serving_g"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/foods/search?q=Greek+yogurt", nil), &found)
	if len(found.Items) != 1 || found.Items[0].Brand != "Chobani" || found.Items[0].ServingG == nil || *found.Items[0].ServingG != 170 {
		t.Fatalf("search = %+v", found)
	}
	alice.mustDo(200, "GET", "/v1/health/foods/search?q=greek%20yogurt", nil)
	if calls.Load() != 1 {
		t.Fatalf("catalogue calls = %d, want 1 (cached)", calls.Load())
	}
	// From a food: 200 g of yogurt is 105.8 kcal.
	alice.mustDo(201, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-11","meal":"breakfast","food_id":"`+found.Items[0].ID+`","grams":200}`))
	if out := alice.mustDo(200, "GET", "/v1/health/foods/barcode/3017620422003", nil); !bytes.Contains(out, []byte(`"brand":"Nutella"`)) {
		t.Fatalf("barcode = %s", out)
	}
	alice.mustDo(422, "GET", "/v1/health/foods/barcode/12ab", nil)
	if out := alice.mustDo(200, "GET", "/v1/health/foods/search?q=banana&source=usda", nil); !bytes.Contains(out, []byte(`"kcal":89`)) {
		t.Fatalf("usda = %s", out)
	}

	var day foodDayJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/food?date=2026-10-11", nil), &day)
	if len(day.Entries) != 2 || !strings.HasPrefix(day.Entries[0].Name, "Nonfat Greek Yogurt (Chobani)") || day.Totals.Kcal != 426.3 ||
		len(day.Week) != 7 || day.Week[6].Date != "2026-10-11" || day.Week[6].Kcal != 426.3 {
		t.Fatalf("day = %+v", day)
	}

	// Training.
	var wk struct {
		ID   string `json:"id"`
		Sets []struct {
			Exercise string   `json:"exercise"`
			WeightKg *float64 `json:"weight_kg"`
		} `json:"sets"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/health/workouts", []byte(`{"done_on":"2026-10-10","kind":"strength","title":"Full body","minutes":45,"effort":7,
		"sets":[{"exercise":"Squat","reps":5,"weight_kg":60},{"exercise":"Push-up","reps":12}]}`)), &wk)
	if len(wk.Sets) != 2 || *wk.Sets[0].WeightKg != 60 {
		t.Fatalf("workout = %+v", wk)
	}
	alice.mustDo(422, "POST", "/v1/health/workouts", []byte(`{"done_on":"2026-10-10","kind":"strength","title":"x","effort":11}`))
	alice.mustDo(200, "PUT", "/v1/health/workouts/"+wk.ID, []byte(`{"done_on":"2026-10-10","kind":"cardio","title":"Run","minutes":30,"sets":[{"exercise":"Run","distance_km":5,"seconds":1650}]}`))
	if out := alice.mustDo(200, "GET", "/v1/health/workouts?from=2026-10-01&to=2026-10-31", nil); !bytes.Contains(out, []byte(`"title":"Run"`)) || !bytes.Contains(out, []byte(`"distance_km":5`)) {
		t.Fatalf("workouts = %s", out)
	}
	alice.mustDo(400, "GET", "/v1/health/workouts?from=2026-10-31&to=2026-10-01", nil)
	// A device a day ahead of the account's time zone logs "today" as
	// tomorrow: the workout still shows in the default list.
	ahead := time.Now().UTC().AddDate(0, 0, 1).Format(time.DateOnly)
	alice.mustDo(201, "POST", "/v1/health/workouts", []byte(`{"done_on":"`+ahead+`","kind":"cardio","title":"Swim from abroad"}`))
	if out := alice.mustDo(200, "GET", "/v1/health/workouts", nil); !bytes.Contains(out, []byte("Swim from abroad")) {
		t.Fatalf("a workout dated tomorrow is missing: %s", out)
	}

	// Private, and out of search and the graph.
	bob.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	bob.mustDo(404, "PUT", "/v1/health/workouts/"+wk.ID, []byte(`{"done_on":"2026-10-10","kind":"cardio","title":"Mine"}`))
	bob.mustDo(404, "DELETE", "/v1/health/food/"+day.Entries[1].ID, nil)
	if out := bob.mustDo(200, "GET", "/v1/health/food?date=2026-10-11", nil); bytes.Contains(out, []byte("Lentil")) {
		t.Fatal("bob sees alice's food")
	}
	if out := alice.mustDo(200, "GET", "/v1/search?q=lentil", nil); bytes.Contains(out, []byte("Lentil")) {
		t.Fatal("health data is in search")
	}
	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	var exported []byte
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			r, _ := f.Open()
			exported, _ = io.ReadAll(r)
		}
	}
	if !bytes.Contains(exported, []byte("Lentil soup")) || !bytes.Contains(exported, []byte(`"workouts"`)) {
		t.Fatal("health data missing from the export")
	}

	alice.mustDo(204, "DELETE", "/v1/health/food/"+day.Entries[1].ID, nil)
	alice.mustDo(204, "DELETE", "/v1/health/workouts/"+wk.ID, nil)
	alice.mustDo(201, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-12","meal":"snack","name":"Apple","kcal":80}`))

	// Withdrawing the consent deletes the data and closes the endpoints.
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":false}`))
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM food_entries f JOIN users u ON u.id = f.user_id WHERE u.display_name = 'Alice'`).Scan(&stored)
	if stored != 0 {
		t.Fatalf("%d entries kept after withdrawal", stored)
	}
	alice.mustDo(403, "GET", "/v1/health/food", nil)
	bob.mustDo(201, "POST", "/v1/health/food", []byte(`{"eaten_on":"2026-10-12","meal":"snack","name":"Pear","kcal":60}`))
	bob.mustDo(204, "DELETE", "/v1/health/data", nil)
	if out := bob.mustDo(200, "GET", "/v1/health/food?date=2026-10-12", nil); bytes.Contains(out, []byte("Pear")) {
		t.Fatal("delete all kept data")
	}
}
