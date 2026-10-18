package app

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thoutmose/axon/api/internal/notes"
	"github.com/thoutmose/axon/api/internal/notesync"
)

// A small vault in the shape of ~/SecondBrain: PARA folders, front matter,
// wikilinks in both front matter and body.
var vault = map[string]string{
	"2-Areas/maths/maths.md": "---\ntype: area\ntitle: Maths\n---\n> Area.\n## Notes & resources\n- [[mit-6042j]]\n",
	"Zettel/literature/mit-6042j.md": "---\ntype: literature\ntitle: MIT 6.042J\narea: \"[[maths]]\"\n---\n## Key ideas\n" +
		"- A proof is a chain of deductions; see [[proofs#induction|induction]].\n| P | ¬P |\n| - | -- |\n",
	"Zettel/permanent/proofs.md": "# Proofs\n\nInduction proves P(n) for all n.\r\nWindows line ending kept.\n",
	"0-Inbox/idea.md":            "untitled idea, no front matter, no heading",
}

func writeVault(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, c := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Ignored by sync: hidden folders and Obsidian state.
	_ = os.MkdirAll(filepath.Join(dir, ".obsidian"), 0o750)
	_ = os.WriteFile(filepath.Join(dir, ".obsidian", "workspace.md"), []byte("ignored"), 0o600)
	// A symlink out of the vault is never followed.
	secret := filepath.Join(t.TempDir(), "secret.md")
	_ = os.WriteFile(secret, []byte("outside"), 0o600)
	_ = os.Symlink(secret, filepath.Join(dir, "0-Inbox", "link.md"))
	return dir
}

func readFile(t *testing.T, dir, p string) (string, bool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p))) //nolint:gosec // test temp dir
	return string(b), err == nil
}

// syncer runs notesync against the test server with a fresh write token.
type syncer struct {
	cfg notesync.Config
	t   *testing.T
}

func (e *env) syncer(c *client, dir string, twoWay bool) *syncer {
	var res struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(c.mustDo(201, "POST", "/v1/me/tokens", []byte(`{"name":"vault","scope":"write","expires_in_days":30}`)), &res)
	// The client calls <url>/api/...; the test server has no /api prefix.
	h := e.srv.Config.Handler
	proxy := httptest.NewServer(http.StripPrefix("/api", h))
	e.t.Cleanup(proxy.Close)
	return &syncer{t: e.t, cfg: notesync.Config{
		Dir: dir, URL: proxy.URL, Token: res.Token, TwoWay: twoWay,
		StatePath: filepath.Join(e.t.TempDir(), "state.json"),
	}}
}

func (s *syncer) run() notesync.Report {
	s.t.Helper()
	rep, err := notesync.Run(context.Background(), s.cfg)
	if err != nil {
		s.t.Fatalf("sync: %v", err)
	}
	return rep
}

type noteJSON struct {
	ID      string  `json:"id"`
	Path    string  `json:"path"`
	Title   string  `json:"title"`
	Kind    string  `json:"kind"`
	Para    string  `json:"para"`
	Content *string `json:"content"`
}

func (c *client) notes() map[string]noteJSON {
	c.e.t.Helper()
	var res struct {
		Items []noteJSON `json:"items"`
	}
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/notes", nil), &res)
	out := map[string]noteJSON{}
	for _, n := range res.Items {
		out[n.Path] = n
	}
	return out
}

func (c *client) note(id string) noteJSON {
	c.e.t.Helper()
	var n noteJSON
	_ = json.Unmarshal(c.mustDo(200, "GET", "/v1/notes/"+id, nil), &n)
	return n
}

func TestImportVaultAndResolveWikilinks(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	dir := writeVault(t, vault)
	s := e.syncer(alice, dir, false)

	if rep := s.run(); rep.Pushed != 4 || len(rep.Conflicts) != 0 {
		t.Fatalf("first import: %+v", rep)
	}
	if rep := s.run(); rep.Pushed != 0 || rep.Unchanged != 4 {
		t.Fatalf("second import should be a no-op: %+v", rep)
	}
	notes := alice.notes()
	lit := notes["Zettel/literature/mit-6042j.md"]
	if lit.Title != "MIT 6.042J" || lit.Kind != "literature" || lit.Para != "other" {
		t.Fatalf("literature note = %+v", lit)
	}
	if notes["2-Areas/maths/maths.md"].Para != "areas" || notes["0-Inbox/idea.md"].Title != "idea" {
		t.Fatalf("metadata = %+v", notes)
	}
	// [[maths]] (front matter) and [[proofs#induction|induction]] (body) became links.
	out := alice.mustDo(200, "GET", "/v1/entities/"+lit.ID+"/links", nil)
	for _, want := range []string{notes["2-Areas/maths/maths.md"].ID, notes["Zettel/permanent/proofs.md"].ID} {
		if !bytes.Contains(out, []byte(want)) {
			t.Fatalf("missing link to %s in %s", want, out)
		}
	}
	// Backlinks: the maths area links back to the literature note.
	back := alice.mustDo(200, "GET", "/v1/entities/"+lit.ID+"/links?direction=incoming", nil)
	if !bytes.Contains(back, []byte(notes["2-Areas/maths/maths.md"].ID)) {
		t.Fatalf("backlinks = %s", back)
	}
	// Import never writes files and never overwrites Axon.
	if got, _ := readFile(t, dir, "0-Inbox/idea.md"); got != vault["0-Inbox/idea.md"] {
		t.Fatal("import modified a file")
	}
}

