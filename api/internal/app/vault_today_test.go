package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The vault in step with what came on 2026-10-17 (EP-156): a drawing made
// in Axon reaches the vault byte for byte, its shapes with it, and comes
// back when edited there; a note with maths goes both ways untouched.
func TestVaultCarriesDrawingsAndMaths(t *testing.T) {
	e := newEnv(t, true)
	alice := e.client()
	alice.register("Alice")
	dir := t.TempDir()
	s := e.syncer(alice, dir, true)

	drawing := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 800" width="1200" height="800"><title>Raft</title>` +
		`<metadata id="axon-sketch">{&quot;version&quot;:1,&quot;shapes&quot;:[{&quot;kind&quot;:&quot;line&quot;,&quot;color&quot;:&quot;#1f2328&quot;,&quot;width&quot;:4,&quot;x1&quot;:0,&quot;y1&quot;:0,&quot;x2&quot;:10,&quot;y2&quot;:10}]}</metadata>` +
		`<rect width="1200" height="800" fill="#ffffff"/><line x1="0" y1="0" x2="10" y2="10" stroke="#1f2328" stroke-width="4"/></svg>` + "\n"
	code, out := alice.do("POST", "/v1/notes/files?name=raft.svg", []byte(drawing), "Content-Type", "image/svg+xml")
	if code != 201 {
		t.Fatalf("upload = %d %s", code, out)
	}
	var f struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(out, &f)
	note := "# Raft\n\n![raft](" + f.Path + ")\n\nA majority: $\\lfloor n/2 \\rfloor + 1$\n\n$$\nP(\\text{split vote}) \\approx e^{-k}\n$$\n"
	alice.mustDo(201, "POST", "/v1/notes", mustJSON(map[string]string{"path": "Zettel/permanent/raft.md", "content": note}))

	s.run()
	if got, ok := readFile(t, dir, f.Path); !ok || got != drawing {
		t.Fatalf("the drawing in the vault = %q", got)
	}
	if got, ok := readFile(t, dir, "Zettel/permanent/raft.md"); !ok || got != note {
		t.Fatalf("the note in the vault = %q", got)
	}

	// Edited in the vault (Obsidian, Excalidraw's SVG export): Axon follows.
	edited := strings.Replace(drawing, `x2="10" y2="10"`, `x2="90" y2="40"`, 1)
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(f.Path)), []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	moreMaths := note + "\nAnd $\\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}$.\n"
	if err := os.WriteFile(filepath.Join(dir, "Zettel", "permanent", "raft.md"), []byte(moreMaths), 0o600); err != nil {
		t.Fatal(err)
	}
	s.run()
	if code, got := alice.do("GET", "/v1/notes/file?path="+f.Path, nil); code != 200 || string(got) != edited {
		t.Fatalf("the drawing in Axon = %d %q", code, got)
	}
	if n := alice.notes()["Zettel/permanent/raft.md"]; n.ID == "" {
		t.Fatal("the note is gone")
	} else if c := alice.mustDo(200, "GET", "/v1/notes/"+n.ID, nil); !strings.Contains(string(c), `\\sum_{i=1}^{n}`) {
		t.Fatalf("the maths did not come back: %s", c)
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
