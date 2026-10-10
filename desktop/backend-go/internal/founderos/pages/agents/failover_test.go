package agentspage

import (
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/agent-failover.test.ts.

const fableOut = "Internal error: You're out of usage credits. Run /usage-credits to keep using Fable 5 or /model to switch models."
const codexQuota = "You've hit your usage limit. Upgrade to Pro (https://openai.com/chatgpt/pricing) or try again at 4:36 AM."
const oauthExpired = "Failed to authenticate: OAuth session expired and could not be refreshed"

func fa(mod func(*FailoverAgent)) FailoverAgent {
	a := FailoverAgent{ID: "a-conductor", Name: "Conductor", Model: ptr("claude-fable-5"), Status: "error"}
	if mod != nil {
		mod(&a)
	}
	return a
}

func fr(mod func(*FailoverRunIn)) FailoverRunIn {
	r := FailoverRunIn{AgentID: "a-conductor", Status: "failed", Error: ptr(fableOut), Model: ptr("claude-fable-5"), FinishedAt: ptr("2026-08-18T17:58:31.645Z")}
	if mod != nil {
		mod(&r)
	}
	return r
}

var fNow = time.Date(2026, 8, 18, 18, 0, 0, 0, time.UTC)

func TestCreditExhaustionWording(t *testing.T) {
	for _, s := range []string{
		fableOut,
		"You're out of usage credits. Switch to another model, or manage usage credits at claude.ai/settings/usage?from=cc_cli_limit_message, to continue.",
		"Claude usage limit reached — weekly limit reached. Try again in 2 days.",
		"5-hour limit reached.", "You've hit your session limit", "out of extra usage",
	} {
		if !IsCreditExhaustion(s) {
			t.Errorf("should match: %q", s)
		}
	}
	for _, s := range []string{"Please log in. Run `claude login` first.", "Maximum turns reached.", "HTTP 429: Too Many Requests", "", "Manage usage credits at claude.ai/settings/usage", codexQuota, oauthExpired} {
		if IsCreditExhaustion(s) {
			t.Errorf("should not match: %q", s)
		}
	}
}

func TestClassifyFailure(t *testing.T) {
	cases := map[string]string{
		codexQuota: "quota_window", oauthExpired: "auth", "Please log in. Run `claude login` first.": "auth",
		"Not logged in · Please run /login": "auth", fableOut: "credits", "Maximum turns reached.": "", "Internal error": "",
		"Your organization has disabled subscription access for Claude Code": "org_policy",
	}
	for in, want := range cases {
		if got := ClassifyFailure(in); got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}

func TestChainsAreSaneAndFableDropsToOpus(t *testing.T) {
	if FailoverChain["claude-fable-5"][0] != "claude-opus-5" {
		t.Fatal("Fable must drop to Opus 5 first")
	}
	for from, chain := range FailoverChain {
		for _, to := range chain {
			if to == from {
				t.Fatalf("%s loops onto itself", from)
			}
			if _, ok := FailoverChain[to]; !ok {
				t.Fatalf("%s falls to %s which has no ladder", from, to)
			}
		}
	}
}

func TestPlanDemotesEveryoneOnTheExhaustedModel(t *testing.T) {
	p := PlanFailover([]FailoverAgent{fa(nil), fa(func(a *FailoverAgent) { a.ID, a.Name, a.Status = "a-forge", "Forge", "idle" }),
		fa(func(a *FailoverAgent) {
			a.ID, a.Name, a.Model, a.Status = "a-sales", "Sales", ptr("claude-sonnet-4-6"), "idle"
		})},
		[]FailoverRunIn{fr(nil)}, fNow)
	var names []string
	for _, a := range p.Actions {
		names = append(names, a.AgentName)
		if a.To != "claude-opus-5" || !strings.Contains(a.Reason, "credit") {
			t.Fatalf("action %+v", a)
		}
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "Conductor,Forge" || strings.Join(p.Exhausted, ",") != "claude-fable-5" {
		t.Fatalf("plan %+v", p)
	}
}

func TestPlanOnlyLatestRunCountsAndNonCreditFailuresDoNothing(t *testing.T) {
	p := PlanFailover([]FailoverAgent{fa(func(a *FailoverAgent) { a.Status = "idle" })},
		[]FailoverRunIn{fr(func(r *FailoverRunIn) {
			r.Status, r.Error, r.FinishedAt = "succeeded", nil, ptr("2026-08-18T18:30:00.000Z")
		}), fr(nil)}, fNow)
	if len(p.Actions) != 0 || len(p.Exhausted) != 0 {
		t.Fatalf("recovered agent demoted: %+v", p)
	}
	p = PlanFailover([]FailoverAgent{fa(nil)}, []FailoverRunIn{fr(func(r *FailoverRunIn) { r.Error = ptr("Maximum turns reached.") })}, fNow)
	if len(p.Actions) != 0 {
		t.Fatalf("non-credit failure acted: %+v", p)
	}
	p = PlanFailover([]FailoverAgent{fa(nil)}, []FailoverRunIn{fr(func(r *FailoverRunIn) { r.Model = nil })}, fNow)
	if len(p.Actions) != 1 || p.Actions[0].To != "claude-opus-5" {
		t.Fatalf("seat model fallback: %+v", p)
	}
	p = PlanFailover([]FailoverAgent{fa(func(a *FailoverAgent) { a.Model = nil })}, []FailoverRunIn{fr(nil)}, fNow)
	if len(p.Actions) != 0 {
		t.Fatalf("no-model seat guessed at: %+v", p)
	}
	p = PlanFailover([]FailoverAgent{fa(nil)}, []FailoverRunIn{fr(func(r *FailoverRunIn) { r.Error, r.Summary = ptr("Internal error"), ptr(fableOut) })}, fNow)
	if len(p.Actions) != 1 {
		t.Fatalf("summary-only exhaustion ignored: %+v", p)
	}
}

func TestPlanSkipsExhaustedFallbacksAndReportsAnEmptyLadder(t *testing.T) {
	p := PlanFailover([]FailoverAgent{fa(nil), fa(func(a *FailoverAgent) { a.ID, a.Name, a.Model = "a-mkt", "Marketing", ptr("claude-opus-5") })},
		[]FailoverRunIn{fr(nil), fr(func(r *FailoverRunIn) { r.AgentID, r.Model = "a-mkt", ptr("claude-opus-5") })}, fNow)
	for _, a := range p.Actions {
		if a.AgentID == "a-conductor" && a.To != "claude-sonnet-5" {
			t.Fatalf("skip exhausted fallback: %+v", a)
		}
	}
	agents := []FailoverAgent{fa(nil), fa(func(a *FailoverAgent) { a.ID, a.Name, a.Model = "a-o", "O", ptr("claude-opus-5") }), fa(func(a *FailoverAgent) { a.ID, a.Name, a.Model = "a-s", "S", ptr("claude-sonnet-5") })}
	runs := []FailoverRunIn{fr(nil), fr(func(r *FailoverRunIn) { r.AgentID, r.Model = "a-o", ptr("claude-opus-5") }), fr(func(r *FailoverRunIn) { r.AgentID, r.Model = "a-s", ptr("claude-sonnet-5") })}
	p = PlanFailover(agents, runs, fNow)
	for _, a := range p.Actions {
		if a.AgentID == "a-conductor" {
			t.Fatalf("acted on an exhausted ladder: %+v", a)
		}
	}
	if !regexp.MustCompile(`no healthy fallback`).MatchString(strings.Join(p.Notes, " ")) {
		t.Fatalf("notes %v", p.Notes)
	}
	if e := PlanFailover(nil, nil, fNow); len(e.Actions)+len(e.Resumes)+len(e.Alerts)+len(e.Handoffs)+len(e.Exhausted)+len(e.Notes) != 0 || e.Actions == nil {
		t.Fatalf("empty board: %+v", e)
	}
}

func TestQuotaWindowAlertsThenResumes(t *testing.T) {
	codex := func(status string) FailoverAgent {
		return fa(func(a *FailoverAgent) { a.Model, a.Status = ptr("gpt-5.5"), status })
	}
	cr := fr(func(r *FailoverRunIn) {
		r.Error, r.Summary, r.Model, r.FinishedAt = ptr("Internal error"), ptr(codexQuota), ptr("gpt-5.5"), ptr("2026-09-05T05:09:00.000Z")
	})
	p := PlanFailover([]FailoverAgent{codex("error")}, []FailoverRunIn{cr}, time.Date(2026, 9, 5, 6, 0, 0, 0, time.UTC))
	if len(p.Actions)+len(p.Resumes)+len(p.Exhausted) != 0 || len(p.Alerts) != 1 || p.Alerts[0].Kind != "quota_window" ||
		p.Alerts[0].RunFinishedAt != "2026-09-05T05:09:00.000Z" || !strings.Contains(p.Alerts[0].Message, "2026-09-05T10:09") {
		t.Fatalf("fresh window: %+v", p)
	}
	later := time.Date(2026, 9, 5, 10, 10, 0, 0, time.UTC)
	if p := PlanFailover([]FailoverAgent{codex("error")}, []FailoverRunIn{cr}, later); len(p.Resumes) != 1 || !strings.Contains(p.Resumes[0].Reason, "window") {
		t.Fatalf("resume: %+v", p)
	}
	for _, st := range []string{"idle", "paused"} {
		if p := PlanFailover([]FailoverAgent{codex(st)}, []FailoverRunIn{cr}, later); len(p.Resumes) != 0 {
			t.Fatalf("%s seat resumed", st)
		}
	}
}

func TestExpiredLoginAlertsAndHandsOffToTheStandby(t *testing.T) {
	agents := []FailoverAgent{fa(func(a *FailoverAgent) { a.Model = ptr("claude-sonnet-5") }),
		{ID: "a-astra", Name: "Conductor (Astra)", Model: ptr("gpt-5.5"), Status: "idle"}}
	runs := []FailoverRunIn{fr(func(r *FailoverRunIn) {
		r.Error, r.Model, r.FinishedAt = ptr(oauthExpired), ptr("claude-sonnet-5"), ptr("2026-09-05T04:40:00.000Z")
	})}
	p := PlanFailover(agents, runs, time.Date(2026, 9, 6, 4, 40, 0, 0, time.UTC))
	if len(p.Actions) != 0 || len(p.Alerts) != 1 || p.Alerts[0].Kind != "auth" {
		t.Fatalf("auth: %+v", p)
	}
	if len(p.Handoffs) != 1 || p.Handoffs[0].ToAgentID != "a-astra" {
		t.Fatalf("handoff: %+v", p)
	}
	marker := AlertMarker(p.Alerts[0])
	if !strings.HasPrefix(marker, "[FounderOS v1 failover] Conductor auth 2026-09-05T04:40:00.000Z") || !strings.Contains(RenderAlert(p.Alerts[0]), "no reply needed") {
		t.Fatalf("marker %q", marker)
	}
}
