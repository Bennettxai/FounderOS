package integrations

import (
	"os"
	"path/filepath"
	"testing"
)

// ~/.founderos-bridge/env.local is a symlink into a dev checkout; a key saved
// on the bridge must never be written through it (security review #4).
func TestEnvWritersRefuseSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "other-checkout.env")
	if err := os.WriteFile(target, []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "env.local")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := UpsertEnv(link, map[string]string{"B": "2"}); err == nil {
		t.Fatal("UpsertEnv wrote through a symlink")
	}
	if err := RemoveEnv(link, []string{"A"}); err == nil {
		t.Fatal("RemoveEnv wrote through a symlink")
	}
	raw, _ := os.ReadFile(target)
	if string(raw) != "A=1\n" {
		t.Fatalf("the other checkout's file changed: %q", raw)
	}
}
