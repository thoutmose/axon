// Package roadmaps holds learning roadmaps: skills with prerequisites, built
// by hand or imported from openly licensed curricula (EP-14).
package roadmaps

import (
	"context"
	_ "embed"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/clock"
	"github.com/thoutmose/axon/api/internal/db"
	"github.com/thoutmose/axon/api/internal/graph"
)

// Entity types and the prerequisite relation.
const (
	Type      = "roadmap"
	SkillType = "skill"
	Requires  = "requires"
)

func init() {
	graph.RegisterType(graph.Type{Name: Type, Table: "roadmaps", Export: export})
	graph.RegisterType(graph.Type{Name: SkillType, Table: "skills", Export: exportSkills})
	graph.RegisterRelation(graph.Relation{Name: Requires, Sources: []string{SkillType}, Targets: []string{SkillType}})
}

// ValidationError carries a message safe to show to people.
type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

var statuses = []string{"todo", "learning", "done", "skipped"}

// Roadmap is the API representation; Skills and Edges come with Get.
type Roadmap struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	Attribution string    `json:"attribution"`
	LicenseURL  *string   `json:"license_url"`
	Done        int       `json:"done"`
	Total       int       `json:"total"` // skipped skills are not counted
	PlanID      *string   `json:"plan_id"`      // the library plan it was started from
	PlanCurrent bool      `json:"plan_current"` // started from the plan's current version
	Skills      []Skill   `json:"skills,omitempty"`
	Edges       []Edge    `json:"edges,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Skill is one thing to learn.
type Skill struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         *string   `json:"url"`
	Section     string    `json:"section"`
	Status      string    `json:"status"`
	Kind        string    `json:"kind"` // hard or soft
	Position    int       `json:"position"`
	// Resources are the books, courses, and sites that teach this skill.
	Resources []SkillResource `json:"resources"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// SkillResource is one way to learn a skill.
type SkillResource struct {
	Title string `json:"title"`
	Kind  string `json:"kind"` // book, course, video, channel, paper, site, practice
	By    string `json:"by"`
	Free  bool   `json:"free"`
	URL   string `json:"url"`
	Note  string `json:"note,omitempty"` // e.g. duration and effort
}

// Edge says a skill requires another (from requires to).
type Edge struct {
	LinkID uuid.UUID `json:"link_id"`
	From   uuid.UUID `json:"from"`
	To     uuid.UUID `json:"to"`
}

const roadmapColumns = `r.id, e.title, r.description, r.source, r.attribution, r.license_url,
	(SELECT count(*) FROM skills s WHERE s.roadmap_id = r.id AND s.status = 'done'),
	(SELECT count(*) FROM skills s WHERE s.roadmap_id = r.id AND s.status <> 'skipped'),
	e.created_at, r.updated_at, r.plan_id, r.plan_version`

func scan(row pgx.Row) (Roadmap, error) {
	var r Roadmap
	var version *int
	err := row.Scan(&r.ID, &r.Title, &r.Description, &r.Source, &r.Attribution, &r.LicenseURL, &r.Done, &r.Total, &r.CreatedAt, &r.UpdatedAt, &r.PlanID, &version)
	r.PlanCurrent = r.PlanID != nil && version != nil && currentVersion(*r.PlanID) == *version
	return r, err
}

// currentVersion is the version of a library plan, 0 for an unknown one.
func currentVersion(planID string) int {
	for _, d := range lib.Roadmaps {
		if d.ID == planID && d.Plan != nil {
			return d.Plan.Version
		}
	}
	return 0
}

