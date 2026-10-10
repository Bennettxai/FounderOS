package clients

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/wise"
)

// ---- fakes ------------------------------------------------------------------

type fakeStatus struct {
	st    connectors.Status
	calls atomic.Int32
}

func (f *fakeStatus) Status(context.Context) connectors.Status { f.calls.Add(1); return f.st }

func st(state connectors.State, detail string) *fakeStatus {
	return &fakeStatus{st: connectors.Status{State: state, Detail: detail}}
}

type fakePayments struct {
	procs    []payments.ProcessorInfo
	snap     stripe.Snapshot
	snapErr  error
	status   connectors.Status
	snapped  bool
	statused bool
}

func (f *fakePayments) Processors() []payments.ProcessorInfo { return f.procs }
func (f *fakePayments) StripeSnapshot(context.Context) (stripe.Snapshot, error) {
	f.snapped = true
	return f.snap, f.snapErr
}
func (f *fakePayments) Status(context.Context) connectors.Status { f.statused = true; return f.status }

type fakeRoster struct{ r Roster }

func (f fakeRoster) Roster(context.Context) Roster { return f.r }

type fakeFunnel struct {
	contacts []FunnelContact
	err      error
}

func (f fakeFunnel) Contacts(context.Context) ([]FunnelContact, error) { return f.contacts, f.err }

func usd(v float64) *float64 { return &v }

func run(t *testing.T, a agents.Agent) agents.Result {
	t.Helper()
	res, err := a.Run(context.Background())
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	return res
}

// ---- registry -----------------------------------------------------------------

func TestAgentsMatchFounderosOSMeta(t *testing.T) {
	want := []agents.Meta{
		{ID: "payments-pulse", Name: "Payments Pulse", DepartmentID: "dept-finance",
			Description: "Verifies payment processor connections and reports Stripe balance + recent charges."},
		{ID: "crm-pulse", Name: "Lead Pulse", DepartmentID: "dept-sales",
			Description: "Checks the lead lane end to end: Typeform leads in, calls booked on the calendar, calls held on Fathom."},
		{ID: "client-roster", Name: "Client Roster", DepartmentID: "dept-clients",
			Description: "The live client list: who actually paid (Stripe), with the funnel as the fallback, counted by venture and status."},
		{ID: "client-onboarding", Name: "Onboarding Agent", DepartmentID: "dept-clients",
			Description: "Readiness check for the onboarding SOP: the payment trigger (a Stripe charge) plus the Slack workspace it provisions."},
		{ID: "client-success", Name: "Client Success", DepartmentID: "dept-clients",
			Description: "Servicing rails: Fathom call notes and Plaud in-person recordings for deliverable tracking plus Slack for the check-in cadence."},
	}
	got := Agents(roster.Deps{Res: connectors.Resolver{EnvLocal: t.TempDir() + "/none"}})
	if len(got) != len(want) {
		t.Fatalf("got %d agents", len(got))
	}
	for i, a := range got {
		if a.Meta() != want[i] {
			t.Errorf("agent %d meta = %+v, want %+v", i, a.Meta(), want[i])
		}
	}
	agents.New(agents.NewMemStore(), got...) // panics on a duplicate id
}

// ---- payments-pulse -------------------------------------------------------------

func procs(ids ...string) []payments.ProcessorInfo {
	all := payments.ConfiguredProcessors(func(string) string { return "" })
	for i := range all {
		for _, id := range ids {
			if all[i].ID == id {
				all[i].Configured = true
			}
		}
	}
	return all
}

func TestPaymentsPulseNothingConfigured(t *testing.T) {
	p := &fakePayments{procs: procs()}
	res := run(t, &PaymentsPulse{Payments: p})
	if res.OK || res.Summary != "No payment processors configured — start with STRIPE_SECRET_KEY in .env.local" {
		t.Fatalf("res = %+v", res)
	}
	if p.snapped || p.statused {
		t.Fatal("no processor keyed: nothing should be read")
	}
}

