package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/auth"
	"github.com/thoutmose/axon/api/internal/demo"
)

type overviewJSON struct {
	Days int `json:"days"`
	KPIs []struct {
		ID       string `json:"id"`
		Value    int    `json:"value"`
		Previous *int   `json:"previous"`
		Href     string `json:"href"`
	} `json:"kpis"`
	Series []struct {
		ID     string            `json:"id"`
		Points []json.RawMessage `json:"points"`
	} `json:"series"`
	Activity []struct {
		Day   string `json:"day"`
		Value int    `json:"value"`
	} `json:"activity"`
	Tables   struct {
		DueSoon  []struct{ Title string } `json:"due_soon"`
		Learning []struct{ Title string } `json:"learning"`
	} `json:"tables"`
	Skills struct {
		Nodes []struct {
			ID        string  `json:"id"`
			Type      string  `json:"type"`
			Title     string  `json:"title"`
			Kind      *string `json:"kind"`
			Status    *string `json:"status"`
			RoadmapID *string `json:"roadmap_id"`
		} `json:"nodes"`
		Edges []json.RawMessage `json:"edges"`
	} `json:"skills"`
}

func TestOverviewHasKPIsChartsTablesAndASkillsNetwork(t *testing.T) {
	e := newEnvWith(t, auth.Config{Demo: true})
	if _, err := demo.Reset(context.Background(), pool, time.Now()); err != nil {
		t.Fatal(err)
	}
	c := e.client()
	c.mustDo(204, "POST", "/v1/auth/demo", nil)

	var o overviewJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/dashboard/overview", nil), &o)
	kpi := map[string]int{}
	for _, k := range o.KPIs {
		if k.Href == "" {
			t.Errorf("%s has no link to its items", k.ID)
		}
		kpi[k.ID] = k.Value
	}
	for _, id := range []string{"tasks_done", "habit_consistency", "goal_progress", "skills_done", "learning", "notes_written", "projects_active", "focus_minutes"} {
		if _, ok := kpi[id]; !ok {
			t.Errorf("missing KPI %s", id)
		}
	}
	if kpi["notes_written"] < 8 || kpi["projects_active"] != 3 || kpi["learning"] < 3 {
		t.Errorf("kpis = %v", kpi)
	}
	// A lived-in year: activity on many days, not all on the day of the seed (EP-99).
	active := 0
	for _, d := range o.Activity {
		if d.Value > 0 {
			active++
		}
	}
	if len(o.Activity) != 365 || active < 60 {
		t.Errorf("activity on %d of %d days", active, len(o.Activity))
	}
	// A month of check-ins with a few misses: high, but honest.
	if c := kpi["habit_consistency"]; c < 50 || c > 100 {
		t.Errorf("habit consistency = %d", c)
	}
	if o.Days != 30 || len(o.Series) != 4 || len(o.Series[0].Points) != 30 || len(o.Activity) != 365 {
		t.Fatalf("series %d, points %d, activity %d", len(o.Series), len(o.Series[0].Points), len(o.Activity))
	}
	if len(o.Tables.DueSoon) == 0 || len(o.Tables.Learning) < 3 {
		t.Fatalf("tables = %+v", o.Tables)
	}
	kinds, types := map[string]int{}, map[string]int{}
	for _, n := range o.Skills.Nodes {
		types[n.Type]++
		if n.Kind != nil {
			kinds[*n.Kind]++
		}
	}
	// Skills only: hard and soft, with their prerequisites; no resources, goals, projects, or notes.
	if kinds["hard"] == 0 || kinds["soft"] == 0 || len(types) != 1 || types["skill"] == 0 || len(o.Skills.Edges) == 0 {
		t.Fatalf("network kinds %v types %v edges %d", kinds, types, len(o.Skills.Edges))
	}

	// A roadmap from before skills were competencies, with one skill per
	// course, shows as the skill those courses teach, in one node.
	var legacy struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Old library import"}`)), &legacy)
	for _, course := range []struct{ title, status string }{{"MIT 6.006 Introduction to Algorithms", "learning"}, {"Algorithms", "done"}} {
		var sk struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/roadmaps/"+legacy.ID+"/skills", []byte(`{"title":"`+course.title+`"}`)), &sk)
		c.mustDo(200, "PATCH", "/v1/skills/"+sk.ID, []byte(`{"status":"`+course.status+`"}`))
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/dashboard/overview", nil), &o)
	merged := 0
	for _, n := range o.Skills.Nodes {
		if n.RoadmapID != nil && *n.RoadmapID == legacy.ID {
			merged++
			if n.Title != "Data structures and algorithms" || n.Status == nil || *n.Status != "done" {
				t.Fatalf("legacy node = %+v", n)
			}
		}
	}
	if merged != 1 {
		t.Fatalf("%d nodes for the legacy roadmap, want 1", merged)
	}

	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/dashboard/overview?days=90", nil), &o)
	if o.Days != 90 || len(o.Series[0].Points) != 90 {
		t.Fatalf("days=90 gave %d", len(o.Series[0].Points))
	}
	c.mustDo(400, "GET", "/v1/dashboard/overview?days=3", nil)

	// Skills can be reclassified.
	skill := o.Skills.Nodes[0].ID
	c.mustDo(200, "PATCH", "/v1/skills/"+skill, []byte(`{"kind":"soft"}`))
	c.mustDo(422, "PATCH", "/v1/skills/"+skill, []byte(`{"kind":"medium"}`))
}
