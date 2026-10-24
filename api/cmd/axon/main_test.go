package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
)

// With the demo on, the server still runs every worker, the tech watch and
// its video summaries and graph included (production lost them to AXON_DEMO=1).
func TestDemoServerStartsEveryWorker(t *testing.T) {
	t.Setenv("AXON_DEMO", "1")
	pool, err := pgxpool.New(context.Background(), "postgres://axon@127.0.0.1:1/axon") // connects lazily
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	cfg := auth.Config{Demo: demoEnabled(), BaseURL: "http://localhost", EncryptionKey: strings.Repeat("ab", 32)}
	list, err := backgroundWorkers(cfg, pool, slog.New(slog.DiscardHandler), nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, w := range list {
		if w.run == nil {
			t.Errorf("worker %s has nothing to run", w.name)
		}
		got = append(got, w.name)
	}
	want := "demo maintenance vault-mirror reminders calendars semantic-index replies watch video-summaries watch-knowledge"
	if strings.Join(got, " ") != want {
		t.Fatalf("workers = %v, want %s", got, want)
	}
}

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
