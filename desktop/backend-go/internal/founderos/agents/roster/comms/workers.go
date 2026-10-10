package comms

import (
	"context"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// The five single-channel workers of FounderOS v1 lib/agents/real.ts
// (gmail-worker, whatsapp-worker, slack-worker, postly-publisher,
// adsmith-creative). Each is its run function on its own: the same check the
// comms-agent and social-agent aggregate, so a worker and its lead can never
// disagree about a channel. All are reads; none sends or publishes.
//
// Metas are the seeded founderos_agents rows (lib/seed.ts), which is what the
// rest of the bridge roster carries.

var (
	metaGmailWorker = agents.Meta{ID: "gmail-worker", Name: "Gmail Worker", DepartmentID: "dept-comms",
		Description: "Pulls unread counts and recent mail from up to four IMAP inboxes into /comms. Activates when INBOX_* creds land."}
	metaWhatsAppWorker = agents.Meta{ID: "whatsapp-worker", Name: "WhatsApp Worker", DepartmentID: "dept-comms",
		Description: "Reads the local WhatsApp ChatStorage (600+ chats incl. LC + Vantage teams) into /comms. Works today."}
	metaSlackWorker = agents.Meta{ID: "slack-worker", Name: "Slack Worker", DepartmentID: "dept-comms",
		Description: "Latest messages across joined channels into /comms. Needs SLACK_BOT_TOKEN."}
	metaZernioPublisher = agents.Meta{ID: "postly-publisher", Name: "Zernio Publisher", DepartmentID: "dept-marketing-growth",
		Description: "Publishes and monitors six platforms under @founderos.ai via Zernio. Key already on this machine — works today."}
	metaArcadsCreative = agents.Meta{ID: "adsmith-creative", Name: "Arcads Creative", DepartmentID: "dept-marketing-growth",
		Description: "Generates UGC ads for Vantage (Veo/Sora/Kling) via the Arcads API. Auth on this machine — works today."}
)

// GmailWorker is gmail-worker: unread counts per IMAP inbox (gmailRun).
type GmailWorker struct{ Inbox UnreadCounter }

func (w *GmailWorker) Meta() agents.Meta { return metaGmailWorker }
func (w *GmailWorker) Run(ctx context.Context) (agents.Result, error) {
	return gmailRun(ctx, w.Inbox), nil
}

// WhatsAppWorker is whatsapp-worker (whatsappRun). FounderOS v1 read the
// local ChatStorage.sqlite; the bridge has no such file, so it reads the
// WhatsApp source a Mac's collector pushes (devicepush), exactly as
// comms-agent does. Nil Source means no device receiver: unknown, not ok.
type WhatsAppWorker struct{ Source connectors.Connector }

func (w *WhatsAppWorker) Meta() agents.Meta { return metaWhatsAppWorker }
func (w *WhatsAppWorker) Run(ctx context.Context) (agents.Result, error) {
	return statusRun(ctx, w.Source, "WhatsApp"), nil
}

// SlackWorker is slack-worker: the latest 10 messages (slackRun).
type SlackWorker struct{ Slack SlackFeed }

func (w *SlackWorker) Meta() agents.Meta { return metaSlackWorker }
func (w *SlackWorker) Run(ctx context.Context) (agents.Result, error) {
	return slackRun(ctx, w.Slack), nil
}

// ZernioPublisher is postly-publisher (zernioRun): the Zernio status. It
// checks the account; it never posts.
type ZernioPublisher struct{ Zernio connectors.Connector }

func (w *ZernioPublisher) Meta() agents.Meta { return metaZernioPublisher }
func (w *ZernioPublisher) Run(ctx context.Context) (agents.Result, error) {
	return statusRun(ctx, w.Zernio, "Zernio"), nil
}

// ArcadsCreative is adsmith-creative (arcadsRun): the Arcads status. It
// never starts a generation.
type ArcadsCreative struct{ Arcads connectors.Connector }

func (w *ArcadsCreative) Meta() agents.Meta { return metaArcadsCreative }
func (w *ArcadsCreative) Run(ctx context.Context) (agents.Result, error) {
	return statusRun(ctx, w.Arcads, "Arcads"), nil
}
