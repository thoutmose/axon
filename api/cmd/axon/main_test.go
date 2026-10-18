package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestCancelledRequestsAreNotErrors(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(clientGone{slog.NewTextHandler(&buf, nil)}).With("svc", "api")
	log.Error("dashboard request failed", "err", fmt.Errorf("query: %w", context.Canceled))
	log.Error("trash purge failed", "err", errors.New("relation does not exist"))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "level=INFO") || !strings.Contains(lines[1], "level=ERROR") {
		t.Fatalf("logged:\n%s", buf.String())
	}
}
