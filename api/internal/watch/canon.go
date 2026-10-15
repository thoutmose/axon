package watch

import (
	"crypto/sha256"
	"net/url"
	"path"
	"slices"
	"strings"
)

// trackingParams are dropped from links: they name the campaign, not the page.
var trackingParams = []string{"fbclid", "gclid", "dclid", "mc_cid", "mc_eid", "igshid", "ref_src", "ref", "source", "s", "sk", "spm", "_hsenc", "_hsmi", "mkt_tok", "cmpid", "ncid"}

// Canonical normalises a link so the same article is recognised from any
// feed: lower-case host without "www.", no tracking parameters (utm_*,
// fbclid…), no fragment, sorted query, no trailing slash, http as https.
func Canonical(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(raw)
	}
	u.Scheme = "https"
	u.Host = strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	u.Host = strings.TrimSuffix(strings.TrimSuffix(u.Host, ":443"), ":80")
	u.Fragment, u.User = "", nil
	q := u.Query()
	for k := range q {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "utm_") || slices.Contains(trackingParams, lk) {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode() // sorted by key
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
