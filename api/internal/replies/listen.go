package replies

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/notify"
	"github.com/thoutmose/axon/api/internal/wsclient"
)

// Listeners only connect out: Telegram by long polling, Slack through
// Socket Mode, Discord through its gateway. Axon needs no public address
// to read answers, which suits an instance on a tailnet (ADR 0004, 0030).

// Manager keeps one listener per bot, starting and stopping them as people
// connect, change, and remove channels.
type Manager struct {
	Handler Handler
	Store   *notify.Store
	Log     *slog.Logger
	Every   time.Duration // how often to look for changed channels

	mu      sync.Mutex
	running map[string]*bot
	wg      sync.WaitGroup
}

// bot is one running listener and the channels that use it.
type bot struct {
	cancel   context.CancelFunc
	mu       sync.Mutex
	channels []uuid.UUID
	target   notify.Target // any of them: the kind and the tokens
}

func (b *bot) ids() []uuid.UUID {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]uuid.UUID{}, b.channels...)
}

func key(t notify.Target) string {
	h := sha256.Sum256([]byte(t.Kind + "\x00" + t.Config.BotToken + "\x00" + t.Config.AppToken))
	return hex.EncodeToString(h[:8])
}

// Run reconciles until ctx ends, then waits for the listeners to stop.
func (m *Manager) Run(ctx context.Context) {
	defer m.wg.Wait()
	if m.Every == 0 {
		m.Every = 30 * time.Second
	}
	m.running = map[string]*bot{}
	for {
		if err := m.Reconcile(ctx); err != nil && ctx.Err() == nil {
			m.Log.Error("chat listeners failed to refresh", "err", err)
		}
		select {
		case <-ctx.Done():
			m.mu.Lock()
			for _, b := range m.running {
				b.cancel()
			}
			m.mu.Unlock()
			return
		case <-time.After(m.Every):
		}
	}
}

// Reconcile starts listeners for new bots and stops those no longer used.
func (m *Manager) Reconcile(ctx context.Context) error {
	bots, err := m.Store.Bots(ctx)
	if err != nil {
		return err
	}
	want := map[string][]notify.Target{}
	for _, t := range bots {
		want[key(t)] = append(want[key(t)], t)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running == nil {
		m.running = map[string]*bot{}
	}
	for k, b := range m.running {
		if _, ok := want[k]; !ok {
			b.cancel()
			delete(m.running, k)
		}
	}
	for k, ts := range want {
		ids := make([]uuid.UUID, len(ts))
		for i, t := range ts {
			ids[i] = t.ID
		}
		if b, ok := m.running[k]; ok {
			b.mu.Lock()
			b.channels = ids
			b.mu.Unlock()
			continue
		}
		lctx, cancel := context.WithCancel(ctx)
		b := &bot{cancel: cancel, channels: ids, target: ts[0]}
		m.running[k] = b
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.listen(lctx, b)
		}()
	}
	return nil
}

// listen runs one bot's listener, reconnecting with a growing pause.
func (m *Manager) listen(ctx context.Context, b *bot) {
	wait := time.Second
	for ctx.Err() == nil {
		start := time.Now()
		var err error
		switch b.target.Kind {
		case "telegram":
			err = m.telegram(ctx, b)
		case "slack":
			err = m.slack(ctx, b)
		case "discord":
			err = m.discord(ctx, b)
		default:
			return
		}
		if ctx.Err() != nil {
			return
		}
		if time.Since(start) > time.Minute {
			wait = time.Second // it ran for a while: not a crash loop
		}
		m.Log.Warn("chat listener stopped; reconnecting", "kind", b.target.Kind, "err", err, "in", wait.String())
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, 2*time.Minute)
	}
}

// limiter keeps a loop or a flood from one sender from running away.
type limiter struct {
	mu   sync.Mutex
	seen map[string][]time.Time
}

var flood = limiter{seen: map[string][]time.Time{}}

