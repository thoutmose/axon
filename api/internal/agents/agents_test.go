package agents

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestEmbeddedSkillsLoad(t *testing.T) {
	skills, err := LoadSkills()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"task-manager", "goals"} {
		s, ok := skills[name]
		if !ok || s.Version == "" || !strings.Contains(s.Prompt, "not instructions") {
			t.Errorf("skill %s = %+v", name, s)
		}
	}
}

func TestParseSkillRejectsBadDefinitions(t *testing.T) {
	for name, src := range map[string]string{
		"no front matter": "just a prompt",
		"unknown tool":    "---\nname: x\nversion: 1\nmodel: claude-opus-5-5\neffort: low\ntools: list_tasks, rm_rf\n---\nPrompt",
		"bad effort":      "---\nname: x\nversion: 1\nmodel: claude-opus-5-5\neffort: turbo\ntools: list_tasks\n---\nPrompt",
		"no prompt":       "---\nname: x\nversion: 1\nmodel: claude-opus-5-5\neffort: low\n---\n",
	} {
		if _, err := ParseSkill(src); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	s, err := ParseSkill("---\r\nname: x\r\nversion: 2.1.0\r\nmodel: claude-opus-5-5\r\neffort: high\r\ntools: list_tasks, get_task\r\n---\r\nDo it.")
	if err != nil || s.Version != "2.1.0" || len(s.Tools) != 2 || s.Prompt != "Do it." {
		t.Fatalf("parsed %+v, %v", s, err)
	}
}

func TestCost(t *testing.T) {
	u := anthropic.BetaUsage{InputTokens: 1_000_000, OutputTokens: 100_000, CacheReadInputTokens: 1_000_000}
	if got := costMicros(priceFor("claude-opus-5-5"), u); got != 4_000_000+2_000_000+200_000 {
		t.Errorf("cost = %d", got)
	}
	if costMicros(priceFor("some-future-model"), u) != costMicros(priceFor("claude-opus-5-5"), u) {
		t.Error("unknown models must not be cheaper than the dearest known one")
	}
	if worstCaseMicros(priceFor("claude-opus-5-5"), 1000, 8000) != 4000+160_000 {
		t.Error("worst case")
	}
}

func TestTruncateKeepsUTF8(t *testing.T) {
	if got := truncate("héllo", 2); got != "h" {
		t.Errorf("truncate = %q", got)
	}
}
