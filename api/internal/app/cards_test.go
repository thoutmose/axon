package app

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// noisyPNG is a picture that does not compress, so it exceeds JSON body limits.
func noisyPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(16)) //nolint:gosec // test pixels, not secrets
	_, _ = r.Read(img.Pix)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type flashcardJSON struct {
	ID       string `json:"id"`
	Deck     string `json:"deck"`
	State    int    `json:"state"`
	Due      string `json:"due"`
	Reps     int    `json:"reps"`
	HasImage bool   `json:"has_image"`
}

func TestFlashcardsScheduleReviewIllustrateAndExportToAnki(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var note struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/permanent/spacing.md","content":"---\ntype: permanent\n---\n# Spacing\n\nSpaced practice beats massed practice."}`)), &note)

	var c flashcardJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Why space practice?","back":"Forgetting a little makes retrieval stronger.","deck":"Learning science","source_id":"`+note.ID+`"}`)), &c)
	if c.State != 1 || c.Reps != 0 || c.Deck != "Learning science" {
		t.Fatalf("card = %+v", c)
	}
	alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"What is interleaving?","back":"Mixing related topics.","deck":"Learning science"}`))
	alice.mustDo(422, "POST", "/v1/flashcards", []byte(`{"front":" "}`))
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+note.ID+"/links?direction=incoming", nil), []byte("derived_from")) {
		t.Fatal("the note does not show its card")
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/decks", nil), []byte(`{"name":"Learning science","total":2,"due":2,"new":2}`)) {
		t.Fatal("deck counts wrong")
	}

	// Due cards come with what each rating would do; Good moves to the 10-minute step.
	var due struct {
		Items []struct {
			ID   string `json:"id"`
			Next struct {
				Again, Hard, Good, Easy time.Time
			} `json:"next"`
		} `json:"items"`
		DueTotal int `json:"due_total"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/due?deck=Learning%20science", nil), &due)
	if due.DueTotal != 2 || len(due.Items) != 2 {
		t.Fatalf("due = %+v", due)
	}
	n := due.Items[0].Next
	if !n.Again.Before(n.Hard) || !n.Hard.Before(n.Good) || !n.Good.Before(n.Easy) {
		t.Fatalf("previews out of order: %+v", n)
	}
	var reviewed flashcardJSON
	_ = json.Unmarshal(alice.mustDo(200, "POST", "/v1/flashcards/"+due.Items[0].ID+"/review", []byte(`{"rating":3,"duration_ms":4200}`)), &reviewed)
	dueAt, _ := time.Parse(time.RFC3339Nano, reviewed.Due)
	if reviewed.Reps != 1 || reviewed.State != 1 || time.Until(dueAt) < 9*time.Minute || time.Until(dueAt) > 11*time.Minute {
		t.Fatalf("after Good = %+v", reviewed)
	}
	alice.mustDo(422, "POST", "/v1/flashcards/"+c.ID+"/review", []byte(`{"rating":5}`))
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/due", nil), &due)
	if due.DueTotal != 1 {
		t.Fatalf("a reviewed card is still due: %d", due.DueTotal)
	}

	// Illustrate a card with a real-sized picture (more than the JSON body limit).
	big := noisyPNG(600, 400)
	if len(big) < 200_000 {
		t.Fatalf("fixture too small: %d", len(big))
	}
	if code, out := alice.do("PUT", "/v1/flashcards/"+c.ID+"/image", big, "Content-Type", "image/png"); code != 204 {
		t.Fatalf("image upload = %d %s", code, out)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/flashcards/"+c.ID, nil), []byte(`"has_image":true`)) {
		t.Fatal("has_image not set")
	}
	if code, _ := alice.do("PUT", "/v1/flashcards/"+c.ID+"/image", []byte("nope"), "Content-Type", "image/png"); code != 422 {
		t.Fatalf("garbage image = %d", code)
	}
	alice.mustDo(200, "PATCH", "/v1/flashcards/"+c.ID, []byte(`{"back":"Forgetting a little, then retrieving, strengthens memory."}`))

	// The Anki package: a collection with both notes and the picture.
	code, apkg := alice.do("GET", "/v1/flashcards/export.apkg?deck=Learning%20science", nil)
	if code != 200 {
		t.Fatalf("export = %d", code)
	}
	zr, err := zip.NewReader(bytes.NewReader(apkg), int64(len(apkg)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		var b bytes.Buffer
		_, _ = b.ReadFrom(rc)
		_ = rc.Close()
		files[f.Name] = b.Bytes()
	}
	if !strings.Contains(string(files["media"]), ".png") || len(files["0"]) == 0 {
		t.Fatalf("media missing: %s", files["media"])
	}
	path := filepath.Join(t.TempDir(), "collection.anki2")
	_ = os.WriteFile(path, files["collection.anki2"], 0o600)
	dbc, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbc.Close() }()
	var notesN, cardsN, ver int
	var flds string
	_ = dbc.QueryRow(`SELECT count(*) FROM notes`).Scan(&notesN)
	_ = dbc.QueryRow(`SELECT count(*) FROM cards`).Scan(&cardsN)
	_ = dbc.QueryRow(`SELECT ver FROM col`).Scan(&ver)
	_ = dbc.QueryRow(`SELECT flds FROM notes WHERE flds LIKE '%Why space%'`).Scan(&flds)
	if notesN != 2 || cardsN != 2 || ver != 11 || !strings.Contains(flds, `<img src="axon-`) {
		t.Fatalf("collection: notes %d cards %d ver %d flds %q", notesN, cardsN, ver, flds)
	}

	alice.mustDo(204, "DELETE", "/v1/flashcards/"+c.ID+"/image", nil)
	alice.mustDo(404, "GET", "/v1/flashcards/"+c.ID+"/image", nil)
	alice.mustDo(200, "GET", "/v1/flashcards?deck=Learning%20science", nil)
	alice.mustDo(204, "DELETE", "/v1/flashcards/"+c.ID, nil)
	alice.mustDo(404, "GET", "/v1/flashcards/"+c.ID, nil)
}

