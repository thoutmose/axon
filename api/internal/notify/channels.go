// Package notify sends digests and reminders to the chat apps a person
// connects (EP-70): Slack, Discord, Microsoft Teams, Telegram, ntfy, or any
// web address. Webhook channels only send; bot channels (Slack, Discord,
// Telegram) also read the person's answers (EP-71) once they link the chat
// with a one-time code. Addresses and tokens are credentials: they are
// sealed at rest and never returned.
package notify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/digest"
	"github.com/thoutmose/axon/api/internal/netsafe"
)

// Kinds of channel.
var Kinds = []string{"slack", "discord", "teams", "telegram", "ntfy", "webhook"}

// Errors a person can act on.
var (
	ErrNotFound = errors.New("no such channel")
	ErrNoKey    = errors.New("AXON_ENCRYPTION_KEY is not set")
)

// InvalidError says what to correct.
type InvalidError struct{ Msg string }

func (e InvalidError) Error() string { return e.Msg }

// Config holds a channel's secrets; which fields apply depends on the kind.
type Config struct {
	WebhookURL string `json:"webhook_url,omitempty"` // slack, discord, teams, webhook
	BotToken   string `json:"bot_token,omitempty"`   // slack (xoxb-), discord, telegram
	AppToken   string `json:"app_token,omitempty"`   // slack Socket Mode (xapp-), to read answers
	Server     string `json:"server,omitempty"`      // ntfy, default https://ntfy.sh
	Topic      string `json:"topic,omitempty"`       // ntfy
	Token      string `json:"token,omitempty"`       // ntfy access token
	Secret     string `json:"secret,omitempty"`      // webhook: signs each body (HMAC-SHA256)
}

// Bot reports whether the channel talks through a bot (and can read answers).
func (c Config) Bot() bool { return c.BotToken != "" }

// Channel is what the API shows: no secret.
type Channel struct {
	ID            uuid.UUID  `json:"id"`
	Kind          string     `json:"kind"`
	Name          string     `json:"name"`
	Target        string     `json:"target"`
	Mode          string     `json:"mode"` // webhook (sends) or bot (sends and reads answers)
	Digests       []string   `json:"digests"`
	Reminders     bool       `json:"reminders"`
	Enabled       bool       `json:"enabled"`
	Linked        bool       `json:"linked"`
	PeerName      string     `json:"peer_name"`
	LinkExpiresAt *time.Time `json:"link_expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	LastSentAt    *time.Time `json:"last_sent_at"`
	LastError     string     `json:"last_error"`
}

// Input creates or changes a channel. Secrets left empty on change keep
// their current value.
type Input struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Config    Config   `json:"config"`
	Digests   []string `json:"digests"`
	Reminders *bool    `json:"reminders"`
	Enabled   *bool    `json:"enabled"`
}

var (
	slackHook   = regexp.MustCompile(`^https://hooks\.slack\.com/services/[A-Za-z0-9_/-]{10,200}$`)
	discordHook = regexp.MustCompile(`^https://(ptb\.|canary\.)?(discord|discordapp)\.com/api(/v\d+)?/webhooks/\d{5,30}/[A-Za-z0-9_-]{20,120}$`)
	telegramBot = regexp.MustCompile(`^\d{5,15}:[A-Za-z0-9_-]{30,60}$`)
	ntfyTopic   = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	teamsHosts  = []string{".webhook.office.com", ".logic.azure.com", ".powerplatform.com", ".powerautomate.com"}
)

// teamsURL accepts the addresses Teams incoming webhooks and Workflows give.
func teamsURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Port() != "" || len(raw) > 2000 {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return slices.ContainsFunc(teamsHosts, func(s string) bool { return strings.HasSuffix(h, s) })
}

// freeURL checks an address a person chose (ntfy server, webhook): HTTPS,
// unless the operator allows private networks; the client then refuses
// private addresses whatever the name resolves to.
func freeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || len(raw) > 2000 {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && netsafe.AllowPrivate())
}

