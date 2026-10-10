package metaads

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func newConnector(t *testing.T, envLocal string, files ...string) *Connector {
	t.Helper()
	t.Setenv("META_ADS_ACCESS_TOKEN", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "env.local")
	os.WriteFile(p, []byte(envLocal), 0o600)
	c := New(connectors.Resolver{EnvLocal: p})
	c.Files = []string{filepath.Join(dir, "absent.env")}
	for i, body := range files {
		f := filepath.Join(dir, "cred"+string(rune('a'+i))+".env")
		os.WriteFile(f, []byte(body), 0o600)
		if i == 0 {
			c.Files = nil
		}
		c.Files = append(c.Files, f)
	}
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "meta-ads" || Meta.Name != "Meta Ads" || Meta.Kind != connectors.KindAds {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestStatusNotConfigured(t *testing.T) {
	s := newConnector(t, "").Status(context.Background())
	if s.State != connectors.StateNotConfigured || s.ID != "meta-ads" {
		t.Fatalf("status = %+v", s)
	}
	if s.Detail != "Paid-funnel attribution (ad → opt-in → purchase) for Vantage + Launchpad Cohort. Set META_ADS_ACCESS_TOKEN to wire the Meta Ads MCP." {
		t.Errorf("detail = %q", s.Detail)
	}
}

// Status-only by design: a token is reported as present, and no call is made.
func TestStatusKeyedMakesNoCall(t *testing.T) {
	orig := http.DefaultTransport
	http.DefaultTransport = failRT{t}
	defer func() { http.DefaultTransport = orig }()

	c := newConnector(t, "", "OTHER=1\n", "META_ADS_ACCESS_TOKEN=EAAB-test\n") // second fallback file (social-media)
	s := c.Status(context.Background())
	if s.State != connectors.StateConnected || s.Meta["keyed"] != "yes" {
		t.Fatalf("status = %+v", s)
	}
	if !strings.HasPrefix(s.Detail, "META_ADS_ACCESS_TOKEN present") {
		t.Errorf("detail = %q", s.Detail)
	}
}

type failRT struct{ t *testing.T }

func (f failRT) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("Meta Ads status must not call out, got %s", r.URL)
	return nil, http.ErrHandlerTimeout
}
