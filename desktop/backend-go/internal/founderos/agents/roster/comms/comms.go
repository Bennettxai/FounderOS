// Package comms is the bridge port of FounderOS v1's communications and
// social agents (lib/agents/real.ts): newsletter-agent
// (lib/agents/newsletter-agent.ts), comms-digest, comms-agent,
// the gmail/whatsapp/slack channel workers, social-agent and its
// postly-publisher and adsmith-creative workers, reelkit-editor,
// renderly-creative and dmflow-mcp.
//
// None of them calls an LLM and none sends, posts or DMs. The only write is
// the digest's row in founderos_comms_digests (personal workspace), which is
// the bridge's own Postgres, not an outbound side effect. The newsletter
// agent reads Beehiiv and never drafts, schedules or sends. ManyChat is never
// called: its agent checks the key only.
package comms

import (
	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/arcads"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
)

// Workspace slugs (config/founderos/engine-topology.yaml, docs/founderos/table-map.md).
const (
	personalSlug = "personal" // comms_digests, digest_reads, contact_tags, social_posts
)

// Agents returns the comms group, in FounderOS v1 roster order.
func Agents(d roster.Deps) []agents.Agent {
	mail := email.New(d.Res)
	sl := slack.New(d.Res)

	digest := &DigestAgent{
		Email:    mail,
		Slack:    sl,
		Calendar: gcal.New(d.Res),
		Wins:     payments.New(d.Res),
		Store:    NewPgDigestStore(d.Pool, d.Workspaces[personalSlug]),
	}
	chat := &CommsAgent{Inbox: mail, Slack: sl}
	var localStack connectors.Connector
	if d.Devices != nil {
		digest.WhatsApp = d.Devices.WhatsAppChats
		chat.WhatsApp = d.Devices.Connector(devicepush.SourceWhatsApp)
		localStack = d.Devices.HostConnector(devicepush.SourceLocalStack)
	}
	social := &SocialAgent{
		Zernio: zernio.New(d.Res),
		Arcads: arcads.New(d.Res),
		Queue:  NewPgSocialQueue(d.Pool, d.Workspaces[personalSlug]),
	}

	_, _, _, claudeJSON := connectors.CredFiles()
	res := d.Res
	manychatKey := func() string {
		if k := res.Resolve("MANYCHAT_API_KEY"); k != "" {
			return k
		}
		return connectors.McpEnvKey(claudeJSON, "manychat", "MANYCHAT_API_KEY")
	}

	news := beehiiv.New(d.Res)
	newsletter := &NewsletterAgent{
		Skill: func() (string, error) { return roster.LoadAgentSkill(NewsletterAgentFolder) },
		Posts: news.Posts,
		Configured: func() bool {
			return res.Resolve("BEEHIIV_API_KEY") != "" && res.Resolve("BEEHIIV_PUBLICATION_ID") != ""
		},
	}

	return []agents.Agent{
		newsletter,
		digest,
		chat,
		&GmailWorker{Inbox: chat.Inbox},
		&WhatsAppWorker{Source: chat.WhatsApp},
		&SlackWorker{Slack: chat.Slack},
		social,
		&ZernioPublisher{Zernio: social.Zernio},
		&ArcadsCreative{Arcads: social.Arcads},
		NewRemotionEditor(localStack),
		NewHiggsfieldCreative(localStack),
		&ManyChatAgent{Key: manychatKey},
	}
}
