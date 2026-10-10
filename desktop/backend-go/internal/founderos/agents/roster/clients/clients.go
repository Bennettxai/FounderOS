// Package clients ports the FounderOS v1 finance, lead-lane and client agents
// (lib/agents/real.ts: payments-pulse, crm-pulse, client-roster,
// client-onboarding, client-success) to the bridge.
//
// All five are read-only health and roster checks in FounderOS v1: none sends,
// posts or creates anything, so none does here. None calls an LLM.
package clients

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// Agents builds the group over the ported connectors and the founderos_*
// tables.
func Agents(d roster.Deps) []agents.Agent {
	return []agents.Agent{
		&PaymentsPulse{Payments: payments.New(d.Res)},
		&LeadPulse{Typeform: typeform.New(d.Res), Calendar: gcal.New(d.Res), Fathom: fathomcalls.New(d.Res)},
		&ClientRoster{Funnel: &PgFunnel{Pool: d.Pool, Workspaces: d.Workspaces}, Stripe: NewStripeRoster(d.Res)},
		&Onboarding{Payments: NewStripeRoster(d.Res), Slack: slack.New(d.Res)},
		&ClientSuccess{Slack: slack.New(d.Res), FathomConfigured: fathomKeyed(d.Res), PlaudConfigured: plaud.New(d.Res).Configured},
	}
}

// fathomKeyed is the presence check the TS makes (FATHOM_API_KEY set), over
// the Fathom connector's credential order: env.local, env, clue-agent, social.
func fathomKeyed(res connectors.Resolver) func() bool {
	return func() bool {
		social, clue, _, _ := connectors.CredFiles()
		return res.Resolve("FATHOM_API_KEY", clue, social) != ""
	}
}

// StatusChecker is any connector's honest status read.
type StatusChecker interface {
	Status(ctx context.Context) connectors.Status
}

// stateOf never lets an empty status pass for anything but an error.
func stateOf(s connectors.Status) connectors.State {
	if s.State == "" {
		return connectors.StateError
	}
	return s.State
}

// statuses reads several rails at once.
func statuses(ctx context.Context, checks ...StatusChecker) []connectors.Status {
	out := make([]connectors.Status, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func(i int, c StatusChecker) {
			defer wg.Done()
			out[i] = c.Status(ctx)
			out[i].State = stateOf(out[i])
		}(i, c)
	}
	wg.Wait()
	return out
}

// ---- payments-pulse -----------------------------------------------------------

// PaymentsReader is the slice of *payments.Connector Payments Pulse reads.
type PaymentsReader interface {
	Processors() []payments.ProcessorInfo
	StripeSnapshot(ctx context.Context) (stripe.Snapshot, error)
	Status(ctx context.Context) connectors.Status
}

type PaymentsPulse struct{ Payments PaymentsReader }

func (*PaymentsPulse) Meta() agents.Meta {
	return agents.Meta{
		ID: "payments-pulse", Name: "Payments Pulse", DepartmentID: "dept-finance",
		Description: "Verifies payment processor connections and reports Stripe balance + recent charges.",
	}
}

func (a *PaymentsPulse) Run(ctx context.Context) (agents.Result, error) {
	var configured []payments.ProcessorInfo
	stripeKeyed := false
	for _, p := range a.Payments.Processors() {
		if p.Configured {
			configured = append(configured, p)
			stripeKeyed = stripeKeyed || p.ID == stripe.LaunchpadCohort.ID
		}
	}
	if len(configured) == 0 {
		return agents.Result{OK: false, Summary: "No payment processors configured — start with STRIPE_SECRET_KEY in .env.local"}, nil
	}
	if stripeKeyed {
		snap, err := a.Payments.StripeSnapshot(ctx)
		if err != nil {
			return agents.Result{OK: false, Summary: "Stripe key set but the read failed: " + err.Error()}, nil
		}
		return agents.Result{
			OK:      true,
			Summary: fmt.Sprintf("Stripe: %s available · %d recent charges", stripe.AvailableLabel(snap), len(snap.RecentCharges)),
			Data:    snap,
		}, nil
	}
	// The TS reported the others "configured (no live client yet)" without
	// a check; the bridge has live clients, so the payments status verifies
	// each configured processor and says connected only if one answers.
	s := a.Payments.Status(ctx)
	return agents.Result{OK: stateOf(s) == connectors.StateConnected, Summary: s.Detail, Data: s.Meta}, nil
}

