package finances

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/paykit"
)

// LedgerRow is a categorized statement row filed under a card lane.
type LedgerRow struct {
	ParsedRow
	Category string `json:"category"`
	Card     string `json:"card"`
}

// LedgerHash is the upload dedupe key, verbatim from FounderOS v1:
// <card>|<date>|<desc>|<cents>|<dir>.
func LedgerHash(r LedgerRow) string {
	return NormalizeCardID(r.Card) + "|" + r.Date + "|" + r.Description + "|" + strconv.FormatInt(r.AmountCents, 10) + "|" + r.Direction
}

// SortSpendRows orders a ledger oldest first (FounderOS v1 allRows()).
func SortSpendRows(rows []SpendRow) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
}

// Store is where statements land. Reads return an error when the store is
// unreachable, never an empty list.
type Store interface {
	InsertLedgerRows(ctx context.Context, rows []LedgerRow) (int, error)
	LedgerRows(ctx context.Context) ([]SpendRow, error)
	UpsertBankSummary(ctx context.Context, s BankSummary) error
	BankSummaries(ctx context.Context) ([]BankSummary, error)
}

// PgStore keeps statements in founderos_ledger_rows / founderos_bank_summaries
// and PayKit history in founderos_paykit_customer_snapshots, each row in the
// workspace the table map's per-row rule names.
type PgStore struct {
	Pool *pgxpool.Pool

	mu sync.Mutex
	ws map[string]string
}

var financeSlugs = []string{"launchpad-cohort", "vantage", "personal"}

