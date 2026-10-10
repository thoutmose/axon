package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, h http.Handler, path string) (int, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("%s: invalid JSON %q: %v", path, rec.Body.String(), err)
	}
	return rec.Code, body
}

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestHealthzReportsVersion(t *testing.T) {
	// Liveness must not depend on the database.
	h := New("1.2.3", fakeDB{err: errors.New("down")}, quiet)
	code, body := get(t, h, "/healthz")
	if code != http.StatusOK || body["status"] != "ok" || body["version"] != "1.2.3" {
		t.Fatalf("got %d %v", code, body)
	}
}

func TestReadyz(t *testing.T) {
	tests := []struct {
		name   string
		dbErr  error
		code   int
		status string
	}{
		{"database up", nil, http.StatusOK, "ready"},
		{"database down", errors.New("connection refused"), http.StatusServiceUnavailable, "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := get(t, New("dev", fakeDB{err: tt.dbErr}, quiet), "/readyz")
			if code != tt.code || body["status"] != tt.status {
				t.Fatalf("got %d %v, want %d %s", code, body, tt.code, tt.status)
			}
			if _, leaked := body["error"]; leaked {
				t.Fatal("readiness response must not expose error details")
			}
		})
	}
}

func TestUnknownRouteIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	New("dev", fakeDB{}, quiet).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
}
