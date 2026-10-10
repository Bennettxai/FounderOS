package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceCloudSync pushes a workspace and its scoped data from the LOCAL
// Postgres to the shared CLOUD Postgres (Supabase). This is the local-first →
// cloud "sync" (not a one-time migrate): it upserts by primary key, so it is
// safe to re-run. Only the owner's machine has CLOUD_DATABASE_URL, so only the
// owner can sync up; teammates reach the data through the cloud backend API,
// gated by workspace membership + role.
//
// User IDs differ between local and cloud (better-auth issues random IDs per
// install), so every user reference is remapped by email. Members and invitees
// must already have a cloud account to be linked.
type WorkspaceCloudSync struct {
	local *pgxpool.Pool
}

func NewWorkspaceCloudSync(local *pgxpool.Pool) *WorkspaceCloudSync {
	return &WorkspaceCloudSync{local: local}
}

// CloudDatabaseURL is the cloud DB key; only the owner's machine has it.
func CloudDatabaseURL() string { return os.Getenv("CLOUD_DATABASE_URL") }

type SyncReport struct {
	Workspace    string         `json:"workspace"`
	Members      int            `json:"members"`
	Invites      int            `json:"invites"`
	Tables       map[string]int `json:"tables"`
	SkippedUsers []string       `json:"skipped_users"` // emails with no cloud account
}

// Scoped data tables synced to the cloud (workspace/members/invites handled
// explicitly first). User-reference columns are remapped local→cloud.
var syncDataTables = []string{
	"contexts", "calendar_events", "glossary_terms", "rhythm_entries",
	"campaigns", "offers", "personas", "content_items", "sites",
	"entity_links", "projects", "tasks", "clients", "workspace_preferences",
}

var userRefColumns = map[string]bool{
	"created_by": true, "user_id": true, "invited_by": true, "owner_id": true,
}

func (s *WorkspaceCloudSync) Sync(ctx context.Context, workspaceID string) (*SyncReport, error) {
	cloudURL := CloudDatabaseURL()
	if cloudURL == "" {
		return nil, fmt.Errorf("CLOUD_DATABASE_URL not set on this machine (only the owner can sync up)")
	}
	cfg, err := pgxpool.ParseConfig(cloudURL)
	if err != nil {
		return nil, fmt.Errorf("parse cloud url: %w", err)
	}
	// Supabase transaction pooler (:6543) doesn't support prepared statements.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cloud, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect cloud: %w", err)
	}
	defer cloud.Close()

	rep := &SyncReport{Tables: map[string]int{}}

	// local user_id -> email for everyone referenced by this workspace
	localIDToEmail, err := s.localUserEmails(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("local user emails: %w", err)
	}
	// email -> cloud user_id
	emailToCloud, err := cloudUserIDsByEmail(ctx, cloud, mapValues(localIDToEmail))
	if err != nil {
		return nil, fmt.Errorf("cloud user map: %w", err)
	}
	// local user_id -> cloud user_id, plus the set of valid cloud IDs
	localToCloud := map[string]string{}
	validCloud := map[string]bool{}
	for lid, email := range localIDToEmail {
		if cid, ok := emailToCloud[email]; ok {
			localToCloud[lid] = cid
			validCloud[cid] = true
		} else {
			rep.SkippedUsers = append(rep.SkippedUsers, email)
		}
	}
	remap := func(v interface{}) interface{} {
		if sv, ok := v.(string); ok {
			if cid, ok := localToCloud[sv]; ok {
				return cid
			}
		}
		return v
	}

	// 1. workspace row (remap owner_id)
	if _, err := s.upsert(ctx, cloud, "workspaces", "id = $1", []interface{}{workspaceID}, remap, nil); err != nil {
		return nil, fmt.Errorf("upsert workspace: %w", err)
	}
	rep.Workspace = workspaceID

	// 2. members (only those with a cloud account; remap user_id + invited_by)
	rep.Members, err = s.upsert(ctx, cloud, "workspace_members", "workspace_id = $1", []interface{}{workspaceID}, remap, validCloud)
	if err != nil {
		return nil, fmt.Errorf("sync members: %w", err)
	}

	// 3. invites (keep token so existing emailed links resolve; remap invited_by)
	rep.Invites, err = s.upsert(ctx, cloud, "workspace_invites", "workspace_id = $1", []interface{}{workspaceID}, remap, nil)
	if err != nil {
		return nil, fmt.Errorf("sync invites: %w", err)
	}

	// 4. scoped data tables (best-effort)
	for _, t := range syncDataTables {
		n, err := s.upsert(ctx, cloud, t, "workspace_id = $1", []interface{}{workspaceID}, remap, nil)
		if err != nil {
			slog.Warn("[CloudSync] table skipped", "table", t, "error", err)
			continue
		}
		if n > 0 {
			rep.Tables[t] = n
		}
	}

	slog.Info("[CloudSync] synced workspace to cloud", "workspace", workspaceID, "members", rep.Members, "invites", rep.Invites)
	return rep, nil
}

