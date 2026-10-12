package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/notesync"
)

// eventually polls cond for up to 10 seconds: the acceptance criterion.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("not within 10 s: %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestLiveVaultSyncPropagatesEveryChangeBothWaysWithinSeconds(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	if out := alice.mustDo(200, "GET", "/v1/notes/sync/version", nil); !strings.Contains(string(out), `"version":"0|`) {
		t.Fatalf("version = %s", out)
	}
	dir := t.TempDir()
	s := e.syncer(alice, dir, true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- notesync.Watch(ctx, s.cfg, 100*time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	}()
	defer func() {
		cancel()
		<-done
	}()

	notePath := func(p string) (noteJSON, bool) {
		var list struct {
			Items []noteJSON `json:"items"`
		}
		_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?path="+p, nil), &list)
		for _, n := range list.Items {
			if n.Path == p {
				return n, true
			}
		}
		return noteJSON{}, false
	}
	write := func(p, content string) {
		full := filepath.Join(dir, filepath.FromSlash(p))
		_ = os.MkdirAll(filepath.Dir(full), 0o750)
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Vault → Axon: create, edit, rename, delete.
	write("Inbox/idea.md", "# Idea\n\nFirst draft.")
	eventually(t, "a new file appears in Axon", func() bool { _, ok := notePath("Inbox/idea.md"); return ok })
	write("Inbox/idea.md", "# Idea\n\nSecond draft.")
	eventually(t, "an edit reaches Axon", func() bool {
		n, ok := notePath("Inbox/idea.md")
		if !ok {
			return false
		}
		full := alice.mustDo(200, "GET", "/v1/notes/"+n.ID, nil)
		return strings.Contains(string(full), "Second draft")
	})
	if err := os.Rename(filepath.Join(dir, "Inbox", "idea.md"), filepath.Join(dir, "Inbox", "big-idea.md")); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a rename reaches Axon", func() bool {
		_, oldThere := notePath("Inbox/idea.md")
		_, newThere := notePath("Inbox/big-idea.md")
		return !oldThere && newThere
	})
	if err := os.Remove(filepath.Join(dir, "Inbox", "big-idea.md")); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a deletion reaches Axon", func() bool { _, ok := notePath("Inbox/big-idea.md"); return !ok })

	// Axon → vault: create, edit, rename, delete.
	var n noteJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Areas/health.md","content":"# Health\n\nSleep 8 h."}`)), &n)
	eventually(t, "a new note appears in the vault", func() bool {
		c, ok := readFile(t, dir, "Areas/health.md")
		return ok && strings.Contains(c, "Sleep 8 h")
	})
	if code, out := alice.do("PUT", "/v1/notes/"+n.ID, []byte(`{"path":"Areas/health.md","content":"# Health\n\nSleep 8 h. Walk daily."}`), "If-Match", alice.etag("/v1/notes/"+n.ID)); code != 200 {
		t.Fatalf("update = %d %s", code, out)
	}
	eventually(t, "an edit reaches the vault", func() bool { c, _ := readFile(t, dir, "Areas/health.md"); return strings.Contains(c, "Walk daily") })
	if code, out := alice.do("PUT", "/v1/notes/"+n.ID, []byte(`{"path":"Areas/wellbeing.md","content":"# Health\n\nSleep 8 h. Walk daily."}`), "If-Match", alice.etag("/v1/notes/"+n.ID)); code != 200 {
		t.Fatalf("update = %d %s", code, out)
	}
	eventually(t, "a rename reaches the vault", func() bool {
		_, oldThere := readFile(t, dir, "Areas/health.md")
		_, newThere := readFile(t, dir, "Areas/wellbeing.md")
		return !oldThere && newThere
	})
	alice.mustDo(204, "DELETE", "/v1/notes/"+n.ID, nil)
	eventually(t, "a deletion reaches the vault", func() bool { _, ok := readFile(t, dir, "Areas/wellbeing.md"); return !ok })
}
