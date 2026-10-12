package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

type paraJSON struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Linked int    `json:"linked"`
}

// Areas and resources are entities: anything can be filed under an area,
// archiving moves them to the archive, and another person cannot see them.
func TestAreasAndResourcesFileWorkAndArchive(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	var health, rust paraJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/areas", []byte(`{"title":"Health","description":"Sleep 7 h, move daily."}`)), &health)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/resources", []byte(`{"title":"Rust","url":"https://www.rust-lang.org/"}`)), &rust)
	alice.mustDo(422, "POST", "/v1/areas", []byte(`{"title":"  "}`))
	alice.mustDo(422, "POST", "/v1/resources", []byte(`{"title":"x","url":"javascript:alert(1)"}`))
	alice.mustDo(400, "GET", "/v1/areas?status=gone", nil)

	var goal struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/goals", []byte(`{"title":"Run 10 km"}`)), &goal)
	alice.link(goal.ID, "in_area", health.ID)
	alice.link(rust.ID, "in_area", health.ID) // a resource can be filed too
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/areas/"+health.ID, nil), &health)
	if health.Linked != 2 {
		t.Fatalf("linked = %d", health.Linked)
	}

	// Archive, list, restore, delete.
	if code, body := alice.do("PATCH", "/v1/resources/"+rust.ID, []byte(`{"status":"archived"}`), "If-Match", alice.etag("/v1/resources/"+rust.ID)); code != 200 {
		t.Fatalf("archive = %d %s", code, body)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/resources?status=archived", nil), []byte(rust.ID)) {
		t.Error("archived resource not listed in the archive")
	}
	if bytes.Contains(alice.mustDo(200, "GET", "/v1/resources", nil), []byte(rust.ID)) {
		t.Error("archived resource still listed as active")
	}
	if code, _ := alice.do("PATCH", "/v1/areas/"+health.ID, []byte(`{"title":"x"}`), "If-Match", `"stale"`); code != 412 {
		t.Fatalf("stale PATCH = %d", code)
	}
	alice.mustDo(404, "GET", "/v1/areas/"+rust.ID, nil) // a resource is not an area
	bob.mustDo(404, "GET", "/v1/areas/"+health.ID, nil)
	bob.mustDo(404, "DELETE", "/v1/resources/"+rust.ID, nil)
	if bytes.Contains(bob.mustDo(200, "GET", "/v1/areas", nil), []byte(health.ID)) {
		t.Error("bob sees alice's area")
	}
	alice.mustDo(204, "DELETE", "/v1/areas/"+health.ID, nil)
	alice.mustDo(200, "GET", "/v1/goals/"+goal.ID, nil) // what was filed under it stays
	alice.mustDo(204, "DELETE", "/v1/resources/"+rust.ID, nil)
	alice.mustDo(404, "GET", "/v1/resources/"+rust.ID, nil)
}
