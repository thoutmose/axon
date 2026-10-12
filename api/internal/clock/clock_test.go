package clock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// users answers the time zone query with tz, or err.
type users struct {
	tz  string
	err error
}

type row struct{ users }

func (r row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.tz
	return nil
}

func (u users) QueryRow(context.Context, string, ...any) pgx.Row { return row{u} }
func (users) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not used")
}

func (users) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not used")
}

func TestDay(t *testing.T) {
	at := time.Date(2026, 10, 10, 23, 30, 0, 0, time.UTC)
	for tz, want := range map[string]string{
		"UTC": "2026-10-10", "Pacific/Kiritimati": "2026-10-11", "Pacific/Pago_Pago": "2026-10-10",
		"Europe/Paris": "2026-10-11", "America/Los_Angeles": "2026-10-10",
	} {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			t.Fatal(err)
		}
		got := Day(at, loc)
		if got.Format(time.DateOnly) != want || got.Location() != time.UTC || got.Hour() != 0 {
			t.Errorf("Day in %s = %s, want %s at UTC midnight", tz, got, want)
		}
	}
}

func TestZone(t *testing.T) {
	ctx, id := context.Background(), uuid.New()
	loc, err := Zone(ctx, users{tz: "Europe/Paris"}, id)
	if err != nil || loc.String() != "Europe/Paris" {
		t.Errorf("Zone = %v, %v; want Europe/Paris", loc, err)
	}
	// A name this server does not know is UTC, not a failure.
	if loc, err := Zone(ctx, users{tz: "Mars/Olympus"}, id); err != nil || loc != time.UTC {
		t.Errorf("unknown zone = %v, %v; want UTC", loc, err)
	}
	// A failed read is reported, and UTC is still usable.
	boom := errors.New("boom")
	if loc, err := Zone(ctx, users{err: boom}, id); !errors.Is(err, boom) || loc != time.UTC {
		t.Errorf("failed read = %v, %v; want UTC and the error", loc, err)
	}
}

func TestToday(t *testing.T) {
	now := time.Date(2026, 10, 12, 23, 0, 0, 0, time.UTC)
	day, loc, err := Today(context.Background(), users{tz: "Asia/Tokyo"}, uuid.New(), now)
	if err != nil || day.Format(time.DateOnly) != "2026-10-13" || loc.String() != "Asia/Tokyo" {
		t.Errorf("Today = %s, %v, %v; want 2026-10-13 in Asia/Tokyo", day, loc, err)
	}
}
