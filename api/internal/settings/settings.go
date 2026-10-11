// Package settings holds a person's preferences (EP-31): language,
// appearance, templates, and the AI provider and budget. They can be edited
// in the app or as a YAML file; the same document and validation serve both.
// Secrets are write-only: an exported file never contains a key.
package settings

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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

// Keyboard controls shortcuts and the editor's Vim mode (EP-32).
type Keyboard struct {
	Shortcuts bool `json:"shortcuts" yaml:"shortcuts"` // single-key and g-chord shortcuts (WCAG 2.1.4: they can be turned off)
	Vim       bool `json:"vim" yaml:"vim"`             // Vim keybindings in the note editor
}

// MusicLink is a playlist or stream the person likes for a kind of work.
type MusicLink struct {
	Activity string `json:"activity" yaml:"activity"` // e.g. coding, reading, writing, learning
	URL      string `json:"url" yaml:"url"`
}

// Focus configures Pomodoro sessions (EP-35). The default is three rounds:
// 25 min work + 5 min break, twice, then 25 min work + a 15 min long break.
type Focus struct {
	WorkMinutes int         `json:"work_minutes" yaml:"work_minutes"`
	ShortBreak  int         `json:"short_break_minutes" yaml:"short_break_minutes"`
	LongBreak   int         `json:"long_break_minutes" yaml:"long_break_minutes"`
	Rounds      int         `json:"rounds" yaml:"rounds"`     // work blocks before the long break
	Sessions    int         `json:"sessions" yaml:"sessions"` // how many of those cycles in a row
	Sound       string      `json:"sound" yaml:"sound"`       // none, white, pink, brown, rain
	Volume      int         `json:"volume" yaml:"volume"`     // 0–100
	Music       []MusicLink `json:"music" yaml:"music"`
}

// Gamification turns progress rewards on (EP-15); off means no surface shows them.
type Gamification struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// Doc is the stored part of the settings.
type Doc struct {
	Language     string       `json:"language" yaml:"language"` // en or fr
	Appearance   Appearance   `json:"appearance" yaml:"appearance"`
	Templates    Templates    `json:"templates" yaml:"templates"`
	Keyboard     Keyboard     `json:"keyboard" yaml:"keyboard"`
	Focus        Focus        `json:"focus" yaml:"focus"`
	Gamification Gamification `json:"gamification" yaml:"gamification"`
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
		Keyboard: Keyboard{Shortcuts: true},
		Focus:    Focus{WorkMinutes: 25, ShortBreak: 5, LongBreak: 15, Rounds: 3, Sessions: 1, Sound: "none", Volume: 40, Music: []MusicLink{}},
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
	f := d.Focus
	switch {
	case f.WorkMinutes < 1 || f.WorkMinutes > 180:
		return FieldError{"focus.work_minutes", "is between 1 and 180"}
	case f.ShortBreak < 1 || f.ShortBreak > 60:
		return FieldError{"focus.short_break_minutes", "is between 1 and 60"}
	case f.LongBreak < 1 || f.LongBreak > 120:
		return FieldError{"focus.long_break_minutes", "is between 1 and 120"}
	case f.Rounds < 1 || f.Rounds > 12:
		return FieldError{"focus.rounds", "is between 1 and 12"}
	case f.Sessions < 1 || f.Sessions > 12:
		return FieldError{"focus.sessions", "is between 1 and 12"}
	case f.Volume < 0 || f.Volume > 100:
		return FieldError{"focus.volume", "is between 0 and 100"}
	case len(f.Music) > 20:
		return FieldError{"focus.music", "has up to 20 links"}
	}
	if err := oneOf("focus.sound", f.Sound, "none", "white", "pink", "brown", "rain"); err != nil {
		return err
	}
	for _, m := range f.Music {
		if u, err := url.Parse(m.URL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(m.URL) > 1000 || len([]rune(m.Activity)) > 100 {
			return FieldError{"focus.music", "links are http(s) addresses with an activity up to 100 characters"}
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
