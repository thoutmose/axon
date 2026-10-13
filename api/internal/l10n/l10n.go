// Package l10n writes what the API says to people in their language (EP-87):
// English, or French when the request's Accept-Language prefers it. English
// strings are the keys; one missing from the French catalog stays English.
package l10n

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

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
