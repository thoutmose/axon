// Package l10n writes what the API says to people in their language (EP-87):
// English, or French when the request's Accept-Language prefers it. English
// strings are the keys; one missing from the French catalog stays English.
package l10n

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ctxKey struct{}

// FromRequest is "fr" when the request prefers French to English, else "en".
func FromRequest(r *http.Request) string {
	best, bestQ := "en", -1.0
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q = f
			}
		}
		lang := strings.ToLower(strings.SplitN(tag, "-", 2)[0])
		if (lang == "fr" || lang == "en") && q > 0 && q > bestQ {
			best, bestQ = lang, q
		}
	}
	return best
}

// Middleware puts the language in the request's context and says it in
// Content-Language, which httpx.Problem reads; Vary keeps caches apart.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := FromRequest(r)
		w.Header().Set("Content-Language", lang)
		w.Header().Add("Vary", "Accept-Language")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, lang)))
	})
}

// Lang is the request's language, "en" outside a request.
func Lang(ctx context.Context) string {
	if l, ok := ctx.Value(ctxKey{}).(string); ok {
		return l
	}
	return "en"
}

// T translates s for the request in ctx; with arguments, s is a format.
func T(ctx context.Context, s string, args ...any) string { return Tr(Lang(ctx), s, args...) }

// Tr translates s into lang; with arguments, s is a format.
func Tr(lang, s string, args ...any) string {
	if lang == "fr" {
		if f, ok := catalog[s]; ok {
			s = f
		} else if len(args) == 0 {
			s = patterns.translate(s, 1)
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// patterns translates messages formatted before they reach Tr, such as
// ValidationError{fmt.Sprintf("Line %d is not valid CSV.", n)}: each catalog
// format becomes a regular expression whose verbs capture the arguments.
var patterns formats

type formats struct {
	mu   sync.Mutex
	list []format
}

type format struct {
	re    *regexp.Regexp
	kinds []byte // each verb's letter, in order
	fr    string
}

var verb = regexp.MustCompile(`%[dsqvw]`)

func (p *formats) all() []format {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.list != nil {
		return p.list
	}
	p.list = []format{}
	for en, fr := range catalog {
		locs := verb.FindAllStringIndex(en, -1)
		if len(locs) == 0 || !sameVerbs(en, fr) {
			continue
		}
		var b strings.Builder
		kinds := make([]byte, len(locs))
		b.WriteString("^")
		last := 0
		for i, l := range locs {
			b.WriteString(regexp.QuoteMeta(en[last:l[0]]))
			kinds[i] = en[l[1]-1]
			switch kinds[i] {
			case 'd':
				b.WriteString(`(-?\d+)`)
			case 'q':
				b.WriteString(`("(?:[^"\\]|\\.)*")`)
			default:
				b.WriteString(`(.+?)`)
			}
			last = l[1]
		}
		b.WriteString(regexp.QuoteMeta(en[last:]) + "$")
		p.list = append(p.list, format{regexp.MustCompile(b.String()), kinds, fr})
	}
	// The most specific format wins: the one with the most fixed text.
	slices.SortFunc(p.list, func(a, b format) int {
		return len(b.re.String()) - 8*len(b.kinds) - (len(a.re.String()) - 8*len(a.kinds))
	})
	return p.list
}

// sameVerbs says whether fr takes en's arguments in the same order, so
// captures can fill it one by one.
func sameVerbs(en, fr string) bool {
	return !strings.Contains(fr, "%[") && slices.Equal(verb.FindAllString(en, -1), verb.FindAllString(fr, -1))
}

// translate fills the French of the first format s matches; text arguments
// are translated in turn, depth levels deep.
func (p *formats) translate(s string, depth int) string {
	for _, f := range p.all() {
		m := f.re.FindStringSubmatch(s)
		if m == nil {
			continue
		}
		i := 0
		return verb.ReplaceAllStringFunc(f.fr, func(string) string {
			arg := m[i+1]
			if k := f.kinds[i]; depth > 0 && (k == 's' || k == 'v') {
				if t, ok := catalog[arg]; ok {
					arg = t
				} else {
					arg = p.translate(arg, depth-1)
				}
			}
			i++
			return arg
		})
	}
	return s
}

// Errorf is fmt.Errorf for an error people read: the catalog test finds
// its format, and Tr translates the message it makes.
func Errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }

// Mark returns s unchanged; it marks a string that is translated later,
// where it is shown, so the catalog test finds it.
func Mark(s string) string { return s }

// Num writes f with prec decimals, with a decimal comma in French.
func Num(lang string, f float64, prec int) string {
	s := strconv.FormatFloat(f, 'f', prec, 64)
	if lang == "fr" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

var months = [...]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}

// Date writes a day as "2 January 2006", or "2 janvier 2006" in French.
func Date(lang string, t time.Time) string {
	if lang == "fr" {
		return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()-1], t.Year())
	}
	return t.Format("2 January 2006")
}
