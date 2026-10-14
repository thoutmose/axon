package watch

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Share links (EP-75): a read-only page of a selection of the news, for
// people without an account. The link carries a random token; only its
// SHA-256 is kept. Shared news show what the sources published (title,
// link, summary, labels), never the person's status, notes, or other news.

// Share is a link to a selection of the news.
type Share struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Query        string     `json:"query"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastViewedAt *time.Time `json:"last_viewed_at"`
	Views        int        `json:"views"`
}

// shareKeys are the filter fields a share keeps.
var shareKeys = []string{"status", "kind", "source", "topic", "label", "q", "from", "to"}

// MaxShareDays is the longest a link lives.
const MaxShareDays = 365

func hashShareToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// shareQuery keeps the filter fields of a query string, checked.
func shareQuery(raw string) (string, Filter, error) {
	in, err := url.ParseQuery(strings.TrimPrefix(raw, "?"))
	if err != nil {
		return "", Filter{}, InvalidError{"The filter is not a query string."}
	}
	kept := url.Values{}
	for _, k := range shareKeys {
		if v := in.Get(k); v != "" {
			kept.Set(k, v)
		}
	}
	// Saved news are a list someone curated; unread or read would follow
	// the owner's reading and empty as they read.
	if kept.Get("status") != "saved" {
		kept.Del("status")
	}
	f, err := FilterFrom(kept)
	return kept.Encode(), f, err
}

// ListShares returns a person's links, newest first.
func ListShares(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Share, error) {
	rows, err := q.Query(ctx, `SELECT id, name, query, expires_at, created_at, last_viewed_at, views FROM watch_shares
		WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Share, error) {
		var s Share
		err := r.Scan(&s.ID, &s.Name, &s.Query, &s.ExpiresAt, &s.CreatedAt, &s.LastViewedAt, &s.Views)
		return s, err
	})
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		topics, sources := map[string]string{}, map[uuid.UUID]string{}
		if ts, err := ListTopics(ctx, q, userID); err == nil {
			for _, t := range ts {
				topics[t.ID] = t.Name
			}
		}
		if ss, err := ListSources(ctx, q, userID); err == nil {
			for _, s := range ss {
				sources[s.ID] = s.Title
			}
		}
		for i := range list {
			if _, f, err := shareQuery(list[i].Query); err == nil {
				src := ""
				if f.SourceID != nil {
					src = sources[*f.SourceID]
				}
				list[i].Description = Describe(f, topics[f.TopicID], src)
			}
		}
	}
	return append([]Share{}, list...), nil
}

