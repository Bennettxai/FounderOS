package plaud

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// A refresh token planted via /api/admin/keys resolves from planted.env first,
// so its rotation must land there: writing env.local instead would leave the
// revoked planted token winning the next resolve.
func TestPlantedRefreshTokenRotationUpsertsPlanted(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	f := newFake(t)
	f.rotateTo = "rt_next"
	c, s := newConn(t, f.srv.URL, "OTHER=1\n", "")
	planted := filepath.Join(filepath.Dir(s.envLocal), "planted.env")
	if err := os.WriteFile(planted, []byte("# planted\nPLAUD_REFRESH_TOKEN="+goodRefresh+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.res = connectors.Resolver{Planted: planted, EnvLocal: s.envLocal}

	if st := c.Status(context.Background()); st.State != connectors.StateConnected {
		t.Fatalf("status = %+v", st)
	}
	raw, _ := os.ReadFile(planted)
	if want := "# planted\nPLAUD_REFRESH_TOKEN=rt_next\n"; string(raw) != want {
		t.Errorf("planted.env =\n%s\nwant\n%s", raw, want)
	}
	if env, _ := os.ReadFile(s.envLocal); string(env) != "OTHER=1\n" {
		t.Errorf("env.local must be untouched, got\n%s", env)
	}
	if got := c.res.Resolve(refreshTokenKey); got != "rt_next" {
		t.Errorf("next resolve = %q, want the rotated token", got)
	}
}

// The existing rule: never write through a symlink (env.local points into a
// dev checkout on the operator's machine). The rotation stays in memory only.
func TestRotationRefusesToWriteThroughSymlinks(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	for _, which := range []string{"env.local", "planted.env"} {
		t.Run(which, func(t *testing.T) {
			f := newFake(t)
			f.rotateTo = "rt_next"
			c, s := newConn(t, f.srv.URL, "", "")
			dir := filepath.Dir(s.envLocal)
			target := filepath.Join(t.TempDir(), "other-checkout.env")
			body := "PLAUD_REFRESH_TOKEN=" + goodRefresh + "\n"
			if err := os.WriteFile(target, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(dir, which)
			_ = os.Remove(link)
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			c.res = connectors.Resolver{Planted: filepath.Join(dir, "planted.env"), EnvLocal: s.envLocal}

			if st := c.Status(context.Background()); st.State != connectors.StateConnected {
				t.Fatalf("status = %+v", st)
			}
			if raw, _ := os.ReadFile(target); string(raw) != body {
				t.Errorf("wrote through the %s symlink:\n%s", which, raw)
			}
			if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Errorf("%s symlink replaced", which)
			}
		})
	}
}
