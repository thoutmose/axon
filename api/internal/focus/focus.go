// Package focus records Pomodoro focus sessions (EP-35). The timer runs in
// the browser; the server keeps what was worked on, for how long, and links
// each session to its task, project, skill, note, or any other entity.
package focus

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Type is the entity type; FocusedOn links a session to what it was for.
const (
	Type      = "focus"
	FocusedOn = "focused_on"
)

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "focus_sessions", Export: export})
	graph.RegisterRelation(graph.Relation{Name: FocusedOn, Sources: []string{Type}})
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Target is what a session was for.
type Target struct {
	ID    uuid.UUID `json:"id"`
	Type  string    `json:"type"`
	Title string    `json:"title"`
}

// Session is one focus session.
type Session struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	PlannedMinutes int        `json:"planned_minutes"`
	WorkedMinutes  int        `json:"worked_minutes"`
	Rounds         int        `json:"rounds"`
	Status         string     `json:"status"`
	Target         *Target    `json:"target"`
}

const columns = `s.id, e.title, s.started_at, s.ended_at, s.planned_minutes, s.worked_minutes, s.rounds, s.status,
	t.id, t.type, t.title`

const from = ` FROM focus_sessions s JOIN entities e ON e.id = s.id
	LEFT JOIN links l ON l.source_id = s.id AND l.relation = 'focused_on'
	LEFT JOIN entities t ON t.id = l.target_id`

func scan(row pgx.Row) (Session, error) {
	var s Session
	var tid *uuid.UUID
	var ttype, ttitle *string
	err := row.Scan(&s.ID, &s.Title, &s.StartedAt, &s.EndedAt, &s.PlannedMinutes, &s.WorkedMinutes, &s.Rounds, &s.Status, &tid, &ttype, &ttitle)
	if tid != nil {
		s.Target = &Target{*tid, *ttype, *ttitle}
	}
	return s, err
}

// Get returns one of the person's sessions.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Session, error) {
	s, err := scan(q.QueryRow(ctx, `SELECT `+columns+from+` WHERE s.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return s, graph.ErrNotFound
	}
	return s, err
}

// List returns sessions started in [since, until), newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, since, until time.Time) ([]Session, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+from+` WHERE e.user_id = $1 AND s.started_at >= $2 AND s.started_at < $3
		ORDER BY s.started_at DESC LIMIT 500`, userID, since, until)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Session, error) { return scan(r) })
	if out == nil {
		out = []Session{}
	}
	return out, err
}

// Start records a new running session, linked to target when given.
func Start(ctx context.Context, tx pgx.Tx, userID uuid.UUID, title string, target *uuid.UUID, planned int, now time.Time) (Session, error) {
	title = strings.TrimSpace(title)
	if planned < 1 || planned > 1440 {
		return Session{}, ValidationError{"Plan between 1 minute and 24 hours."}
	}
	if len([]rune(title)) > 500 {
		return Session{}, ValidationError{"The title is up to 500 characters."}
	}
	if title == "" && target != nil {
		var t string
		if err := tx.QueryRow(ctx, `SELECT title FROM entities WHERE id = $1 AND user_id = $2`, *target, userID).Scan(&t); err != nil {
			return Session{}, ValidationError{"That item does not exist."}
		}
		title = t
	}
	if title == "" {
		title = "Focus"
	}
	id, err := graph.CreateEntity(ctx, tx, userID, graph.UserActor, Type, title, "")
	if err != nil {
		return Session{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO focus_sessions (id, started_at, planned_minutes) VALUES ($1, $2, $3)`, id, now, planned); err != nil {
		return Session{}, err
	}
	if target != nil {
		if _, err := graph.CreateLink(ctx, tx, userID, graph.UserActor, id, FocusedOn, *target); err != nil {
			if errors.Is(err, graph.ErrNotFound) {
				return Session{}, ValidationError{"That item does not exist."}
			}
			return Session{}, err
		}
	}
	return Get(ctx, tx, userID, id)
}

// Finish records how a session ended.
func Finish(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, worked, rounds int, status string, now time.Time) (Session, error) {
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return cur, err
	}
	switch {
	case status != "completed" && status != "stopped":
		return cur, ValidationError{"A session ends completed or stopped."}
	case worked < 0 || worked > 1440 || rounds < 0 || rounds > 100:
		return cur, ValidationError{"Worked minutes are 0 to 1,440 and rounds 0 to 100."}
	case cur.Status != "running":
		return cur, ValidationError{"This session has already ended."}
	case float64(worked) > now.Sub(cur.StartedAt).Minutes()+1:
		return cur, ValidationError{"You cannot have worked longer than the session has run."}
	}
	if _, err := tx.Exec(ctx, `UPDATE focus_sessions SET worked_minutes = $2, rounds = $3, status = $4, ended_at = $5,
		updated_at = clock_timestamp() WHERE id = $1`, id, worked, rounds, status, now); err != nil {
		return cur, err
	}
	return Get(ctx, tx, userID, id)
}