// upsert copies rows matching `where` from local to cloud, remapping user-ref
// columns. When dropIfUserUnmapped is non-nil, rows whose user_id is not a
// valid cloud user are skipped (used for members so we never write a dangling
// membership). Returns the number of rows written.
func (s *WorkspaceCloudSync) upsert(ctx context.Context, cloud *pgxpool.Pool, table, where string, args []interface{}, remap func(interface{}) interface{}, dropIfUserUnmapped map[string]bool) (int, error) {
	rows, err := s.local.Query(ctx, fmt.Sprintf("SELECT * FROM %s WHERE %s", table, where), args...)
	if err != nil {
		return 0, err
	}
	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	userIdx := -1
	for i, fd := range fds {
		cols[i] = string(fd.Name)
		if cols[i] == "user_id" {
			userIdx = i
		}
	}

	var batch [][]interface{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			rows.Close()
			return 0, err
		}
		for i, c := range cols {
			if userRefColumns[c] {
				vals[i] = remap(vals[i])
			}
		}
		batch = append(batch, vals)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, vals := range batch {
		if dropIfUserUnmapped != nil && userIdx >= 0 {
			cid, ok := vals[userIdx].(string)
			if !ok || !dropIfUserUnmapped[cid] {
				continue // member with no cloud account - skip
			}
		}
		if err := upsertValues(ctx, cloud, table, cols, vals); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *WorkspaceCloudSync) localUserEmails(ctx context.Context, wsID string) (map[string]string, error) {
	out := map[string]string{}
	rows, err := s.local.Query(ctx, `
		SELECT DISTINCT u.id, u.email
		FROM "user" u
		WHERE u.id IN (
			SELECT user_id FROM workspace_members WHERE workspace_id = $1
			UNION SELECT owner_id FROM workspaces WHERE id = $1
			UNION SELECT invited_by FROM workspace_invites WHERE workspace_id = $1
		)`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[id] = email
	}
	return out, rows.Err()
}

func cloudUserIDsByEmail(ctx context.Context, cloud *pgxpool.Pool, emails []string) (map[string]string, error) {
	out := map[string]string{}
	if len(emails) == 0 {
		return out, nil
	}
	rows, err := cloud.Query(ctx, `SELECT id, email FROM "user" WHERE email = ANY($1)`, emails)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[email] = id
	}
	return out, rows.Err()
}

func upsertValues(ctx context.Context, cloud *pgxpool.Pool, table string, cols []string, vals []interface{}) error {
	ph := make([]string, len(cols))
	sets := make([]string, 0, len(cols))
	for i, c := range cols {
		ph[i] = fmt.Sprintf("$%d", i+1)
		if c != "id" {
			sets = append(sets, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
		}
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s",
		table, strings.Join(cols, ", "), strings.Join(ph, ", "), strings.Join(sets, ", "))
	_, err := cloud.Exec(ctx, sql, vals...)
	return err
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
