package watch

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"html"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Exports of the news (EP-75): the items a filter selects, in CSV, ODS,
// Markdown, a zip of feed notes, JSON, or PDF. Each says where it comes
// from.

// Attribution is written into every export and shared page.
const Attribution = "Exported from Axon Project (https://github.com/thoutmose/axon)"

// MaxExport bounds the news one export carries.
const MaxExport = 5000

// Formats maps each export format to its media type and file extension.
var Formats = map[string][2]string{
	"csv":  {"text/csv; charset=utf-8", "csv"},
	"ods":  {"application/vnd.oasis.opendocument.spreadsheet", "ods"},
	"md":   {"text/markdown; charset=utf-8", "md"},
	"zip":  {"application/zip", "zip"},
	"json": {"application/json", "json"},
	"pdf":  {"application/pdf", "pdf"},
}

// Export is what an export carries.
type Export struct {
	Title       string
	Description string // the filter, in words
	Items       []Item
	Topics      map[string]string // id → name
	At          time.Time
}

// LoadExport reads the news matching f, with the words describing f.
func LoadExport(ctx context.Context, q db.Querier, userID uuid.UUID, f Filter, title string, at time.Time) (Export, error) {
	e := Export{Title: title, At: at, Topics: map[string]string{}}
	topics, err := ListTopics(ctx, q, userID)
	if err != nil {
		return e, err
	}
	for _, t := range topics {
		e.Topics[t.ID] = t.Name
	}
	source := ""
	if f.SourceID != nil {
		if s, err := GetSource(ctx, q, userID, *f.SourceID); err == nil {
			source = s.Title
		}
	}
	f.Limit = MaxExport
	if e.Items, err = ListItems(ctx, q, userID, f); err != nil {
		return e, err
	}
	e.Description = Describe(f, e.Topics[f.TopicID], source)
	return e, nil
}

// Describe puts a filter into words: "Label: ai · Videos · 1 Oct 2026 to 14 Oct 2026".
func Describe(f Filter, topic, source string) string {
	var parts []string
	switch f.Status {
	case "", "all":
	default:
		parts = append(parts, strings.ToUpper(f.Status[:1])+f.Status[1:])
	}
	if f.Kind != "" {
		parts = append(parts, strings.ToUpper(f.Kind[:1])+f.Kind[1:]+"s")
	}
	if topic != "" {
		parts = append(parts, "Topic: "+topic)
	}
	if source != "" {
		parts = append(parts, "Source: "+source)
	}
	if f.Label != "" {
		parts = append(parts, "Label: "+f.Label)
	}
	if strings.TrimSpace(f.Query) != "" {
		parts = append(parts, fmt.Sprintf("Words: %q", strings.TrimSpace(f.Query)))
	}
	const day = "2 Jan 2006"
	switch {
	case f.From != nil && f.To != nil:
		parts = append(parts, f.From.Format(day)+" to "+f.To.Add(-time.Second).Format(day))
	case f.From != nil:
		parts = append(parts, "Since "+f.From.Format(day))
	case f.To != nil:
		parts = append(parts, "Until "+f.To.Add(-time.Second).Format(day))
	}
	if len(parts) == 0 {
		return "All news"
	}
	return strings.Join(parts, " · ")
}

// Write writes the export in a format of Formats.
func (e Export) Write(w io.Writer, format string) error {
	switch format {
	case "csv":
		return e.csv(w)
	case "ods":
		return e.ods(w)
	case "md":
		_, err := io.WriteString(w, e.Markdown())
		return err
	case "zip":
		return e.zip(w)
	case "json":
		return e.json(w)
	case "pdf":
		_, err := w.Write(e.pdf())
		return err
	}
	return InvalidError{"format is csv, ods, md, zip, json, pdf, or opml."}
}

// Filename is the download name: axon-tech-watch-2026-10-14.csv.
func (e Export) Filename(format string) string {
	return fmt.Sprintf("axon-tech-watch-%s.%s", e.At.Format(time.DateOnly), Formats[format][1])
}

var columns = []string{"Title", "Link", "Source", "Kind", "Author", "Published", "Status", "Labels", "Topics", "Reading minutes", "Also in", "Summary"}

