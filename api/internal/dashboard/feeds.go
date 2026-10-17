package dashboard

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/audit"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Feed is a calendar subscription; its URL is shown once, at creation.
type Feed struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Types      []string   `json:"types"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func hashFeedToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func (h *Handlers) listFeeds(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rows, err := h.db.Query(r.Context(), `SELECT id, name, types, created_at, last_used_at FROM calendar_feeds
		WHERE user_id = $1 ORDER BY created_at LIMIT 50`, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Feed])
	if err != nil {
		h.fail(w, err)
		return
	}
	if items == nil {
		items = []Feed{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) createFeed(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Name  string   `json:"name"`
		Types []string `json:"types"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(body.Name)
	types, ok := parseTypes(strings.Join(body.Types, ","))
	for _, t := range types { // what a feed's table allows
		ok = ok && slices.Contains([]string{"task", "goal", "milestone"}, t)
	}
	switch {
	case name == "" || utf8.RuneCountInString(name) > 100:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Name the feed in 1 to 100 characters, e.g. the calendar app it is for.")
		return
	case len(body.Types) == 0 || !ok:
		httpx.Problem(w, http.StatusUnprocessableEntity, "Choose what the feed contains: task, goal, and/or milestone.")
		return
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := base64.RawURLEncoding.EncodeToString(b)
	f := Feed{ID: uuid.Must(uuid.NewV7()), Name: name, Types: types, CreatedAt: time.Now().UTC()}
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `INSERT INTO calendar_feeds (id, user_id, name, token_hash, types, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			f.ID, who.UserID, f.Name, hashFeedToken(token), f.Types, f.CreatedAt); err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, who.UserID, audit.CalendarFeedCreated, httpx.ClientIP(r), r.UserAgent(),
			map[string]any{"name": name, "types": types})
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"feed": f, "path": "/api/v1/calendar-feeds/" + token + ".ics"})
}

func (h *Handlers) revokeFeed(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(w, http.StatusNotFound, "No such feed.")
		return
	}
	gone := errors.New("not found")
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var name string
		err := tx.QueryRow(r.Context(), `DELETE FROM calendar_feeds WHERE id = $1 AND user_id = $2 RETURNING name`, id, who.UserID).Scan(&name)
		if errors.Is(err, pgx.ErrNoRows) {
			return gone
		}
		if err != nil {
			return err
		}
		return audit.Record(r.Context(), tx, who.UserID, audit.CalendarFeedRevoked, httpx.ClientIP(r), r.UserAgent(), map[string]any{"name": name})
	})
	switch {
	case errors.Is(err, gone):
		httpx.Problem(w, http.StatusNotFound, "No such feed.")
	case err != nil:
		h.fail(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

// feed serves the iCal file: the selected types, from 30 days ago to a year
// ahead, open items only, titles only.
func (h *Handlers) feed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var userID uuid.UUID
	var types []string
	err := h.db.QueryRow(ctx, `UPDATE calendar_feeds SET last_used_at = now() WHERE token_hash = $1 RETURNING user_id, types`,
		hashFeedToken(chi.URLParam(r, "token"))).Scan(&userID, &types)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Problem(w, http.StatusNotFound, "This calendar feed does not exist or was revoked.")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	day, err := today(ctx, h.db, userID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	evs, err := events(ctx, h.db, userID, day.AddDate(0, 0, -30), day.AddDate(1, 0, 0), types, true, 2000)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write([]byte(ICal(evs, h.now())))
}

var prefixes = map[string]string{"scheduled": "", "due": "Due: ", "target": "Goal: ", "milestone": "Milestone: "}

// ICal renders events as an RFC 5545 calendar of all-day events.
func ICal(evs []Event, now time.Time) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(fold(s) + "\r\n") }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Axon//Calendar feed//EN")
	line("CALSCALE:GREGORIAN")
	line("X-WR-CALNAME:Axon")
	stamp := now.UTC().Format("20060102T150405Z")
	for _, ev := range evs {
		line("BEGIN:VEVENT")
		uid := fmt.Sprintf("%s-%s", ev.ID, ev.Kind)
		if ev.Kind == "block" { // one roadmap, a block a day
			uid += "-" + ev.Date.Format("20060102")
		}
		line("UID:" + uid + "@axon")
		line("DTSTAMP:" + stamp)
		line("DTSTART;VALUE=DATE:" + ev.Date.Format("20060102"))
		line("DTEND;VALUE=DATE:" + ev.Date.AddDate(0, 0, 1).Format("20060102"))
		line("SUMMARY:" + escapeText(prefixes[ev.Kind]+ev.Title))
		line("TRANSP:TRANSPARENT")
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	return b.String()
}

// escapeText escapes a TEXT value (RFC 5545 §3.3.11).
func escapeText(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(s)
}

// fold splits lines longer than 75 octets without breaking a UTF-8 character.
func fold(s string) string {
	if len(s) <= 75 {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		size := utf8.RuneLen(r)
		if n+size > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}
