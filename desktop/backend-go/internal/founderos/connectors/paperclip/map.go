package paperclip

// Pure, defensive mapping of board JSON (lib/connectors/paperclip.ts):
// malformed rows are skipped, never fatal, and unknown agent states collapse
// to idle because the wire is never trusted.

var agentStatuses = map[string]bool{"running": true, "idle": true, "paused": true, "error": true}

type Agent struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	AdapterType     *string `json:"adapterType"`
	Model           *string `json:"model"`
	LastHeartbeatAt *string `json:"lastHeartbeatAt"`
}

type OrgNode struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	Status   string  `json:"status"`
	Depth    int     `json:"depth"`
	ParentID *string `json:"parentId"`
}

type Issue struct {
	ID           string  `json:"id"`
	Identifier   string  `json:"identifier"`
	Title        string  `json:"title"`
	Status       string  `json:"status"`
	AssigneeName *string `json:"assigneeName"`
	UpdatedAt    *string `json:"updatedAt"`
}

type Run struct {
	ID         string  `json:"id"`
	AgentID    string  `json:"agentId"`
	AgentName  *string `json:"agentName"`
	Status     string  `json:"status"`
	StartedAt  *string `json:"startedAt"`
	FinishedAt *string `json:"finishedAt"`
}

type Comment struct {
	ID            string  `json:"id"`
	Body          string  `json:"body"`
	AuthorType    string  `json:"authorType"`
	AuthorAgentID *string `json:"authorAgentId"`
	CreatedAt     string  `json:"createdAt"`
}

// FailoverRun is a heartbeat run shaped for the failover loop: the failure
// text and the model that billed it (lib/agent-failover.ts FailoverRun).
type FailoverRun struct {
	AgentID    string  `json:"agentId"`
	Status     string  `json:"status"`
	Error      *string `json:"error"`
	Model      *string `json:"model"`
	FinishedAt *string `json:"finishedAt"`
	Summary    *string `json:"summary"`
}

// str is the TS str(): a non-empty string, else nil.
func str(v any) *string {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	return &s
}

func or(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asStatus(v any) string {
	if s, ok := v.(string); ok && agentStatuses[s] {
		return s
	}
	return "idle"
}

// listFrom accepts either a bare array or {key: [...]}.
func listFrom(body any, key string) []any {
	if arr, ok := body.([]any); ok {
		return arr
	}
	if arr, ok := obj(body)[key].([]any); ok {
		return arr
	}
	return nil
}

func MapAgents(raw []any) []Agent {
	out := []Agent{}
	for _, rec := range raw {
		r := obj(rec)
		id, idOK := r["id"].(string)
		name := str(r["name"])
		if r == nil || !idOK || name == nil {
			continue
		}
		// the board stores the model inside adapterConfig; top-level wins
		var model *string
		if m, ok := r["model"].(string); ok {
			model = &m
		} else if m, ok := obj(r["adapterConfig"])["model"].(string); ok {
			model = &m
		}
		out = append(out, Agent{
			ID: id, Name: *name, Status: asStatus(r["status"]),
			AdapterType: strAny(r["adapterType"]), Model: model, LastHeartbeatAt: strAny(r["lastHeartbeatAt"]),
		})
	}
	return out
}

// strAny keeps any string, empty included (the TS typeof === 'string').
func strAny(v any) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

// FlattenOrg turns the CEO→reports tree into preorder rows with depth and parent.
func FlattenOrg(raw []any) []OrgNode {
	out := []OrgNode{}
	var walk func(nodes []any, depth int, parent *string)
	walk = func(nodes []any, depth int, parent *string) {
		for _, rec := range nodes {
			r := obj(rec)
			id, idOK := r["id"].(string)
			name := str(r["name"])
			if r == nil || !idOK || name == nil {
				continue
			}
			role := "general"
			if s, ok := r["role"].(string); ok {
				role = s
			}
			out = append(out, OrgNode{ID: id, Name: *name, Role: role, Status: asStatus(r["status"]), Depth: depth, ParentID: parent})
			if reports, ok := r["reports"].([]any); ok {
				idCopy := id
				walk(reports, depth+1, &idCopy)
			}
		}
	}
	walk(raw, 0, nil)
	return out
}

// MapIssues maps board issues (the board calls tasks "issues").
func MapIssues(raw []any) []Issue {
	out := []Issue{}
	for _, rec := range raw {
		r := obj(rec)
		id, title := str(r["id"]), str(r["title"])
		if id == nil || title == nil {
			continue
		}
		ident := or(str(r["identifier"]), id)
		status := "unknown"
		if s := str(r["status"]); s != nil {
			status = *s
		}
		out = append(out, Issue{
			ID: *id, Identifier: *ident, Title: *title, Status: status,
			AssigneeName: or(str(r["assigneeName"]), str(obj(r["assignee"])["name"])),
			UpdatedAt:    str(r["updatedAt"]),
		})
	}
	return out
}

// MapRuns maps heartbeat runs; agent attribution is optional.
func MapRuns(raw []any) []Run {
	out := []Run{}
	for _, rec := range raw {
		r := obj(rec)
		agent := obj(r["agent"])
		id, agentID := str(r["id"]), or(str(r["agentId"]), str(agent["id"]))
		if id == nil || agentID == nil {
			continue
		}
		status := "unknown"
		if s := str(r["status"]); s != nil {
			status = *s
		}
		out = append(out, Run{
			ID: *id, AgentID: *agentID, AgentName: or(str(r["agentName"]), str(agent["name"])),
			Status: status, StartedAt: str(r["startedAt"]), FinishedAt: str(r["finishedAt"]),
		})
	}
	return out
}

// MapComments maps an issue thread; deleted and malformed rows are skipped.
func MapComments(raw []any) []Comment {
	out := []Comment{}
	for _, rec := range raw {
		r := obj(rec)
		if r == nil || truthy(r["deletedAt"]) {
			continue
		}
		id, body := str(r["id"]), str(r["body"])
		if id == nil || body == nil {
			continue
		}
		author := "user"
		if r["authorType"] == "agent" {
			author = "agent"
		}
		created := ""
		if s := str(r["createdAt"]); s != nil {
			created = *s
		}
		out = append(out, Comment{ID: *id, Body: *body, AuthorType: author, AuthorAgentID: str(r["authorAgentId"]), CreatedAt: created})
	}
	return out
}

func mapFailoverRuns(raw []any) []FailoverRun {
	out := []FailoverRun{}
	for _, rec := range raw {
		r := obj(rec)
		agentID := str(r["agentId"])
		if agentID == nil {
			continue
		}
		status := "unknown"
		if s := str(r["status"]); s != nil {
			status = *s
		}
		// A Codex CLI failure is filed as "Internal error"; the CLI's own words
		// live in resultJson.summary.
		out = append(out, FailoverRun{
			AgentID: *agentID, Status: status, Error: str(r["error"]),
			Model: str(obj(r["usageJson"])["model"]), FinishedAt: str(r["finishedAt"]),
			Summary: str(obj(r["resultJson"])["summary"]),
		})
	}
	return out
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	default:
		return true
	}
}
