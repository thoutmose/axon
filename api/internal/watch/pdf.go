package watch

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// A small PDF writer for exports: A4 pages, the standard Helvetica fonts
// (no font files to ship), WinAnsi text, wrapped paragraphs, clickable
// titles, and the attribution in every page's footer.

const (
	pageW, pageH = 595.0, 842.0 // A4 in points
	margin       = 50.0
	footerY      = 28.0
)

// helvetica holds the widths of ASCII 32 to 126 in thousandths of an em.
var helvetica = [95]int{278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584}

type pdfFont int

const (
	regular pdfFont = iota + 1
	bold
	italic
)

// width of s in points; bold is taken a little wider, so wrapping errs on
// the safe side.
func width(s string, f pdfFont, size float64) float64 {
	var w int
	for _, r := range s {
		if r >= 32 && r <= 126 {
			w += helvetica[r-32]
		} else {
			w += 556
		}
	}
	if f == bold {
		w = w * 108 / 100
	}
	return float64(w) * size / 1000
}

// wrap breaks text into lines no wider than max.
func wrap(text string, f pdfFont, size, max float64) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			for width(word, f, size) > max { // a word longer than the line
				n := len([]rune(word))
				for n > 1 && width(string([]rune(word)[:n]), f, size) > max {
					n--
				}
				if line != "" {
					lines = append(lines, line)
					line = ""
				}
				lines = append(lines, string([]rune(word)[:n]))
				word = string([]rune(word)[n:])
			}
			switch {
			case word == "":
			case line == "":
				line = word
			case width(line+" "+word, f, size) <= max:
				line += " " + word
			default:
				lines = append(lines, line)
				line = word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

var winAnsi = encoding.ReplaceUnsupported(charmap.Windows1252.NewEncoder())

// pdfString encodes text as a WinAnsi PDF literal string.
func pdfString(s string) string {
	b, err := winAnsi.Bytes([]byte(s))
	if err != nil {
		b = []byte(strings.Map(func(r rune) rune {
			if r > 126 {
				return '?'
			}
			return r
		}, s))
	}
	var out strings.Builder
	out.WriteByte('(')
	for _, c := range b {
		switch c {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteByte(c)
		case '\r', '\n', '\t':
			out.WriteByte(' ')
		default:
			out.WriteByte(c)
		}
	}
	out.WriteByte(')')
	return out.String()
}

// pdfBlock is a paragraph: its font, size, grey level, link, and the space
// above it.
type pdfBlock struct {
	text   string
	font   pdfFont
	size   float64
	grey   float64 // 0 black, 0.45 muted
	url    string
	before float64
}

type pdfLine struct {
	text  string
	font  pdfFont
	size  float64
	grey  float64
	url   string
	y     float64
	width float64
}

// PDF lays blocks out on pages and returns the document.
func PDF(title string, blocks []pdfBlock, footer string, at time.Time) []byte {
	var pages [][]pdfLine
	var page []pdfLine
	y := pageH - margin
	for _, bl := range blocks {
		lead := bl.size * 1.3
		lines := wrap(bl.text, bl.font, bl.size, pageW-2*margin)
		y -= bl.before
		for _, l := range lines {
			if y-lead < margin {
				pages = append(pages, page)
				page, y = nil, pageH-margin
			}
			y -= lead
			page = append(page, pdfLine{l, bl.font, bl.size, bl.grey, bl.url, y, width(l, bl.font, bl.size)})
		}
	}
	pages = append(pages, page)

	var objs []string // object n is objs[n-1]
	add := func(s string) int { objs = append(objs, s); return len(objs) }
	catalog := add("") // filled once the pages are known
	pagesObj := add("")
	add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique /Encoding /WinAnsiEncoding >>")
	info := add(fmt.Sprintf("<< /Title %s /Producer %s /Creator %s /CreationDate (D:%s) >>",
		pdfString(title), pdfString("Axon Project"), pdfString(footer), at.UTC().Format("20060102150405Z")))
	var kids []string
	for i, lines := range pages {
		var c strings.Builder
		var annots []string
		for _, l := range lines {
			fmt.Fprintf(&c, "BT %.2f g /F%d %.1f Tf %.2f %.2f Td %s Tj ET\n", l.grey, l.font, l.size, margin, l.y, pdfString(l.text))
			if l.url != "" {
				annots = append(annots, fmt.Sprintf("%d 0 R", add(fmt.Sprintf(
					"<< /Type /Annot /Subtype /Link /Border [0 0 0] /Rect [%.2f %.2f %.2f %.2f] /A << /S /URI /URI %s >> >>",
					margin, l.y-2, margin+l.width, l.y+l.size, pdfString(l.url)))))
			}
		}
		foot := fmt.Sprintf("%s · %d/%d", footer, i+1, len(pages))
		fmt.Fprintf(&c, "BT 0.45 g /F1 8 Tf %.2f %.2f Td %s Tj ET\n", margin, footerY, pdfString(foot))
		var z bytes.Buffer
		zw := zlib.NewWriter(&z)
		_, _ = zw.Write([]byte(c.String()))
		_ = zw.Close()
		content := add(fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", z.Len(), z.String()))
		pg := fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.0f %.0f] /Resources << /Font << /F1 3 0 R /F2 4 0 R /F3 5 0 R >> >> /Contents %d 0 R",
			pagesObj, pageW, pageH, content)
		if len(annots) > 0 {
			pg += " /Annots [" + strings.Join(annots, " ") + "]"
		}
		kids = append(kids, fmt.Sprintf("%d 0 R", add(pg+" >>")))
	}
	objs[catalog-1] = fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObj)
	objs[pagesObj-1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(kids))

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, catalog, info, xref)
	return out.Bytes()
}
