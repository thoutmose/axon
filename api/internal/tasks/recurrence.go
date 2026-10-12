package tasks

import (
	"errors"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Recurrence is date-based: occurrences are calendar days, computed at UTC
// midnight. With no times of day there is nothing for daylight-saving
// changes or time zones to shift; "today" comes from the user's time zone.

// allowedRuleParts are the RRULE parts Axon supports. COUNT is excluded
// because each occurrence is computed from the previous one.
var allowedRuleParts = map[string]bool{
	"FREQ": true, "INTERVAL": true, "BYDAY": true, "BYMONTHDAY": true, "BYMONTH": true,
	"BYSETPOS": true, "UNTIL": true, "WKST": true,
}

var errBadRule = errors.New(`recurrence must be an RRULE such as "FREQ=WEEKLY;BYDAY=MO,TH" with FREQ DAILY, WEEKLY, MONTHLY, or YEARLY (COUNT is not supported; use UNTIL)`)

// ParseRule validates a rule and returns it in canonical form.
func ParseRule(rule string) (string, error) {
	rule = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(rule)), "RRULE:")
	if rule == "" || len(rule) > 200 {
		return "", errBadRule
	}
	for _, part := range strings.Split(rule, ";") {
		key, _, ok := strings.Cut(part, "=")
		if !ok || !allowedRuleParts[key] {
			return "", errBadRule
		}
	}
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return "", errBadRule
	}
	switch opt.Freq {
	case rrule.DAILY, rrule.WEEKLY, rrule.MONTHLY, rrule.YEARLY:
	default:
		return "", errBadRule
	}
	return rule, nil
}

// NextOccurrence returns the first date the rule produces strictly after
// current, treating current as an occurrence of the rule. ok is false when
// the series has ended (UNTIL).
func NextOccurrence(rule string, current time.Time) (next time.Time, ok bool, err error) {
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return time.Time{}, false, errBadRule
	}
	opt.Dtstart = day(current)
	r, err := rrule.NewRRule(*opt)
	if err != nil {
		return time.Time{}, false, errBadRule
	}
	next = r.After(opt.Dtstart, false)
	if next.IsZero() {
		return time.Time{}, false, nil
	}
	return day(next), true, nil
}

// day truncates to a calendar date at UTC midnight.
func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
