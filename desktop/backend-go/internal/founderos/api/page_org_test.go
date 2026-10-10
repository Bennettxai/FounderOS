package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/org"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func TestOrgPageComposesRosterAndLiveBoard(t *testing.T) {
	board := &fakeBoard{agents: []paperclip.Agent{
		{ID: "b1", Name: "Conductor", Status: "running", Model: sp("claude-fable-5")},
		{ID: "b2", Name: "Sales", Status: "idle"},
		{ID: "b3", Name: "Forge", Status: "error"},
	}}
	d := pageDeps(t, board)
	ctx := context.Background()
	at := time.Now().UTC()
	if _, err := d.Pool.Exec(ctx, `INSERT INTO founderos_broadcasts (id, workspace_id, message, created_at) SELECT 'bc1', id, 'status?', $1 FROM workspaces WHERE slug='founderos'`, at); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Pool.Exec(ctx, `INSERT INTO founderos_broadcast_replies (id, workspace_id, broadcast_id, agent_id, ok, reply, finished_at) SELECT 'rp1', id, 'bc1', 'crm-pulse', true, 'all good', $1 FROM workspaces WHERE slug='founderos'`, at); err != nil {
		t.Fatal(err)
	}

	var v org.View
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/org", nil, &v); code != 200 {
		t.Fatalf("org: %d", code)
	}
	if v.Conductor == nil || v.Conductor.ID != "conductor" || v.Tree.TotalAgents != 3 {
		t.Fatalf("conductor/tree: %+v", v.Tree)
	}
	if !v.Live.Connected || v.Live.Conductor == nil || v.Live.ByDepartment["dept-sales"].ID != "b2" || len(v.Live.Extras) != 1 {
		t.Fatalf("live: %+v", v.Live)
	}
	if len(v.Crews) != 2 || v.Crews[0].Department.ID != "dept-sales" || v.Crews[0].Leads[0].ID != "sales-agent" || v.Crews[0].Pills[0].Agent.ID != "crm-pulse" {
		t.Fatalf("crews: %+v", v.Crews)
	}
	if v.LastBroadcast == nil || v.LastBroadcast.Message != "status?" || len(v.LastBroadcast.Replies) != 1 {
		t.Fatalf("broadcast: %+v", v.LastBroadcast)
	}
}

func TestOrgPageBoardDownStillServesTheRoster(t *testing.T) {
	d := pageDeps(t, &fakeBoard{down: true})
	var v org.View
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/org", nil, &v); code != 200 {
		t.Fatalf("org: %d", code)
	}
	if v.Live.Connected || v.Live.Error == "" || len(v.Crews) != 2 {
		t.Fatalf("board down: %+v", v.Live)
	}
}

func TestOrgPageWithoutBootstrapIs503(t *testing.T) {
	d := pageDeps(t, &fakeBoard{})
	if _, err := d.Pool.Exec(context.Background(), `DELETE FROM workspaces WHERE slug='founderos'`); err != nil {
		t.Fatal(err)
	}
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/org", nil, nil); code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", code)
	}
}

func TestDepartmentsRoute(t *testing.T) {
	d := pageDeps(t, nil)
	var body struct {
		Departments []osdata.Department `json:"departments"`
	}
	if code := getJSON(t, d, http.MethodGet, "/api/founderos/pages/departments", nil, &body); code != 200 || len(body.Departments) != 2 || body.Departments[0].ID != "dept-sales" {
		t.Fatalf("departments: %d %+v", code, body)
	}
}
