// Package imports brings data from other apps into Axon (EP-60): tasks
// from Todoist and TickTick, books from Goodreads, films from Letterboxd,
// money from Firefly III and bank OFX/QFX files, and pages from Notion.
// Each source is read by a pure parser; writing goes through the domain
// packages, so validation, events, and the vault mirror apply as usual,
// and importing the same file twice creates nothing twice.
package imports

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thoutmose/axon/api/internal/budget"
)

// Task is a task to import.
type Task struct {
	Title, Notes, Context string
	Due                   *time.Time
	Done                  bool
}

// Media is a book, film, or series to import.
type Media struct {
	Kind, Title, Creators, Status, Finished string
	Year, Rating                            int // Rating out of 10; 0: none
}

// Note is a Markdown note to import, and a file it embeds.
type Note struct{ Path, Content string }

// File is a picture a note embeds.
type File struct {
	Path string
	Data []byte
}

// ErrFormat says the file is not what the source exports.
type ErrFormat struct{ Msg string }

func (e ErrFormat) Error() string { return e.Msg }

// table reads a CSV with a header row into records keyed by column name.
func table(data []byte, skipUntil string) ([]map[string]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // a BOM
	if skipUntil != "" {
		// Some exports put a few lines of notes before the header.
		if i := bytes.Index(data, []byte(skipUntil)); i > 0 {
			if j := bytes.LastIndexByte(data[:i], '\n'); j >= 0 {
				data = data[j+1:]
			}
		}
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	head, err := r.Read()
	if err != nil {
		return nil, ErrFormat{"The file is not a CSV export."}
	}
	for i := range head {
		head[i] = strings.TrimSpace(head[i])
	}
	var out []map[string]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, ErrFormat{fmt.Sprintf("Line %d is not valid CSV.", len(out)+2)}
		}
		row := map[string]string{}
		for i, v := range rec {
			if i < len(head) {
				row[head[i]] = strings.TrimSpace(v)
			}
		}
		out = append(out, row)
		if len(out) > 10000 {
			return nil, ErrFormat{"Import at most 10,000 rows at a time."}
		}
	}
}

func need(rows []map[string]string, cols ...string) error {
	if len(rows) == 0 {
		return nil
	}
	for _, c := range cols {
		if _, ok := rows[0][c]; !ok {
			return ErrFormat{fmt.Sprintf("This is not the expected export: it has no %q column.", c)}
		}
	}
	return nil
}

func day(s string, layouts ...string) *time.Time {
	s = strings.TrimSpace(s)
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return &d
		}
	}
	return nil
}

var labelRe = regexp.MustCompile(`(^|\s)@([\p{L}\p{N}_-]+)`)

// Todoist reads a Todoist project export (CSV: TYPE, CONTENT, DESCRIPTION,
// DATE, …). Labels (@phone) become the context; a date in the export's
// own format (2026-10-14) becomes the due date, and a spoken one ("every
// monday") goes into the notes.
func Todoist(data []byte) ([]Task, error) {
	rows, err := table(data, "")
	if err != nil {
		return nil, err
	}
	if err := need(rows, "TYPE", "CONTENT"); err != nil {
		return nil, err
	}
	var out []Task
	for _, r := range rows {
		if !strings.EqualFold(r["TYPE"], "task") || r["CONTENT"] == "" {
			continue
		}
		t := Task{Title: r["CONTENT"], Notes: r["DESCRIPTION"]}
		if m := labelRe.FindStringSubmatch(t.Title); m != nil {
			t.Context = "@" + m[2]
			t.Title = strings.TrimSpace(labelRe.ReplaceAllString(t.Title, "$1"))
		}
		if d := day(r["DATE"], time.DateOnly, "2006-01-02T15:04:05", time.RFC3339); d != nil {
			t.Due = d
		} else if r["DATE"] != "" {
			t.Notes = strings.TrimSpace(t.Notes + "\n\nTodoist date: " + r["DATE"])
		}
		out = append(out, t)
	}
	return out, nil
}

// TickTick reads a TickTick backup (CSV with a few lines of notes before
// the header: Title, Content, Due Date, Tags, Status, …). Status 0 is open.
func TickTick(data []byte) ([]Task, error) {
	rows, err := table(data, `"Folder Name"`)
	if err != nil {
		return nil, err
	}
	if err := need(rows, "Title", "Status"); err != nil {
		return nil, err
	}
	var out []Task
	for _, r := range rows {
		if r["Title"] == "" {
			continue
		}
		t := Task{Title: r["Title"], Notes: r["Content"], Done: r["Status"] != "0" && r["Status"] != ""}
		t.Due = day(r["Due Date"], "2006-01-02T15:04:05-0700", time.RFC3339, time.DateOnly)
		if tags := strings.Split(r["Tags"], ","); strings.TrimSpace(tags[0]) != "" {
			t.Context = "@" + strings.ReplaceAll(strings.TrimSpace(tags[0]), " ", "-")
		}
		out = append(out, t)
	}
	return out, nil
}

