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

// resolve picks the person's provider, or the server default.
func (r *Runner) resolve(ctx context.Context, q db.Querier, userID uuid.UUID, skill Skill) (*resolved, error) {
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
		Model:     anthropic.Model(model),
		MaxTokens: 16,
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Reply with the word OK."))},
	}
}
