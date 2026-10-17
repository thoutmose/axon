package agents

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// Providers a person can choose. Everything but Anthropic uses the
// OpenAI-compatible chat API at a fixed address; "local" is any server the
// person runs (Ollama, LM Studio, llama.cpp) at an address they give.
var providerKinds = map[string]struct {
	baseURL, maxTokensField string
}{
	"anthropic": {},
	"openai":    {"https://api.openai.com/v1", "max_completion_tokens"},
	"mistral":   {"https://api.mistral.ai/v1", "max_tokens"},
	"gemini":    {"https://generativelanguage.googleapis.com/v1beta/openai", "max_tokens"},
	"deepseek":  {"https://api.deepseek.com/v1", "max_tokens"},
	"local":     {"", "max_tokens"},
}

// ProviderSettings is what a person chose; the key itself is never returned.
type ProviderSettings struct {
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	BaseURL     *string  `json:"base_url"`
	HasKey      bool     `json:"has_key"`
	InputPrice  *float64 `json:"input_price"`  // US$ per million input tokens
	OutputPrice *float64 `json:"output_price"` // US$ per million output tokens
}

var errProviderIncomplete = errors.New("provider settings incomplete")

// resolved is the provider, model, and prices a run will use.
type resolved struct {
	provider Provider
	name     string // provider kind, for status and records
	model    string
	price    price
	effort   bool // the provider understands Anthropic effort levels
}

func (r *Runner) providerSettings(ctx context.Context, q db.Querier, userID uuid.UUID) (*ProviderSettings, []byte, error) {
	var s ProviderSettings
	var sealed []byte
	err := q.QueryRow(ctx, `SELECT provider, model, base_url, api_key_sealed, input_price, output_price
		FROM agent_providers WHERE user_id = $1`, userID).Scan(&s.Provider, &s.Model, &s.BaseURL, &sealed, &s.InputPrice, &s.OutputPrice)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	s.HasKey = len(sealed) > 0
	return &s, sealed, err
}

// resolve picks the person's provider, or the server default, and the
// model they chose for the agent (EP-151).
func (r *Runner) resolve(ctx context.Context, q db.Querier, userID uuid.UUID, skill Skill) (*resolved, error) {
	out, err := r.resolveProvider(ctx, q, userID, skill)
	if err != nil {
		return nil, err
	}
	chosen, err := r.chosenModel(ctx, q, userID, skill.Name)
	if err != nil {
		return nil, err
	}
	if chosen != "" && chosen != out.model {
		out.model, out.price = chosen, priceFor(chosen)
	}
	return out, nil
}

func (r *Runner) resolveProvider(ctx context.Context, q db.Querier, userID uuid.UUID, skill Skill) (*resolved, error) {
	s, sealed, err := r.providerSettings(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		if r.provider == nil {
			return nil, ErrUnavailable
		}
		return &resolved{provider: r.provider, name: "anthropic", model: skill.Model, price: priceFor(skill.Model), effort: true}, nil
	}
	var key string
	if len(sealed) > 0 {
		if r.sealer == nil {
			return nil, errProviderIncomplete
		}
		b, err := r.sealer.Open(sealed)
		if err != nil {
			return nil, err
		}
		key = string(b)
	}
	out := &resolved{name: s.Provider, model: s.Model}
	switch s.Provider {
	case "anthropic":
		switch {
		case key != "":
			out.provider = r.newAnthropic(key)
		case r.provider != nil:
			out.provider = r.provider // the server's key or OAuth profile
		default:
			return nil, errProviderIncomplete
		}
		out.price, out.effort = priceFor(s.Model), true
	case "local":
		if s.BaseURL == nil {
			return nil, errProviderIncomplete
		}
		out.provider = r.newOpenAI(*s.BaseURL, key, providerKinds["local"].maxTokensField)
	default:
		if key == "" {
			return nil, errProviderIncomplete
		}
		k := providerKinds[s.Provider]
		out.provider = r.newOpenAI(k.baseURL, key, k.maxTokensField)
		out.price = priceFor(s.Model) // unknown models: the dearest known price, so budgets err on the safe side
	}
	if s.InputPrice != nil {
		out.price.input = *s.InputPrice
	}
	if s.OutputPrice != nil {
		out.price.output = *s.OutputPrice
	}
	return out, nil
}

// validBaseURL accepts an http(s) address for a local model server.
func validBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && len(raw) <= 300
}

// validModel accepts provider model names such as "llama3.1:8b" or "org/model".
func validModel(m string) bool {
	return m != "" && len(m) <= 100 && !strings.ContainsAny(m, " \t\n")
}

