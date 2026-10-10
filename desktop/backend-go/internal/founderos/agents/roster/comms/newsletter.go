package comms

import (
	"context"
	"errors"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
)

// NewsletterAgentFolder is the skill folder (agents/newsletter/skill.md).
const NewsletterAgentFolder = "newsletter"

var metaNewsletterAgent = agents.Meta{ID: "newsletter-agent", Name: "Newsletter Agent", DepartmentID: "dept-marketing-growth",
	Description: "Reads Beehiiv send performance, builds a brief that is honest about how thin the history is, and drafts the next issue against the skill file. Drafts only, never schedules or sends."}

// NewsletterData is the run's data: the brief and the skill file's open
// questions, as in FounderOS v1.
type NewsletterData struct {
	Brief                 NewsletterBrief `json:"brief"`
	AwaitingFromFounderos []string        `json:"awaitingFromFounderos"`
}

// NewsletterAgent is newsletter-agent (lib/agents/newsletter-agent.ts): it
// reads what the list opened and clicked out of Beehiiv and hands back the
// brief. FounderOS v1 stopped drafting when its AI Gateway was removed
// (2026-09-21), so neither side calls a model; nothing is scheduled or sent.
type NewsletterAgent struct {
	// Skill loads agents/newsletter/skill.md (roster.LoadAgentSkill).
	Skill func() (string, error)
	// Posts is beehiiv.Connector.Posts: nil when unconfigured, failing or
	// empty, exactly like beehiivPosts' null.
	Posts func(ctx context.Context) []beehiiv.Newsletter
	// Configured reports whether both BEEHIIV_API_KEY and
	// BEEHIIV_PUBLICATION_ID resolve, so a nil read is not blamed on keys
	// that are present.
	Configured func() bool
}

func (a *NewsletterAgent) Meta() agents.Meta { return metaNewsletterAgent }

func (a *NewsletterAgent) Run(ctx context.Context) (agents.Result, error) {
	if a.Skill == nil {
		return agents.Result{}, errors.New("newsletter skill loader not wired")
	}
	skill, err := a.Skill()
	if err != nil {
		return agents.Result{}, err
	}
	var posts []beehiiv.Newsletter
	if a.Posts != nil {
		posts = a.Posts(ctx)
	}
	if posts == nil {
		summary := "Beehiiv is not connected (BEEHIIV_API_KEY / BEEHIIV_PUBLICATION_ID missing), so there is no send history to write against."
		if a.Posts != nil && a.Configured != nil && a.Configured() {
			summary = "Beehiiv send history unavailable (the posts read failed or returned no sent issues), so there is no send history to write against."
		}
		return agents.Result{OK: false, Summary: summary}, nil
	}
	brief := BuildNewsletterBrief(posts)
	return agents.Result{
		OK:      true,
		Summary: BriefSummary(brief),
		Data:    NewsletterData{Brief: brief, AwaitingFromFounderos: roster.OpenSkillQuestions(skill)},
	}, nil
}
