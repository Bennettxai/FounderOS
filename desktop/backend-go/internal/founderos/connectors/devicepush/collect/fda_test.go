package collect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Without Full Disk Access the collector must not touch the macOS-protected
// sources at all: opening them raises a privacy prompt every tick (and an
// unanswered one hung the mini for 19 hours). It says what to grant instead.
func TestWithoutFullDiskAccessProtectedSourcesAreSkippedNotOpened(t *testing.T) {
	cfg := fakeMac(t)
	// both would block forever if opened
	cfg.WhatsAppDBs = []string{stuckFile(t, filepath.Join(cfg.Home, "Library", "Group Containers", "group.net.whatsapp.WhatsApp.shared", "ChatStorage.sqlite"))}
	stuckFile(t, filepath.Join(cfg.ObsidianVault, "stuck.md"))
	cfg.FullDiskAccess = func() bool { return false }
	cfg.SourceTimeout = 2 * time.Second

	c := NewCollector(cfg)
	start := time.Now()
	p := c.Collect(context.Background(), now)
	if took := time.Since(start); took > time.Second {
		t.Fatalf("protected sources were opened (took %s)", took)
	}
	if got := c.TimedOut(); len(got) != 0 {
		t.Fatalf("nothing should time out: %v", got)
	}
	for _, s := range p.Statuses {
		if s.ID == "whatsapp" || s.ID == "obsidian" {
			if s.State != connectors.StateError || !strings.Contains(s.Detail, "Full Disk Access") {
				t.Fatalf("%s = %+v, want an honest 'needs Full Disk Access' error", s.ID, s)
			}
		}
	}
	if p.WhatsApp != nil || p.Obsidian != nil {
		t.Fatalf("no data from skipped sources: %+v %+v", p.WhatsApp, p.Obsidian)
	}
}

func TestHasFullDiskAccessReadsTheTCCDatabaseWithoutPrompting(t *testing.T) {
	home := t.TempDir()
	if !HasFullDiskAccess(home) {
		t.Fatal("no TCC database (not a protected Mac): nothing to guard")
	}
	db := filepath.Join(home, "Library", "Application Support", "com.apple.TCC", "TCC.db")
	write(t, db, "x")
	if !HasFullDiskAccess(home) {
		t.Fatal("a readable TCC.db means access")
	}
	if os.Getuid() == 0 {
		t.Skip("root reads everything")
	}
	if err := os.Chmod(db, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(db, 0o600) })
	if HasFullDiskAccess(home) {
		t.Fatal("a refused TCC.db (EPERM/EACCES) means no Full Disk Access")
	}
}

func TestSkippedForAccessNamesTheSkippedSources(t *testing.T) {
	cfg := fakeMac(t)
	cfg.FullDiskAccess = func() bool { return false }
	p := NewCollector(cfg).Collect(context.Background(), now)
	if got := SkippedForAccess(p); strings.Join(got, ",") != "obsidian,whatsapp" {
		t.Fatalf("skipped = %v", got)
	}
	cfg.FullDiskAccess = func() bool { return true }
	if got := SkippedForAccess(NewCollector(cfg).Collect(context.Background(), now)); len(got) != 0 {
		t.Fatalf("with access nothing is skipped: %v", got)
	}
}
