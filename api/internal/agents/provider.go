package agents

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// Provider sends one Messages API request. The Anthropic client implements
// it; tests and evals use a scripted fake, so no test ever calls the network.
type Provider interface {
	New(ctx context.Context, p anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error)
}

// anthropicProvider calls Claude through the official SDK.
type anthropicProvider struct{ client anthropic.Client }

// NewAnthropic returns the Anthropic provider for an API key.
func NewAnthropic(apiKey string) Provider {
	return anthropicProvider{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

// NewAnthropicFromEnvironment uses the SDK's own credential resolution,
// including an OAuth profile made on the server with "ant auth login", so
// no API key has to be stored anywhere.
func NewAnthropicFromEnvironment() Provider {
	return anthropicProvider{client: anthropic.NewClient()}
}

func (a anthropicProvider) New(ctx context.Context, p anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error) {
	// A safety refusal is re-served by a fallback model inside the same call.
	p.Betas = append(p.Betas, anthropic.AnthropicBetaServerSideFallback2026_07_01)
	p.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
	return a.client.Beta.Messages.New(ctx, p)
}

// price is US dollars per million tokens (Anthropic list prices).
type price struct{ input, output, cacheWrite, cacheRead float64 }

var prices = map[string]price{
	"claude-opus-5-5":   {input: 4, output: 20, cacheWrite: 5, cacheRead: 0.20},
	"claude-sonnet-5-5": {input: 2, output: 10, cacheWrite: 2.5, cacheRead: 0.20},
	"claude-haiku-5-5":  {input: 0.10, output: 0.50, cacheWrite: 0.125, cacheRead: 0.01},
}

// priceFor returns the list price of a known Anthropic model. Unknown models
// are priced as the dearest known one, never as free: a person sets their
// provider's prices (or 0 for a local model) in their settings.
func priceFor(model string) price {
	if p, ok := prices[model]; ok {
		return p
	}
	return prices["claude-opus-5-5"]
}

// costMicros is the cost of one response in millionths of a dollar.
func costMicros(p price, u anthropic.BetaUsage) int64 {
	return int64(float64(u.InputTokens)*p.input + float64(u.OutputTokens)*p.output +
		float64(u.CacheCreationInputTokens)*p.cacheWrite + float64(u.CacheReadInputTokens)*p.cacheRead)
}

// worstCaseMicros bounds the cost of the next call: its input is at most the
// history so far, its output at most maxTokens.
func worstCaseMicros(p price, inputTokens, maxTokens int64) int64 {
	return int64(float64(inputTokens)*p.input + float64(maxTokens)*p.output)
}
