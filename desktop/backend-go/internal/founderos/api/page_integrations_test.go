package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/integrations"
)

// integrationsProbeConn counts its status checks.
type integrationsProbeConn struct {
	st    connectors.Status
	calls *int32
}

func (p integrationsProbeConn) Status(context.Context) connectors.Status {
	atomic.AddInt32(p.calls, 1)
	return p.st
}

func integrationsDeps(t *testing.T) (*Deps, string, *int32) {
	t.Helper()
	envLocal := filepath.Join(t.TempDir(), "env.local")
	reg := connectors.NewRegistry()
	var fathomCalls int32
	reg.Register(connectors.Meta{ID: "slack", Name: "Slack", Kind: connectors.KindSlack}, fakeConn{connectors.Status{State: connectors.StateConnected, Detail: "workspace ok"}})
	reg.Register(connectors.Meta{ID: "zernio", Name: "Zernio", Kind: connectors.KindSocial}, fakeConn{connectors.Status{State: connectors.StateError, Detail: "HTTP 500"}})
	reg.Register(connectors.Meta{ID: "fathom", Name: "Fathom", Kind: connectors.KindCRM}, integrationsProbeConn{st: connectors.Status{State: connectors.StateConnected, Detail: "3 meetings"}, calls: &fathomCalls})
	reg.Register(connectors.Meta{ID: "whatsapp", Name: "WhatsApp", Kind: connectors.KindSocial}, fakeConn{connectors.Status{State: connectors.StateNotConfigured, Detail: "no collector has pushed WhatsApp yet"}})
	return &Deps{Board: reg, Resolver: connectors.Resolver{EnvLocal: envLocal}}, envLocal, &fathomCalls
}

