package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

type caffeineJSON struct {
	TotalMG     int        `json:"total_mg"`
	LimitMG     int        `json:"limit_mg"`
	NowMG       float64    `json:"now_mg"`
	AtBedtimeMG float64    `json:"at_bedtime_mg"`
	Bedtime     time.Time  `json:"bedtime"`
	Cutoff      *time.Time `json:"cutoff"`
	HalfLife    float64    `json:"half_life_hours"`
	Entries     []struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
		MG   int    `json:"mg"`
	} `json:"entries"`
	Week []struct {
		TotalMG int `json:"total_mg"`
	} `json:"week"`
	Drinks []struct {
		Kind string `json:"kind"`
		MG   int    `json:"mg"`
	} `json:"drinks"`
}

// Coffee and tea (EP-117): caffeine against the day's limit and the night
// after it, under the health consent, never someone else's, gone with it.
func TestCaffeineTracker(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	alice.mustDo(403, "POST", "/v1/health/caffeine", []byte(`{"kind":"coffee"}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))

	var cup struct {
		ID string `json:"id"`
		MG int    `json:"mg"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/health/caffeine", []byte(`{"kind":"coffee"}`)), &cup)
	if cup.MG != 95 {
		t.Fatalf("a coffee = %+v", cup)
	}
	alice.mustDo(201, "POST", "/v1/health/caffeine", []byte(`{"kind":"other","mg":63}`))
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	for _, bad := range []string{`{"kind":"other"}`, `{"kind":"coffee","mg":2000}`, `{"kind":"wine"}`, `{"kind":"tea","at":"` + future + `"}`} {
		alice.mustDo(422, "POST", "/v1/health/caffeine", []byte(bad))
	}

	var d caffeineJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/caffeine", nil), &d)
	if d.TotalMG != 158 || d.LimitMG != 400 || len(d.Entries) != 2 || len(d.Week) != 7 || d.Week[6].TotalMG != 158 || len(d.Drinks) < 9 {
		t.Fatalf("day = %+v", d)
	}
	// It decays toward bedtime; the cut-off, if any, comes before bed.
	if d.HalfLife != 5 || d.NowMG < 150 || d.AtBedtimeMG >= d.NowMG || (d.Cutoff != nil && !d.Cutoff.Before(d.Bedtime)) {
		t.Fatalf("decay = %+v", d)
	}

	// A slower metabolism, and pregnancy, change the numbers.
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc["caffeine"] = map[string]any{"bedtime": "22:30", "metabolism": "slow"}
	body, _ := json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)
	alice.mustDo(200, "PUT", "/v1/health/profile", []byte(`{"birth_year":1990,"sex":"female","height_cm":168,"weight_kg":62,"activity":"moderate","pregnant_or_breastfeeding":true}`))
	var slow caffeineJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/caffeine", nil), &slow)
	if slow.HalfLife != 7 || slow.LimitMG != 200 || slow.AtBedtimeMG < d.AtBedtimeMG-1 {
		t.Fatalf("slow = %+v", slow)
	}

	// Never Bob's; in the export; gone with the consent.
	bob.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	bob.mustDo(404, "DELETE", "/v1/health/caffeine/"+cup.ID, nil)
	if ex := exportedJSON(t, alice); !bytes.Contains(ex, []byte(`"caffeine_entries"`)) || !bytes.Contains(ex, []byte(`"coffee"`)) {
		t.Error("caffeine is not in the export")
	}
	alice.mustDo(204, "DELETE", "/v1/health/caffeine/"+cup.ID, nil)
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":false}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/caffeine", nil), &d)
	if d.TotalMG != 0 || len(d.Entries) != 0 {
		t.Fatalf("after withdrawal = %+v", d)
	}
}
