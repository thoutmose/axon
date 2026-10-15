package notesync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Watch keeps the vault and Axon in step continuously (EP-36): it runs a
// sync at start, then whenever the vault's files change (a cheap scan of
// names, sizes, and modification times) or Axon's notes change (a cheap
// version check), at most once per interval. Each sync is the same
// conflict-safe Run as a one-off sync. Errors back off up to a minute.
// After each run, and every few minutes when nothing changes, it tells
// Axon how the vault is doing (EP-106).
func Watch(ctx context.Context, cfg Config, interval time.Duration, log *slog.Logger, onRun func(Report)) error {
	c := client{cfg: cfg}
	var lastLocal, lastRemote string
	var last Report
	var lastBeat time.Time
	beat := func(rep Report, err error) {
		if herr := Heartbeat(ctx, cfg, rep, err); herr != nil && ctx.Err() == nil {
			log.Warn("notes sync: cannot report to Axon", "err", herr)
		}
		lastBeat = time.Now()
	}
	wait := interval
	for {
		local, lerr := fingerprint(cfg.Dir)
		remote, rerr := c.version(ctx)
		switch {
		case lerr != nil:
			log.Warn("notes sync: cannot read the vault", "err", lerr)
		case rerr != nil:
			log.Warn("notes sync: cannot reach Axon", "err", rerr)
		case local != lastLocal || remote != lastRemote:
			rep, err := Run(ctx, cfg)
			if err != nil {
				log.Warn("notes sync failed; retrying", "err", err)
				beat(last, err)
				wait = min(2*wait, time.Minute)
				break
			}
			wait = interval
			last = rep
			beat(rep, nil)
			if onRun != nil && (rep.Pushed+rep.Pulled+rep.Deleted > 0 || len(rep.Conflicts) > 0) {
				onRun(rep)
			}
			// Remember the state from before the run: a change made on either
			// side while it ran is then seen next time. Our own writes cost one
			// extra run that finds nothing to do.
			lastLocal, lastRemote = local, remote
		case time.Since(lastBeat) > HeartbeatEvery:
			beat(last, nil)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// HeartbeatEvery is how often a quiet vault still says it is there.
const HeartbeatEvery = 5 * time.Minute

// fingerprint summarises the vault's notes and attachments without reading them.
func fingerprint(dir string) (string, error) {
	var lines []string
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
		if !strings.HasSuffix(name, ".md") && !isAttachment(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		lines = append(lines, fmt.Sprintf("%s|%d|%d", rel, info.Size(), info.ModTime().UnixNano()))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	h := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(h[:]), nil
}

func (c client) version(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/v1/notes/sync/version", nil, &out); err != nil {
		return "", err
	}
	return out.Version, nil
}
