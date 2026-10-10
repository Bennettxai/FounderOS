package analytics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore reads the page's Postgres rows: agent runs and metric history
// (founderos), social accounts/snapshots and the email list (personal).
type PgStore struct {
	Pool       *pgxpool.Pool
	Workspaces map[string]string // slug → workspace UUID
}

func (s *PgStore) ws(slug string) (string, error) {
	if s == nil || s.Pool == nil {
		return "", errors.New("analytics: founderos Postgres is not wired")
	}
	id := s.Workspaces[slug]
	if id == "" {
		return "", fmt.Errorf("analytics: workspace %q is not resolved", slug)
	}
	return id, nil
}

// Runs started at or after since, newest first.
func (s *PgStore) Runs(ctx context.Context, since time.Time) ([]Run, error) {
	ws, err := s.ws("founderos")
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT agent_id, ok, to_char(started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM founderos_agent_runs WHERE workspace_id = $1 AND started_at >= $2 ORDER BY started_at DESC`, ws, since)
	if err != nil {
		return nil, fmt.Errorf("analytics runs: %w", err)
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.AgentID, &r.OK, &r.StartedAt); err != nil {
			return nil, fmt.Errorf("analytics runs: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RunCount is the all-time run count and the count of the trailing 7 days.
func (s *PgStore) RunCount(ctx context.Context, now time.Time) (all, last7 int, err error) {
	ws, err := s.ws("founderos")
	if err != nil {
		return 0, 0, err
	}
	cut := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	err = s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE started_at >= $2) FROM founderos_agent_runs WHERE workspace_id = $1`, ws, cut).Scan(&all, &last7)
	if err != nil {
		return 0, 0, fmt.Errorf("analytics run count: %w", err)
	}
	return all, last7, nil
}

// AgentNames maps agent ids to names (founderos_agents).
func (s *PgStore) AgentNames(ctx context.Context) (map[string]string, error) {
	ws, err := s.ws("founderos")
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name FROM founderos_agents WHERE workspace_id = $1`, ws)
	if err != nil {
		return nil, fmt.Errorf("analytics agents: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("analytics agents: %w", err)
		}
		out[id] = name
	}
	return out, rows.Err()
}

// Platform is one social account with its snapshot history (oldest first).
type Platform struct {
	Platform string  `json:"platform"`
	Handle   string  `json:"handle"`
	URL      *string `json:"url"`
	Points   []Point `json:"-"`
}

// Social returns the accounts in display order with their follower history.
func (s *PgStore) Social(ctx context.Context) ([]Platform, error) {
	ws, err := s.ws("personal")
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT platform, handle, url FROM founderos_social_accounts WHERE workspace_id = $1 ORDER BY ord, platform`, ws)
	if err != nil {
		return nil, fmt.Errorf("analytics social: %w", err)
	}
	out := []Platform{}
	index := map[string]int{}
	for rows.Next() {
		var p Platform
		if err := rows.Scan(&p.Platform, &p.Handle, &p.URL); err != nil {
			rows.Close()
			return nil, fmt.Errorf("analytics social: %w", err)
		}
		p.Points = []Point{}
		index[p.Platform] = len(out)
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	srows, err := s.Pool.Query(ctx, `SELECT platform, to_char(captured_on, 'YYYY-MM-DD'), followers FROM founderos_social_snapshots
		WHERE workspace_id = $1 ORDER BY platform, captured_on`, ws)
	if err != nil {
		return nil, fmt.Errorf("analytics social snapshots: %w", err)
	}
	defer srows.Close()
	for srows.Next() {
		var platform, at string
		var followers int64
		if err := srows.Scan(&platform, &at, &followers); err != nil {
			return nil, fmt.Errorf("analytics social snapshots: %w", err)
		}
		if i, ok := index[platform]; ok {
			out[i].Points = append(out[i].Points, Point{CapturedAt: at, Value: float64(followers)})
		}
	}
	return out, srows.Err()
}

// EmailPoints is the real email-list history: quality ok and never a seeded
// row (lib/email-list.ts realEmailSnapshots), oldest first.
func (s *PgStore) EmailPoints(ctx context.Context) ([]Point, error) {
	ws, err := s.ws("personal")
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT to_char(captured_on, 'YYYY-MM-DD'), subscribers FROM founderos_email_list_snapshots
		WHERE workspace_id = $1 AND quality = 'ok' AND source NOT LIKE 'seed%' ORDER BY captured_on`, ws)
	if err != nil {
		return nil, fmt.Errorf("analytics email list: %w", err)
	}
	defer rows.Close()
	out := []Point{}
	for rows.Next() {
		var p Point
		var n int64
		if err := rows.Scan(&p.CapturedAt, &n); err != nil {
			return nil, fmt.Errorf("analytics email list: %w", err)
		}
		p.Value = float64(n)
		out = append(out, p)
	}
	return out, rows.Err()
}

// MetricHistory is the last captured value per UTC day for one metric over
// the trailing days ending on today (db.metricSnapshots.history).
func (s *PgStore) MetricHistory(ctx context.Context, metricID string, days int, today string) ([]float64, error) {
	ws, err := s.ws("founderos")
	if err != nil {
		return nil, err
	}
	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return nil, err
	}
	if days < 1 {
		days = 1
	}
	from := t.AddDate(0, 0, -days).Format("2006-01-02")
	rows, err := s.Pool.Query(ctx, `SELECT value FROM (
			SELECT to_char(captured_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS d, value,
				ROW_NUMBER() OVER (PARTITION BY to_char(captured_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') ORDER BY captured_at DESC) AS rn
			FROM founderos_metric_snapshots WHERE workspace_id = $1 AND metric_id = $2
		) x WHERE rn = 1 AND d >= $3 ORDER BY d`, ws, metricID, from)
	if err != nil {
		return nil, fmt.Errorf("analytics metric history: %w", err)
	}
	defer rows.Close()
	out := []float64{}
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
