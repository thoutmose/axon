package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/roadmaps"
	"github.com/thoutmose/axon/api/internal/web"
)

// Learning tools for the Tutor and Resource agents (EP-27). They read the
// curated library, the person's roadmaps, and allowlisted public pages.
// Creating a roadmap or adding resources is a proposal the person approves.

// maxFetches bounds the pages one run may read.
const maxFetches = 15

type fetchKey struct{}

func withFetchBudget(ctx context.Context) context.Context {
	return context.WithValue(ctx, fetchKey{}, new(atomic.Int32))
}

type draftRoadmapInput struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Skills      []roadmaps.DraftSkill `json:"skills"`
	Reason      string                `json:"reason"`
}

type resourceInput struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Section     string `json:"section"`
	Description string `json:"description"`
}

type suggestInput struct {
	RoadmapID string          `json:"roadmap_id"`
	Resources []resourceInput `json:"resources"`
	Reason    string          `json:"reason"`
}

// sources counts the distinct sites a draft cites.
func sources(urls []string) int {
	hosts := map[string]bool{}
	for _, raw := range urls {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			hosts[u.Host] = true
		}
	}
	return len(hosts)
}

var citedURL = map[string]any{"type": "string", "description": "The http(s) source this is based on: a library resource, or a page you read with fetch_page."}