// ---- crm-pulse (Lead Pulse) -------------------------------------------------------

// LeadPulse was the Attio CRM worker; Attio is retired (2026-09-23), so it
// reports the lead lane that replaced it: Typeform leads, calendar bookings,
// Fathom calls.
type LeadPulse struct{ Typeform, Calendar, Fathom StatusChecker }

func (*LeadPulse) Meta() agents.Meta {
	return agents.Meta{
		ID: "crm-pulse", Name: "Lead Pulse", DepartmentID: "dept-sales",
		Description: "Checks the lead lane end to end: Typeform leads in, calls booked on the calendar, calls held on Fathom.",
	}
}

func (a *LeadPulse) Run(ctx context.Context) (agents.Result, error) {
	s := statuses(ctx, a.Typeform, a.Calendar, a.Fathom)
	live := 0
	for _, r := range s {
		if r.State == connectors.StateConnected {
			live++
		}
	}
	return agents.Result{
		OK:      live > 0,
		Summary: fmt.Sprintf("Lead lane · Typeform %s · calendar %s · Fathom %s", s[0].State, s[1].State, s[2].State),
		Data: map[string]any{
			"typeform": s[0].State, "calendar": s[1].State, "fathom": s[2].State,
			"detail": map[string]string{"typeform": s[0].Detail, "calendar": s[1].Detail, "fathom": s[2].Detail},
		},
	}, nil
}

// ---- client-roster ----------------------------------------------------------------

type ClientRoster struct {
	Funnel FunnelStore
	Stripe RosterSource
}

func (*ClientRoster) Meta() agents.Meta {
	return agents.Meta{
		ID: "client-roster", Name: "Client Roster", DepartmentID: "dept-clients",
		Description: "The live client list: who actually paid (Stripe), with the funnel as the fallback, counted by venture and status.",
	}
}

func stripeLabel(r Roster) string {
	if r.State == connectors.StateError && r.Detail != "" {
		return fmt.Sprintf("%s (%s)", r.State, r.Detail)
	}
	return string(r.State)
}

func (a *ClientRoster) Run(ctx context.Context) (agents.Result, error) {
	var (
		contacts []FunnelContact
		ferr     error
		live     Roster
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); contacts, ferr = a.Funnel.Contacts(ctx) }()
	go func() { defer wg.Done(); live = a.Stripe.Roster(ctx) }()
	wg.Wait()
	if live.State == "" {
		live.State = connectors.StateError
	}

	converted := []RosterClient{}
	var ventureOrder []string
	byVenture := map[string]int{}
	for _, c := range contacts {
		if c.Status != "converted" {
			continue
		}
		converted = append(converted, RosterClient{ID: c.ID, Name: c.Name, Venture: c.Venture, Status: c.Status, AmountUSD: c.AmountUSD, Source: "funnel"})
		if byVenture[c.Venture] == 0 {
			ventureOrder = append(ventureOrder, c.Venture)
		}
		byVenture[c.Venture]++
	}
	servingStripe := live.State == connectors.StateConnected && len(live.Clients) > 0
	source := "funnel"
	if servingStripe {
		source = "stripe"
	}
	data := map[string]any{
		"source": source,
		"stripe": map[string]any{"state": live.State, "clients": len(live.Clients), "detail": live.Detail},
	}

	if ferr != nil {
		data["funnel"] = map[string]any{"error": ferr.Error()}
		if servingStripe {
			data["clients"] = live.Clients
			return agents.Result{
				OK:      true,
				Summary: fmt.Sprintf("Serving Stripe live: %d paying clients on the roster · funnel backup unknown (%s)", len(live.Clients), ferr.Error()),
				Data:    data,
			}, nil
		}
		return agents.Result{
			OK:      false,
			Summary: fmt.Sprintf("Client roster unknown: funnel unavailable (%s) · Stripe %s", ferr.Error(), stripeLabel(live)),
			Data:    data,
		}, nil
	}

	data["clients"] = converted
	if servingStripe {
		return agents.Result{
			OK:      true,
			Summary: fmt.Sprintf("Serving Stripe live: %d paying clients on the roster · funnel backup holds %d clients", len(live.Clients), len(converted)),
			Data:    data,
		}, nil
	}
	parts := make([]string, 0, len(ventureOrder))
	for _, v := range ventureOrder {
		parts = append(parts, fmt.Sprintf("%s %d", v, byVenture[v]))
	}
	ventures := strings.Join(parts, " · ")
	if ventures == "" {
		ventures = "none yet"
	}
	return agents.Result{
		OK: true,
		Summary: fmt.Sprintf("Serving seeded funnel: %d clients (%s) · %d in pipeline · Stripe %s",
			len(converted), ventures, len(contacts)-len(converted), stripeLabel(live)),
		Data: data,
	}, nil
}