// Goodreads reads a Goodreads library export: shelves become statuses,
// stars become a rating out of 10.
func Goodreads(data []byte) ([]Media, error) {
	rows, err := table(data, "")
	if err != nil {
		return nil, err
	}
	if err := need(rows, "Title", "Author", "Exclusive Shelf"); err != nil {
		return nil, err
	}
	var out []Media
	for _, r := range rows {
		m := Media{Kind: "book", Title: r["Title"], Creators: r["Author"]}
		switch r["Exclusive Shelf"] {
		case "read":
			m.Status = "done"
		case "currently-reading":
			m.Status = "doing"
		default:
			m.Status = "want"
		}
		if n, err := strconv.Atoi(r["My Rating"]); err == nil && n > 0 {
			m.Rating = n * 2
		}
		for _, y := range []string{r["Original Publication Year"], r["Year Published"]} {
			if n, err := strconv.Atoi(y); err == nil && n > 0 {
				m.Year = n
				break
			}
		}
		if d := day(r["Date Read"], "2006/01/02", time.DateOnly); d != nil {
			m.Finished = d.Format(time.DateOnly)
		}
		if m.Title != "" {
			out = append(out, m)
		}
	}
	return out, nil
}

// Letterboxd reads a Letterboxd export: the ZIP (watched, ratings,
// watchlist, diary) or one of its CSV files. A film appears once: rated
// if it was rated, finished on its last diary date.
func Letterboxd(data []byte) ([]Media, error) {
	files := map[string][]byte{}
	if zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		for _, f := range zr.File {
			name := path.Base(f.Name)
			if !strings.HasSuffix(name, ".csv") || f.UncompressedSize64 > 20<<20 || strings.Contains(f.Name, "deleted/") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			b, err := io.ReadAll(io.LimitReader(rc, 20<<20))
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
			files[name] = b
		}
	} else {
		files["single.csv"] = data
	}
	type film struct {
		Media
		order   int
		watched bool
	}
	films := map[string]*film{}
	var order int
	get := func(r map[string]string) *film {
		key := strings.ToLower(r["Name"]) + "|" + r["Year"]
		f := films[key]
		if f == nil {
			order++
			f = &film{Media: Media{Kind: "film", Title: r["Name"], Status: "done"}, order: order}
			f.Year, _ = strconv.Atoi(r["Year"])
			films[key] = f
		}
		return f
	}
	found := false
	for _, name := range []string{"watched.csv", "diary.csv", "ratings.csv", "watchlist.csv", "single.csv"} {
		b, ok := files[name]
		if !ok {
			continue
		}
		rows, err := table(b, "")
		if err != nil {
			return nil, err
		}
		if err := need(rows, "Name", "Year"); err != nil {
			if name == "single.csv" {
				return nil, err
			}
			continue
		}
		found = true
		for _, r := range rows {
			if r["Name"] == "" {
				continue
			}
			f := get(r)
			if name == "watchlist.csv" {
				if !f.watched {
					f.Status = "want"
				}
				continue
			}
			f.Status, f.watched = "done", true
			if v, err := strconv.ParseFloat(r["Rating"], 64); err == nil && v > 0 {
				f.Rating = int(math.Round(v * 2))
			}
			for _, d := range []string{r["Watched Date"], r["Date"]} {
				if t := day(d, time.DateOnly); t != nil && name != "watched.csv" && t.Format(time.DateOnly) > f.Finished {
					f.Finished = t.Format(time.DateOnly)
					break
				}
			}
		}
	}
	if !found {
		return nil, ErrFormat{"This is not a Letterboxd export: no watched, diary, ratings, or watchlist file."}
	}
	out := make([]Media, len(films))
	for _, f := range films {
		out[f.order-1] = f.Media
	}
	return out, nil
}

