// Package settings holds a person's preferences (EP-31): language,
// appearance, templates, and the AI provider and budget. They can be edited
// in the app or as a YAML file; the same document and validation serve both.
// Secrets are write-only: an exported file never contains a key.
package settings

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Appearance is how Axon looks.
type Appearance struct {
	Theme    string `json:"theme" yaml:"theme"`         // system, light, dark
	Font     string `json:"font" yaml:"font"`           // system, serif, rounded, mono
	FontSize string `json:"font_size" yaml:"font_size"` // small, normal, large
	Icons    string `json:"icons" yaml:"icons"`         // emoji, symbols, none
	Motion   string `json:"motion" yaml:"motion"`       // system, reduced, full
}

// ProjectTemplate are defaults for new projects.
type ProjectTemplate struct {
	Description string `json:"description" yaml:"description"`
	WIPLimit    *int   `json:"wip_limit" yaml:"wip_limit"`
	DocsFolder  string `json:"docs_folder" yaml:"docs_folder"` // "{slug}" is the project's slug
}

// ResourceTemplate are defaults for skills added to roadmaps.
type ResourceTemplate struct {
	Section string `json:"section" yaml:"section"`
	Kind    string `json:"kind" yaml:"kind"` // hard or soft
}

// Templates fill new notes, projects, and resources.
type Templates struct {
	Note     string           `json:"note" yaml:"note"`
	Project  ProjectTemplate  `json:"project" yaml:"project"`
	Resource ResourceTemplate `json:"resource" yaml:"resource"`
}

// Doc is the stored part of the settings.
type Doc struct {
	Language   string     `json:"language" yaml:"language"` // en or fr
	Appearance Appearance `json:"appearance" yaml:"appearance"`
	Templates  Templates  `json:"templates" yaml:"templates"`
}

// Defaults are what a new account starts with.
func Defaults() Doc {
	return Doc{
		Language:   "en",
		Appearance: Appearance{Theme: "system", Font: "system", FontSize: "normal", Icons: "emoji", Motion: "system"},
		Templates: Templates{
			Note:     "---\ntags: []\n---\n# {title}\n\n",
			Project:  ProjectTemplate{DocsFolder: "1-Projects/{slug}"},
			Resource: ResourceTemplate{Kind: "hard"},
		},
	}
}

// FieldError is a value a person can correct; Path names it, like
// "appearance.theme", so YAML errors can point to a line.
type FieldError struct {
	Path string
	Msg  string
}

func (e FieldError) Error() string { return e.Path + ": " + e.Msg }

func oneOf(path, v string, allowed ...string) error {
	if !slices.Contains(allowed, v) {
		return FieldError{path, fmt.Sprintf("must be one of %s (got %q)", strings.Join(allowed, ", "), v)}
	}
	return nil
}

// Validate checks every field.
func (d Doc) Validate() error {
	checks := []error{
		oneOf("language", d.Language, "en", "fr"),
		oneOf("appearance.theme", d.Appearance.Theme, "system", "light", "dark"),
		oneOf("appearance.font", d.Appearance.Font, "system", "serif", "rounded", "mono"),
		oneOf("appearance.font_size", d.Appearance.FontSize, "small", "normal", "large"),
		oneOf("appearance.icons", d.Appearance.Icons, "emoji", "symbols", "none"),
		oneOf("appearance.motion", d.Appearance.Motion, "system", "reduced", "full"),
		oneOf("templates.resource.kind", d.Templates.Resource.Kind, "hard", "soft"),
	}
	for _, err := range checks {
		if err != nil {
			return err
		}
	}
	switch {
	case len(d.Templates.Note) > 20000:
		return FieldError{"templates.note", "is limited to 20,000 characters"}
	case len(d.Templates.Project.Description) > 20000:
		return FieldError{"templates.project.description", "is limited to 20,000 characters"}
	case d.Templates.Project.WIPLimit != nil && (*d.Templates.Project.WIPLimit < 1 || *d.Templates.Project.WIPLimit > 50):
		return FieldError{"templates.project.wip_limit", "is between 1 and 50, or empty for no limit"}
	case len([]rune(d.Templates.Project.DocsFolder)) > 200 || strings.Contains(d.Templates.Project.DocsFolder, ".."):
		return FieldError{"templates.project.docs_folder", "is a folder path up to 200 characters, without .."}
	case len([]rune(d.Templates.Resource.Section)) > 200:
		return FieldError{"templates.resource.section", "is up to 200 characters"}
	}
	return nil
}

// Get returns the person's settings, filled with defaults.
func Get(ctx context.Context, q db.Querier, userID uuid.UUID) (Doc, error) {
	d := Defaults()
	err := q.QueryRow(ctx, `SELECT doc FROM user_settings WHERE user_id = $1`, userID).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return Defaults(), nil
	}
	return d, err
}

// Put validates and stores the settings.
func Put(ctx context.Context, q db.Querier, userID uuid.UUID, d Doc) error {
	if err := d.Validate(); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `INSERT INTO user_settings (user_id, doc) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET doc = $2, updated_at = now()`, userID, d)
	return err
}

// Export is the settings part of a personal data export.
func Export(ctx context.Context, q db.Querier, userID uuid.UUID) (any, error) {
	return Get(ctx, q, userID)
}
