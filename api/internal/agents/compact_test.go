package agents

import (
	"encoding/json"
	"strings"
	"testing"
)

// Compact results say the same in fewer characters (EP-152): empty values
// go, times are to the minute, records become rows; ids, numbers, false,
// and zero stay.
func TestCompactToolResults(t *testing.T) {
	in := `{"items":[
		{"id":"a1","title":"Plan the garden","status":"next","due_on":null,"notes":"","tags":[],"done":false,"position":0,"updated_at":"2026-10-17T11:36:12.716117Z"},
		{"id":"a2","title":"Book the train","status":"next","due_on":"2026-10-20","notes":"window seat","tags":["travel"],"done":false,"position":1,"updated_at":"2026-10-17T11:37:00+02:00"},
		{"id":"a3","title":"Call Ana","status":"waiting","due_on":null,"notes":"","tags":[],"done":true,"position":2,"updated_at":"2026-10-17T11:38:59Z"}
	],"next_cursor":null,"total":3}`
	out, raw := compact([]byte(in))
	if raw != len(in) || len(out) >= len(in)*3/4 {
		t.Fatalf("%d → %d characters: %s", raw, len(out), out)
	}
	var got struct {
		Items struct {
			Cols []string `json:"cols"`
			Rows [][]any  `json:"rows"`
		} `json:"items"`
		Total     int  `json:"total"`
		NextCurse *any `json:"next_cursor"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 || len(got.Items.Rows) != 3 || strings.Join(got.Items.Cols, ",") != "done,due_on,id,notes,position,status,tags,title,updated_at" {
		t.Fatalf("compact = %s", out)
	}
	s := string(out)
	for _, want := range []string{`"a1"`, `false`, `true`, `"2026-10-17T11:36Z"`, `"2026-10-17T11:37+02:00"`, `"window seat"`, `0,"next"`} {
		if !strings.Contains(s, want) {
			t.Errorf("lost %s: %s", want, s)
		}
	}
	if strings.Contains(s, `next_cursor`) || strings.Contains(s, `.716117`) {
		t.Errorf("kept noise: %s", s)
	}
	// Two records, or records that differ, stay as they are.
	if out, _ := compact([]byte(`[{"a":1},{"b":2},{"c":3}]`)); string(out) != `[{"a":1},{"b":2},{"c":3}]` {
		t.Errorf("unlike records = %s", out)
	}
	if out, _ := compact([]byte(`not json`)); string(out) != `not json` {
		t.Errorf("not JSON = %s", out)
	}
}
