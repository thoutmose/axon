package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the budget.
type Handlers struct {
	db  *pgxpool.Pool
	log *slog.Logger
	now func() time.Time
}

// NewHandlers returns the budget endpoints.
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger) *Handlers {
	return &Handlers{db: pool, log: log, now: time.Now}
}

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/budget", h.summary)
	r.Put("/v1/budget/currency", h.currency)
	r.Get("/v1/budget/categories", h.categories)
	r.Post("/v1/budget/categories", h.createCategory)
	r.Patch("/v1/budget/categories/{id}", h.updateCategory)
	r.Delete("/v1/budget/categories/{id}", h.deleteCategory)
	r.Get("/v1/budget/transactions", h.transactions)
	r.Post("/v1/budget/transactions", h.createTransaction)
	r.Patch("/v1/budget/transactions/{id}", h.updateTransaction)
	r.Delete("/v1/budget/transactions/{id}", h.deleteTransaction)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such category or transaction.")
	default:
		h.log.Error("budget request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

// fields reads a JSON object, telling "absent" from "null".
type fields map[string]json.RawMessage

func decode(r *http.Request) (fields, error) {
	var raw fields
	if err := httpx.Decode(r, &raw); err != nil {
		return nil, ValidationError{err.Error()}
	}
	return raw, nil
}

func (f fields) take(key string, dst any, nullable bool) (null bool, err error) {
	v, ok := f[key]
	if !ok {
		return false, nil
	}
	delete(f, key)
	if string(v) == "null" {
		if !nullable {
			return false, ValidationError{fmt.Sprintf("%s cannot be null.", key)}
		}
		return true, nil
	}
	if err := json.Unmarshal(v, dst); err != nil {
		return false, ValidationError{fmt.Sprintf("%s has the wrong type.", key)}
	}
	return false, nil
}

func (f fields) rest() error {
	for key := range f {
		return ValidationError{fmt.Sprintf("Unknown field %q.", key)}
	}
	return nil
}

func run(steps ...func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return id, ErrNotFound
	}
	return id, nil
}

// month is ?month=, or the current month in UTC.
func (h *Handlers) month(r *http.Request) string {
	if m := r.URL.Query().Get("month"); m != "" {
		return m
	}
	return h.now().UTC().Format("2006-01")
}

func (h *Handlers) summary(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	m, err := Summary(r.Context(), h.db, who.UserID, h.month(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

func (h *Handlers) currency(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var body struct {
		Currency string `json:"currency"`
	}
	if err := httpx.Decode(r, &body); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := SetCurrency(r.Context(), h.db, who.UserID, body.Currency); err != nil {
		h.fail(w, err)
		return
	}
	c, err := Currency(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"currency": c})
}

func (h *Handlers) categories(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := Categories(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func categoryFields(r *http.Request) (CategoryFields, error) {
	raw, err := decode(r)
	if err != nil {
		return CategoryFields{}, err
	}
	var f CategoryFields
	return f, run(
		func() error { _, err := raw.take("name", &f.Name, false); return err },
		func() error { _, err := raw.take("kind", &f.Kind, false); return err },
		func() (err error) { f.ClearMonthly, err = raw.take("monthly_cents", &f.MonthlyCents, true); return err },
		func() error { _, err := raw.take("archived", &f.Archived, false); return err },
		raw.rest,
	)
}

func (h *Handlers) createCategory(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := categoryFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := CreateCategory(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handlers) updateCategory(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := categoryFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	c, err := UpdateCategory(r.Context(), h.db, who.UserID, id, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handlers) deleteCategory(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err == nil {
		err = DeleteCategory(r.Context(), h.db, who.UserID, id)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) transactions(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var category *uuid.UUID
	if c := r.URL.Query().Get("category"); c != "" {
		id, err := uuid.Parse(c)
		if err != nil {
			httpx.Problem(w, http.StatusBadRequest, "?category= must be a category id.")
			return
		}
		category = &id
	}
	items, err := Transactions(r.Context(), h.db, who.UserID, h.month(r), category)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func transactionFields(r *http.Request) (TransactionFields, error) {
	raw, err := decode(r)
	if err != nil {
		return TransactionFields{}, err
	}
	var f TransactionFields
	return f, run(
		func() error { _, err := raw.take("occurred_on", &f.OccurredOn, false); return err },
		func() error { _, err := raw.take("kind", &f.Kind, false); return err },
		func() error { _, err := raw.take("amount_cents", &f.AmountCents, false); return err },
		func() (err error) { f.ClearCategory, err = raw.take("category_id", &f.CategoryID, true); return err },
		func() error { _, err := raw.take("payee", &f.Payee, false); return err },
		func() error { _, err := raw.take("note", &f.Note, false); return err },
		raw.rest,
	)
}

func (h *Handlers) createTransaction(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := transactionFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	t, err := CreateTransaction(r.Context(), h.db, who.UserID, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handlers) updateTransaction(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := transactionFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	t, err := UpdateTransaction(r.Context(), h.db, who.UserID, id, f)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handlers) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err == nil {
		err = DeleteTransaction(r.Context(), h.db, who.UserID, id)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
