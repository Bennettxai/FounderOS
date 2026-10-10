package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// The bridge's host device must be the id this box's collector pushes as:
// FOUNDEROS_HOST_DEVICE, else the collector's own FOUNDEROS_COLLECTOR_DEVICE
// (env.local, as the collector resolves it), else the hostname slug.
func TestHostDeviceMatchesThisBoxsCollector(t *testing.T) {
	envLocal := func(lines string) connectors.Resolver {
		p := filepath.Join(t.TempDir(), "env.local")
		if err := os.WriteFile(p, []byte(lines), 0o600); err != nil {
			t.Fatal(err)
		}
		return connectors.Resolver{EnvLocal: p}
	}
	t.Setenv("FOUNDEROS_HOST_DEVICE", "")
	t.Setenv("FOUNDEROS_COLLECTOR_DEVICE", "")
	if got := hostDeviceFrom(envLocal(""), "Alexs-Mac-mini.local"); got != "alexs-mac-mini" {
		t.Fatalf("hostname slug = %q", got)
	}
	if got := hostDeviceFrom(envLocal("FOUNDEROS_COLLECTOR_DEVICE=mini-box\n"), "Alexs-Mac-mini.local"); got != "mini-box" {
		t.Fatalf("the collector's device id must win over the hostname: %q", got)
	}
	if got := hostDeviceFrom(envLocal("FOUNDEROS_COLLECTOR_DEVICE=mini-box\nFOUNDEROS_HOST_DEVICE=the-host\n"), "x"); got != "the-host" {
		t.Fatalf("FOUNDEROS_HOST_DEVICE wins: %q", got)
	}
	t.Setenv("FOUNDEROS_HOST_DEVICE", "env-host")
	if got := hostDeviceFrom(envLocal(""), "x"); got != "env-host" {
		t.Fatalf("process env FOUNDEROS_HOST_DEVICE: %q", got)
	}
}
