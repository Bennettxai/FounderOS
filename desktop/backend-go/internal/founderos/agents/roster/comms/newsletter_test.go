package comms

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
)

const newsletterSkill = "# Newsletter\n- [ ] **Cadence.** How often.\n- [x] **Done.**"

func skillOK() (string, error) { return newsletterSkill, nil }

func postsOf(list []beehiiv.Newsletter, calls *int) func(context.Context) []beehiiv.Newsletter {
	return func(context.Context) []beehiiv.Newsletter {
		if calls != nil {
			*calls++
		}
		return list
	}
}

func TestNewsletterAgentMetaIsTheSeededRow(t *testing.T) {
	want := agents.Meta{ID: "newsletter-agent", Name: "Newsletter Agent", DepartmentID: "dept-marketing-growth",
		Description: "Reads Beehiiv send performance, builds a brief that is honest about how thin the history is, and drafts the next issue against the skill file. Drafts only, never schedules or sends."}
	if m := (&NewsletterAgent{}).Meta(); m != want {
		t.Fatalf("meta = %+v", m)
	}
}

func TestNewsletterAgentBriefsFromSendHistory(t *testing.T) {
	list := []beehiiv.Newsletter{send(func(n *beehiiv.Newsletter) { n.ID = "a" }), send(func(n *beehiiv.Newsletter) { n.ID = "b" })}
	a := &NewsletterAgent{Skill: skillOK, Posts: postsOf(list, nil), Configured: func() bool { return true }}
	res, err := a.Run(context.Background())
	if err != nil || !res.OK || res.Summary != "2 sends · median 40% open / 5% click · only 2 sends, treat as anecdote" {
		t.Fatalf("res %+v err %v", res, err)
	}
	d := res.Data.(NewsletterData)
	if d.Brief.Sends != 2 || len(d.AwaitingFromFounderos) != 1 || d.AwaitingFromFounderos[0] != "Cadence." {
		t.Fatalf("data = %+v", d)
	}
}

// FounderOS v1 returns null for unconfigured, failed and empty reads alike,
// and the agent fails the run. The bridge keeps ok=false for all three and
// keeps the TS sentence when the keys are missing, but does not blame
// missing keys when they are present.
func TestNewsletterAgentWithoutSendHistoryFails(t *testing.T) {
	calls := 0
	a := &NewsletterAgent{Skill: skillOK, Posts: postsOf(nil, &calls), Configured: func() bool { return false }}
	res, err := a.Run(context.Background())
	if err != nil || res.OK || res.Summary != "Beehiiv is not connected (BEEHIIV_API_KEY / BEEHIIV_PUBLICATION_ID missing), so there is no send history to write against." || res.Data != nil {
		t.Fatalf("unconfigured: res %+v err %v", res, err)
	}
	a.Configured = func() bool { return true }
	res, _ = a.Run(context.Background())
	if res.OK || res.Summary != "Beehiiv send history unavailable (the posts read failed or returned no sent issues), so there is no send history to write against." {
		t.Fatalf("keyed but empty: %+v", res)
	}
	if res, _ := (&NewsletterAgent{Skill: skillOK}).Run(context.Background()); res.OK {
		t.Fatalf("unwired = %+v", res)
	}
}

// The skill file is read first and a missing one fails the run, before
// Beehiiv is asked anything.
func TestNewsletterAgentNeedsItsSkillFile(t *testing.T) {
	calls := 0
	a := &NewsletterAgent{
		Skill: func() (string, error) {
			return "", errors.New("missing or empty skill file: agents/newsletter/skill.md")
		},
		Posts: postsOf([]beehiiv.Newsletter{send(nil)}, &calls), Configured: func() bool { return true },
	}
	res, err := a.Run(context.Background())
	if err == nil || err.Error() != "missing or empty skill file: agents/newsletter/skill.md" || res.OK || calls != 0 {
		t.Fatalf("res %+v err %v calls %d", res, err, calls)
	}
}
