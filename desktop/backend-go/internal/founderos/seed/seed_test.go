package seed

import (
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t.Add(15 * time.Hour)
}

func open(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func scalar[T any](t *testing.T, db *sql.DB, q string) T {
	t.Helper()
	var v T
	if err := db.QueryRow(q).Scan(&v); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return v
}

func TestMaterializeWritesEveryDemoDatabase(t *testing.T) {
	dir := t.TempDir()
	if err := Materialize(dir, day(DumpedOn)); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"founderos-os.db", "bank.db", "ledger.db", "paykit.db"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	db := open(t, filepath.Join(dir, "founderos-os.db"))
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM agents`); n != 32 {
		t.Errorf("agents = %d, want the demo's 32", n)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM funnel_contacts`); n != 14 {
		t.Errorf("funnel_contacts = %d, want 14", n)
	}
	if n := scalar[int](t, db, `SELECT COUNT(*) FROM roadmap_items`); n == 0 {
		t.Error("roadmap_items are part of the demo and must be seeded")
	}
}

// v1 resolves funnel and agent-run dates relative to seed time; the frozen
// fixtures move forward by the days since they were dumped, so the demo
// never goes stale (no "70 days quiet" a month later).
func TestMaterializeShiftsRelativeDatesToToday(t *testing.T) {
	base, shifted := t.TempDir(), t.TempDir()
	if err := Materialize(base, day(DumpedOn)); err != nil {
		t.Fatal(err)
	}
	later := day(DumpedOn).AddDate(0, 0, 10)
	if err := Materialize(shifted, later); err != nil {
		t.Fatal(err)
	}
	b, s := open(t, filepath.Join(base, "founderos-os.db")), open(t, filepath.Join(shifted, "founderos-os.db"))

	for _, q := range []string{
		`SELECT MAX("at") FROM funnel_touches`,
		`SELECT MAX(created_at) FROM funnel_contacts`,
	} {
		bv, sv := scalar[string](t, b, q), scalar[string](t, s, q)
		bt, _ := time.Parse("2006-01-02", bv)
		if want := bt.AddDate(0, 0, 10).Format("2006-01-02"); sv != want {
			t.Errorf("%s: %s shifted to %s, want %s", q, bv, sv, want)
		}
	}
	for _, q := range []string{`SELECT MAX(started_at) FROM agent_runs`, `SELECT MIN(finished_at) FROM agent_runs`} {
		bv, sv := scalar[string](t, b, q), scalar[string](t, s, q)
		bt, err1 := time.Parse(time.RFC3339Nano, bv)
		st, err2 := time.Parse(time.RFC3339Nano, sv)
		if err1 != nil || err2 != nil {
			t.Fatalf("%s: unparseable %q / %q", q, bv, sv)
		}
		if st.Sub(bt) != 240*time.Hour {
			t.Errorf("%s: %s → %s, want +10 days", q, bv, sv)
		}
	}
	// Fixed-date rows (anything not seeded relative to now) do not move.
	if bv, sv := scalar[string](t, b, `SELECT MIN(captured_at) FROM social_snapshots`), scalar[string](t, s, `SELECT MIN(captured_at) FROM social_snapshots`); bv != sv {
		t.Errorf("social_snapshots moved: %s → %s", bv, sv)
	}
}

func TestDemoFixturesCarryNoPrivateData(t *testing.T) {
	private := regexp.MustCompile(`(?i)bennett|spooner|glanville|merydian|agency accelerant|fanbasis|tail090dce`)
	err := fs.WalkDir(demoFS, "demo", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, _ := demoFS.ReadFile(path)
		if m := private.Find(raw); m != nil {
			t.Errorf("%s mentions %q", path, m)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// G-Brain is gone: the demo's data names the Optimal Engine ("Brain").
func TestDemoFixturesNeverMentionGBrain(t *testing.T) {
	raw, err := demoFS.ReadFile("demo/founderos-os.sql")
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`(?i)g-?brain`).Find(raw); m != nil {
		t.Fatalf("demo data still mentions %q", m)
	}
}