// ---- client-onboarding -----------------------------------------------------------

// Onboarding is a readiness check for the onboarding SOP. The trigger was a
// closed-won Attio deal until Attio was retired; a settled payment is the
// trigger now. It provisions nothing: FounderOS v1's agent only reports.
type Onboarding struct {
	Payments RosterSource
	Slack    StatusChecker
}

func (*Onboarding) Meta() agents.Meta {
	return agents.Meta{
		ID: "client-onboarding", Name: "Onboarding Agent", DepartmentID: "dept-clients",
		Description: "Readiness check for the onboarding SOP: the payment trigger (a Stripe charge) plus the Slack workspace it provisions.",
	}
}

func (a *Onboarding) Run(ctx context.Context) (agents.Result, error) {
	var (
		pay Roster
		slk connectors.Status
		wg  sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); pay = a.Payments.Roster(ctx) }()
	go func() { defer wg.Done(); slk = statuses(ctx, a.Slack)[0] }()
	wg.Wait()
	if pay.State == "" {
		pay.State = connectors.StateError
	}
	live := 0
	for _, s := range []connectors.State{pay.State, slk.State} {
		if s == connectors.StateConnected {
			live++
		}
	}
	suffix := ""
	if live < 2 {
		suffix = " — connect the missing rail to run onboarding end to end"
	}
	return agents.Result{
		OK:      live > 0,
		Summary: fmt.Sprintf("Onboarding rails: payments (Stripe) %s · Slack %s%s", pay.State, slk.State, suffix),
		Data: map[string]any{
			"payments": pay.State, "slack": slk.State,
			"detail": map[string]string{"payments": pay.Detail, "slack": slk.Detail},
		},
	}, nil
}

// ---- client-success -------------------------------------------------------------

// ClientSuccess reports the servicing rails. As in FounderOS v1, Fathom and
// Plaud are presence checks (no network, no Plaud token refresh) and Slack
// is a live auth.test.
type ClientSuccess struct {
	Slack            StatusChecker
	FathomConfigured func() bool
	PlaudConfigured  func() bool
}

func (*ClientSuccess) Meta() agents.Meta {
	return agents.Meta{
		ID: "client-success", Name: "Client Success", DepartmentID: "dept-clients",
		Description: "Servicing rails: Fathom call notes and Plaud in-person recordings for deliverable tracking plus Slack for the check-in cadence.",
	}
}

func presence(ok bool) string {
	if ok {
		return "configured"
	}
	return "not_configured"
}

func (a *ClientSuccess) Run(ctx context.Context) (agents.Result, error) {
	slk := statuses(ctx, a.Slack)[0]
	fathom, plaudState := presence(a.FathomConfigured()), presence(a.PlaudConfigured())
	live := 0
	if slk.State == connectors.StateConnected {
		live++
	}
	if fathom == "configured" {
		live++
	}
	if plaudState == "configured" {
		live++
	}
	suffix := ""
	if live == 0 {
		suffix = " — set FATHOM_API_KEY, PLAUD_REFRESH_TOKEN and a Slack bot token to service clients"
	}
	return agents.Result{
		OK:      live > 0,
		Summary: fmt.Sprintf("Servicing rails: Fathom %s · Plaud %s · Slack %s%s", fathom, plaudState, slk.State, suffix),
		Data:    map[string]any{"fathom": fathom, "plaud": plaudState, "slack": slk.State},
	}, nil
}
