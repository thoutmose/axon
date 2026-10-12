package reminders

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
	"github.com/thoutmose/axon/api/internal/settings"
)

// Routes mounts the endpoints; the caller wraps them in auth.Require.
func (w *Worker) Routes(r chi.Router) {
	r.Get("/v1/me/reminders", w.status)
	r.Post("/v1/me/push-subscriptions", w.subscribe)
	r.Delete("/v1/me/push-subscriptions/{id}", w.unsubscribe)
	r.Post("/v1/me/reminders/test", w.test)
}

func (w *Worker) fail(rw http.ResponseWriter, err error) {
	w.log.Error("reminders request failed", "err", err)
	httpx.Problem(rw, http.StatusInternalServerError, "Something went wrong. Try again.")
}

type device struct {
	ID         uuid.UUID  `json:"id"`
	Device     string     `json:"device"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSentAt *time.Time `json:"last_sent_at"`
}

// status returns the key browsers subscribe with, whether the instance can
// email, and the devices that receive reminders.
func (w *Worker) status(rw http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	v, err := w.key(r.Context())
	if err != nil {
		w.fail(rw, err)
		return
	}
	rows, err := w.db.Query(r.Context(), `SELECT id, device, created_at, last_sent_at FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at`, who.UserID)
	if err != nil {
		w.fail(rw, err)
		return
	}
	devices, err := pgx.CollectRows(rows, pgx.RowToStructByPos[device])
	if err != nil {
		w.fail(rw, err)
		return
	}
	if devices == nil {
		devices = []device{}
	}
	httpx.JSON(rw, http.StatusOK, map[string]any{"public_key": v.PublicKey(), "email": w.mail != nil, "devices": devices})
}

func (w *Worker) subscribe(rw http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	var b struct {
		Endpoint string `json:"endpoint"`
		P256dh   string `json:"p256dh"`
		Auth     string `json:"auth"`
		Device   string `json:"device"`
	}
	if err := httpx.Decode(r, &b); err != nil {
		httpx.Problem(rw, http.StatusBadRequest, err.Error())
		return
	}
	u, err := url.Parse(b.Endpoint)
	if err != nil || len(b.Endpoint) > 2000 || !AllowedEndpoint(u) {
		httpx.Problem(rw, http.StatusUnprocessableEntity, "That is not a browser push service Axon sends to.")
		return
	}
	if len(b.P256dh) < 80 || len(b.P256dh) > 100 || len(b.Auth) < 16 || len(b.Auth) > 30 || len([]rune(b.Device)) > 100 {
		httpx.Problem(rw, http.StatusUnprocessableEntity, "The subscription's keys are not valid.")
		return
	}
	var id uuid.UUID
	// A browser keeps its endpoint: subscribing again moves it to this account.
	err = w.db.QueryRow(r.Context(), `INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, device) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = $2, p256dh = $4, auth = $5, device = $6 RETURNING id`,
		uuid.Must(uuid.NewV7()), who.UserID, b.Endpoint, b.P256dh, b.Auth, b.Device).Scan(&id)
	if err != nil {
		w.fail(rw, err)
		return
	}
	httpx.JSON(rw, http.StatusCreated, map[string]any{"id": id})
}

func (w *Worker) unsubscribe(rw http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Problem(rw, http.StatusNotFound, "No such device.")
		return
	}
	tag, err := w.db.Exec(r.Context(), `DELETE FROM push_subscriptions WHERE id = $1 AND user_id = $2`, id, who.UserID)
	if err != nil {
		w.fail(rw, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Problem(rw, http.StatusNotFound, "No such device.")
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

// test sends today's digest now (or, with nothing to say, a short hello),
// so the person sees what reminders look like.
func (w *Worker) test(rw http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	doc, err := settings.Get(r.Context(), w.db, who.UserID)
	if err != nil {
		w.fail(rw, err)
		return
	}
	msg, err := Morning(r.Context(), w.db, who.UserID, w.now())
	if err != nil {
		w.fail(rw, err)
		return
	}
	if msg.Title == "" {
		msg = Message{Title: "Reminders are on", Body: "Nothing is due right now. Axon will tell you when something is.", URL: "/today", Tag: "test"}
	}
	got, err := w.Deliver(r.Context(), who.UserID, doc.Reminders.Email, msg)
	if errors.Is(err, nil) && got.Devices == 0 && !got.Email {
		httpx.Problem(rw, http.StatusUnprocessableEntity, "No device or email received it: turn notifications on for this device, or add an email address.")
		return
	}
	if err != nil {
		w.fail(rw, err)
		return
	}
	httpx.JSON(rw, http.StatusOK, map[string]any{"message": msg, "devices": got.Devices, "email": got.Email})
}
