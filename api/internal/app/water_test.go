package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

type waterJSON struct {
	Date     string `json:"date"`
	TotalML  int    `json:"total_ml"`
	TargetML *int   `json:"target_ml"`
	Entries  []struct {
		ID string `json:"id"`
		ML int    `json:"ml"`
	} `json:"entries"`
	Week []struct {
		Date    string `json:"date"`
		TotalML int    `json:"total_ml"`
	} `json:"week"`
}

// Water drunk in a day (EP-101): health data under its consent, with the
// drinks' share of the daily need as a target when the profile gives one,
// never someone else's, gone with the consent.
func TestWaterTracker(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	alice.mustDo(403, "POST", "/v1/health/water", []byte(`{"ml":250}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))

	var glass struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/health/water", []byte(`{"ml":250}`)), &glass)
	alice.mustDo(201, "POST", "/v1/health/water", []byte(`{"ml":500}`))
	for _, bad := range []string{`{"ml":0}`, `{"ml":3001}`, `{"ml":250,"drunk_on":"2999-01-01"}`, `{"ml":250,"drunk_on":"yesterday"}`} {
		alice.mustDo(422, "POST", "/v1/health/water", []byte(bad))
	}
	var d waterJSON
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/water", nil), &d)
	if d.TotalML != 750 || len(d.Entries) != 2 || d.TargetML != nil || len(d.Week) != 7 || d.Week[6].TotalML != 750 {
		t.Fatalf("day = %+v", d)
	}
	// With a profile, the target is the drinks' share of the daily need: 80 % of 2.0 L.
	alice.mustDo(200, "PUT", "/v1/health/profile", []byte(`{"birth_year":1990,"sex":"female","height_cm":168,"weight_kg":62,"activity":"moderate"}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/water", nil), &d)
	if d.TargetML == nil || *d.TargetML != 1600 {
		t.Fatalf("target = %v", d.TargetML)
	}
	// Yesterday's glass counts for yesterday.
	yesterday := time.Now().UTC().AddDate(0, 0, -2).Format(time.DateOnly)
	alice.mustDo(201, "POST", "/v1/health/water", []byte(`{"ml":330,"drunk_on":"`+yesterday+`"}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/water?date="+yesterday, nil), &d)
	if d.TotalML != 330 {
		t.Fatalf("%s = %+v", yesterday, d)
	}

	bob.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	bob.mustDo(404, "DELETE", "/v1/health/water/"+glass.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/health/water/"+glass.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/health/water/"+glass.ID, nil)

	// In the export, and gone with the consent.
	out := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			found = bytes.Contains(b, []byte(`"water_entries"`)) && bytes.Contains(bytes.ReplaceAll(b, []byte(" "), nil), []byte(`"ml":500`))
		}
	}
	if !found {
		t.Error("water is not in the export")
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":false}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/health/water", nil), &d)
	if d.TotalML != 0 || len(d.Entries) != 0 {
		t.Fatalf("after withdrawal = %+v", d)
	}
}
