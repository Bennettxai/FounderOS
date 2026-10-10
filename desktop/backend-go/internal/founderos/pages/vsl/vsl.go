// Package vsl reads the Vidalytics VSL snapshots (founderos_vsl_snapshots,
// launchpad-cohort) and keeps the Instagram-funnel video association
// (founderos_seed_meta.vsl_ig_video, founderos): FounderOS v1 db.vsl.
package vsl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUnknownVideo: the association names a video with no snapshot.
var ErrUnknownVideo = errors.New("unknown video")

const igKey = "vsl_ig_video"

type Store struct {
	Pool       *pgxpool.Pool
	Workspaces map[string]string // slug → workspace UUID
}

func (s *Store) ws(slug string) (string, error) {
	if s == nil || s.Pool == nil {
		return "", errors.New("vsl: founderos Postgres is not wired")
	}
	id := s.Workspaces[slug]
	if id == "" {
		return "", fmt.Errorf("vsl: workspace %q is not resolved", slug)
	}
	return id, nil
}

// Snapshots returns every saved snapshot payload (VslSnapshotSchema rows),
// newest period first, as stored.
func (s *Store) Snapshots(ctx context.Context) ([]json.RawMessage, error) {
	ws, err := s.ws("launchpad-cohort")
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT payload FROM founderos_vsl_snapshots WHERE workspace_id = $1 ORDER BY date_from DESC, video_id`, ws)
	if err != nil {
		return nil, fmt.Errorf("vsl: %w", err)
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("vsl: %w", err)
		}
		out = append(out, json.RawMessage(raw))
	}
	return out, rows.Err()
}

// IGVideo is the video the operator assigned to the Instagram funnel; nil when unset.
func (s *Store) IGVideo(ctx context.Context) (*string, error) {
	ws, err := s.ws("founderos")
	if err != nil {
		return nil, err
	}
	var v string
	err = s.Pool.QueryRow(ctx, `SELECT value FROM founderos_seed_meta WHERE key = $1 AND workspace_id = $2`, igKey, ws).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && v == "") {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("vsl: %w", err)
	}
	return &v, nil
}

// SetIGVideo stores the association (bridge data in Postgres, not an outbound
// side effect); "" clears it. A video with no snapshot is ErrUnknownVideo.
func (s *Store) SetIGVideo(ctx context.Context, videoID string) error {
	ws, err := s.ws("founderos")
	if err != nil {
		return err
	}
	if videoID != "" {
		aa, err := s.ws("launchpad-cohort")
		if err != nil {
			return err
		}
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM founderos_vsl_snapshots WHERE video_id = $1 AND workspace_id = $2)`, videoID, aa).Scan(&ok); err != nil {
			return fmt.Errorf("vsl: %w", err)
		}
		if !ok {
			return ErrUnknownVideo
		}
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO founderos_seed_meta (key, workspace_id, value) VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, workspace_id = EXCLUDED.workspace_id`, igKey, ws, videoID)
	if err != nil {
		return fmt.Errorf("vsl: %w", err)
	}
	return nil
}
