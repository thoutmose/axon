package journal

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/l10n"
	"github.com/thoutmose/axon/api/internal/privacy"
)

var errNoConsent = errors.New("no health consent")

// Handlers serves the journal.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the journal endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/journal", h.list)
	r.Get("/v1/journal/helps", h.helps)
	r.Get("/v1/journal/{day}", h.get)
	r.Put("/v1/journal/{day}", h.put)
	r.Delete("/v1/journal/{day}", h.delete)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, errNoConsent):
		httpx.Problem(w, http.StatusForbidden, "The journal keeps health data (mood and sleep). Turn on the health data consent to keep it.")
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "Nothing written that day.")
	default:
		h.log.Error("journal request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// guard returns the person and their today, or writes 403 without the consent.
func (h *Handlers) guard(w http.ResponseWriter, r *http.Request) (uuid.UUID, time.Time, bool) {
	who, _ := auth.FromContext(r.Context())
	ok, err := privacy.Granted(r.Context(), h.db, who.UserID, Consent)
	if err == nil && !ok {
		err = errNoConsent
	}
	if err != nil {
		h.fail(w, err)
		return who.UserID, time.Time{}, false
	}
	today, _, err := clock.Today(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return who.UserID, time.Time{}, false
	}
	return who.UserID, today, true
}

func day(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	d, err := time.Parse(time.DateOnly, chi.URLParam(r, "day"))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The day looks like 2026-10-12.")
		return d, false
	}
	return d, true
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	user, today, ok := h.guard(w, r)
	if !ok {
		return
	}
	from, to := today.AddDate(0, 0, -29), today
	for _, p := range []struct {
		name string
		dst  *time.Time
	}{{"from", &from}, {"to", &to}} {
		if v := r.URL.Query().Get(p.name); v != "" {
			d, err := time.Parse(time.DateOnly, v)
			if err != nil {
				httpx.Problem(w, http.StatusBadRequest, "?from= and ?to= must look like 2026-10-10.")
				return
			}
			*p.dst = d
		}
	}
	if to.Before(from) || to.Sub(from) > 400*24*time.Hour {
		httpx.Problem(w, http.StatusBadRequest, "?from= comes before ?to=, at most 400 days apart.")
		return
	}
	items, err := List(r.Context(), h.db, user, from.Format(time.DateOnly), to.Format(time.DateOnly))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "today": today.Format(time.DateOnly)})
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	user, _, ok := h.guard(w, r)
	if !ok {
		return
	}
	d, ok := day(w, r)
	if !ok {
		return
	}
	e, err := Get(r.Context(), h.db, user, d.Format(time.DateOnly))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

func (h *Handlers) put(w http.ResponseWriter, r *http.Request) {
	user, today, ok := h.guard(w, r)
	if !ok {
		return
	}
	d, ok := day(w, r)
	if !ok {
		return
	}
	var f Fields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var e Entry
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) (err error) {
		e, err = Put(r.Context(), tx, user, d, today, f)
		return err
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, e)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	user, _, ok := h.guard(w, r)
	if !ok {
		return
	}
	d, ok := day(w, r)
	if !ok {
		return
	}
	if err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		return Delete(r.Context(), tx, user, d.Format(time.DateOnly))
	}); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) helps(w http.ResponseWriter, r *http.Request) {
	user, today, ok := h.guard(w, r)
	if !ok {
		return
	}
	loc, err := clock.Zone(r.Context(), h.db, user)
	if err != nil {
		h.fail(w, err)
		return
	}
	out, err := WhatHelps(r.Context(), h.db, user, today, loc.String(), l10n.Lang(r.Context()))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
