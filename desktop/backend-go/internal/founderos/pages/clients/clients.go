// Package clients is the /clients page logic ported from FounderOS v1:
// the confirmed client roster (lib/client-projects.ts), the Request Volume
// view-model (lib/clients-volume.ts), the client_work repo (lib/db.ts
// clientWork) over founderos_client_work, and the Superset draft launch
// (lib/client-work.ts).
package clients

// Project is one explicitly confirmed client project, not every payer or
// every local repo. Project ids were read from Superset on 2026-09-24. No
// fees or contract terms are inferred.
type Project struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Service     string   `json:"service"`
	ProjectName string   `json:"projectName"`
	ProjectID   string   `json:"projectId"`
	Context     string   `json:"context"`
	Sources     []string `json:"sources"`
	Retainer    string   `json:"retainer"`
}

var Projects = []Project{{
	ID: "silvio-big-mamas", Name: "Silvio / Big Mama's", Service: "Marketing",
	ProjectName: "Silvio-Big Mamas", ProjectID: "2ffe9fec-9041-4577-83c3-72902a6aae98",
	Context:  "Marketing for Big Mama's two restaurants. Use packages/brand as the source of business facts. Existing material is in marketing/ and docs/.",
	Sources:  []string{"packages/brand", "marketing/", "docs/", "CLAUDE.md"},
	Retainer: "Terms not recorded",
}}

func ProjectByID(id string) *Project {
	for i := range Projects {
		if Projects[i].ID == id {
			return &Projects[i]
		}
	}
	return nil
}

// Refs is the roster as the view-model takes it.
func Refs() []ClientRef {
	out := make([]ClientRef, 0, len(Projects))
	for _, p := range Projects {
		out = append(out, ClientRef{ID: p.ID, Name: p.Name})
	}
	return out
}

type Status string

const (
	StatusSaved          Status = "saved"
	StatusLaunching      Status = "launching"
	StatusLaunched       Status = "launched"
	StatusNeedsAttention Status = "needs_attention"
)

func (s Status) Valid() bool {
	switch s {
	case StatusSaved, StatusLaunching, StatusLaunched, StatusNeedsAttention:
		return true
	}
	return false
}

// Work is ClientWorkSchema: one client request. WorkspaceID is the Superset
// workspace id (founderos_client_work.superset_workspace_id).
type Work struct {
	ID          string  `json:"id"`
	ClientID    string  `json:"clientId"`
	Brief       string  `json:"brief"`
	Status      Status  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
	WorkspaceID *string `json:"workspaceId"`
	Detail      *string `json:"detail"`
}