// List returns the user's roadmaps, newest first.
func List(ctx context.Context, q db.Querier, userID uuid.UUID) ([]Roadmap, error) {
	rows, err := q.Query(ctx, `SELECT `+roadmapColumns+` FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE e.user_id = $1 ORDER BY e.created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Roadmap, error) { return scan(r) })
	if out == nil {
		out = []Roadmap{}
	}
	return out, err
}

const skillColumns = `s.id, e.title, s.description, s.url, s.section, s.status, s.kind, s.position, s.resources, s.updated_at`

func scanSkill(row pgx.Row) (Skill, error) {
	var s Skill
	err := row.Scan(&s.ID, &s.Title, &s.Description, &s.URL, &s.Section, &s.Status, &s.Kind, &s.Position, &s.Resources, &s.UpdatedAt)
	return s, err
}

// Get returns a roadmap with its skills and prerequisite edges.
func Get(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Roadmap, error) {
	r, err := scan(q.QueryRow(ctx, `SELECT `+roadmapColumns+` FROM roadmaps r JOIN entities e ON e.id = r.id
		WHERE r.id = $1 AND e.user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return r, graph.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	rows, err := q.Query(ctx, `SELECT `+skillColumns+` FROM skills s JOIN entities e ON e.id = s.id
		WHERE s.roadmap_id = $1 ORDER BY s.position, e.created_at LIMIT 2000`, id)
	if err != nil {
		return r, err
	}
	if r.Skills, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Skill, error) { return scanSkill(row) }); err != nil {
		return r, err
	}
	rows, err = q.Query(ctx, `SELECT l.id, l.source_id, l.target_id FROM links l
		JOIN skills a ON a.id = l.source_id JOIN skills b ON b.id = l.target_id
		WHERE l.relation = $2 AND a.roadmap_id = $1 AND b.roadmap_id = $1 AND l.user_id = $3`, id, Requires, userID)
	if err != nil {
		return r, err
	}
	r.Edges, err = pgx.CollectRows(rows, pgx.RowToStructByPos[Edge])
	if r.Skills == nil {
		r.Skills = []Skill{}
	}
	if r.Edges == nil {
		r.Edges = []Edge{}
	}
	return r, err
}

// Fields are the editable parts of a roadmap.
type Fields struct{ Title, Description *string }

func (f Fields) validate() error {
	if f.Title != nil && (strings.TrimSpace(*f.Title) == "" || len([]rune(*f.Title)) > 500) {
		return ValidationError{"A roadmap needs a title of 1 to 500 characters."}
	}
	if f.Description != nil && len(*f.Description) > 20000 {
		return ValidationError{"The description is limited to 20,000 characters."}
	}
	return nil
}

func or(p *string, def string) string {
	if p == nil {
		return def
	}
	return strings.TrimSpace(*p)
}

// Create adds a manual roadmap.
func Create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, f Fields) (Roadmap, error) {
	if f.Title == nil {
		return Roadmap{}, ValidationError{"A roadmap needs a title."}
	}
	if err := f.validate(); err != nil {
		return Roadmap{}, err
	}
	id, err := create(ctx, tx, userID, actor, or(f.Title, ""), or(f.Description, ""), "manual", "", nil)
	if err != nil {
		return Roadmap{}, err
	}
	return Get(ctx, tx, userID, id)
}

func create(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, title, desc, source, attribution string, license *string) (uuid.UUID, error) {
	id, err := graph.CreateEntity(ctx, tx, userID, actor, Type, title, desc)
	if err != nil {
		return id, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO roadmaps (id, description, source, attribution, license_url) VALUES ($1, $2, $3, $4, $5)`,
		id, desc, source, attribution, license)
	return id, err
}

// Update changes a roadmap's title or description.
func Update(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, f Fields) (Roadmap, error) {
	if err := f.validate(); err != nil {
		return Roadmap{}, err
	}
	cur, err := Get(ctx, tx, userID, id)
	if err != nil {
		return Roadmap{}, err
	}
	title, desc := or(f.Title, cur.Title), or(f.Description, cur.Description)
	if title == cur.Title && desc == cur.Description {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, title, desc, []string{"title", "description"}); err != nil {
		return Roadmap{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE roadmaps SET description = $2, updated_at = clock_timestamp() WHERE id = $1`, id, desc); err != nil {
		return Roadmap{}, err
	}
	return Get(ctx, tx, userID, id)
}

