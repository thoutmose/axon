package watch

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Origins (EP-124): a source that is not an RSS feed — a podcast on the
// person's Audiobookshelf — brings its episodes, and their audio for
// Whisper, through a package that registers itself here, so the tech watch
// does not depend on it.

// Origin reads a source's entries, and an item's audio.
type Origin struct {
	Fetch func(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, src Source) (Feed, error)
	Audio func(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, guid string) (audio []byte, name string, err error)
}

var origins = map[string]Origin{}

// RegisterOrigin makes sources of that origin read through o.
func RegisterOrigin(name string, o Origin) { origins[name] = o }
