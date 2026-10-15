package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/notify"
)

// chatPost is one request a chat service received.
type chatPost struct {
	host, path, auth, signature string
	body                        string // JSON, or payload_json / text fields of a form
	file                        bool   // a picture was attached
}

// fakeChats plays Slack, Discord, Teams, Telegram, ntfy, and a webhook
// receiver: every request from notify goes to one test server, which
// records it under the host it was meant for.
type fakeChats struct {
	mu      sync.Mutex
	posts   []chatPost
	fail    map[string]bool          // hosts that answer 500
	answers map[string]func() string // answers by path suffix, for listeners
}

func (f *fakeChats) RoundTrip(req *http.Request) (*http.Response, error) {
	p := chatPost{host: req.URL.Host, path: req.URL.Path, auth: req.Header.Get("Authorization"), signature: req.Header.Get("X-Axon-Signature")}
	ct, params, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if ct == "multipart/form-data" {
		r := multipart.NewReader(req.Body, params["boundary"])
		for {
			part, err := r.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(part)
			if part.FileName() != "" {
				p.file = len(b) > 1000 && string(b[1:4]) == "PNG"
			} else {
				p.body += part.FormName() + "=" + string(b) + "\n"
			}
		}
	} else if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		p.body = string(b)
	}
	f.mu.Lock()
	f.posts = append(f.posts, p)
	failing := f.fail[req.URL.Host]
	var answer func() string
	for suffix, a := range f.answers {
		if strings.HasSuffix(req.URL.Path, suffix) {
			answer = a
		}
	}
	f.mu.Unlock()
	rec := httptest.NewRecorder()
	switch {
	case answer != nil:
		_, _ = rec.WriteString(answer())
	case failing:
		rec.WriteHeader(500)
		_, _ = rec.WriteString("down")
	case req.URL.Host == "slack.com":
		_, _ = rec.WriteString(`{"ok":true}`)
	case req.URL.Host == "api.telegram.org":
		_, _ = rec.WriteString(`{"ok":true,"result":{}}`)
	case strings.HasSuffix(req.URL.Path, "/users/@me/channels"):
		_, _ = rec.WriteString(`{"id":"555"}`)
	default:
		_, _ = rec.WriteString(`{}`)
	}
	res := rec.Result()
	res.Request = req
	return res, nil
}

func (f *fakeChats) to(host string) []chatPost {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []chatPost
	for _, p := range f.posts {
		if p.host == host {
			out = append(out, p)
		}
	}
	return out
}

func (f *fakeChats) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.posts)
}

func newFakeChats(t *testing.T) *fakeChats {
	f := &fakeChats{fail: map[string]bool{}, answers: map[string]func() string{}}
	client, free := notify.Client, notify.FreeClient
	notify.Client, notify.FreeClient = &http.Client{Transport: f}, &http.Client{Transport: f}
	t.Cleanup(func() { notify.Client, notify.FreeClient = client, free })
	return f
}

type channelJSON struct {
	ID        string `json:"id"`
	Mode      string `json:"mode"`
	Target    string `json:"target"`
	Linked    bool   `json:"linked"`
	LastError string `json:"last_error"`
}

func (c *client) addChannel(body string) (channelJSON, string) {
	c.e.t.Helper()
	var res struct {
		Channel  channelJSON `json:"channel"`
		LinkCode string      `json:"link_code"`
	}
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/me/channels", []byte(body)), &res)
	return res.Channel, res.LinkCode
}

