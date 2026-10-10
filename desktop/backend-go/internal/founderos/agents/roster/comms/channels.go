package comms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
)

// UnreadCounter is the email connector's IMAP STATUS read.
type UnreadCounter interface {
	UnreadCounts(ctx context.Context) ([]email.InboxUnread, error)
}

// ChannelResults is the comms-agent's data: one worker result per channel.
type ChannelResults struct {
	Gmail    agents.Result `json:"gmail"`
	WhatsApp agents.Result `json:"whatsapp"`
	Slack    agents.Result `json:"slack"`
}

// CommsAgent is comms-agent: it runs the Gmail, WhatsApp and Slack worker
// checks that feed /comms and reports which are live.
type CommsAgent struct {
	Inbox    UnreadCounter
	WhatsApp connectors.Connector // the pushed WhatsApp source (devicepush)
	Slack    SlackFeed
}

func (a *CommsAgent) Meta() agents.Meta {
	return agents.Meta{
		ID:           "comms-agent",
		Name:         "Comms Agent",
		DepartmentID: "dept-comms",
		Description:  "Owns the unified /comms feed. Aggregates its three channel workers and reports which are live.",
	}
}

func label(r agents.Result) string {
	if r.OK {
		return "LIVE"
	}
	return "DOWN"
}

func (a *CommsAgent) Run(ctx context.Context) (agents.Result, error) {
	var d ChannelResults
	done := make(chan struct{}, 3)
	go func() { d.Gmail = gmailRun(ctx, a.Inbox); done <- struct{}{} }()
	go func() { d.WhatsApp = statusRun(ctx, a.WhatsApp, "WhatsApp"); done <- struct{}{} }()
	go func() { d.Slack = slackRun(ctx, a.Slack); done <- struct{}{} }()
	for range 3 {
		<-done
	}
	live := 0
	for _, r := range []agents.Result{d.Gmail, d.WhatsApp, d.Slack} {
		if r.OK {
			live++
		}
	}
	return agents.Result{
		OK:      live > 0,
		Summary: fmt.Sprintf("%d/3 channels live → /comms · Gmail %s · WhatsApp %s · Slack %s", live, label(d.Gmail), label(d.WhatsApp), label(d.Slack)),
		Data:    d,
	}, nil
}

// gmailRun is the gmail-worker check: unread counts per inbox.
func gmailRun(ctx context.Context, in UnreadCounter) agents.Result {
	if in == nil {
		return agents.Result{OK: false, Summary: "Gmail unknown: email connector not wired"}
	}
	counts, err := in.UnreadCounts(ctx)
	if errors.Is(err, email.ErrNotConfigured) {
		return agents.Result{OK: false, Summary: "No inboxes configured — set INBOX_1..4_HOST/_USER/_PASS in .env.local"}
	}
	if err != nil {
		return agents.Result{OK: false, Summary: "Gmail error: " + err.Error()}
	}
	parts := make([]string, 0, len(counts)+1)
	failed, total := 0, 0
	for _, c := range counts {
		if c.Error != "" || c.Unread == nil {
			failed++
			msg := c.Error
			if len(msg) > 60 {
				msg = msg[:60]
			}
			parts = append(parts, fmt.Sprintf("%s: ERROR %s", c.Inbox, msg))
			continue
		}
		total += *c.Unread
		parts = append(parts, fmt.Sprintf("%s: %d unread", c.Inbox, *c.Unread))
	}
	return agents.Result{
		OK:      failed < len(counts),
		Summary: strings.Join(parts, " · ") + fmt.Sprintf(" · total %d unread", total),
		Data:    counts,
	}
}

// statusRun turns a connector status into a worker result (whatsappRun,
// zernioRun, arcadsRun).
func statusRun(ctx context.Context, c connectors.Connector, name string) agents.Result {
	if c == nil {
		return agents.Result{OK: false, Summary: name + " unknown: no source wired on the bridge"}
	}
	s := c.Status(ctx)
	r := agents.Result{OK: s.State == connectors.StateConnected, Summary: s.Detail}
	if s.Meta != nil {
		r.Data = s.Meta
	}
	return r
}

// slackRun is the slack-worker check: the latest 10 messages.
func slackRun(ctx context.Context, s SlackFeed) agents.Result {
	if s == nil {
		return agents.Result{OK: false, Summary: "Slack unknown: slack connector not wired"}
	}
	msgs, err := s.RecentMessages(ctx, 10)
	if errors.Is(err, slack.ErrNotConfigured) {
		return agents.Result{OK: false, Summary: "Slack not configured — set SLACK_BOT_TOKEN in .env.local"}
	}
	if err != nil {
		return agents.Result{OK: false, Summary: "Slack error: " + err.Error()}
	}
	channels := map[string]bool{}
	for _, m := range msgs {
		channels[m.Channel] = true
	}
	return agents.Result{
		OK:      true,
		Summary: fmt.Sprintf("%d recent messages across %d channels", len(msgs), len(channels)),
		Data:    msgs,
	}
}
