package paykit

import (
	"sort"
	"strconv"
	"sync"
)

// Source marks how a snapshot came to exist.
type Source string

const (
	// SourceLive was read from the API.
	SourceLive Source = "live"
	// SourceReconstructed was derived from an aggregate recorded elsewhere and
	// never measured per customer. Kept distinct so an inference is never read
	// as a measurement.
	SourceReconstructed Source = "reconstructed"
)

// Snapshot is one day's /customers pull.
type Snapshot struct {
	// CapturedOn is the UTC date of the pull, YYYY-MM-DD. One per day: a later
	// pull the same day replaces the earlier one.
	CapturedOn string     `json:"capturedOn"`
	Source     Source     `json:"source"`
	Customers  []Customer `json:"customers"`
}

// HistoryPort keeps daily snapshots (FounderOS v1's PaykitHistoryPort).
// Record must replace the whole day, and must skip customers with no id.
type HistoryPort interface {
	Record(Snapshot) error
	Snapshots() ([]Snapshot, error)
}

// MonthsSpanned lists every YYYY-MM the inclusive date range touches.
func MonthsSpanned(fromDate, toDate string) []string {
	y, m := atoi(slice(fromDate, 0, 4)), atoi(slice(fromDate, 5, 7))
	end := slice(toDate, 0, 7)
	var months []string
	for guard := 0; guard < 1200; guard++ {
		cur := strconv.Itoa(y) + "-" + pad2(m)
		months = append(months, cur)
		if cur >= end {
			break
		}
		if m++; m > 12 {
			y, m = y+1, 1
		}
	}
	return months
}

// WindowDelta is the money proven to have moved between two consecutive
// snapshots. ExactCents is money whose month is certain; UpperCents adds
// money that moved in the window but cannot be pinned to one month in it.
type WindowDelta struct {
	ExactCents         map[string]int64 `json:"exactCents"`
	UpperCents         map[string]int64 `json:"upperCents"`
	AmbiguousCustomers int              `json:"ambiguousCustomers"`
}

// DiffSnapshots differences two snapshots. Attribution is exact when the
// window lies in one calendar month, or when exactly one transaction was added
// (its date is the customer's latest). Several transactions across a month
// boundary raise the ceiling of every candidate month and the floor of none.
// A customer absent from prev is all-new money. A decrease (a refund) is
// reported in neither figure: the month it reverses is unknowable.
func DiffSnapshots(prev, curr Snapshot) WindowDelta {
	before := make(map[string]Customer, len(prev.Customers))
	for _, c := range prev.Customers {
		before[c.ID] = c
	}
	window := MonthsSpanned(prev.CapturedOn, curr.CapturedOn)
	d := WindowDelta{ExactCents: map[string]int64{}, UpperCents: map[string]int64{}}
	for _, c := range curr.Customers {
		was := before[c.ID] // zero value when absent: all-new money
		deltaCents := c.TotalSpentCents - was.TotalSpentCents
		deltaTxns := c.Transactions - was.Transactions
		if deltaCents <= 0 {
			continue
		}
		if len(window) == 1 {
			d.ExactCents[window[0]] += deltaCents
			d.UpperCents[window[0]] += deltaCents
			continue
		}
		if deltaTxns == 1 && c.LastTransactionDate != nil && *c.LastTransactionDate != "" {
			dated := slice(*c.LastTransactionDate, 0, 7)
			if contains(window, dated) {
				d.ExactCents[dated] += deltaCents
				d.UpperCents[dated] += deltaCents
				continue
			}
		}
		d.AmbiguousCustomers++
		for _, m := range window {
			d.UpperCents[m] += deltaCents
		}
	}
	return d
}

// MonthFromSnapshots computes a month by differencing a history, or nil when
// the history cannot answer: fewer than two snapshots, or none strictly before
// the month's 1st (money early in the month would be invisible, and the
// figure would understate without saying so). The caller falls back to the
// band.
func MonthFromSnapshots(snapshots []Snapshot, month string) *MonthIncome {
	ordered := append([]Snapshot(nil), snapshots...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].CapturedOn < ordered[j].CapturedOn })
	if len(ordered) < 2 || ordered[0].CapturedOn >= month+"-01" {
		return nil
	}
	var out MonthIncome
	for i := 1; i < len(ordered); i++ {
		d := DiffSnapshots(ordered[i-1], ordered[i])
		out.ExactCents += d.ExactCents[month]
		upper := d.UpperCents[month]
		out.UpperCents += upper
		if d.AmbiguousCustomers > 0 && upper > d.ExactCents[month] {
			out.UnsplittableCustomers += d.AmbiguousCustomers
		}
	}
	return &out
}

