// Package branddeals ports FounderOS v1's Notion connector: the "Brand Deals
// Hub" data source read into /brand-deals (lib/connectors/brand-deals.ts) and
// the Notion status check (the retired lib/connectors/notion.ts, whose id,
// name and kind this keeps).
//
// Notion stays the source of truth; the operator's brand-deal agents and his
// friend's guest access both live there. The data-source query is a POST but
// a read, so it is allowlisted in ReadPOSTs. Page create/update are outbound
// writes behind guard.Outbound.
package branddeals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
)

// Meta is the Notion connector as FounderOS v1 last registered it. Notion was
// taken off the Connections board on 2026-08-19 while this read stayed alive
// for /brand-deals, so it does not count toward /api/connections.
var Meta = connectors.Meta{ID: "notion", Name: "Notion", Kind: connectors.KindNotion}

// ReadPOSTs lets the data-source query through the write guard.
var ReadPOSTs = []string{"api.notion.com/v1/data_sources/"}

const (
	defaultBaseURL = "https://api.notion.com/v1"
	// @notionhq/client v5's default; data sources need 2025-09-03 or later.
	notionVersion = "2025-09-03"
	// The DATA SOURCE id of "Brand Deals the operator", the collection shared with
	// the live integration. A wrong id fails quietly into seeded examples.
	DefaultSourceID = "3bd9c1cb-baa1-8171-9b0d-000bb2d72175"
	// Outlives the 15-minute sweep that warms it.
	CacheTTL = 20 * time.Minute
)

// PipelineOrder is the Status lanes in the operator's Notion board order.
var PipelineOrder = []string{
	"New", "Negotiating", "Aligned", "Researching", "Filming", "Editing",
	"Delivered", "Invoiced", "Approved", "Paid", "Declined", "Paused",
}

// Deal mirrors BrandDealSchema; nil pointers are the schema's nulls.
type Deal struct {
	ID               string   `json:"id"`
	Brand            string   `json:"brand"`
	Status           string   `json:"status"`
	Tier             *string  `json:"tier"`
	DealValueUSD     *float64 `json:"dealValueUsd"`
	BudgetUSD        *float64 `json:"budgetUsd"`
	AmountAgreedUSD  *float64 `json:"amountAgreedUsd"`
	SuggestedRateUSD *float64 `json:"suggestedRateUsd"`
	PaidInFull       bool     `json:"paidInFull"`
	Deadline         *string  `json:"deadline"`
	FollowUpDate     *string  `json:"followUpDate"`
	ContactName      *string  `json:"contactName"`
	ContactEmail     *string  `json:"contactEmail"`
	MainChannel      *string  `json:"mainChannel"`
	VideoType        *string  `json:"videoType"`
	Source           *string  `json:"source"`
	ICPFit           *string  `json:"icpFit"`
	NotionURL        string   `json:"notionUrl"`
	LastEdited       string   `json:"lastEdited"`
	Seeded           bool     `json:"seeded"`
}

type Lane struct {
	Status string `json:"status"`
	Deals  []Deal `json:"deals"`
}

type Mode string

const (
	ModeLive   Mode = "live"
	ModeSeeded Mode = "seeded"
	ModeError  Mode = "error"
)

// Result is what /brand-deals renders. SyncedAt is epoch milliseconds of the
// read the deals came from, nil when nothing synced.
type Result struct {
	Deals    []Deal `json:"deals"`
	Mode     Mode   `json:"mode"`
	Detail   string `json:"detail"`
	SyncedAt *int64 `json:"syncedAt"`
}

type Connector struct {
	res     connectors.Resolver
	baseURL string
	http    *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache *dealCache
}

type dealCache struct {
	at    time.Time
	deals []Deal
}

func New(res connectors.Resolver) *Connector {
	return &Connector{
		res:     res,
		baseURL: defaultBaseURL,
		http:    connectors.HTTPClient(20*time.Second, ReadPOSTs...),
		now:     time.Now,
	}
}

func (c *Connector) key() string { return c.res.Resolve("NOTION_API_KEY") }

// SourceID is resolved per call so a source planted while running takes
// effect at once; an empty override falls back to the default.
func (c *Connector) SourceID() string {
	if v := c.res.Resolve("NOTION_BRAND_DEALS_SOURCE"); v != "" {
		return v
	}
	return DefaultSourceID
}

