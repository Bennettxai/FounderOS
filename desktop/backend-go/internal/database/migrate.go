// Package database migration runner.
//
// Migrations were historically applied by hand against the cloud DB with no
// ledger, which let files drift (a deploy 500'd because tables 143/144 were
// never applied). This runner embeds the migration SQL into the binary,
// records applied versions in schema_migrations, and applies pending files on
// boot BEFORE the schema-health check — so drift can no longer happen silently.
package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationFS embeds every migration so the runner works inside the deployed
// container (the migrations/ directory is not shipped as loose files).
//
//go:embed migrations/*.sql
var migrationFS embed.FS

//go:embed schema.sql
var bootstrapSchema string

// baselineCutoff is the last historical migration that may be adopted without
// execution when an existing database predates schema_migrations. Repair
// migrations after this point must run because they reconcile schema objects
// that an older database may never have received.
const baselineCutoff = "157_communication_routes.sql"

// RunMigrations ensures the schema_migrations ledger exists, then applies every
// embedded migration whose filename is not yet recorded, in filename order,
// each in its own transaction. The version key is the full filename, so the
// repo's duplicate-numbered files (e.g. two 141_*.sql) are tracked distinctly.
//
// Baseline adoption: on an existing database that predates this ledger (empty
// schema_migrations but the core `workspaces` table already present), all
// current migrations are recorded as applied WITHOUT re-running them. This
// avoids re-executing non-idempotent historical migrations against live data.
// Only genuinely new migrations added after adoption run normally.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var hasCore bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.workspaces') IS NOT NULL`).Scan(&hasCore); err != nil {
		return fmt.Errorf("bootstrap probe: %w", err)
	}
	if !hasCore {
		if _, err := pool.Exec(ctx, bootstrapSchema); err != nil {
			return fmt.Errorf("bootstrap fresh database: %w", err)
		}
		slog.Info("migrations: bootstrapped fresh database from canonical schema")
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	// Baseline: adopt historical migrations in an already-provisioned DB without
	// re-running them, then continue so post-baseline repair migrations execute.
	if len(applied) == 0 {
		if err := pool.QueryRow(ctx, `SELECT to_regclass('public.workspaces') IS NOT NULL`).Scan(&hasCore); err != nil {
			return fmt.Errorf("baseline probe: %w", err)
		}
		if hasCore {
			recorded := 0
			for _, f := range files {
				if !shouldBaselineMigration(f) {
					continue
				}
				if _, err := pool.Exec(ctx,
					`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, f); err != nil {
					return fmt.Errorf("baseline record %s: %w", f, err)
				}
				applied[f] = true
				recorded++
			}
			slog.Info("migrations: baselined existing database", "recorded", recorded, "cutoff", baselineCutoff)
		}
	}

	appliedNow := 0
	for _, f := range files {
		if applied[f] {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", f, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", f, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, f); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", f, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", f, err)
		}
		appliedNow++
		slog.Info("migrations: applied", "version", f)
	}

	slog.Info("migrations: up to date", "applied_now", appliedNow, "total", len(files))
	return nil
}

func shouldBaselineMigration(filename string) bool {
	return filename <= baselineCutoff
}

// migrationFiles returns the embedded *.sql filenames sorted in apply order.
// Filenames are zero-padded numeric prefixes, so lexical sort == numeric order.
func migrationFiles() ([]string, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	return files, nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}
