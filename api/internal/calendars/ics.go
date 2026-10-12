// Package calendars shows the person's other calendars in Axon (EP-58):
// Google, Nextcloud or any CalDAV server, Outlook, or anything that
// publishes an iCalendar (ICS) address. Read only: Axon fetches the
// address, keeps the events in a window around today, and never writes
// back.
package calendars

import (
	"bufio"
	"bytes"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Event is one occurrence, ready to show.
type Event struct {
	UID      string    `json:"uid"`
	Title    string    `json:"title"`
	Location string    `json:"location"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"all_day"`
}

type prop struct {
	name   string
	params map[string]string
	value  string
}

// lines unfolds an ICS file (RFC 5545 §3.1) into properties.
func lines(data []byte) []prop {
	var out []prop
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		if p, ok := parseLine(cur.String()); ok {
			out = append(out, p)
		}
		cur.Reset()
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		l := strings.TrimRight(sc.Text(), "\r")
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			cur.WriteString(l[1:])
			continue
		}
		flush()
		cur.WriteString(l)
	}
	flush()
	return out
}

func parseLine(l string) (prop, bool) {
	// The value starts at the first colon outside a quoted parameter.
	inQuote, colon := false, -1
	for i, r := range l {
		if r == '"' {
			inQuote = !inQuote
		} else if r == ':' && !inQuote {
			colon = i
			break
		}
	}
	if colon < 0 {
		return prop{}, false
	}
	head, value := l[:colon], l[colon+1:]
	parts := strings.Split(head, ";")
	p := prop{name: strings.ToUpper(parts[0]), params: map[string]string{}, value: value}
	for _, kv := range parts[1:] {
		if k, v, ok := strings.Cut(kv, "="); ok {
			p.params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return p, true
}

func unescape(s string) string {
	r := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return r.Replace(s)
}

// windowsZones maps the zone names Outlook writes to IANA names.
var windowsZones = map[string]string{
	"Romance Standard Time": "Europe/Paris", "W. Europe Standard Time": "Europe/Berlin",
	"Central Europe Standard Time": "Europe/Budapest", "GMT Standard Time": "Europe/London",
	"Eastern Standard Time": "America/New_York", "Central Standard Time": "America/Chicago",
	"Mountain Standard Time": "America/Denver", "Pacific Standard Time": "America/Los_Angeles",
	"UTC": "UTC", "Coordinated Universal Time": "UTC",
}

func location(tzid string, fallback *time.Location) *time.Location {
	if tzid == "" {
		return fallback
	}
	if n, ok := windowsZones[tzid]; ok {
		tzid = n
	}
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}
	return fallback
}

// when reads a DTSTART-like property: a date (all day), a UTC time, or a
// local time in its TZID (or floating, read in loc).
func when(p prop, loc *time.Location) (time.Time, bool, bool) {
	v := strings.TrimSpace(p.value)
	if p.params["VALUE"] == "DATE" || len(v) == 8 {
		t, err := time.ParseInLocation("20060102", v, loc)
		return t, true, err == nil
	}
	if strings.HasSuffix(v, "Z") {
		t, err := time.Parse("20060102T150405Z", v)
		return t, false, err == nil
	}
	t, err := time.ParseInLocation("20060102T150405", v, location(p.params["TZID"], loc))
	return t, false, err == nil
}

// duration reads an RFC 5545 duration such as PT1H30M or P1D.
func duration(s string) time.Duration {
	s = strings.TrimPrefix(strings.TrimPrefix(strings.ToUpper(s), "+"), "P")
	var d time.Duration
	num := ""
	inTime := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			num += string(r)
		case r == 'T':
			inTime = true
		default:
			n, _ := strconv.Atoi(num)
			num = ""
			switch {
			case r == 'W':
				d += time.Duration(n) * 7 * 24 * time.Hour
			case r == 'D':
				d += time.Duration(n) * 24 * time.Hour
			case r == 'H' && inTime:
				d += time.Duration(n) * time.Hour
			case r == 'M' && inTime:
				d += time.Duration(n) * time.Minute
			case r == 'S' && inTime:
				d += time.Duration(n) * time.Second
			}
		}
	}
	return d
}

