// Package trading is the pure logic and the reads behind the /os/trading
// page, ported from FounderOS v1 lib/trading-view.ts, lib/trading-chart.ts
// (agentSummary), lib/trading-payload.ts and the limits read of
// app/api/trading/limits (clampLimits over DEFAULT_LIMITS / LIMIT_BOUNDS).
//
// The board is monitor-only: broker numbers arrive by push from the Markets
// Agent through the compat /api/trading/* routes into founderos_trading_*; the
// page only reads them. Honest: nothing fed is unknown, never $0.
package trading

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/robinhood"
)

// AgenticID is the sleeve the agent is allowed to trade; everything else is
// read-only to it.
const AgenticID = "agentic"

// Snapshot and Position are the robinhood feed's rows (TradingAccountSnapshot
// and TradingPosition in FounderOS v1 lib/schemas.ts).
type (
	Snapshot = robinhood.Snapshot
	Position = robinhood.Position
)

// Activity mirrors TradeActivitySchema: one row of the trade log.
type Activity struct {
	ID        string  `json:"id"`
	At        string  `json:"at"`
	AccountID string  `json:"accountId"`
	Agent     string  `json:"agent"`
	Action    string  `json:"action"`
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	PriceUSD  float64 `json:"priceUsd"`
	Rationale string  `json:"rationale"`
	Status    string  `json:"status"`
}

// AnalysisRow mirrors TradeAnalysisRowSchema; a nil score is "unscored".
type AnalysisRow struct {
	Ticker  string   `json:"ticker"`
	Score   *float64 `json:"score"`
	Verdict string   `json:"verdict"`
	Reason  string   `json:"reason"`
}

// Analysis mirrors TradeAnalysisSchema: what the agent looked at on a run.
type Analysis struct {
	ID        string        `json:"id"`
	At        string        `json:"at"`
	AccountID string        `json:"accountId"`
	Agent     string        `json:"agent"`
	Examined  int           `json:"examined"`
	Signals   int           `json:"signals"`
	Notes     string        `json:"notes"`
	Rows      []AnalysisRow `json:"rows"`
}

// Order mirrors TradingOrderSchema: live broker state, not history.
type Order struct {
	ID              string   `json:"id"`
	AccountID       string   `json:"accountId"`
	Symbol          string   `json:"symbol"`
	Side            string   `json:"side"`
	Type            string   `json:"type"`
	State           string   `json:"state"`
	Quantity        float64  `json:"quantity"`
	FilledQuantity  float64  `json:"filledQuantity"`
	DollarAmountUSD *float64 `json:"dollarAmountUsd"`
	LimitPriceUSD   *float64 `json:"limitPriceUsd"`
	PlacedAgent     string   `json:"placedAgent"`
	CreatedAt       string   `json:"createdAt"`
}

// Wallet is PhantomView: a public address read from a public RPC. Nil USD
// fields mean the price feed was down; the balance still stands.
type Wallet struct {
	Address   string   `json:"address"`
	SOL       float64  `json:"sol"`
	USDPerSOL *float64 `json:"usdPerSol"`
	USDValue  *float64 `json:"usdValue"`
	FetchedAt string   `json:"fetchedAt"`
}

// ShortAddress is "68SH…BST2": the first and last four of a base58 address.
func ShortAddress(a string) string {
	r := []rune(a)
	if len(r) <= 10 {
		return a
	}
	return string(r[:4]) + "…" + string(r[len(r)-4:])
}

// ── freshness ──────────────────────────────────────────────────────────

// Freshness is the feed's liveness: none, seeded, live or stale.
type Freshness struct {
	State string `json:"state"`
	Label string `json:"label"`
}

func parseAt(iso string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, iso)
	return t, err == nil
}

