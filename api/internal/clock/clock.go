// Package clock answers what day it is for a person: their time zone
// (users.timezone) and their calendar date, stored as a UTC midnight like
// every date column.
package clock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/db"
)

// Zone is the person's time zone. A name this server does not know is UTC;
// when the read fails, the error is returned with UTC, so callers that can
// do without may ignore it.
func Zone(ctx context.Context, q db.Querier, userID uuid.UUID) (*time.Location, error) {
	var tz string
	if err := q.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&tz); err != nil {
		return time.UTC, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC, nil
	}
	return loc, nil
}

// Day is t's calendar date in loc, as a UTC midnight.
func Day(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Today is the person's date at now, and their time zone.
func Today(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (time.Time, *time.Location, error) {
	loc, err := Zone(ctx, q, userID)
	if err != nil {
		return time.Time{}, loc, err
	}
	return Day(now, loc), loc, nil
}
