package watch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/netsafe"
)

// Client fetches feeds and pages; it refuses private networks unless the
// operator allows them. Tests replace it.
var Client = netsafe.Client(30 * time.Second)

// UserAgent says who is asking, as feed publishers ask, in the product/version
// form bot filters expect: dev.to answers 403 to anything else.
const UserAgent = "Axon/1.0 (+https://github.com/thoutmose/axon)"

const maxBody = 10 << 20

// response is what a GET returned.
type response struct {
	status      int
	body        []byte
	etag, mod   string
	url         string // after redirects
	contentType string
	notModified bool
}

// get fetches a URL, conditionally when etag or mod are given.
func get(ctx context.Context, raw, etag, mod string) (response, error) {
	var r response
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && (u.Scheme != "http" || !netsafe.AllowPrivate())) {
		return r, l10n.Errorf("%q is not an https address", raw)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return r, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/feed+json, application/xml;q=0.9, text/xml;q=0.9, text/html;q=0.8, */*;q=0.5")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if mod != "" {
		req.Header.Set("If-Modified-Since", mod)
	}
	res, err := Client.Do(req)
	if err != nil {
		if errors.Is(err, netsafe.ErrPrivate) {
			return r, l10n.Errorf("that address is on a private network")
		}
		return r, err
	}
	defer func() { _ = res.Body.Close() }()
	r.status, r.etag, r.mod, r.url = res.StatusCode, res.Header.Get("ETag"), res.Header.Get("Last-Modified"), res.Request.URL.String()
	r.contentType, _, _ = mime.ParseMediaType(res.Header.Get("Content-Type"))
	if res.StatusCode == http.StatusNotModified {
		r.notModified = true
		return r, nil
	}
	if res.StatusCode != http.StatusOK {
		return r, l10n.Errorf("%s answered %d", u.Host, res.StatusCode)
	}
	r.body, err = io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return r, err
	}
	if len(r.body) > maxBody {
		return r, l10n.Errorf("the feed is larger than 10 MB")
	}
	return r, nil
}

var (
	ytChannel  = regexp.MustCompile(`^/channel/(UC[\w-]{20,30})`)
	ytChannelI = regexp.MustCompile(`"(?:channelId|externalId)":"(UC[\w-]{20,30})"`)
	ghRepo     = regexp.MustCompile(`^/([\w.-]+)/([\w.-]+)/?$`)
	ghUser     = regexp.MustCompile(`^/([\w.-]+)/?$`)
	subreddit  = regexp.MustCompile(`^/r/([\w]+)`)
)

// Known returns the feed of a page on a service whose feeds follow a known
// pattern, without a request: Medium, Substack, YouTube, GitHub, Reddit,
// dev.to, Hacker News. ok is false for anything else.
func Known(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	p := strings.TrimSuffix(u.Path, "/")
	switch {
	case host == "medium.com":
		if strings.HasPrefix(p, "/feed/") {
			return "https://medium.com" + p, true
		}
		seg := strings.Split(strings.TrimPrefix(p, "/"), "/")[0]
		if seg == "" {
			return "", false
		}
		return "https://medium.com/feed/" + seg, true // an author (@name) or a publication
	case strings.HasSuffix(host, ".medium.com"):
		return "https://" + host + "/feed", true
	case strings.HasSuffix(host, ".substack.com"):
		return "https://" + host + "/feed", true
	case host == "youtube.com" || host == "m.youtube.com":
		if m := ytChannel.FindStringSubmatch(p); m != nil {
			return "https://www.youtube.com/feeds/videos.xml?channel_id=" + m[1], true
		}
		if list := u.Query().Get("list"); list != "" && p == "/playlist" {
			return "https://www.youtube.com/feeds/videos.xml?playlist_id=" + url.QueryEscape(list), true
		}
	case host == "github.com":
		if m := ghRepo.FindStringSubmatch(p); m != nil {
			return "https://github.com/" + m[1] + "/" + m[2] + "/releases.atom", true
		}
		if m := ghUser.FindStringSubmatch(p); m != nil {
			return "https://github.com/" + m[1] + ".atom", true
		}
	case host == "reddit.com" || host == "old.reddit.com":
		if m := subreddit.FindStringSubmatch(p); m != nil {
			return "https://www.reddit.com/r/" + m[1] + "/.rss", true
		}
	case host == "dev.to":
		if user := strings.Trim(p, "/"); user != "" && !strings.Contains(user, "/") {
			return "https://dev.to/feed/" + user, true
		}
	case host == "news.ycombinator.com":
		return "https://news.ycombinator.com/rss", true
	}
	return "", false
}

// commonPaths are tried when a page advertises no feed.
var commonPaths = []string{"/feed", "/rss", "/rss.xml", "/atom.xml", "/feed.xml", "/index.xml", "/feed.json"}

// Found is a feed Discover found, read once.
type Found struct {
	URL  string
	Feed Feed
}

// Discover finds the feed behind what a person typed: a feed's address, a
// site, a Medium author, a Substack, a YouTube channel or handle, a GitHub
// repository, a subreddit.
func Discover(ctx context.Context, input string) (Found, error) {
	input = strings.TrimSpace(input)
	if !strings.Contains(input, "://") {
		input = "https://" + input
	}
	if feedURL, ok := Known(input); ok {
		return readFeed(ctx, feedURL)
	}
	r, err := get(ctx, input, "", "")
	if err != nil {
		return Found{}, err
	}
	if f, err := Parse(r.body); err == nil {
		return Found{URL: r.url, Feed: f}, nil
	}
	base, _ := url.Parse(r.url)
	// A YouTube handle's page names its channel.
	if host := strings.TrimPrefix(base.Hostname(), "www."); host == "youtube.com" {
		if m := ytChannelI.FindSubmatch(r.body); m != nil {
			return readFeed(ctx, "https://www.youtube.com/feeds/videos.xml?channel_id="+string(m[1]))
		}
	}
	for _, link := range Alternates(r.body, base) {
		if found, err := readFeed(ctx, link); err == nil {
			return found, nil
		}
	}
	for _, p := range commonPaths {
		if found, err := readFeed(ctx, base.Scheme+"://"+base.Host+p); err == nil {
			return found, nil
		}
	}
	return Found{}, l10n.Errorf("paste the feed's address, or the site's home page")
}

func readFeed(ctx context.Context, feedURL string) (Found, error) {
	r, err := get(ctx, feedURL, "", "")
	if err != nil {
		return Found{}, err
	}
	f, err := Parse(r.body)
	if err != nil {
		return Found{}, err
	}
	return Found{URL: r.url, Feed: f}, nil
}

var feedTypes = map[string]bool{"application/rss+xml": true, "application/atom+xml": true, "application/feed+json": true, "application/json": true, "application/rdf+xml": true}

// Alternates lists the feeds an HTML page advertises with <link rel="alternate">.
func Alternates(page []byte, base *url.URL) []string {
	var out []string
	z := html.NewTokenizer(bytes.NewReader(page))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return out
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			if string(name) == "body" {
				return out
			}
			if string(name) != "link" || !hasAttr {
				continue
			}
			var rel, typ, href string
			for {
				k, v, more := z.TagAttr()
				switch strings.ToLower(string(k)) {
				case "rel":
					rel = strings.ToLower(string(v))
				case "type":
					typ = strings.ToLower(string(v))
				case "href":
					href = string(v)
				}
				if !more {
					break
				}
			}
			if strings.Contains(rel, "alternate") && feedTypes[typ] && href != "" {
				if u, err := base.Parse(href); err == nil {
					out = append(out, u.String())
				}
			}
		}
	}
}
