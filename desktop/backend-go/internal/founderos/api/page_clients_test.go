package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/superset"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	clientspage "github.com/rhl/businessos-backend/internal/founderos/pages/clients"
)

// Ported from FounderOS v1 tests/client-work-route.test.ts, plus the page
// payload GET /api/founderos/pages/clients.

type clientsMemStore struct {
	mu       sync.Mutex
	rows     []clientspage.Work
	err      error
	claimed  []string
	finished map[string]*string
}

func (m *clientsMemStore) All(context.Context) ([]clientspage.Work, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]clientspage.Work{}, m.rows...), nil
}

func (m *clientsMemStore) Get(_ context.Context, id string) (*clientspage.Work, error) {
	for i := range m.rows {
		if m.rows[i].ID == id {
			w := m.rows[i]
			return &w, nil
		}
	}
	return nil, m.err
}

func (m *clientsMemStore) Add(_ context.Context, w clientspage.Work) error {
	if m.err != nil {
		return m.err
	}
	m.rows = append([]clientspage.Work{w}, m.rows...)
	return nil
}

func (m *clientsMemStore) Claim(_ context.Context, id string) (bool, error) {
	m.claimed = append(m.claimed, id)
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].Status == clientspage.StatusSaved {
			m.rows[i].Status = clientspage.StatusLaunching
			return true, nil
		}
	}
	return false, nil
}

func (m *clientsMemStore) Finish(_ context.Context, id string, ws *string) error {
	if m.finished == nil {
		m.finished = map[string]*string{}
	}
	m.finished[id] = ws
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].WorkspaceID = ws
			if ws != nil {
				m.rows[i].Status = clientspage.StatusLaunched
			} else {
				m.rows[i].Status = clientspage.StatusNeedsAttention
			}
		}
	}
	return nil
}

type clientsFakeLauncher struct {
	calls int
	id    string
	err   error
}

func (f *clientsFakeLauncher) ListProjects(context.Context) ([]superset.Project, error) {
	return []superset.Project{{ID: "2ffe9fec-9041-4577-83c3-72902a6aae98", Name: "Silvio-Big Mamas"}}, nil
}

func (f *clientsFakeLauncher) CreateWorkspace(context.Context, superset.WorkspaceSpec) (string, error) {
	f.calls++
	return f.id, f.err
}

const clientsTestID = "ea6f9168-b554-41c3-b2c8-1cbfe51021ff"

var clientsToken = strings.Repeat("x", 32)

func clientsHarness(t *testing.T, store *clientsMemStore, l *clientsFakeLauncher, token string) http.Handler {
	t.Helper()
	oldStore, oldLaunch := clientsStoreFor, clientsLauncherFor
	clientsStoreFor = func(*Deps) clientspage.Store { return store }
	clientsLauncherFor = func(*Deps) clientspage.Launcher { return l }
	t.Cleanup(func() { clientsStoreFor, clientsLauncherFor = oldStore, oldLaunch })
	env := filepath.Join(t.TempDir(), "env.local")
	body := ""
	if token != "" {
		body = "CLIENT_WORK_OPERATOR_TOKEN=" + token + "\n"
	}
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLIENT_WORK_OPERATOR_TOKEN", "")
	return router(t, &Deps{Resolver: connectors.Resolver{EnvLocal: env}})
}

