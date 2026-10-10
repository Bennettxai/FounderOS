// Package sales is the Sales pillar of the FounderOS agent roster on the
// bridge (spec §5): the Go port of the Brand Deal Agent
// (lib/agents/brand-deal-agent.ts), the Conductor, the Sales Agent, its
// account and payment lanes (Stripe included), and Sales Calls Data from
// FounderOS v1 lib/agents/real.ts. None of these agents calls an LLM.
//
// Sales Calls Data also owns the two memory writers of spec 5.4 in this
// group: the Plaud ingest (plaudingest.go) and the Fathom call archive
// (callarchive.go). Both write through Deps.Memory (Optimal Engine) only.
package sales

import (
	"context"
	"fmt"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
)

// Agents returns the Sales group in FounderOS v1's registry order.
func Agents(d roster.Deps) []agents.Agent {
	lookup := func(k string) string { return d.Res.Resolve(k) }
	pay := payments.New(d.Res)
	fathom := fathomcalls.New(d.Res)

	conductor := &Conductor{}
	if d.Devices != nil {
		conductor.LocalStack = d.Devices.HostConnector(devicepush.SourceLocalStack)
	}
	return []agents.Agent{
		&BrandDealAgent{
			Skill: func() (string, error) { return roster.LoadAgentSkill(BrandDealAgentFolder) },
			Deals: NewPgDealStore(d.Pool, d.Workspaces[BrandDealWorkspace]),
		},
		conductor,
		&SalesAgent{Leads: typeform.New(d.Res), Calls: fathom, Processors: pay.Processors},
		&LaunchpadCohort{Trakyo: trakyo.New(d.Res)},
		vantageSales(),
		paykitSales(lookup),
		vantagePaykit(lookup),
		&StripeSales{Stripe: pay.Stripe},
		&ProcessorConfirmation{Processors: pay.Processors},
		flexpayFinancing(lookup),
		NewCallsData(d, plaud.New(d.Res), fathom),
	}
}

var (
	metaConductor = agents.Meta{ID: "conductor", Name: "Conductor", DepartmentID: "dept-tech",
		Description: "Fans your message out to every agent at once and checks which instance hosts (OpenClaw, Ollama, tmux) are available for future bindings."}
	metaSalesAgent = agents.Meta{ID: "sales-agent", Name: "Sales Agent", DepartmentID: "dept-sales",
		Description: "Owns the sales pillar. Aggregates Lead Pulse and reports the live pipeline: Typeform leads, Fathom calls, payments."}
	metaLaunchpadCohort = agents.Meta{ID: "launchpad-cohort-sales", Name: "Launchpad Cohort", DepartmentID: "dept-sales",
		Description: "Launchpad Cohort sales lane: offers, calls, payment confirmation, and CRM context."}
	metaVantageSales = agents.Meta{ID: "vantage-sales", Name: "Vantage", DepartmentID: "dept-sales",
		Description: "Vantage sales lane: account pipeline, PayKit context, payment confirmation, and call data."}
	metaPaykitSales = agents.Meta{ID: "paykit-sales", Name: "PayKit", DepartmentID: "dept-finance",
		Description: "PayKit sales platform connection for offers and customer/payment context."}
	metaVantagePaykit = agents.Meta{ID: "vantage-paykit", Name: "Vantage PayKit", DepartmentID: "dept-sales",
		Description: "PayKit lane specifically under Vantage for offer, payment, and customer context."}
	metaProcessorConfirmation = agents.Meta{ID: "processor-confirmation", Name: "Processor Confirm", DepartmentID: "dept-finance",
		Description: "APIs to payment processors for confirming paid, failed, disputed, and pending states."}
	metaFlexPay = agents.Meta{ID: "flexpay-financing", Name: "FlexPay Financing", DepartmentID: "dept-finance",
		Description: "FlexPay financing options lane for sales offers and payment-plan context."}
	metaCallsData = agents.Meta{ID: "sales-calls-data", Name: "Sales Calls Data", DepartmentID: "dept-sales",
		Description: "Sales calls data lane for recordings, notes, outcomes, and follow-up context: Fathom on the calls, Plaud in the room."}
)

func live(s connectors.Status) string {
	if s.State == connectors.StateConnected {
		return "LIVE"
	}
	return "DOWN"
}

// ---- conductor ----------------------------------------------------------------

// Conductor reports the instance hosts (OpenClaw gateway, Ollama, tmux) from
// the local-stack reading a Mac pushes. Broadcast fan-out itself is the
// runtime's (agents.Runtime.Broadcast).
type Conductor struct {
	// LocalStack is the pushed local-stack source; nil when the bridge has
	// no device receiver.
	LocalStack connectors.Connector
}

func (c *Conductor) Meta() agents.Meta { return metaConductor }

func (c *Conductor) Run(ctx context.Context) (agents.Result, error) {
	if c.LocalStack == nil {
		return agents.Result{OK: false, Summary: "Instance hosts unknown: no device receiver on this bridge, so no Mac has pushed its local stack"}, nil
	}
	s := c.LocalStack.Status(ctx)
	return agents.Result{
		OK:      s.State == connectors.StateConnected,
		Summary: fmt.Sprintf("Instance hosts on this machine: %s · all agents bound to builtin runtime until the Mac mini lands", s.Detail),
		Data:    s.Meta,
	}, nil
}

// ---- sales agent ----------------------------------------------------------------