func TestTeacherReadsOnlyStudyNotesAndProposesCards(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.enableAgents()
	var perm, fleeting struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/permanent/retrieval.md","content":"---\ntype: permanent\n---\n# Retrieval\n\nTesting yourself strengthens memory."}`)), &perm)
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"0-Inbox/diary.md","content":"---\ntype: fleeting\n---\n# Diary\n\nPrivate thoughts."}`)), &fleeting)

	fake.load(
		toolUse("list_study_notes", `{}`, "read_study_note", `{"id":"`+fleeting.ID+`"}`, "read_study_note", `{"id":"`+perm.ID+`"}`),
		toolUse("propose_cards", `{"cards":[{"front":"What does diary say?","back":"x","note_id":"`+fleeting.ID+`"}],"reason":"x"}`,
			"propose_cards", `{"cards":[{"front":"Why test yourself?","back":"Retrieval strengthens memory.","note_id":"`+perm.ID+`"}],"reason":"From your note on retrieval."}`),
		finalText("I proposed one card."),
	)
	r := alice.runAgent("teacher", "Make cards from my notes")
	if r.Run.Status != "completed" || len(r.Proposals) != 1 {
		t.Fatalf("run = %+v", r)
	}
	want := []string{"list_study_notes", "read_study_note:error", "read_study_note", "propose_cards:error", "propose_cards:proposed"}
	for i, w := range want {
		if r.Run.ToolsCalled[i] != w {
			t.Fatalf("tools_called = %v", r.Run.ToolsCalled)
		}
	}
	sent, _ := json.Marshal(fake.lastRequests()[1].Messages)
	if bytes.Contains(sent, []byte("Private thoughts")) || bytes.Contains(sent, []byte("diary.md")) {
		t.Fatal("a fleeting note reached the model")
	}
	if !bytes.Contains(sent, []byte("Testing yourself strengthens memory")) {
		t.Fatal("the permanent note did not reach the model")
	}
	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/decks", nil), []byte(`"name":"Retrieval","total":1`)) {
		t.Fatal("approved card not in a deck named after its note")
	}
}

// Reviews of cards in long-term memory keep the scheduler's prediction, so
// the person can see whether it holds for them.
func TestFlashcardCalibration(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	var cal struct {
		Reviews    int        `json:"reviews"`
		Predicted  float64    `json:"predicted"`
		Remembered float64    `json:"remembered"`
		Target     float64    `json:"target"`
		Buckets    []struct{} `json:"buckets"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/calibration", nil), &cal)
	if cal.Reviews != 0 || cal.Target != 0.9 || cal.Buckets == nil {
		t.Fatalf("no reviews yet = %+v", cal)
	}
	var c struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/flashcards", []byte(`{"front":"Capital of Peru?","back":"Lima"}`)), &c)
	for _, rating := range []string{"4", "3", "1"} { // Easy: into long-term memory; then two predicted reviews
		alice.mustDo(200, "POST", "/v1/flashcards/"+c.ID+"/review", []byte(`{"rating":`+rating+`}`))
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/flashcards/calibration", nil), &cal)
	if cal.Reviews != 2 || cal.Remembered != 0.5 || cal.Predicted <= 0.9 || len(cal.Buckets) != 1 {
		t.Fatalf("after two predicted reviews = %+v", cal)
	}
}