// InvalidateCache drops the cache so a manual refresh sees fresh state.
func (c *Connector) InvalidateCache() {
	c.mu.Lock()
	c.cache = nil
	c.mu.Unlock()
}

func (c *Connector) st(state connectors.State, detail string) connectors.Status {
	return connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind, State: state, Detail: detail}
}

// Status calls users/me. It never writes.
func (c *Connector) Status(ctx context.Context) connectors.Status {
	key := c.key()
	if key == "" {
		return c.st(connectors.StateNotConfigured,
			"Set NOTION_API_KEY (internal integration secret) in ~/.founderos/.env or under API keys, and share target pages with it.")
	}
	var me struct {
		Name *string `json:"name"`
	}
	if err := c.call(ctx, key, http.MethodGet, "/users/me", nil, &me); err != nil {
		return c.st(connectors.StateError, "Key set but auth failed: "+err.Error())
	}
	name := "integration"
	if me.Name != nil {
		name = *me.Name
	}
	return c.st(connectors.StateConnected, "Connected as "+name)
}

// FetchDeals reads every deal in the data source. Without a key it returns
// the seeded examples, marked. A live read is cached for 20 minutes, and only
// a non-empty one, so an outage retries instead of pinning an empty board;
// when a fetch fails after a good read, the last good board is served and
// said to be stale.
func (c *Connector) FetchDeals(ctx context.Context) Result {
	key := c.key()
	if key == "" {
		return Result{
			Deals:  SeededDeals(),
			Mode:   ModeSeeded,
			Detail: "Showing examples. Plant NOTION_API_KEY (internal integration secret, shared with Brand Deals Hub) to go live.",
		}
	}
	c.mu.Lock()
	if dc := c.cache; dc != nil && c.now().Sub(dc.at) < CacheTTL {
		c.mu.Unlock()
		return live(dc, fmt.Sprintf("Brand Deals Hub · %d deals", len(dc.deals)))
	}
	c.mu.Unlock()

	deals, err := c.queryAll(ctx, key)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if c.cache != nil {
			return live(c.cache, "Notion unreachable · showing the last good read")
		}
		return Result{
			Deals:  []Deal{},
			Mode:   ModeError,
			Detail: fmt.Sprintf("Key set but the fetch failed: %s. Is the integration shared with Brand Deals Hub?", err),
		}
	}
	at := c.now()
	if len(deals) > 0 {
		c.cache = &dealCache{at: at, deals: deals}
	}
	ms := at.UnixMilli()
	return Result{Deals: append([]Deal{}, deals...), Mode: ModeLive, Detail: fmt.Sprintf("Brand Deals Hub · %d deals", len(deals)), SyncedAt: &ms}
}

func live(dc *dealCache, detail string) Result {
	ms := dc.at.UnixMilli()
	return Result{Deals: append([]Deal{}, dc.deals...), Mode: ModeLive, Detail: detail, SyncedAt: &ms}
}

func (c *Connector) queryAll(ctx context.Context, key string) ([]Deal, error) {
	deals := []Deal{}
	path := "/data_sources/" + url.PathEscape(c.SourceID()) + "/query"
	cursor := ""
	for {
		body := map[string]any{"page_size": 100}
		if cursor != "" {
			body["start_cursor"] = cursor
		}
		var page struct {
			Results    []json.RawMessage `json:"results"`
			HasMore    bool              `json:"has_more"`
			NextCursor *string           `json:"next_cursor"`
		}
		if err := c.call(ctx, key, http.MethodPost, path, body, &page); err != nil {
			return nil, err
		}
		for _, raw := range page.Results {
			if d, ok := ParseDeal(raw); ok {
				deals = append(deals, d)
			}
		}
		if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
			return deals, nil
		}
		cursor = *page.NextCursor
	}
}

// ---- writes (guarded) -----------------------------------------------------

// UpdatePage patches a deal page's properties (Notion property-value shape).
func (c *Connector) UpdatePage(ctx context.Context, pageID string, properties map[string]any) error {
	return guard.Outbound("notion.pages.update", func() error {
		key := c.key()
		if key == "" {
			return fmt.Errorf("NOTION_API_KEY is not set")
		}
		if err := c.call(ctx, key, http.MethodPatch, "/pages/"+url.PathEscape(pageID), map[string]any{"properties": properties}, nil); err != nil {
			return err
		}
		c.InvalidateCache()
		return nil
	})
}

