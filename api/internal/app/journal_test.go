package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The journal (EP-91): health data behind its consent, one entry a day,
// never for the future, exported, and deleted with the consent.
func TestJournal(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	today := time.Now().UTC().Format(time.DateOnly) // a new account's zone is UTC here
	alice.mustDo(403, "PUT", "/v1/journal/"+today, []byte(`{"mood":4}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	var entry struct {
		Day        string   `json:"day"`
		Mood       *int     `json:"mood"`
		SleepHours *float64 `json:"sleep_hours"`
		Gratitude  []string `json:"gratitude"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "PUT", "/v1/journal/"+today,
		[]byte(`{"mood":4,"energy":3,"sleep_hours":7.5,"gratitude":["Coffee with Sam",""],"note":"Good."}`)), &entry)
	if entry.Day != today || *entry.Mood != 4 || *entry.SleepHours != 7.5 || len(entry.Gratitude) != 1 {
		t.Fatalf("entry = %+v", entry)
	}
	// Writing again replaces the day.
	alice.mustDo(200, "PUT", "/v1/journal/"+today, []byte(`{"mood":2}`))
	if out := alice.mustDo(200, "GET", "/v1/journal/"+today, nil); !strings.Contains(string(out), `"mood":2`) || !strings.Contains(string(out), `"energy":null`) {
		t.Fatalf("replaced = %s", out)
	}
	tomorrow := time.Now().UTC().AddDate(0, 0, 2).Format(time.DateOnly)
	alice.mustDo(422, "PUT", "/v1/journal/"+tomorrow, []byte(`{"mood":3}`))
	alice.mustDo(422, "PUT", "/v1/journal/"+today, []byte(`{"mood":7}`))
	alice.mustDo(422, "PUT", "/v1/journal/"+today, []byte(`{}`))
	if out := alice.mustDo(200, "GET", "/v1/journal", nil); !strings.Contains(string(out), today) {
		t.Fatalf("list = %s", out)
	}
	bob.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	bob.mustDo(404, "GET", "/v1/journal/"+today, nil)
	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			r, _ := f.Open()
			_ = json.NewDecoder(r).Decode(&data)
		}
	}
	if !strings.Contains(string(data["journal"]), today) {
		t.Fatalf("the journal is not in the export: %s", data["journal"])
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":false}`))
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	alice.mustDo(404, "GET", "/v1/journal/"+today, nil)
	alice.mustDo(404, "DELETE", "/v1/journal/"+today, nil)
}

// What helps you (EP-92): a pattern written into 20 days is found, explained,
// and in French when asked.
func TestWhatHelps(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	if out := alice.mustDo(200, "GET", "/v1/journal/helps", nil); !strings.Contains(string(out), `"needed":14`) {
		t.Fatalf("empty = %s", out)
	}
	for i := range 20 {
		day := time.Now().UTC().AddDate(0, 0, -i).Format(time.DateOnly)
		sleep, mood := 6.0, 2
		if i%2 == 0 {
			sleep, mood = 8, 4
		}
		if i%7 == 3 {
			mood = 3
		}
		alice.mustDo(200, "PUT", "/v1/journal/"+day, []byte(fmt.Sprintf(`{"mood":%d,"sleep_hours":%v}`, mood, sleep)))
	}
	var h struct {
		Days     int `json:"days"`
		Findings []struct {
			Factor, Outcome, Sentence string
		} `json:"findings"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/journal/helps", nil), &h)
	if h.Days != 20 || len(h.Findings) == 0 || h.Findings[0].Factor != "sleep" || h.Findings[0].Outcome != "mood" ||
		!strings.Contains(h.Findings[0].Sentence, "slept 7 h or more") {
		t.Fatalf("helps = %+v", h)
	}
	if code, out := alice.do("GET", "/v1/journal/helps", nil, "Accept-Language", "fr"); code != 200 || !strings.Contains(string(out), "avez dormi 7 h ou plus") {
		t.Fatalf("fr = %d %s", code, out)
	}
}