// ---- paykit.db rows ---------------------------------------------------

// SnapshotRow is one row of paykit.db's paykit_customer_snapshots:
// PRIMARY KEY (account, captured_on, customer_id). No PII.
type SnapshotRow struct {
	Account             string  `json:"account"`
	CapturedOn          string  `json:"captured_on"`
	CustomerID          string  `json:"customer_id"`
	TotalSpentCents     int64   `json:"total_spent_cents"`
	TotalTransactions   int     `json:"total_transactions"`
	LastTransactionDate *string `json:"last_transaction_date"`
	Source              string  `json:"source"`
}

// SnapshotRows produces the rows a snapshot persists as. Customers without an
// id are skipped: they cannot be joined across snapshots, and storing them
// would collide onto one key and fabricate deltas. A writer must replace the
// (account, captured_on) day wholesale, so a customer who vanished from the
// API does not survive as a stale row.
func SnapshotRows(account string, s Snapshot) []SnapshotRow {
	rows := make([]SnapshotRow, 0, len(s.Customers))
	for _, c := range s.Customers {
		if c.ID == "" {
			continue
		}
		rows = append(rows, SnapshotRow{
			Account:             account,
			CapturedOn:          s.CapturedOn,
			CustomerID:          c.ID,
			TotalSpentCents:     c.TotalSpentCents,
			TotalTransactions:   c.Transactions,
			LastTransactionDate: c.LastTransactionDate,
			Source:              string(s.Source),
		})
	}
	return rows
}

// SnapshotsFromRows regroups one account's rows into snapshots, oldest day
// first. Any source other than "reconstructed" reads as live.
func SnapshotsFromRows(rows []SnapshotRow) []Snapshot {
	sorted := append([]SnapshotRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].CapturedOn < sorted[j].CapturedOn })
	var out []Snapshot
	idx := map[string]int{}
	for _, r := range sorted {
		i, ok := idx[r.CapturedOn]
		if !ok {
			src := SourceLive
			if r.Source == string(SourceReconstructed) {
				src = SourceReconstructed
			}
			out = append(out, Snapshot{CapturedOn: r.CapturedOn, Source: src, Customers: []Customer{}})
			i = len(out) - 1
			idx[r.CapturedOn] = i
		}
		c := Customer{ID: r.CustomerID, TotalSpentCents: r.TotalSpentCents, Transactions: r.TotalTransactions, LastTransactionDate: r.LastTransactionDate}
		if r.LastTransactionDate != nil {
			m := slice(*r.LastTransactionDate, 0, 7)
			c.Month = &m
		}
		out[i].Customers = append(out[i].Customers, c)
	}
	return out
}

// SeedRows returns the reconstructed 2026-08-20 rows to install for an
// account, or nil. Only paykit-lc is seeded, and only when that date is
// absent, so a later real capture is never overwritten by an inference.
func SeedRows(account string, capturedDates []string) []SnapshotRow {
	if account != LaunchpadCohort.ID || contains(capturedDates, Seed20260820.CapturedOn) {
		return nil
	}
	return SnapshotRows(account, Seed20260820)
}

// ---- in-memory history --------------------------------------------------

// MemoryHistory is a HistoryPort with paykit.db's semantics (day replaced
// wholesale, id-less customers skipped, LC seeded once) held in memory. It is
// the reference the persisted store must behave like, and a usable store
// until persistence lands.
type MemoryHistory struct {
	account string
	mu      sync.Mutex
	rows    []SnapshotRow
}

func NewMemoryHistory(account string) *MemoryHistory {
	h := &MemoryHistory{account: account}
	h.rows = append(h.rows, SeedRows(account, nil)...)
	return h
}

func (h *MemoryHistory) Record(s Snapshot) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	kept := h.rows[:0:0]
	for _, r := range h.rows {
		if r.CapturedOn != s.CapturedOn {
			kept = append(kept, r)
		}
	}
	h.rows = append(kept, SnapshotRows(h.account, s)...)
	return nil
}

func (h *MemoryHistory) Snapshots() ([]Snapshot, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return SnapshotsFromRows(h.rows), nil
}

// CapturedDates lists the distinct capture dates held, oldest first.
func (h *MemoryHistory) CapturedDates() []string {
	snaps, _ := h.Snapshots()
	dates := make([]string, 0, len(snaps))
	for _, s := range snaps {
		dates = append(dates, s.CapturedOn)
	}
	return dates
}

// ---- small helpers ------------------------------------------------------

func slice(s string, from, to int) string {
	if from > len(s) {
		return ""
	}
	if to > len(s) {
		to = len(s)
	}
	return s[from:to]
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func pad2(n int) string {
	if n < 10 && n >= 0 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
