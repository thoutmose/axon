// Package httpx holds small HTTP helpers shared by the API's handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MaxBody caps request bodies; no Axon request needs more.
const MaxBody = 64 << 10

// WriteJSON encodes body after the caller has written status and headers.
func WriteJSON(w http.ResponseWriter, body any) { _ = json.NewEncoder(w).Encode(body) }

// JSON writes body as JSON with the given status.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Problem writes an RFC 9457 problem document. detail must never contain
// internal error text: callers log that and send a human-readable sentence.
func Problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// Decode reads a JSON body into v, rejecting unknown fields and oversized bodies.
func Decode(r *http.Request, v any) error { return DecodeN(r, v, MaxBody) }

// DecodeN is Decode with a custom size limit, for endpoints that carry documents.
func DecodeN(r *http.Request, v any, limit int64) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, limit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.New("the request body is not valid JSON for this endpoint")
	}
	return nil
}

// ClientIP returns the caller's address. Caddy sets X-Real-IP from the
// forwarded chain after discarding private-range proxies (deploy/Caddyfile),
// and only Caddy can reach the API, so the header is trusted.
func ClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ETag renders a row version (its updated_at) as a strong entity tag.
func ETag(version time.Time) string {
	return `"` + strconv.FormatInt(version.UnixMicro(), 36) + `"`
}

// IfMatch parses the If-Match header back into a row version. It answers 428
// when the header is missing and 412 when it cannot be a current version.
func IfMatch(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	h := r.Header.Get("If-Match")
	if h == "" {
		Problem(w, http.StatusPreconditionRequired, "Send If-Match with the ETag you last read, so concurrent edits are not lost.")
		return time.Time{}, false
	}
	n, err := strconv.ParseInt(strings.Trim(h, `"`), 36, 64)
	if err != nil {
		Problem(w, http.StatusPreconditionFailed, "The If-Match value does not match the current version. Reload and try again.")
		return time.Time{}, false
	}
	return time.UnixMicro(n).UTC(), true
}

// List is the response shape of every paginated endpoint. NextCursor is set
// when another page may exist; pass it back as ?after= (or ?before=).
type List[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// Limit reads ?limit= (default and maximum 100).
func Limit(r *http.Request) (int, bool) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return 100, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil && n >= 1 && n <= 100
}
