// Package notesync keeps a folder of Markdown files (an Obsidian-style vault,
// usually a Git repository) and Axon's notes in step. It runs where the files
// live and talks to Axon over HTTPS with a personal access token.
//
// For every path it remembers the content hash both sides last agreed on.
// A change on one side is copied to the other; a change on both sides is a
// conflict: Axon's version is written next to the file as
// "<name>.axon-conflict.md" and nothing is overwritten. Delete the conflict
// file once you have merged, and the next run sends your version.
package notesync

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// ConflictSuffix marks Axon's side of a conflict, next to the local file.
const ConflictSuffix = ".axon-conflict.md"

// Config describes one sync.
type Config struct {
	Dir       string // the vault
	URL       string // Axon base URL, e.g. https://srv-prod.tail781d53.ts.net
	Token     string // write-scope personal access token
	StatePath string // where agreed hashes are kept between runs
	TwoWay    bool   // false: one-way import, files → Axon only
	HTTP      *http.Client
}

// Report summarizes a run.
type Report struct {
	Pushed, Pulled, Deleted, Unchanged int
	Conflicts                          []string
}

type state struct {
	Hashes map[string]string `json:"hashes"` // path → agreed content hash
}

type remoteNote struct {
	Path    string  `json:"path"`
	Hash    string  `json:"hash"`
	Content *string `json:"content"`
}

// Run performs one sync.
func Run(ctx context.Context, cfg Config) (Report, error) {
	var rep Report
	st, err := loadState(cfg.StatePath)
	if err != nil {
		return rep, err
	}
	c := client{cfg: cfg}

	remote, tombstones, err := c.fetchAll(ctx)
	if err != nil {
		return rep, err
	}
	local, err := scan(cfg.Dir)
	if err != nil {
		return rep, err
	}

	// 1. Axon → files (two-way only).
	for p, n := range remote {
		agreed := st.Hashes[p]
		l, onDisk := local[p]
		switch {
		case n.Hash == agreed:
			rep.Unchanged++
		case onDisk && l.hash == n.Hash:
			st.Hashes[p] = n.Hash // both sides made the same change
		case !cfg.TwoWay:
			// Import mode never writes files; the push below reports a conflict if needed.
		case !onDisk && agreed == "" || onDisk && l.hash == agreed:
			if err := writeFile(cfg.Dir, p, *n.Content); err != nil {
				return rep, err
			}
			local[p] = file{hash: n.Hash, content: *n.Content}
			st.Hashes[p] = n.Hash
			rep.Pulled++
		case !onDisk: // deleted here, edited in Axon: keep Axon's edit
			if err := writeFile(cfg.Dir, p, *n.Content); err != nil {
				return rep, err
			}
			local[p] = file{hash: n.Hash, content: *n.Content}
			st.Hashes[p] = n.Hash
			rep.Pulled++
		default:
			if err := keepBoth(cfg.Dir, p, *n.Content, &rep); err != nil {
				return rep, err
			}
			st.Hashes[p] = n.Hash
		}
	}
	if cfg.TwoWay {
		for _, p := range tombstones {
			if _, stillThere := remote[p]; stillThere {
				continue
			}
			l, onDisk := local[p]
			agreed, known := st.Hashes[p]
			switch {
			case !known:
			case onDisk && l.hash == agreed:
				if err := os.Remove(filepath.Join(cfg.Dir, filepath.FromSlash(p))); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return rep, err
				}
				delete(local, p)
				delete(st.Hashes, p)
				rep.Deleted++
			default: // edited here after Axon deleted it: send it back as new
				delete(st.Hashes, p)
			}
		}
	}

	// 2. Files → Axon.
	paths := make([]string, 0, len(local))
	for p := range local {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		l := local[p]
		if l.hash == st.Hashes[p] || conflictPending(cfg.Dir, p) {
			continue
		}
		current, err := c.push(ctx, p, l.content, st.Hashes[p])
		switch {
		case errors.Is(err, errConflict):
			if cfg.TwoWay && current != nil {
				if err := keepBoth(cfg.Dir, p, *current.Content, &rep); err != nil {
					return rep, err
				}
				st.Hashes[p] = current.Hash
			} else {
				rep.Conflicts = append(rep.Conflicts, p)
			}
		case err != nil:
			return rep, fmt.Errorf("push %s: %w", p, err)
		default:
			st.Hashes[p] = l.hash
			rep.Pushed++
		}
	}

	// 3. Files deleted here → delete in Axon (two-way only; import never deletes).
	if cfg.TwoWay {
		for p, agreed := range st.Hashes {
			if _, onDisk := local[p]; onDisk {
				continue
			}
			current, err := c.delete(ctx, p, agreed)
			switch {
			case errors.Is(err, errConflict) && current != nil: // edited in Axon: restore it here
				if err := writeFile(cfg.Dir, p, *current.Content); err != nil {
					return rep, err
				}
				st.Hashes[p] = current.Hash
				rep.Pulled++
			case err != nil:
				return rep, fmt.Errorf("delete %s: %w", p, err)
			default:
				delete(st.Hashes, p)
				rep.Deleted++
			}
		}
	}
	return rep, saveState(cfg.StatePath, st)
}

