-- ============================================================================
-- FounderOS v1 bridge tables (spec criterion 3.2; design in docs/founderos/table-map.md)
--
-- 51 founderos_* tables that hold FounderOS v1's SQLite data after the ETL
-- (cmd/founderos-etl). No BusinessOS table is altered or dropped. Every
-- statement is IF NOT EXISTS, so this file is safe to re-apply and is mirrored
-- verbatim at the end of schema.sql.
--
-- Conventions (table-map.md "Conventions"):
--   * workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
--     indexed; resolved by slug by the ETL.
--   * The operator's own key is the primary key, verbatim (TEXT ids stay TEXT).
--   * imported_at is set by the ETL, not copied from the source.
--   * JSON-in-TEXT becomes JSONB; calendar days become DATE; instants become
--     TIMESTAMPTZ.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- Org and agent roster
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_departments (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    tagline TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL,
    ord INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_departments_workspace ON founderos_departments(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_agents (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active', 'idle', 'training', 'planned')),
    tier TEXT NOT NULL CHECK (tier IN ('lead', 'specialist', 'worker')),
    description TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    tools JSONB NOT NULL DEFAULT '[]',
    parent_id TEXT,
    instance TEXT NOT NULL DEFAULT 'builtin',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agents_workspace ON founderos_agents(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agents_department ON founderos_agents(department_id);

CREATE TABLE IF NOT EXISTS founderos_people (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    tools JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_people_workspace ON founderos_people(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_sop_tasks (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    department_id TEXT NOT NULL REFERENCES founderos_departments(id),
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    steps JSONB NOT NULL DEFAULT '[]',
    assignee_kind TEXT NOT NULL CHECK (assignee_kind IN ('agent', 'person')),
    assignee_id TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_sop_tasks_workspace ON founderos_sop_tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_sop_tasks_assignee ON founderos_sop_tasks(assignee_kind, assignee_id);

CREATE TABLE IF NOT EXISTS founderos_tools (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('connected', 'available', 'planned')),
    color TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_tools_workspace ON founderos_tools(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_skills (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_agent_id TEXT,
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('live', 'learning', 'planned')),
    tools JSONB NOT NULL DEFAULT '[]',
    markdown TEXT NOT NULL DEFAULT '',
    ord INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_skills_workspace ON founderos_skills(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_personas (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    ord INTEGER NOT NULL,
    name TEXT NOT NULL,
    archetype TEXT NOT NULL,
    tagline TEXT NOT NULL,
    summary TEXT NOT NULL,
    accent TEXT NOT NULL,
    north_star TEXT NOT NULL,
    pillars JSONB NOT NULL DEFAULT '[]',
    connectors JSONB NOT NULL DEFAULT '[]',
    metrics JSONB NOT NULL DEFAULT '[]',
    brain_use TEXT NOT NULL,
    signature_play TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_personas_workspace ON founderos_personas(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_workflows (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    subtitle TEXT NOT NULL DEFAULT '',
    revenue_usd INTEGER NOT NULL DEFAULT 0,
    ord INTEGER NOT NULL DEFAULT 0,
    steps JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_workflows_workspace ON founderos_workflows(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_metrics (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL DEFAULT '',
    delta DOUBLE PRECISION NOT NULL DEFAULT 0,
    period TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_metrics_workspace ON founderos_metrics(workspace_id);

-- ----------------------------------------------------------------------------
-- Agent runtime
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_agent_runs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    ok BOOLEAN NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    model TEXT,
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost_usd DOUBLE PRECISION,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_workspace ON founderos_agent_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_started ON founderos_agent_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_runs_agent ON founderos_agent_runs(agent_id, started_at DESC);

CREATE TABLE IF NOT EXISTS founderos_agent_messages (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content TEXT NOT NULL DEFAULT '',
    tool_calls JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_messages_workspace ON founderos_agent_messages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_messages_agent ON founderos_agent_messages(agent_id, created_at);

CREATE TABLE IF NOT EXISTS founderos_agent_tasks (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'doing', 'review', 'done')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_tasks_workspace ON founderos_agent_tasks(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_tasks_agent ON founderos_agent_tasks(agent_id, status);

CREATE TABLE IF NOT EXISTS founderos_agent_crons (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    schedule TEXT NOT NULL,
    description TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_agent_crons_workspace ON founderos_agent_crons(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_cron_runs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    cron_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    ok BOOLEAN NOT NULL,
    summary TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_workspace ON founderos_cron_runs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_cron ON founderos_cron_runs(cron_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_founderos_cron_runs_started ON founderos_cron_runs(started_at DESC);

CREATE TABLE IF NOT EXISTS founderos_broadcasts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcasts_workspace ON founderos_broadcasts(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_broadcast_replies (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    broadcast_id TEXT NOT NULL REFERENCES founderos_broadcasts(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    ok BOOLEAN NOT NULL,
    reply TEXT NOT NULL DEFAULT '',
    finished_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcast_replies_workspace ON founderos_broadcast_replies(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_broadcast_replies_broadcast ON founderos_broadcast_replies(broadcast_id);

-- ----------------------------------------------------------------------------
-- Comms and knowledge
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_comms_digests (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    generated_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_comms_digests_workspace ON founderos_comms_digests(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_comms_digests_generated ON founderos_comms_digests(generated_at DESC);

CREATE TABLE IF NOT EXISTS founderos_digest_reads (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_digest_reads_workspace ON founderos_digest_reads(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_digest_reads_read_at ON founderos_digest_reads(read_at);

CREATE TABLE IF NOT EXISTS founderos_contact_tags (
    person TEXT NOT NULL,
    channel TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    tier INTEGER NOT NULL CHECK (tier BETWEEN 1 AND 3),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (person, channel)
);
CREATE INDEX IF NOT EXISTS idx_founderos_contact_tags_workspace ON founderos_contact_tags(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_plaud_ingests (
    file_id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    recorded_at TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL,
    via TEXT NOT NULL CHECK (via IN ('gbrain', 'store')),
    slug TEXT NOT NULL,
    claims INTEGER NOT NULL DEFAULT 0,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_plaud_ingests_workspace ON founderos_plaud_ingests(workspace_id);

-- ----------------------------------------------------------------------------
-- Social and content
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_social_accounts (
    platform TEXT PRIMARY KEY CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    handle TEXT NOT NULL,
    url TEXT,
    ord INTEGER NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_accounts_workspace ON founderos_social_accounts(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_snapshots (
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    captured_on DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    followers INTEGER NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform, captured_on)
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_snapshots_workspace ON founderos_social_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dms (
    platform TEXT PRIMARY KEY CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    count INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dms_workspace ON founderos_social_dms(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dm_snapshots (
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    captured_on DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    count INTEGER NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform, captured_on)
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_snapshots_workspace ON founderos_social_dm_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_social_dm_messages (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform TEXT NOT NULL CHECK (platform IN ('instagram', 'tiktok', 'twitter', 'youtube', 'linkedin')),
    subscriber_id TEXT NOT NULL,
    name TEXT NOT NULL,
    handle TEXT,
    text TEXT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    tag TEXT,
    ts TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_workspace ON founderos_social_dm_messages(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_ts ON founderos_social_dm_messages(ts);
CREATE INDEX IF NOT EXISTS idx_founderos_social_dm_messages_subscriber ON founderos_social_dm_messages(platform, subscriber_id);

CREATE TABLE IF NOT EXISTS founderos_social_posts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    caption TEXT NOT NULL,
    media_url TEXT,
    platforms JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'published', 'failed')),
    scheduled_for TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_social_posts_workspace ON founderos_social_posts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_social_posts_status ON founderos_social_posts(status);

CREATE TABLE IF NOT EXISTS founderos_email_list_snapshots (
    captured_on DATE PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    subscribers INTEGER NOT NULL,
    source TEXT NOT NULL,
    publication_id TEXT NOT NULL DEFAULT '',
    metric TEXT NOT NULL DEFAULT '',
    quality TEXT NOT NULL DEFAULT 'ok' CHECK (quality IN ('ok', 'suspect')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_email_list_snapshots_workspace ON founderos_email_list_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_email_list_sync_misses (
    attempted_on DATE PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_email_list_sync_misses_workspace ON founderos_email_list_sync_misses(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_lead_magnets (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    offer TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('live', 'draft', 'paused', 'archived')),
    captures TEXT NOT NULL CHECK (captures IN ('email', 'booking', 'none')),
    destination TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    launched_at TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'os')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_lead_magnets_workspace ON founderos_lead_magnets(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_brand_deals (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    brand TEXT NOT NULL,
    status TEXT NOT NULL,
    tier TEXT,
    deal_value_usd DOUBLE PRECISION,
    budget_usd DOUBLE PRECISION,
    amount_agreed_usd DOUBLE PRECISION,
    suggested_rate_usd DOUBLE PRECISION,
    paid_in_full BOOLEAN NOT NULL DEFAULT false,
    deadline TIMESTAMPTZ,
    follow_up_date TIMESTAMPTZ,
    contact_name TEXT,
    contact_email TEXT,
    main_channel TEXT,
    video_type TEXT,
    source TEXT,
    icp_fit TEXT,
    notion_url TEXT NOT NULL,
    last_edited TIMESTAMPTZ NOT NULL,
    seeded BOOLEAN NOT NULL DEFAULT false,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_brand_deals_workspace ON founderos_brand_deals(workspace_id);

-- ----------------------------------------------------------------------------
-- Sales and clients
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_funnel_contacts (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    venture TEXT NOT NULL CHECK (venture IN ('vantage', 'launchpad-cohort')),
    status TEXT NOT NULL CHECK (status IN ('first_touch', 'engaged', 'nurtured', 'opted_in', 'converted')),
    product TEXT,
    amount_usd DOUBLE PRECISION,
    relationship TEXT NOT NULL DEFAULT 'warm' CHECK (relationship IN ('cold', 'warm', 'hot')),
    likelihood INTEGER NOT NULL DEFAULT 50 CHECK (likelihood BETWEEN 0 AND 100),
    email TEXT,
    phone TEXT,
    person TEXT,
    company TEXT,
    role TEXT,
    linkedin TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_contacts_workspace ON founderos_funnel_contacts(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_contacts_venture ON founderos_funnel_contacts(venture, status);

CREATE TABLE IF NOT EXISTS founderos_funnel_touches (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    contact_id TEXT NOT NULL REFERENCES founderos_funnel_contacts(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL CHECK (seq > 0),
    stage TEXT NOT NULL CHECK (stage IN ('first_touch', 'engaged', 'nurtured', 'opted_in', 'converted')),
    channel TEXT NOT NULL CHECK (channel IN ('organic', 'ads', 'dm', 'email', 'webinar', 'call', 'checkout', 'crm')),
    label TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('typeform', 'calendar', 'fathom', 'trakyo', 'meta-ads', 'stripe', 'manual', 'attio', 'ghl')),
    at DATE NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_touches_workspace ON founderos_funnel_touches(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_funnel_touches_contact ON founderos_funnel_touches(contact_id, seq);

CREATE TABLE IF NOT EXISTS founderos_proposals (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client TEXT NOT NULL,
    brand TEXT NOT NULL CHECK (brand IN ('vantage', 'launchpad-cohort')),
    url TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'sent', 'won', 'lost')),
    amount_usd DOUBLE PRECISION,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    origin TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'os')),
    -- The StatiCrypt code itself lives in credential_vault
    -- (provider_id 'founderos-staticrypt'); only its presence is recorded here.
    has_access_code BOOLEAN NOT NULL DEFAULT false,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_proposals_workspace ON founderos_proposals(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_deliverable_decisions (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    decision TEXT NOT NULL CHECK (decision IN ('approved', 'dismissed')),
    decided_at TIMESTAMPTZ NOT NULL,
    decided_revision TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_deliverable_decisions_workspace ON founderos_deliverable_decisions(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_client_work (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    brief TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('saved', 'launching', 'launched', 'needs_attention')),
    created_at TIMESTAMPTZ NOT NULL,
    superset_workspace_id TEXT,
    detail TEXT,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_client_work_workspace ON founderos_client_work(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_client_work_created ON founderos_client_work(created_at DESC);

CREATE TABLE IF NOT EXISTS founderos_vsl_snapshots (
    video_id TEXT NOT NULL,
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (video_id, date_from, date_to),
    CHECK (date_from <= date_to)
);
CREATE INDEX IF NOT EXISTS idx_founderos_vsl_snapshots_workspace ON founderos_vsl_snapshots(workspace_id);

-- ----------------------------------------------------------------------------
-- Trading
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_trading_snapshots (
    account_id TEXT NOT NULL DEFAULT 'individual',
    captured_at TIMESTAMPTZ NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_label TEXT NOT NULL DEFAULT 'Individual',
    account_value_usd DOUBLE PRECISION NOT NULL,
    buying_power_usd DOUBLE PRECISION NOT NULL,
    cash_usd DOUBLE PRECISION NOT NULL,
    day_pnl_usd DOUBLE PRECISION NOT NULL,
    total_pnl_usd DOUBLE PRECISION NOT NULL,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, captured_at)
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_snapshots_workspace ON founderos_trading_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_trading_positions (
    account_id TEXT NOT NULL DEFAULT 'individual',
    captured_at TIMESTAMPTZ NOT NULL,
    symbol TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    quantity DOUBLE PRECISION NOT NULL,
    avg_cost_usd DOUBLE PRECISION NOT NULL,
    market_value_usd DOUBLE PRECISION NOT NULL,
    unrealized_pnl_usd DOUBLE PRECISION NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, captured_at, symbol)
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_positions_workspace ON founderos_trading_positions(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_trading_orders (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL DEFAULT 'agentic',
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    type TEXT NOT NULL,
    state TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    filled_quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    dollar_amount_usd DOUBLE PRECISION,
    limit_price_usd DOUBLE PRECISION,
    placed_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_orders_workspace ON founderos_trading_orders(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_orders_account ON founderos_trading_orders(account_id);

CREATE TABLE IF NOT EXISTS founderos_trading_analysis (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    at TIMESTAMPTZ NOT NULL,
    account_id TEXT NOT NULL DEFAULT 'agentic',
    agent TEXT NOT NULL,
    examined INTEGER NOT NULL DEFAULT 0,
    signals INTEGER NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    rows JSONB NOT NULL DEFAULT '[]',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_analysis_workspace ON founderos_trading_analysis(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_analysis_account ON founderos_trading_analysis(account_id, at DESC);

CREATE TABLE IF NOT EXISTS founderos_trading_activity (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    at TIMESTAMPTZ NOT NULL,
    account_id TEXT NOT NULL DEFAULT 'individual',
    agent TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('buy', 'sell')),
    symbol TEXT NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price_usd DOUBLE PRECISION NOT NULL,
    rationale TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('filled', 'pending', 'cancelled', 'rejected')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_activity_workspace ON founderos_trading_activity(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_trading_activity_account ON founderos_trading_activity(account_id, at DESC);

-- One row per workspace (replaces the source's id = 1 singleton). autopilot is
-- always imported OFF by the ETL: the bridge never inherits a live trading switch.
CREATE TABLE IF NOT EXISTS founderos_trading_limits (
    workspace_id UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    max_notional_per_trade_usd DOUBLE PRECISION NOT NULL CHECK (max_notional_per_trade_usd > 0),
    max_position_pct_of_sleeve DOUBLE PRECISION NOT NULL CHECK (max_position_pct_of_sleeve > 0 AND max_position_pct_of_sleeve <= 100),
    max_risk_pct_per_trade DOUBLE PRECISION NOT NULL CHECK (max_risk_pct_per_trade > 0 AND max_risk_pct_per_trade <= 100),
    max_concurrent_positions INTEGER NOT NULL CHECK (max_concurrent_positions > 0),
    max_trades_per_day INTEGER NOT NULL CHECK (max_trades_per_day > 0),
    min_sleeve_value_usd DOUBLE PRECISION NOT NULL CHECK (min_sleeve_value_usd > 0),
    max_deployed_capital_usd DOUBLE PRECISION NOT NULL CHECK (max_deployed_capital_usd > 0),
    autopilot BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----------------------------------------------------------------------------
-- Analytics, usage and bookkeeping
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_metric_snapshots (
    metric_id TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (metric_id, captured_at)
);
CREATE INDEX IF NOT EXISTS idx_founderos_metric_snapshots_workspace ON founderos_metric_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_usage_snapshots (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_usage_snapshots_workspace ON founderos_usage_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_ollama_snapshots (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_ollama_snapshots_workspace ON founderos_ollama_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_seed_meta (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_seed_meta_workspace ON founderos_seed_meta(workspace_id);

-- ----------------------------------------------------------------------------
-- Side databases: bank.db, ledger.db, paykit.db, slack-bridge state.db
-- ----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS founderos_bank_summaries (
    account TEXT NOT NULL,
    month DATE NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    business TEXT NOT NULL,
    credits_cents BIGINT NOT NULL,
    debits_cents BIGINT NOT NULL,
    net_cents BIGINT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account, month)
);
CREATE INDEX IF NOT EXISTS idx_founderos_bank_summaries_workspace ON founderos_bank_summaries(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_ledger_rows (
    hash TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    description TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    category TEXT NOT NULL,
    card TEXT NOT NULL DEFAULT 'platinum' CHECK (card IN ('gold', 'platinum', 'blue')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_ledger_rows_workspace ON founderos_ledger_rows(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_ledger_rows_card_date ON founderos_ledger_rows(card, date);

CREATE TABLE IF NOT EXISTS founderos_paykit_customer_snapshots (
    account TEXT NOT NULL,
    captured_on DATE NOT NULL,
    customer_id TEXT NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    total_spent_cents BIGINT NOT NULL,
    total_transactions INTEGER NOT NULL,
    last_transaction_date TIMESTAMPTZ,
    source TEXT NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account, captured_on, customer_id)
);
CREATE INDEX IF NOT EXISTS idx_founderos_paykit_customer_snapshots_workspace ON founderos_paykit_customer_snapshots(workspace_id);

CREATE TABLE IF NOT EXISTS founderos_slack_bridge_jobs (
    id TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    seq BIGINT NOT NULL,
    payload JSONB NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('queued', 'processing', 'done', 'uncertain')),
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_jobs_workspace ON founderos_slack_bridge_jobs(workspace_id);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_jobs_state ON founderos_slack_bridge_jobs(state, seq);

CREATE TABLE IF NOT EXISTS founderos_slack_bridge_sessions (
    key TEXT PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_founderos_slack_bridge_sessions_workspace ON founderos_slack_bridge_sessions(workspace_id);
