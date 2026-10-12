package app

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/reminders"
)

// fakePushService plays a browser vendor's push service and the browser
// behind it: it decrypts what it receives with the browser's keys.
type fakePushService struct {
	srv  *httptest.Server
	ua   *ecdh.PrivateKey
	auth []byte
	mu   sync.Mutex
	got  []receivedPush
}

type receivedPush struct {
	path, authorization string
	msg                 reminders.Message
}

func newFakePushService(t *testing.T) *fakePushService {
	t.Helper()
	ua, _ := ecdh.P256().GenerateKey(rand.Reader)
	f := &fakePushService{ua: ua, auth: make([]byte, 16)}
	_, _ = rand.Read(f.auth)
	f.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/gone") {
			w.WriteHeader(http.StatusGone)
			return
		}
		body, _ := io.ReadAll(r.Body)
		plain, err := f.decrypt(body)
		if err != nil || r.Header.Get("Content-Encoding") != "aes128gcm" {
			t.Errorf("push not decryptable: %v (%s)", err, r.Header.Get("Content-Encoding"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var m reminders.Message
		_ = json.Unmarshal(plain, &m)
		f.mu.Lock()
		f.got = append(f.got, receivedPush{r.URL.Path, r.Header.Get("Authorization"), m})
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(f.srv.Close)
	client, allowed := reminders.Client, reminders.AllowedEndpoint
	reminders.Client = f.srv.Client()
	reminders.AllowedEndpoint = func(u *url.URL) bool { return u.Scheme == "https" && u.Host == f.srv.Listener.Addr().String() }
	t.Cleanup(func() { reminders.Client, reminders.AllowedEndpoint = client, allowed })
	return f
}

// decrypt is the browser's side of RFC 8291.
func (f *fakePushService) decrypt(body []byte) ([]byte, error) {
	salt, keyLen := body[:16], int(body[20])
	asPublic := body[21 : 21+keyLen]
	as, err := ecdh.P256().NewPublicKey(asPublic)
	if err != nil {
		return nil, err
	}
	shared, err := f.ua.ECDH(as)
	if err != nil {
		return nil, err
	}
	uaPublic := f.ua.PublicKey().Bytes()
	prkKey, _ := hkdf.Extract(sha256.New, shared, f.auth)
	ikm, _ := hkdf.Expand(sha256.New, prkKey, "WebPush: info\x00"+string(uaPublic)+string(asPublic), 32)
	prk, _ := hkdf.Extract(sha256.New, ikm, salt)
	cek, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Expand(sha256.New, prk, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, body[21+keyLen:], nil)
	if err != nil {
		return nil, err
	}
	return []byte(strings.TrimRight(string(plain), "\x00"))[:len(strings.TrimRight(string(plain), "\x00"))-1], nil
}

func (f *fakePushService) received() []receivedPush {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]receivedPush{}, f.got...)
}

func (f *fakePushService) subscription(path string) []byte {
	b, _ := json.Marshal(map[string]string{
		"endpoint": f.srv.URL + path,
		"p256dh":   base64.RawURLEncoding.EncodeToString(f.ua.PublicKey().Bytes()),
		"auth":     base64.RawURLEncoding.EncodeToString(f.auth),
		"device":   "Test browser",
	})
	return b
}

