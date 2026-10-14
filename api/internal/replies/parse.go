// Package replies reads what a person writes back to Axon in a chat app
// (EP-71) and does it: "done 2" checks off the second item of the latest
// digest, "add call the bank tomorrow at 3pm" captures a task, "note …"
// a fleeting note, "snooze 3 monday" moves a task, "undo" takes the last
// one back. Free sentences such as "I finished the passport" are matched
// to open tasks and habits. The person is the actor: nothing is guessed
// silently, every answer says exactly what changed, and "undo" reverts it.
// It understands English and French.
package replies

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Verbs a message can carry.
const (
	Done    = "done"
	Undo    = "undo"
	Add     = "add"
	Note    = "note"
	Snooze  = "snooze"
	List    = "list"
	Help    = "help"
	Link    = "link"
	Unknown = "unknown"
)

// When is a day, and maybe a time, said in words.
type When struct {
	Days int    // from today
	Date string // YYYY-MM-DD, when given as a date
	At   string // HH:MM, or empty
}

// Command is a parsed message.
type Command struct {
	Verb    string
	Numbers []int
	Text    string // what is left: a title, a code, words to match
	When    *When
}

// fold lower-cases and removes accents, for matching keywords.
func fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, strings.ToLower(s))
	if err != nil {
		return strings.ToLower(s)
	}
	return out
}

var keywords = map[string][]string{
	Done:   {"done", "did", "finished", "finish", "complete", "completed", "fait", "faite", "fini", "finie", "termine", "terminee", "ok", "✅", "✔", "✔️", "☑️", "x"},
	Undo:   {"undo", "annule", "annuler", "oops", "revert"},
	Add:    {"add", "todo", "task", "ajoute", "ajouter", "tache", "+"},
	Note:   {"note", "idea", "idee", "capture", "remember", "retiens"},
	Snooze: {"snooze", "later", "postpone", "defer", "reporte", "reporter", "decale"},
	List:   {"list", "today", "tasks", "agenda", "liste", "aujourd'hui", "aujourdhui", "taches", "digest"},
	Help:   {"help", "aide", "?", "commands", "commandes", "start"},
	Link:   {"link", "lier", "relier"},
}

// completion words found inside a sentence: "I finished the report".
var completion = []string{"done", "did", "finished", "completed", "achieved", "fait", "fini", "finie", "termine", "terminee", "reussi"}

// filler is left out of the words matched against titles.
var filler = map[string]bool{
	"i": true, "ive": true, "i've": true, "have": true, "has": true, "just": true, "the": true, "a": true, "an": true, "my": true, "with": true, "is": true, "it": true, "now": true, "today": true, "already": true, "and": true,
	"j'ai": true, "jai": true, "j": true, "ai": true, "le": true, "la": true, "les": true, "l": true, "un": true, "une": true, "de": true, "du": true, "des": true, "mon": true, "ma": true, "mes": true, "est": true, "c'est": true, "deja": true, "enfin": true, "et": true, "aujourd'hui": true,
}

var (
	numberList = regexp.MustCompile(`^#?(\d{1,3})(?:\s*-\s*(\d{1,3}))?$`)
	isoDate    = regexp.MustCompile(`\b(\d{4}-\d{2}-\d{2})\b`)
	// "at 15:00", "at 3pm", "at 3:30 pm", "à 18h30", "à 18h", "18h30".
	timeRe   = regexp.MustCompile(`(?i)\b(\d{1,2})(?:(?::|h)(\d{2}))?\s*(am|pm|h)?\b`)
	atWord   = regexp.MustCompile(`(?i)(?:^|\s)(?:at|a|à|vers)\s*$`)
	inDays   = regexp.MustCompile(`\b(?:in|dans)\s+(\d{1,3})\s+(?:days?|jours?)\b`)
	weekdays = map[string]time.Weekday{
		"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
		"dimanche": time.Sunday, "lundi": time.Monday, "mardi": time.Tuesday, "mercredi": time.Wednesday, "jeudi": time.Thursday, "vendredi": time.Friday, "samedi": time.Saturday,
	}
)

