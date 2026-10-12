package learning

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/knowledge"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Cards drafted from a note's own markup, with no AI: the person already
// marked what matters while writing (generation: making the card yourself
// beats being handed one). Recognised, as in Obsidian's spaced repetition
// plugin and common note-taking habits:
//
//	question :: answer          a card; ":::" also makes the reverse
//	Q: question / A: answer     on consecutive lines (R: in French)
//	a sentence with ==a key term== in it      a cloze: the term is hidden
//	**Term**: definition (or — definition)    a definition card

// Draft is a proposed card; nothing is stored until the person adds it.
type Draft struct {
	Front  string `json:"front"`
	Back   string `json:"back"`
	Kind   string `json:"kind"` // qa, reverse, cloze, definition
	Line   int    `json:"line"` // 1-based, in the note
	Exists bool   `json:"exists"`
}

var (
	qaInline   = regexp.MustCompile(`^\s*(?:[-*+]\s+)?(.+?)\s+(:{2,3})\s+(.+?)\s*$`)
	qLine      = regexp.MustCompile(`^\s*(?:[-*+]\s+)?(?i:q)\s*:\s*(.+)$`)
	aLine      = regexp.MustCompile(`^\s*(?:[-*+]\s+)?(?i:a|r)\s*:\s*(.+)$`)
	highlight  = regexp.MustCompile(`==([^=\n]+)==`)
	definition = regexp.MustCompile(`^\s*(?:[-*+]\s+)?\*\*([^*\n]{1,80})\*\*\s*(?::|—|–|-)\s+(.{3,})$`)
	listMark   = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+`)
)

// Extract finds card drafts in a note's Markdown.
func Extract(content string) []Draft {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	start, fenced := 0, false
	if len(lines) > 0 && lines[0] == "---" {
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				start = i + 1
				break
			}
		}
	}
	define := "What is %s?"
	if knowledge.Language(knowledge.Clean(content)) == "fr" {
		define = "Qu’est-ce que %s ?"
	}
	out := []Draft{}
	for i := start; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		if fenced || strings.HasPrefix(strings.TrimSpace(l), "#") || strings.HasPrefix(strings.TrimSpace(l), "|") {
			continue
		}
		if m := qLine.FindStringSubmatch(l); m != nil && i+1 < len(lines) {
			if a := aLine.FindStringSubmatch(lines[i+1]); a != nil {
				out = append(out, Draft{Front: tidy(m[1]), Back: tidy(a[1]), Kind: "qa", Line: i + 1})
				i++
				continue
			}
		}
		if m := qaInline.FindStringSubmatch(l); m != nil && !strings.Contains(m[1], "://") {
			out = append(out, Draft{Front: tidy(m[1]), Back: tidy(m[3]), Kind: "qa", Line: i + 1})
			if m[2] == ":::" {
				out = append(out, Draft{Front: tidy(m[3]), Back: tidy(m[1]), Kind: "reverse", Line: i + 1})
			}
			continue
		}
		if m := definition.FindStringSubmatch(l); m != nil {
			out = append(out, Draft{Front: strings.Replace(define, "%s", tidy(m[1]), 1), Back: tidy(m[2]), Kind: "definition", Line: i + 1})
			continue
		}
		if hs := highlight.FindAllStringSubmatch(l, -1); hs != nil {
			sentence := listMark.ReplaceAllString(l, "")
			for _, h := range hs {
				front := highlight.ReplaceAllStringFunc(sentence, func(m string) string {
					if m == "=="+h[1]+"==" {
						return "[…]"
					}
					return strings.Trim(m, "=")
				})
				out = append(out, Draft{Front: tidy(front), Back: tidy(h[1]), Kind: "cloze", Line: i + 1})
			}
		}
	}
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

func tidy(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// Drafts returns the card drafts of one of the person's notes, marking
// those already made into cards from this note.
func Drafts(ctx context.Context, q db.Querier, userID, noteID uuid.UUID) ([]Draft, error) {
	n, err := notes.Get(ctx, q, userID, noteID)
	if err != nil {
		return nil, err
	}
	drafts := Extract(*n.Content)
	rows, err := q.Query(ctx, `SELECT c.front FROM cards c JOIN links l ON l.source_id = c.id
		WHERE l.user_id = $1 AND l.target_id = $2 AND l.relation = 'derived_from'`, userID, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	have := map[string]bool{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		have[f] = true
	}
	for i := range drafts {
		drafts[i].Exists = have[drafts[i].Front]
	}
	return drafts, rows.Err()
}
