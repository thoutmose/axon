package notes

import (
	"path"
	"regexp"
	"strings"
)

// Meta is what Axon reads from a note; the file itself is kept verbatim.
type Meta struct {
	Title string
	Kind  string // front matter "type", or "note"
	Para  string
	Tags  []string
}

// frontMatter returns the YAML block between the opening and closing "---"
// lines, and the body after it.
func frontMatter(content string) (fm, body string) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return "", content
	}
	rest := content[strings.Index(content, "\n")+1:]
	for i := 0; i < len(rest); {
		end := strings.IndexByte(rest[i:], '\n')
		line := rest[i:]
		if end >= 0 {
			line = rest[i : i+end]
		}
		if strings.TrimRight(line, "\r") == "---" {
			if end < 0 {
				return rest[:i], ""
			}
			return rest[:i], rest[i+end+1:]
		}
		if end < 0 {
			break
		}
		i += end + 1
	}
	return "", content // unterminated: treat it all as body
}

// field reads a top-level "key: value" from front matter, without a YAML
// library: Axon only needs a few scalar fields and must never rewrite them.
func field(fm, key string) string {
	for _, line := range strings.Split(fm, "\n") {
		k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), ":")
		if ok && k == key {
			v = strings.TrimSpace(v)
			if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
			return v
		}
	}
	return ""
}

// Parse derives a note's metadata from its path and content.
func Parse(p, content string) Meta {
	fm, body := frontMatter(content)
	m := Meta{Title: field(fm, "title"), Kind: strings.ToLower(field(fm, "type")), Para: paraOf(p)}
	if m.Title == "" {
		for _, line := range strings.Split(body, "\n") {
			if h, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "# "); ok {
				m.Title = strings.TrimSpace(h)
				break
			}
		}
	}
	if m.Title == "" {
		m.Title = Stem(p)
	}
	if len([]rune(m.Title)) > 500 {
		m.Title = string([]rune(m.Title)[:500])
	}
	if m.Kind == "" || len(m.Kind) > 40 {
		m.Kind = "note"
	}
	m.Tags = tags(fm, body)
	return m
}

// inlineTag is "#tag" after a space or the start of a line: letters,
// digits, _, -, / (nested tags), with at least one letter.
var inlineTag = regexp.MustCompile(`(?:^|[\s(])#([\p{L}\p{N}_/-]*\p{L}[\p{L}\p{N}_/-]*)`)

// tags reads front matter "tags: [a, b]" or a "- a" list, and #tags in the
// body outside code. They are lower-cased, de-duplicated, and capped.
func tags(fm, body string) []string {
	seen := map[string]bool{}
	out := []string{}
	add := func(t string) {
		t = strings.ToLower(strings.Trim(strings.TrimSpace(t), `"'#`))
		if t != "" && len(t) <= 60 && !seen[t] && len(out) < 50 {
			seen[t] = true
			out = append(out, t)
		}
	}
	lines := strings.Split(fm, "\n")
	for i, line := range lines {
		k, v, ok := strings.Cut(strings.TrimRight(line, "\r"), ":")
		if !ok || (k != "tags" && k != "tag") {
			continue
		}
		v = strings.TrimSpace(v)
		if v != "" {
			for _, t := range strings.Split(strings.Trim(v, "[]"), ",") {
				add(t)
			}
			continue
		}
		for _, item := range lines[i+1:] {
			item = strings.TrimSpace(strings.TrimRight(item, "\r"))
			t, isItem := strings.CutPrefix(item, "- ")
			if !isItem {
				break
			}
			add(t)
		}
	}
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		line = inlineCode.ReplaceAllString(line, "")
		for _, m := range inlineTag.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
	}
	return out
}

var inlineCode = regexp.MustCompile("`[^`]*`")

// Stem is the file name without folder or extension: what [[wikilinks]] name.
func Stem(p string) string { return strings.TrimSuffix(path.Base(p), ".md") }

var paraFolders = map[string]string{
	"inbox": "inbox", "projects": "projects", "areas": "areas",
	"resources": "resources", "archive": "archive", "archives": "archive",
}

// paraOf maps the top folder to a PARA category: "1-Projects/x.md" → projects.
func paraOf(p string) string {
	top, _, nested := strings.Cut(p, "/")
	if !nested {
		return "inbox"
	}
	name := strings.ToLower(strings.TrimLeft(top, "0123456789-_ "))
	if para, ok := paraFolders[name]; ok {
		return para
	}
	return "other"
}

// Wikilink is one [[target#heading|alias]] occurrence.
type Wikilink struct {
	Target string // file stem or title, as written
}

var wikilinkRE = regexp.MustCompile(`\[\[([^\[\]|#\n]+)(#[^\[\]|\n]*)?(\|[^\[\]\n]*)?\]\]`)

// Wikilinks returns the distinct link targets in content (front matter included).
func Wikilinks(content string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range wikilinkRE.FindAllStringSubmatch(content, -1) {
		t := strings.TrimSpace(m[1])
		if t != "" && !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			out = append(out, t)
		}
	}
	return out
}

// RenameLinks rewrites [[old]], [[old#h]], and [[old|alias]] to point at
// newStem, leaving headings and aliases intact.
func RenameLinks(content, old, newStem string) string {
	return wikilinkRE.ReplaceAllStringFunc(content, func(link string) string {
		m := wikilinkRE.FindStringSubmatch(link)
		if !strings.EqualFold(strings.TrimSpace(m[1]), old) {
			return link
		}
		return "[[" + newStem + m[2] + m[3] + "]]"
	})
}

// CleanPath validates a vault-relative path such as "Zettel/2026-idea.md".
func CleanPath(p string) (string, bool) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	clean := path.Clean(p)
	ok := clean == p && strings.HasSuffix(clean, ".md") && !strings.HasPrefix(clean, "/") &&
		!strings.HasPrefix(clean, ".") && !strings.Contains(clean, "/.") && len(clean) <= 500 && Stem(clean) != ""
	return clean, ok
}

// Slug turns a title into a file stem: "Raft & Paxos!" → "raft-paxos".
func Slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 && r != ' ':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "note"
	}
	if len(s) > 80 {
		s = strings.TrimRight(s[:80], "-")
	}
	return s
}
