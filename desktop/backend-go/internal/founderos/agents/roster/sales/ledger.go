package sales

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerWorkspace is where the ingest and archive ledgers live (table map:
// plaud_ingests → founderos).
const LedgerWorkspace = "founderos"

// ---- founderos_plaud_ingests ------------------------------------------------------

// PgPlaudLedger is the Plaud ingest ledger in founderos_plaud_ingests. Rows are
// keyed by Plaud file id (the table's PK), so a recording prod ingested before
// the ETL is never filed twice.
type PgPlaudLedger struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgPlaudLedger(pool *pgxpool.Pool, workspaceID string) *PgPlaudLedger {
	return &PgPlaudLedger{pool: pool, workspaceID: workspaceID}
}

var errNoLedgerWorkspace = fmt.Errorf("workspace %q is not resolved (Deps.Workspaces), so the ledger has no home", LedgerWorkspace)

// Ready proves a via='oe' row can be written, inside a transaction that is
// always rolled back. Migration 160's via CHECK still lists only the two
// historic values (5.4 widens it); until then the ingest refuses to capture anything rather than
// land pages it cannot record.
func (l *PgPlaudLedger) Ready(ctx context.Context) error {
	if l.workspaceID == "" {
		return errNoLedgerWorkspace
	}
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("plaud ingest ledger: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO founderos_plaud_ingests (file_id, workspace_id, title, ingested_at, via, slug)
		VALUES ('__bridge_probe__', $1, 'probe', now(), $2, 'probe')`, l.workspaceID, ViaOE)
	if err != nil {
		return fmt.Errorf("founderos_plaud_ingests cannot record a bridge ingest (via=%q): %w; its via CHECK must allow %q (spec 5.4)", ViaOE, err, ViaOE)
	}
	return nil
}

func (l *PgPlaudLedger) Ingested(ctx context.Context) (map[string]bool, error) {
	rows, err := l.pool.Query(ctx, `SELECT file_id FROM founderos_plaud_ingests`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (l *PgPlaudLedger) Insert(ctx context.Context, r PlaudIngest) error {
	if l.workspaceID == "" {
		return errNoLedgerWorkspace
	}
	var recorded *time.Time
	if t, err := time.Parse(time.RFC3339Nano, r.RecordedAt); err == nil {
		recorded = &t
	}
	_, err := l.pool.Exec(ctx, `INSERT INTO founderos_plaud_ingests
		(file_id, workspace_id, title, recorded_at, ingested_at, via, slug, claims)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (file_id) DO NOTHING`,
		r.FileID, l.workspaceID, r.Title, recorded, r.IngestedAt, r.Via, r.Slug, r.Claims)
	return err
}

// ---- call archive ledger (founderos_call_archive, migration 161) -------------

const archiveSource = "fathom"

// PgArchiveLedger keeps one founderos_call_archive row per archived call, with
// the full ArchiveEntry in its entry column.
type PgArchiveLedger struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgArchiveLedger(pool *pgxpool.Pool, workspaceID string) *PgArchiveLedger {
	return &PgArchiveLedger{pool: pool, workspaceID: workspaceID}
}

func (l *PgArchiveLedger) Archived(ctx context.Context) (map[string]ArchiveEntry, error) {
	rows, err := l.pool.Query(ctx, `SELECT external_id, entry FROM founderos_call_archive WHERE source = $1`, archiveSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ArchiveEntry{}
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var e ArchiveEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, fmt.Errorf("call archive ledger row %s: %w", id, err)
		}
		e.RecordingID = id
		out[id] = e
	}
	return out, rows.Err()
}

func (l *PgArchiveLedger) Record(ctx context.Context, e ArchiveEntry) error {
	if l.workspaceID == "" {
		return errNoLedgerWorkspace
	}
	if e.RecordingID == "" {
		return errors.New("call archive ledger: empty recording id")
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	var signal *string
	if e.SignalID != "" {
		signal = &e.SignalID
	}
	_, err = l.pool.Exec(ctx, `INSERT INTO founderos_call_archive (source, external_id, workspace_id, slug, signal_id, entry, archived_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (source, external_id) DO UPDATE SET slug = EXCLUDED.slug, signal_id = EXCLUDED.signal_id, entry = EXCLUDED.entry, archived_at = EXCLUDED.archived_at`,
		archiveSource, e.RecordingID, l.workspaceID, e.Page, signal, raw, e.ArchivedAt)
	return err
}
