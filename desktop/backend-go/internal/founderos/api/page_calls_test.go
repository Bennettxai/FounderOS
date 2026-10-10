package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func TestPlaudIngestRouteRunsTheCallsDataAgent(t *testing.T) {
	rt := agents.New(agents.NewMemStore(), stubAgent{"sales-calls-data"})
	r := router(t, &Deps{Board: connectors.NewRegistry(), Agents: rt})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, authed(http.MethodPost, "/api/founderos/pages/plaud/ingest", nil))
	var run agents.Run
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &run) != nil || run.AgentID != "sales-calls-data" || !run.OK {
		t.Fatalf("ingest: %d %s", w.Code, w.Body)
	}
	runs, _ := rt.Recent(t.Context(), "sales-calls-data", 5)
	if len(runs) != 1 {
		t.Fatalf("the ingest must be a recorded agent run, got %d", len(runs))
	}
}

func TestCallArchiveNeedsTheDatabase(t *testing.T) {
	r := router(t, &Deps{Board: connectors.NewRegistry()})
	for _, m := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authed(m, "/api/founderos/pages/calls/archive", nil))
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s without a pool: %d", m, w.Code)
		}
	}
}
