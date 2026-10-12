package calendars

import (
	"strings"
	"testing"
	"time"
)

const sample = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Google Inc//Google Calendar 70.9054//EN\r\n" +
	"BEGIN:VEVENT\r\nUID:standup@example\r\nDTSTART;TZID=Europe/Paris:20261005T093000\r\nDTEND;TZID=Europe/Paris:20261005T094500\r\n" +
	"RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR;UNTIL=20261231T000000Z\r\nEXDATE;TZID=Europe/Paris:20261014T093000\r\nSUMMARY:Stand-up\\, team\r\n" +
	"LOCATION:Room 2\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:standup@example\r\nRECURRENCE-ID;TZID=Europe/Paris:20261016T093000\r\nDTSTART;TZID=Europe/Paris:20261016T110000\r\n" +
	"DTEND;TZID=Europe/Paris:20261016T111500\r\nSUMMARY:Stand-up (moved)\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:holiday@example\r\nDTSTART;VALUE=DATE:20261020\r\nDTEND;VALUE=DATE:20261023\r\nSUMMARY:Holiday in\r\n  Brittany\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:dentist@example\r\nDTSTART:20261021T130000Z\r\nDURATION:PT45M\r\nSUMMARY:Dentist\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:cancelled@example\r\nDTSTART:20261022T130000Z\r\nSTATUS:CANCELLED\r\nSUMMARY:Cancelled\r\nEND:VEVENT\r\n" +
	"BEGIN:VEVENT\r\nUID:board@example\r\nDTSTART:20260101T170000Z\r\nDTEND:20260101T180000Z\r\nRRULE:FREQ=MONTHLY;BYDAY=-1TH;COUNT=24\r\nSUMMARY:Board\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func TestParse(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	from := time.Date(2026, 10, 12, 0, 0, 0, 0, paris)
	to := time.Date(2026, 11, 2, 0, 0, 0, 0, paris)
	evs := Parse([]byte(sample), from, to, paris)
	var got []string
	for _, e := range evs {
		got = append(got, e.Start.In(paris).Format("01-02 15:04")+" "+e.Title)
	}
	text := strings.Join(got, "\n")
	for _, w := range []string{
		"10-12 09:30 Stand-up, team",
		"10-16 11:00 Stand-up (moved)",
		"10-19 09:30 Stand-up, team",
		"10-20 00:00 Holiday in Brittany",
		"10-21 15:00 Dentist",
		"10-29 18:00 Board",          // the last Thursday of October, at 17:00 UTC
		"10-26 09:30 Stand-up, team", // after the change to winter time, still 09:30 in Paris
	} {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in\n%s", w, text)
		}
	}
	for _, bad := range []string{"10-14 09:30", "10-16 09:30", "Cancelled"} {
		if strings.Contains(text, bad) {
			t.Errorf("should not contain %q:\n%s", bad, text)
		}
	}
	for _, e := range evs {
		if e.Title == "Holiday in Brittany" && (!e.AllDay || e.End.Sub(e.Start) != 72*time.Hour) {
			t.Errorf("holiday = %+v", e)
		}
		if e.Title == "Dentist" && e.End.Sub(e.Start) != 45*time.Minute {
			t.Errorf("dentist = %+v", e)
		}
		if e.Title == "Stand-up, team" && e.Location != "Room 2" {
			t.Errorf("location = %q", e.Location)
		}
	}
}

func TestExpandCountAndInterval(t *testing.T) {
	start := time.Date(2026, 1, 31, 8, 0, 0, 0, time.UTC)
	got := expand(start, "FREQ=MONTHLY;COUNT=4", start, start.AddDate(2, 0, 0))
	// Months without a 31st are skipped, as RFC 5545 says; COUNT counts real occurrences.
	want := []string{"2026-01-31", "2026-03-31", "2026-05-31", "2026-07-31"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i, g := range got {
		if g.Format(time.DateOnly) != want[i] {
			t.Fatalf("got %v", got)
		}
	}
	every2 := expand(time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC), "FREQ=DAILY;INTERVAL=2", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	if len(every2) != 2 || every2[0].Day() != 11 || every2[1].Day() != 13 {
		t.Fatalf("every two days = %v", every2)
	}
}