// SyncAge is "synced Nm ago" in the Brand Deals voice.
func SyncAge(iso string, now time.Time) string {
	at, ok := parseAt(iso)
	if !ok {
		return "synced at an unknown time"
	}
	m := int(math.Floor(now.Sub(at).Minutes()))
	if m < 0 {
		m = 0
	}
	switch {
	case m < 1:
		return "synced just now"
	case m < 60:
		return fmt.Sprintf("synced %dm ago", m)
	}
	h := m / 60
	if h < 24 {
		return fmt.Sprintf("synced %dh ago", h)
	}
	return fmt.Sprintf("synced %dd ago", h/24)
}

// FreshnessOf decides liveness by the freshest real push. Seeded rows are
// never live however recent; a real push older than 90 minutes is stale (the
// feed runs every 30 in market hours, so 90 is three misses).
func FreshnessOf(accounts []Snapshot, now time.Time) Freshness {
	if len(accounts) == 0 {
		return Freshness{"none", "no feed yet"}
	}
	var freshest *Snapshot
	for i := range accounts {
		a := &accounts[i]
		if a.Source == "seed" {
			continue
		}
		if freshest == nil || a.CapturedAt > freshest.CapturedAt {
			freshest = a
		}
	}
	if freshest == nil {
		return Freshness{"seeded", "seeded · example rows"}
	}
	label := SyncAge(freshest.CapturedAt, now)
	at, ok := parseAt(freshest.CapturedAt)
	if ok && now.Sub(at) <= 90*time.Minute {
		return Freshness{"live", label}
	}
	return Freshness{"stale", label}
}

// ── position sizes ─────────────────────────────────────────────────────

// Bucket is one dot-matrix column.
type Bucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

var sizeBuckets = []struct {
	label string
	max   float64
}{{"<$50", 50}, {"$50-250", 250}, {"$250-1k", 1000}, {"$1k+", math.Inf(1)}}

// PositionSizes buckets the book by market value, every bucket present so the
// dot matrix keeps its columns when the sleeve holds one thing.
func PositionSizes(ps []Position) []Bucket {
	out := make([]Bucket, len(sizeBuckets))
	for i, b := range sizeBuckets {
		min := math.Inf(-1)
		if i > 0 {
			min = sizeBuckets[i-1].max
		}
		out[i] = Bucket{Label: b.label}
		for _, p := range ps {
			if p.MarketValueUSD >= min && p.MarketValueUSD < b.max {
				out[i].Count++
			}
		}
	}
	return out
}

// ── the Accounts card (Deal Volume's shape) ────────────────────────────

// One hue per account (anti-drift rule), on the Monolith tokens. FounderOS v1
// used var(--accent), var(--ramp-1) and var(--ramp-4); the bridge has only
// the --bn-* tokens, so the two ramp greys are mixed from --bn-text.
const (
	HueAgentic    = "var(--bn-accent)"
	HueIndividual = "color-mix(in oklab, var(--bn-text) 60%, transparent)"
	HuePhantom    = "color-mix(in oklab, var(--bn-text) 85%, var(--bn-accent))"
)

// Chip is a dot chip beside a count-up headline.
type Chip struct {
	Tone string `json:"tone"`
	Text string `json:"text"`
}

// Meter is one hatched meter. A nil Frac is unknown: the kit draws no bar.
type Meter struct {
	Label   string   `json:"label"`
	Frac    *float64 `json:"frac"`
	Display string   `json:"display"`
	Hue     string   `json:"hue"`
}

// Volume is the Accounts card. A nil Headline means nothing was fed.
type Volume struct {
	Headline *float64 `json:"headline"`
	Chips    []Chip   `json:"chips"`
	Caption  string   `json:"caption"`
	Meters   []Meter  `json:"meters"`
	Foot     string   `json:"foot"`
}

