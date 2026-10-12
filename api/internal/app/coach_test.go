package app

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/agents"
)

// Coach agents need their own consents, read health logs only while both
// health consents hold, answer red flags without the model, and never let
// unsafe advice through, in a reply or in a proposal.
func TestCoachAgentsConsentAndSafety(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()

	code, body := alice.do("POST", "/v1/agents/personal-trainer/runs", []byte(`{"instruction":"Plan my week"}`))
	if code != 403 || !bytes.Contains(body, []byte("health_data")) {
		t.Fatalf("without health consent: %d %s", code, body)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/health_data", []byte(`{"granted":true}`))
	if code, body = alice.do("POST", "/v1/agents/nutritionist/runs", []byte(`{"instruction":"Look at my meals"}`)); code != 403 || !bytes.Contains(body, []byte("health_ai")) {
		t.Fatalf("without health_ai: %d %s", code, body)
	}
	alice.mustDo(204, "PUT", "/v1/me/consents/health_ai", []byte(`{"granted":true}`))
	alice.mustDo(201, "POST", "/v1/health/workouts", []byte(`{"done_on":"`+today()+`","kind":"cardio","title":"Easy run by the river","minutes":30,"effort":4}`))

	// With both consents, the log reaches the model through the tool.
	fake.load(
		toolUse("training_log", `{"days":14}`),
		toolUse("create_task", `{"title":"Workout: 30 min easy run","reason":"Keep the habit with an easy session."}`),
		finalText("You ran once this week at an easy effort. I proposed one more easy run."),
	)
	r := alice.runAgent("personal-trainer", "Plan my next session")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 || !slices.Contains(r.Run.ToolsCalled, "training_log") {
		t.Fatalf("run = %+v", r)
	}
	saw := false
	for _, req := range fake.lastRequests() {
		b, _ := json.Marshal(req.Messages)
		saw = saw || bytes.Contains(b, []byte("Easy run by the river"))
	}
	if !saw {
		t.Fatal("the training log did not reach the coach")
	}

	// A red flag is answered here, with no model call at all.
	fake.load(finalText("should never be used"))
	r = alice.runAgent("personal-trainer", "I get chest pain when I run, how should I train?")
	if r.Run.Status != "completed" || r.Reply == nil || *r.Reply != agents.Referral || len(fake.lastRequests()) != 0 {
		t.Fatalf("red flag: %+v, %d model calls", r, len(fake.lastRequests()))
	}
	r = alice.runAgent("nutritionist", "Je suis enceinte, que dois-je manger ?")
	if r.Reply == nil || *r.Reply != agents.Referral || len(fake.lastRequests()) != 0 {
		t.Fatalf("French red flag: %+v", r)
	}

	// Unsafe advice never gets through: the reply is replaced, the proposal refused.
	fake.load(
		toolUse("create_task", `{"title":"Eat 900 kcal a day","reason":"Lose weight fast."}`),
		finalText("Aim for 800 kcal per day for a month."),
	)
	r = alice.runAgent("nutritionist", "Help me eat better")
	if r.Reply == nil || *r.Reply != agents.UnsafeReply || len(r.Proposals) != 0 || !slices.Contains(r.Run.ToolsCalled, "create_task:unsafe") {
		t.Fatalf("unsafe: %+v", r)
	}

	// Withdrawing health_ai stops the tool even inside a run.
	alice.mustDo(204, "PUT", "/v1/me/consents/health_ai", []byte(`{"granted":false}`))
	alice.mustDo(403, "POST", "/v1/agents/personal-trainer/runs", []byte(`{"instruction":"Plan my week"}`))
	if out := alice.mustDo(200, "GET", "/v1/agents", nil); !bytes.Contains(out, []byte(`"consents":["health_data","health_ai"]`)) {
		t.Fatalf("agents list does not show the consents: %s", truncateBody(out))
	}
}

func today() string { return time.Now().UTC().Format(time.DateOnly) }

func truncateBody(b []byte) string {
	if len(b) > 300 {
		return string(b[:300]) + "…"
	}
	return string(b)
}