// Firefly reads a Firefly III transaction export: withdrawals are money
// out, deposits money in; transfers between own accounts are left out.
func Firefly(data []byte) ([]budget.Row, error) {
	rows, err := table(data, "")
	if err != nil {
		return nil, err
	}
	if err := need(rows, "type", "amount", "date"); err != nil {
		return nil, err
	}
	var out []budget.Row
	for i, r := range rows {
		cents, err := budget.ParseCents(r["amount"])
		if err != nil {
			continue
		}
		if cents < 0 {
			cents = -cents
		}
		payee := r["destination_name"]
		switch strings.ToLower(r["type"]) {
		case "withdrawal":
			cents = -cents
		case "deposit":
			payee = r["source_name"]
		default:
			continue
		}
		d := day(r["date"], time.RFC3339, "2006-01-02T15:04:05-07:00", time.DateOnly)
		if d == nil {
			continue
		}
		note := r["description"]
		if c := r["category"]; c != "" {
			note = strings.TrimSpace(note + " · " + c)
		}
		out = append(out, budget.Row{Line: i + 2, On: d.Format(time.DateOnly), Cents: cents, Payee: payee, Note: note})
	}
	return out, nil
}

var ofxTag = regexp.MustCompile(`(?is)<(DTPOSTED|TRNAMT|NAME|MEMO|PAYEE)>([^<\r\n]*)`)

// OFX reads a bank's OFX or QFX statement (SGML or XML): each STMTTRN is a
// transaction, with its date, signed amount, name, and memo.
func OFX(data []byte) ([]budget.Row, error) {
	text := string(data)
	if !strings.Contains(strings.ToUpper(text), "<OFX>") {
		return nil, ErrFormat{"This is not an OFX or QFX statement."}
	}
	var out []budget.Row
	parts := regexp.MustCompile(`(?i)<STMTTRN>`).Split(text, -1)
	for i, part := range parts[1:] {
		if j := strings.Index(strings.ToUpper(part), "</STMTTRN>"); j >= 0 {
			part = part[:j]
		}
		f := map[string]string{}
		for _, m := range ofxTag.FindAllStringSubmatch(part, -1) {
			f[strings.ToUpper(m[1])] = strings.TrimSpace(m[2])
		}
		cents, err := budget.ParseCents(f["TRNAMT"])
		raw := f["DTPOSTED"]
		if err != nil || len(raw) < 8 {
			continue
		}
		d := day(raw[:8], "20060102")
		if d == nil {
			continue
		}
		name := f["NAME"]
		if name == "" {
			name = f["PAYEE"]
		}
		out = append(out, budget.Row{Line: i + 1, On: d.Format(time.DateOnly), Cents: cents, Payee: name, Note: f["MEMO"]})
	}
	if len(out) == 0 {
		return nil, ErrFormat{"No transaction found in this statement."}
	}
	return out, nil
}

var (
	notionID   = regexp.MustCompile(`\s+[0-9a-f]{32}`)
	notionLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+\.md)\)`)
)

// cleanNotion drops the 32-character ids Notion adds to names.
func cleanNotion(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = strings.TrimSpace(notionID.ReplaceAllString(s, ""))
	}
	return strings.Join(parts, "/")
}

// Notion reads a Notion export (ZIP of Markdown and pictures) into notes
// under Notion/: ids are dropped from names, links between pages become
// [[wikilinks]], and pictures keep their place next to the pages.
func Notion(data []byte) ([]Note, []File, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, ErrFormat{"A Notion export is a ZIP file (Export → Markdown & CSV)."}
	}
	var notes []Note
	var files []File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || f.UncompressedSize64 > 10<<20 {
			continue
		}
		ext := strings.ToLower(path.Ext(f.Name))
		if ext != ".md" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" && ext != ".svg" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, 10<<20))
		_ = rc.Close()
		if err != nil {
			return nil, nil, err
		}
		p := "Notion/" + cleanNotion(strings.TrimPrefix(f.Name, "/"))
		if ext == ".md" {
			content := notionLink.ReplaceAllStringFunc(string(b), func(m string) string {
				sub := notionLink.FindStringSubmatch(m)
				return "[[" + sub[1] + "]]"
			})
			// Pictures are linked by their old, id-bearing, URL-encoded names.
			content = regexp.MustCompile(`\]\(([^)]+)\)`).ReplaceAllStringFunc(content, func(m string) string {
				inner := m[2 : len(m)-1]
				if strings.Contains(inner, "://") {
					return m
				}
				dec := strings.ReplaceAll(inner, "%20", " ")
				return "](" + strings.ReplaceAll(cleanNotion(dec), " ", "%20") + ")"
			})
			notes = append(notes, Note{Path: p, Content: content})
		} else {
			files = append(files, File{Path: p, Data: b})
		}
		if len(notes)+len(files) > 5000 {
			return nil, nil, ErrFormat{"Import at most 5,000 pages and pictures at a time."}
		}
	}
	if len(notes) == 0 {
		return nil, nil, ErrFormat{"No Markdown page in this ZIP: export from Notion as Markdown & CSV."}
	}
	return notes, files, nil
}
