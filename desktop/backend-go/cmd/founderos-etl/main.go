// Command founderos-etl copies FounderOS v1's SQLite databases into the founderos_*
// tables of a BusinessOS Postgres database (spec criterion 3.3). Source files
// are opened read-only; the load is idempotent (mirror upsert + delete), so a
// re-run reports zero changes.
//
//	founderos-etl --from <dir> [--database-url postgres://...] \
//	    [--workspace-map personal=<slug-or-uuid>,...] [--vault-user <user.id>] \
//	    [--report-dir .] [--report-name founderos-etl-report]
//
// <dir> holds founderos-os.db (required) and optionally bank.db, ledger.db,
// paykit.db and the slack-bridge state.db (<dir>/state.db or
// <dir>/slack-bridge/state.db). The database URL defaults to $DATABASE_URL.
// Proposal access codes are encrypted with $TOKEN_ENCRYPTION_KEY, the
// backend's key.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/etl"
)

func main() {
	from := flag.String("from", "", "directory holding founderos-os.db and the optional side databases (required)")
	dbURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "target BusinessOS Postgres URL (default $DATABASE_URL)")
	wsMap := flag.String("workspace-map", "", "override slug resolution: founderos-slug=businessos-slug-or-uuid,...")
	vaultUser := flag.String("vault-user", "", "BusinessOS user.id owning the access-code vault row (default: founderos workspace owner)")
	reportDir := flag.String("report-dir", ".", "where to write the JSON + markdown count report")
	reportName := flag.String("report-name", "founderos-etl-report", "report file base name")
	flag.Parse()
	if *from == "" || *dbURL == "" {
		flag.Usage()
		os.Exit(2)
	}
	mapping, err := etl.ParseWorkspaceMap(*wsMap)
	if err != nil {
		fail(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(*dbURL)
	if err != nil {
		fail(fmt.Errorf("database url: %w", err))
	}
	label := fmt.Sprintf("%s:%d/%s", cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Database)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fail(err)
	}
	defer pool.Close()

	rep, runErr := etl.Run(ctx, pool, etl.Options{
		From: *from, WorkspaceMap: mapping, VaultUserID: *vaultUser,
		EncryptionKey: os.Getenv("TOKEN_ENCRYPTION_KEY"), DatabaseLabel: label,
	})
	if rep != nil {
		jp, mp, err := rep.Write(*reportDir, *reportName)
		if err != nil {
			fail(err)
		}
		fmt.Printf("report: %s\n        %s\n", jp, mp)
		fmt.Printf("tables: %d, changes: %d, ok: %v\n", len(rep.Tables), rep.Changes(), rep.OK)
	}
	if runErr != nil {
		fail(runErr)
	}
	if !rep.OK {
		fail(fmt.Errorf("counts do not match: %v", rep.Problems))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "founderos-etl:", err)
	os.Exit(1)
}
