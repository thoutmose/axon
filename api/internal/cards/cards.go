// Package cards holds flashcards and their reviews (EP-16). Scheduling is
// FSRS-5 (package fsrs); a card is an entity, so it links to the note,
// skill, or roadmap it was made from.
package cards

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/fsrs"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Type is the entity type; DerivedFrom links a card to its source.
const (
	Type        = "card"
	DerivedFrom = "derived_from"
)

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "cards", Export: export})
	graph.RegisterRelation(graph.Relation{Name: DerivedFrom, Sources: []string{Type}})
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

// Card is a flashcard with its schedule.
type Card struct {
	ID         uuid.UUID  `json:"id"`
	Deck       string     `json:"deck"`
	Front      string     `json:"front"`
	Back       string     `json:"back"`
	HasImage   bool       `json:"has_image"`
	State      fsrs.State `json:"state"` // 1 learning, 2 review, 3 relearning
	Due        time.Time  `json:"due"`
	LastReview *time.Time `json:"last_review"`
	Stability  *float64   `json:"stability"`
	Difficulty *float64   `json:"difficulty"`
	Reps       int        `json:"reps"`
	Lapses     int        `json:"lapses"`
	Suspended  bool       `json:"suspended"`
	UpdatedAt  time.Time  `json:"updated_at"`
	// What makes it stick (EP-52): why it is true, a concrete example, a
	// vivid image or mnemonic, and a place in a memory palace.
	Elaboration string     `json:"elaboration"`
	Example     string     `json:"example"`
	Mnemonic    string     `json:"mnemonic"`
	ImageType   *string    `json:"image_type"` // image/png or image/svg+xml; null without a picture
	PalaceID    *uuid.UUID `json:"palace_id"`
	Locus       *int       `json:"locus"`
	// Leech: forgotten so often that reviewing it again will not help;
	// rewrite it, add an example or an image, or split it.
	Leech bool `json:"leech"`
	step  *int
}

// LeechLapses is how many lapses make a card a leech (Anki uses 8; a
// person working on their own benefits from hearing it earlier).
const LeechLapses = 6

const columns = `c.id, c.deck, c.front, c.back, c.image IS NOT NULL, c.state, c.due, c.last_review, c.stability, c.difficulty,
	c.reps, c.lapses, c.suspended, c.updated_at, c.step, c.elaboration, c.example, c.mnemonic,
	CASE WHEN c.image IS NULL THEN NULL ELSE c.image_type END, c.palace_id, c.locus`

func scan(row pgx.Row) (Card, error) {
	var c Card
	var step, locus *int16
	err := row.Scan(&c.ID, &c.Deck, &c.Front, &c.Back, &c.HasImage, &c.State, &c.Due, &c.LastReview, &c.Stability, &c.Difficulty,
		&c.Reps, &c.Lapses, &c.Suspended, &c.UpdatedAt, &step, &c.Elaboration, &c.Example, &c.Mnemonic, &c.ImageType, &c.PalaceID, &locus)
	if step != nil {
		s := int(*step)
		c.step = &s
	}
	if locus != nil {
		l := int(*locus)
		c.Locus = &l
	}
	c.Leech = c.Lapses >= LeechLapses
	return c, err
}

func (c Card) schedule() fsrs.Card {
	step := c.step
	if step == nil && c.State != fsrs.Review {
		zero := 0 // learning and relearning cards always have a step
		step = &zero
	}
	return fsrs.Card{State: c.State, Step: step, Stability: c.Stability, Difficulty: c.Difficulty, Due: c.Due, LastReview: c.LastReview}
}

// Get returns one of the person's cards.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Card, error) {
	c, err := scan(q.QueryRow(ctx, `SELECT `+columns+` FROM cards c JOIN entities e ON e.id = c.id WHERE c.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, graph.ErrNotFound
	}
	return c, err
}

// List returns cards in a deck (all decks when empty), newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID, deck string) ([]Card, error) {
	rows, err := q.Query(ctx, `SELECT `+columns+` FROM cards c JOIN entities e ON e.id = c.id
		WHERE e.user_id = $1 AND ($2 = '' OR c.deck = $2) ORDER BY e.created_at DESC LIMIT 1000`, userID, deck)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Card, error) { return scan(r) })
	if out == nil {
		out = []Card{}
	}
	return out, err
}

// Deck summarises one deck.
type Deck struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
	Due   int    `json:"due"` // what a review of the deck would offer now
	New   int    `json:"new"` // never reviewed
}

// Decks lists the person's decks. A deck's due count is what a review of
// it would offer now: known cards due and new cards within today's allowance.
func Decks(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) ([]Deck, error) {
	p, err := LoadPrefs(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	introduced, err := newIntroduced(ctx, q, userID, now)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT c.deck, count(*), count(*) FILTER (WHERE NOT c.suspended AND c.reps > 0 AND c.due <= $2),
		count(*) FILTER (WHERE c.reps = 0), count(*) FILTER (WHERE c.reps = 0 AND NOT c.suspended) FROM cards c JOIN entities e ON e.id = c.id
		WHERE e.user_id = $1 GROUP BY c.deck ORDER BY c.deck`, userID, now)
	if err != nil {
		return nil, err
	}
	allowance := max(0, p.NewPerDay-introduced)
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Deck, error) {
		var d Deck
		var fresh int
		err := r.Scan(&d.Name, &d.Total, &d.Due, &d.New, &fresh)
		d.Due += min(fresh, allowance)
		return d, err
	})
	if out == nil {
		out = []Deck{}
	}
	return out, err
}