// Parse reads one message.
func Parse(msg string) Command {
	text := strings.TrimSpace(msg)
	if text == "" {
		return Command{Verb: Help}
	}
	text = strings.TrimPrefix(text, "/")
	if i := strings.Index(text, "@"); i > 0 && !strings.Contains(text[:i], " ") {
		text = text[:i] + strings.TrimLeft(text[i:], "@abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_") // "/done@MyBot 2"
	}
	words := strings.Fields(text)
	first := strings.Trim(fold(words[0]), ".,!:;")
	rest := strings.TrimSpace(strings.TrimPrefix(text, words[0]))
	verb := ""
	for v, list := range keywords {
		for _, k := range list {
			if first == k {
				verb = v
			}
		}
	}
	// "2 done", "2 ✅".
	if verb == "" && len(words) == 2 && numberList.MatchString(words[0]) {
		if second := fold(words[1]); slicesContains(keywords[Done], second) {
			verb, rest = Done, words[0]
		}
	}
	// A bare list of numbers means done: "2", "1 3".
	if verb == "" {
		if nums, ok := numbers(text); ok {
			return Command{Verb: Done, Numbers: nums}
		}
	}
	switch verb {
	case Done:
		if nums, ok := numbers(rest); ok {
			return Command{Verb: Done, Numbers: nums}
		}
		if rest == "" {
			return Command{Verb: Help}
		}
		return Command{Verb: Done, Text: rest}
	case Snooze:
		c := Command{Verb: Snooze}
		parts := strings.Fields(rest)
		for len(parts) > 0 {
			n, ok := numbers(parts[0])
			if !ok {
				break
			}
			c.Numbers = append(c.Numbers, n...)
			parts = parts[1:]
		}
		w, _ := parseWhen(strings.Join(parts, " "))
		if w == nil {
			w = &When{Days: 1}
		}
		c.When = w
		return c
	case Add, Note:
		if rest == "" {
			return Command{Verb: Help}
		}
		c := Command{Verb: verb, Text: rest}
		if verb == Add {
			c.When, c.Text = parseWhen(rest)
		}
		return c
	case Link:
		return Command{Verb: Link, Text: rest}
	case Help:
		if first == "start" && rest != "" { // Telegram's /start CODE
			return Command{Verb: Link, Text: rest}
		}
		return Command{Verb: Help}
	case Undo, List:
		return Command{Verb: verb}
	}
	// A sentence that says something was done: "I finished the report".
	folded := fold(text)
	for _, w := range strings.FieldsFunc(folded, notWord) {
		if slicesContains(completion, w) {
			return Command{Verb: Done, Text: text}
		}
	}
	return Command{Verb: Unknown, Text: text}
}

