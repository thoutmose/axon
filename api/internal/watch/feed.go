// Package watch is the tech watch (EP-110): sources a person follows (RSS,
// Atom, JSON Feed, or a site, a Medium author, a Substack, a YouTube
// channel, a GitHub repository found behind them), fetched on a schedule,
// kept once however many sources carry the same story, labelled by local
// NLP, and read as feed notes. It is separate from the notes: a feed note
// becomes a real note only when the person keeps it.
package watch

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// Feed is a parsed feed.
type Feed struct {
	Title   string
	SiteURL string
	Items   []Entry
}

// Entry is one item of a feed, before Axon stores it.
type Entry struct {
	GUID       string
	URL        string
	Title      string
	Author     string
	Published  time.Time // zero when the feed gives no date
	Summary    string    // HTML or text, as the feed gives it
	Content    string    // HTML, when the feed carries the full article
	Markdown   string    // the article's page read by Axon, when the feed has an excerpt only
	Categories []string
	Image      string
	Audio      string // a podcast episode's enclosure
}

// ErrNotFeed means the document is not a feed Axon reads.
var ErrNotFeed = errors.New("not an RSS, Atom, or JSON feed")

// MaxEntries bounds what one fetch reads from a feed.
const MaxEntries = 300

// Parse reads RSS 2.0, RSS 1.0 (RDF), Atom, or JSON Feed.
func Parse(body []byte) (Feed, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return parseJSON(trimmed)
	}
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return Feed{}, ErrNotFeed
		}
		if err != nil {
			return Feed{}, errors.Join(ErrNotFeed, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch strings.ToLower(start.Name.Local) {
		case "rss", "rdf":
			var doc rssDoc
			if err := dec.DecodeElement(&doc, &start); err != nil {
				return Feed{}, errors.Join(ErrNotFeed, err)
			}
			return doc.feed(), nil
		case "feed":
			var doc atomFeed
			if err := dec.DecodeElement(&doc, &start); err != nil {
				return Feed{}, errors.Join(ErrNotFeed, err)
			}
			return doc.feed(), nil
		default:
			return Feed{}, ErrNotFeed
		}
	}
}

// --- RSS 2.0 and RSS 1.0 ---