func TestExportThenImportIsByteIdentical(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	e.syncer(alice, writeVault(t, vault), false).run()

	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	exported := map[string]string{}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		if p, ok := strings.CutPrefix(f.Name, "notes/"); ok {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			exported[p] = string(b)
			w, _ := zw.Create(f.Name)
			_, _ = w.Write(b)
		}
	}
	_ = zw.Close()
	for p, want := range vault {
		if exported[p] != want {
			t.Errorf("export of %s differs:\n%q\nwant\n%q", p, exported[p], want)
		}
	}

	// Import the archive into a fresh account: same bytes again.
	e2 := newEnv(t, false)
	bob := e2.client()
	bob.register("Bob")
	var res struct {
		Created   []string `json:"created"`
		Conflicts []string `json:"conflicts"`
	}
	code, out := bob.do("POST", "/v1/notes/import", buf.Bytes(), "Content-Type", "application/zip")
	_ = json.Unmarshal(out, &res)
	if code != 200 || len(res.Created) != len(vault) {
		t.Fatalf("import = %+v", res)
	}
	for p, n := range bob.notes() {
		if got := bob.note(n.ID); *got.Content != vault[p] {
			t.Errorf("re-imported %s differs", p)
		}
	}
	// Importing again changes nothing and overwrites nothing.
	res.Created = nil
	_, out = bob.do("POST", "/v1/notes/import", buf.Bytes(), "Content-Type", "application/zip")
	_ = json.Unmarshal(out, &res)
	if len(res.Created) != 0 || len(res.Conflicts) != 0 {
		t.Fatalf("second import = %+v", res)
	}
}

func TestRenamingANoteRewritesLinksToIt(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	e.syncer(alice, writeVault(t, vault), false).run()
	notes := alice.notes()
	proofs := alice.note(notes["Zettel/permanent/proofs.md"].ID)

	etag := alice.etag("/v1/notes/" + proofs.ID)
	body, _ := json.Marshal(map[string]string{"path": "Zettel/permanent/mathematical-proofs.md", "content": *proofs.Content})
	if code, out := alice.do("PUT", "/v1/notes/"+proofs.ID, body, "If-Match", etag); code != 200 {
		t.Fatalf("rename = %d %s", code, out)
	}
	lit := alice.note(notes["Zettel/literature/mit-6042j.md"].ID)
	if !strings.Contains(*lit.Content, "[[mathematical-proofs#induction|induction]]") {
		t.Fatalf("link not rewritten:\n%s", *lit.Content)
	}
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/entities/"+lit.ID+"/links", nil), []byte(proofs.ID)) {
		t.Fatal("graph link lost on rename")
	}
	// The old path is a tombstone for sync.
	if !bytes.Contains(alice.mustDo(200, "GET", "/v1/notes/sync", nil), []byte(`"path":"Zettel/permanent/proofs.md"`)) {
		t.Fatal("no tombstone for the old path")
	}
	alice.mustDo(409, "POST", "/v1/notes", []byte(`{"path":"2-Areas/maths/maths.md","content":"dup"}`))
	alice.mustDo(422, "POST", "/v1/notes", []byte(`{"path":"../etc/passwd.md","content":"x"}`))
}