// Delete removes a roadmap, its skills, and their links.
func Delete(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	r, err := Get(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	for _, s := range r.Skills {
		if err := graph.DeleteEntity(ctx, tx, userID, s.ID, actor); err != nil {
			return err
		}
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// SkillFields are the editable parts of a skill.
type SkillFields struct {
	Title, Description, URL, Section, Status, Kind *string
	Position                                       *int
	Requires                                       []uuid.UUID     // on create only
	Resources                                      []SkillResource // set by imports only
}

func (f *SkillFields) validate() error {
	if f.Title != nil && (strings.TrimSpace(*f.Title) == "" || len([]rune(*f.Title)) > 500) {
		return ValidationError{"A skill needs a title of 1 to 500 characters."}
	}
	if f.Description != nil && len(*f.Description) > 5000 {
		return ValidationError{"The description is limited to 5,000 characters."}
	}
	if f.URL != nil && *f.URL != "" {
		u, err := url.Parse(*f.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(*f.URL) > 1000 {
			return ValidationError{"A link starts with http:// or https://."}
		}
	}
	if f.Section != nil && len([]rune(*f.Section)) > 200 {
		return ValidationError{"A section name is up to 200 characters."}
	}
	if f.Status != nil && !slices.Contains(statuses, *f.Status) {
		return ValidationError{"Status is todo, learning, done, or skipped."}
	}
	if f.Kind != nil && *f.Kind != "hard" && *f.Kind != "soft" {
		return ValidationError{"Kind is hard or soft."}
	}
	return nil
}

func nullable(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// GetSkill returns one of the user's skills.
func GetSkill(ctx context.Context, q db.Querier, userID, id uuid.UUID) (Skill, uuid.UUID, error) {
	var roadmap uuid.UUID
	var s Skill
	err := q.QueryRow(ctx, `SELECT `+skillColumns+`, s.roadmap_id FROM skills s JOIN entities e ON e.id = s.id WHERE s.id = $1 AND e.user_id = $2`, id, userID).
		Scan(&s.ID, &s.Title, &s.Description, &s.URL, &s.Section, &s.Status, &s.Kind, &s.Position, &s.Resources, &s.UpdatedAt, &roadmap)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, roadmap, graph.ErrNotFound
	}
	return s, roadmap, err
}

// AddSkill adds a skill (at the end unless a position is given) and its prerequisites.
func AddSkill(ctx context.Context, tx pgx.Tx, userID, roadmapID uuid.UUID, actor graph.Actor, f SkillFields) (Skill, error) {
	if f.Title == nil {
		return Skill{}, ValidationError{"A skill needs a title."}
	}
	if err := f.validate(); err != nil {
		return Skill{}, err
	}
	if _, err := scan(tx.QueryRow(ctx, `SELECT `+roadmapColumns+` FROM roadmaps r JOIN entities e ON e.id = r.id WHERE r.id = $1 AND e.user_id = $2`, roadmapID, userID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Skill{}, graph.ErrNotFound
		}
		return Skill{}, err
	}
	id, err := addSkill(ctx, tx, userID, roadmapID, actor, f)
	if err != nil {
		return Skill{}, err
	}
	for _, req := range f.Requires {
		_, other, err := GetSkill(ctx, tx, userID, req)
		if err != nil || other != roadmapID {
			return Skill{}, ValidationError{"Prerequisites must be skills on the same roadmap."}
		}
		if _, err := graph.CreateLink(ctx, tx, userID, actor, id, Requires, req); err != nil {
			return Skill{}, err
		}
	}
	s, _, err := GetSkill(ctx, tx, userID, id)
	return s, err
}

func addSkill(ctx context.Context, tx pgx.Tx, userID, roadmapID uuid.UUID, actor graph.Actor, f SkillFields) (uuid.UUID, error) {
	title := strings.TrimSpace(*f.Title)
	id, err := graph.CreateEntity(ctx, tx, userID, actor, SkillType, title, or(f.Description, ""))
	if err != nil {
		return id, err
	}
	pos := 0
	if f.Position != nil {
		pos = *f.Position
	} else if err := tx.QueryRow(ctx, `SELECT coalesce(max(position), 0) + 1 FROM skills WHERE roadmap_id = $1`, roadmapID).Scan(&pos); err != nil {
		return id, err
	}
	status, kind := "todo", "hard"
	if f.Status != nil {
		status = *f.Status
	}
	if f.Kind != nil {
		kind = *f.Kind
	}
	resources := f.Resources
	if resources == nil {
		resources = []SkillResource{}
	}
	_, err = tx.Exec(ctx, `INSERT INTO skills (id, roadmap_id, description, url, section, status, kind, position, resources)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, roadmapID, or(f.Description, ""), nullable(f.URL), or(f.Section, ""), status, kind, pos, resources)
	return id, err
}

// UpdateSkill changes a skill.
func UpdateSkill(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor, f SkillFields) (Skill, error) {
	if err := f.validate(); err != nil {
		return Skill{}, err
	}
	cur, _, err := GetSkill(ctx, tx, userID, id)
	if err != nil {
		return Skill{}, err
	}
	next := cur
	var changed []string
	set := func(name string, p *string, dst *string) {
		if p != nil && strings.TrimSpace(*p) != *dst {
			*dst = strings.TrimSpace(*p)
			changed = append(changed, name)
		}
	}
	set("title", f.Title, &next.Title)
	set("description", f.Description, &next.Description)
	set("section", f.Section, &next.Section)
	set("status", f.Status, &next.Status)
	set("kind", f.Kind, &next.Kind)
	if f.URL != nil {
		next.URL = nullable(f.URL)
		changed = append(changed, "url")
	}
	if f.Position != nil && *f.Position != cur.Position {
		next.Position = *f.Position
		changed = append(changed, "position")
	}
	if len(changed) == 0 {
		return cur, nil
	}
	if err := graph.UpdateEntity(ctx, tx, userID, id, actor, next.Title, next.Description, changed); err != nil {
		return Skill{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE skills SET description = $2, url = $3, section = $4, status = $5, position = $6, kind = $7,
		updated_at = clock_timestamp() WHERE id = $1`, id, next.Description, next.URL, next.Section, next.Status, next.Position, next.Kind); err != nil {
		return Skill{}, err
	}
	s, _, err := GetSkill(ctx, tx, userID, id)
	return s, err
}

// DeleteSkill removes a skill and its links.
func DeleteSkill(ctx context.Context, tx pgx.Tx, userID, id uuid.UUID, actor graph.Actor) error {
	if _, _, err := GetSkill(ctx, tx, userID, id); err != nil {
		return err
	}
	return graph.DeleteEntity(ctx, tx, userID, id, actor)
}

// Sources lists what can be imported: the library's roadmaps.
func Sources() []Source {
	var out []Source
	for _, d := range lib.Roadmaps {
		out = append(out, Source{ID: d.ID, Kind: "library", Title: d.Title, Description: d.Description, Attribution: lib.Attribution,
			Count: len(d.Resources), Skills: len(d.Skills)})
	}
	return out
}

// Import creates a roadmap from the library. A roadmap with a plan starts
// the whole plan (EP-95), dated from the person's today.
func Import(ctx context.Context, tx pgx.Tx, userID uuid.UUID, actor graph.Actor, source string, now time.Time) (Roadmap, error) {
	for _, d := range lib.Roadmaps {
		if d.ID != source {
			continue
		}
		if d.Plan == nil {
			return importLibrary(ctx, tx, userID, actor, d)
		}
		today, _, err := clock.Today(ctx, tx, userID, now)
		if err != nil {
			return Roadmap{}, err
		}
		s, err := StartPlan(ctx, tx, userID, actor, d, d.Plan.Shift(today), now)
		return s.Roadmap, err
	}
	return Roadmap{}, ValidationError{"Import a roadmap listed by GET /v1/roadmaps/sources. roadmap.sh content may not be redistributed."}
}

func export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+roadmapColumns+` FROM roadmaps r JOIN entities e ON e.id = r.id WHERE e.user_id = $1 ORDER BY r.id`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Roadmap, error) { return scan(r) })
}

func exportSkills(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	rows, err := q.Query(ctx, `SELECT `+skillColumns+` FROM skills s JOIN entities e ON e.id = s.id WHERE e.user_id = $1 ORDER BY s.id`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Skill, error) { return scanSkill(r) })
}
