package cards

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/fsrs"
)

// Calibration compares what the scheduler predicted with what happened: at
// each review of a card in long-term memory, FSRS predicts the chance of
// recalling it; a review counts as remembered unless rated Again. When the
// two agree, the schedule can be trusted to hold the target retention.
type Calibration struct {
	Reviews    int      `json:"reviews"`
	Predicted  float64  `json:"predicted"`  // mean predicted chance of recall
	Remembered float64  `json:"remembered"` // share remembered
	Target     float64  `json:"target"`     // the retention the person chose
	Buckets    []Bucket `json:"buckets"`
}

// Bucket is the reviews whose prediction fell in [From, To).
type Bucket struct {
	From       float64 `json:"from"`
	To         float64 `json:"to"`
	Reviews    int     `json:"reviews"`
	Predicted  float64 `json:"predicted"`
	Remembered float64 `json:"remembered"`
}

// Finer where predictions usually fall (most reviews are at 80–97%).
var bucketEdges = []float64{0, 0.5, 0.7, 0.8, 0.9, 0.95, 1}

type reviewed struct {
	predicted  float64
	remembered bool
}

func calibrate(rs []reviewed, target float64) Calibration {
	c := Calibration{Target: target, Buckets: []Bucket{}}
	bs := make([]Bucket, len(bucketEdges)-1)
	for i := range bs {
		bs[i].From, bs[i].To = bucketEdges[i], bucketEdges[i+1]
	}
	for _, r := range rs {
		i := len(bs) - 1 // a prediction of exactly 1 goes in the last bucket
		for j := range bs {
			if r.predicted < bs[j].To {
				i = j
				break
			}
		}
		bs[i].Reviews++
		bs[i].Predicted += r.predicted
		c.Reviews++
		c.Predicted += r.predicted
		if r.remembered {
			bs[i].Remembered++
			c.Remembered++
		}
	}
	if c.Reviews > 0 {
		c.Predicted /= float64(c.Reviews)
		c.Remembered /= float64(c.Reviews)
	}
	for _, b := range bs {
		if b.Reviews > 0 {
			b.Predicted /= float64(b.Reviews)
			b.Remembered /= float64(b.Reviews)
			c.Buckets = append(c.Buckets, b)
		}
	}
	return c
}

// GetCalibration reads the person's reviews of the last year that carry a
// prediction (those of cards in long-term memory, since EP-77).
func GetCalibration(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) (Calibration, error) {
	p, err := LoadPrefs(ctx, q, userID)
	if err != nil {
		return Calibration{}, err
	}
	rows, err := q.Query(ctx, `SELECT r.predicted, r.rating > 1 FROM card_reviews r JOIN cards c ON c.id = r.card_id
		JOIN entities e ON e.id = c.id
		WHERE e.user_id = $1 AND r.predicted IS NOT NULL AND r.reviewed_at > $2 ORDER BY r.reviewed_at DESC LIMIT 50000`,
		userID, now.AddDate(-1, 0, 0))
	if err != nil {
		return Calibration{}, err
	}
	rs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (reviewed, error) {
		var x reviewed
		err := r.Scan(&x.predicted, &x.remembered)
		return x, err
	})
	if err != nil {
		return Calibration{}, err
	}
	return calibrate(rs, p.Retention), nil
}

// ReplayPredictions gives reviews from before EP-77 the prediction the
// scheduler would have made, by replaying each card's history: stability
// follows from the ratings and the time between reviews. Only cards whose
// whole history is here are replayed (an imported card's earlier reviews
// are not), and a card whose history does not fit the scheduler's states is
// left alone. It fills only empty predictions, so it can run again.
func ReplayPredictions(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `SELECT c.id, e.user_id FROM cards c JOIN entities e ON e.id = c.id
		WHERE c.reps = (SELECT count(*) FROM card_reviews r WHERE r.card_id = c.id)
		  AND EXISTS (SELECT 1 FROM card_reviews r WHERE r.card_id = c.id AND r.predicted IS NULL AND r.state = $1)
		LIMIT 2000`, int16(fsrs.Review))
	if err != nil {
		return err
	}
	type target struct{ Card, User uuid.UUID }
	targets, err := pgx.CollectRows(rows, pgx.RowToStructByPos[target])
	if err != nil {
		return err
	}
	schedulers := map[uuid.UUID]fsrs.Scheduler{}
	for _, t := range targets {
		s, ok := schedulers[t.User]
		if !ok {
			p, err := LoadPrefs(ctx, pool, t.User)
			if err != nil {
				return err
			}
			s = p.scheduler()
			schedulers[t.User] = s
		}
		if err := replay(ctx, pool, t.Card, s); err != nil {
			return err
		}
	}
	return nil
}

func replay(ctx context.Context, q db.Querier, cardID uuid.UUID, s fsrs.Scheduler) error {
	rows, err := q.Query(ctx, `SELECT id, rating, reviewed_at, state, predicted IS NOT NULL FROM card_reviews
		WHERE card_id = $1 ORDER BY reviewed_at, id`, cardID)
	if err != nil {
		return err
	}
	type review struct {
		ID     uuid.UUID
		Rating int16
		At     time.Time
		State  int16
		Known  bool
	}
	history, err := pgx.CollectRows(rows, pgx.RowToStructByPos[review])
	if err != nil || len(history) == 0 {
		return err
	}
	c := fsrs.New(history[0].At)
	var ids []uuid.UUID
	var predicted []float64
	for _, r := range history {
		if c.State != fsrs.State(r.State) {
			return nil // the history does not fit the scheduler: leave it
		}
		if c.State == fsrs.Review && !r.Known {
			ids = append(ids, r.ID)
			predicted = append(predicted, s.Retrievability(c, r.At))
		}
		c = s.Review(c, fsrs.Rating(r.Rating), r.At)
	}
	if len(ids) == 0 {
		return nil
	}
	_, err = q.Exec(ctx, `UPDATE card_reviews SET predicted = x.p FROM unnest($1::uuid[], $2::float8[]) AS x(id, p)
		WHERE card_reviews.id = x.id AND card_reviews.predicted IS NULL`, ids, predicted)
	return err
}
