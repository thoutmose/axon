package notesync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Paired is what Axon hands a vault server for a pairing code (EP-106).
type Paired struct {
	Token     string   `json:"token"`
	Folders   []string `json:"folders"`  // to create when the vault was empty
	Scaffold  []string `json:"scaffold"` // notes Axon added for it, written by the first sync
	ExpiresAt string   `json:"expires_at"`
}

// Pair exchanges a one-time code for a write token, saves it to tokenFile
// (readable by its owner only), and, when the vault has no notes yet,
// creates its folders. cfg.Token is set for the sync that follows.
func Pair(ctx context.Context, cfg *Config, code, tokenFile string) (Paired, error) {
	var out Paired
	empty, err := Empty(cfg.Dir)
	if err != nil {
		return out, err
	}
	host, _ := os.Hostname()
	c := client{cfg: *cfg}
	if _, err := c.do(ctx, http.MethodPost, "/v1/vault/pair", map[string]any{"code": code, "host": host, "empty": empty}, &out); err != nil {
		return out, fmt.Errorf("pairing: %w", err)
	}
	if out.Token == "" {
		return out, errors.New("pairing: Axon sent no token")
	}
	if err := os.MkdirAll(filepath.Dir(tokenFile), 0o700); err != nil {
		return out, err
	}
	if err := os.WriteFile(tokenFile, []byte(out.Token+"\n"), 0o600); err != nil {
		return out, err
	}
	vault, err := os.OpenRoot(cfg.Dir)
	if err != nil {
		return out, err
	}
	defer func() { _ = vault.Close() }()
	for _, f := range out.Folders {
		if err := vault.MkdirAll(filepath.FromSlash(f), 0o700); err != nil {
			return out, err
		}
	}
	cfg.Token = out.Token
	return out, nil
}

// Empty reports whether the vault holds no Markdown note yet.
func Empty(dir string) (bool, error) {
	vault, err := os.OpenRoot(dir)
	if err != nil {
		return false, err
	}
	defer func() { _ = vault.Close() }()
	found, err := scan(vault)
	return len(found) == 0, err
}

// Heartbeat tells Axon how the last sync went, so it can show whether the
// vault is in step. Axon versions without vault servers answer 404: ignored.
func Heartbeat(ctx context.Context, cfg Config, rep Report, runErr error) error {
	host, _ := os.Hostname()
	body := map[string]any{
		"host": host, "notes": rep.Notes, "files": rep.Files, "pushed": rep.Pushed, "pulled": rep.Pulled,
		"deleted": rep.Deleted, "conflicts": append([]string{}, rep.Conflicts...), "error": "",
	}
	if runErr != nil {
		body["error"] = runErr.Error()
	}
	_, err := client{cfg: cfg}.do(ctx, http.MethodPost, "/v1/vault/heartbeat", body, nil)
	if err != nil && strings.Contains(err.Error(), ": 404 ") {
		return nil
	}
	return err
}
