package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

type stubAgent struct{ id string }

func (s stubAgent) Meta() agents.Meta {
	return agents.Meta{ID: s.id, Name: "Stub " + s.id, DepartmentID: "tech"}
}
func (s stubAgent) Run(context.Context) (agents.Result, error) {
	return agents.Result{OK: true, Summary: "ran " + s.id}, nil
}

func authed(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Cookie", "session=ok")
	return req
}

func TestAgentRoutes(t *testing.T) {
	rt := agents.New(agents.NewMemStore(), stubAgent{"payments-pulse"}, stubAgent{"crm-pulse"})
	r := router(t, &Deps{Board: connectors.NewRegistry(), Agents: rt})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/founderos/agents/payments-pulse/run", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("run without session: %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/agents/payments-pulse/run", nil))
	var run agents.Run
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &run) != nil || !run.OK || run.Summary != "ran payments-pulse" {
		t.Fatalf("run: %d %s", w.Code, w.Body)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/agents/ghost/run", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown agent: %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/agents", nil))
	var list struct {
		Agents []struct {
			agents.Meta
			LastRun *agents.Run `json:"lastRun"`
		} `json:"agents"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &list) != nil || len(list.Agents) != 2 {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	if list.Agents[0].ID != "payments-pulse" || list.Agents[0].LastRun == nil || list.Agents[1].LastRun != nil {
		t.Fatalf("last runs wrong: %+v", list.Agents)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/agents/payments-pulse/runs", nil))
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"agentId":"payments-pulse"`)) {
		t.Fatalf("runs: %d %s", w.Code, w.Body)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/agents/broadcast", []byte(`{"message":"status?"}`)))
	var b agents.Broadcast
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &b) != nil || len(b.Replies) != 2 {
		t.Fatalf("broadcast: %d %s", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/agents/broadcast", []byte(`{"message":"  "}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty broadcast: %d", w.Code)
	}
}

func TestAgentRoutesSayWhenTheRuntimeIsMissing(t *testing.T) {
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodGet, "/api/founderos/agents", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("no runtime: %d", w.Code)
	}
}
