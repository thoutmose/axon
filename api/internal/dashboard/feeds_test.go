package dashboard

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/tasks"
)

func TestICal(t *testing.T) {
	day := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("01a12600-0000-7000-8000-000000000001")
	long := strings.Repeat("Ünïcödé ", 12)
	out := ICal([]Event{
		{Date: tasks.Date{Time: day}, Type: "task", Kind: "due", ID: id, Title: "Pay rent; call landlord, then\nrelax \\ rest"},
		{Date: tasks.Date{Time: day}, Type: "goal", Kind: "target", ID: id, Title: long},
	}, day)

	for _, want := range []string{
		"BEGIN:VCALENDAR\r\nVERSION:2.0\r\n",
		"UID:" + id.String() + "-due@axon\r\n",
		"DTSTART;VALUE=DATE:20261012\r\nDTEND;VALUE=DATE:20261013\r\n",
		`SUMMARY:Due: Pay rent\; call landlord\, then\nrelax \\ rest` + "\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	for _, l := range strings.Split(strings.TrimSuffix(out, "\r\n"), "\r\n") {
		if len(l) > 75 || !utf8.ValidString(l) {
			t.Errorf("line not folded at 75 octets on a character boundary: %q", l)
		}
	}
	unfolded := strings.ReplaceAll(out, "\r\n ", "")
	if !strings.Contains(unfolded, "SUMMARY:Goal: "+long) {
		t.Error("folding changed the text")
	}
}