func TestPaymentsPulseReadsStripeBalanceAndCharges(t *testing.T) {
	snap := stripe.Snapshot{
		Available:     []stripe.Money{{Amount: 123456, Currency: "usd"}},
		RecentCharges: []stripe.RecentCharge{{Amount: 1}, {Amount: 2}, {Amount: 3}},
	}
	res := run(t, &PaymentsPulse{Payments: &fakePayments{procs: procs("stripe", wise.Meta.ID), snap: snap}})
	if !res.OK || res.Summary != "Stripe: 1234.56 USD available · 3 recent charges" {
		t.Fatalf("res = %+v", res)
	}
	if got, ok := res.Data.(stripe.Snapshot); !ok || len(got.RecentCharges) != 3 {
		t.Fatalf("data = %#v", res.Data)
	}
}

func TestPaymentsPulseStripeFailureIsNotOK(t *testing.T) {
	res := run(t, &PaymentsPulse{Payments: &fakePayments{procs: procs("stripe"), snapErr: errors.New("Invalid API Key provided")}})
	if res.OK || !strings.Contains(res.Summary, "Invalid API Key provided") || !strings.HasPrefix(res.Summary, "Stripe") {
		t.Fatalf("res = %+v", res)
	}
}

// Without the LC Stripe key the TS said "configured (no live client yet)"
// unchecked; the bridge verifies the others through the payments status.
func TestPaymentsPulseWithoutStripeVerifiesTheOthers(t *testing.T) {
	ok := &fakePayments{procs: procs(wise.Meta.ID), status: connectors.Status{State: connectors.StateConnected, Detail: "Wise configured"}}
	res := run(t, &PaymentsPulse{Payments: ok})
	if !res.OK || res.Summary != "Wise configured" || ok.snapped {
		t.Fatalf("res = %+v", res)
	}
	bad := &fakePayments{procs: procs(wise.Meta.ID), status: connectors.Status{State: connectors.StateError, Detail: "Wise configured but none verified: HTTP 401"}}
	res = run(t, &PaymentsPulse{Payments: bad})
	if res.OK || !strings.Contains(res.Summary, "HTTP 401") {
		t.Fatalf("res = %+v", res)
	}
}

// ---- crm-pulse (Lead Pulse) -------------------------------------------------------

func TestLeadPulseReportsEachRail(t *testing.T) {
	a := &LeadPulse{
		Typeform: st(connectors.StateNotConfigured, "Set TYPEFORM_API_KEY"),
		Calendar: st(connectors.StateConnected, "3 upcoming"),
		Fathom:   st(connectors.StateError, "HTTP 500"),
	}
	res := run(t, a)
	if !res.OK || res.Summary != "Lead lane · Typeform not_configured · calendar connected · Fathom error" {
		t.Fatalf("res = %+v", res)
	}
	data := res.Data.(map[string]any)
	if data["typeform"] != connectors.StateNotConfigured || data["calendar"] != connectors.StateConnected || data["fathom"] != connectors.StateError {
		t.Fatalf("data = %#v", data)
	}
}

func TestLeadPulseNoLiveRailIsNotOK(t *testing.T) {
	res := run(t, &LeadPulse{
		Typeform: st(connectors.StateNotConfigured, ""),
		Calendar: st(connectors.StateError, "CalDAV read failed"),
		Fathom:   &fakeStatus{}, // a status with no state reads as error, never connected
	})
	if res.OK || res.Summary != "Lead lane · Typeform not_configured · calendar error · Fathom error" {
		t.Fatalf("res = %+v", res)
	}
}

// ---- client-roster ----------------------------------------------------------------

var funnel = []FunnelContact{
	{ID: "c1", Name: "Acme", Venture: "vantage", Status: "converted", AmountUSD: usd(5000)},
	{ID: "c2", Name: "Beta", Venture: "launchpad-cohort", Status: "converted", AmountUSD: usd(997)},
	{ID: "c3", Name: "Gamma", Venture: "vantage", Status: "converted"},
	{ID: "c4", Name: "Delta", Venture: "vantage", Status: "engaged"},
}

