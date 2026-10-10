package comms

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

type fakeQueue struct {
	n   int
	err error
}

func (f fakeQueue) QueuedPosts(context.Context) (int, error) { return f.n, f.err }

var (
	zernioUp   = fakeStatus{ID: "zernio", State: connectors.StateConnected, Detail: "6 platforms (@founderos.ai) · 41,200 total followers", Meta: map[string]any{"platforms": 6}}
	arcadsUp   = fakeStatus{ID: "arcads", State: connectors.StateConnected, Detail: "Vantage workspace reachable · 3 products"}
	arcadsDown = fakeStatus{ID: "arcads", State: connectors.StateError, Detail: "Creds found but API check failed: 403"}
	stackUp    = fakeStatus{ID: "local-stack", State: connectors.StateConnected, Detail: "4/4 ports up · tmux 3 sessions · pushed by MacBook 2m ago"}
	stackStale = fakeStatus{ID: "local-stack", State: connectors.StateError, Detail: "stale: last push from Mac mini 2d ago"}
)

func TestSocialMetasMatchFounderosOS(t *testing.T) {
	want := map[string][3]string{
		"social-agent":      {"Social Agent", "dept-marketing-growth", "Owns publishing and content production. Aggregates the Zernio and Arcads workers."},
		"reelkit-editor":    {"Remotion Editor", "dept-marketing-growth", "Editing and rendering pipeline for social media clips, captions, and promotional cuts."},
		"renderly-creative": {"Higgsfield Creative", "dept-marketing-growth", "Higgsfield creative generation for social assets, product shots, and campaign visuals."},
		"dmflow-mcp":        {"ManyChat MCP", "dept-marketing-growth", "ManyChat MCP/API lane for social DM automations, keyword flows, and lead capture."},
	}
	for _, a := range []agents.Agent{&SocialAgent{}, NewRemotionEditor(nil), NewHiggsfieldCreative(nil), &ManyChatAgent{}} {
		m := a.Meta()
		w, ok := want[m.ID]
		if !ok || m.Name != w[0] || m.DepartmentID != w[1] || m.Description != w[2] {
			t.Errorf("meta = %+v", m)
		}
	}
}

func TestSocialAgentLiveWithQueue(t *testing.T) {
	a := &SocialAgent{Zernio: zernioUp, Arcads: arcadsUp, Queue: fakeQueue{n: 1}}
	res, _ := a.Run(context.Background())
	if !res.OK || res.Summary != "2/2 core content APIs live · Zernio LIVE · Arcads LIVE · 1 post queued for publish" {
		t.Fatalf("res = %+v", res)
	}
	d := res.Data.(SocialResults)
	if d.QueuedPosts == nil || *d.QueuedPosts != 1 || d.Zernio.Summary != zernioUp.Detail {
		t.Fatalf("data = %+v", d)
	}
	a.Queue = fakeQueue{n: 3}
	res, _ = a.Run(context.Background())
	if !strings.HasSuffix(res.Summary, "3 posts queued for publish") {
		t.Fatalf("summary = %q", res.Summary)
	}
	a.Queue = fakeQueue{}
	res, _ = a.Run(context.Background())
	if !strings.HasSuffix(res.Summary, "no posts queued") {
		t.Fatalf("summary = %q", res.Summary)
	}
}

func TestSocialAgentUpstreamErrorAndUnknownQueue(t *testing.T) {
	a := &SocialAgent{Zernio: zernioUp, Arcads: arcadsDown, Queue: fakeQueue{err: errors.New("pg down")}}
	res, _ := a.Run(context.Background())
	if !res.OK || res.Summary != "1/2 core content APIs live · Zernio LIVE · Arcads DOWN · queued posts unknown (pg down)" {
		t.Fatalf("res = %+v", res)
	}
	if res.Data.(SocialResults).QueuedPosts != nil {
		t.Fatal("an unreadable queue is unknown, never 0")
	}
	a = &SocialAgent{
		Zernio: fakeStatus{ID: "zernio", State: connectors.StateNotConfigured, Detail: "ZERNIO_API_KEY not found"},
		Arcads: fakeStatus{ID: "arcads", State: connectors.StateNotConfigured, Detail: "ARCADS_BASIC_AUTH not found"},
		Queue:  fakeQueue{},
	}
	res, _ = a.Run(context.Background())
	if res.OK || !strings.HasPrefix(res.Summary, "0/2 core content APIs live") {
		t.Fatalf("res = %+v", res)
	}
}

func TestLocalStackLanes(t *testing.T) {
	res, _ := NewRemotionEditor(stackUp).Run(context.Background())
	if !res.OK || res.Summary != "Remotion/social editing lane mapped · local stack: "+stackUp.Detail {
		t.Fatalf("remotion = %+v", res)
	}
	res, _ = NewHiggsfieldCreative(stackStale).Run(context.Background())
	if res.OK || res.Summary != "Higgsfield creative lane mapped · local stack: "+stackStale.Detail {
		t.Fatalf("higgsfield = %+v", res)
	}
	res, _ = NewRemotionEditor(nil).Run(context.Background())
	if res.OK || !strings.Contains(res.Summary, "unknown") {
		t.Fatalf("no receiver = %+v", res)
	}
}

func TestManyChatChecksTheKeyOnlyAndNeverCallsOut(t *testing.T) {
	res, _ := (&ManyChatAgent{Key: func() string { return "mc_live_xxx" }}).Run(context.Background())
	if !res.OK || res.Summary != "ManyChat credential present · DM automation and lead capture" {
		t.Fatalf("res = %+v", res)
	}
	if strings.Contains(res.Summary, "mc_live") {
		t.Fatal("the key must never appear in a summary")
	}
	res, _ = (&ManyChatAgent{Key: func() string { return "" }}).Run(context.Background())
	if res.OK || res.Summary != "ManyChat not configured — set MANYCHAT_API_KEY · DM automation and lead capture" {
		t.Fatalf("res = %+v", res)
	}
	res, _ = (&ManyChatAgent{}).Run(context.Background())
	if res.OK {
		t.Fatal("no key lookup wired is not configured")
	}
}
