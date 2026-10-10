package brain

import (
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// Ported from tests/brain-dump.test.ts. The markdown file on disk is gone:
// the engine is the store, so a dump becomes one engine capture.

func TestSlugifyTitle(t *testing.T) {
	for in, want := range map[string]string{"Hello, World! Big   Idea": "hello-world-big-idea", "  --  ": "untitled", "Café_ideas": "cafe-ideas"} {
		if got := SlugifyTitle(in); got != want {
			t.Fatalf("SlugifyTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDumpValidateAndTitle(t *testing.T) {
	if _, err := (DumpInput{Text: "   ", Folder: "inbox"}).Validate(); err == nil {
		t.Fatal("empty text must be rejected")
	}
	for _, bad := range []string{"../etc", "a/b", "", "in box"} {
		if _, err := (DumpInput{Text: "x", Folder: bad}).Validate(); err == nil {
			t.Fatalf("folder %q must be rejected", bad)
		}
	}
	d, err := DumpInput{Text: "one two three four five six seven eight nine", Folder: "ideas", Tags: []string{" #vantage ", ""}}.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "one two three four five six seven" {
		t.Fatalf("derived title %q (first seven words)", d.Title)
	}
	if len(d.Tags) != 1 || d.Tags[0] != "vantage" {
		t.Fatalf("tags %v", d.Tags)
	}
	d2, _ := DumpInput{Text: "body", Title: "  Given  ", Folder: "inbox"}.Validate()
	if d2.Title != "Given" {
		t.Fatalf("title %q", d2.Title)
	}
}

func TestDumpDocumentAndRouting(t *testing.T) {
	topo, err := topology.LoadRepo()
	if err != nil {
		t.Fatal(err)
	}
	d, _ := DumpInput{Text: "the idea", Title: "Big Idea", Folder: "inbox"}.Validate()
	doc := d.Document(now)
	for _, want := range []string{"source: founderos-os-brain-dump", "tags: []", "# Big Idea", "the idea", "created: 2026-09-17T12:00:00Z"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document missing %q:\n%s", want, doc)
		}
	}
	if got := d.RelPath(now); got != "inbox/2026-09-17-big-idea.md" {
		t.Fatalf("rel path %q", got)
	}
	if ws := d.Workspace(topo); ws != "founderos" {
		t.Fatalf("inbox routes to %q, want the brain-store default", ws)
	}
	if n := d.Node(); n != "inbox" {
		t.Fatalf("node %q", n)
	}
	m, _ := DumpInput{Text: "x", Folder: "vantage"}.Validate()
	if ws := m.Workspace(topo); ws != "vantage" {
		t.Fatalf("vantage folder routes to %q", ws)
	}
	tagged, _ := DumpInput{Text: "x", Folder: "ideas", Tags: []string{"#launchpad-cohort"}}.Validate()
	if ws := tagged.Workspace(topo); ws != "launchpad-cohort" || tagged.Node() != "knowledge-base" {
		t.Fatalf("a venture tag routes to its workspace: %q / %q", ws, tagged.Node())
	}
	personal, _ := DumpInput{Text: "x", Folder: "memories", Tags: []string{"vantage"}}.Validate()
	if ws := personal.Workspace(topo); ws != "personal" {
		t.Fatalf("a personal folder stays personal whatever the tags: %q", ws)
	}
}
