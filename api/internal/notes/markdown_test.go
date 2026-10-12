package notes

import (
	"slices"
	"strings"
	"testing"
)

const literature = `---
type: literature
title: mit-6042j-mathematics-for-computer-science
area: "[[maths]]"
---
> Literature note.
## Key ideas
- See [[proofs#induction|induction]] and [[maths]] and [[Proofs]].
`

func TestParse(t *testing.T) {
	m := Parse("Zettel/literature/2026-mit.md", literature)
	if m.Title != "mit-6042j-mathematics-for-computer-science" || m.Kind != "literature" || m.Para != "other" {
		t.Fatalf("%+v", m)
	}
	if m := Parse("1-Projects/rustkv/rustkv.md", "# RustKV\nBody"); m.Title != "RustKV" || m.Kind != "note" || m.Para != "projects" {
		t.Fatalf("%+v", m)
	}
	if m := Parse("idea.md", "no title here"); m.Title != "idea" || m.Para != "inbox" {
		t.Fatalf("%+v", m)
	}
	if m := Parse("4-Archives/old.md", "---\ntitle: 'Quoted'\n---\n"); m.Title != "Quoted" || m.Para != "archive" {
		t.Fatalf("%+v", m)
	}
	if m := Parse("2-Areas/x.md", "---\ntitle: unterminated\nbody"); m.Title != "x" {
		t.Fatalf("unterminated front matter: %+v", m)
	}
}

func TestWikilinks(t *testing.T) {
	got := Wikilinks(literature)
	if !slices.Equal(got, []string{"maths", "proofs"}) {
		t.Fatalf("got %v", got)
	}
	if got := Wikilinks("[[ ]] and [[a]]"); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("empty link not ignored: %v", got)
	}
}

func TestRenameLinksKeepsHeadingsAndAliases(t *testing.T) {
	got := RenameLinks("See [[Proofs#induction|induction]], [[proofs]], [[proofsX]].", "proofs", "mathematical-proofs")
	want := "See [[mathematical-proofs#induction|induction]], [[mathematical-proofs]], [[proofsX]]."
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestCleanPath(t *testing.T) {
	for _, ok := range []string{"a.md", "Zettel/literature/x y.md", "0-Inbox/é.md"} {
		if _, valid := CleanPath(ok); !valid {
			t.Errorf("%q rejected", ok)
		}
	}
	for _, bad := range []string{"", "/etc/passwd.md", "../x.md", "a/../../b.md", "a.txt", ".obsidian/x.md", "a/.git/x.md", "a//b.md", ".md"} {
		if _, valid := CleanPath(bad); valid {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Raft & Paxos!": "raft-paxos", "  ": "note", "Réviser le japonais": "réviser-le-japonais"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTags(t *testing.T) {
	got := Parse("Zettel/x.md", "---\ntags: [Maths, \"proofs\"]\ntype: permanent\n---\n# X\nAbout #induction and #maths, not #123 or a#b.\n`#code` stays out.\n```\n#fenced\n```\n- see (#nested/tag)\n").Tags
	want := []string{"maths", "proofs", "induction", "nested/tag"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tags = %v, want %v", got, want)
	}
	if got := Parse("a.md", "---\ntags:\n  - reading\n  - Books\ntitle: A\n---\nbody").Tags; strings.Join(got, ",") != "reading,books" {
		t.Fatalf("list tags = %v", got)
	}
	if got := Parse("a.md", "# Heading only").Tags; len(got) != 0 {
		t.Fatalf("heading read as tag: %v", got)
	}
}
