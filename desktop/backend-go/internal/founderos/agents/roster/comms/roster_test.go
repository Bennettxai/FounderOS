package comms

import (
	"slices"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
)

// The group registers exactly the agents it owns, with FounderOS v1 ids, in
// FounderOS v1 registry order.
// Wiring only: no agent is run here, because the real connectors would reach
// out to live systems.
func TestAgentsRegistersTheCommsGroup(t *testing.T) {
	list := Agents(roster.Deps{Workspaces: map[string]string{"personal": "p", "founderos": "f"}})
	var ids []string
	for _, a := range list {
		ids = append(ids, a.Meta().ID)
	}
	want := []string{"newsletter-agent", "comms-digest", "comms-agent", "gmail-worker", "whatsapp-worker", "slack-worker",
		"social-agent", "postly-publisher", "adsmith-creative", "reelkit-editor", "renderly-creative", "dmflow-mcp"}
	if !slices.Equal(ids, want) {
		t.Fatalf("ids = %v", ids)
	}
	agents.New(agents.NewMemStore(), list...) // panics on a duplicate id

	d := list[1].(*DigestAgent)
	if d.Email == nil || d.Slack == nil || d.Calendar == nil || d.Wins == nil || d.Store == nil {
		t.Fatalf("digest wiring = %+v", d)
	}
	if d.WhatsApp != nil {
		t.Fatal("no device receiver in Deps: WhatsApp must stay unwired (reported unavailable), not faked")
	}
	if st := d.Store.(*PgDigestStore); st.workspaceID != "p" {
		t.Fatalf("digest store workspace = %q, want the personal workspace", st.workspaceID)
	}
	byID := map[string]agents.Agent{}
	for _, a := range list {
		byID[a.Meta().ID] = a
	}
	if q := byID["social-agent"].(*SocialAgent).Queue.(*PgSocialQueue); q.workspaceID != "p" {
		t.Fatalf("social queue workspace = %q, want the personal workspace", q.workspaceID)
	}
	if byID["dmflow-mcp"].(*ManyChatAgent).Key == nil {
		t.Fatal("manychat key lookup not wired")
	}
	news := byID["newsletter-agent"].(*NewsletterAgent)
	if news.Posts == nil || news.Configured == nil || news.Skill == nil {
		t.Fatalf("newsletter wiring = %+v", news)
	}
	if skill, err := news.Skill(); err != nil || skill == "" {
		t.Fatalf("the newsletter skill must load from the repo copy: %v", err)
	}
	// The workers share their lead's connectors, so a worker and the lead
	// that aggregates it read the same source.
	chat := byID["comms-agent"].(*CommsAgent)
	if byID["gmail-worker"].(*GmailWorker).Inbox != chat.Inbox || byID["slack-worker"].(*SlackWorker).Slack != chat.Slack {
		t.Fatal("gmail/slack workers must share the comms-agent connectors")
	}
	if byID["whatsapp-worker"].(*WhatsAppWorker).Source != nil {
		t.Fatal("no device receiver in Deps: the WhatsApp worker must stay unwired, not faked")
	}
	social := byID["social-agent"].(*SocialAgent)
	if byID["postly-publisher"].(*ZernioPublisher).Zernio != social.Zernio || byID["adsmith-creative"].(*ArcadsCreative).Arcads != social.Arcads {
		t.Fatal("zernio/arcads workers must share the social-agent connectors")
	}
}
