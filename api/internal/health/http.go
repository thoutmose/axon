package health

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/privacy"
)

// lookupsPerMinute bounds food lookups per person.
const lookupsPerMinute = 10

var (
	errNoConsent       = errors.New("no health consent")
	errNoLookupConsent = errors.New("no food lookup consent")
)

// Handlers serves the health endpoints.
type Handlers struct {
	db       *pgxpool.Pool
	log      *slog.Logger
	catalogs *Catalogs
	now      func() time.Time

	mu    sync.Mutex
	quota map[uuid.UUID][]time.Time
}

// NewHandlers returns the health endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, catalogs *Catalogs) *Handlers {
	return &Handlers{db: pool, log: log, catalogs: catalogs, now: time.Now, quota: map[uuid.UUID][]time.Time{}}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/health/status", h.status)
	r.Delete("/v1/health/data", h.deleteAll)
	r.Get("/v1/health/food", h.foodDay)
	r.Post("/v1/health/food", h.addEntry)
	r.Delete("/v1/health/food/{id}", h.deleteEntry)
	r.Get("/v1/health/foods/search", h.search)
	r.Get("/v1/health/foods/barcode/{code}", h.barcode)
	r.Get("/v1/health/workouts", h.workouts)
	r.Post("/v1/health/workouts", h.createWorkout)
	r.Put("/v1/health/workouts/{id}", h.replaceWorkout)
	r.Delete("/v1/health/workouts/{id}", h.deleteWorkout)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, errNoConsent):
		httpx.Problem(w, http.StatusForbidden, "Health logging is off. Turn on the health data consent to keep food and training logs.")
	case errors.Is(err, errNoLookupConsent):
		httpx.Problem(w, http.StatusForbidden, "Food lookups are off. Allow them to search Open Food Facts and USDA, or type the numbers in.")
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Not found.")
	case errors.Is(err, ErrBusy):
		httpx.Problem(w, http.StatusTooManyRequests, "The food catalogue is busy. Try again in a few seconds, or type the numbers in.")
	case errors.Is(err, ErrUnavailable):
		h.log.Warn("food catalogue unavailable", "err", err)
		httpx.Problem(w, http.StatusServiceUnavailable, "The food catalogue cannot be reached right now. You can type the numbers in.")
	default:
		h.log.Error("health request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// guard returns the person, or writes 403 when the given consents are missing.
func (h *Handlers) guard(w http.ResponseWriter, r *http.Request, purposes ...string) (uuid.UUID, bool) {
	who, _ := auth.FromContext(r.Context())
	for _, p := range purposes {
		ok, err := privacy.Granted(r.Context(), h.db, who.UserID, p)
		if err == nil && !ok {
			err = errNoConsent
			if p == LookupConsent {
				err = errNoLookupConsent
			}
		}
		if err != nil {
			h.fail(w, err)
			return who.UserID, false
		}
	}
	return who.UserID, true
}

func (h *Handlers) allow(user uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := h.now().Add(-time.Minute)
	recent := slices.DeleteFunc(h.quota[user], func(t time.Time) bool { return t.Before(cutoff) })
	if len(recent) >= lookupsPerMinute {
		h.quota[user] = recent
		return false
	}
	h.quota[user] = append(recent, h.now())
	return true
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, ErrNotFound
	}
	return id, nil
}

// today is the person's date in their time zone.
func (h *Handlers) today(r *http.Request, user uuid.UUID) time.Time {
	loc, _ := clock.Zone(r.Context(), h.db, user) // UTC when it cannot be read
	return clock.Day(h.now(), loc)
}

func (h *Handlers) status(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	health, err := privacy.Granted(r.Context(), h.db, who.UserID, Consent)
	if err != nil {
		h.fail(w, err)
		return
	}
	lookups, err := privacy.Granted(r.Context(), h.db, who.UserID, LookupConsent)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"consented": health, "lookups_consented": lookups, "usda": h.catalogs.USDAEnabled()})
}

func (h *Handlers) deleteAll(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return DeleteAll(r.Context(), tx, who.UserID) }); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) foodDay(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	day := h.today(r, user)
	if d := r.URL.Query().Get("date"); d != "" {
		t, err := time.Parse(time.DateOnly, d)
		if err != nil {
			httpx.Problem(w, http.StatusBadRequest, "The date is YYYY-MM-DD.")
			return
		}
		day = t
	}
	d, err := FoodDay(r.Context(), h.db, user, day.Format(time.DateOnly))
	if err != nil {
		h.fail(w, err)
		return
	}
	week, err := FoodWeek(r.Context(), h.db, user, day)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"date": d.Date, "entries": d.Entries, "totals": d.Totals, "week": week})
}

func (h *Handlers) addEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	var in EntryInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var e Entry
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		e, err = AddEntry(r.Context(), tx, user, in)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, e)
}

func (h *Handlers) deleteEntry(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return DeleteEntry(r.Context(), tx, user, id) })
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) search(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, LookupConsent)
	if !ok {
		return
	}
	if !h.allow(user) {
		h.fail(w, ErrBusy)
		return
	}
	foods, err := h.catalogs.Search(r.Context(), r.URL.Query().Get("source"), r.URL.Query().Get("q"))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": foods})
}

func (h *Handlers) barcode(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, LookupConsent)
	if !ok {
		return
	}
	if !h.allow(user) {
		h.fail(w, ErrBusy)
		return
	}
	f, err := h.catalogs.Barcode(r.Context(), chi.URLParam(r, "code"))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, f)
}

func (h *Handlers) workouts(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	today := h.today(r, user)
	// The last 30 days, and anything dated later: a device in a time zone
	// ahead of the account's logs its "today" as tomorrow.
	from, to := today.AddDate(0, 0, -29).Format(time.DateOnly), today.AddDate(1, 0, 0).Format(time.DateOnly)
	if f := r.URL.Query().Get("from"); f != "" {
		from = f
	}
	if t := r.URL.Query().Get("to"); t != "" {
		to = t
	}
	if !validDate(from) || !validDate(to) || from > to {
		httpx.Problem(w, http.StatusBadRequest, "from and to are YYYY-MM-DD, from first.")
		return
	}
	items, err := Workouts(r.Context(), h.db, user, from, to)
	if err != nil {
		h.fail(w, err)
		return
	}
	week, err := TrainingSince(r.Context(), h.db, user, today.AddDate(0, 0, -6).Format(time.DateOnly))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "week": week})
}

func (h *Handlers) saveWorkout(w http.ResponseWriter, r *http.Request, id *uuid.UUID, status int) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	var in WorkoutInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var out Workout
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		out, err = SaveWorkout(r.Context(), tx, user, id, in)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, status, out)
}

func (h *Handlers) createWorkout(w http.ResponseWriter, r *http.Request) {
	h.saveWorkout(w, r, nil, http.StatusCreated)
}

func (h *Handlers) replaceWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.saveWorkout(w, r, &id, http.StatusOK)
}

func (h *Handlers) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	user, ok := h.guard(w, r, Consent)
	if !ok {
		return
	}
	id, err := pathID(r)
	if err == nil {
		err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error { return DeleteWorkout(r.Context(), tx, user, id) })
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
