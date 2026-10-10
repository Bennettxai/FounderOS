package collect

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// stuckFile is a named pipe: open(2) on it blocks until a writer appears,
// exactly how the mini's collector hung (threads parked in open, no fd, no
// socket) behind a macOS privacy prompt. Cleanup opens the writer end so the
// abandoned reader can finish.
func stuckFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			f.Close()
		}
	})
	return path
}

func TestCollectBoundsEachSourceSoOneStuckReadCannotHangTheTick(t *testing.T) {
	cfg := fakeMac(t)
	cfg.WisprDB = stuckFile(t, filepath.Join(t.TempDir(), "flow.sqlite"))
	stuckFile(t, filepath.Join(cfg.ObsidianVault, "stuck.md"))
	cfg.SourceTimeout = 300 * time.Millisecond

	c := NewCollector(cfg)
	start := time.Now()
	done := make(chan devicepush.Payload, 1)
	go func() { done <- c.Collect(context.Background(), now) }()
	var p devicepush.Payload
	select {
	case p = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Collect hung on a stuck source")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("sources must time out on their own budget, took %s", took)
	}
	if err := devicepush.Validate(p); err != nil {
		t.Fatal(err)
	}
	if got := c.TimedOut(); !reflect.DeepEqual(got, []string{"obsidian-notes", "wispr"}) {
		t.Fatalf("timed out = %v", got)
	}
	states := map[string]connectors.Status{}
	for _, s := range p.Statuses {
		states[s.ID] = s
	}
	if len(p.Statuses) != 4 {
		t.Fatalf("every status row is still pushed: %+v", p.Statuses)
	}
	if w := states["wispr"]; w.State != connectors.StateError || !strings.Contains(w.Detail, "timed out") || p.Wispr != nil {
		t.Fatalf("a stuck Wispr read is an honest error with no data: %+v %+v", w, p.Wispr)
	}
	if o := states["obsidian"]; o.State != connectors.StateConnected || p.Obsidian != nil {
		t.Fatalf("a stuck note read leaves the vault's notes unknown, its status intact: %+v %+v", o, p.Obsidian)
	}
	if states["whatsapp"].State != connectors.StateConnected || p.WhatsApp == nil || len(p.Usage) != 2 {
		t.Fatalf("the other sources still answer: %+v", p)
	}
	if got := c.InFlight(); !reflect.DeepEqual(got, []string{"obsidian-notes", "wispr"}) {
		t.Fatalf("the abandoned reads are still stuck and say so: %v", got)
	}
}
