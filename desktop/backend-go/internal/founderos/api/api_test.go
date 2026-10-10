package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// FounderOS v1 lib/connectors/index.ts CHECKS (2026-09-30), with gbrain
// replaced by the Optimal Engine row: GBrain is retired on the bridge.
var founderosOSBoard = []string{
	"optimal-engine", "paperclip", "whatsapp", "zernio", "beehiiv", "manychat", "trakyo",
	"typeform", "fathom", "plaud", "docusign", "loom", "vidalytics", "meta-ads", "arcads",
	"wispr", "local-stack", "obsidian", "miro", "email", "calendar", "slack", "payments", "robinhood",
}

func TestBoardMatchesFounderosOSConnections(t *testing.T) {
	got := BoardIDs()
	want := append([]string(nil), founderosOSBoard...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("board has %d rows %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("board ids = %v, want %v", got, want)
		}
	}
}

type fakeConn struct{ st connectors.Status }

func (f fakeConn) Status(context.Context) connectors.Status { return f.st }

func router(t *testing.T, d *Deps) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	auth := func(c *gin.Context) {
		if c.GetHeader("Cookie") != "session=ok" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
	Register(api, d, auth)
	return r
}

func TestConnectionsRequiresSessionAndReturnsTheBoard(t *testing.T) {
	reg := connectors.NewRegistry()
	reg.Register(connectors.Meta{ID: "slack", Name: "Slack", Kind: connectors.KindSlack}, fakeConn{connectors.Status{State: connectors.StateConnected, Detail: "ok"}})
	r := router(t, &Deps{Board: reg})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/connections", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/connections", nil)
	req.Header.Set("Cookie", "session=ok")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body struct {
		Connections []connectors.Status `json:"connections"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || len(body.Connections) != 1 || body.Connections[0].ID != "slack" {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}

func TestDevicePushNeedsTheDeviceToken(t *testing.T) {
	t.Setenv("FOUNDEROS_DEVICE_TOKEN", "dev-secret")
	recv := devicepush.NewReceiver(devicepush.NewMemStore(), "macbook")
	r := router(t, &Deps{Board: connectors.NewRegistry(), Devices: recv})
	payload, _ := json.Marshal(map[string]any{"device": "macbook", "capturedAt": "2026-09-30T06:00:00Z"})

	for _, tc := range []struct {
		auth string
		want int
	}{{"", 401}, {"Bearer wrong", 401}} {
		req := httptest.NewRequest(http.MethodPost, "/api/founderos/device/push", bytes.NewReader(payload))
		if tc.auth != "" {
			req.Header.Set("Authorization", tc.auth)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("auth %q: %d, want %d", tc.auth, w.Code, tc.want)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/founderos/device/push", bytes.NewReader([]byte(`{not json`)))
	req.Header.Set("Authorization", "Bearer dev-secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad payload with a good token: %d, want 400", w.Code)
	}
}

func TestDevicePushIsClosedWhenNoTokenIsConfigured(t *testing.T) {
	t.Setenv("FOUNDEROS_DEVICE_TOKEN", "")
	r := router(t, &Deps{Board: connectors.NewRegistry(), Devices: devicepush.NewReceiver(devicepush.NewMemStore())})
	req := httptest.NewRequest(http.MethodPost, "/api/founderos/device/push", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unset token must close the route, got %d", w.Code)
	}
}

func TestGuardEndpointReportsFlags(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	t.Setenv("FOUNDEROS_CRONS", "0")
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/guard", nil)
	req.Header.Set("Cookie", "session=ok")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body struct {
		Writes bool `json:"writes"`
		Crons  bool `json:"crons"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Writes || body.Crons {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}

func TestCSRFExemptOnlyCoversTokenAuthenticatedMachineRoutes(t *testing.T) {
	for _, p := range []string{"/api/founderos/device/push", "/api/v1/founderos/device/push"} {
		if !CSRFExempt(p) {
			t.Errorf("%s authenticates with a bearer token and must skip CSRF", p)
		}
	}
	for _, p := range []string{"/api/founderos/connections", "/api/founderos/guard", "/api/founderos/device/pushx", "/api/founderos/device"} {
		if CSRFExempt(p) {
			t.Errorf("%s is a session route and must keep CSRF", p)
		}
	}
}

func TestPageRoutesRegisterThemselvesBehindAuth(t *testing.T) {
	RegisterPage(func(s *gin.RouterGroup, d *Deps) {
		s.GET("/pages/test-probe", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	})
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/test-probe", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("page route without session: %d", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/test-probe", nil)
	req.Header.Set("Cookie", "session=ok")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("page route with session: %d", w.Code)
	}
}

func TestBodyLimitRaisesOnlyTheStatementUploads(t *testing.T) {
	for _, p := range []string{"/api/founderos/pages/finances/statements", "/api/v1/founderos/pages/finances/bank-statement"} {
		if BodyLimit(p) != 25<<20 {
			t.Errorf("%s: %d", p, BodyLimit(p))
		}
	}
	if BodyLimit("/api/founderos/pages/finances") != 0 || BodyLimit("/api/chat") != 0 {
		t.Error("every other route keeps the server default")
	}
}