// Fields are the editable parts of a card. Palace is a palace id, or ""
// to take the card out of its palace.
type Fields struct {
	Deck, Front, Back              *string
	Elaboration, Example, Mnemonic *string
	Suspended                      *bool
	Palace                         *string
	Locus                          *int
}

func (f *Fields) validate() error {
	for _, p := range []*string{f.Deck, f.Front, f.Back, f.Elaboration, f.Example, f.Mnemonic, f.Palace} {
		if p != nil {
			*p = strings.TrimSpace(*p)
		}
	}
	for _, p := range []*string{f.Elaboration, f.Example, f.Mnemonic} {
		if p != nil && len(*p) > 2000 {
			return ValidationError{"The why, the example, and the memory image are up to 2,000 characters each."}
		}
	}
	switch {
	case f.Deck != nil && (*f.Deck == "" || len([]rune(*f.Deck)) > 200):
		return ValidationError{"A deck name is 1 to 200 characters."}
	case f.Front != nil && (*f.Front == "" || len(*f.Front) > 10000):
		return ValidationError{"The question is 1 to 10,000 characters."}
	case f.Back != nil && len(*f.Back) > 10000:
		return ValidationError{"The answer is up to 10,000 characters."}
	}
	return nil
}

func title(front string) string {
	if r := []rune(front); len(r) > 200 {
		return string(r[:200])
	}
	return front
}

// Create adds a new card, due now, linked to its source when given.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields, source *uuid.UUID, now time.Time) (Card, error) {
	if f.Front == nil {
		return Card{}, ValidationError{"A card needs a question."}
	}
	if err := f.validate(); err != nil {
		return Card{}, err
	}
	deck, back := "Default", ""
	if f.Deck != nil {
		deck = *f.Deck
	}
	if f.Back != nil {
		back = *f.Back
	}
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, title(*f.Front), back)
	if err != nil {
		return Card{}, err
	}
	suspended := f.Suspended != nil && *f.Suspended
	if _, err := tx.Exec(ctx, `INSERT INTO cards (id, deck, front, back, due, step, elaboration, example, mnemonic, suspended) VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8, $9)`,
		id, deck, *f.Front, back, now, deref(f.Elaboration), deref(f.Example), deref(f.Mnemonic), suspended); err != nil {
		return Card{}, err
	}
	if f.Palace != nil && *f.Palace != "" {
		if err := place(ctx, tx, userID, id, *f.Palace, f.Locus); err != nil {
			return Card{}, err
		}
	}
	if source != nil {
		if _, err := graph.CreateLink(ctx, tx, userID, actor, id, DerivedFrom, *source); err != nil {
			if errors.Is(err, graph.ErrNotFound) {
				return Card{}, ValidationError{"The source item does not exist."}
			}
			return Card{}, err
		}
	}
	return Get(ctx, tx, userID, id)
}

// Update changes a card's text, deck, or suspension.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, f Fields) (Card, error) {
	if err := f.validate(); err != nil {
		return Card{}, err
	}
	c, err := Get(ctx, tx, userID, id)
	if err != nil {
		return c, err
	}
	var changed []string
	set := func(name string, p *string, dst *string) {
		if p != nil && *p != *dst {
			*dst, changed = *p, append(changed, name)
		}
	}
	set("deck", f.Deck, &c.Deck)
	set("front", f.Front, &c.Front)
	set("back", f.Back, &c.Back)
	set("elaboration", f.Elaboration, &c.Elaboration)
	set("example", f.Example, &c.Example)
	set("mnemonic", f.Mnemonic, &c.Mnemonic)
	if f.Palace != nil || f.Locus != nil {
		palace := ""
		if c.PalaceID != nil {
			palace = c.PalaceID.String()
		}
		if f.Palace != nil {
			palace = *f.Palace
		}
		locus := c.Locus
		if f.Locus != nil {
			locus = f.Locus
		}
		current := ""
		if c.PalaceID != nil {
			current = c.PalaceID.String()
		}
		samePlace := palace == "" || c.Locus != nil && locus != nil && *c.Locus == *locus
		if palace != current || !samePlace {
			if err := place(ctx, tx, userID, id, palace, locus); err != nil {
				return c, err
			}
			changed = append(changed, "palace")
		}
	}
	if f.Suspended != nil && *f.Suspended != c.Suspended {
		c.Suspended, changed = *f.Suspended, append(changed, "suspended")
	}
	if len(changed) == 0 {
		return c, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, title(c.Front), c.Back, changed); err != nil {
		return c, err
	}
	if _, err := tx.Exec(ctx, `UPDATE cards SET deck = $2, front = $3, back = $4, suspended = $5, elaboration = $6, example = $7, mnemonic = $8,
		updated_at = clock_timestamp() WHERE id = $1`, id, c.Deck, c.Front, c.Back, c.Suspended, c.Elaboration, c.Example, c.Mnemonic); err != nil {
		return c, err
	}
	return Get(ctx, tx, userID, id)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// place puts a card at a locus of one of the person's palaces, or with