// SalesAgent aggregates the pipeline: Typeform leads in, Fathom calls held,
// payment processors out (Attio was the pipeline read until 2026-09-23).
type SalesAgent struct {
	Leads      connectors.Connector
	Calls      connectors.Connector
	Processors func() []payments.ProcessorInfo
}

func (a *SalesAgent) Meta() agents.Meta { return metaSalesAgent }

func (a *SalesAgent) Run(ctx context.Context) (agents.Result, error) {
	leadsCh, callsCh := make(chan connectors.Status, 1), make(chan connectors.Status, 1)
	go func() { leadsCh <- a.Leads.Status(ctx) }()
	go func() { callsCh <- a.Calls.Status(ctx) }()
	procs := processorRun(a.Processors)
	leads, calls := <-leadsCh, <-callsCh
	on := "DOWN"
	if procs.OK {
		on = "LIVE"
	}
	return agents.Result{
		OK: leads.State == connectors.StateConnected || calls.State == connectors.StateConnected || procs.OK,
		Summary: fmt.Sprintf("Sales pipeline · leads (Typeform) %s · calls (Fathom) %s · processors %s · PayKit/FlexPay lanes mapped",
			live(leads), live(calls), on),
		Data: map[string]any{"leads": leads, "calls": calls, "processors": procs},
	}, nil
}

// ---- agency launchpads ------------------------------------------------------------

// LaunchpadCohort runs on Trakyo attribution alone (the webinar funnel was
// retired 2026-08-19).
type LaunchpadCohort struct{ Trakyo connectors.Connector }

func (a *LaunchpadCohort) Meta() agents.Meta { return metaLaunchpadCohort }

func (a *LaunchpadCohort) Run(ctx context.Context) (agents.Result, error) {
	s := a.Trakyo.Status(ctx)
	ok := s.State == connectors.StateConnected
	summary := "Launchpad Cohort · Trakyo " + string(s.State)
	if !ok {
		summary += " — no live attribution source for this lane"
	}
	return agents.Result{OK: ok, Summary: summary, Data: map[string]any{"trakyo": s}}, nil
}

// ---- lanes -------------------------------------------------------------------------

// PlannedLane is a lane with no source wired yet; it always says so.
type PlannedLane struct {
	meta         agents.Meta
	name, detail string
}

func (l *PlannedLane) Meta() agents.Meta { return l.meta }

func (l *PlannedLane) Run(context.Context) (agents.Result, error) {
	return agents.Result{OK: false, Summary: fmt.Sprintf("%s lane planned — %s", l.name, l.detail)}, nil
}

func vantageSales() *PlannedLane {
	return &PlannedLane{meta: metaVantageSales, name: "Vantage sales", detail: "connect Vantage-specific CRM/payment/call sources"}
}

// EnvLane reports whether its credential is present; it reads nothing else.
type EnvLane struct {
	meta               agents.Meta
	name, key, purpose string
	lookup             func(string) string
}

func (l *EnvLane) Meta() agents.Meta { return l.meta }

func (l *EnvLane) Run(context.Context) (agents.Result, error) {
	if strings.TrimSpace(l.lookup(l.key)) == "" {
		return agents.Result{OK: false, Summary: fmt.Sprintf("%s not configured — set %s · %s", l.name, l.key, l.purpose)}, nil
	}
	return agents.Result{OK: true, Summary: fmt.Sprintf("%s credential present · %s", l.name, l.purpose)}, nil
}

// The PayKit lanes check PAYKIT_API_KEY exactly as FounderOS v1 does, even
// though the live PayKit reads use PAYKIT_LC_KEY / PAYKIT_VANTAGE_KEY.
func paykitSales(lookup func(string) string) *EnvLane {
	return &EnvLane{meta: metaPaykitSales, name: "PayKit", key: "PAYKIT_API_KEY", purpose: "offers, customers, and payment context", lookup: lookup}
}

func vantagePaykit(lookup func(string) string) *EnvLane {
	return &EnvLane{meta: metaVantagePaykit, name: "Vantage PayKit", key: "PAYKIT_API_KEY", purpose: "Vantage offer/payment context", lookup: lookup}
}

func flexpayFinancing(lookup func(string) string) *EnvLane {
	return &EnvLane{meta: metaFlexPay, name: "FlexPay", key: "FLEXPAY_API_KEY", purpose: "financing options for sales offers", lookup: lookup}
}

// ---- processor confirmation ----------------------------------------------------------

// ProcessorConfirmation lists the payment processors whose APIs are
// configured for confirming payment states.
type ProcessorConfirmation struct {
	Processors func() []payments.ProcessorInfo
}

func (p *ProcessorConfirmation) Meta() agents.Meta { return metaProcessorConfirmation }

func (p *ProcessorConfirmation) Run(context.Context) (agents.Result, error) {
	return processorRun(p.Processors), nil
}

func processorRun(list func() []payments.ProcessorInfo) agents.Result {
	configured := []payments.ProcessorInfo{}
	names := []string{}
	for _, p := range list() {
		if p.Configured {
			configured = append(configured, p)
			names = append(names, p.Name)
		}
	}
	if len(configured) == 0 {
		return agents.Result{OK: false, Summary: "No payment processor APIs configured yet — start with STRIPE_SECRET_KEY"}
	}
	return agents.Result{OK: true, Summary: strings.Join(names, ", ") + " configured for payment confirmation", Data: configured}
}
