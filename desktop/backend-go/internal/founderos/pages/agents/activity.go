package agentspage

import (
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// ActivityEvent is one line of the agent activity feed (lib/agents/activity).
type ActivityEvent struct {
	Kind    string `json:"kind"` // run | message | broadcast
	AgentID string `json:"agentId"`
	At      string `json:"at"`
	Summary string `json:"summary"`
	OK      *bool  `json:"ok,omitempty"`
}

func clip200(s string) string {
	r := []rune(s)
	if len(r) > 200 {
		return string(r[:200])
	}
	return s
}

// RecentActivity unions runs, the agents' own chat turns and broadcast
// replies into one newest-first stream. The operator's own turns are not activity.
func RecentActivity(runs []osdata.AgentRun, msgs []osdata.AgentMessage, broadcasts []osdata.Broadcast, limit int) []ActivityEvent {
	ev := []ActivityEvent{}
	for _, r := range runs {
		ok := r.OK
		ev = append(ev, ActivityEvent{Kind: "run", AgentID: r.AgentID, At: r.StartedAt, Summary: r.Summary, OK: &ok})
	}
	for _, m := range msgs {
		if m.Role == "user" {
			continue
		}
		summary := m.Content
		if m.Role == "tool" {
			names := make([]string, len(m.ToolCalls))
			for i, c := range m.ToolCalls {
				names[i] = c.Name
			}
			summary = "tool · " + strings.Join(names, ", ")
		}
		ev = append(ev, ActivityEvent{Kind: "message", AgentID: m.AgentID, At: m.CreatedAt, Summary: clip200(summary)})
	}
	for _, b := range broadcasts {
		for _, r := range b.Replies {
			ok := r.OK
			ev = append(ev, ActivityEvent{Kind: "broadcast", AgentID: r.AgentID, At: r.FinishedAt, Summary: clip200(r.Reply), OK: &ok})
		}
	}
	sort.SliceStable(ev, func(i, j int) bool { return ev[i].At > ev[j].At })
	if limit > 0 && len(ev) > limit {
		ev = ev[:limit]
	}
	return ev
}
