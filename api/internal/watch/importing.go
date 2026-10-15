package watch

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Planned is a source to add, as an import file or a service lists it.
type Planned struct {
	URL      string
	Title    string
	Topics   []string // topic names; missing ones are created
	Schedule string
	FullText bool
}

// ImportResult says what an import did, line by line for what failed.
type ImportResult struct {
	Added    int            `json:"added"`
	Existing int            `json:"existing"`
	Failed   []ImportFailed `json:"failed"`
	Topics   int            `json:"topics_created"`
}

// ImportFailed is a line that was not imported, and why.
type ImportFailed struct {
	Line   int    `json:"line"`
	URL    string `json:"url"`
	Reason string `json:"reason"`
}

// MaxImport bounds one import.
const MaxImport = 2000

// CSVHeader is the normalised CSV format, also what the export writes.
var CSVHeader = []string{"url", "title", "topics", "schedule", "full_text"}

// ParseCSV reads the normalised CSV: url (required), title, topics
// (separated by | or ;), schedule (cron, default midnight), full_text (yes/no).
// The header line is required; columns may come in any order.
func ParseCSV(r io.Reader) ([]Planned, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	head, err := cr.Read()
	if err != nil {
		return nil, ValidationError{"The CSV file is empty or not CSV."}
	}
	col := map[string]int{}
	for i, h := range head {
		col[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))] = i
	}
	if _, ok := col["url"]; !ok {
		return nil, ValidationError{"The CSV needs a header line with a url column: url,title,topics,schedule,full_text"}
	}
	get := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	var out []Planned
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, ValidationError{fmt.Sprintf("The CSV is malformed: %v", err)}
		}
		if len(out) == MaxImport {
			return nil, ValidationError{fmt.Sprintf("Import up to %d sources at a time.", MaxImport)}
		}
		p := Planned{URL: get(rec, "url"), Title: get(rec, "title"), Schedule: get(rec, "schedule")}
		for _, t := range strings.FieldsFunc(get(rec, "topics"), func(r rune) bool { return r == '|' || r == ';' }) {
			if t = strings.TrimSpace(t); t != "" {
				p.Topics = append(p.Topics, t)
			}
		}
		switch strings.ToLower(get(rec, "full_text")) {
		case "yes", "true", "1", "oui":
			p.FullText = true
		}
		out = append(out, p)
	}
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Category string        `xml:"category,attr"`
	Children []opmlOutline `xml:"outline"`
}

// ParseOPML reads the OPML that Feedly, FreshRSS, Inoreader, and most
// readers export: folders become topics.
func ParseOPML(r io.Reader) ([]Planned, error) {
	var doc struct {
		Body struct {
			Outlines []opmlOutline `xml:"outline"`
		} `xml:"body"`
	}
	dec := xml.NewDecoder(io.LimitReader(r, 10<<20))
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		return nil, ValidationError{"That is not an OPML file."}
	}
	var out []Planned
	var walk func(list []opmlOutline, folder string) error
	walk = func(list []opmlOutline, folder string) error {
		for _, o := range list {
			name := firstOf(o.Title, o.Text)
			if o.XMLURL == "" {
				if err := walk(o.Children, name); err != nil {
					return err
				}
				continue
			}
			if len(out) == MaxImport {
				return ValidationError{fmt.Sprintf("Import up to %d sources at a time.", MaxImport)}
			}
			p := Planned{URL: o.XMLURL, Title: name}
			if folder != "" {
				p.Topics = append(p.Topics, folder)
			}
			for _, c := range strings.Split(o.Category, ",") {
				if c = strings.Trim(strings.TrimSpace(c), "/"); c != "" && !strings.Contains(c, "/") {
					p.Topics = append(p.Topics, c)
				}
			}
			out = append(out, p)
		}
		return nil
	}
	if err := walk(doc.Body.Outlines, ""); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ValidationError{"The OPML file lists no feeds."}
	}
	return out, nil
}

