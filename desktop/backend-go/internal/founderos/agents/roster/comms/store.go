package comms

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// errNoWorkspace: the founderos_* tables are workspace-scoped, and a store
// without a pool or workspace can read nothing. It says so rather than
// returning an empty result.
var errNoWorkspace = errors.New("workspace not bootstrapped on the bridge (run founderos-bootstrap)")

// PgDigestStore is DigestStore over founderos_comms_digests, founderos_digest_reads
// and founderos_contact_tags, all in the personal workspace.
type PgDigestStore struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgDigestStore(pool *pgxpool.Pool, workspaceID string) *PgDigestStore {
	return &PgDigestStore{pool: pool, workspaceID: workspaceID}
}

func (s *PgDigestStore) ready() error {
	if s.pool == nil || s.workspaceID == "" {
		return errNoWorkspace
	}
	return nil
}

// Latest is commsDigests.latest(): newest generated_at, ties broken by
// imported_at then id (docs/founderos/table-map.md §17).
func (s *PgDigestStore) Latest(ctx context.Context) ([]byte, bool, error) {
	if err := s.ready(); err != nil {
		return nil, false, err
	}
	var payload string
	err := s.pool.QueryRow(ctx, `
		SELECT payload::text FROM founderos_comms_digests
		WHERE workspace_id = $1
		ORDER BY generated_at DESC, imported_at DESC, id DESC
		LIMIT 1`, s.workspaceID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(payload), true, nil
}

// ClearedKeys is digestReads.keys(), newest first.
func (s *PgDigestStore) ClearedKeys(ctx context.Context) ([]string, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT key FROM founderos_digest_reads WHERE workspace_id = $1 ORDER BY read_at DESC`, s.workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *PgDigestStore) ContactTags(ctx context.Context) ([]ContactTag, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT person, channel, tag, tier FROM founderos_contact_tags WHERE workspace_id = $1 ORDER BY person, channel`, s.workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (ContactTag, error) {
		var t ContactTag
		err := r.Scan(&t.Person, &t.Channel, &t.Tag, &t.Tier)
		return t, err
	})
}

func (s *PgDigestStore) Insert(ctx context.Context, id string, generatedAt time.Time, payload []byte) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO founderos_comms_digests (id, workspace_id, generated_at, payload)
		VALUES ($1, $2, $3, $4::jsonb)`, id, s.workspaceID, generatedAt, string(payload))
	return err
}

// PgSocialQueue counts queued posts in founderos_social_posts (personal),
// the bridge's socialPosts.queued().
type PgSocialQueue struct {
	pool        *pgxpool.Pool
	workspaceID string
}

func NewPgSocialQueue(pool *pgxpool.Pool, workspaceID string) *PgSocialQueue {
	return &PgSocialQueue{pool: pool, workspaceID: workspaceID}
}

func (q *PgSocialQueue) QueuedPosts(ctx context.Context) (int, error) {
	if q.pool == nil || q.workspaceID == "" {
		return 0, errNoWorkspace
	}
	var n int
	err := q.pool.QueryRow(ctx, `SELECT count(*) FROM founderos_social_posts WHERE workspace_id = $1 AND status = 'queued'`, q.workspaceID).Scan(&n)
	return n, err
}
