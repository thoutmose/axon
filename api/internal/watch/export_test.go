package watch

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func sampleExport(n int) Export {
	at := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	e := Export{Title: "Tech watch", Description: "Label: ai", At: at, Topics: map[string]string{"t1": "AI"}}
	for i := range n {
		e.Items = append(e.Items, Item{
			Title: fmt.Sprintf("Story %d: “quotes” & <tags> café", i), URL: fmt.Sprintf("https://example.org/%d", i),
			SourceTitle: "Example", Kind: "article", PublishedAt: at.Add(-time.Duration(i) * 30 * time.Hour),
			Labels: []string{"ai", "llm"}, TopicIDs: []string{"t1"}, Minutes: 3, Status: "unread",
			Summary: strings.Repeat("A summary sentence about models and agents. ", 8),
		})
	}
	return e
}

func write(t *testing.T, e Export, format string) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := e.Write(&b, format); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestExportCSV(t *testing.T) {
	e := sampleExport(2)
	e.Items[0].Title = "=HYPERLINK(\"http://evil\")"
	if _, err := csv.NewReader(bytes.NewReader(write(t, e, "csv"))).ReadAll(); err == nil {
		t.Fatal("rows of different lengths should not parse strictly") // the last row is the attribution alone
	}
	r := csv.NewReader(bytes.NewReader(write(t, e, "csv")))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "Title" || rows[1][0] != `'=HYPERLINK("http://evil")` || rows[1][8] != "AI" || len(rows) != 4 {
		t.Fatalf("rows = %q", rows)
	}
	if last := rows[3][0]; !strings.HasPrefix(last, Attribution) || !strings.Contains(last, "Label: ai") {
		t.Fatalf("attribution = %q", last)
	}
}

func TestExportODS(t *testing.T) {
	out := write(t, sampleExport(3), "ods")
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	if f := zr.File[0]; f.Name != "mimetype" || f.Method != zip.Store {
		t.Fatalf("first entry = %s (method %d)", f.Name, f.Method)
	}
	if !bytes.Contains(out[:100], []byte("mimetypeapplication/vnd.oasis.opendocument.spreadsheet")) {
		t.Fatalf("the media type is not readable at the start of the file: %q", out[:100])
	}
	for _, f := range zr.File {
		rc, _ := f.Open()
		body, _ := io.ReadAll(rc)
		_ = rc.Close()
		if strings.HasSuffix(f.Name, ".xml") {
			d := xml.NewDecoder(bytes.NewReader(body))
			for {
				if _, err := d.Token(); errors.Is(err, io.EOF) {
					break
				} else if err != nil {
					t.Fatalf("%s is not XML: %v", f.Name, err)
				}
			}
		}
		if f.Name == "content.xml" && (!bytes.Contains(body, []byte(Attribution)) || !bytes.Contains(body, []byte(`office:date-value="2026-10-14"`)) ||
			!bytes.Contains(body, []byte("&lt;tags&gt;")) || bytes.Count(body, []byte("<table:table-row>")) != 3+1+3) {
			t.Fatalf("content.xml = %s", body)
		}
	}
}

func TestExportMarkdownZipJSON(t *testing.T) {
	e := sampleExport(3)
	md := string(write(t, e, "md"))
	if !strings.HasPrefix(md, "# Tech watch\n") || !strings.Contains(md, "## Wednesday 14 October 2026") || !strings.Contains(md, "## Tuesday 13 October 2026") ||
		!strings.Contains(md, "### [Story 0") || !strings.Contains(md, "#ai #llm") || !strings.Contains(md, Attribution) {
		t.Fatalf("markdown = %s", md)
	}

	e.Items[2].Title = e.Items[1].Title // two notes with one name
	e.Items[2].PublishedAt = e.Items[1].PublishedAt
	out := write(t, e, "zip")
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	if len(names) != 4 || names[3] != "README.md" || !strings.HasPrefix(names[0], "Feeds/2026/10/") || names[1]+"" == names[2] || zr.Comment != Attribution {
		t.Fatalf("zip = %v, %q", names, zr.Comment)
	}
	rc, _ := zr.File[0].Open()
	note, _ := io.ReadAll(rc)
	if !bytes.HasPrefix(note, []byte("---\ntype: feed\n")) || !bytes.Contains(note, []byte(Attribution)) {
		t.Fatalf("note = %s", note)
	}

	var doc struct {
		Generator string
		Count     int
		Items     []struct{ Kind string }
	}
	if err := json.Unmarshal(write(t, e, "json"), &doc); err != nil || doc.Generator != Attribution || doc.Count != 3 || doc.Items[0].Kind != "article" {
		t.Fatalf("json = %+v, %v", doc, err)
	}
	if err := e.Write(io.Discard, "docx"); err == nil {
		t.Fatal("an unknown format was accepted")
	}
}