func (e Export) row(it Item) []string {
	var topics []string
	for _, id := range it.TopicIDs {
		if n, ok := e.Topics[id]; ok {
			topics = append(topics, n)
		}
	}
	return []string{it.Title, it.URL, it.SourceTitle, it.Kind, it.Author, it.PublishedAt.UTC().Format(time.DateOnly), it.Status,
		strings.Join(it.Labels, "; "), strings.Join(topics, "; "), strconv.Itoa(it.Minutes), strings.Join(it.AlsoIn, "; "), it.Summary}
}

// spreadsheetSafe keeps a cell from being read as a formula when the CSV
// is opened in a spreadsheet (CSV injection).
func spreadsheetSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func (e Export) csv(w io.Writer) error {
	cw := csv.NewWriter(w)
	_ = cw.Write(columns)
	for _, it := range e.Items {
		row := e.row(it)
		for i := range row {
			row[i] = spreadsheetSafe(row[i])
		}
		_ = cw.Write(row)
	}
	// The attribution closes the file, so the header stays the first line.
	_ = cw.Write([]string{fmt.Sprintf("%s on %s. %s.", Attribution, e.At.UTC().Format(time.DateOnly), e.Description)})
	cw.Flush()
	return cw.Error()
}

// ods writes an OpenDocument spreadsheet: a title, the attribution, the
// filter, then one row per item, dates as dates and links as links.
func (e Export) ods(w io.Writer) error {
	esc := html.EscapeString
	var c strings.Builder
	c.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0" xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:number="urn:oasis:names:tc:opendocument:xmlns:datastyle:1.0" office:version="1.3">
<office:automatic-styles><number:date-style style:name="D"><number:year number:style="long"/><number:text>-</number:text><number:month number:style="long"/><number:text>-</number:text><number:day number:style="long"/></number:date-style><style:style style:name="d" style:family="table-cell" style:data-style-name="D"/><style:style style:name="b" style:family="table-cell"><style:text-properties fo:font-weight="bold"/></style:style><style:style style:name="t" style:family="table-cell"><style:text-properties fo:font-weight="bold" fo:font-size="14pt"/></style:style></office:automatic-styles>
<office:body><office:spreadsheet><table:table table:name="News">
`)
	cell := func(s, style string) {
		if style != "" {
			style = ` table:style-name="` + style + `"`
		}
		fmt.Fprintf(&c, `<table:table-cell office:value-type="string"%s><text:p>%s</text:p></table:table-cell>`, style, esc(s))
	}
	line := func(s, style string) {
		c.WriteString("<table:table-row>")
		cell(s, style)
		c.WriteString("</table:table-row>\n")
	}
	line(e.Title, "t")
	line(fmt.Sprintf("%s on %s", Attribution, e.At.UTC().Format(time.DateOnly)), "")
	line(fmt.Sprintf("%s · %d news", e.Description, len(e.Items)), "")
	c.WriteString("<table:table-row/>\n<table:table-row>")
	for _, h := range columns {
		cell(h, "b")
	}
	c.WriteString("</table:table-row>\n")
	for _, it := range e.Items {
		c.WriteString("<table:table-row>")
		for i, v := range e.row(it) {
			switch columns[i] {
			case "Link":
				fmt.Fprintf(&c, `<table:table-cell office:value-type="string"><text:p><text:a xlink:type="simple" xlink:href="%s">%s</text:a></text:p></table:table-cell>`, esc(v), esc(v))
			case "Published":
				fmt.Fprintf(&c, `<table:table-cell table:style-name="d" office:value-type="date" office:date-value="%s"><text:p>%s</text:p></table:table-cell>`, v, v)
			case "Reading minutes":
				fmt.Fprintf(&c, `<table:table-cell office:value-type="float" office:value="%s"><text:p>%s</text:p></table:table-cell>`, v, v)
			default:
				cell(v, "")
			}
		}
		c.WriteString("</table:table-row>\n")
	}
	c.WriteString("</table:table></office:spreadsheet></office:body></office:document-content>\n")

	zw := zip.NewWriter(w)
	// The media type comes first and uncompressed, as ODF requires.
	media := []byte(Formats["ods"][0])
	mt, err := zw.CreateRaw(&zip.FileHeader{Name: "mimetype", Method: zip.Store, CRC32: crc32.ChecksumIEEE(media),
		CompressedSize64: uint64(len(media)), UncompressedSize64: uint64(len(media))})
	if err != nil {
		return err
	}
	_, _ = mt.Write(media)
	files := []struct{ name, body string }{
		{"META-INF/manifest.xml", `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.3">
<manifest:file-entry manifest:full-path="/" manifest:media-type="application/vnd.oasis.opendocument.spreadsheet"/>
<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
<manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
</manifest:manifest>
`},
		{"meta.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" xmlns:dc="http://purl.org/dc/elements/1.1/" office:version="1.3">
<office:meta><meta:generator>Axon Project</meta:generator><dc:title>%s</dc:title><dc:description>%s</dc:description><meta:creation-date>%s</meta:creation-date></office:meta>
</office:document-meta>
`, esc(e.Title), esc(Attribution), e.At.UTC().Format("2006-01-02T15:04:05"))},
		{"content.xml", c.String()},
	}
	for _, f := range files {
		fw, err := zw.Create(f.name)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(fw, f.body)
	}
	return zw.Close()
}

