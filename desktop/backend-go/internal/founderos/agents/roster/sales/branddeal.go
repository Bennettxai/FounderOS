package sales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/agents/roster"
)

// BrandDealAgentFolder is the skill folder (agents/brand-deals/skill.md).
const BrandDealAgentFolder = "brand-deals"

// BrandDealWorkspace owns founderos_brand_deals (etl.WSPersonal).
const BrandDealWorkspace = "personal"

var metaBrandDealAgent = agents.Meta{ID: "brand-deal-agent", Name: "Brand Deal Agent", DepartmentID: "dept-sales",
	Description: "Negotiates as Victor, the operator’s brand deal manager: qualifies inbound, anchors and counters, chases unpaid invoices, and bumps stalled threads. A tested contact governor decides whether a thread may be touched at all (five bumps maximum, one revival per brand per six months). Drafts only, never sends."}

// DealStore is the OS's own brand deal store (brandDeals.all() minus the
// seeded placeholders), newest edit first.
type DealStore interface {
	Deals(ctx context.Context) ([]BrandDeal, error)
}

// BrandDealData is the run's data, as in FounderOS v1. Today is absent on the
// empty run, as there.
type BrandDealData struct {
	Today                 string       `json:"today,omitempty"`
	DealCount             int          `json:"dealCount"`
	Actions               []DealAction `json:"actions"`
	AwaitingFromFounderos []string     `json:"awaitingFromFounderos"`
}

// BrandDealAgent is brand-deal-agent (lib/agents/brand-deal-agent.ts): the
// deal store supplies the pipeline, TriageDeals decides what is urgent, and
// the skill file's open questions say what Victor is still negotiating
// blind about. FounderOS v1 stopped drafting outreach when its AI Gateway was
// removed (2026-09-21), so the ranked actions are the output. It reads the
// bridge's own founderos_brand_deals (the ETL of the OS store), never Notion,
// and sends nothing.
type BrandDealAgent struct {
	Skill func() (string, error)
	Deals DealStore
	Now   func() time.Time
}

func (a *BrandDealAgent) Meta() agents.Meta { return metaBrandDealAgent }

func (a *BrandDealAgent) Run(ctx context.Context) (agents.Result, error) {
	if a.Skill == nil {
		return agents.Result{}, errors.New("brand deal skill loader not wired")
	}
	skill, err := a.Skill()
	if err != nil {
		return agents.Result{}, err
	}
	if a.Deals == nil {
		return agents.Result{OK: false, Summary: "Brand deals unknown: no deal store wired on the bridge"}, nil
	}
	deals, err := a.Deals.Deals(ctx)
	if err != nil {
		return agents.Result{OK: false, Summary: fmt.Sprintf("Brand deals unknown: the OS deal store could not be read (%v)", err)}, nil
	}
	missing := roster.OpenSkillQuestions(skill)
	if len(deals) == 0 {
		return agents.Result{
			OK:      true,
			Summary: "No brand deals in the OS yet, so there is nothing to work. Ingest a deal and the agent has something to do.",
			Data:    BrandDealData{Actions: []DealAction{}, AwaitingFromFounderos: missing},
		}, nil
	}
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}
	today := now().UTC().Format("2006-01-02")
	actions := TriageDeals(deals, today)
	return agents.Result{
		OK:      true,
		Summary: TriageSummary(actions, len(deals)),
		Data:    BrandDealData{Today: today, DealCount: len(deals), Actions: actions, AwaitingFromFounderos: missing},
	}, nil
}

// PgDealStore reads founderos_brand_deals in the personal workspace.
type PgDealStore struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgDealStore(pool *pgxpool.Pool, workspaceID string) *PgDealStore {
	return &PgDealStore{pool: pool, workspaceID: workspaceID}
}

// Deals is brandDeals.all() without the seeded rows: ORDER BY last_edited
// DESC, brand, with brand compared bytewise as SQLite's BINARY collation does.
func (s *PgDealStore) Deals(ctx context.Context) ([]BrandDeal, error) {
	if s.pool == nil || s.workspaceID == "" {
		return nil, fmt.Errorf("workspace %q is not resolved on the bridge (run founderos-bootstrap)", BrandDealWorkspace)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, brand, status, deal_value_usd, budget_usd, amount_agreed_usd, suggested_rate_usd,
		       paid_in_full, deadline, follow_up_date, notion_url, last_edited, seeded
		FROM founderos_brand_deals
		WHERE workspace_id = $1 AND NOT seeded
		ORDER BY last_edited DESC, brand COLLATE "C"`, s.workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BrandDeal{}
	for rows.Next() {
		var d BrandDeal
		if err := rows.Scan(&d.ID, &d.Brand, &d.Status, &d.DealValueUSD, &d.BudgetUSD, &d.AmountAgreedUSD, &d.SuggestedRateUSD,
			&d.PaidInFull, &d.Deadline, &d.FollowUpDate, &d.NotionURL, &d.LastEdited, &d.Seeded); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
