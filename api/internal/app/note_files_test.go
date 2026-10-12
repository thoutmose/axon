package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	stdjpeg "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type noteFileJSON struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Hash        string `json:"hash"`
}

// getNoteFile fetches a file as the browser does: status, headers, bytes.
func (c *client) getNoteFile(p string) (int, http.Header, []byte) {
	c.e.t.Helper()
	req, _ := http.NewRequest("GET", c.e.srv.URL+"/v1/notes/file?path="+url.QueryEscape(p), nil)
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	out, _ := io.ReadAll(res.Body)
	checkContract(c.e.t, req, nil, res, out)
	return res.StatusCode, res.Header, out
}

// Pictures go into notes from the editor (re-encoded, no metadata), are
// served so nothing in them can run, sync with the vault both ways, and are
// part of the export.
func TestNoteFiles(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	// The editor's upload: re-encoded, scaled to 2400 px, named by content.
	code, out := alice.do("POST", "/v1/notes/files?name=Whiteboard%20photo.png", noisyPNG(3000, 300), "Content-Type", "image/png")
	if code != 201 {
		t.Fatalf("upload = %d %s", code, out)
	}
	var up noteFileJSON
	_ = json.Unmarshal(out, &up)
	if !strings.HasPrefix(up.Path, "Attachments/whiteboard-photo-") || !strings.HasSuffix(up.Path, ".png") || up.ContentType != "image/png" {
		t.Fatalf("uploaded = %+v", up)
	}
	code, header, pic := alice.getNoteFile(up.Path)
	img, err := png.Decode(bytes.NewReader(pic))
	if code != 200 || err != nil || img.Bounds().Dx() != 2400 || img.Bounds().Dy() != 240 {
		t.Fatalf("served picture: %d %v %v", code, err, img)
	}
	if csp := header.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") || header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("picture served without the sandbox: %v", header)
	}
	// A photo stays a JPEG, without its metadata.
	img2, _ := png.Decode(bytes.NewReader(noisyPNG(64, 48)))
	var jpg bytes.Buffer
	_ = stdjpeg.Encode(&jpg, img2, nil)
	if code, out := alice.do("POST", "/v1/notes/files?name=photo.jpeg", jpg.Bytes(), "Content-Type", "image/jpeg"); code != 201 || !bytes.Contains(out, []byte(`.jpg"`)) {
		t.Fatalf("jpeg upload = %d %s", code, out)
	}
	// The same picture again is the same file.
	_, out = alice.do("POST", "/v1/notes/files?name=Whiteboard%20photo.png", noisyPNG(3000, 300), "Content-Type", "image/png")
	var again noteFileJSON
	_ = json.Unmarshal(out, &again)
	if again.Path != up.Path {
		t.Fatalf("same picture stored twice: %s and %s", up.Path, again.Path)
	}
	// Not a picture, a type mismatch, or someone else's file: refused.
	if code, _ := alice.do("POST", "/v1/notes/files?name=x.png", []byte("not a picture"), "Content-Type", "image/png"); code != 422 {
		t.Fatalf("garbage upload = %d", code)
	}
	if code, _ := alice.do("POST", "/v1/notes/files?name=x.exe", []byte("MZ"), "Content-Type", "application/octet-stream"); code != 415 {
		t.Fatalf("exe upload = %d", code)
	}
	if code, _ := alice.do("POST", "/v1/notes/files?name=x.gif", noisyPNG(4, 4), "Content-Type", "image/gif"); code != 422 {
		t.Fatalf("PNG named .gif = %d", code)
	}
	if code, _ := bob.do("GET", "/v1/notes/file?path="+url.QueryEscape(up.Path), nil); code != 404 {
		t.Fatalf("bob reads alice's picture: %d", code)
	}
	if code, _ := alice.do("GET", "/v1/notes/file?path=../etc/passwd.png", nil); code != 422 {
		t.Fatalf("path escape = %d", code)
	}

	// An SVG keeps its bytes and is sandboxed; a PDF downloads.
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><script>alert(1)</script><circle r="4" cx="5" cy="5"/></svg>`)
	if code, out := alice.do("POST", "/v1/notes/files?name=diagram.svg", svg, "Content-Type", "image/svg+xml"); code != 201 {
		t.Fatalf("svg upload = %d %s", code, out)
	}
	if code, out := alice.do("POST", "/v1/notes/files?name=paper.pdf", []byte("%PDF-1.4\n%%EOF\n"), "Content-Type", "application/pdf"); code != 201 {
		t.Fatalf("pdf upload = %d %s", code, out)
	}
	var list struct {
		Items []noteFileJSON `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/files", nil), &list)
	if len(list.Items) != 4 {
		t.Fatalf("files = %+v", list.Items)
	}
	for _, f := range list.Items {
		if f.ContentType == "application/pdf" {
			if _, h, _ := alice.getNoteFile(f.Path); !strings.HasPrefix(h.Get("Content-Disposition"), "attachment") {
				t.Fatalf("pdf served inline: %v", h)
			}
		}
	}

	// Vault → Axon: a picture beside a note arrives byte for byte.
	photo := noisyPNG(20, 10)
	dir := writeVault(t, map[string]string{"Trip.md": "# Trip\n\n![[beach.png]]\n"})
	if err := os.WriteFile(filepath.Join(dir, "beach.png"), photo, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fake.jpg"), []byte("not a jpeg"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := e.syncer(alice, dir, true)
	rep := s.run()
	if _, _, got := alice.getNoteFile("beach.png"); !bytes.Equal(got, photo) {
		t.Fatal("synced picture differs")
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0] != "fake.jpg" {
		t.Fatalf("skipped = %v", rep.Skipped)
	}
	// Axon → vault: the editor's pictures land in the vault.
	if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(up.Path))); err != nil || !bytes.Equal(b, pic) { //nolint:gosec // test temp dir
		t.Fatalf("uploaded picture not in the vault: %v", err)
	}

	// Changed on both sides: Axon's copy is kept beside the file.
	beach := noteFileJSON{}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes/files", nil), &list)
	for _, f := range list.Items {
		if f.Path == "beach.png" {
			beach = f
		}
	}
	if code, out := alice.do("PUT", "/v1/notes/file?path=beach.png&base_hash="+beach.Hash, noisyPNG(30, 10), "Content-Type", "application/octet-stream"); code != 200 {
		t.Fatalf("put = %d %s", code, out)
	}
	if code, _ := alice.do("PUT", "/v1/notes/file?path=beach.png&base_hash="+beach.Hash, noisyPNG(40, 10), "Content-Type", "application/octet-stream"); code != 409 {
		t.Fatalf("stale put = %d", code)
	}
	if err := os.WriteFile(filepath.Join(dir, "beach.png"), noisyPNG(50, 10), 0o600); err != nil {
		t.Fatal(err)
	}
	rep = s.run()
	if _, ok := readFile(t, dir, "beach.axon-conflict.png"); !ok || len(rep.Conflicts) != 1 {
		t.Fatalf("no conflict copy: %+v", rep)
	}
	_ = os.Remove(filepath.Join(dir, "beach.axon-conflict.png"))
	s.run() // the local version wins once the copy is gone

	// Deleted in the vault → deleted in Axon; deleted in Axon → deleted in the vault.
	_ = os.Remove(filepath.Join(dir, "beach.png"))
	s.run()
	if code, _ := alice.do("GET", "/v1/notes/file?path=beach.png", nil); code != 404 {
		t.Fatalf("deleted picture still served: %d", code)
	}
	alice.mustDo(204, "DELETE", "/v1/notes/file?path="+url.QueryEscape(up.Path), nil)
	s.run()
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(up.Path))); !os.IsNotExist(err) {
		t.Fatalf("picture deleted in Axon kept in the vault: %v", err)
	}

	// The export carries the files at their vault paths.
	archive := alice.mustDo(200, "GET", "/v1/me/export", nil)
	zr, _ := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	found := false
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "notes/Attachments/diagram-") {
			found = true
		}
	}
	if !found {
		t.Fatal("export lacks the note files")
	}
}
