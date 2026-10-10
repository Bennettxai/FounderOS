package comms

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
)

// The five channel/content workers carry the seeded founderos_agents row's
// name, department and description (FounderOS v1 lib/seed.ts), so /agents and
// /org place them under their parent's pillar.
func TestWorkerMetasMatchTheSeededRows(t *testing.T) {
	want := []agents.Meta{
		{ID: "gmail-worker", Name: "Gmail Worker", DepartmentID: "dept-comms",
			Description: "Pulls unread counts and recent mail from up to four IMAP inboxes into /comms. Activates when INBOX_* creds land."},
		{ID: "whatsapp-worker", Name: "WhatsApp Worker", DepartmentID: "dept-comms",
			Description: "Reads the local WhatsApp ChatStorage (600+ chats incl. LC + Vantage teams) into /comms. Works today."},
		{ID: "slack-worker", Name: "Slack Worker", DepartmentID: "dept-comms",
			Description: "Latest messages across joined channels into /comms. Needs SLACK_BOT_TOKEN."},
		{ID: "postly-publisher", Name: "Zernio Publisher", DepartmentID: "dept-marketing-growth",
			Description: "Publishes and monitors six platforms under @founderos.ai via Zernio. Key already on this machine — works today."},
		{ID: "adsmith-creative", Name: "Arcads Creative", DepartmentID: "dept-marketing-growth",
			Description: "Generates UGC ads for Vantage (Veo/Sora/Kling) via the Arcads API. Auth on this machine — works today."},
	}
	got := []agents.Agent{&GmailWorker{}, &WhatsAppWorker{}, &SlackWorker{}, &ZernioPublisher{}, &ArcadsCreative{}}
	for i, a := range got {
		if a.Meta() != want[i] {
			t.Errorf("meta %d:\n got %+v\nwant %+v", i, a.Meta(), want[i])
		}
	}
}

func runOK(t *testing.T, a agents.Agent) agents.Result {
	t.Helper()
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatalf("%s: unexpected error %v", a.Meta().ID, err)
	}
	return res
}

// gmailRun: per-inbox unread counts; ok while at least one inbox answers.
func TestGmailWorker(t *testing.T) {
	res := runOK(t, &GmailWorker{Inbox: fakeUnread{counts: []email.InboxUnread{{Inbox: "Launchpad Cohort", Unread: ip(4)}, {Inbox: "Personal", Error: "auth failed"}}}})
	if !res.OK || res.Summary != "Launchpad Cohort: 4 unread · Personal: ERROR auth failed · total 4 unread" {
		t.Fatalf("res = %+v", res)
	}
	if n := len(res.Data.([]email.InboxUnread)); n != 2 {
		t.Fatalf("data rows = %d", n)
	}
	res = runOK(t, &GmailWorker{Inbox: fakeUnread{counts: []email.InboxUnread{{Inbox: "A", Error: "auth"}}}})
	if res.OK {
		t.Fatalf("every inbox failing is not ok: %+v", res)
	}
	res = runOK(t, &GmailWorker{Inbox: fakeUnread{err: email.ErrNotConfigured}})
	if res.OK || res.Summary != "No inboxes configured — set INBOX_1..4_HOST/_USER/_PASS in .env.local" {
		t.Fatalf("unconfigured = %+v", res)
	}
	if res := runOK(t, &GmailWorker{}); res.OK {
		t.Fatalf("unwired = %+v", res)
	}
}

// whatsappRun reads the collector's push on the bridge (FounderOS v1 read the
// local ChatStorage): connected is ok, anything else is not.
func TestWhatsAppWorker(t *testing.T) {
	up := fakeStatus{ID: "whatsapp", State: connectors.StateConnected, Detail: "612 chats · pushed by MacBook 3m ago", Meta: map[string]any{"chats": 612}}
	res := runOK(t, &WhatsAppWorker{Source: up})
	if !res.OK || res.Summary != up.Detail || res.Data.(map[string]any)["chats"] != 612 {
		t.Fatalf("res = %+v", res)
	}
	res = runOK(t, &WhatsAppWorker{Source: fakeStatus{ID: "whatsapp", State: connectors.StateError, Detail: "stale: last push from MacBook 2d ago"}})
	if res.OK || res.Summary != "stale: last push from MacBook 2d ago" {
		t.Fatalf("stale = %+v", res)
	}
	res = runOK(t, &WhatsAppWorker{})
	if res.OK || !strings.Contains(res.Summary, "WhatsApp unknown") {
		t.Fatalf("no device receiver = %+v", res)
	}
}

func TestSlackWorker(t *testing.T) {
	res := runOK(t, &SlackWorker{Slack: fakeSlack{msgs: []slack.Message{{Channel: "#a"}, {Channel: "#b"}, {Channel: "#a"}}}})
	if !res.OK || res.Summary != "3 recent messages across 2 channels" {
		t.Fatalf("res = %+v", res)
	}
	res = runOK(t, &SlackWorker{Slack: fakeSlack{err: slack.ErrNotConfigured}})
	if res.OK || res.Summary != "Slack not configured — set SLACK_BOT_TOKEN in .env.local" {
		t.Fatalf("unconfigured = %+v", res)
	}
	res = runOK(t, &SlackWorker{Slack: fakeSlack{err: errors.New("ratelimited")}})
	if res.OK || !strings.Contains(res.Summary, "ratelimited") {
		t.Fatalf("upstream error = %+v", res)
	}
}

// zernioRun / arcadsRun: the connector status, ok exactly when connected.
func TestZernioPublisherAndArcadsCreative(t *testing.T) {
	res := runOK(t, &ZernioPublisher{Zernio: zernioUp})
	if !res.OK || res.Summary != zernioUp.Detail || res.Data.(map[string]any)["platforms"] != 6 {
		t.Fatalf("zernio = %+v", res)
	}
	res = runOK(t, &ZernioPublisher{Zernio: fakeStatus{ID: "zernio", State: connectors.StateNotConfigured, Detail: "ZERNIO_API_KEY not found"}})
	if res.OK || res.Summary != "ZERNIO_API_KEY not found" {
		t.Fatalf("zernio unconfigured = %+v", res)
	}
	res = runOK(t, &ArcadsCreative{Arcads: arcadsUp})
	if !res.OK || res.Summary != arcadsUp.Detail {
		t.Fatalf("arcads = %+v", res)
	}
	res = runOK(t, &ArcadsCreative{Arcads: arcadsDown})
	if res.OK || res.Summary != arcadsDown.Detail {
		t.Fatalf("arcads down = %+v", res)
	}
}
