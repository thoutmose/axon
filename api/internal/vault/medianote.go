package vault

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// MediaNote is a film's, series', book's, or game's literature note, filled
// from its vault template as the web app fills it (mediaNote in
// web/src/lib/noteTemplates.ts), for an agent's approved proposal (EP-153).
func MediaNote(kind, title, creators string, year int, genres []string, total int, now time.Time) (path, content string, ok bool) {
	var files []file
	if err := json.Unmarshal(templatesJSON, &files); err != nil {
		return "", "", false
	}
	for _, f := range files {
		if f.Path != "Templates/"+kind+".md" {
			continue
		}
		num := func(n int) string {
			if n == 0 {
				return ""
			}
			return strconv.Itoa(n)
		}
		list := ""
		if len(genres) > 0 {
			list = "[" + strings.Join(genres, ", ") + "]"
		}
		c := strings.NewReplacer("{{creator}}", creators, "{{year}}", num(year), "{{genres}}", list, "{{total}}", num(total),
			"{{cover}}", "", "{{title}}", strings.ReplaceAll(title, `"`, `'`),
			"{{created}}", now.Format("2006-01-02 15:04"), "{{date}}", now.Format(time.DateOnly)).Replace(f.Content)
		return "Zettel/literature/" + now.Format("200601021504") + "-" + slug(title) + ".md", c, true
	}
	return "", "", false
}

var notWord = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// slug is a title as a file name, as the web app writes it.
func slug(title string) string {
	s := strings.Trim(notWord.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if r := []rune(s); len(r) > 80 {
		s = strings.TrimRightFunc(string(r[:80]), func(r rune) bool { return r == '-' || unicode.IsSpace(r) })
	}
	if s == "" {
		return "note"
	}
	return s
}
