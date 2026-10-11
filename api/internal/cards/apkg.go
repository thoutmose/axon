package cards

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // Anki's note checksum is defined as SHA-1; not used for security
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // pure-Go SQLite for the Anki collection file
)

// The Anki package (.apkg) export: a zip holding collection.anki2 (an SQLite
// database in Anki's schema 11, which every current Anki imports) and the
// card pictures. Cards are exported as new cards: Axon keeps the FSRS
// schedule, and Anki starts its own.

const schema = `
CREATE TABLE col (id integer primary key, crt integer not null, mod integer not null, scm integer not null, ver integer not null,
	dty integer not null, usn integer not null, ls integer not null, conf text not null, models text not null, decks text not null,
	dconf text not null, tags text not null);
CREATE TABLE notes (id integer primary key, guid text not null, mid integer not null, mod integer not null, usn integer not null,
	tags text not null, flds text not null, sfld integer not null, csum integer not null, flags integer not null, data text not null);
CREATE TABLE cards (id integer primary key, nid integer not null, did integer not null, ord integer not null, mod integer not null,
	usn integer not null, type integer not null, queue integer not null, due integer not null, ivl integer not null, factor integer not null,
	reps integer not null, lapses integer not null, left integer not null, odue integer not null, odid integer not null, flags integer not null,
	data text not null);
CREATE TABLE revlog (id integer primary key, cid integer not null, usn integer not null, ease integer not null, ivl integer not null,
	lastIvl integer not null, factor integer not null, time integer not null, type integer not null);
CREATE TABLE graves (usn integer not null, oid integer not null, type integer not null);
CREATE INDEX ix_notes_usn on notes (usn);
CREATE INDEX ix_cards_usn on cards (usn);
CREATE INDEX ix_revlog_usn on revlog (usn);
CREATE INDEX ix_cards_nid on cards (nid);
CREATE INDEX ix_cards_sched on cards (did, queue, due);
CREATE INDEX ix_revlog_cid on revlog (cid);
CREATE INDEX ix_notes_csum on notes (csum);
`

// ankiID turns a UUID into a stable positive 53-bit id (Anki uses JS numbers).
func ankiID(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[8:]) & (1<<53 - 1)) //nolint:gosec // masked to 53 bits
}

func checksum(s string) int64 {
	h := sha1.Sum([]byte(s)) //nolint:gosec // see import
	return int64(binary.BigEndian.Uint32(h[:4]))
}