func (h *Handlers) shares(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	list, err := ListShares(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": list})
}

func (h *Handlers) createShare(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Name  string `json:"name"`
		Query string `json:"query"`
		Days  int    `json:"days"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(b.Name)
	if b.Days == 0 {
		b.Days = 30
	}
	query, _, err := shareQuery(b.Query)
	switch {
	case name == "" || utf8.RuneCountInString(name) > 100:
		err = InvalidError{"Name the link in 1 to 100 characters, e.g. who it is for."}
	case b.Days < 1 || b.Days > MaxShareDays:
		err = InvalidError{fmt.Sprintf("A link lives 1 to %d days.", MaxShareDays)}
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := h.now().UTC()
	s := Share{ID: uuid.Must(uuid.NewV7()), Name: name, Query: query, CreatedAt: now, ExpiresAt: now.AddDate(0, 0, b.Days)}
	err = h.tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `INSERT INTO watch_shares (id, user_id, name, token_hash, query, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			s.ID, who.UserID, s.Name, hashShareToken(token), s.Query, s.ExpiresAt, s.CreatedAt); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, who.UserID, audit.WatchShareCreated, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": name, "query": query, "days": b.Days})
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	if list, err := ListShares(r.Context(), h.db, who.UserID); err == nil {
		for _, x := range list {
			if x.ID == s.ID {
				s = x
			}
		}
	}
	// The token is shown once: only its hash is kept.
	httpx.JSON(w, http.StatusCreated, map[string]any{"share": s, "path": "/shared/watch/" + token})
}

func (h *Handlers) deleteShare(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	err := h.tx(r.Context(), func(tx pgx.Tx) error {
		var name string
		err := tx.QueryRow(r.Context(), `DELETE FROM watch_shares WHERE id = $1 AND user_id = $2 RETURNING name`, id, who.UserID).Scan(&name)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, who.UserID, audit.WatchShareRevoked, httpx.ClientIP(r), r.UserAgent(), map[string]any{"name": name})
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SharedItem is what a shared page shows of a news item.
type SharedItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	SourceTitle string    `json:"source_title"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"published_at"`
	Summary     string    `json:"summary"`
	Kind        string    `json:"kind"`
	Labels      []string  `json:"labels"`
	Minutes     int       `json:"reading_minutes"`
}

// public strips what belongs to the person, not to the source.
func public(it Item) Item {
	return Item{URL: it.URL, Title: it.Title, SourceTitle: it.SourceTitle, Author: it.Author, PublishedAt: it.PublishedAt, FetchedAt: it.FetchedAt,
		Summary: it.Summary, Image: it.Image, Audio: it.Audio, Lang: it.Lang, Words: it.Words, Minutes: it.Minutes, Keywords: it.Keywords,
		Labels: it.Labels, Categories: it.Categories, TopicIDs: it.TopicIDs, AlsoIn: it.AlsoIn, Kind: it.Kind}
}

// sharedPage serves a link: the page's data, or a download with ?format=.
func (h *Handlers) sharedPage(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if _, ok := Formats[format]; format != "" && !ok {
		h.fail(w, InvalidError{"format is csv, ods, md, zip, json, or pdf."})
		return
	}
	ctx := r.Context()
	var userID uuid.UUID
	var name, query string
	var expires time.Time
	err := h.db.QueryRow(ctx, `UPDATE watch_shares SET views = views + 1, last_viewed_at = now()
		WHERE token_hash = $1 AND expires_at > now() RETURNING user_id, name, query, expires_at`,
		hashShareToken(chi.URLParam(r, "token"))).Scan(&userID, &name, &query, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Problem(w, http.StatusNotFound, "This link does not exist, has expired, or was revoked.")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	_, f, err := shareQuery(query)
	if err != nil {
		h.fail(w, err)
		return
	}
	e, err := LoadExport(ctx, h.db, userID, f, name, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	// Only the topics these news carry, and nothing of the person's own.
	used := map[string]string{}
	for i, it := range e.Items {
		e.Items[i] = public(it)
		for _, id := range it.TopicIDs {
			if n, ok := e.Topics[id]; ok {
				used[id] = n
			}
		}
	}
	e.Topics = used
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	if format != "" {
		h.download(w, e, format)
		return
	}
	items := make([]SharedItem, 0, min(len(e.Items), 200))
	for _, it := range e.Items[:min(len(e.Items), 200)] {
		items = append(items, SharedItem{it.Title, it.URL, it.SourceTitle, it.Author, it.PublishedAt, it.Summary, it.Kind, nonNil(it.Labels), it.Minutes})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"name": name, "description": e.Description, "count": len(e.Items), "items": items,
		"generated_at": e.At.UTC(), "expires_at": expires, "attribution": Attribution})
}

// export downloads the news matching the filter.
func (h *Handlers) export(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	format := r.URL.Query().Get("format")
	if _, ok := Formats[format]; !ok {
		h.fail(w, InvalidError{"format is csv, ods, md, zip, json, or pdf."})
		return
	}
	f, err := ParseFilter(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	e, err := LoadExport(r.Context(), h.db, who.UserID, f, "Tech watch", h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	h.download(w, e, format)
}

func (h *Handlers) download(w http.ResponseWriter, e Export, format string) {
	w.Header().Set("Content-Type", Formats[format][0])
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", e.Filename(format)))
	if err := e.Write(w, format); err != nil {
		h.log.Error("tech watch export failed", "format", format, "err", err)
	}
}