// keepBoth writes Axon's version beside the local file.
func keepBoth(dir, p, content string, rep *Report) error {
	rep.Conflicts = append(rep.Conflicts, p)
	return writeFile(dir, strings.TrimSuffix(p, ".md")+ConflictSuffix, content)
}

func conflictPending(dir, p string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(p, ".md")+ConflictSuffix)))
	return err == nil
}

type file struct {
	hash, content string
}

// scan reads every Markdown file, skipping hidden folders (.git, .obsidian)
// and conflict copies.
func scan(dir string) (map[string]file, error) {
	out := map[string]file{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != dir && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ConflictSuffix) || strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = file{hash: hash(b), content: string(b)}
		return nil
	})
	return out, err
}

func hash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func writeFile(dir, p, content string) error {
	clean := path.Clean(p)
	if strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || clean == ".." {
		return fmt.Errorf("refusing to write outside the vault: %q", p)
	}
	full := filepath.Join(dir, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	tmp := full + ".axon-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, full)
}

func loadState(p string) (*state, error) {
	st := &state{Hashes: map[string]string{}}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, st); err != nil {
		return nil, fmt.Errorf("sync state %s is corrupt: %w", p, err)
	}
	if st.Hashes == nil {
		st.Hashes = map[string]string{}
	}
	return st, nil
}

func saveState(p string, st *state) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// --- HTTP ---

var errConflict = errors.New("conflict")

type client struct{ cfg Config }

func (c client) do(ctx context.Context, method, p string, body any, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.URL, "/")+"/api"+p, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	hc := c.cfg.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	res, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode >= 400 && res.StatusCode != http.StatusConflict {
		return res.StatusCode, fmt.Errorf("axon: %s %s: %d %s", method, p, res.StatusCode, strings.TrimSpace(string(b)))
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return res.StatusCode, err
		}
	}
	return res.StatusCode, nil
}

func (c client) fetchAll(ctx context.Context) (map[string]remoteNote, []string, error) {
	notes := map[string]remoteNote{}
	var tombstones []string
	cursor := ""
	for {
		var page struct {
			Notes      []remoteNote `json:"notes"`
			Tombstones []struct {
				Path string `json:"path"`
			} `json:"tombstones"`
			Cursor string `json:"cursor"`
			More   bool   `json:"more"`
		}
		p := "/v1/notes/sync"
		if cursor != "" {
			p += "?cursor=" + cursor
		}
		if _, err := c.do(ctx, http.MethodGet, p, nil, &page); err != nil {
			return nil, nil, err
		}
		for _, n := range page.Notes {
			notes[n.Path] = n
		}
		if cursor == "" {
			for _, t := range page.Tombstones {
				tombstones = append(tombstones, t.Path)
			}
		}
		if !page.More {
			return notes, tombstones, nil
		}
		cursor = page.Cursor
	}
}

func (c client) push(ctx context.Context, p, content, base string) (*remoteNote, error) {
	var res struct {
		Current *remoteNote `json:"current"`
	}
	code, err := c.do(ctx, http.MethodPost, "/v1/notes/sync", map[string]string{"path": p, "content": content, "base_hash": base}, &res)
	if err != nil {
		return nil, err
	}
	if code == http.StatusConflict {
		return res.Current, errConflict
	}
	return nil, nil
}

func (c client) delete(ctx context.Context, p, base string) (*remoteNote, error) {
	var res struct {
		Current *remoteNote `json:"current"`
	}
	code, err := c.do(ctx, http.MethodPost, "/v1/notes/sync/delete", map[string]string{"path": p, "base_hash": base}, &res)
	if err != nil {
		return nil, err
	}
	if code == http.StatusConflict {
		return res.Current, errConflict
	}
	return nil, nil
}
