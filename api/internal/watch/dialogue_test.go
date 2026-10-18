package watch

import (
	"errors"
	"testing"
	"time"
)

// The conversation keeps lines of known hosts about known stories, cleaned
// for a voice, and is refused when a story is left out.
func TestParseDialogueCoversEveryStory(t *testing.T) {
	ep := Episode{Lang: "fr", Chapters: []Chapter{{Title: "A"}, {Title: "B"}}}
	answer := `Voici : {"turns": [
		{"host": 0, "chapter": -1, "text": "Salut Hugo ! On a deux sujets aujourd'hui."},
		{"host": 1, "chapter": 0, "text": "On commence par **le premier**. Plus sur https://example.com"},
		{"host": 0, "chapter": 0, "text": "Attends, c'est énorme."},
		{"host": 2, "chapter": 1, "text": "Un troisième animateur ?"},
		{"host": 1, "chapter": 7, "text": "Un sujet qui n'existe pas."},
		{"host": 0, "chapter": 1, "text": "Et le second sujet."},
		{"host": 1, "chapter": -1, "text": "À demain !"}]}`
	turns, err := ParseDialogue(answer, ep, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Markdown goes, and a sentence that only points to a link says nothing.
	if len(turns) != 5 || turns[1].Text != "On commence par le premier." {
		t.Fatalf("turns = %+v", turns)
	}
	if _, err := ParseDialogue(`{"turns": [{"host": 0, "chapter": 0, "text": "Seulement A."}, {"host": 1, "chapter": -1, "text": "Fin."}, {"host": 0, "chapter": -1, "text": "Fin."}]}`, ep, 2); !errors.Is(err, ErrNotDialogue) {
		t.Errorf("a story left out = %v", err)
	}
	if _, err := ParseDialogue("Je ne peux pas.", ep, 2); !errors.Is(err, ErrNotDialogue) {
		t.Errorf("prose = %v", err)
	}
}

// The same episode built again later keeps its written conversation; other
// words or another number of hosts need a new one.
func TestTalkKeyFollowsWhatTheHostsSay(t *testing.T) {
	ep := Episode{Lang: "en", Format: "brief", Intro: "Hello.", Chapters: []Chapter{{Title: "Rust"}}, From: time.Now()}
	later := ep
	later.From, later.To = ep.From.Add(time.Hour), ep.From.Add(2*time.Hour)
	if talkKey(ep, 2) != talkKey(later, 2) {
		t.Error("rebuilt episode got a new key")
	}
	other := ep
	other.Intro = "Good evening."
	if talkKey(ep, 2) == talkKey(ep, 1) || talkKey(ep, 2) == talkKey(other, 2) {
		t.Error("key ignores hosts or words")
	}
}
