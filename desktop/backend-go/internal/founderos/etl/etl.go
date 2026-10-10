package etl

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options configures one ETL run.
type Options struct {
	// From is the directory holding founderos-os.db and the optional side files.
	From string
	// WorkspaceMap overrides how a FounderOS slug resolves: the value is a
	// BusinessOS workspace slug or UUID. Unmapped slugs resolve to themselves.
	WorkspaceMap map[string]string
	// VaultUserID owns the credential_vault row for proposal access codes.
	// Default: the owner of the founderos workspace.
	VaultUserID string
	// EncryptionKey is the backend's TOKEN_ENCRYPTION_KEY (base64, 32 bytes).
	// Required only when there are access codes to move.
	EncryptionKey string
	// DatabaseLabel names the target in the report (never a password).
	DatabaseLabel string
	Now           func() time.Time
}

// runState carries cross-table facts through the transform phase.
type runState struct {
	contactWS       map[string]string
	proposalWS      map[string]string
	accessCodes     map[string]string
	sourceAutopilot *bool
	pendingNulled   int
}

type workspaceRef struct {
	id    string
	owner string
}

type rowOut struct {
	vals map[string]any
	ws   string
}

type planned struct {
	spec *tableSpec
	tr   *TableReport
	rows []rowOut
}

// Run executes the ETL: preflight, transform everything in memory, then load
// each source file's tables in one transaction with mirror semantics, then
// move proposal access codes into credential_vault.
func Run(ctx context.Context, pool *pgxpool.Pool, opts Options) (*Report, error) {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	files, err := locateSources(opts.From)
	if err != nil {
		return nil, err
	}
	srcs := map[string]*sourceDB{}
	defer func() {
		for _, s := range srcs {
			s.Close()
		}
	}()
	for name, p := range files {
		s, err := openSource(p)
		if err != nil {
			return nil, err
		}
		srcs[name] = s
	}

	rep := &Report{
		GeneratedAt: now().UTC(), From: opts.From, Database: opts.DatabaseLabel,
		SourceFiles: files, Workspaces: map[string]string{},
	}

	// Preflight 1: every workspace slug resolves.
	ws, err := resolveWorkspaces(ctx, pool, opts.WorkspaceMap)
	if err != nil {
		return nil, err
	}
	owned := make([]string, 0, len(ws))
	for slug, w := range ws {
		rep.Workspaces[slug] = w.id
		owned = append(owned, w.id)
	}
	sort.Strings(owned)

	specs := Specs()
	// Preflight 2: the target tables exist (migration 160).
	var missing []string
	for _, sp := range specs {
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+sp.target).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			missing = append(missing, sp.target)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("target tables missing (%s); apply migrations/160_founderos_tables.sql first", strings.Join(missing, ", "))
	}

	// Preflight 3: state.db holds only Slack rows.
	if st, ok := srcs[FileState]; ok {
		rep.StateDBOther = map[string]int{}
		for tbl := range st.tables {
			if tbl == "slack_bridge_jobs" || tbl == "slack_bridge_sessions" || strings.HasPrefix(tbl, "sqlite_") {
				continue
			}
			n, err := st.count(tbl)
			if err != nil {
				return nil, err
			}
			rep.StateDBOther[tbl] = n
			if n > 0 {
				return nil, fmt.Errorf("state.db table %s has %d rows; only slack_bridge_* are expected there", tbl, n)
			}
		}
	}

	// Dropped tables are counted, never loaded.
	for _, tbl := range DroppedTables {
		n := 0
		if srcs[FileOS].tables[tbl] {
			if n, err = srcs[FileOS].count(tbl); err != nil {
				return nil, err
			}
		}
		rep.Dropped = append(rep.Dropped, DroppedReport{Table: tbl, SourceRows: n})
	}

	// Transform phase: nothing is written until every row has converted.
	st := &runState{contactWS: map[string]string{}, proposalWS: map[string]string{}, accessCodes: map[string]string{}}
	var plans []*planned
	var rowErrs []string
	for _, sp := range specs {
		p, errs, err := transformTable(sp, srcs, ws, st)
		if err != nil {
			return nil, err
		}
		rowErrs = append(rowErrs, errs...)
		plans = append(plans, p)
		rep.Tables = append(rep.Tables, p.tr)
	}
	if len(rowErrs) > 0 {
		if len(rowErrs) > 20 {
			rowErrs = append(rowErrs[:20], fmt.Sprintf("... and %d more", len(rowErrs)-20))
		}
		return rep, fmt.Errorf("%d rows failed to convert; nothing was written:\n  %s", len(rowErrs), strings.Join(rowErrs, "\n  "))
	}

	// Load phase: one transaction per source file group.
	var groups []string
	byGroup := map[string][]*planned{}
	for _, p := range plans {
		if p.tr.Skipped != "" {
			continue
		}
		if _, ok := byGroup[p.spec.group]; !ok {
			groups = append(groups, p.spec.group)
		}
		byGroup[p.spec.group] = append(byGroup[p.spec.group], p)
	}
	for _, g := range groups {
		if err := loadGroup(ctx, pool, byGroup[g], owned); err != nil {
			return rep, fmt.Errorf("load %s: %w", g, err)
		}
	}

	// Target counts and the verdict.
	rep.OK = true
	for _, p := range plans {
		if p.tr.Skipped != "" {
			continue
		}
		if err := pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s WHERE workspace_id::text = ANY($1::text[])`, p.spec.target), owned).Scan(&p.tr.TargetRows); err != nil {
			return rep, err
		}
		p.tr.Match = p.tr.TargetRows == p.tr.SourceRows && p.tr.KeyCollisions == 0
		if !p.tr.Match {
			rep.OK = false
			rep.Problems = append(rep.Problems, fmt.Sprintf("%s: source %d rows, target %d rows, %d key collisions",
				p.spec.target, p.tr.SourceRows, p.tr.TargetRows, p.tr.KeyCollisions))
		}
	}

	rep.TradingLimits = TradingLimitsReport{
		SourceAutopilot: st.sourceAutopilot, ImportedAutopilot: false,
		Note: "Forced OFF on import: the bridge never inherits a live trading switch; flipping it is a deliberate act on the bridge's /trading card.",
	}
	if tl := rep.Table("founderos_trading_limits"); tl != nil {
		rep.TradingLimits.SourceRows = tl.SourceRows
	}
	rep.PendingNulled = st.pendingNulled

	// Secrets last, in their own transaction.
	proposalsLoaded := rep.Table("founderos_proposals") != nil && rep.Table("founderos_proposals").Skipped == ""
	vr, err := moveAccessCodes(ctx, pool, opts, ws, st.accessCodes, proposalsLoaded)
	rep.Vault = vr
	if err != nil {
		rep.OK = false
		return rep, err
	}
	return rep, nil
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func resolveWorkspaces(ctx context.Context, pool *pgxpool.Pool, override map[string]string) (map[string]workspaceRef, error) {
	out := map[string]workspaceRef{}
	var missing []string
	for _, slug := range Slugs {
		want := slug
		if v, ok := override[slug]; ok && v != "" {
			want = v
		}
		q := `SELECT id::text, owner_id FROM workspaces WHERE slug = $1`
		if uuidRE.MatchString(want) {
			q = `SELECT id::text, owner_id FROM workspaces WHERE id::text = lower($1)`
		}
		var w workspaceRef
		err := pool.QueryRow(ctx, q, want).Scan(&w.id, &w.owner)
		if errors.Is(err, pgx.ErrNoRows) {
			if want == slug {
				missing = append(missing, slug)
			} else {
				missing = append(missing, fmt.Sprintf("%s (mapped to %s)", slug, want))
			}
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("look up workspace %s: %w", want, err)
		}
		out[slug] = w
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("BusinessOS workspaces missing: %s. Create them with cmd/founderos-bootstrap (criterion 2.2) or pass --workspace-map; the ETL never creates workspaces",
			strings.Join(missing, ", "))
	}
	return out, nil
}

// transformTable reads and converts one target table's rows. Row-level
// conversion failures are returned as messages (the run then fails before any
// write); structural problems are errors.
func transformTable(sp *tableSpec, srcs map[string]*sourceDB, ws map[string]workspaceRef, st *runState) (*planned, []string, error) {
	tr := &TableReport{Target: sp.target}
	p := &planned{spec: sp, tr: tr}
	var present []srcRef
	var absent []string
	for _, ref := range sp.sources {
		s, ok := srcs[ref.file]
		if !ok {
			absent = append(absent, ref.file+" absent")
			continue
		}
		if !s.tables[ref.table] {
			absent = append(absent, ref.file+" has no "+ref.table)
			continue
		}
		present = append(present, ref)
	}
	if len(present) == 0 {
		tr.Skipped = strings.Join(absent, "; ")
		return p, nil, nil
	}

	order := []string{}
	merged := map[string]rowOut{}
	var errs []string
	cs := &convStats{}
	defaulted := map[string]bool{}
	for _, ref := range present {
		s := srcs[ref.file]
		have, err := s.columns(ref.table)
		if err != nil {
			return nil, nil, err
		}
		have[rowidCol] = true
		for _, cl := range sp.cols {
			if cl.src != "" && !have[cl.src] && !cl.hasDef {
				return nil, nil, fmt.Errorf("%s:%s lacks column %s and the map gives it no default", ref.file, ref.table, cl.src)
			}
		}
		rows, err := s.readAll(ref.table)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s:%s: %w", ref.file, ref.table, err)
		}
		tr.Sources = append(tr.Sources, SourceCount{File: ref.file, Table: ref.table, Rows: len(rows)})
		seen := map[string]bool{}
		for _, raw := range rows {
			out := map[string]any{}
			var rowErr error
			for _, cl := range sp.cols {
				if cl.src == "" {
					continue
				}
				v, ok := raw[cl.src]
				if !have[cl.src] {
					v, ok = cl.def, true
					defaulted[cl.src] = true
				}
				if !ok {
					rowErr = fmt.Errorf("missing %s", cl.src)
					break
				}
				cv, err := convert(v, cl.k, cs)
				if err != nil {
					rowErr = fmt.Errorf("%s: %w", cl.src, err)
					break
				}
				out[cl.dst] = cv
			}
			if rowErr == nil && sp.prep != nil {
				rowErr = sp.prep(raw, out, st, tr)
			}
			slug := sp.ws
			if rowErr == nil && sp.wsFn != nil {
				slug, rowErr = sp.wsFn(raw, out, st)
			}
			if rowErr != nil {
				errs = append(errs, fmt.Sprintf("%s:%s rowid %v: %v", ref.file, ref.table, raw[rowidCol], rowErr))
				continue
			}
			w := ws[slug]
			out["workspace_id"] = w.id
			keyVals := make([]any, len(sp.key))
			for i, k := range sp.key {
				keyVals[i] = out[k]
			}
			ks := keyString(keyVals)
			if seen[ks] {
				tr.KeyCollisions++
			}
			seen[ks] = true
			if _, dup := merged[ks]; !dup {
				order = append(order, ks)
			}
			merged[ks] = rowOut{vals: out, ws: slug}
		}
	}
	for _, ks := range order {
		p.rows = append(p.rows, merged[ks])
	}
	tr.SourceRows = len(p.rows)
	tr.D2DateOnly, tr.D6BlankToNull, tr.NoOffsetAsUTC = cs.d2, cs.d6, cs.noOffset
	for k := range defaulted {
		tr.DefaultedColumn = append(tr.DefaultedColumn, k)
	}
	sort.Strings(tr.DefaultedColumn)
	return p, errs, nil
}

// targetCols returns the insert column list: the mapped columns plus
// workspace_id.
func targetCols(sp *tableSpec) ([]string, []string) {
	var names, types []string
	for _, cl := range sp.cols {
		names = append(names, cl.dst)
		types = append(types, cl.k.pgType())
	}
	names = append(names, "workspace_id")
	types = append(types, "uuid")
	return names, types
}

func upsertSQL(sp *tableSpec) (string, []string) {
	names, types := targetCols(sp)
	params := make([]string, len(names))
	for i, t := range types {
		if t == "uuid" {
			params[i] = fmt.Sprintf("$%d::text::uuid", i+1)
			continue
		}
		params[i] = fmt.Sprintf("$%d::%s", i+1, t)
	}
	isKey := map[string]bool{}
	for _, k := range sp.key {
		isKey[k] = true
	}
	var sets, oldT, newT []string
	for _, n := range names {
		if isKey[n] {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", n, n))
		oldT = append(oldT, "t."+n)
		newT = append(newT, "EXCLUDED."+n)
	}
	sets = append(sets, "imported_at = now()")
	changed := fmt.Sprintf("(%s) IS DISTINCT FROM (%s)", strings.Join(oldT, ", "), strings.Join(newT, ", "))
	if len(oldT) == 1 {
		changed = fmt.Sprintf("%s IS DISTINCT FROM %s", oldT[0], newT[0])
	}
	where := changed
	if sp.newerOnly != "" {
		where = fmt.Sprintf("EXCLUDED.%[1]s > t.%[1]s OR (EXCLUDED.%[1]s = t.%[1]s AND %[2]s)", sp.newerOnly, changed)
	}
	q := fmt.Sprintf(`INSERT INTO %s AS t (%s) VALUES (%s)
ON CONFLICT (%s) DO UPDATE SET %s
WHERE %s
RETURNING (xmax = 0)`, sp.target, strings.Join(names, ", "), strings.Join(params, ", "),
		strings.Join(sp.key, ", "), strings.Join(sets, ", "), where)
	return q, names
}

// deleteSQL removes target rows in the ETL-owned workspaces whose key is not
// in the source (mirror mode).
func deleteSQL(sp *tableSpec) string {
	var arrays, aliases, conds []string
	for i, k := range sp.key {
		typ := "text"
		cmp := fmt.Sprintf("d.%s = k.c%d", k, i)
		if k == "workspace_id" {
			cmp = fmt.Sprintf("d.workspace_id::text = k.c%d", i)
		} else if cl, ok := sp.colByDst(k); ok {
			typ = cl.k.pgType()
		}
		arrays = append(arrays, fmt.Sprintf("$%d::%s[]", i+2, typ))
		aliases = append(aliases, fmt.Sprintf("c%d", i))
		conds = append(conds, cmp)
	}
	return fmt.Sprintf(`DELETE FROM %s AS d WHERE d.workspace_id::text = ANY($1::text[])
AND NOT EXISTS (SELECT 1 FROM unnest(%s) AS k(%s) WHERE %s)`,
		sp.target, strings.Join(arrays, ", "), strings.Join(aliases, ", "), strings.Join(conds, " AND "))
}

// keyArrays builds one typed array per key column for deleteSQL.
func keyArrays(sp *tableSpec, rows []rowOut) []any {
	out := make([]any, len(sp.key))
	for i, k := range sp.key {
		cl, _ := sp.colByDst(k)
		switch {
		case k == "workspace_id" || cl.k == kText || cl.k == kTextN:
			a := make([]string, 0, len(rows))
			for _, r := range rows {
				a = append(a, fmt.Sprint(r.vals[k]))
			}
			out[i] = a
		case cl.k == kTS || cl.k == kDate || cl.k == kMonth:
			a := make([]time.Time, 0, len(rows))
			for _, r := range rows {
				a = append(a, r.vals[k].(time.Time))
			}
			out[i] = a
		default:
			a := make([]int64, 0, len(rows))
			for _, r := range rows {
				a = append(a, r.vals[k].(int64))
			}
			out[i] = a
		}
	}
	return out
}

func loadGroup(ctx context.Context, pool *pgxpool.Pool, plans []*planned, owned []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	for _, p := range plans {
		q, names := upsertSQL(p.spec)
		batch := &pgx.Batch{}
		for _, r := range p.rows {
			args := make([]any, len(names))
			for i, n := range names {
				args[i] = r.vals[n]
			}
			batch.Queue(q, args...)
		}
		br := tx.SendBatch(ctx, batch)
		for range p.rows {
			rows, err := br.Query()
			if err != nil {
				br.Close()
				return fmt.Errorf("upsert %s: %w", p.spec.target, err)
			}
			wrote := false
			for rows.Next() {
				var inserted bool
				if err := rows.Scan(&inserted); err != nil {
					rows.Close()
					br.Close()
					return err
				}
				wrote = true
				if inserted {
					p.tr.Inserted++
				} else {
					p.tr.Updated++
				}
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				br.Close()
				return fmt.Errorf("upsert %s: %w", p.spec.target, err)
			}
			if !wrote {
				p.tr.Unchanged++
			}
		}
		if err := br.Close(); err != nil {
			return fmt.Errorf("upsert %s: %w", p.spec.target, err)
		}
	}

	// Deletes run children first so a removed parent never strands a row.
	for i := len(plans) - 1; i >= 0; i-- {
		p := plans[i]
		args := append([]any{owned}, keyArrays(p.spec, p.rows)...)
		tag, err := tx.Exec(ctx, deleteSQL(p.spec), args...)
		if err != nil {
			return fmt.Errorf("mirror delete %s: %w", p.spec.target, err)
		}
		p.tr.Deleted = int(tag.RowsAffected())
	}
	return tx.Commit(ctx)
}
