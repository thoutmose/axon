package server

import "testing"

func TestFeatureOfARoute(t *testing.T) {
	for route, want := range map[string]string{
		"/v1/watch/items/{id}":     "watch",
		"/v1/me/agent-models":      "agent",
		"/v1/agent-runs/{id}":      "agent",
		"/v1/notes/file":           "note",
		"/v1/roadmaps/{id}/plan":   "roadmap",
		"/v1/tasks":                "tasks",
		"/v1/me":                   "me",
		"unmatched":                "unmatched",
		"/":                        "other",
		"/v1/budget/report":        "budget",
		"/v1/calendar/events":      "calendar",
		"/v1/households/{id}/list": "household",
	} {
		if got := Feature(route); got != want {
			t.Errorf("Feature(%q) = %q, want %q", route, got, want)
		}
	}
}