// FreshRSS lists the subscriptions of a FreshRSS (or any Google Reader API)
// account: its address, the user, and the API password set in its profile.
func FreshRSS(ctx context.Context, base, user, apiPassword string) ([]Planned, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if !strings.HasSuffix(base, "/api/greader.php") {
		base += "/api/greader.php"
	}
	form := url.Values{"Email": {user}, "Passwd": {apiPassword}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/accounts/ClientLogin", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, ValidationError{"That is not a FreshRSS address."}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)
	res, err := Client.Do(req)
	if err != nil {
		return nil, ValidationError{"FreshRSS could not be reached: " + err.Error()}
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, ValidationError{"FreshRSS refused the user name or API password (set one in FreshRSS: Profile → API management)."}
	}
	auth := ""
	for _, line := range strings.Split(string(body), "\n") {
		if v, ok := strings.CutPrefix(line, "Auth="); ok {
			auth = strings.TrimSpace(v)
		}
	}
	if auth == "" {
		return nil, ValidationError{"FreshRSS gave no session."}
	}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, base+"/reader/api/0/subscription/list?output=json", nil)
	req.Header.Set("Authorization", "GoogleLogin auth="+auth)
	req.Header.Set("User-Agent", UserAgent)
	res, err = Client.Do(req)
	if err != nil {
		return nil, ValidationError{"FreshRSS could not be reached: " + err.Error()}
	}
	defer func() { _ = res.Body.Close() }()
	var subs struct {
		Subscriptions []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			URL        string `json:"url"`
			Categories []struct {
				Label string `json:"label"`
			} `json:"categories"`
		} `json:"subscriptions"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 10<<20)).Decode(&subs); err != nil {
		return nil, ValidationError{"FreshRSS answered with something other than its subscription list."}
	}
	var out []Planned
	for _, s := range subs.Subscriptions[:min(len(subs.Subscriptions), MaxImport)] {
		p := Planned{URL: firstOf(s.URL, strings.TrimPrefix(s.ID, "feed/")), Title: s.Title}
		for _, c := range s.Categories {
			if c.Label != "" {
				p.Topics = append(p.Topics, c.Label)
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// Import adds planned sources: each is fetched by the worker within a
// minute (and discovered then when its address is a site, not a feed).
// Topics are matched by name, created when missing.
func Import(ctx context.Context, tx pgx.Tx, userID uuid.UUID, plan []Planned, origin string, now time.Time) (ImportResult, error) {
	res := ImportResult{Failed: []ImportFailed{}}
	topics, err := ListTopics(ctx, tx, userID)
	if err != nil {
		return res, err
	}
	byName := map[string]string{}
	for _, t := range topics {
		byName[strings.ToLower(t.Name)] = t.ID
	}
	for i, p := range plan {
		line := i + 2 // after the header
		fail := func(reason string) {
			res.Failed = append(res.Failed, ImportFailed{Line: line, URL: cut(p.URL, 200), Reason: reason})
		}
		raw := strings.TrimSpace(p.URL)
		if !strings.Contains(raw, "://") && raw != "" {
			raw = "https://" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || len(raw) > 2000 {
			fail("not a web address")
			continue
		}
		if known, ok := Known(raw); ok {
			raw = known
		}
		sched := firstOf(p.Schedule, DefaultSchedule)
		if _, err := ParseSchedule(sched); err != nil {
			fail("schedule: " + err.Error())
			continue
		}
		var ids []string
		for _, name := range p.Topics {
			name = strings.TrimSpace(name)
			if name == "" || utf8.RuneCountInString(name) > 80 {
				continue
			}
			id, ok := byName[strings.ToLower(name)]
			if !ok {
				id = uuid.Must(uuid.NewV7()).String()
				if _, err := tx.Exec(ctx, `INSERT INTO watch_topics (id, user_id, name) VALUES ($1, $2, $3)`, id, userID, name); err != nil {
					return res, err
				}
				byName[strings.ToLower(name)] = id
				res.Topics++
			}
			ids = append(ids, id)
		}
		title := cut(strings.TrimSpace(firstOf(p.Title, hostOf(raw))), 200)
		tag, err := tx.Exec(ctx, `INSERT INTO watch_sources (id, user_id, url, title, topic_ids, schedule, full_text, origin, next_fetch_at)
			VALUES ($1, $2, $3, $4, $5::uuid[], $6, $7, $8, $9) ON CONFLICT (user_id, url) DO NOTHING`,
			uuid.Must(uuid.NewV7()), userID, raw, title, nonNil(ids), sched, p.FullText, origin, now)
		if err != nil {
			return res, err
		}
		if tag.RowsAffected() == 0 {
			res.Existing++
			continue
		}
		res.Added++
	}
	return res, nil
}

// OPML writes the sources as OPML, folders by topic, for any reader.
func OPML(sources []Source, topics []Topic, title string) []byte {
	names := map[string]string{}
	for _, t := range topics {
		names[t.ID] = t.Name
	}
	folders := map[string][]Source{}
	var order []string
	for _, s := range sources {
		folder := ""
		if len(s.TopicIDs) > 0 {
			folder = names[s.TopicIDs[0]]
		}
		if _, ok := folders[folder]; !ok {
			order = append(order, folder)
		}
		folders[folder] = append(folders[folder], s)
	}
	var b bytes.Buffer
	esc := func(s string) string {
		var x bytes.Buffer
		_ = xml.EscapeText(&x, []byte(s))
		return strings.ReplaceAll(x.String(), `"`, "&quot;")
	}
	fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!-- Exported from Axon Project, https://github.com/thoutmose/axon -->\n<opml version=\"2.0\">\n<head><title>%s</title><docs>http://opml.org/spec2.opml</docs></head>\n<body>\n", esc(title))
	for _, folder := range order {
		indent := ""
		if folder != "" {
			fmt.Fprintf(&b, "<outline text=\"%s\" title=\"%s\">\n", esc(folder), esc(folder))
			indent = "  "
		}
		for _, s := range folders[folder] {
			fmt.Fprintf(&b, "%s<outline type=\"rss\" text=\"%s\" title=\"%s\" xmlUrl=\"%s\" htmlUrl=\"%s\"/>\n", indent, esc(s.Title), esc(s.Title), esc(s.URL), esc(s.SiteURL))
		}
		if folder != "" {
			b.WriteString("</outline>\n")
		}
	}
	b.WriteString("</body>\n</opml>\n")
	return b.Bytes()
}
