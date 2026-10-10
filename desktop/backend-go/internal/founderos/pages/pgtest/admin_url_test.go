package pgtest

import (
	"os"
	"path/filepath"
	"testing"
)

// Without FOUNDEROS_PG_ADMIN_URL the tests use this checkout's own dev
// Postgres (POSTGRES_PORT in the repo's .env.dev, which dev-local.sh starts),
// never whatever else happens to listen on the upstream default port.
func TestAdminURLFollowsTheRepoDevPostgresPort(t *testing.T) {
	t.Setenv("FOUNDEROS_PG_ADMIN_URL", "")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env.dev"), []byte("BACKEND_PORT=8901\nPOSTGRES_PORT=25532\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "desktop", "backend-go", "internal", "x")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := adminURLFrom(deep), "postgres://postgres@127.0.0.1:25532/postgres?sslmode=disable"; got != want {
		t.Fatalf("adminURLFrom = %q, want %q", got, want)
	}
	if got := adminURLFrom(t.TempDir()); got != DefaultAdminURL {
		t.Fatalf("with no .env.dev: %q, want the default %q", got, DefaultAdminURL)
	}
}

func TestAdminURLPrefersTheEnvironment(t *testing.T) {
	t.Setenv("FOUNDEROS_PG_ADMIN_URL", "postgres://x@db:5432/postgres")
	if got := AdminURL(); got != "postgres://x@db:5432/postgres" {
		t.Fatalf("AdminURL = %q", got)
	}
}
