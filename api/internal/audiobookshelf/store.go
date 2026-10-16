package audiobookshelf

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/watch"
)

// Origin is the origin of the tech watch sources a server brings.
const Origin = "audiobookshelf"

// MaxShows bounds the shows one sync follows.
const MaxShows = 300

// ValidationError says what to correct.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// ErrNotConnected means the person has no server.
var ErrNotConnected = errors.New("no Audiobookshelf server")

// Status is the person's server as they see it, without its token.
type Status struct {
	Connected bool       `json:"connected"`
	URL       string     `json:"url"`
	SyncedAt  *time.Time `json:"synced_at"`
	Error     string     `json:"error"`
	Shows     int        `json:"shows"`
}

// Store keeps each person's server.
type Store struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
}

// NewStore returns the store, and makes the tech watch read its shows.
func NewStore(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer) *Store {
	s := &Store{db: pool, log: log, sealer: sealer}
	watch.RegisterOrigin(Origin, watch.Origin{Fetch: s.fetch, Audio: s.audio})
	return s
}

func (s *Store) seal(token string) []byte {
	if s.sealer != nil {
		return append([]byte{1}, s.sealer.Seal([]byte(token))...)
	}
	return append([]byte{0}, token...)
}

func (s *Store) open(b []byte) (string, error) {
	if len(b) == 0 {
		return "", errors.New("no token")
	}
	raw := b[1:]
	if b[0] == 1 {
		if s.sealer == nil {
			return "", errors.New("the token is sealed and AXON_ENCRYPTION_KEY is not set")
		}
		var err error
		if raw, err = s.sealer.Open(raw); err != nil {
			return "", err
		}
	}
	return string(raw), nil
}

// client is the person's server, ready to call.
func (s *Store) client(ctx context.Context, q db.Querier, userID uuid.UUID) (*Client, error) {
	var base string
	var token []byte
	err := q.QueryRow(ctx, `SELECT url, token FROM audiobookshelf_servers WHERE user_id = $1`, userID).Scan(&base, &token)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotConnected
	}
	if err != nil {
		return nil, err
	}
	t, err := s.open(token)
	return &Client{Base: base, Token: t}, err
}

// Get is the person's server.
func (s *Store) Get(ctx context.Context, userID uuid.UUID) (Status, error) {
	var st Status
	err := s.db.QueryRow(ctx, `SELECT url, synced_at, error,
		(SELECT count(*) FROM watch_sources WHERE user_id = $1 AND origin = $2) FROM audiobookshelf_servers WHERE user_id = $1`,
		userID, Origin).Scan(&st.URL, &st.SyncedAt, &st.Error, &st.Shows)
	if errors.Is(err, pgx.ErrNoRows) {
		return Status{}, nil
	}
	st.Connected = err == nil
	return st, err
}

// checkURL keeps an http(s) address with no password in it.
func checkURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || len(raw) > 500 || u.RawQuery != "" {
		return "", ValidationError{"Give your Audiobookshelf's address, like https://abs.example.org, with no password in it."}
	}
	return raw, nil
}

// Connect checks the address and token with the server, keeps them, and
// syncs. An empty token keeps the one already saved.
func (s *Store) Connect(ctx context.Context, userID uuid.UUID, rawURL, token string, now time.Time) (Status, error) {
	base, err := checkURL(rawURL)
	if err != nil {
		return Status{}, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		old, err := s.client(ctx, s.db, userID)
		if err != nil {
			return Status{}, ValidationError{"Give an API key from Audiobookshelf (Settings → API Keys)."}
		}
		token = old.Token
	}
	if len(token) > 2000 {
		return Status{}, ValidationError{"That API token is too long."}
	}
	if _, err := (&Client{Base: base, Token: token}).Libraries(ctx); err != nil {
		return Status{}, err
	}
	if _, err := s.db.Exec(ctx, `INSERT INTO audiobookshelf_servers (user_id, url, token) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET url = EXCLUDED.url, token = EXCLUDED.token, error = ''`, userID, base, s.seal(token)); err != nil {
		return Status{}, err
	}
	if _, err := s.Sync(ctx, userID, now); err != nil {
		return Status{}, err
	}
	return s.Get(ctx, userID)
}

