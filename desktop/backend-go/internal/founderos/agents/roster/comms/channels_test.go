package comms

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
)

type fakeUnread struct {
	counts []email.InboxUnread
	err    error
}

func (f fakeUnread) UnreadCounts(context.Context) ([]email.InboxUnread, error) {
	return f.counts, f.err
}

type fakeStatus connectors.Status

func (f fakeStatus) Status(context.Context) connectors.Status { return connectors.Status(f) }

func ip(n int) *int { return &n }

func liveCommsAgent() *CommsAgent {
	return &CommsAgent{
		Inbox: fakeUnread{counts: []email.InboxUnread{{Inbox: "Launchpad Cohort", Unread: ip(4)}, {Inbox: "Personal", Unread: ip(9)}}},
		WhatsApp: fakeStatus{ID: "whatsapp", State: connectors.StateConnected, Detail: "612 chats · pushed by MacBook 3m ago",
			Meta: map[string]any{"chats": 612}},
		Slack: fakeSlack{msgs: []slack.Message{{Channel: "#a"}, {Channel: "#b"}, {Channel: "#a"}}},
	}
}

func TestCommsAgentMetaMatchesFounderosOS(t *testing.T) {
	m := (&CommsAgent{}).Meta()
	if m.ID != "comms-agent" || m.Name != "Comms Agent" || m.DepartmentID != "dept-comms" ||
		m.Description != "Owns the unified /comms feed. Aggregates its three channel workers and reports which are live." {
		t.Fatalf("meta = %+v", m)
	}
}

func TestCommsAgentAllLive(t *testing.T) {
	res, err := liveCommsAgent().Run(context.Background())
	if err != nil || !res.OK {
		t.Fatalf("res %+v err %v", res, err)
	}
	if res.Summary != "3/3 channels live → /comms · Gmail LIVE · WhatsApp LIVE · Slack LIVE" {
		t.Fatalf("summary = %q", res.Summary)
	}
	d := res.Data.(ChannelResults)
	if d.Gmail.Summary != "Launchpad Cohort: 4 unread · Personal: 9 unread · total 13 unread" {
		t.Fatalf("gmail = %q", d.Gmail.Summary)
	}
	if d.Slack.Summary != "3 recent messages across 2 channels" {
		t.Fatalf("slack = %q", d.Slack.Summary)
	}
	if d.WhatsApp.Summary != "612 chats · pushed by MacBook 3m ago" {
		t.Fatalf("whatsapp = %q", d.WhatsApp.Summary)
	}
}

func TestCommsAgentGmailPartialAndTotalFailure(t *testing.T) {
	a := liveCommsAgent()
	long := strings.Repeat("x", 80)
	a.Inbox = fakeUnread{counts: []email.InboxUnread{{Inbox: "A", Unread: ip(2)}, {Inbox: "B", Error: long}}}
	res, _ := a.Run(context.Background())
	g := res.Data.(ChannelResults).Gmail
	if !g.OK || g.Summary != "A: 2 unread · B: ERROR "+long[:60]+" · total 2 unread" {
		t.Fatalf("partial gmail = %+v", g)
	}
	a.Inbox = fakeUnread{counts: []email.InboxUnread{{Inbox: "A", Error: "auth"}}}
	res, _ = a.Run(context.Background())
	if res.Data.(ChannelResults).Gmail.OK || !strings.Contains(res.Summary, "Gmail DOWN") {
		t.Fatalf("all-failed gmail must be DOWN: %s", res.Summary)
	}
}

func TestCommsAgentMissingCredsAreHonest(t *testing.T) {
	a := &CommsAgent{
		Inbox:    fakeUnread{err: email.ErrNotConfigured},
		WhatsApp: fakeStatus{ID: "whatsapp", State: connectors.StateNotConfigured, Detail: "No collector has pushed WhatsApp yet."},
		Slack:    fakeSlack{err: slack.ErrNotConfigured},
	}
	res, _ := a.Run(context.Background())
	if res.OK || res.Summary != "0/3 channels live → /comms · Gmail DOWN · WhatsApp DOWN · Slack DOWN" {
		t.Fatalf("res = %+v", res)
	}
	d := res.Data.(ChannelResults)
	if d.Gmail.Summary != "No inboxes configured — set INBOX_1..4_HOST/_USER/_PASS in .env.local" {
		t.Fatalf("gmail = %q", d.Gmail.Summary)
	}
	if d.Slack.Summary != "Slack not configured — set SLACK_BOT_TOKEN in .env.local" {
		t.Fatalf("slack = %q", d.Slack.Summary)
	}
}

func TestCommsAgentUpstreamErrorsAreDownNotEmpty(t *testing.T) {
	a := liveCommsAgent()
	a.Slack = fakeSlack{err: errors.New("slack conversations.list: ratelimited")}
	a.Inbox = fakeUnread{err: errors.New("dial tcp: timeout")}
	a.WhatsApp = nil
	res, _ := a.Run(context.Background())
	d := res.Data.(ChannelResults)
	if d.Slack.OK || !strings.Contains(d.Slack.Summary, "ratelimited") {
		t.Fatalf("slack = %+v", d.Slack)
	}
	if d.Gmail.OK || !strings.Contains(d.Gmail.Summary, "timeout") {
		t.Fatalf("gmail = %+v", d.Gmail)
	}
	if d.WhatsApp.OK || d.WhatsApp.Summary == "" {
		t.Fatalf("whatsapp = %+v", d.WhatsApp)
	}
	if res.OK {
		t.Fatal("no channel live means not ok")
	}
}