func clientsDo(t *testing.T, h http.Handler, method, path string, body any, auth string) (int, map[string]any) {
	t.Helper()
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case string:
		rd = bytes.NewReader([]byte(b))
	default:
		raw, _ := json.Marshal(b)
		rd = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", "session=ok")
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	out := map[string]any{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestClientsPageRequiresSession(t *testing.T) {
	h := clientsHarness(t, &clientsMemStore{}, &clientsFakeLauncher{}, "")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/clients", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

func TestClientsPagePayload(t *testing.T) {
	store := &clientsMemStore{rows: []clientspage.Work{{ID: "a", ClientID: "silvio-big-mamas", Brief: "Menu refresh copy", Status: clientspage.StatusNeedsAttention, CreatedAt: "2026-09-21T10:00:00.000Z"}}}
	h := clientsHarness(t, store, &clientsFakeLauncher{}, "")
	code, body := clientsDo(t, h, http.MethodGet, "/api/founderos/pages/clients", nil, "")
	if code != 200 {
		t.Fatalf("%d %v", code, body)
	}
	if n := len(body["clients"].([]any)); n != 1 {
		t.Fatalf("clients %v", body["clients"])
	}
	if len(body["work"].([]any)) != 1 || body["windowDays"].(float64) != 14 {
		t.Fatalf("%v", body)
	}
	v := body["volume"].(map[string]any)
	if v["headline"].(float64) != 1 || len(v["series"].([]any)) != 14 || v["insight"].(map[string]any)["value"].(float64) != 1 {
		t.Fatalf("volume %v", v)
	}
	if len(body["statusOrder"].([]any)) != 4 || body["slackBridge"] != "not activated" {
		t.Fatalf("%v", body)
	}
	if _, ok := body["launchEnabled"].(bool); !ok {
		t.Fatalf("launchEnabled missing: %v", body)
	}
}

func TestClientsPageUnreadableStoreIsAnErrorNotEmpty(t *testing.T) {
	h := clientsHarness(t, &clientsMemStore{err: errors.New("db down")}, &clientsFakeLauncher{}, "")
	code, body := clientsDo(t, h, http.MethodGet, "/api/founderos/pages/clients", nil, "")
	if code != http.StatusServiceUnavailable || !strings.Contains(body["error"].(string), "db down") {
		t.Fatalf("%d %v", code, body)
	}
}

func TestClientsWorkSavesAValidatedBriefWithoutLaunching(t *testing.T) {
	store, l := &clientsMemStore{}, &clientsFakeLauncher{}
	h := clientsHarness(t, store, l, "")
	code, body := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "save", "clientId": "silvio-big-mamas", "brief": "  Draft five social posts  "}, "")
	if code != http.StatusCreated {
		t.Fatalf("%d %v", code, body)
	}
	w := body["work"].(map[string]any)
	if w["status"] != "saved" || w["brief"] != "Draft five social posts" || len(store.rows) != 1 || l.calls != 0 {
		t.Fatalf("%v %v", w, store.rows)
	}
}

func TestClientsWorkRejectsBadInput(t *testing.T) {
	h := clientsHarness(t, &clientsMemStore{}, &clientsFakeLauncher{}, "")
	for _, b := range []any{
		"not json",
		map[string]any{"action": "save", "clientId": "silvio-big-mamas", "brief": "hi"},
		map[string]any{"action": "save", "clientId": "silvio-big-mamas", "brief": strings.Repeat("y", 6001)},
		map[string]any{"action": "launch", "id": "not-a-uuid"},
		map[string]any{"action": "delete"},
	} {
		if code, body := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", b, ""); code != http.StatusBadRequest {
			t.Fatalf("%v → %d %v", b, code, body)
		}
	}
	if code, _ := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "save", "clientId": "nobody", "brief": "Draft five posts"}, ""); code != http.StatusNotFound {
		t.Fatalf("unknown client %d", code)
	}
	if code, _ := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", strings.Repeat("z", 30001), ""); code != http.StatusRequestEntityTooLarge {
		t.Fatalf("too large %d", code)
	}
}

func TestClientsWorkLaunchRequiresTheOperatorSecret(t *testing.T) {
	store := &clientsMemStore{rows: []clientspage.Work{{ID: clientsTestID, ClientID: "silvio-big-mamas", Brief: "Draft copy", Status: clientspage.StatusSaved}}}
	for _, tc := range []struct{ configured, supplied string }{{"", clientsToken}, {"short", "short"}, {clientsToken, ""}, {clientsToken, strings.Repeat("y", 32)}} {
		h := clientsHarness(t, store, &clientsFakeLauncher{}, tc.configured)
		if code, _ := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": clientsTestID}, tc.supplied); code != http.StatusUnauthorized {
			t.Fatalf("%+v → %d", tc, code)
		}
	}
	if len(store.claimed) != 0 {
		t.Fatal("claimed without the secret")
	}
}