// workspaces resolves the three workspaces finance rows live in.
func (s *PgStore) workspaces(ctx context.Context) (map[string]string, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("finances store: no database")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil {
		return s.ws, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT slug, id::text FROM workspaces WHERE slug = ANY($1)`, financeSlugs)
	if err != nil {
		return nil, fmt.Errorf("finances store: %w", err)
	}
	defer rows.Close()
	ws := map[string]string{}
	for rows.Next() {
		var slug, id string
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, err
		}
		ws[slug] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, slug := range financeSlugs {
		if ws[slug] == "" {
			return nil, fmt.Errorf("finances store: workspace %q is missing (run founderos-bootstrap)", slug)
		}
	}
	s.ws = ws
	return ws, nil
}

func ids(ws map[string]string) []string {
	out := make([]string, 0, len(ws))
	for _, id := range ws {
		out = append(out, id)
	}
	return out
}

func (s *PgStore) InsertLedgerRows(ctx context.Context, rows []LedgerRow) (int, error) {
	ws, err := s.workspaces(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	inserted := 0
	for _, r := range rows {
		card := NormalizeCardID(r.Card)
		r.Card = card
		tag, err := tx.Exec(ctx, `INSERT INTO founderos_ledger_rows (hash, workspace_id, date, description, amount_cents, direction, category, card)
			VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8) ON CONFLICT (hash) DO NOTHING`,
			LedgerHash(r), ws[CardWorkspace(card)], r.Date, r.Description, r.AmountCents, r.Direction, r.Category, card)
		if err != nil {
			return 0, fmt.Errorf("ledger insert: %w", err)
		}
		inserted += int(tag.RowsAffected())
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (s *PgStore) LedgerRows(ctx context.Context) ([]SpendRow, error) {
	ws, err := s.workspaces(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT to_char(date, 'YYYY-MM-DD'), description, amount_cents, direction, category, card
		FROM founderos_ledger_rows WHERE workspace_id::text = ANY($1) ORDER BY date ASC, hash ASC`, ids(ws))
	if err != nil {
		return nil, fmt.Errorf("ledger read: %w", err)
	}
	defer rows.Close()
	out := []SpendRow{}
	for rows.Next() {
		var r SpendRow
		if err := rows.Scan(&r.Date, &r.Description, &r.AmountCents, &r.Direction, &r.Category, &r.Card); err != nil {
			return nil, err
		}
		r.Card = NormalizeCardID(r.Card)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *PgStore) UpsertBankSummary(ctx context.Context, b BankSummary) error {
	ws, err := s.workspaces(ctx)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO founderos_bank_summaries (account, month, workspace_id, business, credits_cents, debits_cents, net_cents)
		VALUES ($1, ($2 || '-01')::date, $3, $4, $5, $6, $7)
		ON CONFLICT (account, month) DO UPDATE SET workspace_id = excluded.workspace_id, business = excluded.business,
			credits_cents = excluded.credits_cents, debits_cents = excluded.debits_cents, net_cents = excluded.net_cents, imported_at = now()`,
		b.Account, b.Month, ws[BankWorkspace(b.Business)], b.Business, b.CreditsCents, b.DebitsCents, b.CreditsCents-b.DebitsCents)
	if err != nil {
		return fmt.Errorf("bank upsert: %w", err)
	}
	return nil
}

func (s *PgStore) BankSummaries(ctx context.Context) ([]BankSummary, error) {
	ws, err := s.workspaces(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT account, business, to_char(month, 'YYYY-MM'), credits_cents, debits_cents, net_cents
		FROM founderos_bank_summaries WHERE workspace_id::text = ANY($1) ORDER BY month ASC, business ASC`, ids(ws))
	if err != nil {
		return nil, fmt.Errorf("bank read: %w", err)
	}
	defer rows.Close()
	out := []BankSummary{}
	for rows.Next() {
		var b BankSummary
		if err := rows.Scan(&b.Account, &b.Business, &b.Month, &b.CreditsCents, &b.DebitsCents, &b.NetCents); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ---- PayKit snapshot history (lib/paykit-history.ts) -------------------

// PgPaykitHistory is paykit.HistoryPort over
// founderos_paykit_customer_snapshots. Each day is replaced wholesale; the
// reconstructed 2026-08-20 datapoint is merged into reads when the table does
// not hold it, so a read never writes.
type PgPaykitHistory struct {
	Store   *PgStore
	Account string
}

func (h *PgPaykitHistory) workspace(ctx context.Context) (string, error) {
	ws, err := h.Store.workspaces(ctx)
	if err != nil {
		return "", err
	}
	if h.Account != paykit.LaunchpadCohort.ID && h.Account != paykit.Vantage.ID {
		return "", fmt.Errorf("paykit history: unknown account %q", h.Account)
	}
	if h.Account == paykit.Vantage.ID {
		return ws["vantage"], nil
	}
	return ws["launchpad-cohort"], nil
}

func (h *PgPaykitHistory) Record(s paykit.Snapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, err := h.workspace(ctx)
	if err != nil {
		return err
	}
	tx, err := h.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `DELETE FROM founderos_paykit_customer_snapshots WHERE account = $1 AND captured_on = $2::date`, h.Account, s.CapturedOn); err != nil {
		return err
	}
	for _, r := range paykit.SnapshotRows(h.Account, s) {
		// The raw text PayKit sent, as FounderOS v1 stores it (migration 163).
		last := r.LastTransactionDate
		if _, err := tx.Exec(ctx, `INSERT INTO founderos_paykit_customer_snapshots
			(account, captured_on, customer_id, workspace_id, total_spent_cents, total_transactions, last_transaction_date, source)
			VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8)`,
			r.Account, r.CapturedOn, r.CustomerID, ws, r.TotalSpentCents, r.TotalTransactions, last, r.Source); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (h *PgPaykitHistory) Snapshots() ([]paykit.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := h.workspace(ctx); err != nil {
		return nil, err
	}
	rows, err := h.Store.Pool.Query(ctx, `SELECT to_char(captured_on, 'YYYY-MM-DD'), customer_id, total_spent_cents, total_transactions, last_transaction_date, source
		FROM founderos_paykit_customer_snapshots WHERE account = $1`, h.Account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []paykit.SnapshotRow
	dates := map[string]bool{}
	for rows.Next() {
		r := paykit.SnapshotRow{Account: h.Account}
		if err := rows.Scan(&r.CapturedOn, &r.CustomerID, &r.TotalSpentCents, &r.TotalTransactions, &r.LastTransactionDate, &r.Source); err != nil {
			return nil, err
		}
		dates[r.CapturedOn] = true
		out = append(out, r)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	captured := make([]string, 0, len(dates))
	for d := range dates {
		captured = append(captured, d)
	}
	out = append(out, paykit.SeedRows(h.Account, captured)...)
	return paykit.SnapshotsFromRows(out), nil
}