// stripHTML is the sort field: the question as plain text.
func stripHTML(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// field turns card text into Anki HTML: escaped, with line breaks kept.
func field(s string) string {
	return strings.ReplaceAll(html.EscapeString(s), "\n", "<br>")
}

// APKG writes the cards as an Anki package, one Anki deck per Axon deck.
func APKG(ctx context.Context, list []Card, image func(uuid.UUID) ([]byte, error), now time.Time) ([]byte, error) {
	dir, err := os.MkdirTemp("", "axon-apkg-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, "collection.anki2")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(ctx, schema); err != nil {
		return nil, err
	}

	ms := now.UnixMilli()
	const modelID = int64(1_700_000_000_016) // "Axon Basic"; stable so re-imports update notes
	decks := map[string]any{"1": deckJSON(1, "Default", ms)}
	deckIDs := map[string]int64{}
	for _, c := range list {
		if _, ok := deckIDs[c.Deck]; !ok {
			id := checksum("axon deck "+c.Deck)&(1<<40-1) + 1_000_000
			deckIDs[c.Deck] = id
			decks[strconv.FormatInt(id, 10)] = deckJSON(id, c.Deck, ms)
		}
	}
	model := map[string]any{
		"id": modelID, "name": "Axon Basic", "type": 0, "mod": ms / 1000, "usn": -1, "sortf": 0, "did": 1, "tags": []string{}, "vers": []int{},
		"flds": []map[string]any{
			{"name": "Front", "ord": 0, "sticky": false, "rtl": false, "font": "Arial", "size": 20, "media": []string{}},
			{"name": "Back", "ord": 1, "sticky": false, "rtl": false, "font": "Arial", "size": 20, "media": []string{}},
		},
		"tmpls":    []map[string]any{{"name": "Card 1", "ord": 0, "qfmt": "{{Front}}", "afmt": "{{FrontSide}}<hr id=answer>{{Back}}", "did": nil, "bqfmt": "", "bafmt": ""}},
		"css":      ".card { font-family: system-ui, sans-serif; font-size: 20px; text-align: center; } img { max-width: 100%; }",
		"latexPre": "\\documentclass[12pt]{article}\n\\begin{document}\n", "latexPost": "\\end{document}", "req": [][]any{{0, "all", []int{0}}},
	}
	conf := map[string]any{"nextPos": len(list) + 1, "estTimes": true, "activeDecks": []int{1}, "sortType": "noteFld", "timeLim": 0,
		"sortBackwards": false, "addToCur": true, "curDeck": 1, "newBury": true, "newSpread": 0, "dueCounts": true, "curModel": strconv.FormatInt(modelID, 10), "collapseTime": 1200}
	dconf := map[string]any{"1": map[string]any{"id": 1, "name": "Default", "mod": 0, "usn": 0, "maxTaken": 60, "autoplay": true, "timer": 0, "replayq": true, "dyn": false,
		"new":   map[string]any{"bury": true, "delays": []int{1, 10}, "initialFactor": 2500, "ints": []int{1, 4, 7}, "order": 1, "perDay": 20, "separate": true},
		"lapse": map[string]any{"delays": []int{10}, "leechAction": 0, "leechFails": 8, "minInt": 1, "mult": 0},
		"rev":   map[string]any{"bury": true, "ease4": 1.3, "fuzz": 0.05, "ivlFct": 1, "maxIvl": 36500, "minSpace": 1, "perDay": 100}}}
	j := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	if _, err := conn.ExecContext(ctx, `INSERT INTO col VALUES (1, ?, ?, ?, 11, 0, 0, 0, ?, ?, ?, ?, '{}')`,
		now.Unix(), ms, ms, j(conf), j(map[string]any{strconv.FormatInt(modelID, 10): model}), j(decks), j(dconf)); err != nil {
		return nil, err
	}

	media := map[string]string{}
	var files [][2]string // zip name, file path
	for i, c := range list {
		back := field(c.Back)
		if c.HasImage {
			img, err := image(c.ID)
			if err != nil {
				return nil, err
			}
			if img != nil {
				name := fmt.Sprintf("axon-%s.png", c.ID)
				key := strconv.Itoa(len(media))
				media[key] = name
				p := filepath.Join(dir, key)
				if err := os.WriteFile(p, img, 0o600); err != nil {
					return nil, err
				}
				files = append(files, [2]string{key, p})
				back = fmt.Sprintf(`<img src="%s"><br>%s`, name, back)
			}
		}
		front := field(c.Front)
		nid := ankiID(c.ID)
		if _, err := conn.ExecContext(ctx, `INSERT INTO notes VALUES (?, ?, ?, ?, -1, ' axon ', ?, ?, ?, 0, '')`,
			nid, c.ID.String(), modelID, now.Unix(), front+"\x1f"+back, stripHTML(front), checksum(stripHTML(front))); err != nil {
			return nil, err
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO cards VALUES (?, ?, ?, 0, ?, -1, 0, 0, ?, 0, 0, 0, 0, 0, 0, 0, 0, '')`,
			nid, nid, deckIDs[c.Deck], now.Unix(), i+1); err != nil {
			return nil, err
		}
	}
	if err := conn.Close(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err == nil {
			_, err = w.Write(data)
		}
		return err
	}
	col, err := os.ReadFile(path) //nolint:gosec // our own temp file
	if err != nil {
		return nil, err
	}
	if err := add("collection.anki2", col); err != nil {
		return nil, err
	}
	m, _ := json.Marshal(media)
	if err := add("media", m); err != nil {
		return nil, err
	}
	for _, f := range files {
		data, err := os.ReadFile(f[1])
		if err != nil {
			return nil, err
		}
		if err := add(f[0], data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func deckJSON(id int64, name string, ms int64) map[string]any {
	return map[string]any{"id": id, "name": name, "mod": ms / 1000, "usn": -1, "lrnToday": []int{0, 0}, "revToday": []int{0, 0},
		"newToday": []int{0, 0}, "timeToday": []int{0, 0}, "collapsed": false, "desc": "Exported from Axon", "dyn": 0, "conf": 1,
		"extendNew": 10, "extendRev": 50}
}