func TestClientsWorkLaunchIsRefusedWhileBridgeWritesAreOff(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "0")
	guard.Reset()
	store := &clientsMemStore{rows: []clientspage.Work{{ID: clientsTestID, ClientID: "silvio-big-mamas", Brief: "Draft copy", Status: clientspage.StatusSaved}}}
	l := &clientsFakeLauncher{id: "ws-1"}
	h := clientsHarness(t, store, l, clientsToken)
	code, body := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": clientsTestID}, clientsToken)
	if code != http.StatusServiceUnavailable || !strings.Contains(body["error"].(string), "FOUNDEROS_WRITES") {
		t.Fatalf("%d %v", code, body)
	}
	if len(store.claimed) != 0 || l.calls != 0 || store.rows[0].Status != clientspage.StatusSaved {
		t.Fatal("a refused launch must leave the request saved and untouched")
	}
	found := false
	for _, r := range guard.Refused() {
		if r.Action == "clients.work.launch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("refusal not logged: %v", guard.Refused())
	}
}

func TestClientsWorkLaunchNotFoundAndNoRelaunch(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	store := &clientsMemStore{rows: []clientspage.Work{{ID: clientsTestID, ClientID: "silvio-big-mamas", Brief: "Draft copy", Status: clientspage.StatusLaunching}}}
	l := &clientsFakeLauncher{id: "ws-1"}
	h := clientsHarness(t, store, l, clientsToken)
	if code, _ := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": "0a6f9168-b554-41c3-b2c8-1cbfe51021ff"}, clientsToken); code != http.StatusNotFound {
		t.Fatalf("missing → %d", code)
	}
	if code, _ := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": clientsTestID}, clientsToken); code != http.StatusConflict {
		t.Fatalf("claimed → %d", code)
	}
	if l.calls != 0 {
		t.Fatal("relaunched")
	}
}

func TestClientsWorkLaunchRecordsAnUncertainLaunchWithoutLeaking(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	store := &clientsMemStore{rows: []clientspage.Work{{ID: clientsTestID, ClientID: "silvio-big-mamas", Brief: "Draft copy", Status: clientspage.StatusSaved}}}
	h := clientsHarness(t, store, &clientsFakeLauncher{err: errors.New("sensitive subprocess output")}, clientsToken)
	code, body := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": clientsTestID}, clientsToken)
	raw, _ := json.Marshal(body)
	if code != http.StatusBadGateway || strings.Contains(string(raw), "sensitive") {
		t.Fatalf("%d %s", code, raw)
	}
	if ws, ok := store.finished[clientsTestID]; !ok || ws != nil {
		t.Fatalf("finish not recorded as uncertain: %v", store.finished)
	}
}

func TestClientsWorkLaunchSucceeds(t *testing.T) {
	t.Setenv("FOUNDEROS_WRITES", "1")
	store := &clientsMemStore{rows: []clientspage.Work{{ID: clientsTestID, ClientID: "silvio-big-mamas", Brief: "Draft copy", Status: clientspage.StatusSaved}}}
	h := clientsHarness(t, store, &clientsFakeLauncher{id: "ws-1"}, clientsToken)
	code, body := clientsDo(t, h, http.MethodPost, "/api/founderos/pages/clients/work", map[string]any{"action": "launch", "id": clientsTestID}, clientsToken)
	if code != 200 || body["work"].(map[string]any)["status"] != "launched" || body["work"].(map[string]any)["workspaceId"] != "ws-1" {
		t.Fatalf("%d %v", code, body)
	}
}
