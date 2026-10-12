package mirror

import (
	"regexp"
	"strconv"
	"strings"
)

// A mirrored note is plain Markdown that Obsidian and any editor show well:
//
//	---
//	type: task
//	axon_id: 0190…
//	status: next
//	---
//	# Call the plumber
//
//	## Notes
//	…
//
//	<!-- footer -->
//
// Front matter holds the fields, sections hold the text. Sections marked
// read-only are rewritten by Axon; the others, the title, and the editable
// front matter flow back to the item.

type kv struct{ k, v string }

type section struct {
	heading  string
	body     string
	readOnly bool
}

type page struct {
	fm       []kv
	title    string
	sections []section
	footer   string
}

var needsQuote = regexp.MustCompile(`^[\s\-?:,\[\]{}#&*!|>'"%@` + "`" + `]|[:#]\s|\s$|^(true|false|null|yes|no|~)$|^[0-9.+-]+$`)

// yamlValue writes a scalar so YAML reads it back as the same string.
func yamlValue(v string) string {
	if v == "" {
		return ""
	}
	if needsQuote.MatchString(v) || strings.ContainsAny(v, "\n\"\\") {
		return strconv.Quote(v)
	}
	return v
}

func (p page) markdown() string {
	var b strings.Builder
	b.WriteString("---\n")
	for _, f := range p.fm {
		b.WriteString(f.k)
		b.WriteString(":")
		if f.v != "" {
			b.WriteString(" ")
			b.WriteString(f.v)
		}
		b.WriteString("\n")
	}
	b.WriteString("---\n# ")
	b.WriteString(p.title)
	b.WriteString("\n")
	for _, s := range p.sections {
		b.WriteString("\n## ")
		b.WriteString(s.heading)
		b.WriteString("\n")
		if body := strings.TrimSpace(s.body); body != "" {
			b.WriteString(body)
			b.WriteString("\n")
		}
	}
	if p.footer != "" {
		b.WriteString("\n<!-- ")
		b.WriteString(p.footer)
		b.WriteString(" -->\n")
	}
	return b.String()
}

// parsed is what a note says.
type parsed struct {
	fm       map[string]string
	title    string
	sections map[string]string
}

var comment = regexp.MustCompile(`(?s)<!--.*?-->`)

// headings are the sections Axon writes. Any other "## " line is text the
// person wrote inside a section (a task's notes can have headings of their
// own), so it stays in that section instead of starting a new one.
var headings = map[string]bool{
	"Notes": true, "Outcome": true, "Milestones": true, "Description": true, "Identity": true,
	"Make it obvious (cue)": true, "Make it attractive (craving)": true, "Make it easy (two-minute version)": true,
	"Make it satisfying (reward)": true, "Habit contract": true, "Recent check-ins": true, "Overview": true,
	"Review": true, "Meals": true, "Totals": true, "Sets": true,
}

// parse reads front matter (flat "key: value" lines), the first "# " title,
// and "## " sections. Unknown parts are ignored, never an error.
func parse(content string) parsed {
	p := parsed{fm: map[string]string{}, sections: map[string]string{}}
	body := strings.ReplaceAll(content, "\r\n", "\n")
	if strings.HasPrefix(body, "---\n") {
		rest := body[4:]
		if end := strings.Index(rest, "\n---"); end >= 0 {
			for _, line := range strings.Split(rest[:end], "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "#") {
					continue
				}
				p.fm[strings.TrimSpace(k)] = unquote(strings.TrimSpace(v))
			}
			body = rest[end+4:]
			if i := strings.IndexByte(body, '\n'); i >= 0 {
				body = body[i+1:]
			} else {
				body = ""
			}
		}
	}
	body = comment.ReplaceAllString(body, "")
	current := ""
	var buf []string
	flush := func() {
		if current != "" {
			p.sections[current] = strings.TrimSpace(strings.Join(buf, "\n"))
		}
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		switch {
		case p.title == "" && current == "" && strings.HasPrefix(line, "# "):
			p.title = strings.TrimSpace(line[2:])
		case strings.HasPrefix(line, "## ") && (headings[strings.TrimSpace(line[3:])] || current == ""):
			flush()
			current = strings.TrimSpace(line[3:])
		default:
			if current != "" {
				buf = append(buf, line)
			}
		}
	}
	flush()
	return p
}

func unquote(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		if s, err := strconv.Unquote(v); err == nil {
			return s
		}
	}
	if len(v) >= 2 && v[0] == '\'' && v[len(v)-1] == '\'' {
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v
}

// list writes a YAML flow list of strings.
func list(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = yamlValue(s)
		if q[i] == "" {
			q[i] = `""`
		}
	}
	return "[" + strings.Join(q, ", ") + "]"
}

// unlist reads "[a, b]" or "a, b".
func unlist(v string) []string {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(v), "["), "]"))
	out := []string{}
	if v == "" {
		return out
	}
	for _, s := range strings.Split(v, ",") {
		if s = unquote(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

var unsafeName = regexp.MustCompile(`[/\\:*?"<>|#^\[\]]+`)

// fileName turns a title into an Obsidian-friendly file name.
func fileName(title string) string {
	s := strings.TrimSpace(unsafeName.ReplaceAllString(title, " "))
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 120 {
		s = strings.TrimSpace(string(r[:120]))
	}
	if s == "" || strings.HasPrefix(s, ".") {
		s = strings.TrimSpace("Untitled " + s)
	}
	return s
}