type vevent struct {
	uid, title, location, rrule string
	start, end                  time.Time
	allDay, cancelled           bool
	exdates                     []time.Time
	recurrenceID                *time.Time
}

// Parse reads the events of an ICS file and expands repeating ones into
// the occurrences that overlap [from, to). Floating times are read in loc.
// ICS is UTF-8; bytes that are not become U+FFFD, so one bad calendar entry
// cannot stop the calendar loading.
func Parse(data []byte, from, to time.Time, loc *time.Location) []Event {
	data = bytes.ToValidUTF8(data, []byte("\uFFFD"))
	var evs []vevent
	var cur *vevent
	var dur time.Duration
	hasEnd := false
	for _, p := range lines(data) {
		switch {
		case p.name == "BEGIN" && strings.EqualFold(p.value, "VEVENT"):
			cur, dur, hasEnd = &vevent{}, 0, false
		case p.name == "END" && strings.EqualFold(p.value, "VEVENT") && cur != nil:
			if !hasEnd {
				switch {
				case dur > 0:
					cur.end = cur.start.Add(dur)
				case cur.allDay:
					cur.end = cur.start.AddDate(0, 0, 1)
				default:
					cur.end = cur.start
				}
			}
			if !cur.start.IsZero() {
				evs = append(evs, *cur)
			}
			cur = nil
		case cur == nil:
		case p.name == "UID":
			cur.uid = p.value
		case p.name == "SUMMARY":
			cur.title = unescape(p.value)
		case p.name == "LOCATION":
			cur.location = unescape(p.value)
		case p.name == "STATUS":
			cur.cancelled = strings.EqualFold(p.value, "CANCELLED")
		case p.name == "DTSTART":
			cur.start, cur.allDay, _ = when(p, loc)
		case p.name == "DTEND":
			cur.end, _, hasEnd = when(p, loc)
		case p.name == "DURATION":
			dur = duration(p.value)
		case p.name == "RRULE":
			cur.rrule = p.value
		case p.name == "EXDATE":
			for _, v := range strings.Split(p.value, ",") {
				if t, _, ok := when(prop{params: p.params, value: v}, loc); ok {
					cur.exdates = append(cur.exdates, t)
				}
			}
		case p.name == "RECURRENCE-ID":
			if t, _, ok := when(p, loc); ok {
				cur.recurrenceID = &t
			}
		}
	}
	// Moved or cancelled occurrences replace the ones the rule would make.
	overrides := map[string]map[int64]bool{}
	for _, e := range evs {
		if e.recurrenceID != nil {
			if overrides[e.uid] == nil {
				overrides[e.uid] = map[int64]bool{}
			}
			overrides[e.uid][e.recurrenceID.Unix()] = true
		}
	}
	var out []Event
	for _, e := range evs {
		length := e.end.Sub(e.start)
		starts := []time.Time{e.start}
		if e.rrule != "" && e.recurrenceID == nil {
			starts = expand(e.start, e.rrule, from.Add(-length), to)
		}
		for _, s := range starts {
			if e.cancelled || excluded(s, e.exdates) || (e.recurrenceID == nil && e.rrule != "" && overrides[e.uid][s.Unix()]) {
				continue
			}
			end := s.Add(length)
			if e.allDay {
				end = s.AddDate(0, 0, int(length.Hours()/24+0.5))
			}
			if !end.After(from) && !s.Equal(from) || !s.Before(to) {
				continue
			}
			out = append(out, Event{UID: e.uid, Title: e.title, Location: e.location, Start: s, End: end, AllDay: e.allDay})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func excluded(t time.Time, ex []time.Time) bool {
	for _, x := range ex {
		if x.Equal(t) {
			return true
		}
	}
	return false
}
