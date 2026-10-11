package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/cards"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Teacher tools (EP-16). The Teacher reads only literature and permanent
// notes — the person's processed knowledge — never fleeting notes, journals,
// or anything else, and proposes cards the person approves.

// studyKinds are the note types the Teacher may read.
var studyKinds = []string{"literature", "permanent"}

type cardInput struct {
	Front  string `json:"front"`
	Back   string `json:"back"`
	Deck   string `json:"deck"`
	NoteID string `json:"note_id"`
}

type proposeCardsInput struct {
	Cards  []cardInput `json:"cards"`
	Reason string      `json:"reason"`
}

func init() {
	registry["list_study_notes"] = Tool{
		Name:        "list_study_notes",
		Description: "List the person's literature and permanent notes (id, title, path, type, tags), newest first. Fleeting notes are never listed.",
		Properties:  map[string]any{"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Limit int `json:"limit"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			if in.Limit <= 0 || in.Limit > 100 {
				in.Limit = 50
			}
			rows, err := q.Query(ctx, `SELECT n.id, e.title, n.path, n.kind FROM notes n JOIN entities e ON e.id = n.id
				WHERE n.user_id = $1 AND n.kind = ANY($2) ORDER BY n.updated_at DESC LIMIT $3`, userID, studyKinds, in.Limit)
			if err != nil {
				return nil, err
			}
			type row struct {
				ID    uuid.UUID `json:"id"`
				Title string    `json:"title"`
				Path  string    `json:"path"`
				Kind  string    `json:"kind"`
			}
			out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[row])
			if out == nil {
				out = []row{}
			}
			return out, err
		},
	}
	registry["read_study_note"] = Tool{
		Name:        "read_study_note",
		Description: "Read a literature or permanent note's Markdown. Any other kind of note is refused.",
		Properties:  map[string]any{"id": idField},
		Required:    []string{"id"},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				ID string `json:"id"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			id, err := parseID(in.ID, "id")
			if err != nil {
				return nil, err
			}
			n, err := notes.Get(ctx, q, userID, id)
			if err != nil {
				return nil, notFound(err, "note")
			}
			if n.Kind != "literature" && n.Kind != "permanent" {
				return nil, toolErrorf("only literature and permanent notes can be read; this one is %q", n.Kind)
			}
			return map[string]any{"id": n.ID, "title": n.Title, "path": n.Path, "kind": n.Kind, "content": n.Content}, nil
		},
	}
	registry["list_decks"] = Tool{
		Name:        "list_decks",
		Description: "List the person's flashcard decks with how many cards each holds and how many are due.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			return cards.Decks(ctx, q, userID, time.Now())
		},
	}
	registry["propose_cards"] = Tool{
		Name: "propose_cards",
		Description: "Propose flashcards for retrieval practice. Each card asks one thing, has a short answer, and names the note it comes from. " +
			"Nothing is created until the person approves.",
		Properties: map[string]any{
			"cards": map[string]any{"type": "array", "minItems": 1, "maxItems": 30, "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"front":   map[string]any{"type": "string", "description": "A question that makes the person recall, not recognise."},
					"back":    map[string]any{"type": "string", "description": "The answer, short; add an example when it helps."},
					"deck":    str,
					"note_id": map[string]any{"type": "string", "description": "The literature or permanent note it comes from."},
				},
				"required": []string{"front", "back", "note_id"},
			}},
			"reason": reason,
		},
		Required: []string{"cards", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in proposeCardsInput
			_ = json.Unmarshal(raw, &in)
			var fronts []string
			for i, c := range in.Cards {
				if i == 3 {
					fronts = append(fronts, "…")
					break
				}
				fronts = append(fronts, "“"+truncate(c.Front, 80)+"”")
			}
			return fmt.Sprintf("Add %d flashcards: %s", len(in.Cards), strings.Join(fronts, ", ")), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in proposeCardsInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if len(in.Cards) == 0 || len(in.Cards) > 30 {
				return toolErrorf("propose 1 to 30 cards")
			}
			for _, c := range in.Cards {
				nid, err := parseID(c.NoteID, "note_id")
				if err != nil {
					return err
				}
				n, err := notes.Get(ctx, tx, userID, nid)
				if err != nil {
					return notFound(err, "note")
				}
				if n.Kind != "literature" && n.Kind != "permanent" {
					return toolErrorf("cards come from literature or permanent notes only")
				}
				deck := strings.TrimSpace(c.Deck)
				if deck == "" {
					deck = n.Title
				}
				f := cards.Fields{Front: &c.Front, Back: &c.Back, Deck: &deck}
				if _, err := cards.Create(ctx, tx, userID, actor, f, &nid, time.Now()); err != nil {
					var v cards.ValidationError
					if errors.As(err, &v) {
						return toolError{v.Msg}
					}
					return err
				}
			}
			return nil
		},
	}
}
