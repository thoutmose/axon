package agents

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestTranslateToChatCompletions(t *testing.T) {
	var prior anthropic.BetaMessage
	_ = json.Unmarshal([]byte(`{"id":"m","type":"message","role":"assistant","model":"x","stop_reason":"tool_use",
		"content":[{"type":"text","text":"Looking."},{"type":"tool_use","id":"call_1","name":"list_tasks","input":{"status":"inbox"}}],
		"usage":{"input_tokens":1,"output_tokens":1,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}`), &prior)
	tool := anthropic.BetaToolParam{Name: "list_tasks", Description: anthropic.String("List"),
		InputSchema: anthropic.BetaToolInputSchemaParam{Properties: map[string]any{"status": map[string]any{"type": "string"}}}}
	p := anthropic.BetaMessageNewParams{
		Model: "llama3.1:8b", MaxTokens: 100,
		System: []anthropic.BetaTextBlockParam{{Text: "Be helpful."}},
		Tools:  []anthropic.BetaToolUnionParam{{OfTool: &tool}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Triage")),
			prior.ToParam(),
			anthropic.NewBetaUserMessage(anthropic.NewBetaToolResultBlock("call_1", "<data>[]</data>", false)),
		},
	}
	body, err := openAICompatible{maxTokensField: "max_tokens"}.translate(p)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(body)
	got := string(b)
	for _, want := range []string{
		`"max_tokens":100`, `"model":"llama3.1:8b"`,
		`{"role":"system","content":"Be helpful."}`,
		`{"role":"user","content":"Triage"}`,
		`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"list_tasks","arguments":"{\"status\":\"inbox\"}"}}]`,
		`{"role":"tool","content":"`, `data`, `"tool_call_id":"call_1"}`,
		`"parameters":{"properties":{"status":{"type":"string"}},"type":"object"}`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in\n%s", want, got)
		}
	}
}

func TestChatReplyBecomesMessage(t *testing.T) {
	var cr chatResponse
	_ = json.Unmarshal([]byte(`{"model":"mistral-large","choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,
		"tool_calls":[{"id":"c1","type":"function","function":{"name":"get_task","arguments":"{\"id\":\"x\"}"}},
		{"id":"c2","type":"function","function":{"name":"list_tasks","arguments":"not json"}}]}}],
		"usage":{"prompt_tokens":120,"completion_tokens":30}}`), &cr)
	m, err := toMessage(cr)
	if err != nil {
		t.Fatal(err)
	}
	if m.StopReason != anthropic.BetaStopReasonToolUse || len(m.Content) != 2 || m.Usage.InputTokens != 120 || m.Usage.OutputTokens != 30 {
		t.Fatalf("message = %+v", m)
	}
	if m.Content[0].AsAny().(anthropic.BetaToolUseBlock).Name != "get_task" || m.Content[1].JSON.Input.Raw() != "{}" {
		t.Fatalf("tool calls = %+v", m.Content)
	}
	var filtered chatResponse
	_ = json.Unmarshal([]byte(`{"choices":[{"finish_reason":"content_filter","message":{"role":"assistant","content":""}}]}`), &filtered)
	if m, _ := toMessage(filtered); m.StopReason != anthropic.BetaStopReasonRefusal {
		t.Fatalf("content filter = %s", m.StopReason)
	}
}

// A photo goes to an OpenAI-style vision model as an image_url part (EP-89).
func TestImagesBecomeImageURLParts(t *testing.T) {
	p := anthropic.BetaMessageNewParams{
		Model: "gpt-5", MaxTokens: 100,
		Messages: []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(
			anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{Data: "AAAA", MediaType: anthropic.BetaBase64ImageSourceMediaTypeImageJPEG}),
			anthropic.NewBetaTextBlock("What is on the plate?"),
		)},
	}
	body, err := openAICompatible{maxTokensField: "max_completion_tokens"}.translate(p)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(body)
	want := `{"role":"user","content":[{"text":"What is on the plate?","type":"text"},{"image_url":{"url":"data:image/jpeg;base64,AAAA"},"type":"image_url"}]}`
	if !strings.Contains(string(b), want) {
		t.Fatalf("got\n%s", b)
	}
}