// check validates a channel's settings and returns what to show of it.
func check(kind string, c Config) (string, error) {
	switch kind {
	case "slack":
		switch {
		case c.BotToken != "":
			if !strings.HasPrefix(c.BotToken, "xoxb-") || len(c.BotToken) > 200 {
				return "", InvalidError{"A Slack bot token starts with xoxb-."}
			}
			if !strings.HasPrefix(c.AppToken, "xapp-") || len(c.AppToken) > 300 {
				return "", InvalidError{"To read your answers, Slack also needs an app-level token (xapp-) with connections:write, for Socket Mode."}
			}
			return "Slack app", nil
		case slackHook.MatchString(c.WebhookURL):
			return "hooks.slack.com/…", nil
		}
		return "", InvalidError{"Paste a Slack incoming webhook (https://hooks.slack.com/services/…), or a bot token and an app token."}
	case "discord":
		switch {
		case c.BotToken != "":
			if len(c.BotToken) < 50 || len(c.BotToken) > 100 || strings.ContainsAny(c.BotToken, " \r\n") {
				return "", InvalidError{"That is not a Discord bot token."}
			}
			return "Discord bot", nil
		case discordHook.MatchString(c.WebhookURL):
			return "discord.com/api/webhooks/…", nil
		}
		return "", InvalidError{"Paste a Discord webhook (https://discord.com/api/webhooks/…), or a bot token."}
	case "teams":
		if !teamsURL(c.WebhookURL) {
			return "", InvalidError{"Paste the address of a Teams Workflows webhook (…webhook.office.com or …logic.azure.com)."}
		}
		u, _ := url.Parse(c.WebhookURL)
		return u.Hostname() + "/…", nil
	case "telegram":
		if !telegramBot.MatchString(c.BotToken) {
			return "", InvalidError{"Paste the token @BotFather gave your bot, like 123456789:AA…"}
		}
		return "Telegram bot", nil
	case "ntfy":
		server := c.Server
		if server == "" {
			server = "https://ntfy.sh"
		}
		if !freeURL(server) || !ntfyTopic.MatchString(c.Topic) || len(c.Token) > 200 {
			return "", InvalidError{"ntfy needs an HTTPS server (https://ntfy.sh by default) and a topic of letters, digits, - and _."}
		}
		u, _ := url.Parse(server)
		return u.Host + "/" + c.Topic, nil
	case "webhook":
		if !freeURL(c.WebhookURL) || len(c.Secret) > 200 {
			return "", InvalidError{"A webhook is an HTTPS address."}
		}
		u, _ := url.Parse(c.WebhookURL)
		return u.Host + "/…", nil
	}
	return "", InvalidError{"The kind is one of " + strings.Join(Kinds, ", ") + "."}
}

// merge keeps the current secrets where the change leaves them empty.
func merge(old, in Config) Config {
	keep := func(n, o string) string {
		if n == "" {
			return o
		}
		return n
	}
	return Config{
		WebhookURL: keep(in.WebhookURL, old.WebhookURL), BotToken: keep(in.BotToken, old.BotToken), AppToken: keep(in.AppToken, old.AppToken),
		Server: keep(in.Server, old.Server), Topic: keep(in.Topic, old.Topic), Token: keep(in.Token, old.Token), Secret: keep(in.Secret, old.Secret),
	}
}

// Store keeps channels; the sealer may be nil, then secrets are stored
// readable by the database's administrator only.
type Store struct {
	db     db.Querier
	sealer *auth.Sealer
}

// NewStore returns a store.
func NewStore(q db.Querier, sealer *auth.Sealer) *Store { return &Store{db: q, sealer: sealer} }

func (s *Store) seal(c Config) ([]byte, error) {
	b, err := json.Marshal(c) //nolint:gosec // the secrets are stored on purpose, sealed below
	if err != nil {
		return nil, err
	}
	if s.sealer != nil {
		return append([]byte{1}, s.sealer.Seal(b)...), nil
	}
	return append([]byte{0}, b...), nil
}

func (s *Store) open(raw []byte) (Config, error) {
	var c Config
	if len(raw) == 0 {
		return c, errors.New("empty channel config")
	}
	b := raw[1:]
	if raw[0] == 1 {
		if s.sealer == nil {
			return c, ErrNoKey
		}
		var err error
		if b, err = s.sealer.Open(b); err != nil {
			return c, err
		}
	}
	return c, json.Unmarshal(b, &c)
}

const channelColumns = `id, kind, name, target, digests, reminders, enabled, peer <> '', peer_name, link_expires_at, created_at, last_sent_at, last_error`

func scanChannel(r pgx.Row, cfg *[]byte) (Channel, error) {
	var c Channel
	err := r.Scan(&c.ID, &c.Kind, &c.Name, &c.Target, &c.Digests, &c.Reminders, &c.Enabled, &c.Linked, &c.PeerName, &c.LinkExpiresAt, &c.CreatedAt, &c.LastSentAt, &c.LastError, cfg)
	return c, err
}