func init() {
	registry["search_library"] = Tool{
		Name:        "search_library",
		Description: "Search Axon's curated library of widely recommended books, papers, courses, videos, and channels across computing domains. Every word must match the title, author, section, or domain.",
		Properties: map[string]any{
			"query": map[string]any{"type": "string", "description": "Words such as \"kubernetes\" or \"distributed systems paper\"."},
			"kind":  map[string]any{"type": "string", "enum": []string{"book", "course", "video", "channel", "paper", "site", "practice"}},
			"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 25},
		},
		Required: []string{"query"},
		Read: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				Query string `json:"query"`
				Kind  string `json:"kind"`
				Limit int    `json:"limit"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			if in.Limit <= 0 || in.Limit > 25 {
				in.Limit = 25
			}
			found := roadmaps.SearchLibrary(in.Query, in.Kind, in.Limit)
			if found == nil {
				found = []roadmaps.LibraryMatch{}
			}
			return found, nil
		},
	}
	registry["list_library"] = Tool{
		Name:        "list_library",
		Description: "List the curated library's domains (id, title, number of resources). Import them as roadmaps from the Learn page.",
		Properties:  map[string]any{},
		Read: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			type row struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Resources int    `json:"resources"`
			}
			out := []row{}
			for _, d := range roadmaps.Library() {
				out = append(out, row{d.ID, d.Title, len(d.Resources)})
			}
			return out, nil
		},
	}
	registry["fetch_page"] = Tool{
		Name: "fetch_page",
		Description: fmt.Sprintf("Read the text of a public learning page (up to 12,000 characters). Only allowlisted educational sites "+
			"(%s and their subdomains) are reachable, robots.txt is honoured, and a run may read %d pages.", strings.Join(web.Allow, ", "), maxFetches),
		Properties: map[string]any{"url": map[string]any{"type": "string", "description": "An https URL on an allowlisted site."}},
		Required:   []string{"url"},
		Read: func(ctx context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (any, error) {
			var in struct {
				URL string `json:"url"`
			}
			if err := decode(raw, &in); err != nil {
				return nil, err
			}
			if n, ok := ctx.Value(fetchKey{}).(*atomic.Int32); ok && n.Add(1) > maxFetches {
				return nil, toolErrorf("this run already read %d pages; answer with what you have", maxFetches)
			}
			p, err := web.Fetch(ctx, in.URL)
			switch {
			case errors.Is(err, web.ErrNotAllowed):
				return nil, toolErrorf("%v; use search_library or an allowlisted site", err)
			case errors.Is(err, web.ErrRobots):
				return nil, toolErrorf("the site's robots.txt asks agents not to read this page")
			case err != nil:
				return nil, toolErrorf("%v", err)
			}
			return p, nil
		},
	}
	registry["list_roadmaps"] = Tool{
		Name:        "list_roadmaps",
		Description: "List the person's roadmaps with progress.",
		Properties:  map[string]any{},
		Read: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (any, error) {
			if err := decode(raw, &struct{}{}); err != nil {
				return nil, err
			}
			return roadmaps.List(ctx, q, userID)
		},
	}
	registry["get_roadmap"] = Tool{
		Name:        "get_roadmap",
		Description: "Get a roadmap's skills (title, section, status, link) and prerequisites.",
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
			r, err := roadmaps.Get(ctx, q, userID, id)
			if err != nil {
				return nil, notFound(err, "roadmap")
			}
			type skill struct {
				ID      uuid.UUID `json:"id"`
				Title   string    `json:"title"`
				Section string    `json:"section,omitempty"`
				Status  string    `json:"status"`
				URL     *string   `json:"url,omitempty"`
			}
			skills := []skill{}
			for _, s := range r.Skills {
				skills = append(skills, skill{s.ID, s.Title, s.Section, s.Status, s.URL})
			}
			edges := [][2]uuid.UUID{}
			for _, e := range r.Edges {
				edges = append(edges, [2]uuid.UUID{e.From, e.To})
			}
			return map[string]any{"id": r.ID, "title": r.Title, "description": r.Description, "skills": skills, "requires": edges}, nil
		},
	}
	registry["draft_roadmap"] = Tool{
		Name: "draft_roadmap",
		Description: "Propose a new roadmap. Every skill cites the source it is based on; prerequisites name earlier skills by title. " +
			"Nothing is created until the person approves.",
		Properties: map[string]any{
			"title":       str,
			"description": map[string]any{"type": "string", "description": "Who it is for and the goal, in one or two sentences."},
			"skills": map[string]any{"type": "array", "minItems": 1, "maxItems": 150, "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": str, "section": str, "url": citedURL,
					"description": map[string]any{"type": "string", "description": "What to learn and how long it takes."},
					"requires":    map[string]any{"type": "array", "items": str, "description": "Titles of earlier skills."},
				},
				"required": []string{"title", "url"},
			}},
			"reason": reason,
		},
		Required: []string{"title", "skills", "reason"},
		Write:    true,
		Summary: func(_ context.Context, _ db.Querier, _ uuid.UUID, raw json.RawMessage) (string, error) {
			var in draftRoadmapInput
			_ = json.Unmarshal(raw, &in)
			var urls []string
			for _, s := range in.Skills {
				urls = append(urls, s.URL)
			}
			return fmt.Sprintf("Create roadmap “%s” with %d skills citing %d sources", in.Title, len(in.Skills), sources(urls)), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in draftRoadmapInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			_, err := roadmaps.CreateDraft(ctx, tx, userID, actor, in.Title, in.Description, in.Skills)
			return notFound(err, "roadmap")
		},
	}
	registry["suggest_resources"] = Tool{
		Name:        "suggest_resources",
		Description: "Propose adding learning resources to one of the person's roadmaps, each with its link and attribution. Nothing changes until the person approves.",
		Properties: map[string]any{
			"roadmap_id": idField,
			"resources": map[string]any{"type": "array", "minItems": 1, "maxItems": 20, "items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": str, "url": citedURL, "section": str,
					"description": map[string]any{"type": "string", "description": "Kind, author, whether it is free, and why it helps."},
				},
				"required": []string{"title", "url", "description"},
			}},
			"reason": reason,
		},
		Required: []string{"roadmap_id", "resources", "reason"},
		Write:    true,
		Summary: func(ctx context.Context, q db.Querier, userID uuid.UUID, raw json.RawMessage) (string, error) {
			var in suggestInput
			_ = json.Unmarshal(raw, &in)
			title := "a roadmap"
			if id, err := uuid.Parse(in.RoadmapID); err == nil {
				if r, err := roadmaps.Get(ctx, q, userID, id); err == nil {
					title = "“" + r.Title + "”"
				}
			}
			var names []string
			for _, r := range in.Resources {
				names = append(names, r.Title)
			}
			return fmt.Sprintf("Add %d resources to %s: %s", len(in.Resources), title, strings.Join(names, "; ")), nil
		},
		Apply: func(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, raw json.RawMessage) error {
			var in suggestInput
			if err := decode(raw, &in); err != nil {
				return err
			}
			id, err := parseID(in.RoadmapID, "roadmap_id")
			if err != nil {
				return err
			}
			if len(in.Resources) == 0 || len(in.Resources) > 20 {
				return toolErrorf("suggest 1 to 20 resources")
			}
			for _, r := range in.Resources {
				if strings.TrimSpace(r.URL) == "" {
					return toolErrorf("resource %q needs its link", r.Title)
				}
				f := roadmaps.SkillFields{Title: &r.Title, URL: &r.URL, Description: &r.Description}
				if r.Section != "" {
					f.Section = &r.Section
				}
				if _, err := roadmaps.AddSkill(ctx, tx, userID, id, actor, f); err != nil {
					return notFound(err, "roadmap")
				}
			}
			return nil
		},
	}
}
