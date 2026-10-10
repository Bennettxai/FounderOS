package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AutoSyncProfileTarget is a lightweight view of a profile used by the background sync job.
type AutoSyncProfileTarget struct {
	UserID     string
	Name       string
	RootPath   string
	SyncCommit string
	SyncSource string
	SyncBranch string
}

// saveProfile saves the application profile to the database
func (s *AppProfilerService) saveProfile(ctx context.Context, profile *ApplicationProfile) error {
	techStackJSON, err := json.Marshal(profile.TechStack)
	if err != nil {
		return fmt.Errorf("marshal tech_stack: %w", err)
	}
	languagesJSON, err := json.Marshal(profile.Languages)
	if err != nil {
		return fmt.Errorf("marshal languages: %w", err)
	}
	structureJSON, err := json.Marshal(profile.StructureTree)
	if err != nil {
		return fmt.Errorf("marshal structure_tree: %w", err)
	}
	componentsJSON, err := json.Marshal(profile.Components)
	if err != nil {
		return fmt.Errorf("marshal components: %w", err)
	}
	modulesJSON, err := json.Marshal(profile.Modules)
	if err != nil {
		return fmt.Errorf("marshal modules: %w", err)
	}
	endpointsJSON, err := json.Marshal(profile.APIEndpoints)
	if err != nil {
		return fmt.Errorf("marshal api_endpoints: %w", err)
	}
	dbSchemaJSON, err := json.Marshal(profile.DatabaseSchema)
	if err != nil {
		return fmt.Errorf("marshal database_schema: %w", err)
	}
	conventionsJSON, err := json.Marshal(profile.Conventions)
	if err != nil {
		return fmt.Errorf("marshal conventions: %w", err)
	}
	integrationsJSON, err := json.Marshal(profile.IntegrationPoints)
	if err != nil {
		return fmt.Errorf("marshal integration_points: %w", err)
	}
	metadataJSON, err := json.Marshal(profile.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	frameworksJSON, err := json.Marshal(profile.Frameworks)
	if err != nil {
		return fmt.Errorf("marshal frameworks: %w", err)
	}

	// Try upsert first. Some environments may not have the unique constraint yet;
	// if ON CONFLICT fails, fall back to update-or-insert.
	_, err = s.pool.Exec(ctx,
		`INSERT INTO application_profiles
		 (id, user_id, name, description, root_path, app_type, tech_stack, languages, frameworks,
		  structure_tree, components, total_components, modules, total_modules, api_endpoints,
		  total_endpoints, database_schema, conventions, integration_points, readme_summary,
		  lines_of_code, file_count, last_analyzed_at, metadata, created_at, updated_at,
		  auto_sync_enabled, last_synced_at, sync_source, sync_branch, sync_commit)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31)
		 ON CONFLICT (user_id, name) DO UPDATE SET
		    description = EXCLUDED.description,
		    root_path = EXCLUDED.root_path,
		    app_type = EXCLUDED.app_type,
		    tech_stack = EXCLUDED.tech_stack,
		    languages = EXCLUDED.languages,
		    frameworks = EXCLUDED.frameworks,
		    structure_tree = EXCLUDED.structure_tree,
		    components = EXCLUDED.components,
		    total_components = EXCLUDED.total_components,
		    modules = EXCLUDED.modules,
		    total_modules = EXCLUDED.total_modules,
		    api_endpoints = EXCLUDED.api_endpoints,
		    total_endpoints = EXCLUDED.total_endpoints,
		    database_schema = EXCLUDED.database_schema,
		    conventions = EXCLUDED.conventions,
		    integration_points = EXCLUDED.integration_points,
		    readme_summary = EXCLUDED.readme_summary,
		    lines_of_code = EXCLUDED.lines_of_code,
		    file_count = EXCLUDED.file_count,
		    last_analyzed_at = EXCLUDED.last_analyzed_at,
		    metadata = EXCLUDED.metadata,
		    auto_sync_enabled = EXCLUDED.auto_sync_enabled,
		    last_synced_at = EXCLUDED.last_synced_at,
		    sync_source = EXCLUDED.sync_source,
		    sync_branch = EXCLUDED.sync_branch,
		    sync_commit = EXCLUDED.sync_commit,
		    updated_at = EXCLUDED.updated_at`,
		profile.ID, profile.UserID, profile.Name, profile.Description, profile.RootPath,
		string(profile.AppType), techStackJSON, languagesJSON, frameworksJSON, structureJSON,
		componentsJSON, profile.TotalComponents, modulesJSON, profile.TotalModules,
		endpointsJSON, profile.TotalEndpoints, dbSchemaJSON, conventionsJSON,
		integrationsJSON, profile.ReadmeSummary, profile.LinesOfCode, profile.FileCount,
		profile.LastAnalyzedAt, metadataJSON, profile.CreatedAt, profile.UpdatedAt,
		profile.AutoSyncEnabled, profile.LastSyncedAt, profile.SyncSource, profile.SyncBranch, profile.SyncCommit,
	)
	if err == nil {
		return nil
	}

	// Fallback path: update existing row first.
	cmdTag, updErr := s.pool.Exec(ctx,
		`UPDATE application_profiles
		 SET description=$3, root_path=$4, app_type=$5, tech_stack=$6, languages=$7, frameworks=$8,
		     structure_tree=$9, components=$10, total_components=$11, modules=$12, total_modules=$13,
		     api_endpoints=$14, total_endpoints=$15, database_schema=$16, conventions=$17,
		     integration_points=$18, readme_summary=$19, lines_of_code=$20, file_count=$21,
		     last_analyzed_at=$22, metadata=$23, auto_sync_enabled=$24, last_synced_at=$25,
		     sync_source=$26, sync_branch=$27, sync_commit=$28, updated_at=$29
		 WHERE user_id=$1 AND name=$2`,
		profile.UserID, profile.Name,
		profile.Description, profile.RootPath, string(profile.AppType), techStackJSON, languagesJSON, frameworksJSON,
		structureJSON, componentsJSON, profile.TotalComponents, modulesJSON, profile.TotalModules,
		endpointsJSON, profile.TotalEndpoints, dbSchemaJSON, conventionsJSON, integrationsJSON,
		profile.ReadmeSummary, profile.LinesOfCode, profile.FileCount, profile.LastAnalyzedAt,
		metadataJSON, profile.AutoSyncEnabled, profile.LastSyncedAt, profile.SyncSource, profile.SyncBranch, profile.SyncCommit,
		profile.UpdatedAt,
	)
	if updErr == nil && cmdTag.RowsAffected() > 0 {
		return nil
	}

	// If update didn't touch anything, insert without ON CONFLICT.
	_, insErr := s.pool.Exec(ctx,
		`INSERT INTO application_profiles
		 (id, user_id, name, description, root_path, app_type, tech_stack, languages, frameworks,
		  structure_tree, components, total_components, modules, total_modules, api_endpoints,
		  total_endpoints, database_schema, conventions, integration_points, readme_summary,
		  lines_of_code, file_count, last_analyzed_at, metadata, created_at, updated_at,
		  auto_sync_enabled, last_synced_at, sync_source, sync_branch, sync_commit)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31)`,
		profile.ID, profile.UserID, profile.Name, profile.Description, profile.RootPath,
		string(profile.AppType), techStackJSON, languagesJSON, frameworksJSON, structureJSON,
		componentsJSON, profile.TotalComponents, modulesJSON, profile.TotalModules,
		endpointsJSON, profile.TotalEndpoints, dbSchemaJSON, conventionsJSON,
		integrationsJSON, profile.ReadmeSummary, profile.LinesOfCode, profile.FileCount,
		profile.LastAnalyzedAt, metadataJSON, profile.CreatedAt, profile.UpdatedAt,
		profile.AutoSyncEnabled, profile.LastSyncedAt, profile.SyncSource, profile.SyncBranch, profile.SyncCommit,
	)
	if insErr != nil {
		// Return original upsert error as it's usually the most informative.
		return err
	}
	return nil
}

// GetProfile retrieves an application profile
func (s *AppProfilerService) GetProfile(ctx context.Context, userID, name string) (*ApplicationProfile, error) {
	var profile ApplicationProfile
	var techStackJSON, languagesJSON, structureJSON, componentsJSON []byte
	var modulesJSON, endpointsJSON, dbSchemaJSON, conventionsJSON []byte
	var integrationsJSON, metadataJSON, frameworksJSON []byte

	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, root_path, app_type, tech_stack, languages, frameworks,
		        structure_tree, components, total_components, modules, total_modules, api_endpoints,
		        total_endpoints, database_schema, conventions, integration_points, readme_summary,
		        lines_of_code, file_count, last_analyzed_at, metadata, created_at, updated_at
		 FROM application_profiles WHERE user_id = $1 AND name = $2`,
		userID, name).Scan(
		&profile.ID, &profile.UserID, &profile.Name, &profile.Description, &profile.RootPath,
		&profile.AppType, &techStackJSON, &languagesJSON, &frameworksJSON, &structureJSON,
		&componentsJSON, &profile.TotalComponents, &modulesJSON, &profile.TotalModules,
		&endpointsJSON, &profile.TotalEndpoints, &dbSchemaJSON, &conventionsJSON,
		&integrationsJSON, &profile.ReadmeSummary, &profile.LinesOfCode, &profile.FileCount,
		&profile.LastAnalyzedAt, &metadataJSON, &profile.CreatedAt, &profile.UpdatedAt)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(techStackJSON, &profile.TechStack)
	json.Unmarshal(languagesJSON, &profile.Languages)
	json.Unmarshal(frameworksJSON, &profile.Frameworks)
	json.Unmarshal(structureJSON, &profile.StructureTree)
	json.Unmarshal(componentsJSON, &profile.Components)
	json.Unmarshal(modulesJSON, &profile.Modules)
	json.Unmarshal(endpointsJSON, &profile.APIEndpoints)
	json.Unmarshal(dbSchemaJSON, &profile.DatabaseSchema)
	json.Unmarshal(conventionsJSON, &profile.Conventions)
	json.Unmarshal(integrationsJSON, &profile.IntegrationPoints)
	json.Unmarshal(metadataJSON, &profile.Metadata)

	return &profile, nil
}

// ListProfiles lists all profiles for a user
func (s *AppProfilerService) ListProfiles(ctx context.Context, userID string) ([]ApplicationProfile, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, description, app_type, lines_of_code, file_count, last_analyzed_at, created_at
		 FROM application_profiles WHERE user_id = $1 ORDER BY updated_at DESC`,
		userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make([]ApplicationProfile, 0)
	for rows.Next() {
		var p ApplicationProfile
		err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.AppType,
			&p.LinesOfCode, &p.FileCount, &p.LastAnalyzedAt, &p.CreatedAt)
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}

	return profiles, nil
}

