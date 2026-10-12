package notesync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Pictures and PDFs that notes embed sync the same way as notes (EP-51):
// per path, the hash both sides last agreed on; a change on both sides keeps
// Axon's copy beside the file as "<name>.axon-conflict.<ext>".

// maxFile is the largest file Axon stores; bigger ones stay local.
const maxFile = 10 << 20

var fileExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true, ".pdf": true}

func isAttachment(name string) bool {
	return fileExts[strings.ToLower(path.Ext(name))] && !strings.Contains(name, ".axon-conflict.") && !strings.HasPrefix(name, ".")
}

func fileConflictPath(p string) string {
	ext := path.Ext(p)
	return strings.TrimSuffix(p, ext) + ".axon-conflict" + ext
}

type remoteFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int    `json:"size"`
}

// scanFiles hashes every attachment, skipping hidden folders, symlinks,
// conflict copies, and files too large for Axon.
func scanFiles(vault *os.Root) (map[string]string, error) {
	out := map[string]string{}
	err := fs.WalkDir(vault.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !isAttachment(d.Name()) {
			return nil
		}
		if info, err := d.Info(); err != nil || info.Size() > maxFile {
			return err
		}
		b, err := vault.ReadFile(filepath.FromSlash(p))
		if err != nil {
			return err
		}
		out[p] = hash(b)
		return nil
	})
	return out, err
}

func syncFiles(ctx context.Context, c client, vault *os.Root, st *state, rep *Report) error {
	remote, tombstones, err := c.listFiles(ctx)
	if err != nil {
		return err
	}
	local, err := scanFiles(vault)
	if err != nil {
		return err
	}
	two := c.cfg.TwoWay
	pull := func(p, target string) error {
		b, err := c.getFile(ctx, p)
		if err != nil {
			return err
		}
		return writeFile(vault, target, string(b))
	}

	// 1. Axon → files (two-way only).
	for p, f := range remote {
		agreed := st.Files[p]
		l, onDisk := local[p]
		switch {
		case f.Hash == agreed:
			rep.Unchanged++
		case onDisk && l == f.Hash:
			st.Files[p] = f.Hash
		case !two:
		case !onDisk || l == agreed:
			if err := pull(p, p); err != nil {
				return err
			}
			local[p] = f.Hash
			st.Files[p] = f.Hash
			rep.Pulled++
		default:
			rep.Conflicts = append(rep.Conflicts, p)
			if err := pull(p, fileConflictPath(p)); err != nil {
				return err
			}
			st.Files[p] = f.Hash
		}
	}
	if two {
		for _, p := range tombstones {
			if _, stillThere := remote[p]; stillThere {
				continue
			}
			l, onDisk := local[p]
			agreed, known := st.Files[p]
			switch {
			case !known:
			case onDisk && l == agreed:
				if err := vault.Remove(filepath.FromSlash(p)); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				delete(local, p)
				delete(st.Files, p)
				rep.Deleted++
			default:
				delete(st.Files, p)
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
		if local[p] == st.Files[p] {
			continue
		}
		if _, err := vault.Stat(filepath.FromSlash(fileConflictPath(p))); err == nil {
			continue
		}
		b, err := vault.ReadFile(filepath.FromSlash(p))
		if err != nil {
			return err
		}
		current, code, err := c.putFile(ctx, p, b, st.Files[p])
		switch {
		case errors.Is(err, errConflict):
			if two && current != nil {
				rep.Conflicts = append(rep.Conflicts, p)
				if err := pull(p, fileConflictPath(p)); err != nil {
					return err
				}
				st.Files[p] = current.Hash
			} else {
				rep.Conflicts = append(rep.Conflicts, p)
			}
		case code == http.StatusUnprocessableEntity || code == http.StatusRequestEntityTooLarge || code == http.StatusInsufficientStorage:
			rep.Skipped = append(rep.Skipped, p) // not a real picture, or no room: it stays local
		case err != nil:
			return fmt.Errorf("push %s: %w", p, err)
		default:
			st.Files[p] = local[p]
			rep.Pushed++
		}
	}

	// 3. Files deleted here → delete in Axon (two-way only).
	if two {
		for p, agreed := range st.Files {
			if _, onDisk := local[p]; onDisk {
				continue
			}
			current, err := c.deleteFile(ctx, p, agreed)
			switch {
			case errors.Is(err, errConflict) && current != nil:
				if err := pull(p, p); err != nil {
					return err
				}
				st.Files[p] = current.Hash
				rep.Pulled++
			case err != nil:
				return fmt.Errorf("delete %s: %w", p, err)
			default:
				delete(st.Files, p)
				rep.Deleted++
			}
		}
	}
	return nil
}

func (c client) listFiles(ctx context.Context) (map[string]remoteFile, []string, error) {
	var res struct {
		Items      []remoteFile `json:"items"`
		Tombstones []struct {
			Path string `json:"path"`
		} `json:"tombstones"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/v1/notes/files", nil, &res); err != nil {
		return nil, nil, err
	}
	out := map[string]remoteFile{}
	for _, f := range res.Items {
		out[f.Path] = f
	}
	stones := make([]string, 0, len(res.Tombstones))
	for _, t := range res.Tombstones {
		stones = append(stones, t.Path)
	}
	return out, stones, nil
}

func (c client) getFile(ctx context.Context, p string) ([]byte, error) {
	_, b, err := c.send(ctx, http.MethodGet, "/v1/notes/file?path="+url.QueryEscape(p), "", nil)
	return b, err
}

func (c client) putFile(ctx context.Context, p string, data []byte, base string) (*remoteFile, int, error) {
	q := url.Values{"path": {p}, "base_hash": {base}}
	code, b, err := c.send(ctx, http.MethodPut, "/v1/notes/file?"+q.Encode(), "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		return nil, code, err
	}
	if code == http.StatusConflict {
		var res struct {
			Current *remoteFile `json:"current"`
		}
		_ = decode(b, &res)
		return res.Current, code, errConflict
	}
	return nil, code, nil
}

func (c client) deleteFile(ctx context.Context, p, base string) (*remoteFile, error) {
	q := url.Values{"path": {p}, "base_hash": {base}}
	code, b, err := c.send(ctx, http.MethodDelete, "/v1/notes/file?"+q.Encode(), "", nil)
	if err != nil {
		return nil, err
	}
	if code == http.StatusConflict {
		var res struct {
			Current *remoteFile `json:"current"`
		}
		_ = decode(b, &res)
		return res.Current, errConflict
	}
	return nil, nil
}