// Digests reach every channel a person connects, rich where the channel
// allows it, with a picture, on their days and times.
func TestDigestsOnEveryChannel(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Ada Lovelace")
	bob.register("Bob")
	chats := newFakeChats(t)
	today := time.Now().UTC().Format(time.DateOnly)
	alice.createTask(`{"title":"Renew the passport","status":"next","due_on":"` + today + `"}`)
	alice.createTask(`{"title":"Plan the trip","status":"next","due_on":"` + time.Now().UTC().AddDate(0, 0, 3).Format(time.DateOnly) + `"}`)
	alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page","two_minute":"Open the book"}`))

	// The digest as it would go out: numbered, with a quote and a picture.
	var d struct {
		Greeting string `json:"greeting"`
		Sections []struct {
			Key   string `json:"key"`
			Items []struct {
				N    int    `json:"n"`
				Text string `json:"text"`
			} `json:"items"`
		} `json:"sections"`
		Quote struct {
			Text, Author string
		} `json:"quote"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/digest", nil), &d)
	if !strings.HasSuffix(d.Greeting, ", Ada") || len(d.Sections) < 3 || d.Sections[0].Items[0].N != 1 || d.Sections[0].Items[0].Text != "Renew the passport" || d.Quote.Author == "" {
		t.Fatalf("daily digest = %+v", d)
	}
	alice.mustDo(200, "GET", "/v1/me/digest?kind=weekly", nil)
	alice.mustDo(200, "GET", "/v1/me/digest?kind=monthly", nil)
	alice.mustDo(400, "GET", "/v1/me/digest?kind=hourly", nil)
	if png := alice.mustDo(200, "GET", "/v1/me/digest/picture?kind=weekly", nil); string(png[1:4]) != "PNG" {
		t.Fatal("the picture is not a PNG")
	}

	// Channels: each kind checks its address; secrets never come back.
	for _, bad := range []string{
		`{"kind":"slack","name":"S","config":{"webhook_url":"https://evil.example/services/x"}}`,
		`{"kind":"slack","name":"S","config":{"bot_token":"xoxb-1"}}`,
		`{"kind":"discord","name":"D","config":{"webhook_url":"https://discord.com.evil.example/api/webhooks/1/x"}}`,
		`{"kind":"teams","name":"T","config":{"webhook_url":"https://example.com/hook"}}`,
		`{"kind":"telegram","name":"T","config":{"bot_token":"nope"}}`,
		`{"kind":"ntfy","name":"N","config":{"topic":"a b"}}`,
		`{"kind":"webhook","name":"W","config":{"webhook_url":"ftp://example.com"}}`,
		`{"kind":"sms","name":"X","config":{}}`,
		`{"kind":"ntfy","name":"","config":{"topic":"ok"}}`,
	} {
		alice.mustDo(422, "POST", "/v1/me/channels", []byte(bad))
	}
	slackHook := "https://hooks.slack.com/services/T000/B000/XXXXXXXXXXXXXXXX"
	slack, _ := alice.addChannel(`{"kind":"slack","name":"Slack","config":{"webhook_url":"` + slackHook + `"}}`)
	alice.addChannel(`{"kind":"discord","name":"Discord","config":{"webhook_url":"https://discord.com/api/webhooks/123456789/abcdefghijklmnopqrstuvwxyz"}}`)
	alice.addChannel(`{"kind":"teams","name":"Teams","config":{"webhook_url":"https://contoso.webhook.office.com/webhookb2/abc"}}`)
	alice.addChannel(`{"kind":"ntfy","name":"Phone","config":{"topic":"ada-axon","token":"tk_secret"},"digests":["daily"],"reminders":false}`)
	hook, _ := alice.addChannel(`{"kind":"webhook","name":"Home","config":{"webhook_url":"https://home.example/axon","secret":"s3cret"},"digests":["weekly"]}`)
	tg, code := alice.addChannel(`{"kind":"telegram","name":"Telegram","config":{"bot_token":"123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}`)
	if slack.Mode != "webhook" || slack.Target != "hooks.slack.com/…" || tg.Mode != "bot" || tg.Linked || len(code) != 9 {
		t.Fatalf("channels: %+v %+v %q", slack, tg, code)
	}
	list := alice.mustDo(200, "GET", "/v1/me/channels", nil)
	for _, secret := range []string{"XXXXXXXXXXXXXXXX", "abcdefghijklmnop", "tk_secret", "s3cret", "AAAAAAAAAAAA", "webhookb2"} {
		if strings.Contains(string(list), secret) {
			t.Fatalf("a secret came back: %s", secret)
		}
	}
	bob.mustDo(404, "PUT", "/v1/me/channels/"+slack.ID, []byte(`{"name":"Mine"}`))
	bob.mustDo(404, "DELETE", "/v1/me/channels/"+slack.ID, nil)
	bob.mustDo(404, "POST", "/v1/me/channels/"+slack.ID+"/test", nil)

	// A test to one channel; a bot not linked yet cannot be tested.
	alice.mustDo(204, "POST", "/v1/me/channels/"+slack.ID+"/test", nil)
	got := chats.to("hooks.slack.com")
	if len(got) != 1 || got[0].path != "/services/T000/B000/XXXXXXXXXXXXXXXX" || !strings.Contains(got[0].body, `"blocks"`) || !strings.Contains(got[0].body, "Renew the passport") {
		t.Fatalf("slack = %+v", got)
	}
	alice.mustDo(422, "POST", "/v1/me/channels/"+tg.ID+"/test", nil)
	alice.mustDo(422, "POST", "/v1/me/channels/"+slack.ID+"/link", nil) // a webhook needs no link
	var relink struct {
		Code string `json:"link_code"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/me/channels/"+tg.ID+"/link", nil), &relink)
	if relink.Code == "" || relink.Code == code {
		t.Fatalf("relink = %+v", relink)
	}
	// Linking happens when the person sends the code to the bot (EP-108); here, directly.
	if _, err := pool.Exec(context.Background(), `UPDATE notify_channels SET peer = '42', peer_name = 'Ada' WHERE id = $1`, tg.ID); err != nil {
		t.Fatal(err)
	}
	// Changing the name keeps the secret.
	alice.mustDo(200, "PUT", "/v1/me/channels/"+slack.ID, []byte(`{"name":"Slack, renamed"}`))

	// The morning digest goes out once, to every channel that takes it.
	w, err := Reminders(auth.Config{BaseURL: origin, EncryptionKey: strings.Repeat("ab", 32)}, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	now := time.Now().UTC()
	weekday := strings.ToLower(now.Weekday().String())
	doc["reminders"] = map[string]any{"enabled": true, "morning": "08:00", "evening": "20:00", "email": "",
		"weekly_day": weekday, "weekly_at": "09:00", "monthly_day": min(now.Day(), 28), "monthly_at": "10:00"}
	body, _ := json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)
	chats.fail["home.example"] = true
	at := func(hhmm string) time.Time {
		tm, _ := time.Parse("2006-01-02 15:04", today+" "+hhmm)
		return tm
	}
	before := chats.count()
	for range 2 {
		if err := w.Step(context.Background(), at("08:30")); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(chats.to("hooks.slack.com")); n != 2 {
		t.Fatalf("slack received %d messages, want the test and one digest", n)
	}
	disc := chats.to("discord.com")
	if len(disc) != 1 || !disc[0].file || !strings.Contains(disc[0].body, "attachment://scene.png") || !strings.Contains(disc[0].body, `"parse":[]`) {
		t.Fatalf("discord = %+v", disc)
	}
	if teams := chats.to("contoso.webhook.office.com"); len(teams) != 1 || !strings.Contains(teams[0].body, "AdaptiveCard") {
		t.Fatalf("teams = %+v", teams)
	}
	if ntfy := chats.to("ntfy.sh"); len(ntfy) != 1 || ntfy[0].auth != "Bearer tk_secret" || !strings.Contains(ntfy[0].body, `"topic":"ada-axon"`) {
		t.Fatalf("ntfy = %+v", ntfy)
	}
	tgPosts := chats.to("api.telegram.org")
	if len(tgPosts) != 2 || !strings.HasSuffix(tgPosts[0].path, "/sendPhoto") || !tgPosts[0].file || !strings.Contains(tgPosts[1].body, `"chat_id":"42"`) {
		t.Fatalf("telegram = %+v", tgPosts)
	}
	if len(chats.to("home.example")) != 0 {
		t.Fatal("the webhook takes weekly digests only")
	}
	morning := chats.count() - before

	// The weekly and monthly digests on their day; a failing channel is
	// recorded and does not stop the others.
	if err := w.Step(context.Background(), at("10:30")); err != nil {
		t.Fatal(err)
	}
	hooks := chats.to("home.example")
	if len(hooks) != 1 {
		t.Fatalf("webhook = %+v", hooks)
	}
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write([]byte(hooks[0].body))
	if hooks[0].signature != "sha256="+hex.EncodeToString(mac.Sum(nil)) || !strings.Contains(hooks[0].body, `"kind":"weekly"`) {
		t.Fatalf("webhook signature %q, body %.200s", hooks[0].signature, hooks[0].body)
	}
	var chans struct {
		Items []channelJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/channels", nil), &chans)
	for _, c := range chans.Items {
		if c.ID == hook.ID && !strings.Contains(c.LastError, "500") {
			t.Fatalf("the failing webhook's error = %q", c.LastError)
		}
	}
	monthly := 0
	for _, p := range chats.to("hooks.slack.com") {
		if strings.Contains(p.body, "Your month") {
			monthly++
		}
	}
	if now.Day() <= 28 && monthly != 1 {
		t.Fatalf("monthly digests to slack = %d", monthly)
	}

	// The evening nudge and task reminders go to channels that take reminders.
	before = chats.count()
	if err := w.Step(context.Background(), at("20:05")); err != nil {
		t.Fatal(err)
	}
	if len(chats.to("ntfy.sh")) != 1 {
		t.Fatal("ntfy takes no reminders, but got the nudge")
	}
	if chats.count()-before < 3 || morning < 5 {
		t.Fatalf("evening posts %d, morning posts %d", chats.count()-before, morning)
	}
	// The test button with nothing else to receive it still reaches channels.
	alice.mustDo(200, "POST", "/v1/me/reminders/test?kind=monthly", nil)
	alice.mustDo(400, "POST", "/v1/me/reminders/test?kind=yearly", nil)

	alice.mustDo(204, "DELETE", "/v1/me/channels/"+slack.ID, nil)
	alice.mustDo(404, "DELETE", "/v1/me/channels/"+slack.ID, nil)
}