// palace "" takes it out.
func place(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, palace string, locus *int) error {
	if palace == "" {
		_, err := tx.Exec(ctx, `UPDATE cards SET palace_id = NULL, locus = NULL WHERE id = $1`, id)
		return err
	}
	pid, err := uuid.Parse(palace)
	if err != nil {
		return ValidationError{"No such memory palace."}
	}
	var loci int
	err = tx.QueryRow(ctx, `SELECT cardinality(loci) FROM memory_palaces WHERE id = $1 AND user_id = $2`, pid, userID).Scan(&loci)
	if errors.Is(err, pgx.ErrNoRows) {
		return ValidationError{"No such memory palace."}
	}
	if err != nil {
		return err
	}
	if locus == nil || *locus < 0 || *locus >= loci {
		return ValidationError{"Choose one of the palace's places (locus 0 to the number of places minus one)."}
	}
	_, err = tx.Exec(ctx, `UPDATE cards SET palace_id = $2, locus = $3 WHERE id = $1`, id, pid, *locus)
	return err
}

// Preview is when a card would next be due for each rating.
type Preview struct {
	Again time.Time `json:"again"`
	Hard  time.Time `json:"hard"`
	Good  time.Time `json:"good"`
	Easy  time.Time `json:"easy"`
}

// Next previews the four possible outcomes of reviewing c now.
func Next(c Card, now time.Time, p Prefs) Preview {
	s := p.scheduler()
	sc := c.schedule()
	return Preview{s.Review(sc, fsrs.Again, now).Due, s.Review(sc, fsrs.Hard, now).Due, s.Review(sc, fsrs.Good, now).Due, s.Review(sc, fsrs.Easy, now).Due}
}

// Review records a review and reschedules the card.
func Review(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, rating fsrs.Rating, duration *int, now time.Time) (Card, error) {
	p, err := LoadPrefs(ctx, tx, userID)
	if err != nil {
		return Card{}, err
	}
	if rating < fsrs.Again || rating > fsrs.Easy {
		return Card{}, ValidationError{"Rate 1 (again), 2 (hard), 3 (good), or 4 (easy)."}
	}
	if duration != nil && (*duration < 0 || *duration > 3_600_000) {
		return Card{}, ValidationError{"The duration is 0 to 3,600,000 milliseconds."}
	}
	c, err := Get(ctx, tx, userID, id)
	if err != nil {
		return c, err
	}
	s := p.scheduler()
	next := s.Review(c.schedule(), rating, now)
	lapse := 0
	var predicted *float64 // only cards in long-term memory have a meaningful prediction
	if c.State == fsrs.Review {
		r := s.Retrievability(c.schedule(), now)
		predicted = &r
		if rating == fsrs.Again {
			lapse = 1
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO card_reviews (id, card_id, rating, reviewed_at, state, duration_ms, predicted) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.Must(uuid.NewV7()), id, rating, now, c.State, duration, predicted); err != nil {
		return c, err
	}
	if _, err := tx.Exec(ctx, `UPDATE cards SET state = $2, step = $3, stability = $4, difficulty = $5, due = $6, last_review = $7,
		reps = reps + 1, lapses = lapses + $8, updated_at = clock_timestamp() WHERE id = $1`,
		id, next.State, next.Step, next.Stability, next.Difficulty, next.Due, next.LastReview, lapse); err != nil {
		return c, err
	}
	return Get(ctx, tx, userID, id)
}

// SetImage stores (or with nil, removes) a card's picture: a PNG, or a
// drawing as sanitised SVG (contentType image/svg+xml).
func SetImage(ctx context.Context, q db.Querier, userID, id uuid.UUID, data []byte, contentType ...string) error {
	ct := "image/png"
	if len(contentType) > 0 {
		ct = contentType[0]
	}
	tag, err := q.Exec(ctx, `UPDATE cards c SET image = $3, image_type = $4, updated_at = clock_timestamp() FROM entities e
		WHERE c.id = $1 AND e.id = c.id AND e.user_id = $2`, id, userID, data, ct)
	if err == nil && tag.RowsAffected() == 0 {
		return graph.ErrNotFound
	}
	return err
}

// Image returns a card's picture and its type, or nil.
func Image(ctx context.Context, q db.Querier, userID, id uuid.UUID) ([]byte, string, error) {
	var img []byte
	var ct string
	err := q.QueryRow(ctx, `SELECT c.image, c.image_type FROM cards c JOIN entities e ON e.id = c.id WHERE c.id = $1 AND e.user_id = $2`, id, userID).Scan(&img, &ct)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", graph.ErrNotFound
	}
	return img, ct, err
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return List(ctx, q, userID, "")
}
