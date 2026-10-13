package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A compressing proxy suffixes the entity tag with its encoding (Caddy's
// encode writes "…-zstd" and "…-gzip"), and browsers send it back as it
// came: it is the same version (EP-74 regression found in EP-100).
func TestIfMatchAcceptsWhatTheProxyMadeOfTheTag(t *testing.T) {
	v := time.Date(2026, 10, 14, 10, 48, 0, 123000, time.UTC)
	tag := ETag(v)
	for _, h := range []string{tag, tag[:len(tag)-1] + `-zstd"`, tag[:len(tag)-1] + `-gzip"`, `W/` + tag, `W/` + tag[:len(tag)-1] + `-br"`} {
		r := httptest.NewRequest(http.MethodPut, "/", nil)
		r.Header.Set("If-Match", h)
		w := httptest.NewRecorder()
		got, ok := IfMatch(w, r)
		if !ok || !got.Equal(v) {
			t.Errorf("If-Match %s = %v %v (%d)", h, got, ok, w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPut, "/", nil)
	r.Header.Set("If-Match", `"not-a-version"`)
	if _, ok := IfMatch(httptest.NewRecorder(), r); ok {
		t.Error("a tag that is no version passed")
	}
}
