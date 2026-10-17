package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Jarvis (EP-153) works only through the other agents: "I'm reading The
// Martian" goes to the Media agent, whose proposal (the book, being read,
// with its note) waits in Jarvis's run for approval. Other agents cannot
// ask, and Jarvis cannot ask itself.
func TestJarvisAsksTheOtherAgents(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(204, "PUT", "/v1/me/consents/ai_agents", []byte(`{"granted":true}`))

	fake.load(
		toolUse("ask_agent", `{"agent":"jarvis","instruction":"loop"}`),
		toolUse("ask_agent", `{"agent":"media","instruction":"Add The Martian by Andy Weir (2011, 369 pages, science fiction) as being read, with its note."}`),
		toolUse("suggest_media", `{"kind":"book","title":"The Martian","creators":"Andy Weir","year":2011,"total":369,"genres":["Science fiction"],"status":"doing","note":true,"reason":"You said you are reading it."}`),
		finalText("I proposed adding The Martian as being read, with its note."),
		finalText("The Media agent proposed The Martian, being read, with its note: approve it below."),
	)
	r := alice.runAgent("jarvis", "I'm reading a new book, The Martian, add it to the books tracker.")
	if r.Run.Status != "completed" || r.Run.Proposals != 1 || r.Reply == nil || !strings.Contains(*r.Reply, "approve") {
		t.Fatalf("run = %+v, reply %v", r.Run, r.Reply)
	}
	tools := strings.Join(r.Run.ToolsCalled, ",")
	if !strings.Contains(tools, "ask_agent:media") || !strings.Contains(tools, "media/suggest_media:proposed") {
		t.Fatalf("tools = %s", tools)
	}
	reqs := fake.lastRequests()
	if len(reqs) != 5 || !strings.Contains(reqs[0].System[0].Text, "- media: ") || strings.Contains(reqs[0].System[0].Text, "- jarvis: ") {
		t.Fatalf("Jarvis's prompt lacks the agents: %d requests", len(reqs))
	}
	// Asking itself came back as an error to the model, not a loop.
	if !strings.Contains(string(must(json.Marshal(reqs[1].Messages))), "no agent called") {
		t.Error("asking itself was not refused")
	}
	// The media agent ran with its own tools, not Jarvis's.
	for _, tl := range reqs[2].Tools {
		if tl.OfTool != nil && tl.OfTool.Name == "ask_agent" {
			t.Fatal("the media agent was given ask_agent")
		}
	}

	alice.mustDo(200, "POST", "/v1/proposals/"+r.Proposals[0].ID+"/approve", nil)
	var books struct {
		Items []mediaJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/media?kind=book", nil), &books)
	if len(books.Items) != 1 || books.Items[0].Title != "The Martian" || books.Items[0].Status != "doing" || books.Items[0].Total == nil || *books.Items[0].Total != 369 {
		t.Fatalf("books = %+v", books.Items)
	}
	n := alice.mustDo(200, "GET", "/v1/notes?q=Martian", nil)
	if !bytes.Contains(n, []byte("Zettel/literature/")) || !bytes.Contains(n, []byte("the-martian.md")) {
		t.Fatalf("the book's note = %s", n)
	}
	links := alice.mustDo(200, "GET", "/v1/entities/"+books.Items[0].ID+"/links?direction=incoming", nil)
	if !bytes.Contains(links, []byte(`"relates_to"`)) {
		t.Fatalf("the note is not linked to the book: %s", links)
	}

	// Another agent cannot ask.
	fake.load(toolUse("ask_agent", `{"agent":"media","instruction":"x"}`), finalText("ok"))
	if r := alice.runAgent("task-manager", "Ask the media agent."); !strings.Contains(strings.Join(r.Run.ToolsCalled, ","), "denied:ask_agent") {
		t.Fatalf("task manager tools = %v", r.Run.ToolsCalled)
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