func integrationsReq(method, path string, body any) *http.Request {
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func integrationsServe(t *testing.T, d *Deps, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router(t, d).ServeHTTP(w, req)
	return w
}

func TestIntegrationsPagesConnectionsNeedsASession(t *testing.T) {
	d, _, _ := integrationsDeps(t)
	w := httptest.NewRecorder()
	router(t, d).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/connections", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
}

func TestIntegrationsPagesConnectionsReturnsBoardCatalogVolumeAndSlots(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	if err := os.WriteFile(envLocal, []byte("FATHOM_API_KEY=fth-secret-9\nDISCORD_API_KEY=dsc-secret-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := integrationsServe(t, d, integrationsReq(http.MethodGet, "/api/founderos/pages/connections", nil))
	if w.Code != 200 {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "fth-secret-9") || strings.Contains(w.Body.String(), "dsc-secret-1") {
		t.Fatal("the board leaked a stored value")
	}
	var body struct {
		Connections []connectors.Status                    `json:"connections"`
		Catalog     []integrations.Entry                   `json:"catalog"`
		Categories  []integrations.CategoryGroup           `json:"categories"`
		Volume      integrations.VolumeModel               `json:"volume"`
		Keys        []integrations.SlotStatus              `json:"keys"`
		OAuth       map[string]integrations.OAuthReadiness `json:"oauth"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Connections) != 4 || len(body.Catalog) != len(integrations.Catalog) || len(body.Categories) == 0 {
		t.Fatalf("shape: %d conns, %d catalog, %d cats", len(body.Connections), len(body.Catalog), len(body.Categories))
	}
	by := map[string]integrations.Entry{}
	for _, e := range body.Catalog {
		by[e.Slug] = e
	}
	if !by["slack"].Connected || by["zernio"].Connected || !by["fathom"].Connected || !by["fathom"].KeySaved {
		t.Fatalf("merge wrong: slack %+v zernio %+v fathom %+v", by["slack"], by["zernio"], by["fathom"])
	}
	if !by["discord"].KeySaved || by["discord"].Connected {
		t.Fatal("a saved key on a connector-less tile is stored, not connected")
	}
	if body.Volume.Counts.Connected != 2 || body.Volume.Counts.Error != 1 || body.Volume.Counts.Total != 4 {
		t.Fatalf("volume counts %+v", body.Volume.Counts)
	}
	present := map[string]bool{}
	for _, k := range body.Keys {
		present[k.EnvVar] = k.Present
	}
	if !present["FATHOM_API_KEY"] || present["STRIPE_SECRET_KEY"] {
		t.Fatalf("slot presence %v", present)
	}
	if _, ok := body.OAuth["github"]; !ok {
		t.Fatal("OAuth readiness missing for github")
	}
	if _, ok := body.OAuth["fathom"]; ok {
		t.Fatal("non-OAuth tile got readiness")
	}
}

func TestIntegrationsConnectSavesOnlyDeclaredKeysAndNeverEchoes(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", map[string]any{"slug": "fathom", "values": map[string]string{"FATHOM_API_KEY": " ntn_secret_123 "}}))
	if w.Code != 200 || strings.Contains(w.Body.String(), "ntn_secret_123") {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	var ok struct{ OK, KeySaved bool }
	_ = json.Unmarshal(w.Body.Bytes(), &ok)
	if !ok.OK || !ok.KeySaved {
		t.Fatalf("body %s", w.Body)
	}
	if integrations.ReadEnv(envLocal)["FATHOM_API_KEY"] != "ntn_secret_123" {
		t.Fatal("key not saved (trimmed) to env.local")
	}

	big := strings.Repeat("QUJD", 600) // ~2.4KB base64 RSA key
	w = integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", map[string]any{"slug": "docusign", "values": map[string]string{
		"DOCUSIGN_INTEGRATION_KEY": "ik", "DOCUSIGN_USER_ID": "uid", "DOCUSIGN_ACCOUNT_ID": "aid", "DOCUSIGN_PRIVATE_KEY_B64": big,
	}}))
	if w.Code != 200 || integrations.ReadEnv(envLocal)["DOCUSIGN_PRIVATE_KEY_B64"] != big {
		t.Fatalf("multi-kilobyte key: %d %s", w.Code, w.Body)
	}

	if w = integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", map[string]any{"slug": "discord", "values": map[string]string{"DISCORD_API_KEY": "dsc-1"}})); w.Code != 200 {
		t.Fatalf("generic key: %d", w.Code)
	}

	for name, body := range map[string]any{
		"unknown slug":    map[string]any{"slug": "not-a-tool", "values": map[string]string{"X_API_KEY": "v"}},
		"foreign key":     map[string]any{"slug": "fathom", "values": map[string]string{"ATTIO_API_KEY": "steal"}},
		"multi-line":      map[string]any{"slug": "fathom", "values": map[string]string{"FATHOM_API_KEY": "a\nb"}},
		"blank":           map[string]any{"slug": "fathom", "values": map[string]string{"FATHOM_API_KEY": "   "}},
		"no values":       map[string]any{"slug": "fathom", "values": map[string]string{}},
		"guidance only":   map[string]any{"slug": "whatsapp", "values": map[string]string{"WHATSAPP_API_KEY": "x"}},
		"too long":        map[string]any{"slug": "fathom", "values": map[string]string{"FATHOM_API_KEY": strings.Repeat("a", 4097)}},
		"unknown slot":    map[string]any{"slot": "PATH", "value": "/tmp"},
		"slot multi-line": map[string]any{"slot": "SLACK_BOT_TOKEN", "value": "a\rb"},
	} {
		if w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", body)); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", name, w.Code)
		}
	}
	env := integrations.ReadEnv(envLocal)
	if _, ok := env["ATTIO_API_KEY"]; ok {
		t.Fatal("a foreign key was written")
	}
	if _, ok := env["PATH"]; ok {
		t.Fatal("an arbitrary env var was written")
	}
}

func TestIntegrationsConnectSavesAKeySlot(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", map[string]any{"slot": "CLAUDE_OAUTH_TOKEN", "value": "sk-ant-oat-xyz"}))
	if w.Code != 200 || strings.Contains(w.Body.String(), "sk-ant") {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if integrations.ReadEnv(envLocal)["CLAUDE_OAUTH_TOKEN"] != "sk-ant-oat-xyz" {
		t.Fatal("slot not saved")
	}
}

func TestIntegrationsDisconnectRemovesExactlyThatIntegrationsKeys(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	_ = integrations.UpsertEnv(envLocal, map[string]string{"FATHOM_API_KEY": "k1", "DISCORD_API_KEY": "k2"})
	w := integrationsServe(t, d, integrationsReq(http.MethodDelete, "/api/founderos/pages/connections/connect", map[string]any{"slug": "fathom"}))
	var body struct{ OK, KeySaved bool }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || !body.OK || body.KeySaved {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	env := integrations.ReadEnv(envLocal)
	if _, ok := env["FATHOM_API_KEY"]; ok || env["DISCORD_API_KEY"] != "k2" {
		t.Fatalf("env after disconnect %v", env)
	}
	if w := integrationsServe(t, d, integrationsReq(http.MethodDelete, "/api/founderos/pages/connections/connect", map[string]any{"slug": "nope"})); w.Code != 400 {
		t.Fatalf("unknown slug: %d", w.Code)
	}
}

func TestIntegrationsConnectRefusesWithoutAnEnvLocal(t *testing.T) {
	d, _, _ := integrationsDeps(t)
	d.Resolver = connectors.Resolver{}
	w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/connections/connect", map[string]any{"slug": "fathom", "values": map[string]string{"FATHOM_API_KEY": "k"}}))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no env.local configured: %d", w.Code)
	}
}

func TestIntegrationsKeyTestRunsOnlyThatConnectorAndTimesIt(t *testing.T) {
	d, _, calls := integrationsDeps(t)
	w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/admin/keys/test", map[string]string{"envVar": "FATHOM_API_KEY"}))
	var body struct {
		OK     bool   `json:"ok"`
		State  string `json:"state"`
		Detail string `json:"detail"`
		MS     *int64 `json:"ms"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || !body.OK || body.State != "connected" || body.Detail != "3 meetings" || body.MS == nil {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if atomic.LoadInt32(calls) != 1 {
		t.Fatalf("fathom checked %d times, want 1", *calls)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("test result must not be cached")
	}

	for name, tc := range map[string]struct {
		body map[string]string
		want int
	}{
		"bad name":     {map[string]string{"envVar": "lower"}, 400},
		"unknown slot": {map[string]string{"envVar": "NOT_A_SLOT"}, 400},
		"no connector": {map[string]string{"envVar": "CLAUDE_OAUTH_TOKEN"}, 400},
		"not on board": {map[string]string{"envVar": "STRIPE_SECRET_KEY"}, 400},
	} {
		if w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/admin/keys/test", tc.body)); w.Code != tc.want {
			t.Errorf("%s: %d, want %d (%s)", name, w.Code, tc.want, w.Body)
		}
	}
}

func TestIntegrationsKeyTestReportsAFailingConnectorAsNotOK(t *testing.T) {
	d, _, _ := integrationsDeps(t)
	d.Board.Register(connectors.Meta{ID: "typeform", Name: "Typeform"}, fakeConn{connectors.Status{State: connectors.StateError, Detail: "HTTP 401"}})
	w := integrationsServe(t, d, integrationsReq(http.MethodPost, "/api/founderos/pages/admin/keys/test", map[string]string{"envVar": "TYPEFORM_API_KEY"}))
	var body struct {
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.OK || body.Detail != "HTTP 401" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}

// Disconnect must actually stop the key resolving: it leaves planted.env,
// env.local and the process env (loadPlanted copied it there at boot).
func TestIntegrationsDisconnectStopsTheKeyResolving(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	planted := filepath.Join(filepath.Dir(envLocal), "planted.env")
	d.Resolver.Planted = planted
	_ = integrations.UpsertEnv(planted, map[string]string{"FATHOM_API_KEY": "k1", "DISCORD_API_KEY": "k2"})
	_ = integrations.UpsertEnv(envLocal, map[string]string{"FATHOM_API_KEY": "k1", "OTHER": "1"})
	t.Setenv("FATHOM_API_KEY", "k1")

	w := integrationsServe(t, d, integrationsReq(http.MethodDelete, "/api/founderos/pages/connections/connect", map[string]any{"slug": "fathom"}))
	var body struct{ OK, KeySaved bool }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || !body.OK || body.KeySaved {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if got := d.Resolver.Resolve("FATHOM_API_KEY"); got != "" {
		t.Fatalf("FATHOM_API_KEY still resolves to %q after disconnect", got)
	}
	if _, set := os.LookupEnv("FATHOM_API_KEY"); set {
		t.Error("the process env still holds FATHOM_API_KEY")
	}
	if p := integrations.ReadEnv(planted); p["DISCORD_API_KEY"] != "k2" {
		t.Errorf("planted.env lost an unrelated key: %v", p)
	}
	if e := integrations.ReadEnv(envLocal); e["OTHER"] != "1" {
		t.Errorf("env.local lost an unrelated key: %v", e)
	}
}

// A symlinked env.local (the operator's machine: it points into the dev checkout)
// is never written through, so the response says honestly that the key still
// resolves from it instead of claiming a disconnect.
func TestIntegrationsDisconnectReportsAKeyStillResolvingFromASymlinkedEnvLocal(t *testing.T) {
	d, envLocal, _ := integrationsDeps(t)
	dir := filepath.Dir(envLocal)
	planted := filepath.Join(dir, "planted.env")
	d.Resolver.Planted = planted
	_ = integrations.UpsertEnv(planted, map[string]string{"FATHOM_API_KEY": "k1"})
	target := filepath.Join(t.TempDir(), "dev-checkout.env.local")
	if err := os.WriteFile(target, []byte("FATHOM_API_KEY=k1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, envLocal); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FATHOM_API_KEY", "k1")

	w := integrationsServe(t, d, integrationsReq(http.MethodDelete, "/api/founderos/pages/connections/connect", map[string]any{"slug": "fathom"}))
	var body struct {
		OK            bool     `json:"ok"`
		KeySaved      bool     `json:"keySaved"`
		Error         string   `json:"error"`
		StillResolves []string `json:"stillResolves"`
	}
	if json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("body %s", w.Body)
	}
	if w.Code != http.StatusConflict || body.OK || !body.KeySaved || len(body.StillResolves) != 1 || body.StillResolves[0] != "FATHOM_API_KEY" || !strings.Contains(body.Error, "~/.founderos/.env (a symlink FounderOS will not write through)") {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if raw, _ := os.ReadFile(target); string(raw) != "FATHOM_API_KEY=k1\n" {
		t.Errorf("wrote through the env.local symlink: %q", raw)
	}
	if fi, err := os.Lstat(envLocal); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("env.local symlink replaced")
	}
	if _, ok := integrations.ReadEnv(planted)["FATHOM_API_KEY"]; ok {
		t.Error("planted.env still holds the key")
	}
	if _, set := os.LookupEnv("FATHOM_API_KEY"); set {
		t.Error("the process env still holds FATHOM_API_KEY")
	}
}
