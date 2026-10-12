package settings

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.yaml.in/yaml/v4"

	"github.com/thoutmose/axon/api/internal/agents"
	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/httpx"
)

// Handlers serves the settings endpoints.
type Handlers struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	runner *agents.Runner
}

// NewHandlers returns the settings endpoints; runner may be nil (tests).
func NewHandlers(pool *pgxpool.Pool, log *slog.Logger, runner *agents.Runner) *Handlers {
	return &Handlers{db: pool, log: log, runner: runner}
}

// Routes mounts the endpoints; the caller wraps them in auth.RequireSession,
// because the file can carry an AI key.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/v1/me/settings", h.get)
	r.Put("/v1/me/settings", h.put)
	r.Get("/v1/me/settings.yaml", h.getYAML)
	r.Put("/v1/me/settings.yaml", h.putYAML)
}

// AIFile is the AI part of the YAML file.
type AIFile struct {
	MonthlyBudgetCents *int                  `yaml:"monthly_budget_cents,omitempty"`
	Provider           *agents.ProviderInput `yaml:"provider,omitempty"`
}

// File is the YAML document.
type File struct {
	Version int `yaml:"version"`
	Doc     `yaml:",inline"`
	AI      AIFile `yaml:"ai"`
}

func (h *Handlers) fail(w http.ResponseWriter, err error) {
	var f FieldError
	var ie agents.InputError
	switch {
	case errors.As(err, &f):
		httpx.Problem(w, http.StatusUnprocessableEntity, f.Error())
	case errors.As(err, &ie):
		httpx.Problem(w, http.StatusUnprocessableEntity, "ai: "+ie.Msg)
	default:
		h.log.Error("settings request failed", "err", err)
		httpx.Problem(w, http.StatusInternalServerError, "Something went wrong. Try again.")
	}
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	d, err := Get(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := map[string]any{"language": d.Language, "appearance": d.Appearance, "templates": d.Templates, "keyboard": d.Keyboard, "focus": d.Focus, "gamification": d.Gamification, "vault": d.Vault}
	if h.runner != nil {
		ai, err := h.runner.GetAIConfig(r.Context(), h.db, who.UserID)
		if err != nil {
			h.fail(w, err)
			return
		}
		out["ai"] = ai
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handlers) put(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	d := Defaults() // sections left out keep their defaults
	if err := httpx.Decode(r, &d); err != nil {
		httpx.Problem(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := Put(r.Context(), h.db, who.UserID, d); err != nil {
		h.fail(w, err)
		return
	}
	h.get(w, r)
}

const yamlHeader = `# Axon settings. Edit and upload this file under Account → Preferences,
# or PUT it to /api/v1/me/settings.yaml. Unknown keys are rejected.
# ai.provider.api_key is write-only: add it to set a key; exports never contain it.
`

func (h *Handlers) getYAML(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	d, err := Get(r.Context(), h.db, who.UserID)
	if err != nil {
		h.fail(w, err)
		return
	}
	f := File{Version: 1, Doc: d}
	if h.runner != nil {
		ai, err := h.runner.GetAIConfig(r.Context(), h.db, who.UserID)
		if err != nil {
			h.fail(w, err)
			return
		}
		f.AI.MonthlyBudgetCents = &ai.MonthlyBudgetCents
		if p := ai.Provider; p != nil {
			f.AI.Provider = &agents.ProviderInput{Provider: p.Provider, Model: p.Model, BaseURL: p.BaseURL, InputPrice: p.InputPrice, OutputPrice: p.OutputPrice}
		}
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f); err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="axon-settings.yaml"`)
	_, _ = io.WriteString(w, yamlHeader+buf.String())
}

// lineOf finds the line of a dotted path in a YAML document, or 0.
func lineOf(root *yaml.Node, path string) int {
	n := root
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	line := 0
	for _, key := range strings.Split(path, ".") {
		found := false
		for i := 0; n.Kind == yaml.MappingNode && i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				line, n, found = n.Content[i].Line, n.Content[i+1], true
				break
			}
		}
		if !found {
			return line
		}
	}
	return line
}

func (h *Handlers) putYAML(w http.ResponseWriter, r *http.Request) {
	who, _ := auth.FromContext(r.Context())
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if err != nil {
		httpx.Problem(w, http.StatusBadRequest, "The file could not be read.")
		return
	}
	var root yaml.Node
	if err := yaml.Unmarshal(body, &root); err != nil {
		httpx.Problem(w, http.StatusUnprocessableEntity, "The YAML is not valid: "+strings.TrimPrefix(err.Error(), "yaml: "))
		return
	}
	f := File{Doc: Defaults()}
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		httpx.Problem(w, http.StatusUnprocessableEntity, "The settings file has a problem: "+strings.TrimPrefix(err.Error(), "yaml: "))
		return
	}
	if f.Version != 1 {
		httpx.Problem(w, http.StatusUnprocessableEntity, fmt.Sprintf("line %d: version must be 1", lineOf(&root, "version")))
		return
	}
	if err := f.Validate(); err != nil {
		var fe FieldError
		errors.As(err, &fe)
		httpx.Problem(w, http.StatusUnprocessableEntity, fmt.Sprintf("line %d: %s", lineOf(&root, fe.Path), fe.Error()))
		return
	}
	if who.Demo && f.AI.Provider != nil {
		httpx.Problem(w, http.StatusForbidden, "The demo account cannot set AI providers.")
		return
	}
	err = pgx.BeginFunc(r.Context(), h.db, func(tx pgx.Tx) error {
		if err := Put(r.Context(), tx, who.UserID, f.Doc); err != nil {
			return err
		}
		if f.AI.MonthlyBudgetCents != nil {
			if err := agents.SetBudget(r.Context(), tx, who.UserID, *f.AI.MonthlyBudgetCents); err != nil {
				return err
			}
		}
		if f.AI.Provider != nil && h.runner != nil {
			return h.runner.SetProvider(r.Context(), tx, who.UserID, *f.AI.Provider)
		}
		return nil
	})
	var ie agents.InputError
	if errors.As(err, &ie) {
		httpx.Problem(w, http.StatusUnprocessableEntity, fmt.Sprintf("line %d: ai: %s", lineOf(&root, "ai"), ie.Msg))
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.get(w, r)
}
