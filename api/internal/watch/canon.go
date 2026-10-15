package watch

import (
	"crypto/sha256"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

// trackingParams are dropped from links: they name the campaign, not the page.
var trackingParams = []string{"fbclid", "gclid", "dclid", "mc_cid", "mc_eid", "igshid", "ref_src", "ref", "source", "s", "sk", "spm", "_hsenc", "_hsmi", "mkt_tok", "cmpid", "ncid",
	// and those of YouTube's share button, Facebook, LinkedIn, X, and Google (EP-123)
	"si", "feature", "pp", "mibextid", "trk", "rcm", "__twitter_impression", "gs_lcp", "ved", "ei", "msclkid", "yclid", "twclid", "wt_mc"}

// videoIDRe is a YouTube video id: 11 characters of base64url.
var videoIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// VideoID is the YouTube video a link points to, or "".
func VideoID(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	id := ""
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		switch p := strings.Split(strings.Trim(u.Path, "/"), "/"); {
		case len(p) == 1 && p[0] == "watch":
			id = u.Query().Get("v")
		case len(p) == 2 && (p[0] == "shorts" || p[0] == "embed" || p[0] == "live"):
			id = p[1]
		}
	}
	if !videoIDRe.MatchString(id) {
		return ""
	}
	return id
}

// linkInText finds links in Markdown or plain text.
var linkInText = regexp.MustCompile(`https?://[^\s<>()\[\]"']+`)

// CleanLinks rewrites every link in a text to its canonical form, without
// tracking parameters, keeping a sentence's final punctuation outside.
func CleanLinks(text string) string {
	return linkInText.ReplaceAllStringFunc(text, func(l string) string {
		trail := ""
		for strings.HasSuffix(l, ".") || strings.HasSuffix(l, ",") || strings.HasSuffix(l, ";") || strings.HasSuffix(l, ":") {
			trail = l[len(l)-1:] + trail
			l = l[:len(l)-1]
		}
		return Clean(l) + trail
	})
}

// Clean drops a link's trackers (utm_*, fbclid…) and fragment, and turns
// every form of a YouTube link into one; its scheme, host, and port stay,
// so it still leads where it led (a server on the LAN over http).
func Clean(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(raw)
	}
	if id := VideoID(raw); id != "" { // every form of a YouTube link, one video
		q := u.Query()
		u.Scheme, u.Host, u.Path = "https", "youtube.com", "/watch"
		keep := url.Values{"v": {id}}
		if t := q.Get("t"); t != "" {
			keep.Set("t", t)
		}
		u.RawQuery = keep.Encode()
	}
	u.Fragment, u.User = "", nil
	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "utm_") || slices.Contains(trackingParams, lk) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode() // sorted by key
	return u.String()
}

// Canonical normalises a link so the same article is recognised from any
// feed: Clean, then lower-case host without "www.", no trailing slash, and
// http as https.
func Canonical(raw string) string {
	u, err := url.Parse(Clean(raw))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(raw)
	}
	u.Scheme = "https"
	u.Host = strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	u.Host = strings.TrimSuffix(strings.TrimSuffix(u.Host, ":443"), ":80")
	if u.Path != "/" {
		u.Path = strings.TrimSuffix(path.Clean("/"+u.Path), "/")
	}
	if u.Path == "/" {
		u.Path = ""
	}
	u.RawPath = ""
	return u.String()
}

// URLHash identifies a canonical link.
func URLHash(raw string) []byte {
	h := sha256.Sum256([]byte(Canonical(raw)))
	return h[:16]
}
