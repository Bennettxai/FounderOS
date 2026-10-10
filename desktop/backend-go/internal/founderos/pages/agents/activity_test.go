package agentspage

import (
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func TestRecentActivityUnionsNewestFirst(t *testing.T) {
	runs := []osdata.AgentRun{{AgentID: "a", StartedAt: "2026-09-30T10:00:00.000Z", Summary: "ran", OK: true}}
	msgs := []osdata.AgentMessage{
		{AgentID: "b", Role: "user", Content: "hi", CreatedAt: "2026-09-30T12:00:00.000Z"},
		{AgentID: "b", Role: "assistant", Content: strings.Repeat("x", 300), CreatedAt: "2026-09-30T11:00:00.000Z"},
		{AgentID: "b", Role: "tool", ToolCalls: []osdata.ToolCallBrief{{Name: "gmail"}, {Name: "slack"}}, CreatedAt: "2026-09-30T09:00:00.000Z"},
	}
	bcs := []osdata.Broadcast{{Replies: []osdata.BroadcastReply{{AgentID: "c", Reply: "yo", OK: false, FinishedAt: "2026-09-30T13:00:00.000Z"}}}}
	ev := RecentActivity(runs, msgs, bcs, 10)
	if len(ev) != 4 {
		t.Fatalf("user turns are not activity: %+v", ev)
	}
	kinds := []string{ev[0].Kind, ev[1].Kind, ev[2].Kind, ev[3].Kind}
	if strings.Join(kinds, ",") != "broadcast,message,run,message" {
		t.Fatalf("order %v", kinds)
	}
	if len(ev[1].Summary) != 200 || ev[3].Summary != "tool · gmail, slack" || ev[0].OK == nil || *ev[0].OK || ev[1].OK != nil {
		t.Fatalf("shape %+v", ev)
	}
	if got := RecentActivity(runs, msgs, bcs, 2); len(got) != 2 {
		t.Fatalf("limit %d", len(got))
	}
}
