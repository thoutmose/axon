package reminders

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/push"
	"github.com/thoutmose/axon/api/internal/settings"
)

// Client sends to push services; tests replace it.
var Client = &http.Client{Timeout: 15 * time.Second}

// pushHosts are the browsers' push services. Endpoints elsewhere are
// refused, so a subscription cannot point Axon at another server.
var pushHosts = []string{"fcm.googleapis.com", "push.services.mozilla.com", "web.push.apple.com", "notify.windows.com"}

// AllowedEndpoint reports whether Axon may send to an endpoint; tests replace it.
var AllowedEndpoint = func(u *url.URL) bool {
	if u.Scheme != "https" || u.Port() != "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return slices.ContainsFunc(pushHosts, func(p string) bool { return h == p || strings.HasSuffix(h, "."+p) })
}

// window is how late a reminder may still go out (after a restart, say).
const window = 3 * time.Hour

// Worker sends reminders when their time comes.
type Worker struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
	mail   *Mailer
	sub    string // VAPID subject: how push services reach the operator
	now    func() time.Time
	vapid  *push.VAPID
}

// NewWorker returns a worker; call Run in a goroutine, or Step in tests.
func NewWorker(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer, mail *Mailer, contact string) *Worker {
	return &Worker{db: pool, log: log, sealer: sealer, mail: mail, sub: contact, now: time.Now}
}

// Run checks every minute until ctx ends.
func (w *Worker) Run(ctx context.Context) {
	for {
		if err := w.Step(ctx, w.now()); err != nil && ctx.Err() == nil {
			w.log.Error("reminders step failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

func (w *Worker) key(ctx context.Context) (push.VAPID, error) {
	if w.vapid == nil {
		k, err := VAPIDKey(ctx, w.db, w.sealer)
		if err != nil {
			return push.VAPID{}, err
		}
		w.vapid = &push.VAPID{Key: k, Subject: subject(w.sub)}
	}
	return *w.vapid, nil
}

// subject turns the operator contact into what RFC 8292 asks for.
func subject(contact string) string {
	switch {
	case strings.HasPrefix(contact, "https://") || strings.HasPrefix(contact, "mailto:"):
		return contact
	case strings.Contains(contact, "@"):
		return "mailto:" + contact
	}
	return "https://github.com/thoutmose/axon"
}

// Step sends what is due at now.
func (w *Worker) Step(ctx context.Context, now time.Time) error {
	rows, err := w.db.Query(ctx, `SELECT s.user_id, u.timezone FROM user_settings s JOIN users u ON u.id = s.user_id
		WHERE coalesce((s.doc->'reminders'->>'enabled')::boolean, false)`)
	if err != nil {
		return err
	}
	type who struct {
		id uuid.UUID
		tz string
	}
	people, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (who, error) {
		var p who
		err := r.Scan(&p.id, &p.tz)
		return p, err
	})
	if err != nil {
		return err
	}
	for _, p := range people {
		if err := w.person(ctx, p.id, p.tz, now); err != nil && ctx.Err() == nil {
			w.log.Error("reminders for one person failed", "user", p.id, "err", err)
		}
	}
	if err := w.tasks(ctx, now); err != nil && ctx.Err() == nil {
		w.log.Error("task reminders failed", "err", err)
	}
	return ctx.Err()
}

// tasks sends the reminders set on tasks (EP-68), once each, whether or not
// the daily reminders are on: the person asked for this one. One that is
// more than 12 hours late (the server was down) is marked, not sent.
func (w *Worker) tasks(ctx context.Context, now time.Time) error {
	rows, err := w.db.Query(ctx, `UPDATE tasks t SET reminded_at = $1 FROM entities e
		WHERE e.id = t.id AND t.remind_at <= $1 AND t.reminded_at IS NULL
		RETURNING t.id, e.user_id, e.title, t.remind_at > $2 AND t.status NOT IN ('done', 'cancelled')`, now, now.Add(-12*time.Hour))
	if err != nil {
		return err
	}
	type due struct {
		id, user uuid.UUID
		title    string
		send     bool
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (due, error) {
		var d due
		err := r.Scan(&d.id, &d.user, &d.title, &d.send)
		return d, err
	})
	if err != nil {
		return err
	}
	for _, d := range list {
		if !d.send {
			continue
		}
		doc, err := settings.Get(ctx, w.db, d.user)
		if err != nil {
			return err
		}
		msg := Message{Title: "Reminder", Body: d.title, URL: "/tasks?open=" + d.id.String(), Tag: "task-" + d.id.String()}
		if _, err := w.Deliver(ctx, d.user, doc.Reminders.Email, msg); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) person(ctx context.Context, userID uuid.UUID, tz string, now time.Time) error {
	doc, err := settings.Get(ctx, w.db, userID)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	day := local.Format(time.DateOnly)
	for _, r := range []struct {
		kind, at string
		build    func(context.Context, db.Querier, uuid.UUID, time.Time) (Message, error)
	}{{"morning", doc.Reminders.Morning, Morning}, {"evening", doc.Reminders.Evening, Evening}} {
		at, err := time.ParseInLocation("2006-01-02 15:04", day+" "+r.at, loc)
		if r.at == "" || err != nil || local.Before(at) || local.Sub(at) >= window {
			continue
		}
		tag, err := w.db.Exec(ctx, `INSERT INTO reminders_sent (user_id, kind, day) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, userID, r.kind, day)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			continue // already sent today
		}
		msg, err := r.build(ctx, w.db, userID, now)
		if err != nil {
			return err
		}
		if msg.Title == "" {
			continue // nothing to say
		}
		if _, err := w.Deliver(ctx, userID, doc.Reminders.Email, msg); err != nil {
			return err
		}
	}
	return nil
}

// Delivered says where a message went.
type Delivered struct {
	Devices int  `json:"devices"`
	Email   bool `json:"email"`
}

// Deliver pushes a message to every device of the person and emails it to
// address when set and the instance can send mail. Subscriptions the
// browser dropped are forgotten.
func (w *Worker) Deliver(ctx context.Context, userID uuid.UUID, address string, msg Message) (Delivered, error) {
	var out Delivered
	rows, err := w.db.Query(ctx, `SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return out, err
	}
	type sub struct {
		id uuid.UUID
		push.Subscription
	}
	subs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (sub, error) {
		var s sub
		err := r.Scan(&s.id, &s.Endpoint, &s.P256dh, &s.Auth)
		return s, err
	})
	if err != nil {
		return out, err
	}
	if len(subs) > 0 {
		v, err := w.key(ctx)
		if err != nil {
			return out, err
		}
		payload, _ := json.Marshal(msg)
		for _, s := range subs {
			u, err := url.Parse(s.Endpoint)
			if err != nil || !AllowedEndpoint(u) {
				continue
			}
			err = push.Send(ctx, Client, v, s.Subscription, payload, 12*time.Hour)
			switch {
			case errors.Is(err, push.ErrGone):
				_, _ = w.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1`, s.id)
			case err != nil:
				w.log.Warn("push failed", "user", userID, "err", err)
			default:
				out.Devices++
				_, _ = w.db.Exec(ctx, `UPDATE push_subscriptions SET last_sent_at = now() WHERE id = $1`, s.id)
			}
		}
	}
	if address != "" && w.mail != nil {
		if err := w.mail.Send(address, msg); err != nil {
			w.log.Warn("reminder email failed", "user", userID, "err", err)
		} else {
			out.Email = true
		}
	}
	return out, nil
}
