package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/media"
)

// Media tools (EP-17). The agent sees only the person's own list; catalogue
// data is not sent to the model. Adding or changing an item is a proposal.

type mediaInput struct {
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Year     int      `json:"year"`
	Creators string   `json:"creators"`
	Genres   []string `json:"genres"`
	Reason   string   `json:"reason"`
}

// mediaFields keeps what the agent may set: the item, not a rating or a review.
func mediaFields(raw json.RawMessage) (media.Fields, error) {
	var in mediaInput
	if err := decode(raw, &in); err != nil {
		return media.Fields{}, err
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Reason) == "" {
		return media.Fields{}, toolErrorf("title and reason are required")
	}
	want := "want"
	f := media.Fields{Title: &in.Title, Kind: &in.Kind, Status: &want, Creators: &in.Creators}
	if in.Year != 0 {
		f.Year = &in.Year
	}
	if in.Genres != nil {
		f.Genres = &in.Genres
	}
	return f, nil
}

func init() {
	registry["list_media"] = Tool{
		Name:        "list_media",
		Description: "List the person's films, series, and books: status (want, doing, done, dropped), rating out of 10, creators, genres, and progress.",
		Properties: map[string]any{
			"kind":   map[string]any{"type": "string", "enum": media.Kinds},
			"status": map[string]any{"type": "string", "enum": media.Statuses},
		},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Kind   string `json:"kind"`
				Status string `json:"status"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			items, err := media.List(ctx, q, userID, media.Filter{Kind: in.Kind, Status: in.Status})
			if err != nil {
				return nil, err
			}
			// Reviews and catalogue descriptions stay out of the prompt.
			type row struct {
				ID       uuid.UUID `json:"id"`
				Title    string    `json:"title"`
				Kind     string    `json:"kind"`
				Status   string    `json:"status"`
				Year     *int      `json:"year,omitempty"`
				Creators string    `json:"creators,omitempty"`
				Genres   []string  `json:"genres,omitempty"`
				Rating   *int      `json:"rating,omitempty"`
				Progress string    `json:"progress,omitempty"`
			}
			out := make([]row, 0, min(len(items), 300))
			for _, it := range items[:min(len(items), 300)] {
				r := row{it.ID, it.Title, it.Kind, it.Status, it.Year, it.Creators, it.Genres, it.Rating, ""}
				if it.Total != nil {
					r.Progress = fmt.Sprintf("%d/%d", it.Progress, *it.Total)
				}
				out = append(out, r)
			}
			return out, nil
		},
	}
	registry["media_recommendations"] = Tool{
		Name:        "media_recommendations",
		Description: "Axon's own ranking of the person's want list and unfinished items, each with its reasons (shared creators and genres with what they rated 8 or more).",
		Properties:  map[string]any{"kind": map[string]any{"type": "string", "enum": media.Kinds}},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Kind string `json:"kind"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			recs, err := media.Recommend(ctx, q, userID, in.Kind, time.Now(), 10)
			if err != nil {
				return nil, err
			}
			type row struct {
				ID      uuid.UUID `json:"id"`
				Title   string    `json:"title"`
				Kind    string    `json:"kind"`
				Reasons []string  `json:"reasons"`
			}
			out := make([]row, len(recs))
			for i, r := range recs {
				out[i] = row{r.Item.ID, r.Item.Title, r.Item.Kind, r.Reasons}
			}
			return out, nil
		},
	}
	registry["suggest_media"] = Tool{
		Name: "suggest_media",
		Description: "Propose adding a film, series, or book to the person's want list. reason says why it fits them, naming the items on their list it relates to. " +
			"Nothing is added until the person approves.",
		Properties: map[string]any{
			"kind":     map[string]any{"type": "string", "enum": media.Kinds},
			"title":    str,
			"year":     map[string]any{"type": "integer"},
			"creators": map[string]any{"type": "string", "description": "Author, director, or creator."},
			"genres":   map[string]any{"type": "array", "maxItems": 5, "items": str},
			"reason":   reason,
		},
		Required: []string{"kind", "title", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in mediaInput
			if err := decode(raw, &in); err != nil {
				return "", err
			}
			by := ""
			if in.Creators != "" {
				by = " by " + in.Creators
			}
			return fmt.Sprintf("Add the %s “%s”%s to your list: %s", in.Kind, in.Title, by, truncate(in.Reason, 200)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			f, err := mediaFields(raw)
			if err != nil {
				return err
			}
			_, err = media.Create(ctx, tx, userID, actor, f, nil, time.Now())
			if v, ok := err.(media.ValidationError); ok { //nolint:errorlint // returned unwrapped by media
				return toolErrorf("%s", v.Msg)
			}
			return err
		},
	}
}