// Markdown is the export as one readable page, grouped by day.
func (e Export) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n_%s · %d news · %s_\n", e.Title, e.Description, len(e.Items), e.At.UTC().Format("2 January 2006"))
	day := ""
	for _, it := range e.Items {
		if d := it.PublishedAt.UTC().Format("Monday 2 January 2006"); d != day {
			day = d
			fmt.Fprintf(&b, "\n## %s\n", d)
		}
		fmt.Fprintf(&b, "\n### [%s](%s)\n\n", strings.ReplaceAll(it.Title, "]", `\]`), it.URL)
		meta := []string{it.SourceTitle, it.Kind}
		if it.Minutes > 0 && it.Kind == "article" {
			meta = append(meta, fmt.Sprintf("%d min", it.Minutes))
		}
		if len(it.Labels) > 0 {
			meta = append(meta, "#"+strings.Join(it.Labels, " #"))
		}
		fmt.Fprintf(&b, "%s\n", strings.Join(meta, " · "))
		if it.Summary != "" {
			fmt.Fprintf(&b, "\n> %s\n", strings.ReplaceAll(it.Summary, "\n", "\n> "))
		}
	}
	fmt.Fprintf(&b, "\n---\n\n%s on %s.\n", Attribution, e.At.UTC().Format(time.DateOnly))
	return b.String()
}

// zip holds one feed note per item, in Feeds/<year>/<month>/ as Keep
// writes them, and the page of them all as README.md.
func (e Export) zip(w io.Writer) error {
	zw := zip.NewWriter(w)
	if err := zw.SetComment(Attribution); err != nil {
		return err
	}
	used := map[string]int{}
	for _, it := range e.Items {
		base := fmt.Sprintf("Feeds/%s/%s", it.PublishedAt.UTC().Format("2006/01"), notes.Slug(it.Title))
		used[base]++
		name := base + ".md"
		if n := used[base]; n > 1 {
			name = fmt.Sprintf("%s-%d.md", base, n)
		}
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(fw, FeedNote(it, e.Topics)+"\n\n"+Attribution+"\n")
	}
	fw, err := zw.Create("README.md")
	if err != nil {
		return err
	}
	_, _ = io.WriteString(fw, e.Markdown())
	return zw.Close()
}

func (e Export) json(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"generator":   Attribution,
		"title":       e.Title,
		"description": e.Description,
		"exported_at": e.At.UTC(),
		"count":       len(e.Items),
		"topics":      e.Topics,
		"items":       e.Items,
	})
}

func (e Export) pdf() []byte {
	blocks := []pdfBlock{
		{text: e.Title, font: bold, size: 20},
		{text: fmt.Sprintf("%s · %d news · %s", e.Description, len(e.Items), e.At.UTC().Format("2 January 2006")), font: italic, size: 10, grey: 0.45, before: 4},
		{text: Attribution, font: regular, size: 9, grey: 0.45, url: "https://github.com/thoutmose/axon"},
	}
	for _, it := range e.Items {
		meta := []string{it.SourceTitle, it.PublishedAt.UTC().Format("2 Jan 2006"), it.Kind}
		if len(it.Labels) > 0 {
			meta = append(meta, strings.Join(it.Labels, ", "))
		}
		blocks = append(blocks,
			pdfBlock{text: it.Title, font: bold, size: 12, url: it.URL, before: 14},
			pdfBlock{text: strings.Join(meta, " · "), font: regular, size: 8.5, grey: 0.45, before: 1})
		if it.Summary != "" {
			blocks = append(blocks, pdfBlock{text: cut(it.Summary, 600), font: regular, size: 10, before: 3})
		}
	}
	return PDF(e.Title, blocks, Attribution, e.At)
}
