package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/calendars"
)

// Calendars you follow: fetched at once, expanded, shown by date, refreshed,
// owned, private addresses refused unless the operator allows them.
func TestFollowedCalendars(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	// Without the operator's permission, private addresses are refused even
	// through a public-looking name; the real client is used here.
	alice.mustDo(422, "POST", "/v1/calendar/sources", []byte(`{"name":"LAN","url":"https://127.0.0.1:1/cal.ics"}`))
	alice.mustDo(422, "POST", "/v1/calendar/sources", []byte(`{"name":"Plain","url":"http://calendar.example/cal.ics"}`))

	today := time.Now().UTC()
	day := func(d int) string { return today.AddDate(0, 0, d).Format("20060102") }
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:a\r\nDTSTART:" + day(1) + "T090000Z\r\nDTEND:" + day(1) + "T100000Z\r\nSUMMARY:Dentist\r\nLOCATION:Rue Haute\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:b\r\nDTSTART;VALUE=DATE:" + day(2) + "\r\nDTEND;VALUE=DATE:" + day(4) + "\r\nSUMMARY:Conference\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:c\r\nDTSTART:" + day(0) + "T070000Z\r\nDTEND:" + day(0) + "T073000Z\r\nRRULE:FREQ=DAILY;COUNT=5\r\nSUMMARY:Run\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	served := ics
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/secret/basic.ics":
			w.Header().Set("Content-Type", "text/calendar")
			_, _ = fmt.Fprint(w, served)
		case "/html":
			_, _ = fmt.Fprint(w, "<html>not a calendar</html>")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AXON_CALENDAR_PRIVATE", "1")
	client := calendars.Client
	calendars.Client = srv.Client()
	t.Cleanup(func() { calendars.Client = client })

	alice.mustDo(422, "POST", "/v1/calendar/sources", []byte(`{"name":"Not one","url":"`+srv.URL+`/html"}`))
	alice.mustDo(422, "POST", "/v1/calendar/sources", []byte(`{"name":"Missing","url":"`+srv.URL+`/gone.ics"}`))
	var src struct {
		ID     string `json:"id"`
		Host   string `json:"host"`
		Events int    `json:"events"`
		Error  string `json:"error"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/calendar/sources", []byte(`{"name":"Personal","url":"`+srv.URL+`/secret/basic.ics","color":"#22aa66"}`)), &src)
	if src.Events != 7 || src.Host != "127.0.0.1" {
		t.Fatalf("source = %+v", src)
	}
	if out := alice.mustDo(200, "GET", "/v1/calendar/sources", nil); strings.Contains(string(out), "secret") {
		t.Fatalf("the secret address leaked: %s", out)
	}

	var evs struct {
		Items []struct {
			Title    string `json:"title"`
			Location string `json:"location"`
			Date     string `json:"date"`
			LastDate string `json:"last_date"`
			AllDay   bool   `json:"all_day"`
		} `json:"items"`
	}
	q := "/v1/calendar/events?from=" + today.Format(time.DateOnly) + "&to=" + today.AddDate(0, 0, 10).Format(time.DateOnly)
	_ = json.Unmarshal(alice.mustDo(200, "GET", q, nil), &evs)
	got := map[string]int{}
	for _, ev := range evs.Items {
		got[ev.Title]++
		if ev.Title == "Conference" && (!ev.AllDay || ev.Date != today.AddDate(0, 0, 2).Format(time.DateOnly) || ev.LastDate != today.AddDate(0, 0, 3).Format(time.DateOnly)) {
			t.Fatalf("conference = %+v", ev)
		}
	}
	if got["Run"] != 5 || got["Dentist"] != 1 || got["Conference"] != 1 {
		t.Fatalf("events = %v", got)
	}
	if out := bob.mustDo(200, "GET", q, nil); strings.Contains(string(out), "Dentist") {
		t.Fatal("bob sees alice's calendar")
	}

	// A refresh picks up changes; a failure is kept and the old events stay.
	served = strings.Replace(ics, "Dentist", "Dentist (moved)", 1)
	if out := alice.mustDo(200, "POST", "/v1/calendar/sources/"+src.ID+"/refresh", nil); !strings.Contains(string(out), `"error":""`) {
		t.Fatalf("refresh = %s", out)
	}
	if out := alice.mustDo(200, "GET", q, nil); !strings.Contains(string(out), "Dentist (moved)") {
		t.Fatalf("not refreshed: %s", out)
	}
	served = "<html>"
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/calendar/sources/"+src.ID+"/refresh", nil), &src)
	if src.Error == "" || src.Events != 7 {
		t.Fatalf("after a failed refresh: %+v", src)
	}
	bob.mustDo(404, "DELETE", "/v1/calendar/sources/"+src.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/calendar/sources/"+src.ID, nil)
	if out := alice.mustDo(200, "GET", q, nil); strings.Contains(string(out), "Dentist") {
		t.Fatal("events kept after unfollowing")
	}
}