// Reminders reach the person's devices, encrypted end to end, once a day at
// their local time, only with something to say; dropped devices are forgotten.
func TestReminders(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	push := newFakePushService(t)
	today := time.Now().UTC().Format(time.DateOnly)
	alice.createTask(`{"title":"Renew the passport","status":"next","due_on":"` + today + `"}`)
	alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page","two_minute":"Open the book"}`))

	// Only browser push services, and valid keys.
	alice.mustDo(422, "POST", "/v1/me/push-subscriptions", []byte(`{"endpoint":"https://169.254.169.254/latest","p256dh":"x","auth":"y"}`))
	var sub struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/me/push-subscriptions", push.subscription("/send/alice")), &sub)
	alice.mustDo(201, "POST", "/v1/me/push-subscriptions", push.subscription("/send/gone"))
	var status struct {
		PublicKey string `json:"public_key"`
		Devices   []struct {
			Device string `json:"device"`
		} `json:"devices"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/reminders", nil), &status)
	if len(status.Devices) != 2 || status.Devices[0].Device != "Test browser" || len(status.PublicKey) < 80 {
		t.Fatalf("status = %+v", status)
	}

	// Off until turned on; then the morning digest at 08:00 local, once.
	w, err := Reminders(auth.Config{BaseURL: origin, EncryptionKey: strings.Repeat("ab", 32)}, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	at := func(hhmm string) time.Time {
		tm, _ := time.Parse("2006-01-02 15:04", today+" "+hhmm)
		return tm
	}
	if err := w.Step(context.Background(), at("08:30")); err != nil {
		t.Fatal(err)
	}
	if len(push.received()) != 0 {
		t.Fatal("reminders sent while off")
	}
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc["reminders"] = map[string]any{"enabled": true, "morning": "08:00", "evening": "20:00", "email": ""}
	body, _ := json.Marshal(doc)
	if out := alice.mustDo(200, "PUT", "/v1/me/settings", body); !strings.Contains(string(out), `"reminders":{"enabled":true,"morning":"08:00"`) {
		t.Fatalf("settings do not show reminders: %s", out)
	}
	for _, hhmm := range []string{"07:59", "08:30", "08:45"} {
		if err := w.Step(context.Background(), at(hhmm)); err != nil {
			t.Fatal(err)
		}
	}
	got := push.received()
	if len(got) != 1 || got[0].path != "/send/alice" || got[0].msg.Title != "Today in Axon" ||
		!strings.Contains(got[0].msg.Body, "1 task") || !strings.Contains(got[0].msg.Body, "1 habit") {
		t.Fatalf("morning pushes = %+v", got)
	}
	if !strings.HasPrefix(got[0].authorization, "vapid t=") || !strings.HasSuffix(got[0].authorization, ", k="+status.PublicKey) {
		t.Fatalf("authorization = %s", got[0].authorization)
	}
	// The browser that dropped its subscription is forgotten.
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/reminders", nil), &status)
	if len(status.Devices) != 1 {
		t.Fatalf("gone device kept: %+v", status.Devices)
	}

	// The evening nudge names the habit still open, with its two-minute version.
	if err := w.Step(context.Background(), at("20:05")); err != nil {
		t.Fatal(err)
	}
	got = push.received()
	if len(got) != 2 || !strings.Contains(got[1].msg.Body, "Read one page (or just: Open the book)") || got[1].msg.URL != "/habits" {
		t.Fatalf("evening pushes = %+v", got)
	}

	// A test on demand; someone else cannot remove the device; removed, nothing receives.
	if out := alice.mustDo(200, "POST", "/v1/me/reminders/test", nil); !strings.Contains(string(out), `"devices":1`) {
		t.Fatalf("test = %s", out)
	}
	bob.mustDo(404, "DELETE", "/v1/me/push-subscriptions/"+sub.ID, nil)
	alice.mustDo(204, "DELETE", "/v1/me/push-subscriptions/"+sub.ID, nil)
	alice.mustDo(422, "POST", "/v1/me/reminders/test", nil)

	// From Friday to Sunday the digest suggests the weekly review, until it is done.
	sat := time.Now().UTC()
	for sat.Weekday() != time.Saturday {
		sat = sat.AddDate(0, 0, 1)
	}
	var me struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me", nil), &me)
	uid, _ := uuid.Parse(me.ID)
	msg, err := reminders.Morning(context.Background(), pool, uid, sat)
	if err != nil || !strings.Contains(msg.Body, "the weekly review") {
		t.Fatalf("saturday digest = %+v, %v", msg, err)
	}
	alice.mustDo(200, "PUT", "/v1/checkins/weekly/"+sat.AddDate(0, 0, -5).Format(time.DateOnly), []byte(`{"note":"Focus: the shed."}`))
	if msg, _ = reminders.Morning(context.Background(), pool, uid, sat); strings.Contains(msg.Body, "weekly review") {
		t.Fatalf("review suggested after it was done: %+v", msg)
	}

	// A task's own reminder goes out at its time, once, even with the daily
	// reminders off; a new time means a new reminder; a stale one is not sent.
	alice.mustDo(201, "POST", "/v1/me/push-subscriptions", push.subscription("/send/alice2"))
	doc["reminders"] = map[string]any{"enabled": false, "morning": "08:00", "evening": "20:00", "email": ""}
	body, _ = json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)
	at3 := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	call := alice.createTask(`{"title":"Call the bank","status":"next","remind_at":"` + at3 + `"}`)
	alice.createTask(`{"title":"Long ago","status":"next","remind_at":"` + time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339) + `"}`)
	before := len(push.received())
	for range 2 {
		if err := w.Step(context.Background(), time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	got = push.received()[before:]
	if len(got) != 1 || got[0].msg.Title != "Reminder" || got[0].msg.Body != "Call the bank" || !strings.Contains(got[0].msg.URL, call.ID) {
		t.Fatalf("task reminders = %+v", got)
	}
	in := time.Now().Add(-30 * time.Second).UTC().Format(time.RFC3339)
	if code, out := alice.do("PATCH", "/v1/tasks/"+call.ID, []byte(`{"remind_at":"`+in+`"}`), "If-Match", alice.etag("/v1/tasks/"+call.ID)); code != 200 {
		t.Fatalf("new reminder time = %d %s", code, out)
	}
	if err := w.Step(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(push.received()) != before+2 {
		t.Fatalf("a new time did not remind again: %d", len(push.received())-before)
	}

	// Settings check the times and the address.
	doc["reminders"] = map[string]any{"enabled": true, "morning": "8am", "evening": "", "email": ""}
	body, _ = json.Marshal(doc)
	alice.mustDo(422, "PUT", "/v1/me/settings", body)
}