// ListAutoSyncTargets returns lightweight sync targets for the background job.
func (s *AppProfilerService) ListAutoSyncTargets(ctx context.Context, limit int) ([]AutoSyncProfileTarget, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.pool.Query(ctx, `
		SELECT user_id, name, root_path, COALESCE(sync_commit, ''), COALESCE(sync_source, ''), COALESCE(sync_branch, '')
		FROM application_profiles
		WHERE auto_sync_enabled = true
		  AND root_path IS NOT NULL
		  AND root_path <> ''
		ORDER BY COALESCE(last_synced_at, last_analyzed_at, updated_at) ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []AutoSyncProfileTarget
	for rows.Next() {
		var t AutoSyncProfileTarget
		if err := rows.Scan(&t.UserID, &t.Name, &t.RootPath, &t.SyncCommit, &t.SyncSource, &t.SyncBranch); err != nil {
			continue
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// UpdateSyncInfo persists updated sync metadata for a profile.
func (s *AppProfilerService) UpdateSyncInfo(ctx context.Context, userID, name string, enabled bool, source, branch, commit string, syncedAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE application_profiles
		SET auto_sync_enabled = $3,
		    sync_source = $4,
		    sync_branch = $5,
		    sync_commit = $6,
		    last_synced_at = $7,
		    updated_at = NOW()
		WHERE user_id = $1 AND name = $2
	`, userID, name, enabled, source, branch, commit, syncedAt)
	return err
}