// The PDF's cross-reference table points at its objects, every page says
// where it comes from, and titles are links.
func TestExportPDF(t *testing.T) {
	out := write(t, sampleExport(40), "pdf")
	if !bytes.HasPrefix(out, []byte("%PDF-1.4\n")) || !bytes.HasSuffix(out, []byte("%%EOF\n")) {
		t.Fatal("not a PDF")
	}
	m := regexp.MustCompile(`startxref\n(\d+)\n`).FindSubmatch(out)
	xref, _ := strconv.Atoi(string(m[1]))
	if !bytes.HasPrefix(out[xref:], []byte("xref\n")) {
		t.Fatal("startxref does not point at the table")
	}
	entries := regexp.MustCompile(`(\d{10}) 00000 n `).FindAllSubmatch(out[xref:], -1)
	for i, e := range entries {
		off, _ := strconv.Atoi(string(e[1]))
		if want := fmt.Sprintf("%d 0 obj\n", i+1); !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Fatalf("object %d is not at %d", i+1, off)
		}
	}
	pages := bytes.Count(out, []byte("/Type /Page /Parent"))
	if pages < 3 || bytes.Count(out, []byte("/S /URI")) < 41 {
		t.Fatalf("%d pages, %d links", pages, bytes.Count(out, []byte("/S /URI")))
	}
	streams := regexp.MustCompile(`(?s)stream\n(.*?)\nendstream`).FindAllSubmatch(out, -1)
	footers := 0
	for _, s := range streams {
		zr, err := zlib.NewReader(bytes.NewReader(s[1]))
		if err != nil {
			t.Fatal(err)
		}
		text, _ := io.ReadAll(zr)
		if bytes.Contains(text, []byte("(Exported from Axon Project \\(https://github.com/thoutmose/axon\\) \xb7 ")) {
			footers++
		}
		if bytes.Contains(text, []byte("caf\xe9")) == false && footers == 1 {
			t.Fatalf("WinAnsi text missing: %s", text[:200])
		}
	}
	if footers != pages {
		t.Fatalf("%d footers on %d pages", footers, pages)
	}
}

func TestWrapAndDescribe(t *testing.T) {
	for _, l := range wrap(strings.Repeat("word ", 200)+strings.Repeat("x", 300), regular, 10, 200) {
		if width(l, regular, 10) > 200 {
			t.Fatalf("line too wide: %q", l)
		}
	}
	from, to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	got := Describe(Filter{Status: "saved", Kind: "video", Label: "ai", From: &from, To: &to}, "AI", "Example")
	if got != "Saved · Videos · Topic: AI · Source: Example · Label: ai · 1 Oct 2026 to 14 Oct 2026" {
		t.Fatalf("describe = %q", got)
	}
	if Describe(Filter{}, "", "") != "All news" {
		t.Fatal("empty filter")
	}
	for url, want := range map[string]string{"https://www.youtube.com/watch?v=x": "video", "https://youtu.be/x": "video", "https://example.org/youtube.com/": "article"} {
		if got := kindOf(url, ""); got != want {
			t.Fatalf("kindOf(%s) = %s", url, got)
		}
	}
	if kindOf("https://youtu.be/x", "https://cdn/x.mp3") != "podcast" {
		t.Fatal("audio makes a podcast")
	}
}
