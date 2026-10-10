-- Migration: 105_optimal_engine.sql
-- Description: OptimalOS Cloud Engine — signal classification, knowledge graph, search index, learning loop
-- Created: 2026-04-13
-- Purpose: Power the OptimalOS engine for cloud users. Local users use SQLite + filesystem;
--          cloud users store everything in PostgreSQL, scoped by workspace_id.

-- ============================================================
-- EXTENSIONS (idempotent)
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- OPTIMAL SIGNALS — Classified markdown files with S=(M,G,T,F,W)
-- Cloud equivalent of the local filesystem signals directory.
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_signals (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    node_id         UUID        REFERENCES nodes(id) ON DELETE SET NULL,
    created_by      VARCHAR(255) NOT NULL REFERENCES "user"(id),

    -- Signal identity
    title           VARCHAR(500) NOT NULL DEFAULT '',
    slug            VARCHAR(255) NOT NULL DEFAULT '',
    uri             TEXT        NOT NULL DEFAULT '',   -- optimal://workspace/nodes/slug/signals/filename
    path            TEXT        NOT NULL DEFAULT '',   -- virtual filesystem path

    -- Content (markdown body — L2 full content)
    content         TEXT        NOT NULL DEFAULT '',

    -- Signal Theory dimensions S=(M,G,T,F,W)
    mode            VARCHAR(50)  NOT NULL DEFAULT 'linguistic',  -- linguistic, visual, code, data, mixed
    genre           VARCHAR(100) NOT NULL DEFAULT 'note',        -- 143+ genres from catalogue
    signal_type     VARCHAR(50)  NOT NULL DEFAULT 'inform',      -- direct, inform, commit, decide, express
    format          VARCHAR(50)  NOT NULL DEFAULT 'markdown',    -- markdown, code, json, yaml, etc.
    structure       VARCHAR(200) NOT NULL DEFAULT '',             -- genre skeleton key

    -- Quality
    sn_ratio        REAL        NOT NULL DEFAULT 0.5,            -- signal-to-noise ratio 0.0–1.0

    -- Pre-computed tier summaries
    l0_abstract     TEXT        NOT NULL DEFAULT '',             -- ~100 tokens headline
    l1_overview     TEXT        NOT NULL DEFAULT '',             -- ~2K tokens summary

    -- Routing
    primary_node    VARCHAR(255) NOT NULL DEFAULT '',            -- node this was routed to
    cross_ref_nodes JSONB       NOT NULL DEFAULT '[]',           -- cross-referenced nodes

    -- Entities extracted from content
    entities        JSONB       NOT NULL DEFAULT '[]',           -- ["Account owner", "Operations lead"]

    -- Temporal
    valid_from      TIMESTAMPTZ DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,                                  -- NULL = still current
    supersedes      UUID        REFERENCES optimal_signals(id) ON DELETE SET NULL,

    -- Flexible metadata
    metadata        JSONB       NOT NULL DEFAULT '{}',

    -- Timestamps
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    indexed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Scalar indexes
CREATE INDEX IF NOT EXISTS idx_optimal_signals_workspace  ON optimal_signals(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_node       ON optimal_signals(node_id);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_genre      ON optimal_signals(genre);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_created    ON optimal_signals(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_mode       ON optimal_signals(mode);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_slug       ON optimal_signals(workspace_id, slug);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_valid      ON optimal_signals(workspace_id, valid_from DESC) WHERE valid_until IS NULL;

-- Generated tsvector column for full-text search (replaces SQLite FTS5)
ALTER TABLE optimal_signals
    ADD COLUMN IF NOT EXISTS search_vector tsvector
        GENERATED ALWAYS AS (
            setweight(to_tsvector('english', COALESCE(title,       '')), 'A') ||
            setweight(to_tsvector('english', COALESCE(l0_abstract, '')), 'B') ||
            setweight(to_tsvector('english', COALESCE(l1_overview, '')), 'C') ||
            setweight(to_tsvector('english', COALESCE(content,     '')), 'D')
        ) STORED;

CREATE INDEX IF NOT EXISTS idx_optimal_signals_fts         ON optimal_signals USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_optimal_signals_title_trgm  ON optimal_signals USING GIN(title gin_trgm_ops);

-- ============================================================
-- OPTIMAL ENTITIES — People, orgs, tools, concepts extracted from signals
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_entities (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    name            VARCHAR(500) NOT NULL,
    canonical_name  VARCHAR(500) NOT NULL DEFAULT '',  -- normalized / deduplicated form
    type            VARCHAR(100) NOT NULL DEFAULT 'person',  -- person, org, tool, concept, project

    properties      JSONB       NOT NULL DEFAULT '{}',
    mention_count   INTEGER     NOT NULL DEFAULT 1,

    first_seen      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id, canonical_name)
);

CREATE INDEX IF NOT EXISTS idx_optimal_entities_workspace  ON optimal_entities(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_entities_name       ON optimal_entities(name);
CREATE INDEX IF NOT EXISTS idx_optimal_entities_type       ON optimal_entities(workspace_id, type);

-- ============================================================
-- OPTIMAL EDGES — Knowledge graph relationships between entities and signals
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_edges (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    source_id       UUID        NOT NULL,                          -- entity or signal UUID
    source_type     VARCHAR(50) NOT NULL DEFAULT 'entity',        -- entity, signal
    target_id       UUID        NOT NULL,
    target_type     VARCHAR(50) NOT NULL DEFAULT 'entity',        -- entity, signal

    relation        VARCHAR(100) NOT NULL DEFAULT 'related',      -- mentioned_in, lives_in, works_on, cross_ref, supersedes
    weight          REAL        NOT NULL DEFAULT 1.0,

    -- Temporal validity
    valid_from      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until     TIMESTAMPTZ,                                   -- NULL = still valid

    reason          TEXT,                                          -- why this relationship exists
    metadata        JSONB       NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id, source_id, target_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_optimal_edges_workspace  ON optimal_edges(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_source     ON optimal_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_target     ON optimal_edges(target_id);
CREATE INDEX IF NOT EXISTS idx_optimal_edges_relation   ON optimal_edges(workspace_id, relation);

-- ============================================================
-- OPTIMAL OBSERVATIONS — Learning loop (SICA friction capture)
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_observations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by      VARCHAR(255) NOT NULL REFERENCES "user"(id),

    category        VARCHAR(100) NOT NULL DEFAULT 'pattern',      -- process, people, tool, decision, pattern, friction
    content         TEXT        NOT NULL,
    confidence      REAL        NOT NULL DEFAULT 0.5,             -- 0.0–1.0
    source          VARCHAR(100) NOT NULL DEFAULT 'explicit',     -- explicit, contextual, mining

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_observations_workspace  ON optimal_observations(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_observations_category   ON optimal_observations(workspace_id, category);
CREATE INDEX IF NOT EXISTS idx_optimal_observations_created    ON optimal_observations(created_at DESC);

-- ============================================================
-- OPTIMAL TOPOLOGY — Per-workspace routing configuration
-- Stores the topology.yaml equivalent: nodes, people, routes, genre skeletons.
-- One row per workspace (UNIQUE constraint enforced).
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_topology (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Full topology config (matches topology.yaml structure)
    -- { nodes: {...}, people: {...}, routes: [...] }
    config          JSONB       NOT NULL DEFAULT '{}',

    -- User-customised genre skeletons (extends the 10 built-in ones)
    genre_skeletons JSONB       NOT NULL DEFAULT '{}',

    -- Temporal decay half-lives: genre → hours
    genre_half_lives JSONB      NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_optimal_topology_workspace ON optimal_topology(workspace_id);

-- ============================================================
-- OPTIMAL SESSIONS — Conversation sessions with signal extraction
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by      VARCHAR(255) NOT NULL REFERENCES "user"(id),

    status          VARCHAR(50) NOT NULL DEFAULT 'active',        -- active, committed, archived
    summary         TEXT        NOT NULL DEFAULT '',

    -- Signal UUIDs extracted during this session
    extracted_signal_ids JSONB  NOT NULL DEFAULT '[]',

    message_count   INTEGER     NOT NULL DEFAULT 0,

    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    committed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_sessions_workspace  ON optimal_sessions(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_sessions_status     ON optimal_sessions(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_optimal_sessions_started    ON optimal_sessions(started_at DESC);

-- ============================================================
-- OPTIMAL DECISION LOG — Append-only audit trail
-- Never UPDATE or DELETE rows; insert only.
-- ============================================================
CREATE TABLE IF NOT EXISTS optimal_decision_log (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    actor           VARCHAR(255) NOT NULL,                        -- user ID or agent name
    action          VARCHAR(100) NOT NULL,                        -- created, updated, promoted, demoted, archived, superseded
    target_type     VARCHAR(100) NOT NULL,                        -- signal, entity, edge, observation
    target_id       UUID        NOT NULL,

    what_changed    TEXT        NOT NULL,
    why             TEXT,
    alternatives_considered JSONB,
    context         JSONB,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_workspace  ON optimal_decision_log(workspace_id);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_target     ON optimal_decision_log(target_id);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_created    ON optimal_decision_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_optimal_decision_log_actor      ON optimal_decision_log(workspace_id, actor);
