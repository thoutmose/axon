package app

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

type roadmapJSON struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	Done        int    `json:"done"`
	Total       int    `json:"total"`
	Skills      []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Section string `json:"section"`
		Status  string `json:"status"`
	} `json:"skills"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
}

func TestRoadmapsImportManualEditingAndProgress(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")

	src := alice.mustDo(200, "GET", "/v1/roadmaps/sources", nil)
	if !bytes.Contains(src, []byte(`"license_url":"https://github.com/ossu/computer-science/blob/master/LICENSE"`)) {
		t.Fatalf("sources = %s", src)
	}
	if code, out := alice.do("POST", "/v1/roadmaps/import", []byte(`{"source":"roadmap.sh"}`)); code != 422 || !bytes.Contains(out, []byte("may not be redistributed")) {
		t.Fatalf("roadmap.sh import = %d %s", code, out)
	}
	var ossu roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/import", []byte(`{"source":"ossu"}`)), &ossu)
	if ossu.Source != "ossu" || len(ossu.Skills) != 63 || len(ossu.Edges) != 24 || ossu.Total != 63 ||
		!bytes.Contains([]byte(ossu.Attribution), []byte("MIT License")) {
		t.Fatalf("ossu = %d skills, %d edges, %+v", len(ossu.Skills), len(ossu.Edges), ossu.Attribution)
	}

	// A manual roadmap with prerequisites.
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Japanese N4"}`)), &rm)
	var kana, kanji, grammar struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"Kana","section":"Basics","url":"https://example.org/kana"}`)), &kana)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"300 kanji","section":"Vocabulary","requires":["`+kana.ID+`"]}`)), &kanji)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"Te-form","section":"Grammar"}`)), &grammar)
	alice.link(grammar.ID, "requires", kana.ID)
	alice.mustDo(422, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"x","requires":["`+ossu.Skills[0].ID+`"]}`))
	alice.mustDo(422, "POST", "/v1/roadmaps/"+rm.ID+"/skills", []byte(`{"title":"x","url":"javascript:alert(1)"}`))
	alice.mustDo(422, "PATCH", "/v1/skills/"+kana.ID, []byte(`{"status":"mastered"}`))

	alice.mustDo(200, "PATCH", "/v1/skills/"+kana.ID, []byte(`{"status":"done"}`))
	alice.mustDo(200, "PATCH", "/v1/skills/"+grammar.ID, []byte(`{"status":"skipped"}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID, nil), &rm)
	if rm.Done != 1 || rm.Total != 2 || len(rm.Edges) != 2 {
		t.Fatalf("progress = %d/%d, edges %d", rm.Done, rm.Total, len(rm.Edges))
	}
	alice.mustDo(200, "PATCH", "/v1/roadmaps/"+rm.ID, []byte(`{"description":"Pass JLPT N4 in July."}`))

	// Roadmaps connect to goals like everything else.
	var g goalJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Pass JLPT N4"}`)), &g)
	alice.link(rm.ID, "supports", g.ID)
	alice.link(kanji.ID, "supports", g.ID)

	alice.mustDo(204, "DELETE", "/v1/skills/"+grammar.ID, nil)
	alice.mustDo(404, "PATCH", "/v1/skills/"+grammar.ID, []byte(`{"status":"done"}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/roadmaps", nil), []byte("Japanese N4")) {
		t.Fatal("roadmap not listed")
	}
	alice.mustDo(204, "DELETE", "/v1/roadmaps/"+rm.ID, nil)
	alice.mustDo(404, "GET", "/v1/entities/"+kanji.ID, nil)
	alice.mustDo(404, "GET", "/v1/roadmaps/"+rm.ID, nil)
}

func TestLargeRoadmapStaysFast(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var rm roadmapJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/roadmaps", []byte(`{"title":"Everything"}`)), &rm)
	u := userID(t, alice)
	_, err := pool.Exec(context.Background(), `
		WITH ids AS (SELECT gen_random_uuid() AS id, n FROM generate_series(1, 500) n),
		     ent AS (INSERT INTO entities (id, user_id, type, title) SELECT id, $1, 'skill', 'Skill ' || n FROM ids RETURNING id),
		     sk AS (INSERT INTO skills (id, roadmap_id, position, section) SELECT id, $2, n, 'Part ' || (n / 50) FROM ids RETURNING id)
		INSERT INTO links (id, user_id, source_id, relation, target_id)
		SELECT gen_random_uuid(), $1, a.id, 'requires', b.id FROM ids a JOIN ids b ON b.n = a.n - 1`, u, rm.ID)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/roadmaps/"+rm.ID, nil), &rm)
	if len(rm.Skills) != 500 || len(rm.Edges) != 499 {
		t.Fatalf("skills %d edges %d", len(rm.Skills), len(rm.Edges))
	}
	if d := time.Since(start); d > 300*time.Millisecond {
		t.Errorf("a 500-skill roadmap took %s", d)
	}
}
