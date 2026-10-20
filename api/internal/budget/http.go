package budget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
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
	r.Get("/v1/budget/accounts", h.accounts)
	r.Post("/v1/budget/accounts", h.createAccount)
	r.Patch("/v1/budget/accounts/{id}", h.updateAccount)
	r.Delete("/v1/budget/accounts/{id}", h.deleteAccount)
	r.Get("/v1/budget/recurring", h.recurring)
	r.Post("/v1/budget/recurring", h.createRecurring)
	r.Patch("/v1/budget/recurring/{id}", h.updateRecurring)
	r.Delete("/v1/budget/recurring/{id}", h.deleteRecurring)
	r.Post("/v1/budget/recurring/run", h.runRecurring)
	r.Get("/v1/budget/rules", h.rules)
	r.Post("/v1/budget/rules", h.createRule)
	r.Patch("/v1/budget/rules/{id}", h.updateRule)
	r.Delete("/v1/budget/rules/{id}", h.deleteRule)
	r.Post("/v1/budget/rules/apply", h.applyRules)
	r.Post("/v1/budget/import", h.importCSV)
	r.Get("/v1/budget/report", h.report)
	r.Get("/v1/budget/pots", h.pots)
	r.Post("/v1/budget/pots", h.createPot)
	r.Patch("/v1/budget/pots/{id}", h.updatePot)
	r.Delete("/v1/budget/pots/{id}", h.deletePot)
	r.Post("/v1/budget/pots/{id}/moves", h.movePot)
	r.Delete("/v1/budget/pots/{id}/moves/{move}", h.deletePotMove)
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var v ValidationError
	switch {
	case errors.As(err, &v):
		httpx.Problem(w, http.StatusUnprocessableEntity, v.Msg)
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, http.StatusNotFound, "No such category, account, rule, or transaction.")
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

// catchUp records due recurring transactions before showing the budget.
func (h *Handlers) catchUp(r *http.Request, userID uuid.UUID) (time.Time, error) {
	today, err := Today(r.Context(), h.db, userID, h.now())
	if err != nil {
		return today, err
	}
	_, err = RunRecurring(r.Context(), h.db, userID, today)
	return today, err
}

