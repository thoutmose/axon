package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// openAICompatible speaks the chat-completions wire format that OpenAI,
// Mistral, Gemini, DeepSeek, and local servers (Ollama, LM Studio,
// llama.cpp) share. It translates Axon's Messages-API requests and replies,
// so the tool loop, approval gate, and budget are the same for every provider.
type openAICompatible struct {
	baseURL, apiKey string
	maxTokensField  string // "max_completion_tokens" for OpenAI, "max_tokens" elsewhere
	http            *http.Client
}

func newOpenAICompatible(baseURL, apiKey, maxTokensField string) Provider {
	return openAICompatible{strings.TrimRight(baseURL, "/"), apiKey, maxTokensField, &http.Client{Timeout: 2 * time.Minute}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // a string, or text and image parts

	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// wire is the Messages-API JSON of a request, decoded generically.
type wire struct {
	Model     string `json:"model"`
	MaxTokens int64  `json:"max_tokens"`
	System    []struct {
		Text string `json:"text"`
	} `json:"system"`
	Messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string          `json:"type"`
			Text      string          `json:"text"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
			ToolUseID string          `json:"tool_use_id"`
			IsError   bool            `json:"is_error"`
			Content   []struct {
				Text string `json:"text"`
			} `json:"content"`
			Source struct {
				MediaType string `json:"media_type"`
				Data      string `json:"data"`
			} `json:"source"`
		} `json:"content"`
	} `json:"messages"`
	Tools []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema struct {
			Properties any      `json:"properties"`
			Required   []string `json:"required"`
		} `json:"input_schema"`
	} `json:"tools"`
}

func strp(s string) *string { return &s }

// translate turns a Messages-API request into a chat-completions body.
func (o openAICompatible) translate(p anthropic.BetaMessageNewParams) (map[string]any, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var w wire
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	var msgs []chatMessage
	if len(w.System) > 0 {
		var sys []string
		for _, s := range w.System {
			sys = append(sys, s.Text)
		}
		msgs = append(msgs, chatMessage{Role: "system", Content: strp(strings.Join(sys, "\n\n"))})
	}
	for _, m := range w.Messages {
		var text []string
		var calls []toolCall
		var images []map[string]any
		for _, c := range m.Content {
			switch c.Type {
			case "text":
				text = append(text, c.Text)
			case "image":
				images = append(images, map[string]any{"type": "image_url",
					"image_url": map[string]any{"url": "data:" + c.Source.MediaType + ";base64," + c.Source.Data}})
			case "tool_use":
				tc := toolCall{ID: c.ID, Type: "function"}
				tc.Function.Name = c.Name
				tc.Function.Arguments = string(c.Input)
				calls = append(calls, tc)
			case "tool_result":
				var out []string
				for _, part := range c.Content {
					out = append(out, part.Text)
				}
				result := strings.Join(out, "\n")
				if c.IsError {
					result = "Error: " + result
				}
				msgs = append(msgs, chatMessage{Role: "tool", ToolCallID: c.ToolUseID, Content: strp(result)})
			}
		}
		if len(images) > 0 {
			parts := []map[string]any{}
			if len(text) > 0 {
				parts = append(parts, map[string]any{"type": "text", "text": strings.Join(text, "\n\n")})
			}
			msgs = append(msgs, chatMessage{Role: m.Role, Content: append(parts, images...)})
		} else if len(text) > 0 || len(calls) > 0 {
			msg := chatMessage{Role: m.Role, ToolCalls: calls}
			if len(text) > 0 || m.Role == "user" {
				msg.Content = strp(strings.Join(text, "\n\n"))
			}
			msgs = append(msgs, msg)
		}
	}
	body := map[string]any{"model": w.Model, "messages": msgs, o.maxTokensField: w.MaxTokens}
	if len(w.Tools) > 0 {
		var tools []map[string]any
		for _, t := range w.Tools {
			props := t.InputSchema.Properties
			if props == nil {
				props = map[string]any{}
			}
			params := map[string]any{"type": "object", "properties": props}
			if len(t.InputSchema.Required) > 0 {
				params["required"] = t.InputSchema.Required
			}
			tools = append(tools, map[string]any{"type": "function", "function": map[string]any{
				"name": t.Name, "description": t.Description, "parameters": params,
			}})
		}
		body["tools"] = tools
	}
	return body, nil
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   *string    `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

func (o openAICompatible) New(ctx context.Context, p anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error) {
	body, err := o.translate(p)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	res, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		// The provider's message can echo input, so only the status is kept.
		return nil, fmt.Errorf("provider answered HTTP %d", res.StatusCode)
	}
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil || len(cr.Choices) == 0 {
		return nil, fmt.Errorf("provider sent an unreadable reply")
	}
	return toMessage(cr)
}

// toMessage builds the Messages-API reply the tool loop understands.
func toMessage(cr chatResponse) (*anthropic.BetaMessage, error) {
	ch := cr.Choices[0]
	var content []map[string]any
	if ch.Message.Content != nil && *ch.Message.Content != "" {
		content = append(content, map[string]any{"type": "text", "text": *ch.Message.Content})
	}
	for _, tc := range ch.Message.ToolCalls {
		input := json.RawMessage(tc.Function.Arguments)
		if !json.Valid(input) || len(bytes.TrimSpace(input)) == 0 {
			input = json.RawMessage(`{}`)
		}
		content = append(content, map[string]any{"type": "tool_use", "id": tc.ID, "name": tc.Function.Name, "input": input})
	}
	stop := map[string]string{"tool_calls": "tool_use", "length": "max_tokens", "content_filter": "refusal"}[ch.FinishReason]
	if stop == "" {
		stop = "end_turn"
	}
	if len(ch.Message.ToolCalls) > 0 {
		stop = "tool_use"
	}
	if content == nil {
		content = []map[string]any{}
	}
	b, _ := json.Marshal(map[string]any{
		"id": "chatcmpl", "type": "message", "role": "assistant", "model": cr.Model, "content": content, "stop_reason": stop,
		"usage": map[string]any{"input_tokens": cr.Usage.PromptTokens, "output_tokens": cr.Usage.CompletionTokens,
			"cache_creation_input_tokens": 0, "cache_read_input_tokens": 0},
	})
	var m anthropic.BetaMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