// Stats summarise focus over a period and the current daily streak.
type Stats struct {
	Minutes  int `json:"minutes"`
	Sessions int `json:"sessions"`
	Streak   int `json:"streak"` // days in a row, up to today, with focus time
}

// GetStats counts worked minutes since a time and the streak in a time zone.
func GetStats(ctx context.Context, q db.Querier, userID uuid.UUID, since time.Time, tz string, today time.Time) (Stats, error) {
	var st Stats
	if err := q.QueryRow(ctx, `SELECT coalesce(sum(s.worked_minutes), 0), count(*) FROM focus_sessions s JOIN entities e ON e.id = s.id
		WHERE e.user_id = $1 AND s.started_at >= $2 AND s.worked_minutes > 0`, userID, since).Scan(&st.Minutes, &st.Sessions); err != nil {
		return st, err
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT (s.started_at AT TIME ZONE $2)::date FROM focus_sessions s JOIN entities e ON e.id = s.id
		WHERE e.user_id = $1 AND s.worked_minutes > 0 AND s.started_at > now() - interval '400 days'`, userID, tz)
	if err != nil {
		return st, err
	}
	days, err := pgx.CollectRows(rows, pgx.RowTo[time.Time])
	if err != nil {
		return st, err
	}
	have := map[string]bool{}
	for _, d := range days {
		have[d.Format(time.DateOnly)] = true
	}
	d := today
	if !have[d.Format(time.DateOnly)] {
		d = d.AddDate(0, 0, -1) // today can still be focused on
	}
	for have[d.Format(time.DateOnly)] {
		st.Streak++
		d = d.AddDate(0, 0, -1)
	}
	return st, nil
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, time.Unix(0, 0), time.Now().AddDate(1, 0, 0))
}

// Handlers serves the focus endpoints.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the focus endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/focus-sessions", h.list)
	r.Post("/v1/focus-sessions", h.start)
	r.Get("/v1/focus-sessions/{id}", h.get)
	r.Post("/v1/focus-sessions/{id}/finish", h.finish)
	r.Delete("/v1/focus-sessions/{id}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, graph.ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such focus session.")
	default:
		h.log.Error("focus request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) id(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, graph.ErrNotFound
	}
	return id, nil
}

func (h *Handlers) tz(ctx context.Context, userID uuid.UUID) (string, *time.Location, error) {
	var tz string
	if err := h.db.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err != nil {
		return "", nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return tz, loc, nil
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	tz, loc, err := h.tz(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	now := h.now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	since := today.AddDate(0, 0, -6)
	items, err := List(r.Context(), h.db, who.UserID, since, now.Add(time.Minute))
	if err != nil {
		h.fail(w, err)
		return
	}
	week, err := GetStats(r.Context(), h.db, who.UserID, since, tz, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
	if err != nil {
		h.fail(w, err)
		return
	}
	todayStats, err := GetStats(r.Context(), h.db, who.UserID, today, tz, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "today_minutes": todayStats.Minutes, "week_minutes": week.Minutes, "streak": week.Streak})
}

func (h *Handlers) start(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Title          string     `json:"title"`
		TargetID       *uuid.UUID `json:"target_id"`
		PlannedMinutes int        `json:"planned_minutes"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var s Session
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		s, err = Start(r.Context(), tx, who.UserID, body.Title, body.TargetID, body.PlannedMinutes, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, s)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := h.id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	s, err := Get(r.Context(), h.db, who.UserID, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handlers) finish(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := h.id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var body struct {
		WorkedMinutes int    `json:"worked_minutes"`
		Rounds        int    `json:"rounds"`
		Status        string `json:"status"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var s Session
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		s, err = Finish(r.Context(), tx, who.UserID, id, body.WorkedMinutes, body.Rounds, body.Status, h.now())
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := h.id(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		if _, err := Get(r.Context(), tx, who.UserID, id); err != nil {
			return err
		}
		return graph.DeleteEntity(r.Context(), tx, who.UserID, id, graph.UserActor)
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