func providerNames() []string {
	out := make([]string, 0, len(providerKinds))
	for k := range providerKinds {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// testParams is the cheapest possible request: no personal data.
func testParams(model string) anthropic.BetaMessageNewParams {
	return anthropic.BetaMessageNewParams{
		Model:     model,
		MaxTokens: 16,
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Reply with the word OK."))},
	}
}

// InputError is a settings value a person can correct.
type InputError struct{ Msg string }

func (e InputError) Error() string { return e.Msg }

// ProviderInput sets a person's provider. APIKey absent keeps the stored
// key; an empty string removes it.
type ProviderInput struct {
	Provider    string   `json:"provider" yaml:"provider"`
	Model       string   `json:"model" yaml:"model"`
	BaseURL     *string  `json:"base_url" yaml:"base_url,omitempty"`
	APIKey      *string  `json:"api_key" yaml:"api_key,omitempty"`
	InputPrice  *float64 `json:"input_price" yaml:"input_price,omitempty"`
	OutputPrice *float64 `json:"output_price" yaml:"output_price,omitempty"`
}

// SetProvider validates and stores a person's provider; the key is sealed.
func (r *Runner) SetProvider(ctx context.Context, q db.Querier, userID uuid.UUID, in ProviderInput) error {
	in.Model = strings.TrimSpace(in.Model)
	switch {
	case !slices.Contains(providerNames(), in.Provider):
		return InputError{"Choose a provider: " + strings.Join(providerNames(), ", ") + "."}
	case !validModel(in.Model):
		return InputError{"Enter the model name your provider uses, like claude-opus-5-5 or llama3.1:8b."}
	case in.Provider == "local" && (in.BaseURL == nil || !validBaseURL(*in.BaseURL)):
		return InputError{"A local model needs its server address, like http://192.168.1.20:11434/v1."}
	case in.Provider != "local" && in.BaseURL != nil:
		return InputError{"Only a local provider takes a server address."}
	case in.APIKey != nil && len(*in.APIKey) > 500:
		return InputError{"That key is too long."}
	case in.APIKey != nil && *in.APIKey != "" && r.sealer == nil:
		return InputError{"This server cannot store keys: set AXON_ENCRYPTION_KEY first."}
	}
	for _, p := range []*float64{in.InputPrice, in.OutputPrice} {
		if p != nil && (*p < 0 || *p > 1000) {
			return InputError{"Prices are US dollars per million tokens, from 0 to 1,000."}
		}
	}
	keep := in.APIKey == nil
	var sealed []byte
	if in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "" {
		sealed = r.sealer.Seal([]byte(strings.TrimSpace(*in.APIKey)))
	}
	_, err := q.Exec(ctx, `INSERT INTO agent_providers (user_id, provider, model, base_url, api_key_sealed, input_price, output_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET provider = $2, model = $3, base_url = $4,
			api_key_sealed = CASE WHEN $8 THEN agent_providers.api_key_sealed ELSE $5 END,
			input_price = $6, output_price = $7, updated_at = now()`,
		userID, in.Provider, in.Model, in.BaseURL, sealed, in.InputPrice, in.OutputPrice, keep)
	return err
}

// SetBudget stores a person's monthly agent budget.
func SetBudget(ctx context.Context, q db.Querier, userID uuid.UUID, cents int) error {
	if cents < 0 || cents > 100000 {
		return InputError{"The monthly budget is between $0 and $1,000."}
	}
	_, err := q.Exec(ctx, `INSERT INTO agent_settings (user_id, monthly_budget_cents) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET monthly_budget_cents = $2, updated_at = now()`, userID, cents)
	return err
}

// SetVideoSummaries turns prose summaries of videos and episodes on or off (EP-151).
func SetVideoSummaries(ctx context.Context, q db.Querier, userID uuid.UUID, on bool) error {
	_, err := q.Exec(ctx, `INSERT INTO agent_settings (user_id, video_summaries) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET video_summaries = $2, updated_at = now()`, userID, on)
	return err
}

// AIConfig is the agent part of a person's settings; the key is never read back.
type AIConfig struct {
	Provider           *ProviderSettings `json:"provider"`
	MonthlyBudgetCents int               `json:"monthly_budget_cents"`
	VideoSummaries     bool              `json:"video_summaries"`
}

// GetAIConfig returns a person's provider (nil for the server default) and budget.
func (r *Runner) GetAIConfig(ctx context.Context, q db.Querier, userID uuid.UUID) (AIConfig, error) {
	s, _, err := r.providerSettings(ctx, q, userID)
	if err != nil {
		return AIConfig{}, err
	}
	c := AIConfig{Provider: s, MonthlyBudgetCents: 200}
	err = q.QueryRow(ctx, `SELECT coalesce((SELECT monthly_budget_cents FROM agent_settings WHERE user_id = $1), 200),
		coalesce((SELECT video_summaries FROM agent_settings WHERE user_id = $1), false)`, userID).Scan(&c.MonthlyBudgetCents, &c.VideoSummaries)
	return c, err
}
