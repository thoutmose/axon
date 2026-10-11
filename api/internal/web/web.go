// Package web fetches public learning pages for agents (EP-27). A tool that
// can reach any URL would let an injected instruction send data out in a
// query string, so fetching is limited to an allowlist of educational sites
// (redirects included), refuses private addresses, honours robots.txt, and
// returns only readable text.
package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// UserAgent identifies Axon to sites and their robots.txt.
const UserAgent = "AxonLearningAgent/1.0 (+self-hosted; reads pages for its owner)"

// Allow lists the hosts agents may read; a host also allows its subdomains.
var Allow = []string{
	"wikipedia.org", "wikibooks.org", "developer.mozilla.org", "ocw.mit.edu", "openlearninglibrary.mit.edu",
	"www.coursera.org", "www.edx.org", "www.khanacademy.org", "arxiv.org", "docs.python.org", "go.dev",
	"pkg.go.dev", "doc.rust-lang.org", "www.freecodecamp.org", "openstax.org", "www.gutenberg.org", "ossu.dev",
}

// Test hooks: plain HTTP and loopback addresses are refused unless allowed.
var (
	AllowHTTP    = false
	AllowPrivate = false
)

// Errors the agent tool turns into messages for the model.
var (
	ErrNotAllowed = errors.New("site not on the allowlist")
	ErrRobots     = errors.New("robots.txt asks agents not to read this page")
)

// Allowed reports whether a URL may be fetched.
func Allowed(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("%w: not a valid URL", ErrNotAllowed)
	}
	if u.Scheme != "https" && (!AllowHTTP || u.Scheme != "http") {
		return nil, fmt.Errorf("%w: only https", ErrNotAllowed)
	}
	host := strings.ToLower(u.Hostname())
	for _, a := range Allow {
		if host == a || strings.HasSuffix(host, "."+a) {
			return u, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotAllowed, host)
}

// dialer refuses private, loopback, and link-local addresses.
func dialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 10 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !AllowPrivate && (ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsUnspecified()) {
				return nil, fmt.Errorf("%w: private address", ErrNotAllowed)
			}
		}
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

var client = &http.Client{
	Timeout:   20 * time.Second,
	Transport: &http.Transport{DialContext: dialer(), ResponseHeaderTimeout: 15 * time.Second},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		_, err := Allowed(req.URL.String())
		return err
	},
}

var (
	robotsMu    sync.Mutex
	robotsCache = map[string]robots{}
)

type robots struct {
	disallow []string
	expires  time.Time
}

// allowedByRobots reads (and caches for an hour) the site's robots.txt rules
// for "*" and for Axon's agent.
func allowedByRobots(ctx context.Context, u *url.URL) bool {
	key := u.Scheme + "://" + u.Host
	robotsMu.Lock()
	r, ok := robotsCache[key]
	robotsMu.Unlock()
	if !ok || time.Now().After(r.expires) {
		r = robots{expires: time.Now().Add(time.Hour)}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, key+"/robots.txt", nil)
		req.Header.Set("User-Agent", UserAgent)
		if res, err := client.Do(req); err == nil {
			body, _ := io.ReadAll(io.LimitReader(res.Body, 512<<10))
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				r.disallow = parseRobots(string(body))
			}
		}
		robotsMu.Lock()
		robotsCache[key] = r
		robotsMu.Unlock()
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	for _, d := range r.disallow {
		if strings.HasPrefix(path, d) {
			return false
		}
	}
	return true
}

func parseRobots(txt string) []string {
	var out []string
	applies := false
	for _, line := range strings.Split(txt, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch k {
		case "user-agent":
			applies = v == "*" || strings.HasPrefix(strings.ToLower(UserAgent), strings.ToLower(v))
		case "disallow":
			if applies && v != "" {
				out = append(out, v)
			}
		}
	}
	return out
}

var (
	dropBlocks = func() []*regexp.Regexp {
		var out []*regexp.Regexp
		for _, t := range []string{"head", "script", "style", "noscript", "svg", "nav", "footer", "header"} {
			out = append(out, regexp.MustCompile(`(?is)<`+t+`\b.*?</\s*`+t+`\s*>`))
		}
		return out
	}()
	blockTags  = regexp.MustCompile(`(?i)</?(p|div|br|li|ul|ol|h[1-6]|tr|table|section|article|pre|blockquote|dd|dt)\b[^>]*>`)
	tags       = regexp.MustCompile(`(?s)<[^>]*>`)
	spaces     = regexp.MustCompile(`[ \t\r\f\v]+`)
	lineEdges  = regexp.MustCompile(` ?\n ?`)
	blankLines = regexp.MustCompile(`\n\s*\n+`)
	titleRE    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	entities   = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&nbsp;", " ")
)

// Page is what an agent sees of a web page.
type Page struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Fetch reads an allowed page and returns its title and up to 12,000
// characters of text.
func Fetch(ctx context.Context, raw string) (Page, error) {
	u, err := Allowed(raw)
	if err != nil {
		return Page{}, err
	}
	if !allowedByRobots(ctx, u) {
		return Page{}, ErrRobots
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Page{}, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,text/plain;q=0.9")
	res, err := client.Do(req)
	if err != nil {
		if errors.Is(err, ErrNotAllowed) {
			return Page{}, err
		}
		return Page{}, fmt.Errorf("the page did not load: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return Page{}, fmt.Errorf("the page answered HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Page{}, err
	}
	html := string(body)
	p := Page{URL: res.Request.URL.String()}
	if m := titleRE.FindStringSubmatch(html); m != nil {
		p.Title = strings.TrimSpace(entities.Replace(tags.ReplaceAllString(m[1], "")))
	}
	text := html
	for _, re := range dropBlocks {
		text = re.ReplaceAllString(text, " ")
	}
	text = blockTags.ReplaceAllString(text, "\n")
	text = tags.ReplaceAllString(text, "")
	text = entities.Replace(text)
	text = spaces.ReplaceAllString(text, " ")
	text = lineEdges.ReplaceAllString(text, "\n")
	text = strings.TrimSpace(blankLines.ReplaceAllString(text, "\n"))
	if r := []rune(text); len(r) > 12000 {
		text = string(r[:12000])
	}
	p.Text = text
	return p, nil
}