// Money is toLocaleString('en-US', {style:'currency', currency:'USD'}) with
// two decimals, or none when cents is false.
func Money(n float64, cents bool) string {
	neg := n < 0
	n = math.Abs(n)
	var s string
	if cents {
		s = strconv.FormatFloat(math.Round(n*100)/100, 'f', 2, 64)
	} else {
		s = strconv.FormatFloat(math.Round(n), 'f', 0, 64)
	}
	intPart, frac, hasFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := "$" + b.String()
	if hasFrac {
		out += "." + frac
	}
	if neg && out != "$0.00" && out != "$0" {
		out = "-" + out
	}
	return out
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func fnum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// VolumeOf is the Accounts card: the brokerage total as the headline, the
// day's P&L and the book as chips, and one meter per account plus the
// wallet, each its honest share of everything held. A wallet with no price,
// or no wallet, is an empty meter that says why (v1 lib/trading-view.ts).
func VolumeOf(accounts []Snapshot, positions []Position, w *Wallet) Volume {
	var brokerage, day, invested float64
	for _, a := range accounts {
		brokerage += a.AccountValueUSD
		day += a.DayPnlUSD
	}
	for _, p := range positions {
		invested += p.MarketValueUSD
	}
	total := brokerage
	if w != nil && w.USDValue != nil {
		total += *w.USDValue
	}
	share := func(n float64) *float64 {
		if total <= 0 {
			return nil
		}
		f := math.Max(0, math.Min(1, n/total))
		return &f
	}

	var v Volume
	if len(accounts) > 0 {
		v.Headline = &brokerage
	}
	dayTone, sign := "accent", ""
	if day > 0 {
		dayTone, sign = "ok", "+"
	} else if day < 0 {
		dayTone, sign = "err", "-"
	}
	book := "all cash"
	if len(positions) > 0 {
		book = Money(invested, false) + " in " + plural(len(positions), "position")
	}
	v.Chips = []Chip{{dayTone, sign + Money(math.Abs(day), true) + " today"}, {"accent", book}}

	for _, a := range accounts {
		label, hue := a.AccountLabel+" · read-only to agents", HueIndividual
		if a.AccountID == AgenticID {
			label, hue = a.AccountLabel+" · agent may trade", HueAgentic
		}
		v.Meters = append(v.Meters, Meter{Label: label, Frac: share(a.AccountValueUSD), Display: Money(a.AccountValueUSD, true), Hue: hue})
	}
	empty := 0.0
	wm := Meter{Label: "Phantom · wallet not reachable", Frac: &empty, Display: "--", Hue: HuePhantom}
	if w != nil {
		wm.Label = fmt.Sprintf("Phantom · %s SOL · %s", fnum(w.SOL), ShortAddress(w.Address))
		wm.Display = "price unavailable"
		if w.USDValue != nil {
			wm.Frac, wm.Display = share(*w.USDValue), Money(*w.USDValue, true)
		}
	}
	v.Meters = append(v.Meters, wm)

	v.Caption = "brokerage, " + plural(len(accounts), "account")
	if w != nil {
		v.Caption += " · " + Money(total, true) + " with the wallet"
	}
	v.Foot = "only the agentic sleeve can be traded by an agent · "
	if w != nil && w.USDPerSOL != nil && *w.USDPerSOL != 0 {
		v.Foot += Money(*w.USDPerSOL, true) + " / SOL"
	} else {
		v.Foot += "wallet read from a public RPC"
	}
	return v
}

// ── the agent's sleeve ─────────────────────────────────────────────────

// AgentSummary is what the agent has actually done in its sleeve. HasActed is
// false until a trade filled: a funded, empty sleeve reads as idle.
type AgentSummary struct {
	HasActed         bool      `json:"hasActed"`
	TradeCount       int       `json:"tradeCount"`
	LastTrade        *Activity `json:"lastTrade"`
	LastTradeAt      *string   `json:"lastTradeAt"`
	DeployedUSD      float64   `json:"deployedUsd"`
	IdleCashUSD      float64   `json:"idleCashUsd"`
	UnrealizedPnlUSD float64   `json:"unrealizedPnlUsd"`
}

func round2(n float64) float64 { return math.Round(n*100) / 100 }

// AgentSummaryOf reads the sleeve from the fed rows only.
func AgentSummaryOf(sleeve *Snapshot, positions []Position, trades []Activity) AgentSummary {
	var filled []Activity
	for _, t := range trades {
		if t.Status == "filled" {
			filled = append(filled, t)
		}
	}
	sort.SliceStable(filled, func(i, j int) bool { return filled[i].At > filled[j].At })
	s := AgentSummary{HasActed: len(filled) > 0, TradeCount: len(filled)}
	if len(filled) > 0 {
		last := filled[0]
		s.LastTrade, s.LastTradeAt = &last, &last.At
	}
	var dep, un float64
	for _, p := range positions {
		dep += p.MarketValueUSD
		un += p.UnrealizedPnlUSD
	}
	s.DeployedUSD, s.UnrealizedPnlUSD = round2(dep), round2(un)
	if sleeve != nil {
		s.IdleCashUSD = sleeve.CashUSD
	}
	return s
}

// ── the agent's limits (read-only here) ────────────────────────────────

// Limits are the Markets Agent's guardrails (TradingLimitsSchema).
type Limits struct {
	MaxNotionalPerTradeUSD float64 `json:"maxNotionalPerTradeUsd"`
	MaxPositionPctOfSleeve float64 `json:"maxPositionPctOfSleeve"`
	MaxRiskPctPerTrade     float64 `json:"maxRiskPctPerTrade"`
	MaxConcurrentPositions float64 `json:"maxConcurrentPositions"`
	MaxTradesPerDay        float64 `json:"maxTradesPerDay"`
	MinSleeveValueUSD      float64 `json:"minSleeveValueUsd"`
	MaxDeployedCapitalUSD  float64 `json:"maxDeployedCapitalUsd"`
	Autopilot              bool    `json:"autopilot"`
}

// DefaultLimits is DEFAULT_LIMITS' seven numbers, autopilot off.
var DefaultLimits = Limits{
	MaxNotionalPerTradeUSD: 150, MaxPositionPctOfSleeve: 40, MaxRiskPctPerTrade: 2.5,
	MaxConcurrentPositions: 6, MaxTradesPerDay: 12, MinSleeveValueUSD: 480, MaxDeployedCapitalUSD: 560,
}

// LimitsView is GET /api/trading/limits' body, served inside the page read.
type LimitsView struct {
	Limits    Limits   `json:"limits"`
	Clamped   []string `json:"clamped"`
	Source    string   `json:"source"`
	UpdatedAt *string  `json:"updatedAt"`
}

// clampLimits holds limits inside LIMIT_BOUNDS, naming every moved field in
// LIMIT_BOUNDS order. The kill-switch floor is bounded from below.
func clampLimits(l Limits) (Limits, []string) {
	clamped := []string{}
	ceil := func(name string, v *float64, max float64) {
		if *v > max {
			*v = max
			clamped = append(clamped, name)
		}
	}
	ceil("maxNotionalPerTradeUsd", &l.MaxNotionalPerTradeUSD, 300)
	ceil("maxPositionPctOfSleeve", &l.MaxPositionPctOfSleeve, 50)
	ceil("maxRiskPctPerTrade", &l.MaxRiskPctPerTrade, 4)
	ceil("maxConcurrentPositions", &l.MaxConcurrentPositions, 16)
	ceil("maxTradesPerDay", &l.MaxTradesPerDay, 16)
	ceil("maxDeployedCapitalUsd", &l.MaxDeployedCapitalUSD, 600)
	if l.MinSleeveValueUSD < 450 {
		l.MinSleeveValueUSD = 450
		clamped = append(clamped, "minSleeveValueUsd")
	}
	return l, clamped
}

// LimitsViewOf is the limits the agent will actually run under: the stored
// row clamped, or the code defaults when none was ever saved.
func LimitsViewOf(stored *Limits, updatedAt string) LimitsView {
	if stored == nil {
		l, c := clampLimits(DefaultLimits)
		return LimitsView{Limits: l, Clamped: c, Source: "default"}
	}
	l, c := clampLimits(*stored)
	v := LimitsView{Limits: l, Clamped: c, Source: "stored"}
	if updatedAt != "" {
		v.UpdatedAt = &updatedAt
	}
	return v
}
