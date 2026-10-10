// Package bootstrap creates the operator's workspaces in BusinessOS and routes each
// one to its home Optimal Engine (workspaces.settings.optimal_engine), per
// config/founderos/engine-topology.yaml. Re-running it only refreshes routing.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
	"github.com/rhl/businessos-backend/internal/services"
)

// Engine is how BusinessOS reaches one Optimal Engine.
type Engine struct {
	URL string
	Key string
}

// EngineConfig is the shape BusinessOS stores under settings.optimal_engine
// (see handlers/workspace_engine.go engineStoredConfig).
type EngineConfig struct {
	Enabled   bool   `json:"enabled"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	Workspace string `json:"workspace"`
	UpdatedAt string `json:"updated_at,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}

// EngineTenant is the Optimal Engine tenant the operator's workspaces live under.
// The engine (v0.3.0) scopes reads by full workspace id, "<tenant>:<slug>";
// a bare slug silently matches nothing, so routing stores the full id.
const EngineTenant = "default"

type Spec struct {
	Slug        string
	Name        string
	Description string
	Engine      EngineConfig
}

// Plan lists the BusinessOS workspaces to create: every business and
// personal topology workspace. Device workspaces (hermes, macbook-dev, ...)
// live only inside their engine.
func Plan(topo *topology.Topology, engines map[string]Engine) ([]Spec, error) {
	var specs []Spec
	for _, w := range topo.Workspaces {
		if w.Class == topology.ClassDevice {
			continue
		}
		e, ok := engines[w.Home]
		if !ok || e.URL == "" {
			return nil, fmt.Errorf("workspace %q: no endpoint for its home engine %q", w.Slug, w.Home)
		}
		name := w.Name
		if name == "" {
			name = w.Slug
		}
		specs = append(specs, Spec{
			Slug:        w.Slug,
			Name:        name,
			Description: w.Holds,
			Engine:      EngineConfig{Enabled: true, BaseURL: e.URL, APIKey: e.Key, Workspace: EngineTenant + ":" + w.Slug},
		})
	}
	return specs, nil
}

type Result struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

// Apply creates missing workspaces through BusinessOS's own WorkspaceService
// (roles, sync policies, owner membership) and refreshes the engine routing
// on existing ones.
func Apply(ctx context.Context, pool *pgxpool.Pool, ownerEmail string, specs []Spec) (Result, error) {
	var res Result
	var ownerID string
	err := pool.QueryRow(ctx, `SELECT id FROM "user" WHERE lower(email) = lower($1)`, ownerEmail).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, fmt.Errorf("no BusinessOS user with email %s: sign up on the bridge first (http://localhost:5273)", ownerEmail)
	}
	if err != nil {
		return res, err
	}
	svc := services.NewWorkspaceService(pool)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, s := range specs {
		eng := s.Engine
		eng.UpdatedAt, eng.UpdatedBy = now, "founderos-bootstrap"
		var id string
		err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = $1`, s.Slug).Scan(&id)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			desc := s.Description
			if _, err := svc.CreateWorkspace(ctx, services.CreateWorkspaceRequest{
				Name:        s.Name,
				Slug:        s.Slug,
				Description: &desc,
				Settings:    map[string]interface{}{"optimal_engine": eng},
			}, ownerID); err != nil {
				return res, fmt.Errorf("create %s: %w", s.Slug, err)
			}
			res.Created++
		case err != nil:
			return res, err
		default:
			if _, err := pool.Exec(ctx,
				`UPDATE workspaces SET settings = jsonb_set(coalesce(settings, '{}'::jsonb), '{optimal_engine}', $2::jsonb), updated_at = NOW() WHERE id = $1`,
				id, eng); err != nil {
				return res, fmt.Errorf("route %s: %w", s.Slug, err)
			}
			res.Updated++
		}
	}
	return res, nil
}
