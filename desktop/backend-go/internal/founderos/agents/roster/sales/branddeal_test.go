package sales

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
)

type fakeDeals struct {
	deals []BrandDeal
	err   error
	calls int
}

func (f *fakeDeals) Deals(context.Context) ([]BrandDeal, error) {
	f.calls++
	return f.deals, f.err
}

const brandSkill = "# Victor\n> - [ ] **Audience.** Platform by platform.\n> - [ ] **Payment terms.**\n- [x] **Known.**"

func brandAgent(store DealStore) *BrandDealAgent {
	return &BrandDealAgent{
		Skill: func() (string, error) { return brandSkill, nil },
		Deals: store,
		Now:   func() time.Time { return time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC) },
	}
}

func TestBrandDealAgentMetaIsTheSeededRow(t *testing.T) {
	want := agents.Meta{ID: "brand-deal-agent", Name: "Brand Deal Agent", DepartmentID: "dept-sales",
		Description: "Negotiates as Victor, the operator’s brand deal manager: qualifies inbound, anchors and counters, chases unpaid invoices, and bumps stalled threads. A tested contact governor decides whether a thread may be touched at all (five bumps maximum, one revival per brand per six months). Drafts only, never sends."}
	if m := (&BrandDealAgent{}).Meta(); m != want {
		t.Fatalf("meta = %+v", m)
	}
}

// An empty store is an honest empty run, not a broken connector.
func TestBrandDealAgentEmptyStore(t *testing.T) {
	res := run(t, brandAgent(&fakeDeals{}))
	if !res.OK || res.Summary != "No brand deals in the OS yet, so there is nothing to work. Ingest a deal and the agent has something to do." {
		t.Fatalf("%+v", res)
	}
	d := res.Data.(BrandDealData)
	if d.DealCount != 0 || d.Actions == nil || len(d.Actions) != 0 || d.Today != "" ||
		!slices.Equal(d.AwaitingFromFounderos, []string{"Audience.", "Payment terms."}) {
		t.Fatalf("data = %+v", d)
	}
}

func TestBrandDealAgentTriagesTheStore(t *testing.T) {
	store := &fakeDeals{deals: []BrandDeal{
		deal(func(d *BrandDeal) {
			d.ID, d.Brand, d.Status, d.Deadline = "a", "Late Co", "Filming", dayp("2026-08-10")
		}),
		deal(func(d *BrandDeal) {
			d.ID, d.Brand, d.Status, d.AmountAgreedUSD = "b", "Owes Co", "Invoiced", money(9000)
		}),
		deal(func(d *BrandDeal) { d.ID, d.Brand = "c", "Quiet Co" }),
	}}
	res := run(t, brandAgent(store))
	if !res.OK || res.Summary != "2 actions across 3 deals · 1 overdue · 1 unpaid · first: Late Co (overdue)" {
		t.Fatalf("%+v", res)
	}
	d := res.Data.(BrandDealData)
	if d.Today != "2026-08-19" || d.DealCount != 3 || len(d.Actions) != 2 || d.Actions[1].Kind != KindChasePayment || len(d.AwaitingFromFounderos) != 2 {
		t.Fatalf("data = %+v", d)
	}
}

func TestBrandDealAgentFailsLoudly(t *testing.T) {
	// No skill file: the run fails before the store is read, as in FounderOS v1.
	store := &fakeDeals{}
	a := brandAgent(store)
	a.Skill = func() (string, error) {
		return "", errors.New("missing or empty skill file: agents/brand-deals/skill.md")
	}
	if res, err := a.Run(context.Background()); err == nil || res.OK || store.calls != 0 {
		t.Fatalf("res %+v err %v calls %d", res, err, store.calls)
	}
	// An unreadable store is unknown, never "no deals".
	res := run(t, brandAgent(&fakeDeals{err: errors.New("pg down")}))
	if res.OK || res.Summary != "Brand deals unknown: the OS deal store could not be read (pg down)" {
		t.Fatalf("%+v", res)
	}
	if res := run(t, brandAgent(nil)); res.OK {
		t.Fatalf("unwired store = %+v", res)
	}
}

func TestPgDealStoreReadsTheWorkspacesRealDeals(t *testing.T) {
	pool, _ := throwawayDB(t)
	ctx := context.Background()
	var brand, other string
	for slug, into := range map[string]*string{"personal": &brand, "elsewhere": &other} {
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $1, 'u1') RETURNING id::text`, slug).Scan(into); err != nil {
			t.Fatal(err)
		}
	}
	ins := `INSERT INTO founderos_brand_deals (id, workspace_id, brand, status, deal_value_usd, budget_usd, amount_agreed_usd,
		suggested_rate_usd, paid_in_full, deadline, follow_up_date, notion_url, last_edited, seeded)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	rows := [][]any{
		{"old", brand, "beta", "New", nil, nil, nil, nil, false, nil, nil, "u1", "2026-08-01T00:00:00Z", false},
		{"new", brand, "Zed", "Invoiced", 5000.0, nil, 4500.0, nil, false, "2026-08-10T00:00:00Z", nil, "u2", "2026-08-18T00:00:00Z", false},
		{"tie", brand, "Alpha", "Negotiating", nil, 900.0, nil, 1200.0, false, nil, "2026-08-19T00:00:00Z", "u3", "2026-08-18T00:00:00Z", false},
		{"seed", brand, "Placeholder", "Invoiced", nil, nil, nil, nil, false, nil, nil, "u4", "2026-08-19T00:00:00Z", true},
		{"foreign", other, "Other", "New", nil, nil, nil, nil, false, nil, nil, "u5", "2026-08-19T00:00:00Z", false},
	}
	for _, r := range rows {
		if _, err := pool.Exec(ctx, ins, r...); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewPgDealStore(pool, brand).Deals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, d := range got {
		ids = append(ids, d.ID)
	}
	// last_edited DESC, then brand in byte order (SQLite's BINARY): "Alpha" < "Zed" < "beta".
	if !slices.Equal(ids, []string{"tie", "new", "old"}) {
		t.Fatalf("ids = %v", ids)
	}
	z := got[1]
	if z.Brand != "Zed" || z.Status != "Invoiced" || *z.DealValueUSD != 5000 || *z.AmountAgreedUSD != 4500 || z.BudgetUSD != nil ||
		z.Deadline == nil || dealDate(*z.Deadline) != "2026-08-10" || z.FollowUpDate != nil || z.NotionURL != "u2" || z.PaidInFull || z.Seeded {
		t.Fatalf("zed = %+v", z)
	}
	if a := got[0]; *a.SuggestedRateUSD != 1200 || *a.BudgetUSD != 900 || dealDate(*a.FollowUpDate) != "2026-08-19" {
		t.Fatalf("alpha = %+v", a)
	}

	if _, err := NewPgDealStore(pool, "").Deals(ctx); err == nil {
		t.Fatal("no workspace must be an error, never an empty pipeline")
	}
	if _, err := NewPgDealStore(nil, brand).Deals(ctx); err == nil {
		t.Fatal("no pool must be an error")
	}
}