// List returns the person's channels.
func (s *Store) List(ctx context.Context, userID uuid.UUID) ([]Channel, error) {
	rows, err := s.db.Query(ctx, `SELECT `+channelColumns+`, config FROM notify_channels WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Channel{}
	for rows.Next() {
		var raw []byte
		c, err := scanChannel(rows, &raw)
		if err != nil {
			return nil, err
		}
		s.fill(&c, raw)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns one channel of the person.
func (s *Store) Get(ctx context.Context, userID, id uuid.UUID) (Channel, error) {
	var raw []byte
	c, err := scanChannel(s.db.QueryRow(ctx, `SELECT `+channelColumns+`, config FROM notify_channels WHERE id = $1 AND user_id = $2`, id, userID), &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	s.fill(&c, raw)
	return c, nil
}

func (s *Store) fill(c *Channel, raw []byte) {
	c.Mode = "webhook"
	if cfg, err := s.open(raw); err == nil && cfg.Bot() {
		c.Mode = "bot"
	}
	if c.Linked || c.Mode == "webhook" {
		c.LinkExpiresAt = nil
	} else {
		c.LinkExpiresAt = nilIfPast(c.LinkExpiresAt)
	}
}

func nilIfPast(t *time.Time) *time.Time {
	if t == nil || t.Before(time.Now()) {
		return nil
	}
	return t
}

func cleanDigests(in []string) ([]string, error) {
	if in == nil {
		return digest.Kinds, nil
	}
	out := []string{}
	for _, k := range in {
		if !digest.Valid(k) {
			return nil, InvalidError{"Digests are daily, weekly, and monthly."}
		}
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out, nil
}

// LinkTTL is how long a code to link a chat lasts.
const LinkTTL = 30 * time.Minute

// Save creates a channel (id is uuid.Nil) or changes one, and returns it
// with a code to link the chat when it is a bot that is not linked yet.
func (s *Store) Save(ctx context.Context, userID, id uuid.UUID, in Input, now time.Time) (Channel, string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > 100 {
		return Channel{}, "", InvalidError{"Name the channel, in up to 100 characters."}
	}
	digests, err := cleanDigests(in.Digests)
	if err != nil {
		return Channel{}, "", err
	}
	cfg := in.Config
	kind := in.Kind
	linked := false
	if id != uuid.Nil {
		var raw []byte
		var peer string
		err := s.db.QueryRow(ctx, `SELECT kind, config, peer FROM notify_channels WHERE id = $1 AND user_id = $2`, id, userID).Scan(&kind, &raw, &peer)
		if errors.Is(err, pgx.ErrNoRows) {
			return Channel{}, "", ErrNotFound
		}
		if err != nil {
			return Channel{}, "", err
		}
		old, err := s.open(raw)
		if err != nil {
			return Channel{}, "", err
		}
		if in.Config.BotToken != "" && in.Config.BotToken != old.BotToken {
			peer = "" // a new bot: link again
		}
		cfg = merge(old, in.Config)
		if in.Config.WebhookURL != "" && in.Config.BotToken == "" { // switched to a webhook
			cfg.BotToken, cfg.AppToken = "", ""
		}
		linked = peer != ""
	}
	if kind == "ntfy" && cfg.Server == "" {
		cfg.Server = "https://ntfy.sh"
	}
	target, err := check(kind, cfg)
	if err != nil {
		return Channel{}, "", err
	}
	sealed, err := s.seal(cfg)
	if err != nil {
		return Channel{}, "", err
	}
	reminders := in.Reminders == nil || *in.Reminders
	enabled := in.Enabled == nil || *in.Enabled
	var code string
	var hash []byte
	var expires *time.Time
	if cfg.Bot() && !linked {
		code, hash = newCode()
		e := now.Add(LinkTTL)
		expires = &e
	}
	if id == uuid.Nil {
		id = uuid.Must(uuid.NewV7())
		_, err = s.db.Exec(ctx, `INSERT INTO notify_channels (id, user_id, kind, name, config, target, digests, reminders, enabled, link_hash, link_expires_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, id, userID, kind, name, sealed, target, digests, reminders, enabled, hash, expires)
	} else {
		_, err = s.db.Exec(ctx, `UPDATE notify_channels SET name = $3, config = $4, target = $5, digests = $6, reminders = $7, enabled = $8,
			link_hash = CASE WHEN $9::bytea IS NULL THEN link_hash ELSE $9 END,
			link_expires_at = CASE WHEN $9::bytea IS NULL THEN link_expires_at ELSE $10 END,
			peer = CASE WHEN $9::bytea IS NULL THEN peer ELSE '' END,
			peer_name = CASE WHEN $9::bytea IS NULL THEN peer_name ELSE '' END,
			last_error = '' WHERE id = $1 AND user_id = $2`, id, userID, name, sealed, target, digests, reminders, enabled, hash, expires)
	}
	if err != nil {
		return Channel{}, "", err
	}
	c, err := s.Get(ctx, userID, id)
	return c, code, err
}