// detectGitInfo reads HEAD commit and branch from a local git repository.
func (s *AppProfilerService) detectGitInfo(ctx context.Context, rootPath string) (branch string, commit string, ok bool) {
	gitDir := filepath.Join(rootPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return "", "", false
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	commitOut, err := exec.CommandContext(cmdCtx, "git", "-C", rootPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", "", false
	}
	commit = strings.TrimSpace(string(commitOut))

	branchOut, err := exec.CommandContext(cmdCtx, "git", "-C", rootPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		branch = strings.TrimSpace(string(branchOut))
	}
	return branch, commit, true
}

// detectFilesystemFingerprint returns a string representing the latest modification
// time across non-excluded files, used as a change fingerprint when git is absent.
func (s *AppProfilerService) detectFilesystemFingerprint(rootPath string, opts *ProfileOptions) (string, error) {
	if opts == nil {
		opts = DefaultProfileOptions()
	}
	latest := time.Time{}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip excluded directories/files
		rel, relErr := filepath.Rel(rootPath, path)
		if relErr == nil {
			rel = filepath.ToSlash(rel)
			for _, ex := range opts.ExcludePatterns {
				ex = strings.TrimSpace(ex)
				if ex == "" {
					continue
				}
				if strings.Contains(rel, ex) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		mt := info.ModTime()
		if mt.After(latest) {
			latest = mt
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if latest.IsZero() {
		return "fs:0", nil
	}
	return fmt.Sprintf("fs:%d", latest.UTC().UnixNano()), nil
}

// SyncAutoProfiles runs one batch of auto-sync work.
// It refreshes profiles whose git HEAD (preferred) or filesystem fingerprint changed.
func (s *AppProfilerService) SyncAutoProfiles(ctx context.Context, limit int) (checked int, refreshed int, err error) {
	targets, err := s.ListAutoSyncTargets(ctx, limit)
	if err != nil {
		return 0, 0, err
	}

	for _, t := range targets {
		checked++
		// Detect change fingerprint
		branch, commit, ok := s.detectGitInfo(ctx, t.RootPath)
		source := t.RootPath
		fingerprint := ""
		if ok {
			fingerprint = commit
		} else {
			fp, fpErr := s.detectFilesystemFingerprint(t.RootPath, nil)
			if fpErr != nil {
				s.logger.Warn("auto-sync fingerprint failed", "user", t.UserID, "name", t.Name, "error", fpErr)
				continue
			}
			fingerprint = fp
		}

		if fingerprint != "" && fingerprint == t.SyncCommit {
			continue
		}

		// Refresh profile
		_, profErr := s.ProfileApplication(ctx, t.UserID, t.RootPath, t.Name, nil)
		if profErr != nil {
			s.logger.Warn("auto-sync profile refresh failed", "user", t.UserID, "name", t.Name, "error", profErr)
			continue
		}
		syncedAt := time.Now().UTC()
		if updErr := s.UpdateSyncInfo(ctx, t.UserID, t.Name, true, source, branch, fingerprint, syncedAt); updErr != nil {
			s.logger.Warn("auto-sync update sync info failed", "user", t.UserID, "name", t.Name, "error", updErr)
			continue
		}
		refreshed++
	}

	return checked, refreshed, nil
}
