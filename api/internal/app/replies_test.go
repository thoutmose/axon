package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/notify"
	"github.com/thoutmose/axon/api/internal/replies"
	"github.com/thoutmose/axon/api/internal/wsclient"
)

func (c *client) task(id string) taskJSON {
	c.e.t.Helper()
	var t taskJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/tasks/"+id, nil), &t)
	return t
}

// Answers to the digest in a chat app do what they say, for the linked
// person only, and can be undone.
func TestAnsweringTheDigest(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Ada")
	newFakeChats(t)
	today := time.Now().UTC().Format(time.DateOnly)
	passport := alice.createTask(`{"title":"Renew the passport","status":"next","due_on":"` + today + `"}`)
	trip := alice.createTask(`{"title":"Plan the summer trip","status":"next","scheduled_on":"` + today + `"}`)
	water := alice.createTask(`{"title":"Water the plants","status":"next","scheduled_on":"` + today + `","recurrence":"FREQ=DAILY"}`)
	alice.createTask(`{"title":"Call mum","status":"next"}`)
	alice.createTask(`{"title":"Call dad","status":"next"}`)
	alice.mustDo(201, "POST", "/v1/habits", []byte(`{"title":"Read one page"}`))

	ch, code := alice.addChannel(`{"kind":"telegram","name":"Telegram","config":{"bot_token":"123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}`)
	id := uuid.MustParse(ch.ID)
	h := replies.Handler{DB: pool, Base: origin, Now: time.Now}
	say := func(from, text string) string {
		t.Helper()
		out, err := h.Handle(context.Background(), replies.Inbound{Channels: []uuid.UUID{id}, Kind: "telegram", From: from, FromName: "Ada", Text: text})
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		return out
	}
	// Nobody is obeyed before linking; a wrong code links nothing.
	if out := say("42", "done 1"); !strings.Contains(out, "answers one person") {
		t.Fatalf("before linking: %s", out)
	}
	if out := say("42", "/start AAAA-AAAA"); !strings.Contains(out, "wrong or has expired") {
		t.Fatalf("wrong code: %s", out)
	}
	if out := say("42", "/start "+strings.ToLower(code)); !strings.Contains(out, "Linked") {
		t.Fatalf("link: %s", out)
	}
	if out := say("99", "list"); !strings.Contains(out, "answers one person") {
		t.Fatalf("a stranger was answered: %s", out)
	}
	if again := say("99", "/start "+code); !strings.Contains(again, "wrong or has expired") {
		t.Fatalf("a used code linked again: %s", again)
	}

	// "list" numbers today's items; "done 1" checks off the first one.
	list := say("42", "list")
	if !strings.Contains(list, "1. Renew the passport") || !strings.Contains(list, "Read one page") {
		t.Fatalf("list:\n%s", list)
	}
	if out := say("42", "done 1"); !strings.Contains(out, "✅ Renew the passport") {
		t.Fatalf("done 1: %s", out)
	}
	if alice.task(passport.ID).Status != "done" {
		t.Fatal("the task is not done")
	}
	if out := say("42", "done 1"); !strings.Contains(out, "already done") {
		t.Fatalf("done twice: %s", out)
	}
	// Undo puts it back as it was.
	if out := say("42", "undo"); !strings.Contains(out, "↩️ Renew the passport") || alice.task(passport.ID).Status != "next" {
		t.Fatalf("undo: %s, status %s", out, alice.task(passport.ID).Status)
	}
	if out := say("42", "undo"); !strings.Contains(out, "Nothing to undo") {
		t.Fatalf("undo twice: %s", out)
	}
	// A number of a habit checks it in; an unknown number says so.
	habitN := ""
	for _, line := range strings.Split(list, "\n") {
		if strings.Contains(line, "Read one page") {
			habitN = strings.SplitN(line, ".", 2)[0]
		}
	}
	if out := say("42", "fait "+habitN+" 77"); !strings.Contains(out, "🌱 Read one page") || !strings.Contains(out, "No item 77") {
		t.Fatalf("habit: %s", out)
	}
	// A sentence finds the task; a close call asks which.
	if out := say("42", "I finished planning the trip"); !strings.Contains(out, "✅ Plan the summer trip") || alice.task(trip.ID).Status != "done" {
		t.Fatalf("sentence: %s", out)
	}
	ask := say("42", "done call")
	if !strings.Contains(ask, "Which one?") || !strings.Contains(ask, "Call mum") || !strings.Contains(ask, "Call dad") {
		t.Fatalf("ambiguous: %s", ask)
	}
	pick := "1"
	if strings.Contains(ask, "2. Call mum") {
		pick = "2"
	}
	if out := say("42", "done "+pick); !strings.Contains(out, "✅ Call mum") {
		t.Fatalf("pick: %s", out)
	}
	if out := say("42", "done the rocket launch"); !strings.Contains(out, "found nothing") {
		t.Fatalf("no match: %s", out)
	}
	// A repeating task's next occurrence is planned.
	say("42", "list")
	if out := say("42", "done plants"); !strings.Contains(out, "next one is planned") || alice.task(water.ID).Status != "done" {
		t.Fatalf("recurring: %s", out)
	}

	// Add a task with a day and a time; move one; capture a note.
	if out := say("42", "add call the bank tomorrow at 3pm"); !strings.Contains(out, "➕ call the bank") || !strings.Contains(out, "15:00 🔔") {
		t.Fatalf("add: %s", out)
	}
	if out := say("42", "add buy 2 apples"); !strings.Contains(out, "➕ buy 2 apples — in your inbox") {
		t.Fatalf("add plain: %s", out)
	}
	say("42", "undo")
	var tasks struct {
		Items []taskJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/tasks?limit=100", nil), &tasks)
	bank := false
	for _, tk := range tasks.Items {
		if tk.Title == "buy 2 apples" {
			t.Fatal("undo kept the added task")
		}
		bank = bank || tk.Title == "call the bank"
	}
	if !bank {
		t.Fatal("the bank task was not added")
	}
	say("42", "list")
	if out := say("42", "snooze 1 next week"); !strings.Contains(out, "⏭️ Renew the passport") {
		t.Fatalf("snooze: %s", out)
	}
	if out := say("42", "note the meeting moved to Thursday"); !strings.Contains(out, "📝") {
		t.Fatalf("note: %s", out)
	}
	found := false
	for p := range alice.notes() {
		found = found || strings.HasPrefix(p, "0-Inbox/") && strings.HasSuffix(p, "the-meeting-moved-to-thursday.md")
	}
	if !found {
		t.Fatal("no inbox note")
	}
	if out := say("42", "thanks"); !strings.Contains(out, "help") {
		t.Fatalf("unknown: %s", out)
	}
	if out := say("42", "help"); !strings.Contains(out, "done 2") {
		t.Fatalf("help: %s", out)
	}

	// French answers for a French person.
	var doc map[string]any
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me/settings", nil), &doc)
	delete(doc, "ai")
	doc["language"] = "fr"
	body, _ := json.Marshal(doc)
	alice.mustDo(200, "PUT", "/v1/me/settings", body)
	if out := say("42", "aide"); !strings.Contains(out, "fait 2") {
		t.Fatalf("aide: %s", out)
	}
	// A disabled channel obeys nobody.
	alice.mustDo(200, "PUT", "/v1/me/channels/"+ch.ID, []byte(`{"name":"Telegram","enabled":false}`))
	if out := say("42", "list"); !strings.Contains(out, "answers one person") {
		t.Fatalf("disabled channel: %s", out)
	}
}

// The listeners only connect out: Telegram by long polling, Slack by Socket
// Mode, Discord by its gateway. Each reads a direct message and answers it.
func TestChatListeners(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Ada")
	chats := newFakeChats(t)
	alice.createTask(`{"title":"Renew the passport","status":"next","due_on":"` + time.Now().UTC().Format(time.DateOnly) + `"}`)

	// A TLS WebSocket server plays Slack's and Discord's sockets.
	var mu sync.Mutex
	got := map[string][]string{}
	srv := httptest.NewTLSServer(websocket.Handler(func(ws *websocket.Conn) {
		recv := func() string {
			var s string
			_ = websocket.Message.Receive(ws, &s)
			mu.Lock()
			got[ws.Request().URL.Path] = append(got[ws.Request().URL.Path], s)
			mu.Unlock()
			return s
		}
		switch ws.Request().URL.Path {
		case "/slack":
			_ = websocket.Message.Send(ws, `{"type":"hello"}`)
			_ = websocket.Message.Send(ws, `{"type":"events_api","envelope_id":"env-1","payload":{"event":{"type":"message","channel_type":"im","channel":"D1","user":"U42","text":"list"}}}`)
			recv() // the acknowledgement
		case "/discord/":
			_ = websocket.Message.Send(ws, `{"op":10,"d":{"heartbeat_interval":1000}}`)
			recv() // identify
			_ = websocket.Message.Send(ws, `{"op":0,"s":1,"t":"MESSAGE_CREATE","d":{"channel_id":"DM9","content":"list","author":{"id":"77","username":"ada"}}}`)
			_ = websocket.Message.Send(ws, `{"op":0,"s":2,"t":"MESSAGE_CREATE","d":{"channel_id":"G1","guild_id":"S1","content":"list","author":{"id":"77","username":"ada"}}}`)
			recv() // a heartbeat
		}
		time.Sleep(3 * time.Second)
	}))
	defer srv.Close()
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	old := wsclient.TLSConfig
	wsclient.TLSConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	t.Cleanup(func() { wsclient.TLSConfig = old })
	wss := "wss://" + srv.Listener.Addr().String()

	var polls int
	chats.mu.Lock()
	chats.answers["/getUpdates"] = func() string {
		polls++
		if polls == 1 {
			return `{"ok":true,"result":[{"update_id":5,"message":{"text":"list","chat":{"id":42,"type":"private"},"from":{"id":42,"first_name":"Ada"}}},
				{"update_id":6,"message":{"text":"list","chat":{"id":-100,"type":"group"},"from":{"id":43,"first_name":"Group"}}}]}`
		}
		time.Sleep(200 * time.Millisecond)
		return `{"ok":true,"result":[]}`
	}
	chats.answers["/apps.connections.open"] = func() string { return `{"ok":true,"url":"` + wss + `/slack"}` }
	chats.answers["/gateway/bot"] = func() string { return `{"url":"` + wss + `/discord"}` }
	chats.mu.Unlock()

	link := func(body, peer string) {
		ch, _ := alice.addChannel(body)
		if _, err := pool.Exec(context.Background(), `UPDATE notify_channels SET peer = $2 WHERE id = $1`, ch.ID, peer); err != nil {
			t.Fatal(err)
		}
	}
	link(`{"kind":"telegram","name":"Telegram","config":{"bot_token":"123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}`, "42")
	link(`{"kind":"slack","name":"Slack","config":{"bot_token":"xoxb-1-2-abc","app_token":"xapp-1-A1-abc"}}`, "U42")
	link(`{"kind":"discord","name":"Discord","config":{"bot_token":"`+strings.Repeat("d", 59)+`"}}`, "77")

	m := &replies.Manager{
		Handler: replies.Handler{DB: pool, Base: origin, Now: time.Now},
		Store:   notify.NewStore(pool, mustSealer(t)),
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Every:   time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { m.Run(ctx); close(stopped) }()
	defer func() { cancel(); <-stopped }()

	replied := func(host, path, want string) func() bool {
		return func() bool {
			for _, p := range chats.to(host) {
				if strings.HasSuffix(p.path, path) && strings.Contains(p.body, want) {
					return true
				}
			}
			return false
		}
	}
	eventually(t, "telegram answers in the private chat", replied("api.telegram.org", "/sendMessage", `"chat_id":"42"`))
	eventually(t, "slack answers in the DM", replied("slack.com", "/chat.postMessage", `"channel":"D1"`))
	eventually(t, "discord answers in the DM", replied("discord.com", "/channels/DM9/messages", "Renew the passport"))
	for _, p := range chats.to("api.telegram.org") {
		if strings.Contains(p.body, `"chat_id":"-100"`) {
			t.Fatal("a group message was answered")
		}
	}
	if len(chats.to("discord.com")) > 0 && replied("discord.com", "/channels/G1/messages", "")() {
		t.Fatal("a server message was answered")
	}
	eventually(t, "slack got its acknowledgement and discord its identify and heartbeat", func() bool {
		mu.Lock()
		defer mu.Unlock()
		d := got["/discord/"]
		return len(got["/slack"]) >= 1 && got["/slack"][0] == `{"envelope_id":"env-1"}` &&
			len(d) >= 2 && strings.Contains(d[0], `"op":2`) && strings.Contains(d[0], `"intents":4096`) && strings.Contains(d[1], `"op":1`)
	})
}

// mustSealer seals as the test server does.
func mustSealer(t *testing.T) *auth.Sealer {
	s, err := auth.NewSealer(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