// Disconnect forgets the server and stops following its shows; the
// episodes already read stay in the tech watch.
func (s *Store) Disconnect(ctx context.Context, userID uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id FROM watch_sources WHERE user_id = $1 AND origin = $2`, userID, Origin)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := watch.DeleteSource(ctx, tx, userID, id, false); err != nil {
				return err
			}
		}
		tag, err := tx.Exec(ctx, `DELETE FROM audiobookshelf_servers WHERE user_id = $1`, userID)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotConnected
		}
		return err
	})
}

// Synced is what a sync brought.
type Synced struct {
	Shows    int `json:"shows"`
	New      int `json:"new_shows"`
	Episodes int `json:"new_episodes"`
}

// Sync follows every show of the server's podcast libraries, and reads
// the new episodes of those already followed.
func (s *Store) Sync(ctx context.Context, userID uuid.UUID, now time.Time) (Synced, error) {
	var out Synced
	c, err := s.client(ctx, s.db, userID)
	if err != nil {
		return out, err
	}
	report := func(cause error) (Synced, error) {
		msg := ""
		if cause != nil {
			msg = cause.Error()
			if len(msg) > 500 {
				msg = msg[:500]
			}
		}
		_, err := s.db.Exec(ctx, `UPDATE audiobookshelf_servers SET synced_at = $2, error = $3 WHERE user_id = $1`, userID, now, msg)
		return out, errors.Join(cause, err)
	}
	libs, err := c.Libraries(ctx)
	if err != nil {
		return report(err)
	}
	for _, l := range libs {
		shows, err := c.Shows(ctx, l.ID)
		if err != nil {
			return report(err)
		}
		for _, show := range shows {
			if out.Shows == MaxShows {
				break
			}
			out.Shows++
			page := c.base() + "/item/" + show.ID
			var id uuid.UUID
			err := s.db.QueryRow(ctx, `SELECT id FROM watch_sources WHERE user_id = $1 AND url = $2`, userID, page).Scan(&id)
			switch {
			case err == nil:
				st, err := watch.FetchSource(ctx, s.db, userID, id, now)
				if err != nil {
					s.log.Warn("audiobookshelf show failed", "source", id, "err", err)
					continue
				}
				out.Episodes += st.New
			case errors.Is(err, pgx.ErrNoRows):
				feed, err := c.Feed(ctx, show.ID)
				if err != nil {
					return report(err)
				}
				title := firstOf(show.Title, feed.Title, "Podcast")
				var st watch.Stats
				err = pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) (err error) {
					_, st, err = watch.CreateSource(ctx, tx, userID, watch.SourceInput{URL: page, Title: &title, Origin: Origin},
						watch.Found{URL: page, Feed: feed}, now)
					return err
				})
				if err != nil {
					return report(err)
				}
				out.New++
				out.Episodes += st.New
			default:
				return report(err)
			}
		}
	}
	return report(nil)
}

func firstOf(s ...string) string {
	for _, x := range s {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}

// fetch reads a show for the tech watch's worker.
func (s *Store) fetch(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, src watch.Source) (watch.Feed, error) {
	c, err := s.client(ctx, pool, userID)
	if err != nil {
		return watch.Feed{}, err
	}
	prefix := c.base() + "/item/"
	if !strings.HasPrefix(src.URL, prefix) {
		return watch.Feed{}, errors.New("this show is on another Audiobookshelf server than yours")
	}
	return c.Feed(ctx, strings.TrimPrefix(src.URL, prefix))
}

// audio reads an episode from the person's server, for Whisper.
func (s *Store) audio(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, guid string) ([]byte, string, error) {
	c, err := s.client(ctx, pool, userID)
	if err != nil {
		return nil, "", err
	}
	return c.Audio(ctx, guid)
}
