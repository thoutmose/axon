package calendars

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (s *Store) Routes(r chi.Router) {
	r.Get("/v1/calendar/sources", s.list)
	r.Post("/v1/calendar/sources", s.create)
	r.Delete("/v1/calendar/sources/{id}", s.delete)
	r.Post("/v1/calendar/sources/{id}/refresh", s.refresh)
	r.Get("/v1/calendar/events", s.events)
}

func (s *Store) fail(w http.ResponseWriter, err error) {
	var ea ErrAddress
	switch {
	case errors.As(err, &ea):
		httpx.Problem(w, http.StatusUnprocessableEntity, ea.Msg)
	case errors.Is(err, pgx.ErrNoRows):
		httpx.Problem(w, http.StatusNotFound, "No such calendar.")
	default:
		s.log.Error("calendars request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// Source is a followed calendar, without its address.
type Source struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Color     string     `json:"color"`
	Host      string     `json:"host"`
	FetchedAt *time.Time `json:"fetched_at"`
	Error     string     `json:"error"`
	Events    int        `json:"events"`
}

func (s *Store) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	rows, err := s.db.Query(r.Context(), `SELECT c.id, c.name, c.color, c.host, c.fetched_at, c.error,
		(SELECT count(*) FROM calendar_events e WHERE e.source_id = c.id) FROM calendar_sources c WHERE c.user_id = $1 ORDER BY c.created_at`, who.UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Source])
	if err != nil {
		s.fail(w, err)
		return
	}
	if items == nil {
		items = []Source{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// create follows a calendar: it is fetched right away, so a wrong address
// is said at once instead of failing later in silence.
func (s *Store) create(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Color string `json:"color"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" || len([]rune(b.Name)) > 100 {
		httpx.Problem(w, http.StatusUnprocessableEntity, "A calendar's name is 1 to 100 characters.")
		return
	}
	if b.Color == "" {
		b.Color = "#5aa9ff"
	}
	if !colorRe.MatchString(b.Color) {
		httpx.Problem(w, http.StatusUnprocessableEntity, "A colour looks like #5aa9ff.")
		return
	}
	var n int
	if err := s.db.QueryRow(r.Context(), `SELECT count(*) FROM calendar_sources WHERE user_id = $1`, who.UserID).Scan(&n); err != nil {
		s.fail(w, err)
		return
	}
	if n >= 20 {
		httpx.Problem(w, http.StatusUnprocessableEntity, "You follow 20 calendars already; remove one first.")
		return
	}
	u, err := Clean(b.URL)
	if err != nil {
		s.fail(w, err)
		return
	}
	id := uuid.Must(uuid.NewV7())
	if _, err := s.db.Exec(r.Context(), `INSERT INTO calendar_sources (id, user_id, name, color, url, host) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, who.UserID, b.Name, b.Color, s.seal(u.String()), u.Hostname()); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Refresh(r.Context(), id); err != nil {
		_, _ = s.db.Exec(r.Context(), `DELETE FROM calendar_sources WHERE id = $1`, id)
		var ea ErrAddress
		if !errors.As(err, &ea) {
			err = ErrAddress{"Axon could not read that calendar: " + err.Error() + "."}
		}
		s.fail(w, err)
		return
	}
	s.one(w, r, who.UserID, id, http.StatusCreated)
}

func (s *Store) one(w http.ResponseWriter, r *http.Request, userID, id uuid.UUID, status int) {
	var src Source
	err := s.db.QueryRow(r.Context(), `SELECT c.id, c.name, c.color, c.host, c.fetched_at, c.error,
		(SELECT count(*) FROM calendar_events e WHERE e.source_id = c.id) FROM calendar_sources c WHERE c.id = $1 AND c.user_id = $2`, id, userID).
		Scan(&src.ID, &src.Name, &src.Color, &src.Host, &src.FetchedAt, &src.Error, &src.Events)
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.JSON(w, status, src)
}

func (s *Store) owned(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return who.UserID, id, pgx.ErrNoRows
	}
	var ok bool
	err = s.db.QueryRow(r.Context(), `SELECT true FROM calendar_sources WHERE id = $1 AND user_id = $2`, id, who.UserID).Scan(&ok)
	return who.UserID, id, err
}

func (s *Store) delete(w http.ResponseWriter, r *http.Request) {
	_, id, err := s.owned(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	if _, err := s.db.Exec(r.Context(), `DELETE FROM calendar_sources WHERE id = $1`, id); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) refresh(w http.ResponseWriter, r *http.Request) {
	userID, id, err := s.owned(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	_ = s.Refresh(r.Context(), id) // a failure is kept on the source and shown
	s.one(w, r, userID, id, http.StatusOK)
}

// Occurrence is an event as the calendar page shows it.
type Occurrence struct {
	SourceID uuid.UUID `json:"source_id"`
	Title    string    `json:"title"`
	Location string    `json:"location"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	AllDay   bool      `json:"all_day"`
	Date     string    `json:"date"`      // the first day, in the person's time zone
	LastDate string    `json:"last_date"` // the last day it covers
}

func (s *Store) events(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	loc, _ := clock.Zone(r.Context(), s.db, who.UserID) // UTC when it cannot be read
	from, err1 := time.ParseInLocation(time.DateOnly, r.URL.Query().Get("from"), loc)
	to, err2 := time.ParseInLocation(time.DateOnly, r.URL.Query().Get("to"), loc)
	if err1 != nil || err2 != nil || to.Before(from) || to.Sub(from) > 92*24*time.Hour {
		httpx.Problem(w, http.StatusBadRequest, "Send ?from= and ?to= as dates like 2026-10-01, at most 92 days apart.")
		return
	}
	end := to.AddDate(0, 0, 1)
	rows, err := s.db.Query(r.Context(), `SELECT source_id, title, location, starts_at, ends_at, all_day FROM calendar_events
		WHERE user_id = $1 AND starts_at < $3 AND (ends_at > $2 OR starts_at >= $2) ORDER BY starts_at LIMIT 3000`, who.UserID, from, end)
	if err != nil {
		s.fail(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Occurrence, error) {
		var o Occurrence
		err := row.Scan(&o.SourceID, &o.Title, &o.Location, &o.Start, &o.End, &o.AllDay)
		o.Date = o.Start.In(loc).Format(time.DateOnly)
		last := o.End
		if o.AllDay || !o.End.After(o.Start) {
			last = o.End.Add(-time.Second) // an all-day event ends at the next midnight
		}
		if last.Before(o.Start) {
			last = o.Start
		}
		o.LastDate = last.In(loc).Format(time.DateOnly)
		return o, err
	})
	if err != nil {
		s.fail(w, err)
		return
	}
	if items == nil {
		items = []Occurrence{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// Export lists the followed calendars for a personal data export: names,
// colours, and hosts; the secret addresses stay out.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT c.id, c.name, c.color, c.host, c.fetched_at, c.error,
		(SELECT count(*) FROM calendar_events e WHERE e.source_id = c.id) FROM calendar_sources c WHERE c.user_id = $1 ORDER BY c.created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Source])
}
