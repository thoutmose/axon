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
	"github.com/thoutmose/axon/api/internal/digest"
	"github.com/thoutmose/axon/api/internal/notify"
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
	db       *pgxpool.Pool
	log      *slog.Logger
	sealer   *auth.Sealer
	mail     *Mailer
	sub      string // VAPID subject: how push services reach the operator
	base     string // AXON_BASE_URL, for links in emails and chat messages
	channels *notify.Store
	now      func() time.Time
	vapid    *push.VAPID
}

// NewWorker returns a worker; call Run in a goroutine, or Step in tests.
func NewWorker(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer, mail *Mailer, contact, base string) *Worker {
	return &Worker{db: pool, log: log, sealer: sealer, mail: mail, sub: contact, base: strings.TrimRight(base, "/"),
		channels: notify.NewStore(pool, sealer), now: time.Now}
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

// slot is one scheduled message of a person's day.
type slot struct {
	kind, at string
}

// slots lists what may go out on a local day: the morning digest and the
// evening nudge every day, the weekly digest on its day, the monthly one
// on its day of the month (EP-70).
func slots(r settings.Reminders, local time.Time) []slot {
	out := []slot{{"morning", r.Morning}, {"evening", r.Evening}}
	if strings.EqualFold(local.Weekday().String(), r.WeeklyDay) {
		out = append(out, slot{"weekly", r.WeeklyAt})
	}
	if local.Day() == r.MonthlyDay {
		out = append(out, slot{"monthly", r.MonthlyAt})
	}
	return out
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
	for _, r := range slots(doc.Reminders, local) {
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
		switch r.kind {
		case "evening":
			msg, err := Evening(ctx, w.db, userID, now)
			if err != nil {
				return err
			}
			if msg.Title == "" {
				continue // nothing to say
			}
			if _, err := w.Deliver(ctx, userID, doc.Reminders.Email, msg); err != nil {
				return err
			}
		default:
			kind := r.kind
			if kind == "morning" {
				kind = digest.Daily
			}
			if _, err := w.DeliverDigest(ctx, userID, doc, kind, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// Delivered says where a message went.
type Delivered struct {
	Devices  int  `json:"devices"`
	Email    bool `json:"email"`
	Channels int  `json:"channels"`
}

// Deliver pushes a message to every device of the person, sends it to the
// channels that take reminders, and emails it to
// address when set and the instance can send mail. Subscriptions the
// browser dropped are forgotten.
func (w *Worker) Deliver(ctx context.Context, userID uuid.UUID, address string, msg Message) (Delivered, error) {
	var out Delivered
	var err error
	if out.Devices, err = w.deliverPush(ctx, userID, msg); err != nil {
		return out, err
	}
	if address != "" && w.mail != nil {
		if err := w.mail.Send(address, msg); err != nil {
			w.log.Warn("reminder email failed", "user", userID, "err", err)
		} else {
			out.Email = true
		}
	}
	targets, err := w.channels.Targets(ctx, userID, "")
	if err != nil {
		return out, err
	}
	for _, t := range targets {
		err := notify.SendText(ctx, t, notify.Message{Title: msg.Title, Body: msg.Body, URL: msg.URL}, w.base)
		w.channels.Sent(ctx, t.ID, err, w.now())
		if err != nil {
			w.log.Warn("reminder to a channel failed", "user", userID, "channel", t.ID, "err", err)
			continue
		}
		out.Channels++
	}
	return out, nil
}

// DeliverDigest sends a digest of kind: a short notification to the
// person's devices, the full digest by email with its picture, and to every
// channel that takes this kind (EP-70). With force, it goes out even with
// nothing to say (the test button).
func (w *Worker) DeliverDigest(ctx context.Context, userID uuid.UUID, doc settings.Doc, kind string, force bool) (Delivered, error) {
	var out Delivered
	d, err := digest.Build(ctx, w.db, userID, kind, doc.Language, w.now())
	if err != nil {
		return out, err
	}
	if d.Empty() && !force {
		return out, nil
	}
	if items := d.Actionable(); len(items) > 0 {
		if err := digest.Remember(ctx, w.db, userID, items); err != nil {
			return out, err
		}
	}
	msg := Message{Tag: kind, URL: d.URL}
	msg.Title, msg.Body = digest.Short(d)
	if kind == digest.Daily {
		// The notification keeps the morning digest's compact wording.
		if m, err := Morning(ctx, w.db, userID, w.now()); err != nil {
			return out, err
		} else if m.Title != "" {
			msg = m
		} else if !force {
			msg = Message{}
		}
	}
	if msg.Title != "" {
		if out.Devices, err = w.deliverPush(ctx, userID, msg); err != nil {
			return out, err
		}
	}
	var scene []byte
	sceneFor := func() []byte {
		if scene == nil {
			scene = digest.Scene(d)
		}
		return scene
	}
	if doc.Reminders.Email != "" && w.mail != nil {
		title, _ := digest.Short(d)
		err := w.mail.SendRich(doc.Reminders.Email, Rich{Subject: title, Text: digest.Text(d, w.base), HTML: digest.HTML(d, w.base, "scene"), PNG: sceneFor()})
		if err != nil {
			w.log.Warn("digest email failed", "user", userID, "err", err)
		} else {
			out.Email = true
		}
	}
	targets, err := w.channels.Targets(ctx, userID, d.Kind)
	if err != nil {
		return out, err
	}
	for _, t := range targets {
		err := notify.SendDigest(ctx, t, d, w.base, sceneFor())
		w.channels.Sent(ctx, t.ID, err, w.now())
		if err != nil {
			w.log.Warn("digest to a channel failed", "user", userID, "channel", t.ID, "err", err)
			continue
		}
		out.Channels++
	}
	return out, nil
}

// deliverPush sends a notification to every device of the person and
// returns how many took it; subscriptions the browser dropped are forgotten.
func (w *Worker) deliverPush(ctx context.Context, userID uuid.UUID, msg Message) (int, error) {
	devices := 0
	rows, err := w.db.Query(ctx, `SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return devices, err
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
		return devices, err
	}
	if len(subs) > 0 {
		v, err := w.key(ctx)
		if err != nil {
			return devices, err
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
				devices++
				_, _ = w.db.Exec(ctx, `UPDATE push_subscriptions SET last_sent_at = now() WHERE id = $1`, s.id)
			}
		}
	}
	return devices, nil
}
