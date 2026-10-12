package calendars

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/auth"
)

// Limits: the size of one calendar, how many events are kept from it, and
// how often it is fetched again.
const (
	maxICS    = 5 << 20
	maxEvents = 5000
	refresh   = 30 * time.Minute
)

// ErrAddress is a calendar address Axon will not fetch, with the reason.
type ErrAddress struct{ Msg string }

func (e ErrAddress) Error() string { return e.Msg }

// allowPrivate lets an instance reach calendars on its own network, such as
// a Nextcloud on the LAN or the tailnet (AXON_CALENDAR_PRIVATE=1).
func allowPrivate() bool { return os.Getenv("AXON_CALENDAR_PRIVATE") == "1" }

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func private(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() || cgnat.Contains(ip)
}

// Client fetches calendars. Unless private addresses are allowed, it
// refuses to connect to them, whatever the name resolves to, so an address
// cannot point Axon at its own network. Tests replace it.
var Client = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: func(_, address string, _ syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				if ip := net.ParseIP(host); ip != nil && private(ip) && !allowPrivate() {
					return ErrAddress{"That address is on a private network. The operator can allow it with AXON_CALENDAR_PRIVATE=1."}
				}
				return nil
			},
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" && !allowPrivate() {
			return ErrAddress{"The calendar redirected to an address that is not HTTPS."}
		}
		return nil
	},
}

// Clean checks a calendar address: webcal:// becomes https://; it must be
// HTTPS (or HTTP when private addresses are allowed).
func Clean(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "webcal://") {
		raw = "https://" + raw[len("webcal://"):]
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || len(raw) > 2000 {
		return nil, ErrAddress{"That is not a calendar address. Copy the iCal (ICS) address your calendar shares."}
	}
	if u.Scheme != "https" && (u.Scheme != "http" || !allowPrivate()) {
		return nil, ErrAddress{"Calendar addresses start with https:// (or webcal://)."}
	}
	if u.User != nil {
		return nil, ErrAddress{"Put no password in the address; use the calendar's secret link instead."}
	}
	return u, nil
}

// Fetch downloads a calendar and reads the events in the window around now.
func Fetch(ctx context.Context, u *url.URL, now time.Time, loc *time.Location) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/calendar, */*;q=0.5")
	req.Header.Set("User-Agent", "Axon calendar reader (self-hosted)")
	res, err := Client.Do(req)
	if err != nil {
		var ea ErrAddress
		if errors.As(err, &ea) {
			return nil, ea
		}
		return nil, fmt.Errorf("could not reach the calendar: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the calendar answered %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxICS+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxICS {
		return nil, ErrAddress{"That calendar is larger than 5 MB."}
	}
	if !strings.Contains(string(data[:min(len(data), 2048)]), "BEGIN:VCALENDAR") {
		return nil, ErrAddress{"That address does not return a calendar (ICS)."}
	}
	evs := Parse(data, now.AddDate(0, 0, -60), now.AddDate(1, 0, 0), loc)
	if len(evs) > maxEvents {
		evs = evs[:maxEvents]
	}
	return evs, nil
}

// Store keeps sources and their events.
type Store struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	sealer *auth.Sealer
	now    func() time.Time
}

// NewStore returns the calendar sources store.
func NewStore(pool *pgxpool.Pool, log *slog.Logger, sealer *auth.Sealer) *Store {
	return &Store{db: pool, log: log, sealer: sealer, now: time.Now}
}

func (s *Store) seal(u string) []byte {
	if s.sealer != nil {
		return append([]byte{1}, s.sealer.Seal([]byte(u))...)
	}
	return append([]byte{0}, u...)
}

func (s *Store) open(b []byte) (*url.URL, error) {
	if len(b) == 0 {
		return nil, errors.New("empty address")
	}
	raw := b[1:]
	if b[0] == 1 {
		if s.sealer == nil {
			return nil, errors.New("the address is sealed and AXON_ENCRYPTION_KEY is not set")
		}
		var err error
		if raw, err = s.sealer.Open(raw); err != nil {
			return nil, err
		}
	}
	return url.Parse(string(raw))
}

func (s *Store) zone(ctx context.Context, userID uuid.UUID) *time.Location {
	var tz string
	if err := s.db.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err == nil {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.UTC
}

// Refresh fetches one source and replaces its events; a failure is kept on
// the source for the person to see, and the old events stay.
func (s *Store) Refresh(ctx context.Context, id uuid.UUID) error {
	var userID uuid.UUID
	var raw []byte
	if err := s.db.QueryRow(ctx, `SELECT user_id, url FROM calendar_sources WHERE id = $1`, id).Scan(&userID, &raw); err != nil {
		return err
	}
	u, err := s.open(raw)
	if err != nil {
		return err
	}
	evs, ferr := Fetch(ctx, u, s.now(), s.zone(ctx, userID))
	if ferr != nil {
		_, err := s.db.Exec(ctx, `UPDATE calendar_sources SET fetched_at = now(), error = $2 WHERE id = $1`, id, ferr.Error())
		if err != nil {
			return err
		}
		return ferr
	}
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM calendar_events WHERE source_id = $1`, id); err != nil {
			return err
		}
		rows := make([][]any, len(evs))
		for i, e := range evs {
			rows[i] = []any{id, userID, trim(e.UID, 500), trim(e.Title, 500), trim(e.Location, 500), e.Start, e.End, e.AllDay}
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"calendar_events"},
			[]string{"source_id", "user_id", "uid", "title", "location", "starts_at", "ends_at", "all_day"}, pgx.CopyFromRows(rows)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE calendar_sources SET fetched_at = now(), error = '' WHERE id = $1`, id)
		return err
	})
}

func trim(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// Run refreshes sources older than the refresh interval, every few minutes.
func (s *Store) Run(ctx context.Context) {
	for {
		rows, err := s.db.Query(ctx, `SELECT id FROM calendar_sources WHERE fetched_at IS NULL OR fetched_at < $1 ORDER BY fetched_at NULLS FIRST LIMIT 50`,
			s.now().Add(-refresh))
		if err == nil {
			ids, cerr := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
			if cerr == nil {
				for _, id := range ids {
					if err := s.Refresh(ctx, id); err != nil && ctx.Err() == nil {
						s.log.Info("calendar refresh failed", "source", id, "err", err)
					}
				}
			}
		} else if ctx.Err() == nil {
			s.log.Error("calendar sources query failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
		}
	}
}
