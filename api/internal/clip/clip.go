// Package clip saves a web page's readable text as Markdown (EP-67), for a
// literature note made from a link shared to Axon. It fetches the page with
// netsafe's client (never on private networks unless allowed), keeps the
// main content (the <article>, <main>, or <body>, without scripts, menus,
// headers, footers, and forms), and writes headings, paragraphs, lists,
// quotes, code, and links as Markdown.
package clip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/thoutmose/axon/api/internal/netsafe"
)

// Page is a clipped page.
type Page struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Site     string `json:"site"`
	Markdown string `json:"markdown"`
}

// ErrPage says why a page could not be clipped, in words a person reads.
type ErrPage struct{ Msg string }

func (e ErrPage) Error() string { return e.Msg }

const (
	maxPage = 5 << 20
	maxText = 200_000
)

// Client fetches pages; tests replace it.
var Client = netsafe.Client(20 * time.Second)

// Fetch downloads and clips a page.
func Fetch(ctx context.Context, raw string) (Page, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "https" && (u.Scheme != "http" || !netsafe.AllowPrivate())) || u.User != nil {
		return Page{}, ErrPage{"Clip an https:// address."}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("Accept", "text/html, application/xhtml+xml;q=0.9")
	req.Header.Set("User-Agent", "Axon clipper (self-hosted)")
	res, err := Client.Do(req)
	if err != nil {
		if errors.Is(err, netsafe.ErrPrivate) {
			return Page{}, ErrPage{"That address is on a private network."}
		}
		return Page{}, ErrPage{"The page could not be reached."}
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return Page{}, ErrPage{fmt.Sprintf("The page answered %d.", res.StatusCode)}
	}
	if ct, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type")); ct != "text/html" && ct != "application/xhtml+xml" {
		return Page{}, ErrPage{"That address is not a web page (" + ct + ")."}
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxPage))
	if err != nil {
		return Page{}, ErrPage{"The page could not be read."}
	}
	p, err := Read(string(data), res.Request.URL)
	if err != nil {
		return Page{}, err
	}
	return p, nil
}

// Read clips HTML fetched from base.
func Read(doc string, base *url.URL) (Page, error) {
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return Page{}, ErrPage{"The page is not readable HTML."}
	}
	p := Page{URL: base.String(), Site: base.Hostname()}
	var article, main, body *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Title:
				if p.Title == "" {
					p.Title = strings.TrimSpace(text(n))
				}
			case atom.Meta:
				prop, content := attr(n, "property"), attr(n, "content")
				switch {
				case prop == "og:title" && content != "":
					p.Title = content
				case prop == "og:site_name" && content != "":
					p.Site = content
				}
			case atom.Article:
				if article == nil {
					article = n
				}
			case atom.Main:
				if main == nil {
					main = n
				}
			case atom.Body:
				body = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	content := article
	if content == nil {
		content = main
	}
	if content == nil {
		content = body
	}
	if content == nil {
		return Page{}, ErrPage{"The page has no text to clip."}
	}
	var b strings.Builder
	md(&b, content, base)
	p.Markdown = tidy(b.String())
	if r := []rune(p.Markdown); len(r) > maxText {
		p.Markdown = string(r[:maxText]) + "\n\n…"
	}
	p.Title = strings.Join(strings.Fields(p.Title), " ")
	if p.Title == "" {
		p.Title = p.Site
	}
	if strings.TrimSpace(p.Markdown) == "" {
		return Page{}, ErrPage{"The page has no text to clip."}
	}
	return p, nil
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// skipped elements carry no reading content.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Nav: true, atom.Header: true, atom.Footer: true,
	atom.Aside: true, atom.Form: true, atom.Button: true, atom.Svg: true, atom.Iframe: true, atom.Template: true, atom.Select: true,
}

// inline writes a run of inline content, collapsing white space.
func inline(b *strings.Builder, n *html.Node, base *url.URL) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			words := strings.Join(strings.Fields(c.Data), " ")
			if words == "" {
				if c.Data != "" && !strings.HasSuffix(b.String(), " ") {
					b.WriteByte(' ')
				}
				continue
			}
			if strings.TrimLeft(c.Data, " \t\n") != c.Data && !strings.HasSuffix(b.String(), " ") && b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(words)
			if strings.TrimRight(c.Data, " \t\n") != c.Data {
				b.WriteByte(' ')
			}
		case c.Type != html.ElementNode || skipped[c.DataAtom]:
		case c.DataAtom == atom.A:
			label := strings.Join(strings.Fields(text(c)), " ")
			href := attr(c, "href")
			if u, err := base.Parse(href); err == nil && (u.Scheme == "http" || u.Scheme == "https") && label != "" {
				fmt.Fprintf(b, "[%s](%s)", strings.ReplaceAll(label, "]", ")"), u.String())
			} else {
				b.WriteString(label)
			}
		case c.DataAtom == atom.Strong || c.DataAtom == atom.B:
			if t := strings.TrimSpace(text(c)); t != "" {
				fmt.Fprintf(b, "**%s**", t)
			}
		case c.DataAtom == atom.Em || c.DataAtom == atom.I:
			if t := strings.TrimSpace(text(c)); t != "" {
				fmt.Fprintf(b, "*%s*", t)
			}
		case c.DataAtom == atom.Code:
			fmt.Fprintf(b, "`%s`", text(c))
		case c.DataAtom == atom.Br:
			b.WriteString("\n")
		default:
			inline(b, c, base)
		}
	}
}

// md writes block content as Markdown.
func md(b *strings.Builder, n *html.Node, base *url.URL) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
				b.WriteString(strings.Join(strings.Fields(c.Data), " ") + " ")
			}
			continue
		}
		if skipped[c.DataAtom] {
			continue
		}
		switch c.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level := int(c.Data[1] - '0')
			fmt.Fprintf(b, "\n\n%s %s\n\n", strings.Repeat("#", max(2, level)), strings.Join(strings.Fields(text(c)), " "))
		case atom.P:
			b.WriteString("\n\n")
			inline(b, c, base)
			b.WriteString("\n\n")
		case atom.Li:
			b.WriteString("\n- ")
			inline(b, c, base)
		case atom.Ul, atom.Ol:
			md(b, c, base)
			b.WriteString("\n\n")
		case atom.Blockquote:
			var q strings.Builder
			inline(&q, c, base)
			fmt.Fprintf(b, "\n\n> %s\n\n", strings.TrimSpace(q.String()))
		case atom.Pre:
			fmt.Fprintf(b, "\n\n```\n%s\n```\n\n", strings.TrimRight(text(c), "\n"))
		default:
			md(b, c, base)
		}
	}
}

// tidy collapses blank lines and trailing spaces.
func tidy(s string) string {
	var out []string
	blank := true
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimRight(l, " \t")
		if strings.TrimSpace(l) == "" {
			if !blank {
				out = append(out, "")
			}
			blank = true
			continue
		}
		out = append(out, strings.TrimLeft(l, " "))
		blank = false
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
