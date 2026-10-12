package progress

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/settings"
)

// FreezeCost is the price of a streak freeze in earned points; MaxFreezes is
// how many can wait in stock.
const (
	FreezeCost = 100
	MaxFreezes = 2
)

// Summary is the person's progress.
type Summary struct {
	Enabled   bool     `json:"enabled"`
	XP        int      `json:"xp"`
	TodayXP   int      `json:"today_xp"`
	Level     int      `json:"level"`
	LevelFrom int      `json:"level_from"`
	LevelTo   int      `json:"level_to"`
	Points    int      `json:"points"` // XP not yet spent on freezes
	Streak    *Streak  `json:"streak"`
	Badges    []Badge  `json:"badges"`
	Message   string   `json:"message"`
	Areas     []string `json:"areas"` // where points come from, for the person to see everything counts
}

// Handlers serves progress.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the progress endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/progress", h.summary)
	r.Get("/v1/progress/ledger", h.ledger)
	r.Get("/v1/progress/rules", h.rules)
	r.Post("/v1/progress/freezes", h.buyFreeze)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	h.log.Error("progress request failed", "err", err)
	httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

// message is encouragement without loss aversion or shame.
func message(st Streak) string {
	switch st.State {
	case Active:
		return "Today counts. Nice."
	case AtRisk:
		return "Any small step today keeps your run going: a task, a habit check-in, a few cards, or a short focus session."
	case Frozen:
		return "A freeze covered the day you missed. Pick up whenever you are ready."
	case Broken:
		return "Fresh start: one small action today begins a new run."
	default:
		return "Points appear as you complete tasks, keep habits, focus, review cards, write notes, and learn skills."
	}
}

func (h *Handlers) zone(ctx context.Context, userID uuid.UUID) (string, time.Time, error) {
	today, loc, err := clock.Today(ctx, h.db, userID, h.now())
	return loc.String(), today, err
}

// Compute builds the summary and saves the freezes the streak consumed.
func Compute(ctx context.Context, q db.Querier, userID uuid.UUID, tz string, today time.Time) (Summary, error) {
	s := Summary{Enabled: true, Areas: []string{"tasks", "projects", "habits", "focus", "flashcards", "notes", "roadmaps", "goals"}}
	if err := q.QueryRow(ctx, `SELECT coalesce(sum(points), 0)::int, coalesce(sum(points) FILTER (WHERE (at AT TIME ZONE $4)::date = $5), 0)::int
		FROM (`+ledger+`) l(at, rule, points, id, title)`, userID, epoch, far(), tz, today).Scan(&s.XP, &s.TodayXP); err != nil {
		return s, err
	}
	s.Level, s.LevelFrom, s.LevelTo = Level(s.XP)
	rows, err := q.Query(ctx, `SELECT DISTINCT (at AT TIME ZONE $4)::date AS d FROM (`+ledger+`) l(at, rule, points, id, title)
		WHERE points > 0 ORDER BY d`, userID, today.AddDate(0, 0, -400), far(), tz)
	if err != nil {
		return s, err
	}
	days, err := pgx.CollectRows(rows, pgx.RowTo[time.Time])
	if err != nil {
		return s, err
	}
	active := map[string]bool{}
	for _, d := range days {
		active[dayKey(d)] = true
	}
	rows, err = q.Query(ctx, `SELECT id::text, bought_on, used_on FROM streak_freezes WHERE user_id = $1`, userID)
	if err != nil {
		return s, err
	}
	freezes, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Freeze, error) {
		var f Freeze
		return f, r.Scan(&f.ID, &f.BoughtOn, &f.UsedOn)
	})
	if err != nil {
		return s, err
	}
	st := ComputeStreak(active, today, freezes)
	for _, u := range st.Used {
		if _, err := q.Exec(ctx, `UPDATE streak_freezes SET used_on = $2 WHERE id = $1 AND used_on IS NULL`, u.ID, u.Day); err != nil {
			return s, err
		}
	}
	s.Streak = &st
	s.Points = s.XP - FreezeCost*len(freezes)
	s.Message = message(st)
	s.Badges, err = Badges(ctx, q, userID, tz, days)
	return s, err
}

func (h *Handlers) enabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	d, err := settings.Get(ctx, h.db, userID)
	return d.Gamification.Enabled, err
}

func (h *Handlers) summary(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	on, err := h.enabled(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	if !on {
		httpx.JSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	tz, today, err := h.zone(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	s, err := Compute(r.Context(), h.db, who.UserID, tz, today)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handlers) ledger(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if on, err := h.enabled(r.Context(), who.UserID); err != nil {
		h.fail(w, err)
		return
	} else if !on {
		httpx.Problem(w, http.StatusConflict, "Progress rewards are off. Turn them on in Preferences.")
		return
	}
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 3650 {
			httpx.Problem(w, http.StatusBadRequest, "?days= is 1 to 3,650.")
			return
		}
		days = n
	}
	tz, _, err := h.zone(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := Ledger(r.Context(), h.db, who.UserID, tz, h.now().AddDate(0, 0, -days), far(), 500)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) rules(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"items": Rules, "freeze_cost": FreezeCost, "max_freezes": MaxFreezes})
}

func (h *Handlers) buyFreeze(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if on, err := h.enabled(r.Context(), who.UserID); err != nil {
		h.fail(w, err)
		return
	} else if !on {
		httpx.Problem(w, http.StatusConflict, "Progress rewards are off. Turn them on in Preferences.")
		return
	}
	tz, today, err := h.zone(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	var s Summary
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1, 15))`, who.UserID.String()); err != nil {
			return err
		}
		cur, err := Compute(r.Context(), tx, who.UserID, tz, today)
		if err != nil {
			return err
		}
		switch {
		case cur.Streak.Freezes >= MaxFreezes:
			return errFull
		case cur.Points < FreezeCost:
			return errPoor
		}
		if _, err := tx.Exec(r.Context(), `INSERT INTO streak_freezes (id, user_id, bought_on) VALUES ($1, $2, $3)`,
			uuid.Must(uuid.NewV7()), who.UserID, today); err != nil {
			return err
		}
		s, err = Compute(r.Context(), tx, who.UserID, tz, today)
		return err
	})
	switch {
	case errors.Is(err, errFull):
		httpx.Problem(w, http.StatusConflict, "You already hold two freezes, the most you can keep.")
	case errors.Is(err, errPoor):
		httpx.Problem(w, http.StatusConflict, "A freeze costs 100 earned points. Points come from what you do, never from money.")
	case err != nil:
		h.fail(w, err)
	default:
		httpx.JSON(w, http.StatusCreated, s)
	}
}

var (
	errFull = errors.New("freezes full")
	errPoor = errors.New("not enough points")
)

// Export is the freezes part of a personal data export (points are derived).
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	type row struct {
		BoughtAt time.Time  `json:"bought_at"`
		UsedOn   *time.Time `json:"used_on"`
	}
	rows, err := q.Query(ctx, `SELECT bought_at, used_on FROM streak_freezes WHERE user_id = $1 ORDER BY bought_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[row])
}
