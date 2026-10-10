package sales

import (
	"context"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
)

type fakeStatus connectors.Status

func (f fakeStatus) Status(context.Context) connectors.Status { return connectors.Status(f) }

func st(state connectors.State, detail string) fakeStatus {
	return fakeStatus{State: state, Detail: detail}
}

func run(t *testing.T, a agents.Agent) agents.Result {
	t.Helper()
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatalf("%s: unexpected error %v", a.Meta().ID, err)
	}
	return res
}

// The roster's id, name, department and description come from FounderOS v1
// lib/seed.ts, the rows the bridge ETL'd into founderos_agents.
func TestAgentsMatchTheSeededRoster(t *testing.T) {
	want := []agents.Meta{
		metaBrandDealAgent,
		{ID: "conductor", Name: "Conductor", DepartmentID: "dept-tech", Description: "Fans your message out to every agent at once and checks which instance hosts (OpenClaw, Ollama, tmux) are available for future bindings."},
		{ID: "sales-agent", Name: "Sales Agent", DepartmentID: "dept-sales", Description: "Owns the sales pillar. Aggregates Lead Pulse and reports the live pipeline: Typeform leads, Fathom calls, payments."},
		{ID: "launchpad-cohort-sales", Name: "Launchpad Cohort", DepartmentID: "dept-sales", Description: "Launchpad Cohort sales lane: offers, calls, payment confirmation, and CRM context."},
		{ID: "vantage-sales", Name: "Vantage", DepartmentID: "dept-sales", Description: "Vantage sales lane: account pipeline, PayKit context, payment confirmation, and call data."},
		{ID: "paykit-sales", Name: "PayKit", DepartmentID: "dept-finance", Description: "PayKit sales platform connection for offers and customer/payment context."},
		{ID: "vantage-paykit", Name: "Vantage PayKit", DepartmentID: "dept-sales", Description: "PayKit lane specifically under Vantage for offer, payment, and customer context."},
		{ID: "stripe-sales", Name: "Stripe", DepartmentID: "dept-finance", Description: "Stripe payment confirmation lane for sales workflows and account-level revenue checks."},
		{ID: "processor-confirmation", Name: "Processor Confirm", DepartmentID: "dept-finance", Description: "APIs to payment processors for confirming paid, failed, disputed, and pending states."},
		{ID: "flexpay-financing", Name: "FlexPay Financing", DepartmentID: "dept-finance", Description: "FlexPay financing options lane for sales offers and payment-plan context."},
		{ID: "sales-calls-data", Name: "Sales Calls Data", DepartmentID: "dept-sales", Description: "Sales calls data lane for recordings, notes, outcomes, and follow-up context: Fathom on the calls, Plaud in the room."},
	}
	got := Agents(roster.Deps{Res: connectors.Resolver{}})
	if len(got) != len(want) {
		t.Fatalf("got %d agents, want %d", len(got), len(want))
	}
	for i, a := range got {
		if a.Meta() != want[i] {
			t.Errorf("agent %d:\n got %+v\nwant %+v", i, a.Meta(), want[i])
		}
	}
	// It registers into the runtime without a duplicate-id panic.
	agents.New(agents.NewMemStore(), got...)

	// The brand deal agent reads the personal workspace's deals and the
	// repo's brand-deals skill.
	bd := Agents(roster.Deps{Res: connectors.Resolver{}, Workspaces: map[string]string{"personal": "pb", "founderos": "f"}})[0].(*BrandDealAgent)
	if st := bd.Deals.(*PgDealStore); st.workspaceID != "pb" {
		t.Fatalf("deal store workspace = %q, want the personal workspace", st.workspaceID)
	}
	if skill, err := bd.Skill(); err != nil || skill == "" {
		t.Fatalf("the brand-deals skill must load from the repo copy: %v", err)
	}
}

func TestConductorReportsInstanceHosts(t *testing.T) {
	res := run(t, &Conductor{LocalStack: fakeStatus{State: connectors.StateConnected, Detail: "OpenClaw up · Ollama up · tmux 3 sessions", Meta: map[string]any{"tmux": 3}}})
	if !res.OK || res.Summary != "Instance hosts on this machine: OpenClaw up · Ollama up · tmux 3 sessions · all agents bound to builtin runtime until the Mac mini lands" {
		t.Fatalf("%+v", res)
	}
	if m, _ := res.Data.(map[string]any); m["tmux"] != 3 {
		t.Fatalf("data = %#v", res.Data)
	}

	res = run(t, &Conductor{LocalStack: st(connectors.StateError, "local-stack: last push 2d ago (stale)")})
	if res.OK || !strings.Contains(res.Summary, "stale") {
		t.Fatalf("stale stack must fail: %+v", res)
	}

	res = run(t, &Conductor{})
	if res.OK || !strings.Contains(res.Summary, "unknown") {
		t.Fatalf("no device receiver must read unknown: %+v", res)
	}
}

