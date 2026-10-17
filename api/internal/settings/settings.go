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
	"regexp"
	"slices"
	"strings"
	"time"

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
	// Spaces turns the themed scenery of each area (EP-38) on or off; empty means on.
	Spaces string `json:"spaces,omitempty" yaml:"spaces,omitempty"`
	// Scenery is the landscape under the sky that follows the sun (EP-96);
	// empty means mountains.
	Scenery string `json:"scenery,omitempty" yaml:"scenery,omitempty"`
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

// Focus configures Pomodoro sessions (EP-35). The default is four rounds:
// 25 min work + 5 min break, three times, then 25 min work + a 15 min long break.
type Focus struct {
	WorkMinutes int         `json:"work_minutes" yaml:"work_minutes"`
	ShortBreak  int         `json:"short_break_minutes" yaml:"short_break_minutes"`
	LongBreak   int         `json:"long_break_minutes" yaml:"long_break_minutes"`
	Rounds      int         `json:"rounds" yaml:"rounds"`     // work blocks before the long break
	Sessions    int         `json:"sessions" yaml:"sessions"` // how many of those cycles in a row
	Sound       string      `json:"sound" yaml:"sound"`       // none, noise (white, pink, brown, rain), or music (music: matched; lofi, ambient, piano)
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
	Vault        Vault        `json:"vault" yaml:"vault"`
	Learning     Learning     `json:"learning" yaml:"learning"`
	Reminders    Reminders    `json:"reminders" yaml:"reminders"`
	Today        Today        `json:"today" yaml:"today"`
	Menu         Menu         `json:"menu" yaml:"menu"`
	Digest       Digest       `json:"digest" yaml:"digest"`
	Podcast      Podcast      `json:"podcast" yaml:"podcast"`
	Caffeine     Caffeine     `json:"caffeine" yaml:"caffeine"`
	Jarvis       Jarvis       `json:"jarvis" yaml:"jarvis"`
}

// Jarvis (EP-153): the master agent's floating button, off until the
// person turns it on.
type Jarvis struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// Caffeine (EP-117): when the person goes to bed, and how fast they clear
// caffeine (its half-life: fast 3 h, average 5 h, slow 7 h; slower in
// pregnancy or with oral contraceptives, faster for smokers).
type Caffeine struct {
	Bedtime    string `json:"bedtime" yaml:"bedtime"`       // "23:00"
	Metabolism string `json:"metabolism" yaml:"metabolism"` // fast, average, slow
}

// Digest (EP-113) is the person's digest template: the sections of each
// digest, in order, and whether the quote and the picture come along.
type Digest struct {
	Daily   []string `json:"daily" yaml:"daily"`
	Weekly  []string `json:"weekly" yaml:"weekly"`
	Monthly []string `json:"monthly" yaml:"monthly"`
	Quote   bool     `json:"quote" yaml:"quote"`
	Picture bool     `json:"picture" yaml:"picture"`
}

// DigestSections are the sections each digest can show.
var DigestSections = map[string][]string{
	"daily":   {"tasks", "habits", "coming", "learn", "watch"},
	"weekly":  {"open", "week", "reflect", "watch"},
	"monthly": {"goals", "month", "watch"},
}

// Podcast (EP-113) is the spoken tech watch, off until the person turns it
// on: its format, the language of its spoken links, its length, the
// period it covers, and the browser voice and pace that read it.
type Podcast struct {
	Enabled bool    `json:"enabled" yaml:"enabled"`
	Format  string  `json:"format" yaml:"format"` // briefing, digest, deep
	Lang    string  `json:"lang" yaml:"lang"`     // en, fr
	Minutes int     `json:"minutes" yaml:"minutes"`
	Days    int     `json:"days" yaml:"days"`   // 1 or 7
	Voice   string  `json:"voice" yaml:"voice"` // a voice name; empty for the device's default
	Rate    float64 `json:"rate" yaml:"rate"`   // 0.5 to 2
	// Hosts is 1, or 2 for a conversation where Voice and Voice2 take turns
	// (EP-132); 0, from before, reads as 1.
	Hosts  int    `json:"hosts" yaml:"hosts"`
	Voice2 string `json:"voice2" yaml:"voice2"`
}

