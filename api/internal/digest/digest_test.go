package digest

import (
	"bytes"
	"encoding/json"
	"image/png"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func sample() Digest {
	id := uuid.New()
	return Digest{
		Kind: Daily, Lang: "en", Title: "Your day", Greeting: "Good morning, Ada", Label: "Wednesday October 14",
		Summary: "2 tasks", Theme: "focus", URL: "/today", Seed: "s",
		Stats: []Stat{{Emoji: "✅", Label: "Tasks done", Value: "3"}},
		Sections: []Section{{Key: "To do today", Emoji: "✅", Title: "To do today", More: 2, Items: []Item{
			{N: 1, ID: id, Type: "task", Text: `Fix <script>alert(1)</script> & *bold*`, Detail: "due today", URL: "/tasks?open=" + id.String()},
			{Type: "card", Text: "3 cards to review", URL: "/cards/review"},
		}}},
		Quote: Quote{Text: "Well begun is half done.", Author: "Aristotle"},
	}
}

func TestEveryStringIsTranslated(t *testing.T) {
	for k := range en {
		if fr[k] == "" {
			t.Errorf("no French for %q", k)
		}
	}
	for k := range fr {
		if _, ok := en[k]; !ok {
			t.Errorf("French %q has no English", k)
		}
	}
	for _, q := range quotes {
		if q.fr == "" || q.en == "" || q.author == "" || len(q.themes) == 0 {
			t.Errorf("incomplete quote %+v", q)
		}
	}
}

func TestQuotesFollowTheThemeAndTheDay(t *testing.T) {
	q := pick("learning", nil, "a", "en")
	found := false
	for _, c := range quotes {
		if c.en == q.Text {
			found = slices.Contains(c.themes, "learning")
		}
	}
	if !found {
		t.Fatalf("%q is not a learning quote", q.Text)
	}
	if pick("learning", nil, "a", "en") != q {
		t.Fatal("the same day gives another quote")
	}
	if fq := pick("learning", nil, "a", "fr"); fq.Author != q.Author || fq.Text == q.Text {
		t.Fatalf("French quote = %+v", fq)
	}
	if got := Domains([]string{"Run a marathon", "Write a novel"}); strings.Join(got, ",") != "writing,health" {
		t.Fatalf("domains = %v", got)
	}
	// With a domain, the quote matches the theme and the domain when one does.
	q = pick("rest", []string{"health"}, "b", "en")
	if q.Author != "John Lubbock" && q.Author != "Anne Lamott" {
		t.Fatalf("rest + health = %+v", q)
	}
}

func TestRenderers(t *testing.T) {
	d := sample()
	base := "https://axon.example"
	if s := HTML(d, base, "scene"); strings.Contains(s, "<script>") || !strings.Contains(s, "&lt;script&gt;") ||
		!strings.Contains(s, `src="cid:scene"`) || !strings.Contains(s, base+"/today") || !strings.Contains(s, "and 2 more") {
		t.Fatalf("HTML:\n%s", s)
	}
	if s := Telegram(d, base); strings.Contains(s, "<script>") || !strings.Contains(s, "<code>1</code>") || !strings.Contains(s, "Reply") {
		t.Fatalf("Telegram:\n%s", s)
	}
	if s := Text(d, ""); !strings.Contains(s, "1. Fix <script>") || strings.Contains(s, "http") {
		t.Fatalf("Text:\n%s", s)
	}
	for name, v := range map[string]any{"slack": Slack(d, base, true), "discord": Discord(d, base, true), "teams": Teams(d, base)} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(name, err)
		}
		if bytes.Contains(b, []byte("<script>")) && name == "slack" {
			t.Errorf("%s keeps markup: %s", name, b)
		}
	}
	if b, _ := json.Marshal(Discord(d, base, false)); !bytes.Contains(b, []byte(`\\*bold\\*`)) || !bytes.Contains(b, []byte(`"parse":[]`)) {
		t.Fatalf("Discord does not escape or allows mentions: %s", b)
	}
	// A long section is cut for Discord's 1024-character fields.
	long := sample()
	for range 80 {
		long.Sections[0].Items = append(long.Sections[0].Items, Item{Text: strings.Repeat("x", 30)})
	}
	b, _ := json.Marshal(Discord(long, "", false))
	var out struct {
		Embeds []struct {
			Fields []struct {
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"embeds"`
	}
	_ = json.Unmarshal(b, &out)
	for _, f := range out.Embeds[0].Fields {
		if len(f.Value) > 1024 {
			t.Fatalf("field of %d characters", len(f.Value))
		}
	}
}

func TestScene(t *testing.T) {
	a, b := Scene(sample()), Scene(sample())
	if !bytes.Equal(a, b) {
		t.Fatal("the same digest draws another picture")
	}
	img, err := png.Decode(bytes.NewReader(a))
	if err != nil || img.Bounds().Dx() != SceneWidth || img.Bounds().Dy() != SceneHeight {
		t.Fatalf("scene: %v %v", err, img.Bounds())
	}
	other := sample()
	other.Theme = "reflection"
	if bytes.Equal(a, Scene(other)) {
		t.Fatal("themes look the same")
	}
}