func TestTwoWaySyncNeverLosesAnEdit(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	dir := writeVault(t, vault)
	s := e.syncer(alice, dir, true)
	s.run()
	notes := alice.notes()
	put := func(id, content string) {
		t.Helper()
		n := alice.note(id)
		body, _ := json.Marshal(map[string]string{"path": n.Path, "content": content})
		if code, out := alice.do("PUT", "/v1/notes/"+id, body, "If-Match", alice.etag("/v1/notes/"+id)); code != 200 {
			t.Fatalf("PUT = %d %s", code, out)
		}
	}
	write := func(p, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(p)), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Axon → file.
	put(notes["0-Inbox/idea.md"].ID, "edited in Axon")
	s.run()
	if got, _ := readFile(t, dir, "0-Inbox/idea.md"); got != "edited in Axon" {
		t.Fatalf("pull: %q", got)
	}
	// File → Axon.
	write("0-Inbox/idea.md", "edited in the vault")
	s.run()
	if got := alice.note(notes["0-Inbox/idea.md"].ID); *got.Content != "edited in the vault" {
		t.Fatalf("push: %q", *got.Content)
	}
	// Both sides: keep both, overwrite nothing.
	put(notes["0-Inbox/idea.md"].ID, "Axon side")
	write("0-Inbox/idea.md", "vault side")
	rep := s.run()
	if !slices.Contains(rep.Conflicts, "0-Inbox/idea.md") {
		t.Fatalf("conflict not reported: %+v", rep)
	}
	local, _ := readFile(t, dir, "0-Inbox/idea.md")
	theirs, _ := readFile(t, dir, "0-Inbox/idea"+notesync.ConflictSuffix)
	axon := alice.note(notes["0-Inbox/idea.md"].ID)
	if local != "vault side" || theirs != "Axon side" || *axon.Content != "Axon side" {
		t.Fatalf("conflict lost data: local %q, conflict copy %q, Axon %q", local, theirs, *axon.Content)
	}
	s.run() // still unresolved: nothing moves
	if got := alice.note(notes["0-Inbox/idea.md"].ID); *got.Content != "Axon side" {
		t.Fatal("unresolved conflict was pushed")
	}
	// Resolve: merge into the file, delete the conflict copy.
	write("0-Inbox/idea.md", "vault side + Axon side")
	_ = os.Remove(filepath.Join(dir, "0-Inbox", "idea"+notesync.ConflictSuffix))
	s.run()
	if got := alice.note(notes["0-Inbox/idea.md"].ID); *got.Content != "vault side + Axon side" {
		t.Fatalf("resolution not pushed: %q", *got.Content)
	}

	// Deleted in Axon → file removed; deleted as a file → removed in Axon.
	alice.mustDo(204, "DELETE", "/v1/notes/"+notes["2-Areas/maths/maths.md"].ID, nil)
	s.run()
	if _, exists := readFile(t, dir, "2-Areas/maths/maths.md"); exists {
		t.Fatal("note deleted in Axon is still a file")
	}
	_ = os.Remove(filepath.Join(dir, "Zettel", "permanent", "proofs.md"))
	s.run()
	if _, ok := alice.notes()["Zettel/permanent/proofs.md"]; ok {
		t.Fatal("file deleted in the vault is still in Axon")
	}
	// New on each side.
	write("0-Inbox/new-local.md", "# From the vault\n")
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"0-Inbox/new-axon.md","content":"# From Axon\n"}`))
	s.run()
	if _, ok := alice.notes()["0-Inbox/new-local.md"]; !ok {
		t.Fatal("new file not pushed")
	}
	if got, _ := readFile(t, dir, "0-Inbox/new-axon.md"); got != "# From Axon\n" {
		t.Fatal("new Axon note not pulled")
	}
	if rep := s.run(); rep.Pushed+rep.Pulled+rep.Deleted != 0 || len(rep.Conflicts) != 0 {
		t.Fatalf("steady state is not quiet: %+v", rep)
	}
}

func TestNoteEndpoints(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	var n noteJSON
	_ = json.Unmarshal(alice.mustDo(201, "POST", "/v1/notes", []byte(`{"content":"# Fleeting thought\nabout [[nothing]]"}`)), &n)
	if n.Path != "0-Inbox/fleeting-thought.md" || n.Para != "inbox" {
		t.Fatalf("default path = %+v", n)
	}
	alice.mustDo(200, "GET", "/v1/notes?para=inbox&limit=10", nil)
	if code, _ := alice.do("PUT", "/v1/notes/"+n.ID, []byte(`{"path":"0-Inbox/fleeting-thought.md","content":"x"}`), "If-Match", `"stale"`); code != 412 {
		t.Fatalf("stale PUT = %d", code)
	}
	alice.mustDo(204, "DELETE", "/v1/notes/"+n.ID, nil)
	alice.mustDo(404, "GET", "/v1/notes/"+n.ID, nil)
	alice.mustDo(204, "POST", "/v1/notes/sync/delete", []byte(`{"path":"0-Inbox/gone.md","base_hash":"x"}`))

	// Workspace: tags, exact paths, plain-text mentions, and note search.
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/spaced.md","content":"---\ntags: [learning]\n---\n# Spaced repetition\nReview cards. #memory"}`))
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Daily/2026-10-10.md","content":"# 2026-10-10\nRead about spaced repetition today."}`))
	if b := alice.mustDo(200, "GET", "/v1/notes?tag=memory", nil); !bytes.Contains(b, []byte("Zettel/spaced.md")) || bytes.Contains(b, []byte("Daily/")) {
		t.Fatalf("tag filter = %s", b)
	}
	if b := alice.mustDo(200, "GET", "/v1/notes/tags", nil); !bytes.Contains(b, []byte(`{"tag":"learning","count":1}`)) {
		t.Fatalf("tags = %s", b)
	}
	var daily struct {
		Items []noteJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?path=Daily/2026-10-10.md", nil), &daily)
	if len(daily.Items) != 1 {
		t.Fatalf("path lookup = %+v", daily)
	}
	var spaced struct {
		Items []noteJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?path=Zettel/spaced.md", nil), &spaced)
	if b := alice.mustDo(200, "GET", "/v1/notes/"+spaced.Items[0].ID+"/mentions", nil); !bytes.Contains(b, []byte(daily.Items[0].ID)) {
		t.Fatalf("mentions = %s", b)
	}
	if b := alice.mustDo(200, "GET", "/v1/search?q=cards&type=note", nil); !bytes.Contains(b, []byte(`\u0002cards\u0003`)) {
		t.Fatalf("note search snippet = %s", b)
	}

	// Sync writes only land on the version the client last saw.
	alice.mustDo(200, "POST", "/v1/notes/sync", []byte(`{"path":"Zettel/a.md","content":"one","base_hash":""}`))
	alice.mustDo(409, "POST", "/v1/notes/sync", []byte(`{"path":"Zettel/a.md","content":"two","base_hash":"not-the-current-hash"}`))
	alice.mustDo(200, "POST", "/v1/notes/sync", []byte(`{"path":"Zettel/a.md","content":"two","base_hash":"`+notes.Hash("one")+`"}`))
	alice.mustDo(409, "POST", "/v1/notes/sync/delete", []byte(`{"path":"Zettel/a.md","base_hash":"`+notes.Hash("one")+`"}`))
}

// A vault of thousands of notes imports in one request, quickly, with every
// link resolved: forward references in the archive, and older notes that
// named an imported one before it existed.
func TestImportingThousandsOfNotesLinksThemAll(t *testing.T) {
	e := newEnv(t, false)
	alice := e.client()
	alice.register("Alice")
	alice.mustDo(201, "POST", "/v1/notes", []byte(`{"path":"Zettel/older.md","content":"# Older\n\nSee [[Imported 1499]]."}`))
	const n = 1500
	filler := strings.Repeat("Spaced practice beats massed practice for long-term memory. ", 20) // about 1 KB, as in a real vault
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := range n {
		w, _ := zw.Create(fmt.Sprintf("Zettel/imported/Imported %d.md", i))
		_, _ = fmt.Fprintf(w, "# Imported %d\n\n%s\n\nNext: [[Imported %d]]. Back to [[older]].\n", i, filler, (i+1)%n)
	}
	_ = zw.Close()
	start := time.Now()
	code, out := alice.do("POST", "/v1/notes/import", buf.Bytes(), "Content-Type", "application/zip")
	if code != 200 {
		t.Fatalf("import = %d %s", code, out)
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Errorf("importing %d notes took %s", n, took)
	}
	var me struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/me", nil), &me)
	var links, toOlder, fromOlder int
	err := pool.QueryRow(context.Background(), `SELECT count(*),
		count(*) FILTER (WHERE t.path = 'Zettel/older.md'),
		count(*) FILTER (WHERE s.path = 'Zettel/older.md')
		FROM links l JOIN notes s ON s.id = l.source_id JOIN notes t ON t.id = l.target_id
		WHERE l.user_id = $1 AND l.relation = $2`, me.ID, notes.References).Scan(&links, &toOlder, &fromOlder)
	if err != nil {
		t.Fatal(err)
	}
	// Each imported note links to the next and to the older note; the older one to the last.
	if links != 2*n+1 || toOlder != n || fromOlder != 1 {
		t.Fatalf("links = %d (to older %d, from older %d), want %d, %d, 1", links, toOlder, fromOlder, 2*n+1, n)
	}
}
