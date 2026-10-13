package health

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Set is one line of a workout: an exercise with reps and weight, or a
// distance and a time.
type Set struct {
	Exercise   string   `json:"exercise"`
	Reps       *int     `json:"reps"`
	WeightKg   *float64 `json:"weight_kg"`
	DistanceKm *float64 `json:"distance_km"`
	Seconds    *int     `json:"seconds"`
}

// Workout is one training session.
type Workout struct {
	ID      uuid.UUID `json:"id"`
	DoneOn  string    `json:"done_on"`
	Kind    string    `json:"kind"`
	Title   string    `json:"title"`
	Minutes *int      `json:"minutes"`
	Effort  *int      `json:"effort"`
	Notes   string    `json:"notes"`
	Sets    []Set     `json:"sets"`
}

// WorkoutInput writes a whole workout; sets replace the previous ones.
type WorkoutInput struct {
	DoneOn  string `json:"done_on"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Minutes *int   `json:"minutes"`
	Effort  *int   `json:"effort"`
	Notes   string `json:"notes"`
	Sets    []Set  `json:"sets"`
}

func (in *WorkoutInput) validate() error {
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case !validDate(in.DoneOn):
		return ValidationError{"The date is YYYY-MM-DD."}
	case !slices.Contains(Kinds, in.Kind):
		return ValidationError{"Kind is strength, cardio, mobility, sport, or other."}
	case in.Title == "" || len([]rune(in.Title)) > 200:
		return ValidationError{"Give the workout a title (up to 200 characters)."}
	case in.Minutes != nil && (*in.Minutes < 1 || *in.Minutes > 1440):
		return ValidationError{"Minutes are 1 to 1,440."}
	case in.Effort != nil && (*in.Effort < 1 || *in.Effort > 10):
		return ValidationError{"Effort is 1 (very easy) to 10 (maximal)."}
	case len([]rune(in.Notes)) > 4000:
		return ValidationError{"Notes are up to 4,000 characters."}
	case len(in.Sets) > 200:
		return ValidationError{"Up to 200 sets."}
	}
	for i := range in.Sets {
		s := &in.Sets[i]
		s.Exercise = strings.TrimSpace(s.Exercise)
		switch {
		case s.Exercise == "" || len([]rune(s.Exercise)) > 120:
			return ValidationError{"Each set names its exercise (up to 120 characters)."}
		case s.Reps != nil && (*s.Reps < 1 || *s.Reps > 1000):
			return ValidationError{"Reps are 1 to 1,000."}
		case s.WeightKg != nil && (*s.WeightKg < 0 || *s.WeightKg > 1000):
			return ValidationError{"Weight is 0 to 1,000 kg."}
		case s.DistanceKm != nil && (*s.DistanceKm <= 0 || *s.DistanceKm > 1000):
			return ValidationError{"Distance is up to 1,000 km."}
		case s.Seconds != nil && (*s.Seconds < 1 || *s.Seconds > 86400):
			return ValidationError{"Time is up to 24 hours."}
		}
	}
	return nil
}

func validDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

// Workouts returns the person's workouts in [from, to], newest first.
func Workouts(ctx context.Context, q db.Querier, userID uuid.UUID, from, to string) ([]Workout, error) {
	rows, err := q.Query(ctx, `SELECT w.id, to_char(w.done_on, 'YYYY-MM-DD'), w.kind, w.title, w.minutes, w.effort, w.notes,
		coalesce((SELECT json_agg(json_build_object('exercise', s.exercise, 'reps', s.reps, 'weight_kg', s.weight_kg,
			'distance_km', s.distance_km, 'seconds', s.seconds) ORDER BY s.position) FROM workout_sets s WHERE s.workout_id = w.id), '[]')
		FROM workouts w WHERE w.user_id = $1 AND w.done_on BETWEEN $2 AND $3
		ORDER BY w.done_on DESC, w.created_at DESC LIMIT 500`, userID, from, to)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Workout, error) {
		var w Workout
		var effort *int16
		err := r.Scan(&w.ID, &w.DoneOn, &w.Kind, &w.Title, &w.Minutes, &effort, &w.Notes, &w.Sets)
		if effort != nil {
			e := int(*effort)
			w.Effort = &e
		}
		return w, err
	})
	if out == nil {
		out = []Workout{}
	}
	return out, err
}

// GetWorkout returns one of the person's workouts.
func GetWorkout(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Workout, error) {
	var day string
	if err := q.QueryRow(ctx, `SELECT to_char(done_on, 'YYYY-MM-DD') FROM workouts WHERE id = $1 AND user_id = $2`, id, userID).Scan(&day); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Workout{}, ErrNotFound
		}
		return Workout{}, err
	}
	all, err := Workouts(ctx, q, userID, day, day)
	if err != nil {
		return Workout{}, err
	}
	for _, w := range all {
		if w.ID == id {
			return w, nil
		}
	}
	return Workout{}, ErrNotFound
}

// SaveWorkout creates a workout (id nil) or replaces one.
func SaveWorkout(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id *uuid.UUID, in WorkoutInput) (Workout, error) {
	if err := in.validate(); err != nil {
		return Workout{}, err
	}
	if id == nil {
		newID := uuid.Must(uuid.NewV7())
		id = &newID
		if _, err := tx.Exec(ctx, `INSERT INTO workouts (id, user_id, done_on, kind, title, minutes, effort, notes) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			id, userID, in.DoneOn, in.Kind, in.Title, in.Minutes, in.Effort, in.Notes); err != nil {
			return Workout{}, err
		}
	} else {
		tag, err := tx.Exec(ctx, `UPDATE workouts SET done_on = $3, kind = $4, title = $5, minutes = $6, effort = $7, notes = $8 WHERE id = $1 AND user_id = $2`,
			id, userID, in.DoneOn, in.Kind, in.Title, in.Minutes, in.Effort, in.Notes)
		if err != nil {
			return Workout{}, err
		}
		if tag.RowsAffected() == 0 {
			return Workout{}, ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM workout_sets WHERE workout_id = $1`, id); err != nil {
			return Workout{}, err
		}
	}
	for i, s := range in.Sets {
		if _, err := tx.Exec(ctx, `INSERT INTO workout_sets (workout_id, position, exercise, reps, weight_kg, distance_km, seconds) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, i, s.Exercise, s.Reps, s.WeightKg, s.DistanceKm, s.Seconds); err != nil {
			return Workout{}, err
		}
	}
	w, err := GetWorkout(ctx, tx, userID, *id)
	if err != nil {
		return w, err
	}
	return w, changed(ctx, tx, userID, "health:workout:"+id.String())
}

