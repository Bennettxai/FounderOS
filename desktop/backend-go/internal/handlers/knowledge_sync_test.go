package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSectionOf(t *testing.T) {
	cases := map[string]string{
		"inbox/signals/x.md": "inbox",
		"decisions/d1.md":    "decisions",
		"readme.md":          "docs",
		"":                   "docs",
	}
	for in, want := range cases {
		if got := sectionOf(in); got != want {
			t.Errorf("sectionOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTreeFromPaths(t *testing.T) {
	items := []kbDoc{
		{Path: "readme.md", Title: "Readme"},
		{Path: "inbox/signals/a.md", Title: "A"},
		{Path: "inbox/signals/b.md", Title: "B"},
		{Path: "inbox/note.md", Title: "Note"},
	}
	tree := treeFromPaths(items)

	// Top level: dir "inbox" must sort before file "readme.md".
	if len(tree) != 2 {
		t.Fatalf("top-level nodes = %d, want 2", len(tree))
	}
	if tree[0].Type != "dir" || tree[0].Name != "inbox" {
		t.Fatalf("first node = %+v, want dir inbox", tree[0])
	}
	if tree[1].Type != "file" || tree[1].Path != "readme.md" {
		t.Fatalf("second node = %+v, want file readme.md", tree[1])
	}

	// inbox children: dir "signals" before file "note.md".
	inbox := tree[0]
	if len(inbox.Children) != 2 {
		t.Fatalf("inbox children = %d, want 2", len(inbox.Children))
	}
	sig := inbox.Children[0]
	if sig.Type != "dir" || sig.Name != "signals" || sig.Path != "inbox/signals" {
		t.Fatalf("signals node = %+v", sig)
	}
	if len(sig.Children) != 2 || sig.Children[0].Path != "inbox/signals/a.md" {
		t.Fatalf("signals children wrong: %+v", sig.Children)
	}
	if sig.Children[0].Title != "A" {
		t.Errorf("title not carried: %+v", sig.Children[0])
	}
}

func TestCollectLocalDocs(t *testing.T) {
	dir := t.TempDir()
	must := func(p, body string) {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("top.md", "# Top")
	must("nodes/one.md", "# One")
	must("nodes/skip.txt", "not markdown")
	must(".hidden/secret.md", "should be skipped")

	h := &KnowledgeHandler{}
	docs := h.collectLocalDocs(dir)
	if len(docs) != 2 {
		t.Fatalf("collected %d docs, want 2: %+v", len(docs), docs)
	}
	found := map[string]string{}
	for _, d := range docs {
		found[d.Path] = d.Section
	}
	if found["top.md"] != "docs" {
		t.Errorf("top.md section = %q, want docs", found["top.md"])
	}
	if found["nodes/one.md"] != "nodes" {
		t.Errorf("nodes/one.md section = %q, want nodes", found["nodes/one.md"])
	}
}