// Relink makes a new code to link a bot's chat, as when someone else took it.
func (s *Store) Relink(ctx context.Context, userID, id uuid.UUID, now time.Time) (string, error) {
	code, hash := newCode()
	tag, err := s.db.Exec(ctx, `UPDATE notify_channels SET link_hash = $3, link_expires_at = $4, peer = '', peer_name = ''
		WHERE id = $1 AND user_id = $2`, id, userID, hash, now.Add(LinkTTL))
	if err == nil && tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return code, err
}

// Remove deletes a channel.
func (s *Store) Remove(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM notify_channels WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// Target is a channel with what sending needs.
type Target struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Kind   string
	Name   string
	Config Config
	Peer   string
	Lang   string
}

// Targets returns the person's enabled channels for a digest kind, or for
// reminders when kind is empty. Bots not linked yet are left out: they do
// not know whom to write to.
func (s *Store) Targets(ctx context.Context, userID uuid.UUID, kind string) ([]Target, error) {
	rows, err := s.db.Query(ctx, `SELECT id, kind, name, config, peer FROM notify_channels
		WHERE user_id = $1 AND enabled AND (($2 = '' AND reminders) OR $2 = ANY (digests)) ORDER BY created_at`, userID, kind)
	if err != nil {
		return nil, err
	}
	type row struct {
		id               uuid.UUID
		kind, name, peer string
		raw              []byte
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.id, &x.kind, &x.name, &x.raw, &x.peer)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, x := range list {
		cfg, err := s.open(x.raw)
		if err != nil {
			return nil, fmt.Errorf("channel %s: %w", x.id, err)
		}
		if cfg.Bot() && x.peer == "" {
			continue
		}
		out = append(out, Target{ID: x.id, UserID: userID, Kind: x.kind, Name: x.name, Config: cfg, Peer: x.peer})
	}
	return out, nil
}

// Sent records how a delivery went.
func (s *Store) Sent(ctx context.Context, id uuid.UUID, sendErr error, now time.Time) {
	msg := ""
	if sendErr != nil {
		msg = sendErr.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		_, _ = s.db.Exec(ctx, `UPDATE notify_channels SET last_error = $2 WHERE id = $1`, id, msg)
		return
	}
	_, _ = s.db.Exec(ctx, `UPDATE notify_channels SET last_sent_at = $2, last_error = '' WHERE id = $1`, id, now)
}

// Export is the channels part of a personal data export, without secrets.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return NewStore(q, nil).listPlain(ctx, userID)
}

func (s *Store) listPlain(ctx context.Context, userID uuid.UUID) ([]Channel, error) {
	rows, err := s.db.Query(ctx, `SELECT id, kind, name, target, digests, reminders, enabled, peer_name, created_at, last_sent_at FROM notify_channels WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Channel, error) {
		var c Channel
		err := r.Scan(&c.ID, &c.Kind, &c.Name, &c.Target, &c.Digests, &c.Reminders, &c.Enabled, &c.PeerName, &c.CreatedAt, &c.LastSentAt)
		return c, err
	})
}

// Codes to link a chat: 8 characters without look-alikes, hashed at rest.
var codeAlphabet = base32.NewEncoding("ABCDEFGHJKLMNPQRSTUVWXYZ23456789").WithPadding(base32.NoPadding)

func newCode() (string, []byte) {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	c := codeAlphabet.EncodeToString(b)
	return c[:4] + "-" + c[4:], HashCode(c)
}

// HashCode hashes a link code as typed, ignoring case, spaces, and dashes.
func HashCode(code string) []byte {
	c := strings.ToUpper(strings.NewReplacer("-", "", " ", "").Replace(code))
	h := sha256.Sum256([]byte(c))
	return h[:]
}

// Bots lists every enabled bot channel, for the listeners (EP-71).
func (s *Store) Bots(ctx context.Context) ([]Target, error) {
	rows, err := s.db.Query(ctx, `SELECT id, user_id, kind, name, config, peer FROM notify_channels WHERE enabled AND kind IN ('telegram', 'slack', 'discord')`)
	if err != nil {
		return nil, err
	}
	type row struct {
		t   Target
		raw []byte
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.t.ID, &x.t.UserID, &x.t.Kind, &x.t.Name, &x.raw, &x.t.Peer)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, x := range list {
		cfg, err := s.open(x.raw)
		if err != nil || !cfg.Bot() {
			continue
		}
		x.t.Config = cfg
		out = append(out, x.t)
	}
	return out, nil
}