// Sceneries are the landscapes under the sky: nature, the earth from
// orbit, and a famous monument on each inhabited continent.
var Sceneries = []string{"mountains", "forest", "ocean", "desert", "earth",
	"eiffel-tower", "pyramids", "taj-mahal", "statue-of-liberty", "christ-the-redeemer", "opera-house"}

// Menu (EP-94) lists the pages left out of the navigation, so a person
// sees what they use; hidden pages still open from a link or a search.
// Chosen is set once the person has said what they want Axon for.
type Menu struct {
	Hidden []string `json:"hidden" yaml:"hidden"`
	Chosen bool     `json:"chosen" yaml:"chosen"`
}

// MenuPages are the pages the menu can leave out (Today always stays).
var MenuPages = []string{"tasks", "projects", "calendar", "focus", "notes", "watch", "areas", "resources", "archive",
	"learn", "goals", "habits", "health", "media", "budget", "household", "journal", "digest", "review", "decisions", "insights", "agents"}

// Today (EP-63) lists the sections the Today page shows, in order, and
// those the person turned off (EP-145): a section in neither list is new
// in this release, and Today shows it.
type Today struct {
	Sections []string `json:"sections" yaml:"sections"`
	Hidden   []string `json:"hidden" yaml:"hidden"`
}

// TodaySections are the sections Today can show.
var TodaySections = []string{"kpis", "roadmap", "tasks", "milestones", "goals", "notes", "learning", "habits", "calendar", "household", "journal", "water", "caffeine", "overview"}

// Reminders (EP-56): a morning digest and an evening nudge at local times
// ("08:00"; empty for none), pushed to the devices the person turned them
// on for, and emailed to Email when set and the instance can send mail.
//
// The weekly and monthly digests (EP-107) go out on WeeklyDay at WeeklyAt
// and on day MonthlyDay of the month at MonthlyAt; an empty time skips
// them. Digests also reach the chat channels the person connected.
type Reminders struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	Morning    string `json:"morning" yaml:"morning"`
	Evening    string `json:"evening" yaml:"evening"`
	Email      string `json:"email" yaml:"email"`
	WeeklyDay  string `json:"weekly_day" yaml:"weekly_day"`   // monday … sunday
	WeeklyAt   string `json:"weekly_at" yaml:"weekly_at"`     // "18:00", or empty for none
	MonthlyDay int    `json:"monthly_day" yaml:"monthly_day"` // 1–28
	MonthlyAt  string `json:"monthly_at" yaml:"monthly_at"`   // "09:00", or empty for none
}

// Weekdays name the days a weekly digest can go out.
var Weekdays = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// Learning tunes the review engine (EP-52); package cards reads the same
// keys. NewPerDay paces new cards, Retention is the chance FSRS aims for
// of still remembering a card when it comes back, Interleave mixes decks in
// a review of everything, and TypeAnswer asks for the answer in writing
// before it is shown (retrieval, not recognition).
type Learning struct {
	NewPerDay  int     `json:"new_per_day" yaml:"new_per_day"`
	Retention  float64 `json:"retention" yaml:"retention"`
	Interleave bool    `json:"interleave" yaml:"interleave"`
	TypeAnswer bool    `json:"type_answer" yaml:"type_answer"`
}

// Vault controls the vault mirror (EP-48): every task, goal, milestone,
// project, area, resource, habit, and film, series, or book is also a
// Markdown note under Folder, kept in step both ways. Health logs are
// special-category data, so they are mirrored only when Health is on (and
// the health consent is given), one way, from Axon to the vault.
type Vault struct {
	Mirror bool   `json:"mirror" yaml:"mirror"`
	Folder string `json:"folder" yaml:"folder"`
	Health bool   `json:"health" yaml:"health"`
}

