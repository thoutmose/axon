package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// The Chef & Nutritionist (EP-90) reads the daily needs and proposes a meal
// plan, with kcal per meal, that becomes a note only once approved.
func TestChefPlansMealsAsANote(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()
	for _, c := range []string{"health_data", "health_ai"} {
		alice.mustDo(204, "PUT", "/v1/me/consents/"+c, []byte(`{"granted":true}`))
	}
	alice.mustDo(200, "PUT", "/v1/health/profile", []byte(fmt.Sprintf(
		`{"birth_year":%d,"sex":"female","height_cm":165,"weight_kg":60,"activity":"sedentary"}`, time.Now().Year()-40)))
	plan := `{"title":"Three days, 2 people, vegetarian","markdown":"## Monday\n- Breakfast (about 400 kcal, 18 g protein, 10 min): oats with yoghurt.\n- Dinner (600 kcal, 30 min): lentil dal with rice.\n\nEach day comes close to your 1,520 kcal reference.\n\n## Shopping list\n- [ ] Red lentils, 500 g\n\nEstimated cost: €24.","reason":"Vegetarian, under €30, 30 minutes a meal."}`
	fake.load(toolUse("daily_needs", `{}`), toolUse("draft_meal_plan", plan), finalText("Your plan waits for your approval; about €24."))
	r := alice.runAgent("chef", "Plan 3 days of breakfasts and dinners for 2 people, vegetarian, €30, 30 minutes a meal.")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 || !strings.Contains(r.Proposals[0].Summary, "Meal plans/") {
		t.Fatalf("run = %+v", r)
	}
	needs, _ := json.Marshal(fake.lastRequests()[1].Messages)
	if !bytes.Contains(needs, []byte(`\"kcal\":1520`)) {
		t.Fatalf("the agent did not get the daily needs: %s", needs)
	}
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	var list struct {
		Items []struct {
			Path string `json:"path"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?q=lentil", nil), &list)
	found := false
	for _, n := range list.Items {
		found = found || strings.HasPrefix(n.Path, "Meal plans/") && strings.HasSuffix(n.Path, "three-days-2-people-vegetarian.md")
	}
	if !found {
		t.Fatalf("notes = %+v", list.Items)
	}

	// The same plan approved again the same day gets its own file.
	fake.load(toolUse("draft_meal_plan", plan), finalText("Waiting for approval."))
	r = alice.runAgent("chef", "Same again, please.")
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)

	// A daily target under 1,200 kcal is still refused.
	fake.load(toolUse("draft_meal_plan", `{"title":"Cut","markdown":"Keep to 900 kcal a day.","reason":"x"}`), finalText("Done."))
	if r := alice.runAgent("chef", "Plan a week."); len(r.Proposals) != 0 {
		t.Fatalf("an unsafe plan was proposed: %+v", r.Proposals)
	}
}
