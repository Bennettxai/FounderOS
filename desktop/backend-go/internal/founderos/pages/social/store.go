package social

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// Store is the social slice of Postgres: founderos_social_*,
// founderos_email_list_snapshots and founderos_social_posts, all in the
// personal workspace.
type Store struct {
	Pool        *pgxpool.Pool
	WorkspaceID string
}

func (s *Store) ready() error {
	if s == nil || s.Pool == nil || s.WorkspaceID == "" {
		return pagekit.ErrNoWorkspace
	}
	return nil
}

func dayOf(t time.Time) string { return t.Format("2006-01-02") }

func isoOf(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// Load reads everything the social pages render.
func (s *Store) Load(ctx context.Context) (*Data, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	x := &Data{Snapshots: map[string][]Snapshot{}}
	ws := s.WorkspaceID

	rows, err := s.Pool.Query(ctx, `SELECT platform, handle, url, ord FROM founderos_social_accounts WHERE workspace_id = $1 ORDER BY ord`, ws)
	if err != nil {
		return nil, err
	}
	x.Accounts, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Account, error) {
		var a Account
		return a, r.Scan(&a.Platform, &a.Handle, &a.URL, &a.Order)
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.Pool.Query(ctx, `SELECT platform, captured_on, followers, source FROM founderos_social_snapshots WHERE workspace_id = $1 ORDER BY platform, captured_on`, ws)
	if err != nil {
		return nil, err
	}
	snaps, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Snapshot, error) {
		var sn Snapshot
		var at time.Time
		err := r.Scan(&sn.Platform, &at, &sn.Followers, &sn.Source)
		sn.CapturedAt = dayOf(at)
		return sn, err
	})
	if err != nil {
		return nil, err
	}
	for _, sn := range snaps {
		x.Snapshots[sn.Platform] = append(x.Snapshots[sn.Platform], sn)
	}

	rows, err = s.Pool.Query(ctx, `SELECT d.platform, d.count, d.updated_at FROM founderos_social_dms d
		LEFT JOIN founderos_social_accounts a ON a.platform = d.platform AND a.workspace_id = d.workspace_id
		WHERE d.workspace_id = $1 ORDER BY a.ord NULLS LAST, d.platform`, ws)
	if err != nil {
		return nil, err
	}
	x.DMs, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (DM, error) {
		var d DM
		var at time.Time
		err := r.Scan(&d.Platform, &d.Count, &at)
		d.UpdatedAt = isoOf(at)
		return d, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.Pool.Query(ctx, `SELECT platform, captured_on, count, source FROM founderos_social_dm_snapshots WHERE workspace_id = $1 ORDER BY platform, captured_on`, ws)
	if err != nil {
		return nil, err
	}
	x.DMSnapshots, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (DMSnapshot, error) {
		var d DMSnapshot
		var at time.Time
		err := r.Scan(&d.Platform, &at, &d.Count, &d.Source)
		d.CapturedAt = dayOf(at)
		return d, err
	})
	if err != nil {
		return nil, err
	}

	if x.DMMessages, err = s.DMMessages(ctx); err != nil {
		return nil, err
	}

	rows, err = s.Pool.Query(ctx, `SELECT captured_on, subscribers, source, publication_id, metric, quality FROM founderos_email_list_snapshots
		WHERE workspace_id = $1 AND quality = 'ok' ORDER BY captured_on`, ws)
	if err != nil {
		return nil, err
	}
	x.Email, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (EmailSnapshot, error) {
		var e EmailSnapshot
		var at time.Time
		err := r.Scan(&at, &e.Subscribers, &e.Source, &e.PublicationID, &e.Metric, &e.Quality)
		e.CapturedAt = dayOf(at)
		return e, err
	})
	if err != nil {
		return nil, err
	}

	if x.Posts, err = s.Posts(ctx); err != nil {
		return nil, err
	}
	return x, nil
}

// DMMessages is every DM message, newest first.
func (s *Store) DMMessages(ctx context.Context) ([]DMMessage, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, platform, subscriber_id, name, handle, text, direction, tag, ts, source
		FROM founderos_social_dm_messages WHERE workspace_id = $1 ORDER BY ts DESC`, s.WorkspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (DMMessage, error) {
		var m DMMessage
		var ts time.Time
		err := r.Scan(&m.ID, &m.Platform, &m.SubscriberID, &m.Name, &m.Handle, &m.Text, &m.Direction, &m.Tag, &ts, &m.Source)
		m.TS = isoOf(ts)
		return m, err
	})
}

// Posts is the publish queue, newest first.
func (s *Store) Posts(ctx context.Context) ([]Post, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, caption, media_url, platforms, status, scheduled_for, created_at
		FROM founderos_social_posts WHERE workspace_id = $1 ORDER BY created_at DESC`, s.WorkspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Post, error) {
		var p Post
		var raw []byte
		var sched *time.Time
		var created time.Time
		if err := r.Scan(&p.ID, &p.Caption, &p.MediaURL, &raw, &p.Status, &sched, &created); err != nil {
			return p, err
		}
		if err := json.Unmarshal(raw, &p.Platforms); err != nil {
			return p, err
		}
		if sched != nil {
			v := isoOf(*sched)
			p.ScheduledFor = &v
		}
		p.CreatedAt = isoOf(created)
		return p, nil
	})
}

// InsertSnapshots upserts today's follower snapshots (same-day re-sync
// overwrites, as the SQLite INSERT OR REPLACE did).
func (s *Store) InsertSnapshots(ctx context.Context, rows []Snapshot) error {
	if err := s.ready(); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := s.Pool.Exec(ctx, `INSERT INTO founderos_social_snapshots (platform, captured_on, workspace_id, followers, source)
			VALUES ($1, $2::date, $3, $4, $5)
			ON CONFLICT (platform, captured_on) DO UPDATE SET followers = EXCLUDED.followers, source = EXCLUDED.source, workspace_id = EXCLUDED.workspace_id`,
			r.Platform, r.CapturedAt, s.WorkspaceID, r.Followers, r.Source); err != nil {
			return err
		}
	}
	return nil
}

// EnqueuePost records a post in the queue with its honest status.
func (s *Store) EnqueuePost(ctx context.Context, p Post) error {
	if err := s.ready(); err != nil {
		return err
	}
	raw, err := json.Marshal(p.Platforms)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO founderos_social_posts (id, workspace_id, caption, media_url, platforms, status, scheduled_for, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7::timestamptz, $8::timestamptz)`,
		p.ID, s.WorkspaceID, p.Caption, p.MediaURL, string(raw), p.Status, p.ScheduledFor, p.CreatedAt)
	return err
}

// UpsertDMMessage stores one DM message (by id).
func (s *Store) UpsertDMMessage(ctx context.Context, m DMMessage) error {
	if err := s.ready(); err != nil {
		return err
	}
	if m.Direction != "in" && m.Direction != "out" {
		return errors.New("direction must be in or out")
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO founderos_social_dm_messages (id, workspace_id, platform, subscriber_id, name, handle, text, direction, tag, ts, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::timestamptz, $11)
		ON CONFLICT (id) DO UPDATE SET text = EXCLUDED.text, name = EXCLUDED.name, handle = EXCLUDED.handle, tag = EXCLUDED.tag, ts = EXCLUDED.ts`,
		m.ID, s.WorkspaceID, m.Platform, m.SubscriberID, m.Name, m.Handle, m.Text, m.Direction, m.Tag, m.TS, m.Source)
	return err
}
