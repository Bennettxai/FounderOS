package comms

import (
	"context"
	"fmt"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// QueueCounter counts posts waiting to publish (socialPosts.queued()).
type QueueCounter interface {
	QueuedPosts(ctx context.Context) (int, error)
}

// SocialResults is the social-agent's data. QueuedPosts is nil when the
// queue could not be read: unknown, never 0.
type SocialResults struct {
	Zernio      agents.Result `json:"zernio"`
	Arcads      agents.Result `json:"arcads"`
	QueuedPosts *int          `json:"queuedPosts"`
}

// SocialAgent is social-agent: the Zernio and Arcads worker checks plus the
// publish queue. Read-only: it never publishes.
type SocialAgent struct {
	Zernio connectors.Connector
	Arcads connectors.Connector
	Queue  QueueCounter
}

func (a *SocialAgent) Meta() agents.Meta {
	return agents.Meta{
		ID:           "social-agent",
		Name:         "Social Agent",
		DepartmentID: "dept-marketing-growth",
		Description:  "Owns publishing and content production. Aggregates the Zernio and Arcads workers.",
	}
}

func (a *SocialAgent) Run(ctx context.Context) (agents.Result, error) {
	var d SocialResults
	done := make(chan struct{}, 2)
	go func() { d.Zernio = statusRun(ctx, a.Zernio, "Zernio"); done <- struct{}{} }()
	go func() { d.Arcads = statusRun(ctx, a.Arcads, "Arcads"); done <- struct{}{} }()
	<-done
	<-done

	var queueNote string
	switch n, err := a.queued(ctx); {
	case err != nil:
		queueNote = "queued posts unknown (" + err.Error() + ")"
	case n == 0:
		d.QueuedPosts = &n
		queueNote = "no posts queued"
	default:
		d.QueuedPosts = &n
		plural := "s"
		if n == 1 {
			plural = ""
		}
		queueNote = fmt.Sprintf("%d post%s queued for publish", n, plural)
	}
	live := 0
	for _, r := range []agents.Result{d.Zernio, d.Arcads} {
		if r.OK {
			live++
		}
	}
	return agents.Result{
		OK:      live > 0,
		Summary: fmt.Sprintf("%d/2 core content APIs live · Zernio %s · Arcads %s · %s", live, label(d.Zernio), label(d.Arcads), queueNote),
		Data:    d,
	}, nil
}

func (a *SocialAgent) queued(ctx context.Context) (int, error) {
	if a.Queue == nil {
		return 0, errNoWorkspace
	}
	return a.Queue.QueuedPosts(ctx)
}

// LaneAgent is a creative lane whose health is the local stack it renders on
// (reelkit-editor, renderly-creative). The stack is pushed by the device
// collector; the backend never probes a Mac directly.
type LaneAgent struct {
	meta       agents.Meta
	lane       string
	LocalStack connectors.Connector
}

func NewRemotionEditor(localStack connectors.Connector) *LaneAgent {
	return &LaneAgent{
		meta: agents.Meta{
			ID: "reelkit-editor", Name: "Remotion Editor", DepartmentID: "dept-marketing-growth",
			Description: "Editing and rendering pipeline for social media clips, captions, and promotional cuts.",
		},
		lane:       "Remotion/social editing lane mapped",
		LocalStack: localStack,
	}
}

func NewHiggsfieldCreative(localStack connectors.Connector) *LaneAgent {
	return &LaneAgent{
		meta: agents.Meta{
			ID: "renderly-creative", Name: "Higgsfield Creative", DepartmentID: "dept-marketing-growth",
			Description: "Higgsfield creative generation for social assets, product shots, and campaign visuals.",
		},
		lane:       "Higgsfield creative lane mapped",
		LocalStack: localStack,
	}
}

func (a *LaneAgent) Meta() agents.Meta { return a.meta }

func (a *LaneAgent) Run(ctx context.Context) (agents.Result, error) {
	if a.LocalStack == nil {
		return agents.Result{OK: false, Summary: a.lane + " · local stack: unknown (no device receiver on the bridge)"}, nil
	}
	s := a.LocalStack.Status(ctx)
	r := agents.Result{OK: s.State == connectors.StateConnected, Summary: a.lane + " · local stack: " + s.Detail}
	if s.Meta != nil {
		r.Data = s.Meta
	}
	return r, nil
}

// ManyChatAgent is dmflow-mcp. As in FounderOS v1 it only checks that the key
// is present: ManyChat blocks every request for 24h once its cap is hit, so
// this agent makes no ManyChat call at all.
type ManyChatAgent struct {
	Key func() string
}

func (a *ManyChatAgent) Meta() agents.Meta {
	return agents.Meta{
		ID:           "dmflow-mcp",
		Name:         "ManyChat MCP",
		DepartmentID: "dept-marketing-growth",
		Description:  "ManyChat MCP/API lane for social DM automations, keyword flows, and lead capture.",
	}
}

func (a *ManyChatAgent) Run(context.Context) (agents.Result, error) {
	const purpose = "DM automation and lead capture"
	if a.Key == nil || a.Key() == "" {
		return agents.Result{OK: false, Summary: "ManyChat not configured — set MANYCHAT_API_KEY · " + purpose}, nil
	}
	return agents.Result{OK: true, Summary: "ManyChat credential present · " + purpose}, nil
}