func notWord(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' }

func slicesContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// numbers reads "1 3", "1,3", "1-3", "#2"; ok is false unless it is only numbers.
func numbers(s string) ([]int, bool) {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' || r == ';' || r == '&' })
	if len(fields) == 0 {
		return nil, false
	}
	// Rejoin "1 - 3" split by spaces.
	joined := strings.Join(fields, " ")
	joined = regexp.MustCompile(`\s*-\s*`).ReplaceAllString(joined, "-")
	var out []int
	for _, f := range strings.Fields(joined) {
		if fold(f) == "and" || fold(f) == "et" {
			continue
		}
		m := numberList.FindStringSubmatch(f)
		if m == nil {
			return nil, false
		}
		a, _ := strconv.Atoi(m[1])
		b := a
		if m[2] != "" {
			b, _ = strconv.Atoi(m[2])
		}
		if a < 1 || b < a || b-a > 20 {
			return nil, false
		}
		for n := a; n <= b; n++ {
			out = append(out, n)
		}
	}
	return out, len(out) > 0
}

// parseWhen finds a day and a time in a title and returns the title without them.
func parseWhen(s string) (*When, string) {
	var w When
	found := false
	f := fold(s)
	cut := func(re string) {
		r := regexp.MustCompile(`(?i)\s*\b` + re + `\b`)
		s = strings.TrimSpace(r.ReplaceAllString(s, ""))
		f = fold(s)
	}
	switch {
	case isoDate.MatchString(s):
		w.Date, found = isoDate.FindString(s), true
		s = strings.TrimSpace(isoDate.ReplaceAllString(s, ""))
		f = fold(s)
	case strings.Contains(f, "after tomorrow") || strings.Contains(f, "apres-demain") || strings.Contains(f, "apres demain"):
		w.Days, found = 2, true
		cut(`(?:the day )?after tomorrow|apr[eè]s[- ]demain`)
	case strings.Contains(f, "tomorrow") || strings.Contains(f, "demain"):
		w.Days, found = 1, true
		cut(`tomorrow|demain`)
	case strings.Contains(f, "today") || strings.Contains(f, "aujourd'hui") || strings.Contains(f, "tonight") || strings.Contains(f, "ce soir"):
		w.Days, found = 0, true
		cut(`today|aujourd'hui|tonight|ce soir`)
	case strings.Contains(f, "next week") || strings.Contains(f, "semaine prochaine"):
		w.Days, found = -1, true // next Monday, resolved against today
		cut(`next week|la semaine prochaine|semaine prochaine`)
	case inDays.MatchString(f):
		m := inDays.FindStringSubmatch(f)
		w.Days, _ = strconv.Atoi(m[1])
		found = true
		cut(`(?:in|dans)\s+\d{1,3}\s+(?:days?|jours?)`)
	default:
		for name, wd := range weekdays {
			if regexp.MustCompile(`\b` + name + `\b`).MatchString(f) {
				w.Days, found = -10-int(wd), true // a weekday, resolved against today
				cut(`(?:on |next |ce |le )?` + name + `(?: prochain)?`)
				break
			}
		}
	}
	for _, m := range timeRe.FindAllStringSubmatchIndex(s, -1) {
		before := s[:m[0]]
		at := atWord.FindStringIndex(before)
		if !hasTimeMarker(s[m[0]:m[1]]) && at == nil {
			continue // "buy 2 apples" is not 2 o'clock
		}
		h, _ := strconv.Atoi(s[m[2]:m[3]])
		minute := 0
		if m[4] >= 0 {
			minute, _ = strconv.Atoi(s[m[4]:m[5]])
		}
		suffix := ""
		if m[6] >= 0 {
			suffix = strings.ToLower(s[m[6]:m[7]])
		}
		if suffix == "pm" && h < 12 {
			h += 12
		}
		if suffix == "am" && h == 12 {
			h = 0
		}
		if h >= 24 || minute >= 60 {
			continue
		}
		w.At, found = fmtTime(h, minute), true
		start := m[0]
		if at != nil {
			start = at[0]
		}
		s = strings.TrimSpace(s[:start] + s[m[1]:])
		break
	}
	if !found {
		return nil, strings.TrimSpace(s)
	}
	return &w, strings.Join(strings.Fields(s), " ")
}

// hasTimeMarker says a number is a time by itself: 15:00, 18h30, 3pm.
func hasTimeMarker(s string) bool {
	f := fold(s)
	return strings.Contains(f, ":") || strings.Contains(f, "h") || strings.HasSuffix(f, "am") || strings.HasSuffix(f, "pm")
}

func fmtTime(h, m int) string {
	return strconv.Itoa(h/10) + strconv.Itoa(h%10) + ":" + strconv.Itoa(m/10) + strconv.Itoa(m%10)
}

// Resolve turns a When into a date (and maybe a time) from today.
func (w When) Resolve(today time.Time) (time.Time, string) {
	switch {
	case w.Date != "":
		if d, err := time.Parse(time.DateOnly, w.Date); err == nil {
			return d, w.At
		}
		return today, w.At
	case w.Days == -1: // next week: next Monday
		return today.AddDate(0, 0, 7-(int(today.Weekday())+6)%7), w.At
	case w.Days <= -10: // a weekday: the next one after today
		target := time.Weekday(-w.Days - 10)
		diff := (int(target) - int(today.Weekday()) + 7) % 7
		if diff == 0 {
			diff = 7
		}
		return today.AddDate(0, 0, diff), w.At
	}
	return today.AddDate(0, 0, w.Days), w.At
}
