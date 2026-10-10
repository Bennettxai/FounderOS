package brainimport

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestFromSnapshotReplaysSourcePackagesByTopology(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snap.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		"CREATE TABLE source_packages (id TEXT, workspace_id TEXT, raw_text TEXT, content_hash TEXT)",
		"INSERT INTO source_packages VALUES ('p1','default:founderos','# Pricing call\nwe agreed on 5k','h1')",
		"INSERT INTO source_packages VALUES ('p2','default:personal','journal entry','h2')",
		"INSERT INTO source_packages VALUES ('p3','default','legacy row','h3')",
		"INSERT INTO source_packages VALUES ('p4','default:vantage','',  'h4')",
		"INSERT INTO source_packages VALUES ('p5','default:vantage',NULL,'h5')",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	db.Close()

	items, empty, err := FromSnapshot(path, topo(t))
	if err != nil {
		t.Fatal(err)
	}
	if empty != 2 {
		t.Errorf("empty source packages = %d, want 2", empty)
	}
	got := map[string]Item{}
	for _, it := range items {
		got[it.Rel] = it
	}
	cases := map[string]struct{ ws, engine, title string }{
		"source_packages/p1": {"founderos", "hub", "Pricing call"},
		"source_packages/p2": {"personal", "macbook", "journal entry"},
		"source_packages/p3": {"founderos", "hub", "legacy row"}, // retired default → founderos
	}
	if len(items) != len(cases) {
		t.Fatalf("items = %+v", items)
	}
	for rel, w := range cases {
		it := got[rel]
		if it.Workspace != w.ws || it.Engine != w.engine || it.Title != w.title {
			t.Errorf("%s → %s/%s %q, want %s/%s %q", rel, it.Workspace, it.Engine, it.Title, w.ws, w.engine, w.title)
		}
	}
}

func TestFromSnapshotAlsoReplaysContextsWithNoSourcePackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snap.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		"CREATE TABLE source_packages (id TEXT, workspace_id TEXT, raw_text TEXT, content_hash TEXT)",
		"CREATE TABLE contexts (id TEXT, workspace_id TEXT, title TEXT, genre TEXT, node TEXT, content TEXT, archived_at TEXT)",
		"INSERT INTO source_packages VALUES ('p1','default:founderos','we agreed on 5k for the retainer','h1')",
		// Same text as p1 wrapped in frontmatter: already covered, not replayed.
		"INSERT INTO contexts VALUES ('c1','default:founderos','Untitled','note','inbox','---\nx: 1\n---\nwe agreed on 5k for the retainer', NULL)",
		// A file-indexed signal with no source package: must be replayed.
		"INSERT INTO contexts VALUES ('c2','default:vantage','Conductor harness','chat','team','the conductor harness routes seats', NULL)",
		// Archived contexts stay behind.
		"INSERT INTO contexts VALUES ('c3','default:vantage','Old','note','team','archived text', '2026-08-01')",
		"INSERT INTO contexts VALUES ('c4','default:vantage','Empty','note','team','   ', NULL)",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	db.Close()

	items, _, err := FromSnapshot(path, topo(t))
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, it := range items {
		rels = append(rels, it.Rel)
		if it.Rel == "contexts/c2" {
			if it.Workspace != "vantage" || it.Title != "Conductor harness" || it.Genre != "chat" || it.Node != "team" {
				t.Errorf("c2 item = %+v", it)
			}
		}
	}
	if len(items) != 2 || rels[0] != "source_packages/p1" || rels[1] != "contexts/c2" {
		t.Fatalf("items = %v, want [source_packages/p1 contexts/c2]", rels)
	}
}

func TestWithoutSupersededDropsOlderCopiesOfStorePages(t *testing.T) {
	store := writeStore(t) // has meetings/2026-09-02-call.md "# Call", sops/onboarding.md "# SOP", ...
	keys, err := StoreTitleKeys(store)
	if err != nil {
		t.Fatal(err)
	}
	items := []Item{
		{Rel: "source_packages/a", Title: "SOP", Text: "# SOP\nold version"},  // H1 matches a store page
		{Rel: "contexts/b", Title: "onboarding", Text: "---\nx: 1\n---\nold"}, // filename slug matches
		{Rel: "source_packages/c", Title: "Pricing decision", Text: "# Pricing decision\nunique"},
		{Rel: "contexts/d", Title: "Untitled", Text: "untitled but unique"}, // generic titles never match
	}
	kept, dropped := WithoutSuperseded(items, keys)
	if dropped != 2 || len(kept) != 2 || kept[0].Rel != "source_packages/c" || kept[1].Rel != "contexts/d" {
		t.Fatalf("kept=%v dropped=%d", kept, dropped)
	}
}

func TestOrphanContextsSkipCopiesOfTheirOwnSourcePackages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snap.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		"CREATE TABLE source_packages (id TEXT, workspace_id TEXT, raw_text TEXT, content_hash TEXT)",
		"CREATE TABLE contexts (id TEXT, workspace_id TEXT, title TEXT, genre TEXT, node TEXT, content TEXT, archived_at TEXT, uri TEXT)",
		"INSERT INTO source_packages VALUES ('p1','default:founderos','# Client Onboarding\nstep one','h1')",
		// Rewritten copy of p1 (different text, same title from its file name): skip.
		"INSERT INTO contexts VALUES ('c1','default:founderos','Untitled','note','team','reformatted step one', NULL, 'optimal://inbox/founderos/team/signals/2026-07-21-client-onboarding.md')",
		// Genuinely separate file: keep, titled from its uri.
		"INSERT INTO contexts VALUES ('c2','default:founderos','Untitled','chat','team','conductor harness notes', NULL, 'optimal://inbox/founderos/team/signals/2026-07-21-conductor-harness.md')",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	db.Close()
	items, _, err := FromSnapshot(path, topo(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Rel != "contexts/c2" || items[1].Title != "conductor harness" {
		t.Fatalf("items = %+v", items)
	}
}