// CreatePage adds a deal to the brand-deals data source and returns its id.
func (c *Connector) CreatePage(ctx context.Context, properties map[string]any) (string, error) {
	var id string
	err := guard.Outbound("notion.pages.create", func() error {
		key := c.key()
		if key == "" {
			return fmt.Errorf("NOTION_API_KEY is not set")
		}
		body := map[string]any{
			"parent":     map[string]any{"type": "data_source_id", "data_source_id": c.SourceID()},
			"properties": properties,
		}
		var out struct {
			ID string `json:"id"`
		}
		if err := c.call(ctx, key, http.MethodPost, "/pages", body, &out); err != nil {
			return err
		}
		id = out.ID
		c.InvalidateCache()
		return nil
	})
	return id, err
}

// ---- plumbing -------------------------------------------------------------

// call sends one Notion request. A non-2xx answer becomes an error carrying
// Notion's own message, as @notionhq/client's APIResponseError does.
func (c *Connector) call(ctx context.Context, key, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Notion-Version", notionVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var e struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &e) == nil && e.Message != "" {
			return fmt.Errorf("%s", e.Message)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// ---- parsing --------------------------------------------------------------

type richText struct {
	PlainText string `json:"plain_text"`
}

type named struct {
	Name *string `json:"name"`
}

type prop struct {
	Title    []richText `json:"title"`
	RichText []richText `json:"rich_text"`
	Select   *named     `json:"select"`
	Status   *named     `json:"status"`
	Number   *float64   `json:"number"`
	Checkbox *bool      `json:"checkbox"`
	Email    *string    `json:"email"`
	Date     *struct {
		Start *string `json:"start"`
	} `json:"date"`
}

func (p *prop) text() *string {
	if p == nil {
		return nil
	}
	parts := p.Title
	if parts == nil {
		parts = p.RichText
	}
	var b strings.Builder
	for _, t := range parts {
		b.WriteString(t.PlainText)
	}
	return nonEmpty(strings.TrimSpace(b.String()))
}

func (p *prop) sel() *string {
	if p == nil {
		return nil
	}
	if p.Select != nil && p.Select.Name != nil {
		return p.Select.Name
	}
	if p.Status != nil && p.Status.Name != nil {
		return p.Status.Name
	}
	return nil
}

func (p *prop) num() *float64 {
	if p == nil {
		return nil
	}
	return p.Number
}

func (p *prop) date() *string {
	if p == nil || p.Date == nil {
		return nil
	}
	return p.Date.Start
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ParseDeal maps one Notion page to a deal; ok is false when it has no brand
// name. Missing properties read null rather than dropping the row.
func ParseDeal(raw []byte) (Deal, bool) {
	var pg struct {
		ID             *string          `json:"id"`
		URL            *string          `json:"url"`
		LastEditedTime *string          `json:"last_edited_time"`
		Properties     map[string]*prop `json:"properties"`
	}
	if json.Unmarshal(raw, &pg) != nil {
		return Deal{}, false
	}
	p := pg.Properties
	brand := p["Brand Name"].text()
	if brand == nil {
		return Deal{}, false
	}
	id := *brand
	if pg.ID != nil {
		id = *pg.ID
	}
	status := "New"
	if s := p["Status"].sel(); s != nil && *s != "" {
		status = *s
	}
	notionURL := ""
	if pg.URL != nil {
		notionURL = *pg.URL
	} else {
		pid := ""
		if pg.ID != nil {
			pid = *pg.ID
		}
		notionURL = "https://www.notion.so/" + strings.ReplaceAll(pid, "-", "")
	}
	lastEdited := time.UnixMilli(0).UTC().Format("2006-01-02T15:04:05.000Z")
	if pg.LastEditedTime != nil {
		lastEdited = *pg.LastEditedTime
	}
	var contactEmail *string
	if ce := p["Contact Email"]; ce != nil {
		contactEmail = ce.Email
	}
	paid := false
	if pf := p["Paid in Full"]; pf != nil && pf.Checkbox != nil {
		paid = *pf.Checkbox
	}
	return Deal{
		ID:               id,
		Brand:            *brand,
		Status:           status,
		Tier:             p["Brand Tier"].sel(),
		DealValueUSD:     p["Deal Value"].num(),
		BudgetUSD:        p["Budget"].num(),
		AmountAgreedUSD:  p["Amount Agreed"].num(),
		SuggestedRateUSD: p["Suggested Rate"].num(),
		PaidInFull:       paid,
		Deadline:         p["Deadline"].date(),
		FollowUpDate:     p["Follow-up Date"].date(),
		ContactName:      p["Contact Name"].text(),
		ContactEmail:     contactEmail,
		MainChannel:      p["Main Channel"].sel(),
		VideoType:        p["Video Type"].sel(),
		Source:           p["Source"].sel(),
		ICPFit:           p["ICP Fit"].sel(),
		NotionURL:        notionURL,
		LastEdited:       lastEdited,
	}, true
}

// GroupDeals returns board lanes in pipeline order; an unknown status gets
// its own trailing lane rather than vanishing.
func GroupDeals(deals []Deal) []Lane {
	lanes := make([]Lane, 0, len(PipelineOrder))
	index := map[string]int{}
	for _, s := range PipelineOrder {
		index[s] = len(lanes)
		lanes = append(lanes, Lane{Status: s, Deals: []Deal{}})
	}
	for _, d := range deals {
		i, ok := index[d.Status]
		if !ok {
			i = len(lanes)
			index[d.Status] = i
			lanes = append(lanes, Lane{Status: d.Status, Deals: []Deal{}})
		}
		lanes[i].Deals = append(lanes[i].Deals, d)
	}
	return lanes
}

func sp(s string) *string   { return &s }
func fp(f float64) *float64 { return &f }

// SeededDeals are the placeholder rows shown before the key lands, each
// marked Seeded so the page badges itself honestly.
func SeededDeals() []Deal {
	return []Deal{
		{ID: "seed-1", Brand: "Notion (example)", Status: "Negotiating", Tier: sp("S"),
			DealValueUSD: fp(6000), BudgetUSD: fp(8000), SuggestedRateUSD: fp(6500),
			Deadline: sp("2026-09-05"), FollowUpDate: sp("2026-08-20"),
			ContactName: sp("Sam Rivera"), ContactEmail: sp("sam@example.com"), MainChannel: sp("Instagram"),
			VideoType: sp("Integration"), Source: sp("Inbound"), ICPFit: sp("Strong"),
			NotionURL: "https://www.notion.so/", LastEdited: "2026-08-13T15:00:00.000Z", Seeded: true},
		{ID: "seed-2", Brand: "Framer (example)", Status: "New", Tier: sp("A"),
			SuggestedRateUSD: fp(4000), FollowUpDate: sp("2026-08-18"), MainChannel: sp("TikTok"),
			VideoType: sp("Short-form"), Source: sp("Outbound"), ICPFit: sp("Decent"),
			NotionURL: "https://www.notion.so/", LastEdited: "2026-08-12T10:00:00.000Z", Seeded: true},
		{ID: "seed-3", Brand: "Shopify (example)", Status: "Filming", Tier: sp("S"),
			DealValueUSD: fp(9000), BudgetUSD: fp(9000), AmountAgreedUSD: fp(9000), SuggestedRateUSD: fp(8500),
			Deadline: sp("2026-08-28"), ContactName: sp("Dana K"), ContactEmail: sp("dana@example.com"),
			MainChannel: sp("YouTube"), VideoType: sp("Dedicated"), Source: sp("Agency"), ICPFit: sp("Strong"),
			NotionURL: "https://www.notion.so/", LastEdited: "2026-08-11T09:00:00.000Z", Seeded: true},
		{ID: "seed-4", Brand: "Riverside (example)", Status: "Paid", Tier: sp("B"),
			DealValueUSD: fp(2500), BudgetUSD: fp(2500), AmountAgreedUSD: fp(2500), SuggestedRateUSD: fp(2500),
			PaidInFull: true, MainChannel: sp("Podcast"), VideoType: sp("Mention"), Source: sp("Referral"), ICPFit: sp("Decent"),
			NotionURL: "https://www.notion.so/", LastEdited: "2026-07-30T16:00:00.000Z", Seeded: true},
	}
}
