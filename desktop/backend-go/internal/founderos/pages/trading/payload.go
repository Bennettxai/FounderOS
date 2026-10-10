package trading

import (
	"context"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/phantom"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/robinhood"
)

// Store is the read side of founderos_trading_*. Every read returns an error
// when the store is unreachable, never an empty list.
type Store interface {
	// LatestSnapshots is the newest snapshot of every account, richest first.
	LatestSnapshots(ctx context.Context) ([]Snapshot, error)
	// History is one account's newest `limit` snapshots, oldest first.
	History(ctx context.Context, accountID string, limit int) ([]Snapshot, error)
	// Positions come from each account's latest snapshot, largest first.
	Positions(ctx context.Context) ([]Position, error)
	// Activity is the trade log, newest first.
	Activity(ctx context.Context, limit int) ([]Activity, error)
	LatestAnalysis(ctx context.Context, accountID string) (*Analysis, error)
	// OpenOrders is live broker state, replaced wholesale on each push.
	OpenOrders(ctx context.Context) ([]Order, error)
	// Limits is the stored row and its updated_at, or nil when never saved.
	Limits(ctx context.Context) (*Limits, string, error)
}

// WalletState says why the wallet is or is not on the board.
type WalletState struct {
	State  string `json:"state"`
	Detail string `json:"detail"`
}

// WalletRead is one read of the Phantom wallet: a balance, or why not.
type WalletRead struct {
	Wallet *Wallet
	State  string
	Detail string
}

// WalletFunc reads the wallet. Injected so tests never reach an RPC.
type WalletFunc func(ctx context.Context) WalletRead

// PhantomWallet reads the wallet through the Phantom connector. It fails
// soft, as FounderOS v1 phantomBalance() does: an unreachable RPC is a nil
// wallet with the reason, and a price outage keeps the balance.
func PhantomWallet(c *phantom.Connector) WalletFunc {
	return func(ctx context.Context) WalletRead {
		if c.Address() == "" {
			return WalletRead{State: string(connectors.StateNotConfigured), Detail: "No PHANTOM_WALLET_ADDRESS set. It is a public address, not a key."}
		}
		b, err := c.Balance(ctx)
		if err != nil {
			return WalletRead{State: string(connectors.StateError), Detail: "Solana RPC did not return a balance: " + err.Error()}
		}
		w := &Wallet{Address: b.Address, SOL: b.SOL, USDPerSOL: b.USDPerSOL, USDValue: b.USDValue, FetchedAt: b.FetchedAt}
		d := ShortAddress(b.Address) + " · " + fnum(b.SOL) + " SOL"
		if b.USDValue == nil {
			d += " (price unavailable)"
		}
		return WalletRead{Wallet: w, State: string(connectors.StateConnected), Detail: d}
	}
}

// View is the pure read of the payload the board draws (lib/trading-view.ts
// and agentSummary), computed once here so it is pinned by Go tests.
type View struct {
	Freshness Freshness    `json:"freshness"`
	Volume    Volume       `json:"volume"`
	Sizes     []Bucket     `json:"sizes"`
	Agent     AgentSummary `json:"agent"`
}

// Payload is FounderOS v1 TradingPayload, plus the wallet's status, the
// limits the agent runs under, and the computed view.
type Payload struct {
	Accounts      []Snapshot            `json:"accounts"`
	History       map[string][]Snapshot `json:"history"`
	Snapshot      *Snapshot             `json:"snapshot"`
	Positions     []Position            `json:"positions"`
	Activity      []Activity            `json:"activity"`
	Analysis      *Analysis             `json:"analysis"`
	OpenOrders    []Order               `json:"openOrders"`
	Status        connectors.Status     `json:"status"`
	Source        *string               `json:"source"`
	Phantom       *Wallet               `json:"phantom"`
	PhantomStatus WalletState           `json:"phantomStatus"`
	Limits        LimitsView            `json:"limits"`
	View          View                  `json:"view"`
	At            string                `json:"at"`
}

const (
	activityLimit = 50
	historyLimit  = 500
)

// Build reads everything the /trading slab draws. The wallet is read beside
// the store, and its failure never blanks the board.
func Build(ctx context.Context, st Store, wallet WalletFunc, now time.Time) (*Payload, error) {
	var (
		wg sync.WaitGroup
		wr WalletRead
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		wr = wallet(ctx)
	}()
	p, err := read(ctx, st)
	wg.Wait()
	if err != nil {
		return nil, err
	}

	p.Phantom = wr.Wallet
	p.PhantomStatus = WalletState{State: wr.State, Detail: wr.Detail}
	p.Status = robinhood.StatusFor(p.Accounts, now)
	if len(p.Accounts) > 0 {
		first := p.Accounts[0]
		p.Snapshot, p.Source = &first, &first.Source
	}
	p.At = now.UTC().Format("2006-01-02T15:04:05.000Z")

	var sleeve *Snapshot
	for i := range p.Accounts {
		if p.Accounts[i].AccountID == AgenticID {
			sleeve = &p.Accounts[i]
		}
	}
	var sleevePos []Position
	for _, x := range p.Positions {
		if x.AccountID == AgenticID {
			sleevePos = append(sleevePos, x)
		}
	}
	var sleeveTrades []Activity
	for _, a := range p.Activity {
		if a.AccountID == AgenticID {
			sleeveTrades = append(sleeveTrades, a)
		}
	}
	p.View = View{
		Freshness: FreshnessOf(p.Accounts, now),
		Volume:    VolumeOf(p.Accounts, p.Positions, p.Phantom),
		Sizes:     PositionSizes(p.Positions),
		Agent:     AgentSummaryOf(sleeve, sleevePos, sleeveTrades),
	}
	return p, nil
}

func read(ctx context.Context, st Store) (*Payload, error) {
	p := &Payload{}
	var err error
	if p.Accounts, err = st.LatestSnapshots(ctx); err != nil {
		return nil, err
	}
	p.History = map[string][]Snapshot{}
	for _, a := range p.Accounts {
		h, err := st.History(ctx, a.AccountID, historyLimit)
		if err != nil {
			return nil, err
		}
		p.History[a.AccountID] = orEmpty(h)
	}
	if p.Positions, err = st.Positions(ctx); err != nil {
		return nil, err
	}
	if p.Activity, err = st.Activity(ctx, activityLimit); err != nil {
		return nil, err
	}
	if p.Analysis, err = st.LatestAnalysis(ctx, AgenticID); err != nil {
		return nil, err
	}
	if p.OpenOrders, err = st.OpenOrders(ctx); err != nil {
		return nil, err
	}
	stored, updated, err := st.Limits(ctx)
	if err != nil {
		return nil, err
	}
	p.Limits = LimitsViewOf(stored, updated)
	p.Accounts, p.Positions, p.Activity, p.OpenOrders = orEmpty(p.Accounts), orEmpty(p.Positions), orEmpty(p.Activity), orEmpty(p.OpenOrders)
	return p, nil
}

func orEmpty[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}
