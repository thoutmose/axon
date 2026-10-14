package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/watch"
)

// Tech watch tools (EP-76). The agent reads the person's topics, sources,
// news, and the trends of their news graph; following a source or adding
// a topic is a proposal. News text comes from other sites: it reaches the
// model as data, cut short.

type watchSourceInput struct {
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Topics   []string `json:"topics"`
	Schedule string   `json:"schedule"`
	Reason   string   `json:"reason"`
}

type watchTopicInput struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Reason   string   `json:"reason"`
}

func init() {
	registry["list_watch_topics"] = Tool{
		Name:        "list_watch_topics",
		Description: "List the person's tech watch topics with their keywords and how many news each had in the last 30 days.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			topics, err := watch.ListTopics(ctx, q, userID)
			if err != nil {
				return nil, err
			}
			type row struct {
				Name     string   `json:"name"`
				Keywords []string `json:"keywords"`
				News     int      `json:"news_30_days"`
			}
			out := make([]row, 0, len(topics))
			for _, t := range topics {
				var n int
				if err := q.QueryRow(ctx, `SELECT count(*) FROM watch_items WHERE user_id = $1 AND $2::uuid = ANY (topic_ids)
					AND published_at > now() - interval '30 days'`, userID, t.ID).Scan(&n); err != nil {
					return nil, err
				}
				out = append(out, row{t.Name, t.Keywords, n})
			}
			return out, nil
		},
	}
	registry["list_watch_sources"] = Tool{
		Name:        "list_watch_sources",
		Description: "List the sources the person follows: title, address, health (ok, failing, waiting, off), news this week, and topics.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			sources, err := watch.ListSources(ctx, q, userID)
			if err != nil {
				return nil, err
			}
			names := map[string]string{}
			if ts, err := watch.ListTopics(ctx, q, userID); err == nil {
				for _, t := range ts {
					names[t.ID] = t.Name
				}
			}
			type row struct {
				Title  string   `json:"title"`
				URL    string   `json:"url"`
				Health string   `json:"health"`
				Week   int      `json:"news_this_week"`
				Topics []string `json:"topics,omitempty"`
			}
			out := make([]row, 0, min(len(sources), 300))
			for _, s := range sources[:min(len(sources), 300)] {
				r := row{Title: s.Title, URL: s.URL, Health: s.Health, Week: s.ItemsWeek}
				for _, id := range s.TopicIDs {
					r.Topics = append(r.Topics, names[id])
				}
				out = append(out, r)
			}
			return out, nil
		},
	}
	registry["list_watch_news"] = Tool{
		Name:        "list_watch_news",
		Description: "List recent news from the person's tech watch, newest first, at most 50: title, source, date, kind, labels, a short summary, and the other sources that carried it.",
		Properties: map[string]any{
			"days":  map[string]any{"type": "integer", "minimum": 1, "maximum": 30, "description": "How far back, 7 by default."},
			"topic": map[string]any{"type": "string", "description": "A topic name."},
			"label": str,
			"kind":  map[string]any{"type": "string", "enum": watch.Kinds},
			"saved": map[string]any{"type": "boolean", "description": "Only what the person saved."},
		},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Days  int    `json:"days"`
				Topic string `json:"topic"`
				Label string `json:"label"`
				Kind  string `json:"kind"`
				Saved bool   `json:"saved"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			if in.Days == 0 {
				in.Days = 7
			}
			if in.Days < 1 || in.Days > 30 {
				return nil, toolErrorf("days is between 1 and 30")
			}
			if in.Kind != "" && !slices.Contains(watch.Kinds, in.Kind) {
				return nil, toolErrorf("kind is article, video, or podcast")
			}
			from := time.Now().AddDate(0, 0, -in.Days)
			f := watch.Filter{Status: "all", Label: in.Label, Kind: in.Kind, From: &from, Limit: 50}
			if in.Saved {
				f.Status = "saved"
			}
			if in.Topic != "" {
				topics, err := watch.ListTopics(ctx, q, userID)
				if err != nil {
					return nil, err
				}
				i := slices.IndexFunc(topics, func(t watch.Topic) bool { return strings.EqualFold(t.Name, in.Topic) })
				if i < 0 {
					return nil, toolErrorf("no topic named %q; list_watch_topics lists them", in.Topic)
				}
				f.TopicID = topics[i].ID
			}
			items, err := watch.ListItems(ctx, q, userID, f)
			if err != nil {
				return nil, err
			}
			type row struct {
				Title     string   `json:"title"`
				Source    string   `json:"source"`
				Published string   `json:"published"`
				Kind      string   `json:"kind"`
				Labels    []string `json:"labels"`
				Summary   string   `json:"summary"`
				AlsoIn    []string `json:"also_in,omitempty"`
				URL       string   `json:"url"`
			}
			out := make([]row, 0, len(items))
			for _, it := range items {
				out = append(out, row{it.Title, it.SourceTitle, it.PublishedAt.Format(time.DateOnly), it.Kind, it.Labels,
					truncate(it.Summary, 300), it.AlsoIn, it.URL})
			}
			return out, nil
		},
	}
	registry["watch_trends"] = Tool{
		Name: "watch_trends",
		Description: "The concepts of the person's news over the last 30 days, from their news graph: how many news carry each, " +
			"its trend (above 1.5 rising, below -1.5 fading), the concepts it goes with, and whether their own notes share it.",
		Properties: map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			now := time.Now()
			g, err := watch.BuildGraph(ctx, q, userID, watch.GraphFilter{From: now.AddDate(0, 0, -30), To: now})
			if err != nil {
				return nil, err
			}
			type row struct {
				Concept string   `json:"concept"`
				News    int      `json:"news"`
				Trend   float64  `json:"trend"`
				With    []string `json:"goes_with,omitempty"`
				InNotes bool     `json:"in_their_notes"`
			}
			out := make([]row, 0, min(len(g.Nodes), 40))
			slices.SortFunc(g.Nodes, func(a, b watch.GNode) int { return b.Count - a.Count })
			for _, n := range g.Nodes[:min(len(g.Nodes), 40)] {
				out = append(out, row{n.Label, n.Count, n.Trend, n.Related, len(n.InNotes) > 0})
			}
			return map[string]any{"news": g.Stats.Items, "concepts": out}, nil
		},
	}
	registry["suggest_watch_source"] = Tool{
		Name: "suggest_watch_source",
		Description: "Propose following a source in the tech watch: a feed, a site, a Medium author, a Substack, a YouTube channel, or a GitHub repository. " +
			"Axon finds the feed behind the address and fetches it once approved. topics are names of existing or new topics; schedule is a five-field cron, " +
			"midnight by default. reason says what it adds to what they already follow. Nothing changes until the person approves.",
		Properties: map[string]any{
			"url":      map[string]any{"type": "string", "description": "An https address the person or a page you read gave, or one you are sure of."},
			"title":    str,
			"topics":   map[string]any{"type": "array", "maxItems": 5, "items": str},
			"schedule": map[string]any{"type": "string", "description": "e.g. 0 7 * * * for 07:00 every day."},
			"reason":   reason,
		},
		Required: []string{"url", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in watchSourceInput
			if err := decode(raw, &in); err != nil {
				return "", err
			}
			name := in.URL
			if in.Title != "" {
				name = in.Title + " (" + in.URL + ")"
			}
			return fmt.Sprintf("Follow %s in the tech watch: %s", truncate(name, 200), truncate(in.Reason, 200)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, _ graph.Actor, raw json.RawMessage) error {
			var in watchSourceInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if !strings.HasPrefix(in.URL, "https://") || strings.TrimSpace(in.Reason) == "" {
				return toolErrorf("url is an https address, and reason is required")
			}
			res, err := watch.Import(ctx, tx, userID, []watch.Planned{{URL: in.URL, Title: in.Title, Topics: in.Topics, Schedule: in.Schedule}}, "agent", time.Now())
			switch {
			case err != nil:
				return err
			case len(res.Failed) > 0:
				return toolErrorf("%s", res.Failed[0].Reason)
			case res.Existing > 0:
				return toolErrorf("the person already follows %s", in.URL)
			}
			return nil
		},
	}
	registry["suggest_watch_topic"] = Tool{
		Name: "suggest_watch_topic",
		Description: "Propose a tech watch topic: a name and keywords (a keyword in a title, or two in the text, label a news item with the topic; " +
			"synonyms count). reason says which news or interests it gathers. Nothing changes until the person approves.",
		Properties: map[string]any{
			"name":     str,
			"keywords": map[string]any{"type": "array", "minItems": 1, "maxItems": 20, "items": str},
			"reason":   reason,
		},
		Required: []string{"name", "keywords", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in watchTopicInput
			if err := decode(raw, &in); err != nil {
				return "", err
			}
			return fmt.Sprintf("Add the topic “%s” (%s): %s", truncate(in.Name, 80), truncate(strings.Join(in.Keywords, ", "), 200), truncate(in.Reason, 200)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, _ graph.Actor, raw json.RawMessage) error {
			var in watchTopicInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			if len(in.Keywords) == 0 || strings.TrimSpace(in.Reason) == "" {
				return toolErrorf("keywords and reason are required")
			}
			_, err := watch.SaveTopic(ctx, tx, userID, "", in.Name, in.Keywords)
			var inv watch.InvalidError
			if errors.As(err, &inv) {
				return toolErrorf("%s", inv.Msg)
			}
			return err
		},
	}
}
