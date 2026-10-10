package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRequestLogOmitsQueryHeadersAndPath(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	reg := prometheus.NewRegistry()
	h := New("dev", fakeDB{}, log, NewMetrics(reg, "dev", fakeDB{}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/private/jane@example.com?token=s3cret", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	h.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	for _, leak := range []string{"s3cret", "jane@example.com", "Bearer"} {
		if strings.Contains(out, leak) {
			t.Fatalf("log contains %q: %s", leak, out)
		}
	}
	for _, want := range []string{`"route":"unmatched"`, `"status":404`, `"request_id":`} {
		if !strings.Contains(out, want) {
			t.Fatalf("log lacks %s: %s", want, out)
		}
	}
}

func TestMetricsCountRequestsByRoutePattern(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg, "1.2.3", fakeDB{})
	h := New("1.2.3", fakeDB{}, quiet, m)
	for range 3 {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil))
	}
	if got := testutil.ToFloat64(m.requests.WithLabelValues("GET", "/healthz", "200")); got != 3 {
		t.Fatalf("requests = %v, want 3", got)
	}
}

func TestDBUpGauge(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want float64
	}{{nil, 1}, {errors.New("down"), 0}} {
		reg := prometheus.NewRegistry()
		NewMetrics(reg, "dev", fakeDB{err: tt.err})
		mfs, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		got := -1.0
		for _, mf := range mfs {
			if mf.GetName() == "axon_db_up" {
				got = mf.GetMetric()[0].GetGauge().GetValue()
			}
		}
		if got != tt.want {
			t.Fatalf("db err %v: axon_db_up = %v, want %v", tt.err, got, tt.want)
		}
	}
}