func TestClientRosterServesStripeWhenItHasPayers(t *testing.T) {
	live := Roster{State: connectors.StateConnected, Clients: []RosterClient{{ID: "stripe-ch_1", Name: "Pat", Venture: "vantage", Status: "won", AmountUSD: usd(10), Source: "stripe"}}}
	res := run(t, &ClientRoster{Funnel: fakeFunnel{contacts: funnel}, Stripe: fakeRoster{live}})
	if !res.OK || res.Summary != "Serving Stripe live: 1 paying clients on the roster · funnel backup holds 3 clients" {
		t.Fatalf("res = %+v", res)
	}
	data := res.Data.(map[string]any)
	if data["source"] != "stripe" {
		t.Fatalf("data = %#v", data)
	}
}

func TestClientRosterFallsBackToTheFunnel(t *testing.T) {
	live := Roster{State: connectors.StateNotConfigured, Clients: []RosterClient{}}
	res := run(t, &ClientRoster{Funnel: fakeFunnel{contacts: funnel}, Stripe: fakeRoster{live}})
	if !res.OK || res.Summary != "Serving seeded funnel: 3 clients (vantage 2 · launchpad-cohort 1) · 1 in pipeline · Stripe not_configured" {
		t.Fatalf("res = %+v", res)
	}
	data := res.Data.(map[string]any)
	clients := data["clients"].([]RosterClient)
	if data["source"] != "funnel" || len(clients) != 3 || clients[0].Source != "funnel" || clients[2].AmountUSD != nil {
		t.Fatalf("data = %#v", data)
	}
}

func TestClientRosterEmptyFunnelSaysNoneYet(t *testing.T) {
	res := run(t, &ClientRoster{Funnel: fakeFunnel{contacts: []FunnelContact{}}, Stripe: fakeRoster{Roster{State: connectors.StateConnected, Clients: []RosterClient{}}}})
	if !res.OK || res.Summary != "Serving seeded funnel: 0 clients (none yet) · 0 in pipeline · Stripe connected" {
		t.Fatalf("res = %+v", res)
	}
}

func TestClientRosterStripeErrorIsNamed(t *testing.T) {
	live := Roster{State: connectors.StateError, Detail: "Stripe: Invalid API Key", Clients: []RosterClient{}}
	res := run(t, &ClientRoster{Funnel: fakeFunnel{contacts: funnel}, Stripe: fakeRoster{live}})
	if !res.OK || !strings.HasSuffix(res.Summary, "· Stripe error (Stripe: Invalid API Key)") {
		t.Fatalf("res = %+v", res)
	}
}

func TestClientRosterFunnelDownWithoutStripeIsNotOK(t *testing.T) {
	res := run(t, &ClientRoster{Funnel: fakeFunnel{err: errors.New("connection refused")}, Stripe: fakeRoster{Roster{State: connectors.StateNotConfigured}}})
	if res.OK || !strings.Contains(res.Summary, "connection refused") || strings.Contains(res.Summary, "0 clients") {
		t.Fatalf("res = %+v", res)
	}
}

func TestClientRosterFunnelDownWithStripeSaysBackupUnknown(t *testing.T) {
	live := Roster{State: connectors.StateConnected, Clients: []RosterClient{{ID: "stripe-ch_1", Name: "Pat", Status: "won", Source: "stripe"}}}
	res := run(t, &ClientRoster{Funnel: fakeFunnel{err: errors.New("connection refused")}, Stripe: fakeRoster{live}})
	if !res.OK || res.Summary != "Serving Stripe live: 1 paying clients on the roster · funnel backup unknown (connection refused)" {
		t.Fatalf("res = %+v", res)
	}
}

// ---- client-onboarding -----------------------------------------------------------

func TestOnboardingBothRailsLive(t *testing.T) {
	res := run(t, &Onboarding{Payments: fakeRoster{Roster{State: connectors.StateConnected}}, Slack: st(connectors.StateConnected, "")})
	if !res.OK || res.Summary != "Onboarding rails: payments (Stripe) connected · Slack connected" {
		t.Fatalf("res = %+v", res)
	}
	data := res.Data.(map[string]any)
	if data["payments"] != connectors.StateConnected || data["slack"] != connectors.StateConnected {
		t.Fatalf("data = %#v", data)
	}
}