// Defaults are what a new account starts with.
func Defaults() Doc {
	return Doc{
		Language:   "en",
		Appearance: Appearance{Theme: "system", Font: "system", FontSize: "normal", Icons: "emoji", Motion: "system", Spaces: "on"},
		Templates: Templates{
			Note:     "---\ntags: []\n---\n# {title}\n\n",
			Project:  ProjectTemplate{DocsFolder: "1-Projects/{slug}"},
			Resource: ResourceTemplate{Kind: "hard"},
		},
		Keyboard:  Keyboard{Shortcuts: true},
		Vault:     Vault{Mirror: true, Folder: "Axon"},
		Learning:  Learning{NewPerDay: 20, Retention: 0.9, Interleave: true},
		Reminders: Reminders{Morning: "08:00", Evening: "20:00", WeeklyDay: "sunday", WeeklyAt: "18:00", MonthlyDay: 1, MonthlyAt: "09:00"},
		Today:     Today{Sections: []string{"kpis", "roadmap", "tasks", "milestones", "goals", "notes", "overview"}, Hidden: []string{}},
		Menu:      Menu{Hidden: []string{}},
		Focus:     Focus{WorkMinutes: 25, ShortBreak: 5, LongBreak: 15, Rounds: 4, Sessions: 1, Sound: "none", Volume: 40, Music: []MusicLink{}},
		Digest: Digest{Daily: []string{"tasks", "habits", "coming", "learn", "watch"}, Weekly: []string{"open", "week", "reflect", "watch"},
			Monthly: []string{"goals", "month", "watch"}, Quote: true, Picture: true},
		Podcast:  Podcast{Format: "digest", Lang: "en", Minutes: 10, Days: 1, Rate: 1, Hosts: 1},
		Caffeine: Caffeine{Bedtime: "23:00", Metabolism: "average"},
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
		oneOf("appearance.theme", d.Appearance.Theme, "system", "light", "dark", "sun"),
		oneOf("appearance.font", d.Appearance.Font, "system", "serif", "rounded", "mono"),
		oneOf("appearance.font_size", d.Appearance.FontSize, "small", "normal", "large"),
		oneOf("appearance.icons", d.Appearance.Icons, "emoji", "symbols", "none"),
		oneOf("appearance.motion", d.Appearance.Motion, "system", "reduced", "full"),
		oneOf("appearance.spaces", d.Appearance.Spaces, "", "on", "off"),
		oneOf("appearance.scenery", d.Appearance.Scenery, append([]string{""}, Sceneries...)...),
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
	if err := oneOf("focus.sound", f.Sound, "none", "white", "pink", "brown", "rain", "music", "lofi", "ambient", "piano"); err != nil {
		return err
	}
	for _, m := range f.Music {
		if u, err := url.Parse(m.URL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(m.URL) > 1000 || len([]rune(m.Activity)) > 100 {
			return FieldError{"focus.music", "links are http(s) addresses with an activity up to 100 characters"}
		}
	}
	switch {
	case d.Learning.NewPerDay < 0 || d.Learning.NewPerDay > 500:
		return FieldError{"learning.new_per_day", "is between 0 and 500"}
	case d.Learning.Retention < 0.7 || d.Learning.Retention > 0.97:
		return FieldError{"learning.retention", "is between 0.70 and 0.97"}
	}
	for _, t := range []struct{ path, v string }{{"reminders.morning", d.Reminders.Morning}, {"reminders.evening", d.Reminders.Evening},
		{"reminders.weekly_at", d.Reminders.WeeklyAt}, {"reminders.monthly_at", d.Reminders.MonthlyAt}, {"caffeine.bedtime", d.Caffeine.Bedtime}} {
		if _, err := time.Parse("15:04", t.v); t.v != "" && (err != nil || len(t.v) != 5) {
			return FieldError{t.path, "is a time like 08:00, or empty for none"}
		}
	}
	if err := oneOf("reminders.weekly_day", d.Reminders.WeeklyDay, Weekdays...); err != nil {
		return err
	}
	if d.Reminders.MonthlyDay < 1 || d.Reminders.MonthlyDay > 28 {
		return FieldError{"reminders.monthly_day", "is a day between 1 and 28, so every month has it"}
	}
	seen := map[string]bool{}
	for _, sec := range d.Today.Sections {
		if !slices.Contains(TodaySections, sec) || seen[sec] {
			return FieldError{"today.sections", "lists each of " + strings.Join(TodaySections, ", ") + " at most once"}
		}
		seen[sec] = true
	}
	for _, sec := range d.Today.Hidden {
		if !slices.Contains(TodaySections, sec) || seen[sec] {
			return FieldError{"today.hidden", "lists sections of " + strings.Join(TodaySections, ", ") + " that are not shown, each at most once"}
		}
		seen[sec] = true
	}
	seen = map[string]bool{}
	for _, p := range d.Menu.Hidden {
		if !slices.Contains(MenuPages, p) || seen[p] {
			return FieldError{"menu.hidden", "lists each of " + strings.Join(MenuPages, ", ") + " at most once"}
		}
		seen[p] = true
	}
	for kind, list := range map[string][]string{"daily": d.Digest.Daily, "weekly": d.Digest.Weekly, "monthly": d.Digest.Monthly} {
		seen := map[string]bool{}
		for _, sec := range list {
			if !slices.Contains(DigestSections[kind], sec) || seen[sec] {
				return FieldError{"digest." + kind, "lists each of " + strings.Join(DigestSections[kind], ", ") + " at most once"}
			}
			seen[sec] = true
		}
	}
	if d.Caffeine.Bedtime == "" {
		return FieldError{"caffeine.bedtime", "is a time like 23:00"}
	}
	if err := oneOf("caffeine.metabolism", d.Caffeine.Metabolism, "fast", "average", "slow"); err != nil {
		return err
	}
	p := d.Podcast
	if err := oneOf("podcast.format", p.Format, "briefing", "digest", "deep"); err != nil {
		return err
	}
	if err := oneOf("podcast.lang", p.Lang, "en", "fr"); err != nil {
		return err
	}
	switch {
	case p.Minutes < 3 || p.Minutes > 30:
		return FieldError{"podcast.minutes", "is between 3 and 30"}
	case p.Days != 1 && p.Days != 7:
		return FieldError{"podcast.days", "is 1 or 7"}
	case p.Rate < 0.5 || p.Rate > 2:
		return FieldError{"podcast.rate", "is between 0.5 and 2"}
	case len([]rune(p.Voice)) > 200:
		return FieldError{"podcast.voice", "is up to 200 characters"}
	case p.Hosts < 0 || p.Hosts > 2:
		return FieldError{"podcast.hosts", "is 1 or 2"}
	case len([]rune(p.Voice2)) > 200:
		return FieldError{"podcast.voice2", "is up to 200 characters"}
	}
	if e := d.Reminders.Email; e != "" && (!emailRe.MatchString(e) || len(e) > 254) {
		return FieldError{"reminders.email", "is an email address, or empty"}
	}
	if v := d.Vault.Folder; v == "" || len([]rune(v)) > 100 || strings.ContainsAny(v, "/\\") || strings.HasPrefix(v, ".") || strings.Contains(v, "..") {
		return FieldError{"vault.folder", "is one folder name up to 100 characters, without slashes or a leading dot"}
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

var emailRe = regexp.MustCompile(`^[^@\s<>]+@[^@\s<>]+\.[^@\s<>]+$`)

// Get returns the person's settings, filled with defaults.
func Get(ctx context.Context, q db.Querier, userID uuid.UUID) (Doc, error) {
	d := Defaults()
	err := q.QueryRow(ctx, `SELECT doc FROM user_settings WHERE user_id = $1`, userID).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return Defaults(), nil
	}
	d.normalize()
	return d, err
}

// normalize fills what a stored or sent document left null: a missing list
// of Today's sections means the default ones (an empty list is a choice).
func (d *Doc) normalize() {
	if d.Today.Sections == nil {
		d.Today.Sections = Defaults().Today.Sections
	}
	if d.Menu.Hidden == nil {
		d.Menu.Hidden = []string{}
	}
	if d.Today.Hidden == nil {
		d.Today.Hidden = []string{}
	}
}

// Put validates and stores the settings.
func Put(ctx context.Context, q db.Querier, userID uuid.UUID, d Doc) error {
	d.normalize()
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