func procs(configured ...string) func() []payments.ProcessorInfo {
	keyOf := map[string]string{"stripe": "STRIPE_SECRET_KEY", "wise-1": "WISE_1_TOKEN"}
	set := map[string]bool{}
	for _, id := range configured {
		set[keyOf[id]] = true
	}
	return func() []payments.ProcessorInfo {
		return payments.ConfiguredProcessors(func(k string) string {
			if set[k] {
				return "x"
			}
			return ""
		})
	}
}

func TestProcessorConfirmation(t *testing.T) {
	res := run(t, &ProcessorConfirmation{Processors: procs()})
	if res.OK || res.Summary != "No payment processor APIs configured yet — start with STRIPE_SECRET_KEY" {
		t.Fatalf("%+v", res)
	}
	res = run(t, &ProcessorConfirmation{Processors: procs("stripe", "wise-1")})
	if !res.OK || res.Summary != "Stripe, Wise configured for payment confirmation" {
		t.Fatalf("%+v", res)
	}
	if d, _ := res.Data.([]payments.ProcessorInfo); len(d) != 2 {
		t.Fatalf("data should carry only the configured processors: %#v", res.Data)
	}
}

func TestSalesAgentAggregatesThePipeline(t *testing.T) {
	a := &SalesAgent{
		Leads:      st(connectors.StateNotConfigured, "set TYPEFORM_API_KEY"),
		Calls:      st(connectors.StateConnected, "Fathom reachable"),
		Processors: procs("stripe"),
	}
	res := run(t, a)
	if !res.OK || res.Summary != "Sales pipeline · leads (Typeform) DOWN · calls (Fathom) LIVE · processors LIVE · PayKit/FlexPay lanes mapped" {
		t.Fatalf("%+v", res)
	}

	a = &SalesAgent{Leads: st(connectors.StateError, "HTTP 500"), Calls: st(connectors.StateError, "HTTP 401"), Processors: procs()}
	res = run(t, a)
	if res.OK || res.Summary != "Sales pipeline · leads (Typeform) DOWN · calls (Fathom) DOWN · processors DOWN · PayKit/FlexPay lanes mapped" {
		t.Fatalf("every source down must fail: %+v", res)
	}
}

func TestLaunchpadCohortRunsOnTrakyo(t *testing.T) {
	res := run(t, &LaunchpadCohort{Trakyo: st(connectors.StateConnected, "ok")})
	if !res.OK || res.Summary != "Launchpad Cohort · Trakyo connected" {
		t.Fatalf("%+v", res)
	}
	res = run(t, &LaunchpadCohort{Trakyo: st(connectors.StateError, "HTTP 502")})
	if res.OK || res.Summary != "Launchpad Cohort · Trakyo error — no live attribution source for this lane" {
		t.Fatalf("%+v", res)
	}
	res = run(t, &LaunchpadCohort{Trakyo: st(connectors.StateNotConfigured, "")})
	if res.OK || res.Summary != "Launchpad Cohort · Trakyo not_configured — no live attribution source for this lane" {
		t.Fatalf("%+v", res)
	}
}

func TestVantageSalesIsAPlannedLane(t *testing.T) {
	res := run(t, vantageSales())
	if res.OK || res.Summary != "Vantage sales lane planned — connect Vantage-specific CRM/payment/call sources" {
		t.Fatalf("%+v", res)
	}
}

func TestEnvLanesReportTheirCredential(t *testing.T) {
	none := func(string) string { return "" }
	set := func(k string) string {
		if k == "PAYKIT_API_KEY" || k == "FLEXPAY_API_KEY" {
			return "k"
		}
		return ""
	}
	cases := []struct {
		lane      func(func(string) string) *EnvLane
		off, on   string
		wantKeyed string
	}{
		{paykitSales, "PayKit not configured — set PAYKIT_API_KEY · offers, customers, and payment context", "PayKit credential present · offers, customers, and payment context", "PAYKIT_API_KEY"},
		{vantagePaykit, "Vantage PayKit not configured — set PAYKIT_API_KEY · Vantage offer/payment context", "Vantage PayKit credential present · Vantage offer/payment context", "PAYKIT_API_KEY"},
		{flexpayFinancing, "FlexPay not configured — set FLEXPAY_API_KEY · financing options for sales offers", "FlexPay credential present · financing options for sales offers", "FLEXPAY_API_KEY"},
	}
	for _, c := range cases {
		res := run(t, c.lane(none))
		if res.OK || res.Summary != c.off {
			t.Errorf("missing cred: %+v", res)
		}
		res = run(t, c.lane(set))
		if !res.OK || res.Summary != c.on {
			t.Errorf("cred present: %+v", res)
		}
	}
}
