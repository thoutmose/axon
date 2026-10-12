package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type importResult struct {
	Found, Created, Skipped, Conflicts int
	Problems, Sample                   []string
	DryRun                             bool `json:"dry_run"`
}

func (c *client) importFrom(source, query string, body []byte) importResult {
	c.e.t.Helper()
	code, out := c.do("POST", "/v1/import/"+source+query, body, "Content-Type", "application/octet-stream")
	if code != 200 {
		c.e.t.Fatalf("import %s = %d %s", source, code, out)
	}
	var r importResult
	_ = json.Unmarshal(out, &r)
	return r
}

// Other apps' exports come in once: tasks, books, films, money, and pages.
func TestImports(t *testing.T) {
	e := newEnv(t, true)
	alice, bob := e.client(), e.client()
	alice.register("Alice")
	bob.register("Bob")

	todoist := []byte("TYPE,CONTENT,DESCRIPTION,PRIORITY,INDENT,AUTHOR,RESPONSIBLE,DATE,DATE_LANG,TIMEZONE\n" +
		"task,Call the bank @phone,About the card,4,1,Ada,,2026-10-14,en,Europe/Paris\ntask,Water plants,,1,1,Ada,,every monday,en,Europe/Paris\n")
	if r := alice.importFrom("todoist", "?dry_run=true", todoist); !r.DryRun || r.Found != 2 || r.Created != 0 || len(r.Sample) != 2 {
		t.Fatalf("dry run = %+v", r)
	}
	if r := alice.importFrom("todoist", "", todoist); r.Created != 2 {
		t.Fatalf("todoist = %+v", r)
	}
	if r := alice.importFrom("todoist", "", todoist); r.Created != 0 || r.Skipped != 2 {
		t.Fatalf("todoist again = %+v", r)
	}
	if out := alice.mustDo(200, "GET", "/v1/tasks?view=next", nil); !bytes.Contains(out, []byte(`"context":"@phone"`)) || !bytes.Contains(out, []byte(`"due_on":"2026-10-14"`)) {
		t.Fatalf("imported task = %s", out)
	}
	if out := bob.mustDo(200, "GET", "/v1/tasks?view=all", nil); bytes.Contains(out, []byte("Call the bank")) {
		t.Fatal("bob sees alice's import")
	}

	goodreads := []byte("Book Id,Title,Author,Author l-f,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf\n" +
		"1,Atomic Habits,James Clear,,,,,5,4.4,,,,2018,2018,2024/03/02,2024/01/01,,,read\n")
	if r := alice.importFrom("goodreads", "", goodreads); r.Created != 1 {
		t.Fatalf("goodreads = %+v", r)
	}
	if r := alice.importFrom("goodreads", "", goodreads); r.Skipped != 1 {
		t.Fatalf("goodreads again = %+v", r)
	}
	if out := alice.mustDo(200, "GET", "/v1/media?kind=book", nil); !bytes.Contains(out, []byte(`"rating":10`)) {
		t.Fatalf("book = %s", out)
	}

	ofx := []byte("<OFX><STMTTRN><DTPOSTED>20261003<TRNAMT>-42.10<NAME>SUPERMARKET<MEMO>Card</STMTTRN><STMTTRN><DTPOSTED>20261005<TRNAMT>15.00<NAME>Refund</STMTTRN></OFX>")
	if r := alice.importFrom("ofx", "", ofx); r.Created != 2 {
		t.Fatalf("ofx = %+v", r)
	}
	if r := alice.importFrom("ofx", "", ofx); r.Skipped != 2 {
		t.Fatalf("ofx again = %+v", r)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"Export/Garden 0123456789abcdef0123456789abcdef.md": "# Garden\n\nSee [Compost](Garden%200123456789abcdef0123456789abcdef/Compost%20fedcba9876543210fedcba9876543210.md).\n\n" +
			"![The shed](Garden%200123456789abcdef0123456789abcdef/shed.png)\n",
		"Export/Garden 0123456789abcdef0123456789abcdef/Compost fedcba9876543210fedcba9876543210.md": "# Compost\n",
		"Export/Garden 0123456789abcdef0123456789abcdef/shed.png":                                    string(pngOf(4, 4)),
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	if r := alice.importFrom("notion", "", buf.Bytes()); r.Created != 3 {
		t.Fatalf("notion = %+v", r)
	}
	if out := alice.mustDo(200, "GET", "/v1/notes?folder=Notion", nil); !strings.Contains(string(out), `"path":"Notion/Export/Garden.md"`) {
		t.Fatalf("notion notes = %s", out)
	}
	// The picture comes along, and the page links it by its new name.
	if out := alice.mustDo(200, "GET", "/v1/notes/files", nil); !strings.Contains(string(out), `"path":"Notion/Export/Garden/shed.png"`) {
		t.Fatalf("notion pictures = %s", out)
	}
	var list struct {
		Items []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		} `json:"items"`
	}
	_ = json.Unmarshal(alice.mustDo(200, "GET", "/v1/notes?folder=Notion", nil), &list)
	for _, n := range list.Items {
		if out := alice.mustDo(200, "GET", "/v1/notes/"+n.ID, nil); n.Path == "Notion/Export/Garden.md" && !strings.Contains(string(out), "![The shed](Garden/shed.png)") {
			t.Fatalf("the page does not link its picture: %s", out)
		}
	}
	// Imported again: nothing twice.
	if r := alice.importFrom("notion", "", buf.Bytes()); r.Created != 0 {
		t.Fatalf("notion again = %+v", r)
	}

	// Not the right file: said clearly.
	if code, out := alice.do("POST", "/v1/import/goodreads", []byte("a,b\n1,2\n"), "Content-Type", "application/octet-stream"); code != 422 || !strings.Contains(string(out), "not the expected export") {
		t.Fatalf("wrong file = %d %s", code, out)
	}
	alice.mustDo(404, "POST", "/v1/import/evernote", []byte("x"))
}
