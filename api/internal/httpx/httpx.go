// Package httpx holds small HTTP helpers shared by the API's handlers.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
)

// MaxBody caps request bodies; no Axon request needs more.
const MaxBody = 64 << 10

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
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, MaxBody))
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
