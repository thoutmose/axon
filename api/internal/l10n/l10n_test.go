package l10n

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLanguageFromAcceptLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"":                        "en",
		"fr":                      "fr",
		"fr-CA,fr;q=0.9,en;q=0.8": "fr",
		"en-GB,en;q=0.9,fr;q=0.8": "en",
		"de-DE,fr;q=0.7":          "fr",
		"de-DE":                   "en",
		"FR":                      "fr",
		"fr;q=0, en":              "en",
		"en;q=0.5, fr;q=0.9":      "fr",
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			r.Header.Set("Accept-Language", header)
		}
		if got := FromRequest(r); got != want {
			t.Errorf("%q: got %q, want %q", header, got, want)
		}
	}
}

func TestMiddlewareTellsHandlersAndCaches(t *testing.T) {
	var seen string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = Lang(r.Context()) }))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "fr-FR")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if seen != "fr" || w.Header().Get("Content-Language") != "fr" || w.Header().Get("Vary") != "Accept-Language" {
		t.Fatalf("seen %q, headers %v", seen, w.Header())
	}
}

func TestTranslatesFromTheCatalog(t *testing.T) {
	catalog["Done %d of %d."] = "%d faits sur %d."
	defer delete(catalog, "Done %d of %d.")
	if got := Tr("fr", "Done %d of %d.", 2, 3); got != "2 faits sur 3." {
		t.Fatalf("fr = %q", got)
	}
	if got := Tr("en", "Done %d of %d.", 2, 3); got != "Done 2 of 3." {
		t.Fatalf("en = %q", got)
	}
	// Not in the catalog: the English, never an empty string.
	if got := Tr("fr", "Nobody translated this."); got != "Nobody translated this." {
		t.Fatalf("missing = %q", got)
	}
	// A message without arguments is not a format: a % stays as it is.
	if got := Tr("en", "100% done."); got != "100% done." {
		t.Fatalf("percent = %q", got)
	}
}

func TestNumbersInTheLanguage(t *testing.T) {
	if got := Num("fr", 1234.5, 1); got != "1234,5" {
		t.Fatalf("fr = %q", got)
	}
	if got := Num("en", 0.25, 2); got != "0.25" {
		t.Fatalf("en = %q", got)
	}
}

func TestDatesInTheLanguage(t *testing.T) {
	d := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	if Date("fr", d) != "2 août 2026" || Date("en", d) != "2 August 2026" {
		t.Fatalf("%s / %s", Date("fr", d), Date("en", d))
	}
}