func (h *Handlers) summary(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	today, err := h.catchUp(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	m, err := Summary(r.Context(), h.db, who.UserID, h.month(r), today)
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
	var f Filter
	for key, dst := range map[string]**uuid.UUID{"category": &f.Category, "account": &f.Account} {
		if c := r.URL.Query().Get(key); c != "" {
			id, err := uuid.Parse(c)
			if err != nil {
				httpx.Problem(w, http.StatusBadRequest, fmt.Sprintf("?%s= must be an id.", key))
				return
			}
			*dst = &id
		}
	}
	f.Tag = r.URL.Query().Get("tag")
	items, err := Transactions(r.Context(), h.db, who.UserID, h.month(r), f)
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
	return f, run(append(f.steps(raw), raw.rest)...)
}

func (f *TransactionFields) steps(raw fields) []func() error {
	return []func() error{
		func() error { _, err := raw.take("occurred_on", &f.OccurredOn, false); return err },
		func() error { _, err := raw.take("kind", &f.Kind, false); return err },
		func() error { _, err := raw.take("amount_cents", &f.AmountCents, false); return err },
		func() (err error) { f.ClearCategory, err = raw.take("category_id", &f.CategoryID, true); return err },
		func() (err error) { f.ClearAccount, err = raw.take("account_id", &f.AccountID, true); return err },
		func() (err error) {
			f.ClearToAccount, err = raw.take("to_account_id", &f.ToAccountID, true)
			return err
		},
		func() error { _, err := raw.take("tags", &f.Tags, false); return err },
		func() error { _, err := raw.take("payee", &f.Payee, false); return err },
		func() error { _, err := raw.take("note", &f.Note, false); return err },
	}
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

// --- Accounts ---

func (h *Handlers) accounts(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := Accounts(r.Context(), h.db, who.UserID)
	h.list(w, items, err)
}

func (h *Handlers) list(w http.ResponseWriter, items any, err error) {
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handlers) reply(w http.ResponseWriter, code int, v any, err error) {
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.JSON(w, code, v)
}

func accountFields(r *http.Request) (AccountFields, error) {
	raw, err := decode(r)
	if err != nil {
		return AccountFields{}, err
	}
	var f AccountFields
	return f, run(
		func() error { _, err := raw.take("name", &f.Name, false); return err },
		func() error { _, err := raw.take("type", &f.Type, false); return err },
		func() error { _, err := raw.take("role", &f.Role, false); return err },
		func() error { _, err := raw.take("opening_cents", &f.OpeningCents, false); return err },
		func() error { _, err := raw.take("archived", &f.Archived, false); return err },
		raw.rest,
	)
}

func (h *Handlers) createAccount(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := accountFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	a, err := CreateAccount(r.Context(), h.db, who.UserID, f)
	h.reply(w, http.StatusCreated, a, err)
}

func (h *Handlers) updateAccount(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := accountFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	a, err := UpdateAccount(r.Context(), h.db, who.UserID, id, f)
	h.reply(w, http.StatusOK, a, err)
}

func (h *Handlers) deleteAccount(w http.ResponseWriter, r *http.Request) {
	h.remove(w, r, DeleteAccount)
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request, del func(context.Context, db.Querier, uuid.UUID, uuid.UUID) error) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err == nil {
		err = del(r.Context(), h.db, who.UserID, id)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Recurring ---

func (h *Handlers) recurring(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := ListRecurring(r.Context(), h.db, who.UserID)
	h.list(w, items, err)
}

func recurringFields(r *http.Request) (RecurringFields, error) {
	raw, err := decode(r)
	if err != nil {
		return RecurringFields{}, err
	}
	var f RecurringFields
	steps := append(f.steps(raw),
		func() error { _, err := raw.take("name", &f.Name, false); return err },
		func() error { _, err := raw.take("rule", &f.Rule, false); return err },
		func() error { _, err := raw.take("start_on", &f.Start, false); return err },
		func() error { _, err := raw.take("active", &f.Active, false); return err },
		func() error {
			if f.OccurredOn != nil {
				return ValidationError{"Use start_on for the first occurrence."}
			}
			return nil
		},
		raw.rest)
	return f, run(steps...)
}

func (h *Handlers) createRecurring(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := recurringFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	x, err := CreateRecurring(r.Context(), h.db, who.UserID, f)
	h.reply(w, http.StatusCreated, x, err)
}

func (h *Handlers) updateRecurring(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := recurringFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	x, err := UpdateRecurring(r.Context(), h.db, who.UserID, id, f)
	h.reply(w, http.StatusOK, x, err)
}

func (h *Handlers) deleteRecurring(w http.ResponseWriter, r *http.Request) {
	h.remove(w, r, DeleteRecurring)
}

func (h *Handlers) runRecurring(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	today, err := Today(r.Context(), h.db, who.UserID, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	n, err := RunRecurring(r.Context(), h.db, who.UserID, today)
	h.reply(w, http.StatusOK, map[string]int{"created": n}, err)
}

// --- Rules ---

func (h *Handlers) rules(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	items, err := Rules(r.Context(), h.db, who.UserID)
	h.list(w, items, err)
}

func ruleFields(r *http.Request) (RuleFields, error) {
	raw, err := decode(r)
	if err != nil {
		return RuleFields{}, err
	}
	var f RuleFields
	return f, run(
		func() error { _, err := raw.take("name", &f.Name, false); return err },
		func() error { _, err := raw.take("field", &f.Field, false); return err },
		func() error { _, err := raw.take("operator", &f.Operator, false); return err },
		func() error { _, err := raw.take("value", &f.Value, false); return err },
		func() (err error) { f.ClearCategory, err = raw.take("category_id", &f.CategoryID, true); return err },
		func() error { _, err := raw.take("tag", &f.Tag, false); return err },
		func() error { _, err := raw.take("position", &f.Position, false); return err },
		func() error { _, err := raw.take("active", &f.Active, false); return err },
		raw.rest,
	)
}

func (h *Handlers) createRule(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, err := ruleFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	x, err := CreateRule(r.Context(), h.db, who.UserID, f)
	h.reply(w, http.StatusCreated, x, err)
}

func (h *Handlers) updateRule(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, err := ruleFields(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	x, err := UpdateRule(r.Context(), h.db, who.UserID, id, f)
	h.reply(w, http.StatusOK, x, err)
}

func (h *Handlers) deleteRule(w http.ResponseWriter, r *http.Request) {
	h.remove(w, r, DeleteRule)
}

func (h *Handlers) applyRules(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	n, err := ApplyRules(r.Context(), h.db, who.UserID)
	h.reply(w, http.StatusOK, map[string]int{"changed": n}, err)
}

// --- Import and report ---

// maxImport bounds an import request body (the CSV travels as JSON text).
const maxImport = 4 << 20

func (h *Handlers) importCSV(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var spec ImportSpec
	if err := httpx.DecodeN(r, &spec, maxImport); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	var res ImportResult
	err := pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) (err error) {
		res, err = Import(r.Context(), tx, who.UserID, spec)
		return err
	})
	h.reply(w, http.StatusOK, res, err)
}

func (h *Handlers) report(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	months := 12
	if m := r.URL.Query().Get("months"); m != "" {
		n, err := strconv.Atoi(m)
		if err != nil || n < 1 || n > 60 {
			httpx.Problem(w, http.StatusBadRequest, "?months= must be between 1 and 60.")
			return
		}
		months = n
	}
	today, err := h.catchUp(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	// ?month=2026-09 ends the report with that month instead of this one (EP-138).
	if m := r.URL.Query().Get("month"); m != "" {
		d, err := time.Parse("2006-01", m)
		if err != nil {
			httpx.Problem(w, http.StatusBadRequest, "?month= looks like 2026-09.")
			return
		}
		today = d
	}
	rep, err := BuildReport(r.Context(), h.db, who.UserID, today, months)
	h.reply(w, http.StatusOK, rep, err)
}

// --- Savings pots ---

func (h *Handlers) today(r *http.Request, user uuid.UUID) (time.Time, error) {
	return Today(r.Context(), h.db, user, h.now())
}

func (h *Handlers) pots(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	items, err := Pots(r.Context(), h.db, who.UserID, today)
	h.reply(w, http.StatusOK, map[string]any{"items": items}, err)
}

func (h *Handlers) potFields(w http.ResponseWriter, r *http.Request) (PotFields, time.Time, bool) {
	who, _ := auth.FromContext(r.Context())
	var f PotFields
	if err := httpx.Decode(r, &f); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return f, time.Time{}, false
	}
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return f, today, false
	}
	return f, today, true
}

func (h *Handlers) createPot(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	f, today, ok := h.potFields(w, r)
	if !ok {
		return
	}
	p, err := CreatePot(r.Context(), h.db, who.UserID, f, today)
	h.reply(w, http.StatusCreated, p, err)
}

func (h *Handlers) updatePot(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	f, today, ok := h.potFields(w, r)
	if !ok {
		return
	}
	p, err := UpdatePot(r.Context(), h.db, who.UserID, id, f, today)
	h.reply(w, http.StatusOK, p, err)
}

func (h *Handlers) deletePot(w http.ResponseWriter, r *http.Request) {
	h.remove(w, r, DeletePot)
}

func (h *Handlers) movePot(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	var m PotMove
	if err := httpx.Decode(r, &m); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	var p Pot
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		p, err = MovePot(r.Context(), tx, who.UserID, id, m, today)
		return err
	})
	h.reply(w, http.StatusCreated, p, err)
}

func (h *Handlers) deletePotMove(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := pathID(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	move, err := uuid.Parse(chi.URLParam(r, "move"))
	if err != nil {
		h.fail(w, ErrNotFound)
		return
	}
	today, err := h.today(r, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	var p Pot
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		var err error
		p, err = DeletePotMove(r.Context(), tx, who.UserID, id, move, today)
		return err
	})
	h.reply(w, http.StatusOK, p, err)
}