func TestOnboardingOneRailMissing(t *testing.T) {
	res := run(t, &Onboarding{Payments: fakeRoster{Roster{State: connectors.StateNotConfigured}}, Slack: st(connectors.StateConnected, "")})
	if !res.OK || res.Summary != "Onboarding rails: payments (Stripe) not_configured · Slack connected — connect the missing rail to run onboarding end to end" {
		t.Fatalf("res = %+v", res)
	}
}

func TestOnboardingNoRailIsNotOK(t *testing.T) {
	res := run(t, &Onboarding{Payments: fakeRoster{Roster{State: connectors.StateError, Detail: "boom"}}, Slack: st(connectors.StateError, "auth failed")})
	if res.OK || res.Summary != "Onboarding rails: payments (Stripe) error · Slack error — connect the missing rail to run onboarding end to end" {
		t.Fatalf("res = %+v", res)
	}
}

// ---- client-success -------------------------------------------------------------

func yes() bool { return true }
func no() bool  { return false }

func TestClientSuccessAllRails(t *testing.T) {
	res := run(t, &ClientSuccess{Slack: st(connectors.StateConnected, ""), FathomConfigured: yes, PlaudConfigured: yes})
	if !res.OK || res.Summary != "Servicing rails: Fathom configured · Plaud configured · Slack connected" {
		t.Fatalf("res = %+v", res)
	}
	data := res.Data.(map[string]any)
	if data["fathom"] != "configured" || data["plaud"] != "configured" || data["slack"] != connectors.StateConnected {
		t.Fatalf("data = %#v", data)
	}
}

func TestClientSuccessSomeRails(t *testing.T) {
	res := run(t, &ClientSuccess{Slack: st(connectors.StateError, "x"), FathomConfigured: yes, PlaudConfigured: no})
	if !res.OK || res.Summary != "Servicing rails: Fathom configured · Plaud not_configured · Slack error" {
		t.Fatalf("res = %+v", res)
	}
}

func TestClientSuccessNoRails(t *testing.T) {
	res := run(t, &ClientSuccess{Slack: st(connectors.StateNotConfigured, ""), FathomConfigured: no, PlaudConfigured: no})
	if res.OK || res.Summary != "Servicing rails: Fathom not_configured · Plaud not_configured · Slack not_configured — set FATHOM_API_KEY, PLAUD_REFRESH_TOKEN and a Slack bot token to service clients" {
		t.Fatalf("res = %+v", res)
	}
}

// ---- wiring over real connectors, no network -------------------------------------

// With no credentials anywhere, every agent built by Agents fails honestly
// and none reaches the network (unkeyed connectors never dial).
func TestAgentsWithNoCredentialsFailHonestly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"STRIPE_SECRET_KEY", "STRIPE_VANTAGE_KEY", "PAYPAL_CLIENT_ID", "PAYPAL_CLIENT_SECRET",
		"PAYKIT_LC_KEY", "PAYKIT_VANTAGE_KEY", "WISE_1_TOKEN", "TYPEFORM_API_KEY", "FATHOM_API_KEY",
		"SLACK_BOT_TOKEN", "PLAUD_REFRESH_TOKEN", "PLAUD_TOKEN_FILE", "FUNNEL_PROVIDER",
		"INBOX_1_HOST", "INBOX_2_HOST", "INBOX_3_HOST", "INBOX_4_HOST"} {
		t.Setenv(k, "")
	}
	d := roster.Deps{Res: connectors.Resolver{EnvLocal: t.TempDir() + "/env.local"}}
	for _, a := range Agents(d) {
		res := run(t, a)
		if res.OK {
			t.Errorf("%s ran OK with no credentials: %q", a.Meta().ID, res.Summary)
		}
		if res.Summary == "" {
			t.Errorf("%s gave no summary", a.Meta().ID)
		}
	}
}
