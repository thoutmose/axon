package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"

	"github.com/thoutmose/axon/api/internal/digest"
	"github.com/thoutmose/axon/api/internal/netsafe"
)

// Clients and addresses; tests replace them.
var (
	// Client talks to the chat services' own hosts.
	Client = &http.Client{Timeout: 20 * time.Second}
	// FreeClient talks to addresses people chose (ntfy, webhooks): it
	// refuses private networks.
	FreeClient = netsafe.Client(20 * time.Second)

	SlackAPI    = "https://slack.com/api"
	DiscordAPI  = "https://discord.com/api/v10"
	TelegramAPI = "https://api.telegram.org"
)

// Message is a short notice: a task reminder, the evening nudge, a reply.
type Message struct {
	Title string
	Body  string
	URL   string // path in Axon, made absolute with the base address
}

// limit cuts s to n bytes on a rune boundary, with an ellipsis.
func limit(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n - len("…")
	for cut > 0 && !utf8Start(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

func absolute(base, path string) string {
	if base == "" || path == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + path
}

// SendDigest sends a digest with its picture.
func SendDigest(ctx context.Context, t Target, d digest.Digest, base string, scene []byte) error {
	c := t.Config
	switch t.Kind {
	case "slack":
		payload := digest.Slack(d, base, c.Bot())
		if c.Bot() {
			payload["channel"] = t.Peer
			payload["unfurl_links"] = false
			return slackCall(ctx, c.BotToken, "chat.postMessage", payload, nil)
		}
		return postJSON(ctx, Client, c.WebhookURL, payload, nil)
	case "discord":
		payload := digest.Discord(d, base, c.Bot())
		return discordSend(ctx, t, payload, scene)
	case "teams":
		return postJSON(ctx, Client, c.WebhookURL, digest.Teams(d, base), nil)
	case "telegram":
		title, summary := digest.Short(d)
		if scene != nil {
			if err := telegramPhoto(ctx, c.BotToken, t.Peer, scene, "<b>"+escapeHTML(title)+"</b>\n"+escapeHTML(summary)); err != nil {
				return err
			}
		}
		return telegramCall(ctx, c.BotToken, "sendMessage", map[string]any{
			"chat_id": t.Peer, "text": limit(digest.Telegram(d, base), 4096), "parse_mode": "HTML", "link_preview_options": map[string]any{"is_disabled": true},
		}, nil)
	case "ntfy":
		title, _ := digest.Short(d)
		return ntfySend(ctx, c, title, digest.Markdown(d, base), absolute(base, d.URL), []string{"sunrise"})
	case "webhook":
		return webhookSend(ctx, c, "digest", map[string]any{"event": "digest", "digest": d, "text": digest.Text(d, base)})
	}
	return fmt.Errorf("unknown channel kind %q", t.Kind)
}

// SendText sends a short message.
func SendText(ctx context.Context, t Target, m Message, base string) error {
	c := t.Config
	link := absolute(base, m.URL)
	switch t.Kind {
	case "slack":
		text := "*" + slackEscape(m.Title) + "*\n" + slackEscape(m.Body)
		if link != "" {
			text += "\n<" + link + "|Axon>"
		}
		payload := map[string]any{"text": text, "unfurl_links": false}
		if c.Bot() {
			payload["channel"] = t.Peer
			return slackCall(ctx, c.BotToken, "chat.postMessage", payload, nil)
		}
		return postJSON(ctx, Client, c.WebhookURL, payload, nil)
	case "discord":
		text := "**" + m.Title + "**\n" + m.Body
		if link != "" {
			text += "\n<" + link + ">"
		}
		return discordSend(ctx, t, map[string]any{"username": "Axon", "content": limit(text, 2000), "allowed_mentions": map[string]any{"parse": []any{}}}, nil)
	case "teams":
		body := []any{
			map[string]any{"type": "TextBlock", "text": m.Title, "weight": "Bolder", "wrap": true},
			map[string]any{"type": "TextBlock", "text": m.Body, "wrap": true},
		}
		card := map[string]any{"$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "type": "AdaptiveCard", "version": "1.4", "body": body}
		if link != "" {
			card["actions"] = []any{map[string]any{"type": "Action.OpenUrl", "title": "Axon", "url": link}}
		}
		return postJSON(ctx, Client, c.WebhookURL, map[string]any{"type": "message", "attachments": []any{map[string]any{"contentType": "application/vnd.microsoft.card.adaptive", "content": card}}}, nil)
	case "telegram":
		text := "<b>" + escapeHTML(m.Title) + "</b>\n" + escapeHTML(m.Body)
		if link != "" {
			text += "\n<a href=\"" + escapeHTML(link) + "\">Axon</a>"
		}
		return telegramCall(ctx, c.BotToken, "sendMessage", map[string]any{"chat_id": t.Peer, "text": limit(text, 4096), "parse_mode": "HTML", "link_preview_options": map[string]any{"is_disabled": true}}, nil)
	case "ntfy":
		return ntfySend(ctx, c, m.Title, m.Body, link, []string{"bell"})
	case "webhook":
		return webhookSend(ctx, c, "message", map[string]any{"event": "message", "title": m.Title, "body": m.Body, "url": link})
	}
	return fmt.Errorf("unknown channel kind %q", t.Kind)
}

func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

func slackEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// do sends a request and fails on a non-2xx answer, quoting a little of it.
func do(hc *http.Client, req *http.Request, out any) error {
	res, err := hc.Do(req)
	if err != nil {
		return redact(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("%s answered %d: %s", req.URL.Host, res.StatusCode, limit(strings.TrimSpace(string(body)), 200))
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

// redact keeps webhook paths and bot tokens (which sit in Telegram's
// paths) out of error messages that end up in the channel's last error.
func redact(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		if u, perr := url.Parse(ue.URL); perr == nil {
			return fmt.Errorf("%s %s://%s/…: %w", ue.Op, u.Scheme, u.Host, ue.Err)
		}
	}
	return err
}

func postJSON(ctx context.Context, hc *http.Client, url string, payload any, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return redact(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return do(hc, req, out)
}

// slackCall calls a Slack Web API method with a bot token.
func slackCall(ctx context.Context, token, method string, payload any, out any) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, SlackAPI+"/"+method, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	var res struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	raw := json.RawMessage{}
	if err := do(Client, req, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}
	if !res.OK {
		return fmt.Errorf("slack %s: %s", method, res.Error)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// discordSend posts through the webhook, or as the bot in the person's DMs,
// with the picture attached when there is one.
func discordSend(ctx context.Context, t Target, payload map[string]any, scene []byte) error {
	url := t.Config.WebhookURL + "?wait=true"
	auth := ""
	if t.Config.Bot() {
		var dm struct {
			ID string `json:"id"`
		}
		if err := discordCall(ctx, t.Config.BotToken, http.MethodPost, "/users/@me/channels", map[string]any{"recipient_id": t.Peer}, &dm); err != nil {
			return err
		}
		url, auth = DiscordAPI+"/channels/"+dm.ID+"/messages", "Bot "+t.Config.BotToken
		delete(payload, "username")
	}
	if scene == nil {
		delete(payload, "attachments")
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	pj, _ := json.Marshal(payload)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="payload_json"`)
	h.Set("Content-Type", "application/json")
	part, _ := w.CreatePart(h)
	_, _ = part.Write(pj)
	if scene != nil {
		h = textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="files[0]"; filename="scene.png"`)
		h.Set("Content-Type", "image/png")
		part, _ = w.CreatePart(h)
		_, _ = part.Write(scene)
	}
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return redact(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return do(Client, req, nil)
}

func discordCall(ctx context.Context, token, method, path string, payload, out any) error {
	var rd io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, DiscordAPI+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")
	return do(Client, req, out)
}

// telegramCall calls a Bot API method; the token is part of the path.
func telegramCall(ctx context.Context, token, method string, payload, out any) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TelegramAPI+"/bot"+token+"/"+method, bytes.NewReader(b))
	if err != nil {
		return redact(err)
	}
	req.Header.Set("Content-Type", "application/json")
	var res struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := do(Client, req, &res); err != nil {
		return redactToken(err, token)
	}
	if !res.OK {
		return fmt.Errorf("telegram %s: %s", method, res.Description)
	}
	if out != nil {
		return json.Unmarshal(res.Result, out)
	}
	return nil
}

func redactToken(err error, token string) error {
	return errors.New(strings.ReplaceAll(err.Error(), token, "…"))
}

func telegramPhoto(ctx context.Context, token, chat string, png []byte, caption string) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("chat_id", chat)
	_ = w.WriteField("caption", limit(caption, 1024))
	_ = w.WriteField("parse_mode", "HTML")
	part, _ := w.CreateFormFile("photo", "scene.png")
	_, _ = part.Write(png)
	_ = w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TelegramAPI+"/bot"+token+"/sendPhoto", &body)
	if err != nil {
		return redactToken(err, token)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := do(Client, req, nil); err != nil {
		return redactToken(err, token)
	}
	return nil
}

func ntfySend(ctx context.Context, c Config, title, message, click string, tags []string) error {
	payload := map[string]any{"topic": c.Topic, "title": limit(title, 250), "message": limit(message, 4000), "markdown": true, "tags": tags}
	if click != "" {
		payload["click"] = click
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.Server, "/")+"/", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return do(FreeClient, req, nil)
}

// webhookSend posts JSON; with a secret, X-Axon-Signature is
// "sha256=" + hex(HMAC-SHA256(secret, body)), as GitHub does.
func webhookSend(ctx context.Context, c Config, event string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(b))
	if err != nil {
		return redact(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Axon")
	req.Header.Set("X-Axon-Event", event)
	if c.Secret != "" {
		m := hmac.New(sha256.New, []byte(c.Secret))
		m.Write(b)
		req.Header.Set("X-Axon-Signature", "sha256="+hex.EncodeToString(m.Sum(nil)))
	}
	return do(FreeClient, req, nil)
}

// Reply answers a message in the chat it came from, as plain text (EP-71).
func Reply(ctx context.Context, t Target, chat, text string) error {
	c := t.Config
	switch t.Kind {
	case "telegram":
		return telegramCall(ctx, c.BotToken, "sendMessage", map[string]any{"chat_id": chat, "text": limit(text, 4096), "link_preview_options": map[string]any{"is_disabled": true}}, nil)
	case "slack":
		return slackCall(ctx, c.BotToken, "chat.postMessage", map[string]any{"channel": chat, "text": slackEscape(text), "unfurl_links": false}, nil)
	case "discord":
		return discordCall(ctx, c.BotToken, http.MethodPost, "/channels/"+chat+"/messages", map[string]any{"content": limit(text, 2000), "allowed_mentions": map[string]any{"parse": []any{}}}, nil)
	}
	return fmt.Errorf("%s cannot answer", t.Kind)
}

// SlackCall and DiscordCall let listeners use the same clients and addresses.
func SlackCall(ctx context.Context, token, method string, payload, out any) error {
	return slackCall(ctx, token, method, payload, out)
}

// DiscordCall calls the Discord API as a bot.
func DiscordCall(ctx context.Context, token, method, path string, payload, out any) error {
	return discordCall(ctx, token, method, path, payload, out)
}

// TelegramCall calls the Bot API.
func TelegramCall(ctx context.Context, token, method string, payload, out any) error {
	return telegramCall(ctx, token, method, payload, out)
}
