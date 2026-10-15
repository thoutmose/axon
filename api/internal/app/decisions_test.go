package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// exportedJSON returns the JSON file of a person's export archive.
func exportedJSON(t *testing.T, c *client) []byte {
	t.Helper()
	out := c.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name == "axon-export.json" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			return b
		}
	}
	t.Fatal("no axon-export.json in the export")
	return nil
}

type decisionJSON struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Expected    string  `json:"expected"`
	Probability float64 `json:"probability"`
	DecidedOn   string  `json:"decided_on"`
	ReviewOn    string  `json:"review_on"`
	Happened    *bool   `json:"happened"`
	ReviewedOn  *string `json:"reviewed_on"`
	Due         bool    `json:"due"`
	Lessons     string  `json:"lessons"`
	Choice      string  `json:"choice"`
	Context     string  `json:"context"`
	Outcome     string  `json:"outcome"`
}

// The decision journal (EP-114): a forecast made once and never changed,
// reviewed on its day, and scored against what happened; never someone else's.
func TestDecisionJournal(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")
	today := time.Now().UTC()
	day := func(n int) string { return today.AddDate(0, 0, n).Format(time.DateOnly) }

	var d decisionJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/decisions", []byte(`{"title":"Move the API to Go","context":"The Python API is slow.",
		"choice":"Rewrite it in Go","expected":"p99 latency under 100 ms","probability":0.8,"decided_on":"`+day(-30)+`","review_on":"`+day(-1)+`"}`)), &d)
	if d.ID == "" || d.Probability != 0.8 || d.Happened != nil || !d.Due {
		t.Fatalf("created = %+v", d)
	}
	var later decisionJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/decisions", []byte(`{"title":"Take the CKA","expected":"Pass on the first try","probability":0.6,"review_on":"`+day(60)+`"}`)), &later)
	if later.DecidedOn != day(0) || later.Due {
		t.Fatalf("decided today = %+v", later)
	}

	// Mistakes are refused with a reason.
	for _, body := range []string{
		`{"title":"","expected":"x","probability":0.5,"review_on":"` + day(5) + `"}`,
		`{"title":"x","expected":"","probability":0.5,"review_on":"` + day(5) + `"}`,
		`{"title":"x","expected":"x","probability":1,"review_on":"` + day(5) + `"}`,
		`{"title":"x","expected":"x","review_on":"` + day(5) + `"}`,
		`{"title":"x","expected":"x","probability":0.5,"review_on":"` + day(0) + `"}`,
		`{"title":"x","expected":"x","probability":0.5,"decided_on":"` + day(1) + `","review_on":"` + day(5) + `"}`,
	} {
		alice.mustDo(422, "POST", "/v1/decisions", []byte(body))
	}

	// The forecast is fixed; the rest can be edited.
	alice.mustDo(422, "PATCH", "/v1/decisions/"+d.ID, []byte(`{"probability":0.95}`))
	alice.mustDo(422, "PATCH", "/v1/decisions/"+d.ID, []byte(`{"expected":"p99 under a second"}`))
	_ = json.Unmarshal(alice.mustDo(200, "PATCH", "/v1/decisions/"+d.ID, []byte(`{"choice":"Rewrite the hot paths in Go"}`)), &d)
	if d.Choice != "Rewrite the hot paths in Go" || d.Probability != 0.8 {
		t.Fatalf("edited = %+v", d)
	}

	// Due ones first; then the review.
	var list struct {
		Items []decisionJSON `json:"items"`
		Today string         `json:"today"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/decisions?status=due", nil), &list)
	if len(list.Items) != 1 || list.Items[0].ID != d.ID {
		t.Fatalf("due = %+v", list.Items)
	}
	alice.mustDo(422, "POST", "/v1/decisions/"+d.ID+"/review", []byte(`{"outcome":"Forgot to say"}`))
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/decisions/"+d.ID+"/review", []byte(`{"happened":false,"outcome":"p99 is 140 ms","lessons":"The database was the bottleneck."}`)), &d)
	if d.Happened == nil || *d.Happened || d.ReviewedOn == nil || d.Due || d.Lessons == "" {
		t.Fatalf("reviewed = %+v", d)
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/decisions?status=open", nil), &list)
	if len(list.Items) != 1 || list.Items[0].ID != later.ID {
		t.Fatalf("open = %+v", list.Items)
	}

	// The score: one 80% forecast that did not come true.
	var c struct {
		Count   int      `json:"count"`
		Brier   *float64 `json:"brier"`
		Buckets []struct {
			From  float64 `json:"from"`
			Count int     `json:"count"`
		} `json:"buckets"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/decisions/calibration", nil), &c)
	if c.Count != 1 || c.Brier == nil || *c.Brier < 0.6399 || *c.Brier > 0.6401 || len(c.Buckets) != 1 || c.Buckets[0].From != 0.8 {
		t.Fatalf("calibration = %+v", c)
	}

	// In the export; never Bob's.
	if ex := exportedJSON(t, alice); !bytes.Contains(ex, []byte(`"decisions"`)) || !bytes.Contains(ex, []byte("The database was the bottleneck.")) {
		t.Error("the export misses the decision")
	}
	bob.mustDo(404, "PATCH", "/v1/decisions/"+d.ID, []byte(`{"title":"Mine"}`))
	bob.mustDo(404, "POST", "/v1/decisions/"+d.ID+"/review", []byte(`{"happened":true}`))
	bob.mustDo(404, "DELETE", "/v1/decisions/"+d.ID, nil)
	_ = json.Unmarshal(bob.mustDo(200, "GET", "/v1/decisions", nil), &list)
	if len(list.Items) != 0 {
		t.Fatalf("bob sees %+v", list.Items)
	}
	alice.mustDo(204, "DELETE", "/v1/decisions/"+d.ID, nil)
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/decisions", nil), &list)
	if len(list.Items) != 1 {
		t.Fatalf("after delete = %+v", list.Items)
	}
}