type rssDoc struct {
	Channel struct {
		Title string    `xml:"title"`
		Link  []xmlLink `xml:"link"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
	Items []rssItem `xml:"item"` // RSS 1.0 puts items beside the channel
}

// xmlLink is RSS's <link>text</link> or an <atom:link href=…/> beside it.
type xmlLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Text string `xml:",chardata"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	Links       []xmlLink `xml:"link"`
	GUID        string    `xml:"guid"`
	About       string    `xml:"about,attr"`
	Description string    `xml:"description"`
	Encoded     string    `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	PubDate     string    `xml:"pubDate"`
	Date        string    `xml:"http://purl.org/dc/elements/1.1/ date"`
	Creator     string    `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Author      string    `xml:"author"`
	Categories  []string  `xml:"category"`
	Enclosure   struct {
		URL  string `xml:"url,attr"`
		Type string `xml:"type,attr"`
	} `xml:"enclosure"`
	Media struct {
		URL string `xml:"url,attr"`
	} `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

func pickLink(links []xmlLink) string {
	for _, l := range links {
		if t := strings.TrimSpace(l.Text); t != "" {
			return t
		}
	}
	for _, l := range links {
		if l.Href != "" && (l.Rel == "" || l.Rel == "alternate") {
			return l.Href
		}
	}
	return ""
}

func (d rssDoc) feed() Feed {
	f := Feed{Title: clean(d.Channel.Title), SiteURL: pickLink(d.Channel.Link)}
	items := d.Channel.Items
	if len(items) == 0 {
		items = d.Items
	}
	for _, it := range items[:min(len(items), MaxEntries)] {
		e := Entry{
			Title:      clean(it.Title),
			URL:        pickLink(it.Links),
			GUID:       strings.TrimSpace(it.GUID),
			Summary:    it.Description,
			Content:    it.Encoded,
			Author:     clean(firstOf(it.Creator, it.Author)),
			Published:  parseDate(firstOf(it.PubDate, it.Date)),
			Categories: cleanAll(it.Categories),
			Image:      it.Media.URL,
		}
		if e.GUID == "" {
			e.GUID = firstOf(it.About, e.URL)
		}
		if e.URL == "" && strings.HasPrefix(e.GUID, "http") {
			e.URL = e.GUID
		}
		switch t := it.Enclosure.Type; {
		case strings.HasPrefix(t, "audio/"):
			e.Audio = it.Enclosure.URL
		case strings.HasPrefix(t, "image/") && e.Image == "":
			e.Image = it.Enclosure.URL
		}
		f.Items = append(f.Items, e)
	}
	return f
}

// --- Atom ---

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",innerxml"`
}

// text returns HTML for html and xhtml content, escaped text otherwise.
func (t atomText) html() string {
	body := strings.TrimSpace(t.Body)
	switch t.Type {
	case "html":
		return html.UnescapeString(unwrapCDATA(body))
	case "xhtml":
		return body
	}
	return html.EscapeString(html.UnescapeString(unwrapCDATA(body)))
}

func unwrapCDATA(s string) string {
	if strings.HasPrefix(s, "<![CDATA[") && strings.HasSuffix(s, "]]>") {
		return s[9 : len(s)-3]
	}
	return s
}

type atomFeed struct {
	Title   atomText    `xml:"title"`
	Links   []xmlLink   `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string    `xml:"id"`
	Title     atomText  `xml:"title"`
	Links     []xmlLink `xml:"link"`
	Published string    `xml:"published"`
	Updated   string    `xml:"updated"`
	Summary   atomText  `xml:"summary"`
	Content   atomText  `xml:"content"`
	Authors   []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Categories []struct {
		Term  string `xml:"term,attr"`
		Label string `xml:"label,attr"`
	} `xml:"category"`
	Group struct { // YouTube puts the description in media:group
		Description string `xml:"description"`
		Thumbnail   struct {
			URL string `xml:"url,attr"`
		} `xml:"thumbnail"`
	} `xml:"http://search.yahoo.com/mrss/ group"`
}

func atomLink(links []xmlLink, rel string) string {
	for _, l := range links {
		if l.Rel == rel || (rel == "alternate" && l.Rel == "") {
			return l.Href
		}
	}
	return ""
}

func (d atomFeed) feed() Feed {
	f := Feed{Title: clean(stripTags(d.Title.html())), SiteURL: atomLink(d.Links, "alternate")}
	for _, en := range d.Entries[:min(len(d.Entries), MaxEntries)] {
		e := Entry{
			GUID:      strings.TrimSpace(en.ID),
			Title:     clean(stripTags(en.Title.html())),
			URL:       atomLink(en.Links, "alternate"),
			Published: parseDate(firstOf(en.Published, en.Updated)),
			Summary:   en.Summary.html(),
			Content:   en.Content.html(),
			Image:     en.Group.Thumbnail.URL,
		}
		if e.Summary == "" && en.Group.Description != "" {
			e.Summary = html.EscapeString(en.Group.Description)
		}
		if len(en.Authors) > 0 {
			e.Author = clean(en.Authors[0].Name)
		}
		for _, c := range en.Categories {
			e.Categories = append(e.Categories, clean(firstOf(c.Label, c.Term)))
		}
		if e.GUID == "" {
			e.GUID = e.URL
		}
		if e.Audio = atomLink(en.Links, "enclosure"); e.Audio != "" && !strings.Contains(e.Audio, ".mp3") && !strings.Contains(e.Audio, "audio") {
			e.Audio = ""
		}
		f.Items = append(f.Items, e)
	}
	return f
}

// --- JSON Feed 1.x ---

func parseJSON(body []byte) (Feed, error) {
	var doc struct {
		Version string `json:"version"`
		Title   string `json:"title"`
		Home    string `json:"home_page_url"`
		Items   []struct {
			ID          json.RawMessage `json:"id"`
			URL         string          `json:"url"`
			Title       string          `json:"title"`
			ContentHTML string          `json:"content_html"`
			ContentText string          `json:"content_text"`
			Summary     string          `json:"summary"`
			Published   string          `json:"date_published"`
			Modified    string          `json:"date_modified"`
			Image       string          `json:"image"`
			Tags        []string        `json:"tags"`
			Authors     []struct {
				Name string `json:"name"`
			} `json:"authors"`
			Author struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &doc); err != nil || !strings.Contains(doc.Version, "jsonfeed.org") {
		return Feed{}, ErrNotFeed
	}
	f := Feed{Title: clean(doc.Title), SiteURL: doc.Home}
	for _, it := range doc.Items[:min(len(doc.Items), MaxEntries)] {
		id := strings.Trim(string(it.ID), `"`)
		e := Entry{GUID: id, URL: it.URL, Title: clean(it.Title), Content: it.ContentHTML, Summary: it.Summary,
			Published: parseDate(firstOf(it.Published, it.Modified)), Image: it.Image, Categories: cleanAll(it.Tags)}
		if e.Content == "" && it.ContentText != "" {
			e.Content = "<p>" + html.EscapeString(it.ContentText) + "</p>"
		}
		e.Author = it.Author.Name
		if len(it.Authors) > 0 {
			e.Author = it.Authors[0].Name
		}
		if e.GUID == "" {
			e.GUID = e.URL
		}
		f.Items = append(f.Items, e)
	}
	return f, nil
}

// --- helpers ---

var dateLayouts = []string{
	time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC3339Nano,
	"Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST", "Mon, 02 Jan 2006 15:04 -0700",
	"2 Jan 2006 15:04:05 -0700", "Mon, 2 January 2006 15:04:05 MST", "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05",
	"2006-01-02 15:04:05", "2006-01-02",
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func firstOf(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

var spaces = regexp.MustCompile(`\s+`)

// clean decodes entities and collapses white space in a short text.
func clean(s string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(html.UnescapeString(unwrapCDATA(strings.TrimSpace(s))), " "))
}

func cleanAll(list []string) []string {
	var out []string
	for _, s := range list {
		if s = clean(s); s != "" && len(s) <= 100 {
			out = append(out, s)
		}
	}
	return out
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return tagRe.ReplaceAllString(s, " ") }
