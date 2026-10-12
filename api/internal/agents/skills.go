package agents

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/thoutmose/axon/api/internal/privacy"
)

//go:embed skills/*/SKILL.md
var skillFiles embed.FS

// Skill is a versioned agent definition: its instructions, model, and the
// only tools it may call.
type Skill struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Model       string   `json:"-"`
	Effort      string   `json:"-"`
	Tools       []string `json:"tools"`
	// Consents are needed on top of ai_agents (health coaches need health_data and health_ai).
	Consents []string `json:"consents"`
	// Safety "health" turns on the red-flag referral and the unsafe-advice guards.
	Safety string `json:"safety,omitempty"`
	Prompt string `json:"-"`
}

// ParseSkill reads a SKILL.md: "---" front matter of key: value lines, then the prompt.
func ParseSkill(src string) (Skill, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	rest, ok := strings.CutPrefix(src, "---\n")
	front, body, ok2 := strings.Cut(rest, "\n---\n")
	if !ok || !ok2 {
		return Skill{}, fmt.Errorf("SKILL.md needs front matter between --- lines")
	}
	var s Skill
	for _, line := range strings.Split(front, "\n") {
		k, v, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "name":
			s.Name = v
		case "title":
			s.Title = v
		case "version":
			s.Version = v
		case "description":
			s.Description = v
		case "model":
			s.Model = v
		case "effort":
			s.Effort = v
		case "consents":
			for _, c := range strings.Split(v, ",") {
				if c = strings.TrimSpace(c); c != "" {
					s.Consents = append(s.Consents, c)
				}
			}
		case "safety":
			s.Safety = v
		case "tools":
			for _, t := range strings.Split(v, ",") {
				if t = strings.TrimSpace(t); t != "" {
					s.Tools = append(s.Tools, t)
				}
			}
		}
	}
	s.Prompt = strings.TrimSpace(body)
	switch {
	case s.Name == "" || s.Version == "" || s.Model == "" || s.Prompt == "":
		return Skill{}, fmt.Errorf("skill %q needs name, version, model, and a prompt", s.Name)
	case !slices.Contains([]string{"low", "medium", "high", "xhigh", "max"}, s.Effort):
		return Skill{}, fmt.Errorf("skill %q: effort must be low, medium, high, xhigh, or max", s.Name)
	}
	if s.Safety != "" && s.Safety != "health" {
		return Skill{}, fmt.Errorf("skill %q: safety must be health", s.Name)
	}
	for _, c := range s.Consents {
		if !slices.ContainsFunc(privacy.Purposes, func(p privacy.Purpose) bool { return p.ID == c }) {
			return Skill{}, fmt.Errorf("skill %q needs unknown consent %q", s.Name, c)
		}
	}
	if s.Consents == nil {
		s.Consents = []string{}
	}
	for _, t := range s.Tools {
		if _, ok := registry[t]; !ok {
			return Skill{}, fmt.Errorf("skill %q lists unknown tool %q", s.Name, t)
		}
	}
	return s, nil
}

// LoadSkills parses every embedded skill; a broken skill stops the server
// from starting rather than failing at run time.
func LoadSkills() (map[string]Skill, error) {
	out := map[string]Skill{}
	paths, _ := fs.Glob(skillFiles, "skills/*/SKILL.md")
	for _, p := range paths {
		b, err := skillFiles.ReadFile(p)
		if err != nil {
			return nil, err
		}
		s, err := ParseSkill(string(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out[s.Name] = s
	}
	return out, nil
}