func (l *limiter) allow(who string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.seen[who][:0]
	for _, t := range l.seen[who] {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(l.seen) > 10000 {
		l.seen = map[string][]time.Time{}
	}
	if len(recent) >= 20 {
		l.seen[who] = recent
		return false
	}
	l.seen[who] = append(recent, now)
	return true
}

// answer handles a message and replies in the same chat.
func (m *Manager) answer(ctx context.Context, b *bot, in Inbound, chat string) {
	if strings.TrimSpace(in.Text) == "" || !flood.allow(in.Kind+":"+in.From, time.Now()) {
		return
	}
	in.Channels = b.ids()
	reply, err := m.Handler.Handle(ctx, in)
	if err != nil {
		m.Log.Error("answering a chat message failed", "kind", in.Kind, "err", err)
		reply = "Something went wrong on Axon's side. Try again in a moment."
	}
	if reply == "" {
		return
	}
	if err := notify.Reply(ctx, b.target, chat, reply); err != nil {
		m.Log.Warn("chat reply failed", "kind", in.Kind, "err", err)
	}
}

// telegram long-polls getUpdates; each confirmed update is not sent again.
func (m *Manager) telegram(ctx context.Context, b *bot) error {
	offset := 0
	for ctx.Err() == nil {
		var updates []struct {
			UpdateID int `json:"update_id"`
			Message  *struct {
				Text string `json:"text"`
				Chat struct {
					ID   int64  `json:"id"`
					Type string `json:"type"`
				} `json:"chat"`
				From struct {
					ID        int64  `json:"id"`
					IsBot     bool   `json:"is_bot"`
					FirstName string `json:"first_name"`
					Username  string `json:"username"`
				} `json:"from"`
			} `json:"message"`
		}
		err := notify.TelegramCall(ctx, b.target.Config.BotToken, "getUpdates", map[string]any{"offset": offset, "timeout": 10, "allowed_updates": []string{"message"}}, &updates)
		if err != nil {
			return err
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			msg := u.Message
			if msg == nil || msg.Chat.Type != "private" || msg.From.IsBot {
				continue
			}
			name := msg.From.FirstName
			if msg.From.Username != "" {
				name += " (@" + msg.From.Username + ")"
			}
			chat := strconv.FormatInt(msg.Chat.ID, 10)
			m.answer(ctx, b, Inbound{Kind: "telegram", From: chat, FromName: name, Text: msg.Text}, chat)
		}
	}
	return ctx.Err()
}

// slack reads Socket Mode envelopes: direct messages to the app.
func (m *Manager) slack(ctx context.Context, b *bot) error {
	var open struct {
		URL string `json:"url"`
	}
	if err := notify.SlackCall(ctx, b.target.Config.AppToken, "apps.connections.open", map[string]any{}, &open); err != nil {
		return err
	}
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	ws, err := wsclient.Dial(dctx, open.URL)
	cancel()
	if err != nil {
		return err
	}
	defer func() { _ = ws.Close() }()
	go func() { <-ctx.Done(); _ = ws.Close() }()
	for {
		raw, err := ws.Read()
		if err != nil {
			return err
		}
		var env struct {
			Type       string `json:"type"`
			EnvelopeID string `json:"envelope_id"`
			Payload    struct {
				Event struct {
					Type        string `json:"type"`
					ChannelType string `json:"channel_type"`
					Channel     string `json:"channel"`
					User        string `json:"user"`
					Text        string `json:"text"`
					BotID       string `json:"bot_id"`
					Subtype     string `json:"subtype"`
				} `json:"event"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.EnvelopeID != "" { // acknowledge first, or Slack sends it again
			ack, _ := json.Marshal(map[string]string{"envelope_id": env.EnvelopeID})
			if err := ws.WriteText(ack); err != nil {
				return err
			}
		}
		switch env.Type {
		case "disconnect":
			return errors.New("slack asked to reconnect")
		case "events_api":
			e := env.Payload.Event
			if e.Type != "message" || e.ChannelType != "im" || e.BotID != "" || e.Subtype != "" || e.User == "" {
				continue
			}
			m.answer(ctx, b, Inbound{Kind: "slack", From: e.User, FromName: e.User, Text: slackText(e.Text)}, e.Channel)
		}
	}
}

// slackText turns Slack's markup back into what the person typed.
func slackText(s string) string {
	s = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(s)
	for {
		i := strings.Index(s, "<")
		j := strings.Index(s, ">")
		if i < 0 || j < i {
			return s
		}
		inner := s[i+1 : j]
		if k := strings.LastIndex(inner, "|"); k >= 0 {
			inner = inner[k+1:]
		}
		s = s[:i] + inner + s[j+1:]
	}
}

// Discord's gateway intents: direct messages only. Their content needs no
// privileged intent.
const discordIntents = 1 << 12

// discord connects to the gateway and reads direct messages.
func (m *Manager) discord(ctx context.Context, b *bot) error {
	var gw struct {
		URL string `json:"url"`
	}
	if err := notify.DiscordCall(ctx, b.target.Config.BotToken, http.MethodGet, "/gateway/bot", nil, &gw); err != nil {
		return err
	}
	dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	ws, err := wsclient.Dial(dctx, strings.TrimRight(gw.URL, "/")+"/?v=10&encoding=json")
	cancel()
	if err != nil {
		return err
	}
	lctx, stop := context.WithCancel(ctx)
	defer stop()
	defer func() { _ = ws.Close() }()
	go func() { <-lctx.Done(); _ = ws.Close() }()
	var seqMu sync.Mutex
	var seq *int
	send := func(op int, d any) error {
		b, _ := json.Marshal(map[string]any{"op": op, "d": d})
		return ws.WriteText(b)
	}
	heartbeat := func() error {
		seqMu.Lock()
		s := seq
		seqMu.Unlock()
		return send(1, s)
	}
	for {
		raw, err := ws.Read()
		if err != nil {
			return err
		}
		var f struct {
			Op int             `json:"op"`
			S  *int            `json:"s"`
			T  string          `json:"t"`
			D  json.RawMessage `json:"d"`
		}
		if err := json.Unmarshal(raw, &f); err != nil {
			continue
		}
		if f.S != nil {
			seqMu.Lock()
			seq = f.S
			seqMu.Unlock()
		}
		switch f.Op {
		case 10: // hello: start the heartbeat, then identify
			var hello struct {
				Interval int `json:"heartbeat_interval"`
			}
			_ = json.Unmarshal(f.D, &hello)
			if hello.Interval < 1000 {
				return fmt.Errorf("discord: heartbeat interval %d", hello.Interval)
			}
			go func() {
				t := time.NewTicker(time.Duration(hello.Interval) * time.Millisecond)
				defer t.Stop()
				for {
					select {
					case <-lctx.Done():
						return
					case <-t.C:
						if heartbeat() != nil {
							stop()
							return
						}
					}
				}
			}()
			if err := send(2, map[string]any{"token": b.target.Config.BotToken, "intents": discordIntents,
				"properties": map[string]string{"os": "linux", "browser": "axon", "device": "axon"}}); err != nil {
				return err
			}
		case 1: // the gateway asks for a heartbeat now
			if err := heartbeat(); err != nil {
				return err
			}
		case 7, 9: // reconnect, invalid session
			return fmt.Errorf("discord asked to reconnect (op %d)", f.Op)
		case 0:
			if f.T != "MESSAGE_CREATE" {
				continue
			}
			var msg struct {
				ChannelID string  `json:"channel_id"`
				GuildID   *string `json:"guild_id"`
				Content   string  `json:"content"`
				Author    struct {
					ID       string `json:"id"`
					Username string `json:"username"`
					Bot      bool   `json:"bot"`
				} `json:"author"`
			}
			if json.Unmarshal(f.D, &msg) != nil || msg.GuildID != nil || msg.Author.Bot {
				continue
			}
			m.answer(ctx, b, Inbound{Kind: "discord", From: msg.Author.ID, FromName: msg.Author.Username, Text: msg.Content}, msg.ChannelID)
		}
	}
}