// DeleteWorkout removes one of the person's workouts.
func DeleteWorkout(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM workouts WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && tag.RowsAffected() == 0 {
		err = ErrNotFound
	}
	if err != nil {
		return err
	}
	return changed(ctx, tx, userID, "health:workout:"+id.String())
}

// Training sums up recent training.
type Training struct {
	Sessions int `json:"sessions"`
	Minutes  int `json:"minutes"`
	Days     int `json:"days"` // distinct days with a workout
}

// TrainingSince sums workouts from a date on.
func TrainingSince(ctx context.Context, q db.Querier, userID uuid.UUID, from string) (Training, error) {
	var t Training
	err := q.QueryRow(ctx, `SELECT count(*), coalesce(sum(minutes), 0), count(DISTINCT done_on) FROM workouts WHERE user_id = $1 AND done_on >= $2`,
		userID, from).Scan(&t.Sessions, &t.Minutes, &t.Days)
	return t, err
}

// Export returns the person's health data for their export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+entryColumns+` FROM food_entries WHERE user_id = $1 ORDER BY eaten_on, created_at`, userID)
	if err != nil {
		return nil, err
	}
	food, err := pgx.CollectRows(rows, scanEntry)
	if err != nil {
		return nil, err
	}
	workouts, err := Workouts(ctx, q, userID, "0001-01-01", "9999-12-31")
	if err != nil {
		return nil, err
	}
	if food == nil {
		food = []Entry{}
	}
	profile, err := GetProfile(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"food_entries": food, "workouts": workouts, "profile": profile}, nil
}
