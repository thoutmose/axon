package watch

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Transcribing videos (EP-123): after a fetch, a few videos at a time get
// their words, which join the item's text and its analysis: its concepts,
// topics, labels, and the graph.

// PendingTranscripts is how many videos each worker step transcribes.
const PendingTranscripts = 3

// Transcribe reads a video item's transcript and analyses the item again
// with it. It records the attempt whatever happens, so a video with no
// words is not asked for again; ErrNoTranscript says there were none.
func Transcribe(ctx context.Context, pool *pgxpool.Pool, userID, itemID uuid.UUID, w *Whisper) (Item, error) {
	it, err := GetItem(ctx, pool, userID, itemID)
	if err != nil {
		return it, err
	}
	id := VideoID(it.URL)
	if id == "" && it.Audio == "" {
		return it, ValidationError{"Only videos and podcast episodes have words to transcribe."}
	}
	if it.TranscriptSource != "" {
		return it, nil
	}
	tctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	var tr Transcript
	var terr error
	if id != "" {
		tr, terr = VideoTranscript(tctx, id, w)
	} else {
		tr, terr = episodeTranscript(tctx, pool, userID, it, w)
	}
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if terr != nil {
			_, err := tx.Exec(ctx, `UPDATE watch_items SET transcript_tried_at = now() WHERE id = $1`, itemID)
			return err
		}
		topics, err := ListTopics(ctx, tx, userID)
		if err != nil {
			return err
		}
		content := *it.Content + "\n\n" + transcriptMarkdown(tr)
		a := Analyse(Entry{URL: it.URL, Title: it.Title, Summary: it.Summary, Markdown: content, Categories: it.Categories}, topics)
		_, err = tx.Exec(ctx, `UPDATE watch_items SET content = $2, lang = $3, words = $4, simhash = $5, terms = $6, keywords = $7,
			labels = $8, topic_ids = $9::uuid[], score = $10, outlier = '', transcript_source = $11, transcript_lang = $12,
			transcript_tried_at = now() WHERE id = $1`,
			itemID, cut(CleanLinks(content), 200000), a.Lang, a.Words, int64(a.SimHash), //nolint:gosec // the same 64 bits
			nonNil(a.Terms), nonNil(a.Keywords), nonNil(uniq(append(a.Labels, it.Labels...), 16)), nonNil(a.Topics), a.Score,
			tr.Source, cut(tr.Lang, 10))
		return err
	})
	if err != nil {
		return it, err
	}
	if terr != nil {
		return it, terr
	}
	return GetItem(ctx, pool, userID, itemID)
}

// maxEpisode bounds a podcast episode sent to Whisper.
const maxEpisode = 250 << 20

// episodeTranscript sends a podcast episode's audio to Whisper: from its
// origin (the person's Audiobookshelf), else its public file.
func episodeTranscript(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, it Item, w *Whisper) (Transcript, error) {
	if w == nil {
		return Transcript{}, ErrNoTranscript
	}
	var origin, guid string
	if err := pool.QueryRow(ctx, `SELECT coalesce(s.origin, ''), i.guid FROM watch_items i LEFT JOIN watch_sources s ON s.id = i.source_id
		WHERE i.id = $1`, it.ID).Scan(&origin, &guid); err != nil {
		return Transcript{}, err
	}
	var audio []byte
	name := "episode.mp3"
	var err error
	if o, ok := origins[origin]; ok && o.Audio != nil {
		audio, name, err = o.Audio(ctx, pool, userID, guid)
	} else {
		audio, err = download(ctx, it.Audio, maxEpisode)
	}
	if err != nil {
		return Transcript{}, err
	}
	return w.Transcribe(ctx, audio, name)
}

// TranscribePending transcribes the newest videos not tried yet, and,
// when a Whisper server is set up, podcast episodes.
func TranscribePending(ctx context.Context, pool *pgxpool.Pool, w *Whisper, n int) error {
	rows, err := pool.Query(ctx, `SELECT i.id, i.user_id FROM watch_items i JOIN users u ON u.id = i.user_id
		WHERE i.transcript_tried_at IS NULL AND NOT u.demo
		  AND (i.url LIKE 'https://youtube.com/watch?%' OR ($2 AND i.audio <> ''))
		ORDER BY i.published_at DESC LIMIT $1`, n, w != nil)
	if err != nil {
		return err
	}
	type video struct{ id, user uuid.UUID }
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[video])
	if err != nil {
		return err
	}
	for _, v := range list {
		if _, err := Transcribe(ctx, pool, v.user, v.id, w); err != nil && !errors.Is(err, ErrNoTranscript) && ctx.Err() == nil {
			// Recorded as tried; YouTube or the Whisper server may be down: the next video still gets its turn.
			continue
		}
	}
	return ctx.Err()
}

// TranscribeUser reads the words of one person's videos not tried yet: the
// shared demo's, after it is reset, from YouTube's captions only.
func TranscribeUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, w *Whisper, n int) (int, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM watch_items WHERE user_id = $1 AND transcript_tried_at IS NULL
		AND url LIKE 'https://youtube.com/watch?%' ORDER BY published_at DESC LIMIT $2`, userID, n)
	if err != nil {
		return 0, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return 0, err
	}
	done := 0
	for _, id := range ids {
		if it, err := Transcribe(ctx, pool, userID, id, w); err == nil && it.TranscriptSource != "" {
			done++
		} else if ctx.Err() != nil {
			return done, ctx.Err()
		}
	}
	return done, nil
}
