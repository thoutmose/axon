package insights

import (
	"encoding/csv"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the insights endpoints.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the insights endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/insights", h.weekly)
	r.Get("/v1/insights/review", h.review)
	r.Get("/v1/insights/metrics/{id}/items", h.items)
	r.Get("/v1/insights/export.csv", h.export)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("insights request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

// clock returns the person's time zone and today's date there.
func (h *Handlers) clock(r *http.Request, user uuid.UUID) (string, time.Time, error) {
	var tz string
	if err := h.db.QueryRow(r.Context(), `SELECT timezone FROM users WHERE id = $1`, user).Scan(&tz); err != nil {
		return "", time.Time{}, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		tz, loc = "UTC", time.UTC
	}
	n := h.now().In(loc)
	return tz, time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC), nil
}

// weeksParam reads ?weeks= (4 to 104).
func weeksParam(w http.ResponseWriter, r *http.Request, def int) (int, bool) {
	v := r.URL.Query().Get("weeks")
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 4 || n > 104 {
		httpx.Problem(w, http.StatusBadRequest, "weeks is 4 to 104.")
		return 0, false
	}
	return n, true
}

// weekParam reads ?week=2026-W41, defaulting to the current week; future weeks are refused.
func weekParam(w http.ResponseWriter, r *http.Request, today time.Time) (time.Time, bool) {
	v := r.URL.Query().Get("week")
	if v == "" {
		return Monday(today), true
	}
	m, ok := ParseWeek(v)
	if !ok || m.After(today) {
		httpx.Problem(w, http.StatusBadRequest, "week is an ISO week like 2026-W41, not in the future.")
		return time.Time{}, false
	}
	return m, true
}

func (h *Handlers) weekly(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	weeks, ok := weeksParam(w, r, 12)
	if !ok {
		return
	}
	tz, today, err := h.clock(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	series, err := Weekly(r.Context(), h.db, who.UserID, tz, today, weeks)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"weeks": weeks, "time_zone": tz, "metrics": series})
}

func (h *Handlers) review(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	tz, today, err := h.clock(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	monday, ok := weekParam(w, r, today)
	if !ok {
		return
	}
	rev, err := WeeklyReview(r.Context(), h.db, who.UserID, tz, monday, today)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rev)
}

func (h *Handlers) items(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	m, ok := Find(chi.URLParam(r, "id"))
	if !ok {
		httpx.Problem(w, http.StatusNotFound, "No such metric.")
		return
	}
	tz, today, err := h.clock(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	monday, ok := weekParam(w, r, today)
	if !ok {
		return
	}
	items, err := Items(r.Context(), h.db, m, who.UserID, tz, monday, monday.AddDate(0, 0, 7), 200)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"metric": m, "week": Label(monday), "items": items})
}

func (h *Handlers) export(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	weeks, ok := weeksParam(w, r, 52)
	if !ok {
		return
	}
	tz, today, err := h.clock(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	series, err := Weekly(r.Context(), h.db, who.UserID, tz, today, weeks)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="axon-insights.csv"`)
	cw := csv.NewWriter(w)
	header := []string{"week", "monday"}
	for _, s := range series {
		header = append(header, s.ID)
	}
	_ = cw.Write(header)
	for i := range weeks {
		row := []string{series[0].Points[i].Week, series[0].Points[i].Monday}
		for _, s := range series {
			row = append(row, strconv.FormatFloat(s.Points[i].Value, 'f', -1, 64))
		}
		_ = cw.Write(row)
	}
	cw.Flush()
}
