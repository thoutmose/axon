package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProblemSpeaksTheResponseLanguage(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Language", "fr")
	Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	if !strings.Contains(w.Body.String(), `"detail":"Une erreur est survenue. Réessayez."`) {
		t.Fatalf("body = %s", w.Body.String())
	}
}
