defmodule OptimalEngine.Store.Migrations do
  @moduledoc """
  Versioned schema migrations for the Optimal Engine.

  Each migration is `{version, description, ddl_statements}` where
  `ddl_statements` is a list of `{label, sql}` pairs. Migrations run in order,
  tracked in the `schema_migrations` table. Idempotent: already-applied
  versions are skipped; statements that would duplicate (columns, indexes)
  are recognized and tolerated via `safe_execute/3`.

  ## When to add a migration

  - Schema change (new table, new column, new index, new trigger).
  - Data backfill that must run once per database.

  ## When NOT to add a migration

  - Pure code change with no schema impact.
  - Data operations that can run on-demand from a Mix task.

  ## Contract with `Store.open_and_migrate/1`

  `Store.open_and_migrate/1` creates its baseline tables (contexts, entities,
  edges, decisions, sessions, vectors, observations), then calls
  `Migrations.run/1` which applies anything this module tracks. Baseline
  DDL stays inline in `Store` for minimal disruption; additive work
  (Phase 1 onward) lives here.
  """

  require Logger

  @type migration :: {
          version :: pos_integer(),
          description :: String.t(),
          statements :: [{label :: String.t(), sql :: String.t()}]
        }

  # ---------------------------------------------------------------------------
  # Migration registry
  # ---------------------------------------------------------------------------

  @doc """
  Returns all migrations in ascending-version order. Add new migrations at
  the end; never rewrite an existing version.
  """
  @spec all() :: [migration()]
  def all do
    [
      migration_001_schema_migrations_table(),
      migration_002_tenancy(),
      migration_003_identity(),
      migration_004_acls(),
      migration_005_chunks_classifications_intents(),
      migration_006_assets(),
      migration_007_clusters(),
      migration_008_wiki(),
      migration_009_connectors(),
      migration_010_retention_legal_hold(),
      migration_011_audiences(),
      migration_012_events(),
      migration_013_tenant_id_on_existing_tables(),
      migration_014_tenant_first_indexes(),
      migration_015_default_tenant_seed(),
      migration_016_backfill_document_chunks(),
      migration_017_workspace_nodes(),
      migration_018_node_members(),
      migration_019_skills(),
      migration_020_principal_skills(),
      migration_021_workspace_indexes(),
      migration_022_backfill_nodes_from_contexts(),
      migration_023_chunk_embeddings(),
      migration_024_compliance_columns(),
      migration_025_data_architectures(),
      migration_026_workspaces(),
      migration_027_surfacing(),
      migration_028_memories(),
      migration_029_memory_content_hash(),
      migration_030_api_keys(),
      migration_031_memories_fts(),
      migration_032_memory_core_spine(),
      migration_033_workspace_topology_surface_spine(),
      migration_034_tool_model_governance_runs(),
      migration_035_asset_governance(),
      migration_036_asset_adapter_runs(),
      migration_037_asset_extraction_projections(),
      migration_038_evaluation_records(),
      migration_039_connector_workspace_and_legacy_node_renames(),
      migration_040_asset_governance_backfill(),
      migration_041_episodes(),
      migration_042_organizations(),
      migration_043_repair_workspace_organization_ownership(),
      migration_044_reconcile_organization_schema(),
      migration_045_customer_node_type(),
      migration_046_operational_stores(),
      migration_047_model_adaptation_store(),
      migration_048_repair_chunk_workspace_scope(),
      migration_049_rebuild_contexts_fts_triggers(),
      migration_050_retire_test_storage_fixtures(),
      migration_051_workspace_storage_policies(),
      migration_052_backfill_workspace_storage_policies()
    ]
  end

  # ---------------------------------------------------------------------------
  # Runner
  # ---------------------------------------------------------------------------

  @doc """
  Applies any migrations with `version > max(schema_migrations.version)`.
  Records each successful migration in the `schema_migrations` table.
  """
  @spec run(any()) :: :ok
  def run(db) do
    ensure_migrations_table!(db)
    applied = applied_versions(db)

    pending =
      all()
      |> Enum.reject(fn {version, _desc, _stmts} -> MapSet.member?(applied, version) end)

    Enum.each(pending, fn {version, description, statements} ->
      Logger.info("[Migrations] Applying #{pad(version)} — #{description}")
      Enum.each(statements, fn {label, sql} -> safe_execute(db, label, sql) end)
      record_migration!(db, version, description)
    end)

    :ok
  end

  @doc """
  Returns the set of applied migration versions.
  """
  @spec applied_versions(any()) :: MapSet.t(pos_integer())
  def applied_versions(db) do
    {:ok, stmt} = Exqlite.Sqlite3.prepare(db, "SELECT version FROM schema_migrations")

    versions = collect_versions(db, stmt, [])
    Exqlite.Sqlite3.release(db, stmt)
    MapSet.new(versions)
  rescue
    _ -> MapSet.new()
  end

  # ---------------------------------------------------------------------------
  # Private — migrations
  # ---------------------------------------------------------------------------

  defp migration_001_schema_migrations_table do
    {1, "schema_migrations tracking table",
     [
       {"schema_migrations",
        """
        CREATE TABLE IF NOT EXISTS schema_migrations (
          version INTEGER PRIMARY KEY,
          applied_at TEXT NOT NULL DEFAULT (datetime('now')),
          description TEXT
        )
        """}
     ]}
  end

  defp migration_002_tenancy do
    {2, "tenants",
     [
       {"tenants",
        """
        CREATE TABLE IF NOT EXISTS tenants (
          id TEXT PRIMARY KEY,
          name TEXT NOT NULL,
          plan TEXT NOT NULL DEFAULT 'default',
          region TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """}
     ]}
  end

  defp migration_003_identity do
    {3, "principals, groups, roles, role_grants",
     [
       {"principals",
        """
        CREATE TABLE IF NOT EXISTS principals (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          kind TEXT NOT NULL,
          display_name TEXT NOT NULL,
          external_id TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(tenant_id, external_id)
        )
        """},
       {"groups",
        """
        CREATE TABLE IF NOT EXISTS groups (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          name TEXT NOT NULL,
          source TEXT NOT NULL DEFAULT 'local',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(tenant_id, name)
        )
        """},
       {"principal_groups",
        """
        CREATE TABLE IF NOT EXISTS principal_groups (
          principal_id TEXT NOT NULL REFERENCES principals(id) ON DELETE CASCADE,
          group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
          PRIMARY KEY (principal_id, group_id)
        )
        """},
       {"roles",
        """
        CREATE TABLE IF NOT EXISTS roles (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          name TEXT NOT NULL,
          description TEXT,
          UNIQUE(tenant_id, name)
        )
        """},
       {"role_grants",
        """
        CREATE TABLE IF NOT EXISTS role_grants (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          principal_id TEXT REFERENCES principals(id) ON DELETE CASCADE,
          group_id TEXT REFERENCES groups(id) ON DELETE CASCADE,
          role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
          granted_at TEXT NOT NULL DEFAULT (datetime('now')),
          CHECK ((principal_id IS NOT NULL) <> (group_id IS NOT NULL))
        )
        """}
     ]}
  end

  defp migration_004_acls do
    {4, "acls",
     [
       {"acls",
        """
        CREATE TABLE IF NOT EXISTS acls (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          resource_uri TEXT NOT NULL,
          principal_id TEXT,
          group_id TEXT,
          permission TEXT NOT NULL,
          granted_at TEXT NOT NULL DEFAULT (datetime('now')),
          CHECK ((principal_id IS NOT NULL) <> (group_id IS NOT NULL))
        )
        """}
     ]}
  end

  defp migration_005_chunks_classifications_intents do
    {5, "chunks + classifications + intents",
     [
       {"chunks",
        """
        CREATE TABLE IF NOT EXISTS chunks (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          signal_id TEXT NOT NULL,
          parent_id TEXT,
          scale TEXT NOT NULL,
          offset_bytes INTEGER NOT NULL DEFAULT 0,
          length_bytes INTEGER NOT NULL DEFAULT 0,
          text TEXT NOT NULL DEFAULT '',
          modality TEXT NOT NULL DEFAULT 'text',
          asset_ref TEXT,
          classification_level TEXT NOT NULL DEFAULT 'internal',
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"classifications",
        """
        CREATE TABLE IF NOT EXISTS classifications (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          chunk_id TEXT NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
          mode TEXT,
          genre TEXT,
          signal_type TEXT,
          format TEXT,
          structure TEXT,
          sn_ratio REAL,
          confidence REAL,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(chunk_id)
        )
        """},
       {"intents",
        """
        CREATE TABLE IF NOT EXISTS intents (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          chunk_id TEXT NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
          intent TEXT NOT NULL,
          confidence REAL NOT NULL,
          evidence TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(chunk_id)
        )
        """}
     ]}
  end

  defp migration_006_assets do
    {6, "assets",
     [
       {"assets",
        """
        CREATE TABLE IF NOT EXISTS assets (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          content_type TEXT NOT NULL,
          size_bytes INTEGER NOT NULL,
          storage_path TEXT NOT NULL,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """}
     ]}
  end

  defp migration_007_clusters do
    {7, "clusters + cluster_members",
     [
       {"clusters",
        """
        CREATE TABLE IF NOT EXISTS clusters (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          theme TEXT NOT NULL,
          intent_dominant TEXT,
          member_count INTEGER NOT NULL DEFAULT 0,
          centroid BLOB,
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"cluster_members",
        """
        CREATE TABLE IF NOT EXISTS cluster_members (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          cluster_id TEXT NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
          chunk_id TEXT NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
          weight REAL NOT NULL DEFAULT 1.0,
          UNIQUE(cluster_id, chunk_id)
        )
        """}
     ]}
  end

  defp migration_008_wiki do
    {8, "wiki_pages + citations",
     [
       {"wiki_pages",
        """
        CREATE TABLE IF NOT EXISTS wiki_pages (
          tenant_id TEXT NOT NULL,
          slug TEXT NOT NULL,
          audience TEXT NOT NULL DEFAULT 'default',
          version INTEGER NOT NULL DEFAULT 1,
          frontmatter TEXT NOT NULL DEFAULT '{}',
          body TEXT NOT NULL,
          last_curated TEXT NOT NULL DEFAULT (datetime('now')),
          curated_by TEXT,
          PRIMARY KEY (tenant_id, slug, audience, version)
        )
        """},
       {"citations",
        """
        CREATE TABLE IF NOT EXISTS citations (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          wiki_slug TEXT NOT NULL,
          wiki_audience TEXT NOT NULL DEFAULT 'default',
          chunk_id TEXT NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
          claim_hash TEXT NOT NULL,
          last_verified TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """}
     ]}
  end

  defp migration_009_connectors do
    {9, "connectors + connector_runs",
     [
       {"connectors",
        """
        CREATE TABLE IF NOT EXISTS connectors (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          kind TEXT NOT NULL,
          config TEXT NOT NULL DEFAULT '{}',
          cursor TEXT,
          enabled INTEGER NOT NULL DEFAULT 1,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"connector_runs",
        """
        CREATE TABLE IF NOT EXISTS connector_runs (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          connector_id TEXT NOT NULL REFERENCES connectors(id) ON DELETE CASCADE,
          started_at TEXT NOT NULL DEFAULT (datetime('now')),
          completed_at TEXT,
          signals_ingested INTEGER NOT NULL DEFAULT 0,
          errors_encountered INTEGER NOT NULL DEFAULT 0,
          cursor_before TEXT,
          cursor_after TEXT,
          status TEXT NOT NULL DEFAULT 'running',
          error_detail TEXT
        )
        """}
     ]}
  end

  defp migration_010_retention_legal_hold do
    {10, "retention_policies + legal_holds",
     [
       {"retention_policies",
        """
        CREATE TABLE IF NOT EXISTS retention_policies (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          scope_type TEXT NOT NULL,
          scope_value TEXT,
          ttl_days INTEGER,
          action TEXT NOT NULL DEFAULT 'archive',
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"legal_holds",
        """
        CREATE TABLE IF NOT EXISTS legal_holds (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          signal_id TEXT NOT NULL,
          held_by TEXT NOT NULL,
          reason TEXT NOT NULL,
          placed_at TEXT NOT NULL DEFAULT (datetime('now')),
          released_at TEXT
        )
        """}
     ]}
  end

  defp migration_011_audiences do
    {11, "audiences",
     [
       {"audiences",
        """
        CREATE TABLE IF NOT EXISTS audiences (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          name TEXT NOT NULL,
          role_ids TEXT NOT NULL DEFAULT '[]',
          description TEXT,
          UNIQUE(tenant_id, name)
        )
        """}
     ]}
  end

  defp migration_012_events do
    {12, "events (append-only audit log)",
     [
       {"events",
        """
        CREATE TABLE IF NOT EXISTS events (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          ts TEXT NOT NULL DEFAULT (datetime('now')),
          principal TEXT NOT NULL,
          kind TEXT NOT NULL,
          target_uri TEXT,
          latency_ms INTEGER,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """}
     ]}
  end

  defp migration_013_tenant_id_on_existing_tables do
    {13, "add tenant_id to existing primary tables",
     [
       {"contexts.tenant_id",
        "ALTER TABLE contexts ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"entities.tenant_id",
        "ALTER TABLE entities ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"edges.tenant_id",
        "ALTER TABLE edges ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"decisions.tenant_id",
        "ALTER TABLE decisions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"sessions.tenant_id",
        "ALTER TABLE sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"vectors.tenant_id",
        "ALTER TABLE vectors ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"},
       {"observations.tenant_id",
        "ALTER TABLE observations ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'"}
     ]}
  end

  defp migration_014_tenant_first_indexes do
    {14, "tenant-first indexes on every primary table",
     [
       {"idx_contexts_tenant_node",
        "CREATE INDEX IF NOT EXISTS idx_contexts_tenant_node ON contexts(tenant_id, node)"},
       {"idx_contexts_tenant_type",
        "CREATE INDEX IF NOT EXISTS idx_contexts_tenant_type ON contexts(tenant_id, type)"},
       {"idx_chunks_tenant_scale",
        "CREATE INDEX IF NOT EXISTS idx_chunks_tenant_scale ON chunks(tenant_id, scale)"},
       {"idx_chunks_tenant_signal",
        "CREATE INDEX IF NOT EXISTS idx_chunks_tenant_signal ON chunks(tenant_id, signal_id)"},
       {"idx_chunks_parent", "CREATE INDEX IF NOT EXISTS idx_chunks_parent ON chunks(parent_id)"},
       {"idx_classifications_tenant_chunk",
        "CREATE INDEX IF NOT EXISTS idx_classifications_tenant_chunk ON classifications(tenant_id, chunk_id)"},
       {"idx_intents_tenant_chunk",
        "CREATE INDEX IF NOT EXISTS idx_intents_tenant_chunk ON intents(tenant_id, chunk_id)"},
       {"idx_cluster_members_tenant_cluster",
        "CREATE INDEX IF NOT EXISTS idx_cluster_members_tenant_cluster ON cluster_members(tenant_id, cluster_id)"},
       {"idx_cluster_members_tenant_chunk",
        "CREATE INDEX IF NOT EXISTS idx_cluster_members_tenant_chunk ON cluster_members(tenant_id, chunk_id)"},
       {"idx_wiki_pages_tenant_slug",
        "CREATE INDEX IF NOT EXISTS idx_wiki_pages_tenant_slug ON wiki_pages(tenant_id, slug)"},
       {"idx_citations_tenant_slug",
        "CREATE INDEX IF NOT EXISTS idx_citations_tenant_slug ON citations(tenant_id, wiki_slug)"},
       {"idx_citations_tenant_chunk",
        "CREATE INDEX IF NOT EXISTS idx_citations_tenant_chunk ON citations(tenant_id, chunk_id)"},
       {"idx_events_tenant_ts",
        "CREATE INDEX IF NOT EXISTS idx_events_tenant_ts ON events(tenant_id, ts)"},
       {"idx_events_tenant_principal",
        "CREATE INDEX IF NOT EXISTS idx_events_tenant_principal ON events(tenant_id, principal, ts)"},
       {"idx_events_tenant_kind",
        "CREATE INDEX IF NOT EXISTS idx_events_tenant_kind ON events(tenant_id, kind, ts)"},
       {"idx_acls_tenant_resource",
        "CREATE INDEX IF NOT EXISTS idx_acls_tenant_resource ON acls(tenant_id, resource_uri)"},
       {"idx_acls_tenant_principal",
        "CREATE INDEX IF NOT EXISTS idx_acls_tenant_principal ON acls(tenant_id, principal_id)"},
       {"idx_acls_tenant_group",
        "CREATE INDEX IF NOT EXISTS idx_acls_tenant_group ON acls(tenant_id, group_id)"},
       {"idx_principal_groups_principal",
        "CREATE INDEX IF NOT EXISTS idx_principal_groups_principal ON principal_groups(principal_id)"},
       {"idx_principal_groups_group",
        "CREATE INDEX IF NOT EXISTS idx_principal_groups_group ON principal_groups(group_id)"},
       {"idx_role_grants_tenant_principal",
        "CREATE INDEX IF NOT EXISTS idx_role_grants_tenant_principal ON role_grants(tenant_id, principal_id)"},
       {"idx_role_grants_tenant_group",
        "CREATE INDEX IF NOT EXISTS idx_role_grants_tenant_group ON role_grants(tenant_id, group_id)"},
       {"idx_connector_runs_connector",
        "CREATE INDEX IF NOT EXISTS idx_connector_runs_connector ON connector_runs(connector_id, started_at)"},
       {"idx_legal_holds_tenant_signal",
        "CREATE INDEX IF NOT EXISTS idx_legal_holds_tenant_signal ON legal_holds(tenant_id, signal_id)"}
     ]}
  end

  defp migration_015_default_tenant_seed do
    {15, "seed the default tenant",
     [
       {"default_tenant",
        """
        INSERT OR IGNORE INTO tenants (id, name, plan)
        VALUES ('default', 'Default Tenant', 'default')
        """}
     ]}
  end

  defp migration_017_workspace_nodes do
    {17, "workspace: nodes (organizational units)",
     [
       {"nodes",
        """
        CREATE TABLE IF NOT EXISTS nodes (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          kind TEXT NOT NULL,
          parent_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
          description TEXT,
          style TEXT NOT NULL DEFAULT 'internal',
          status TEXT NOT NULL DEFAULT 'active',
          path TEXT NOT NULL DEFAULT '',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(tenant_id, slug)
        )
        """}
     ]}
  end

  defp migration_018_node_members do
    {18, "workspace: node_members (principal ↔ node with internal/external/owner/observer)",
     [
       {"node_members",
        """
        CREATE TABLE IF NOT EXISTS node_members (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
          principal_id TEXT NOT NULL REFERENCES principals(id) ON DELETE CASCADE,
          membership TEXT NOT NULL DEFAULT 'internal',
          role TEXT,
          started_at TEXT NOT NULL DEFAULT (datetime('now')),
          ended_at TEXT,
          UNIQUE(node_id, principal_id, membership)
        )
        """}
     ]}
  end

  defp migration_019_skills do
    {19, "workspace: skills (capability registry)",
     [
       {"skills",
        """
        CREATE TABLE IF NOT EXISTS skills (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          name TEXT NOT NULL,
          kind TEXT,
          description TEXT,
          UNIQUE(tenant_id, name)
        )
        """}
     ]}
  end

  defp migration_020_principal_skills do
    {20, "workspace: principal_skills (many-to-many capability grants)",
     [
       {"principal_skills",
        """
        CREATE TABLE IF NOT EXISTS principal_skills (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          principal_id TEXT NOT NULL REFERENCES principals(id) ON DELETE CASCADE,
          skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
          level TEXT NOT NULL DEFAULT 'intermediate',
          evidence TEXT,
          acquired_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(principal_id, skill_id)
        )
        """}
     ]}
  end

  defp migration_021_workspace_indexes do
    {21, "workspace: tenant-first indexes",
     [
       {"idx_nodes_tenant_kind",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_kind ON nodes(tenant_id, kind)"},
       {"idx_nodes_tenant_parent",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_parent ON nodes(tenant_id, parent_id)"},
       {"idx_nodes_tenant_style",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_style ON nodes(tenant_id, style)"},
       {"idx_node_members_tenant_node",
        "CREATE INDEX IF NOT EXISTS idx_node_members_tenant_node ON node_members(tenant_id, node_id)"},
       {"idx_node_members_tenant_principal",
        "CREATE INDEX IF NOT EXISTS idx_node_members_tenant_principal ON node_members(tenant_id, principal_id)"},
       {"idx_skills_tenant_kind",
        "CREATE INDEX IF NOT EXISTS idx_skills_tenant_kind ON skills(tenant_id, kind)"},
       {"idx_principal_skills_tenant_principal",
        "CREATE INDEX IF NOT EXISTS idx_principal_skills_tenant_principal ON principal_skills(tenant_id, principal_id)"},
       {"idx_principal_skills_tenant_skill",
        "CREATE INDEX IF NOT EXISTS idx_principal_skills_tenant_skill ON principal_skills(tenant_id, skill_id)"}
     ]}
  end

  # Backfill: every distinct `contexts.node` value becomes a nodes row so
  # existing routing / retrieval paths continue to resolve, with the node
  # now first-class and upgradable (kind / style / memberships can evolve).
  defp migration_022_backfill_nodes_from_contexts do
    {22, "workspace: backfill nodes rows from distinct contexts.node values",
     [
       {"backfill_nodes",
        """
        INSERT OR IGNORE INTO nodes (id, tenant_id, slug, name, kind, style, status, path)
        SELECT
          COALESCE(c.tenant_id, 'default') || ':' || c.node AS id,
          COALESCE(c.tenant_id, 'default')                  AS tenant_id,
          c.node                                            AS slug,
          c.node                                            AS name,
          'domain'                                          AS kind,
          'internal'                                        AS style,
          'active'                                          AS status,
          'nodes/' || c.node                                AS path
        FROM (SELECT DISTINCT tenant_id, node FROM contexts WHERE node IS NOT NULL AND node <> '') c
        """}
     ]}
  end

  # Phase 5: chunk-level embeddings in the aligned 768-dim nomic space.
  # Keyed on chunk_id so re-embedding overwrites in place. `modality` is the
  # original chunk modality (text/image/audio/code/data/mixed) so a single
  # query can filter to a subset of modalities if needed.
  defp migration_025_data_architectures do
    {25, "data-architectures — model-agnostic data-point schemas (Phase 14)",
     [
       {"architectures",
        """
        CREATE TABLE IF NOT EXISTS architectures (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          name TEXT NOT NULL,
          version INTEGER NOT NULL DEFAULT 1,
          description TEXT,
          modality_primary TEXT NOT NULL DEFAULT 'text',
          spec TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(tenant_id, name, version)
        )
        """},
       {"idx_architectures_tenant",
        "CREATE INDEX IF NOT EXISTS idx_architectures_tenant ON architectures(tenant_id, name)"},
       {"contexts.architecture_id", "ALTER TABLE contexts ADD COLUMN architecture_id TEXT"},
       {"idx_contexts_architecture",
        "CREATE INDEX IF NOT EXISTS idx_contexts_architecture ON contexts(tenant_id, architecture_id)"},
       {"processor_runs",
        """
        CREATE TABLE IF NOT EXISTS processor_runs (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          context_id TEXT NOT NULL,
          architecture_id TEXT NOT NULL,
          processor TEXT NOT NULL,
          field TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'pending',
          started_at TEXT NOT NULL DEFAULT (datetime('now')),
          completed_at TEXT,
          output_ref TEXT,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_processor_runs_context",
        "CREATE INDEX IF NOT EXISTS idx_processor_runs_context ON processor_runs(context_id, processor)"}
     ]}
  end

  defp migration_024_compliance_columns do
    {24, "compliance — contexts.created_by + archived_at for Phase 11",
     [
       {"contexts.created_by", "ALTER TABLE contexts ADD COLUMN created_by TEXT"},
       {"contexts.archived_at", "ALTER TABLE contexts ADD COLUMN archived_at TEXT"},
       {"idx_contexts_tenant_created_by",
        "CREATE INDEX IF NOT EXISTS idx_contexts_tenant_created_by ON contexts(tenant_id, created_by)"}
     ]}
  end

  # Phase 1.5 — workspaces. A workspace is a knowledge base inside an
  # organization (tenant). One tenant can hold many workspaces. Every
  # signal-bearing row gets a workspace_id; existing rows backfill to
  # `<tenant>:default`. Unlike tenant_id (the absolute isolation boundary),
  # workspace_id is a soft scope: a principal can belong to multiple
  # workspaces within their tenant via workspace_members.
  defp migration_026_workspaces do
    {26, "workspaces — multiple knowledge bases per organization (Phase 1.5)",
     [
       {"workspaces",
        """
        CREATE TABLE IF NOT EXISTS workspaces (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          description TEXT,
          status TEXT NOT NULL DEFAULT 'active',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          archived_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(tenant_id, slug)
        )
        """},
       {"idx_workspaces_tenant_status",
        "CREATE INDEX IF NOT EXISTS idx_workspaces_tenant_status ON workspaces(tenant_id, status)"},

       # Membership: principal × workspace × role. A principal in a tenant
       # can be granted access to N workspaces. Role values: owner / member /
       # viewer. Time-bounded via started_at / ended_at.
       {"workspace_members",
        """
        CREATE TABLE IF NOT EXISTS workspace_members (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
          principal_id TEXT NOT NULL REFERENCES principals(id) ON DELETE CASCADE,
          role TEXT NOT NULL DEFAULT 'member',
          started_at TEXT NOT NULL DEFAULT (datetime('now')),
          ended_at TEXT,
          UNIQUE(workspace_id, principal_id)
        )
        """},
       {"idx_workspace_members_tenant_workspace",
        "CREATE INDEX IF NOT EXISTS idx_workspace_members_tenant_workspace ON workspace_members(tenant_id, workspace_id)"},
       {"idx_workspace_members_principal",
        "CREATE INDEX IF NOT EXISTS idx_workspace_members_principal ON workspace_members(principal_id)"},

       # Add workspace_id to every signal-bearing table. Defaults to
       # 'default' so existing rows continue resolving without rewrite.
       {"contexts.workspace_id",
        "ALTER TABLE contexts ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"chunks.workspace_id",
        "ALTER TABLE chunks ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"classifications.workspace_id",
        "ALTER TABLE classifications ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"intents.workspace_id",
        "ALTER TABLE intents ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"entities.workspace_id",
        "ALTER TABLE entities ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"edges.workspace_id",
        "ALTER TABLE edges ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"vectors.workspace_id",
        "ALTER TABLE vectors ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"chunk_embeddings.workspace_id",
        "ALTER TABLE chunk_embeddings ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"clusters.workspace_id",
        "ALTER TABLE clusters ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"cluster_members.workspace_id",
        "ALTER TABLE cluster_members ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"assets.workspace_id",
        "ALTER TABLE assets ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"wiki_pages.workspace_id",
        "ALTER TABLE wiki_pages ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"citations.workspace_id",
        "ALTER TABLE citations ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"events.workspace_id",
        "ALTER TABLE events ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"nodes.workspace_id",
        "ALTER TABLE nodes ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"node_members.workspace_id",
        "ALTER TABLE node_members ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"skills.workspace_id",
        "ALTER TABLE skills ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"principal_skills.workspace_id",
        "ALTER TABLE principal_skills ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"decisions.workspace_id",
        "ALTER TABLE decisions ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"sessions.workspace_id",
        "ALTER TABLE sessions ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"observations.workspace_id",
        "ALTER TABLE observations ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"architectures.workspace_id",
        "ALTER TABLE architectures ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"processor_runs.workspace_id",
        "ALTER TABLE processor_runs ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},

       # Workspace-first indexes on the highest-traffic tables.
       {"idx_contexts_ws_node",
        "CREATE INDEX IF NOT EXISTS idx_contexts_ws_node ON contexts(workspace_id, node)"},
       {"idx_chunks_ws_signal",
        "CREATE INDEX IF NOT EXISTS idx_chunks_ws_signal ON chunks(workspace_id, signal_id)"},
       {"idx_chunks_ws_scale",
        "CREATE INDEX IF NOT EXISTS idx_chunks_ws_scale ON chunks(workspace_id, scale)"},
       {"idx_wiki_pages_ws_slug",
        "CREATE INDEX IF NOT EXISTS idx_wiki_pages_ws_slug ON wiki_pages(workspace_id, slug)"},
       {"idx_events_ws_ts",
        "CREATE INDEX IF NOT EXISTS idx_events_ws_ts ON events(workspace_id, ts)"},
       {"idx_nodes_ws_kind",
        "CREATE INDEX IF NOT EXISTS idx_nodes_ws_kind ON nodes(workspace_id, kind)"},
       {"idx_chunk_embeddings_ws_modality",
        "CREATE INDEX IF NOT EXISTS idx_chunk_embeddings_ws_modality ON chunk_embeddings(workspace_id, modality)"},

       # Seed: a singleton `default` workspace under the default tenant so the
       # soft backfill (`workspace_id = 'default'`) resolves to a real row.
       # Future tenants create their own workspaces at tenant-creation time —
       # this migration is only responsible for the existing default tenant.
       {"seed_default_workspace",
        """
        INSERT OR IGNORE INTO workspaces (id, tenant_id, slug, name, description, status)
        VALUES (
          'default', 'default', 'default', 'Default workspace',
          'Auto-created so existing rows have a workspace.', 'active'
        )
        """}
     ]}
  end

  # Phase 15 — proactive surfacing. Subscriptions describe what an agent
  # wants pushed to them; events log what got pushed. Categories follow
  # Engramme's "Questions in the Wild" taxonomy (Mar 2026), reframed for
  # enterprise: recent_actions, contacts, schedules, ownership, file_loc,
  # procedures, professional_knowledge, factual, etc. Stored as JSON
  # array in `categories` so the taxonomy can evolve without migration.
  defp migration_027_surfacing do
    {27, "surfacing — subscriptions + event log for proactive recall (Phase 15)",
     [
       {"surfacing_subscriptions",
        """
        CREATE TABLE IF NOT EXISTS surfacing_subscriptions (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          principal_id TEXT,
          scope TEXT NOT NULL DEFAULT 'workspace',
          scope_value TEXT,
          categories TEXT NOT NULL DEFAULT '[]',
          activity TEXT,
          status TEXT NOT NULL DEFAULT 'active',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          paused_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_surfacing_subs_workspace_status",
        "CREATE INDEX IF NOT EXISTS idx_surfacing_subs_workspace_status ON surfacing_subscriptions(workspace_id, status)"},
       {"idx_surfacing_subs_principal",
        "CREATE INDEX IF NOT EXISTS idx_surfacing_subs_principal ON surfacing_subscriptions(principal_id)"},
       {"surfacing_events",
        """
        CREATE TABLE IF NOT EXISTS surfacing_events (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          subscription_id TEXT NOT NULL,
          trigger TEXT NOT NULL,
          envelope_slug TEXT,
          envelope_kind TEXT,
          category TEXT,
          score REAL,
          pushed_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_surfacing_events_workspace_pushed_at",
        "CREATE INDEX IF NOT EXISTS idx_surfacing_events_workspace_pushed_at ON surfacing_events(workspace_id, pushed_at)"},
       {"idx_surfacing_events_subscription",
        "CREATE INDEX IF NOT EXISTS idx_surfacing_events_subscription ON surfacing_events(subscription_id, pushed_at)"},
       {"idx_surfacing_events_dedup",
        "CREATE INDEX IF NOT EXISTS idx_surfacing_events_dedup ON surfacing_events(subscription_id, envelope_slug, pushed_at)"}
     ]}
  end

  defp migration_023_chunk_embeddings do
    {23, "chunk_embeddings — aligned 768-dim per-chunk vectors (Phase 5)",
     [
       {"chunk_embeddings",
        """
        CREATE TABLE IF NOT EXISTS chunk_embeddings (
          chunk_id TEXT PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
          tenant_id TEXT NOT NULL,
          model TEXT NOT NULL,
          modality TEXT NOT NULL,
          dim INTEGER NOT NULL DEFAULT 768,
          vector BLOB NOT NULL,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_chunk_embeddings_tenant_modality",
        "CREATE INDEX IF NOT EXISTS idx_chunk_embeddings_tenant_modality ON chunk_embeddings(tenant_id, modality)"},
       {"idx_chunk_embeddings_tenant_model",
        "CREATE INDEX IF NOT EXISTS idx_chunk_embeddings_tenant_model ON chunk_embeddings(tenant_id, model)"}
     ]}
  end

  # Backfill: for each existing contexts row, create a corresponding
  # :document-scale chunk so retrieval paths that expect chunks have something
  # to read. Chunk id = "{context_id}:doc" (deterministic + idempotent).
  defp migration_016_backfill_document_chunks do
    {16, "backfill :document-scale chunks from existing contexts",
     [
       {"backfill_chunks",
        """
        INSERT OR IGNORE INTO chunks
          (id, tenant_id, signal_id, parent_id, scale, offset_bytes, length_bytes, text, modality, classification_level, created_at)
        SELECT
          c.id || ':doc',
          COALESCE(c.tenant_id, 'default'),
          c.id,
          NULL,
          'document',
          0,
          COALESCE(LENGTH(c.content), 0),
          COALESCE(c.content, ''),
          CASE WHEN COALESCE(c.mode, '') = 'code' THEN 'code' ELSE 'text' END,
          'internal',
          COALESCE(c.created_at, datetime('now'))
        FROM contexts c
        """}
     ]}
  end

  # Phase 16 — first-class versioned memory with relations and soft forgetting.
  # Memories are workspace-scoped, versioned, audience-aware, and can reference
  # each other via typed relations (updates, extends, derives, contradicts, cites).
  # Soft forgetting sets is_forgotten=1 without touching the row; hard delete
  # cascades to memory_relations via ON DELETE CASCADE.
  defp migration_028_memories do
    {28, "memories — versioned memory primitive with relations (Phase 16)",
     [
       {"memories",
        """
        CREATE TABLE IF NOT EXISTS memories (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          content TEXT NOT NULL,
          is_static INTEGER NOT NULL DEFAULT 0,
          is_forgotten INTEGER NOT NULL DEFAULT 0,
          forget_after TEXT,
          forget_reason TEXT,
          version INTEGER NOT NULL DEFAULT 1,
          parent_memory_id TEXT REFERENCES memories(id),
          root_memory_id TEXT,
          is_latest INTEGER NOT NULL DEFAULT 1,
          citation_uri TEXT,
          source_chunk_id TEXT,
          audience TEXT NOT NULL DEFAULT 'default',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"memory_relations",
        """
        CREATE TABLE IF NOT EXISTS memory_relations (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          source_memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
          target_memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
          relation TEXT NOT NULL,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(source_memory_id, target_memory_id, relation)
        )
        """},
       {"idx_memories_ws_latest_forgotten",
        "CREATE INDEX IF NOT EXISTS idx_memories_ws_latest_forgotten ON memories(workspace_id, is_latest, is_forgotten)"},
       {"idx_memories_ws_audience",
        "CREATE INDEX IF NOT EXISTS idx_memories_ws_audience ON memories(workspace_id, audience)"},
       {"idx_memories_ws_root_version",
        "CREATE INDEX IF NOT EXISTS idx_memories_ws_root_version ON memories(workspace_id, root_memory_id, version)"},
       {"idx_memory_relations_source",
        "CREATE INDEX IF NOT EXISTS idx_memory_relations_source ON memory_relations(source_memory_id)"},
       {"idx_memory_relations_target",
        "CREATE INDEX IF NOT EXISTS idx_memory_relations_target ON memory_relations(target_memory_id)"}
     ]}
  end

  # Phase 17 — content-hash deduplication for memories.
  # Adds `content_hash TEXT` (SHA-256 of trimmed+downcased content) so that
  # `Memory.Versioned.create/1` can detect duplicate writes within the same
  # workspace/audience scope. A partial unique index covering only live
  # (is_forgotten=0, is_latest=1) rows prevents stale or forgotten memories
  # from blocking fresh inserts. SQLite partial indexes use WHERE clauses.
  defp migration_029_memory_content_hash do
    {29, "memories — content_hash column + dedup partial unique index",
     [
       # Step 1: add nullable column so existing rows are not broken.
       {"memories.content_hash", "ALTER TABLE memories ADD COLUMN content_hash TEXT"},

       # Step 2: backfill existing rows.
       # SQLite has no SHA-256 built-in, so we mark rows with a placeholder
       # that is distinct per-row (rowid-based) to preserve uniqueness.
       # Application code will compute real hashes on new writes; old rows
       # are effectively invisible to the dedup check because the SELECT in
       # create/1 filters on content_hash = computed_hash.
       {"backfill_memories_content_hash",
        """
        UPDATE memories
        SET content_hash = 'legacy:' || id
        WHERE content_hash IS NULL
        """},

       # Step 3: partial unique index — only enforced on live memories.
       # SQLite supports partial (filtered) unique indexes via WHERE.
       {"idx_memories_dedup_key",
        """
        CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_dedup_key
        ON memories (workspace_id, audience, content_hash)
        WHERE is_forgotten = 0 AND is_latest = 1
        """}
     ]}
  end

  # Phase 18 — API key authentication (tenant-scoped, workspace-scopeable).
  # Keys are hashed with bcrypt; only the prefix (first 8 chars of the raw secret)
  # is stored in plaintext for UX display. The `oe_<id>_<secret>` token format
  # is parsed and verified at request time by OptimalEngine.Auth.ApiKey.verify/1.
  defp migration_030_api_keys do
    {30, "api_keys — tenant-scoped API key authentication (Phase 18)",
     [
       {"api_keys",
        """
        CREATE TABLE IF NOT EXISTS api_keys (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          principal_id TEXT REFERENCES principals(id) ON DELETE SET NULL,
          hashed_secret TEXT NOT NULL,
          prefix TEXT NOT NULL,
          name TEXT NOT NULL,
          scopes TEXT NOT NULL DEFAULT '["*"]',
          workspace_scope TEXT NOT NULL DEFAULT '["*"]',
          expires_at TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          last_used_at TEXT,
          revoked_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_api_keys_tenant_revoked",
        "CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_revoked ON api_keys(tenant_id, revoked_at)"},
       {"idx_api_keys_prefix", "CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix)"}
     ]}
  end

  # Phase 19 — FTS5 full-text search index for the versioned memories table.
  # Uses trigger-based sync (same pattern as contexts_fts) so the index
  # stays in sync automatically on INSERT, UPDATE, and DELETE without any
  # application-level maintenance.
  #
  # Why triggers instead of `content=` external content?
  # The `content=` mode requires memories.rowid == FTS docid. Our
  # memories.id is TEXT (UUID), so the hidden SQLite rowid is the reliable
  # integer anchor. Triggers give us explicit control with the same rowid.
  defp migration_031_memories_fts do
    {31, "memories_fts — FTS5 full-text search index for versioned memories (Phase 19)",
     [
       # Step 1: FTS5 virtual table — porter+unicode61 tokenizer for stemming.
       {"memories_fts",
        """
        CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
          content,
          tokenize='porter unicode61'
        )
        """},

       # Step 2: INSERT trigger — syncs new rows into the FTS index.
       {"memories_fts_insert",
        """
        CREATE TRIGGER IF NOT EXISTS memories_fts_insert
        AFTER INSERT ON memories BEGIN
          INSERT INTO memories_fts(rowid, content)
          VALUES (new.rowid, new.content);
        END
        """},

       # Note: no UPDATE trigger. In the memories model, content never changes
       # in-place — version bumps create NEW rows via INSERT (handled by the
       # INSERT trigger above). Metadata-only UPDATEs (is_forgotten, is_latest,
       # updated_at) don't affect search relevance and require no FTS sync.
       # Adding an UPDATE trigger with the FTS5 'delete' command would silently
       # roll back the enclosing UPDATE when executed via Exqlite prepared
       # statements — a known SQLite FTS5 trigger interaction issue.

       # Step 3: DELETE trigger — removes the FTS doc on hard delete.
       # Uses standard DELETE FROM on the FTS virtual table, which is safe
       # inside AFTER DELETE triggers (avoids the FTS5 'delete' command).
       {"memories_fts_delete",
        """
        CREATE TRIGGER IF NOT EXISTS memories_fts_delete
        AFTER DELETE ON memories BEGIN
          DELETE FROM memories_fts WHERE rowid = old.rowid;
        END
        """},

       # Step 5: Backfill — seed all existing memory rows into the FTS index.
       # On a fresh database this inserts zero rows (no-op). On upgrade it
       # indexes all historical memories so search works immediately.
       {"memories_fts_backfill",
        """
        INSERT INTO memories_fts(rowid, content)
        SELECT rowid, content FROM memories
        """}
     ]}
  end

  # Phase 20 — Memory Core spine.
  # These are the governed object tables that let the engine evolve from
  # file/context storage into source-linked governed memory. The existing
  # `contexts` table remains the compatibility/search surface while these
  # tables own provenance, claims, facts, workflows, pools, and tool/model
  # governance.
  defp migration_032_memory_core_spine do
    {32, "memory_core — governed source/claim/fact/memory spine",
     [
       {"source_packages",
        """
        CREATE TABLE IF NOT EXISTS source_packages (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          source_type TEXT NOT NULL DEFAULT 'manual',
          source_class TEXT NOT NULL DEFAULT 'text',
          source_system TEXT,
          source_uri TEXT,
          source_time TEXT,
          received_at TEXT NOT NULL DEFAULT (datetime('now')),
          content_hash TEXT NOT NULL,
          raw_text TEXT NOT NULL DEFAULT '',
          verbatim_archive_uri TEXT,
          trust_label TEXT NOT NULL DEFAULT 'unreviewed',
          retention_class TEXT NOT NULL DEFAULT 'standard',
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          quarantine_state TEXT NOT NULL DEFAULT 'clear',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, content_hash)
        )
        """},
       {"claims",
        """
        CREATE TABLE IF NOT EXISTS claims (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          source_package_id TEXT NOT NULL REFERENCES source_packages(id) ON DELETE CASCADE,
          signal_id TEXT,
          claim_text TEXT NOT NULL,
          claim_type TEXT NOT NULL DEFAULT 'assertion',
          subject_anchor TEXT,
          action_class TEXT,
          object_anchor TEXT,
          semantic_frame TEXT NOT NULL DEFAULT '{}',
          source_span TEXT NOT NULL DEFAULT '{}',
          extraction_run_id TEXT,
          evaluator_id TEXT,
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          raw_component_scores TEXT NOT NULL DEFAULT '{}',
          calibration_dataset_version TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'pending',
          review_status TEXT NOT NULL DEFAULT 'unreviewed',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"facts",
        """
        CREATE TABLE IF NOT EXISTS facts (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          fact_text TEXT NOT NULL,
          fact_type TEXT NOT NULL DEFAULT 'assertion',
          subject_anchor TEXT,
          action_class TEXT,
          object_anchor TEXT,
          scope TEXT NOT NULL DEFAULT '{}',
          accepted_claim_ids TEXT NOT NULL DEFAULT '[]',
          supporting_evidence_links TEXT NOT NULL DEFAULT '[]',
          contradicting_evidence_links TEXT NOT NULL DEFAULT '[]',
          verifier_id TEXT,
          verification_status TEXT NOT NULL DEFAULT 'unverified',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          raw_component_scores TEXT NOT NULL DEFAULT '{}',
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'candidate',
          contradiction_status TEXT NOT NULL DEFAULT 'none',
          event_time TEXT,
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          verification_time TEXT,
          stale_after TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"memory_objects",
        """
        CREATE TABLE IF NOT EXISTS memory_objects (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          memory_type TEXT NOT NULL DEFAULT 'general',
          summary TEXT NOT NULL,
          subject_anchor TEXT,
          action_class TEXT,
          semantic_frame TEXT NOT NULL DEFAULT '{}',
          salience REAL NOT NULL DEFAULT 0.5,
          fact_links TEXT NOT NULL DEFAULT '[]',
          claim_links TEXT NOT NULL DEFAULT '[]',
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'candidate',
          staleness_status TEXT NOT NULL DEFAULT 'current',
          supersession_status TEXT NOT NULL DEFAULT 'none',
          event_time TEXT,
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"memory_detail_objects",
        """
        CREATE TABLE IF NOT EXISTS memory_detail_objects (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          parent_object_type TEXT NOT NULL,
          parent_object_id TEXT NOT NULL,
          detail_type TEXT NOT NULL DEFAULT 'step',
          detail_order INTEGER NOT NULL DEFAULT 0,
          detail_depth INTEGER NOT NULL DEFAULT 0,
          action_class TEXT,
          detail_text TEXT NOT NULL,
          command_or_parameter_value TEXT,
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'candidate',
          reuse_status TEXT NOT NULL DEFAULT 'local',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"relationship_edges",
        """
        CREATE TABLE IF NOT EXISTS relationship_edges (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          from_object_type TEXT NOT NULL,
          from_object_id TEXT NOT NULL,
          to_object_type TEXT NOT NULL,
          to_object_id TEXT NOT NULL,
          relationship_type TEXT NOT NULL,
          confidence REAL NOT NULL DEFAULT 0.5,
          precision_score REAL NOT NULL DEFAULT 0.5,
          evidence_links TEXT NOT NULL DEFAULT '[]',
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'current',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, from_object_type, from_object_id, to_object_type, to_object_id, relationship_type)
        )
        """},
       {"derivation_ledger",
        """
        CREATE TABLE IF NOT EXISTS derivation_ledger (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          activity_type TEXT NOT NULL,
          derivation_stage TEXT NOT NULL,
          input_object_links TEXT NOT NULL DEFAULT '[]',
          output_object_links TEXT NOT NULL DEFAULT '[]',
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          actor_id TEXT,
          evaluator_id TEXT,
          parser_id TEXT,
          model_id TEXT,
          model_version TEXT,
          prompt_template_id TEXT,
          tool_call_links TEXT NOT NULL DEFAULT '[]',
          confidence_delta REAL,
          precision_delta REAL,
          scoring_policy_version TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'recorded',
          replay_status TEXT NOT NULL DEFAULT 'replayable',
          activity_time TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          policy_version TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"workflow_traces",
        """
        CREATE TABLE IF NOT EXISTS workflow_traces (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          case_id TEXT,
          workflow_family TEXT,
          subject_anchor TEXT,
          action_class TEXT,
          outcome TEXT,
          episode_links TEXT NOT NULL DEFAULT '[]',
          step_links TEXT NOT NULL DEFAULT '[]',
          actor_links TEXT NOT NULL DEFAULT '[]',
          input_links TEXT NOT NULL DEFAULT '[]',
          output_links TEXT NOT NULL DEFAULT '[]',
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          lifecycle_state TEXT NOT NULL DEFAULT 'extracted',
          validation_status TEXT NOT NULL DEFAULT 'unvalidated',
          event_time_start TEXT,
          event_time_end TEXT,
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"generalized_workflows",
        """
        CREATE TABLE IF NOT EXISTS generalized_workflows (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          workflow_family TEXT NOT NULL,
          scope TEXT NOT NULL DEFAULT '{}',
          applicability_conditions TEXT NOT NULL DEFAULT '{}',
          outcome_class TEXT,
          workflow_trace_links TEXT NOT NULL DEFAULT '[]',
          supporting_episode_links TEXT NOT NULL DEFAULT '[]',
          contradicting_trace_links TEXT NOT NULL DEFAULT '[]',
          step_pattern_links TEXT NOT NULL DEFAULT '[]',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          validation_score REAL NOT NULL DEFAULT 0.0,
          lifecycle_state TEXT NOT NULL DEFAULT 'candidate',
          validation_status TEXT NOT NULL DEFAULT 'unvalidated',
          supersession_status TEXT NOT NULL DEFAULT 'none',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"procedural_memory_objects",
        """
        CREATE TABLE IF NOT EXISTS procedural_memory_objects (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          capability_name TEXT NOT NULL,
          task_family TEXT,
          applicability_conditions TEXT NOT NULL DEFAULT '{}',
          risk_class TEXT NOT NULL DEFAULT 'low',
          generalized_workflow_links TEXT NOT NULL DEFAULT '[]',
          step_links TEXT NOT NULL DEFAULT '[]',
          validation_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          validation_confidence REAL NOT NULL DEFAULT 0.0,
          required_privileges TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'candidate',
          validation_status TEXT NOT NULL DEFAULT 'unvalidated',
          retirement_status TEXT NOT NULL DEFAULT 'active',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"skill_packages",
        """
        CREATE TABLE IF NOT EXISTS skill_packages (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          version INTEGER NOT NULL DEFAULT 1,
          skill_package_name TEXT NOT NULL,
          task_family TEXT,
          competency_links TEXT NOT NULL DEFAULT '[]',
          risk_class TEXT NOT NULL DEFAULT 'low',
          procedural_memory_links TEXT NOT NULL DEFAULT '[]',
          workflow_links TEXT NOT NULL DEFAULT '[]',
          validation_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          input_contract TEXT NOT NULL DEFAULT '{}',
          output_contract TEXT NOT NULL DEFAULT '{}',
          execution_policy TEXT NOT NULL DEFAULT '{}',
          required_privileges TEXT NOT NULL DEFAULT '[]',
          tool_requirements TEXT NOT NULL DEFAULT '[]',
          model_policy_id TEXT,
          aggregate_confidence REAL NOT NULL DEFAULT 0.5,
          aggregate_precision REAL NOT NULL DEFAULT 0.5,
          review_status TEXT NOT NULL DEFAULT 'draft',
          enabled_state TEXT NOT NULL DEFAULT 'disabled',
          suspension_reason TEXT,
          retirement_status TEXT NOT NULL DEFAULT 'active',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, skill_package_name, version)
        )
        """},
       {"active_memory_pools",
        """
        CREATE TABLE IF NOT EXISTS active_memory_pools (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          pool_scope TEXT NOT NULL DEFAULT '{}',
          task_type TEXT,
          subject_anchor TEXT,
          time_mode TEXT NOT NULL DEFAULT 'current_valid',
          loaded_context_links TEXT NOT NULL DEFAULT '[]',
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          context_package_links TEXT NOT NULL DEFAULT '[]',
          promotion_candidate_links TEXT NOT NULL DEFAULT '[]',
          context_confidence_summary TEXT NOT NULL DEFAULT '{}',
          context_precision_summary TEXT NOT NULL DEFAULT '{}',
          member_links TEXT NOT NULL DEFAULT '[]',
          agent_links TEXT NOT NULL DEFAULT '[]',
          tool_links TEXT NOT NULL DEFAULT '[]',
          membership_policy TEXT NOT NULL DEFAULT '{}',
          delegation_chain_links TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'open',
          refresh_state TEXT NOT NULL DEFAULT 'fresh',
          archive_state TEXT NOT NULL DEFAULT 'active',
          opened_at TEXT NOT NULL DEFAULT (datetime('now')),
          closed_at TEXT,
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          policy_version TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"context_packages",
        """
        CREATE TABLE IF NOT EXISTS context_packages (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          request_id TEXT,
          request_intent TEXT,
          requesting_actor_id TEXT,
          active_memory_pool_id TEXT REFERENCES active_memory_pools(id) ON DELETE SET NULL,
          time_mode TEXT NOT NULL DEFAULT 'current_valid',
          detail_depth INTEGER NOT NULL DEFAULT 1,
          memory_links TEXT NOT NULL DEFAULT '[]',
          fact_links TEXT NOT NULL DEFAULT '[]',
          workflow_links TEXT NOT NULL DEFAULT '[]',
          skill_package_links TEXT NOT NULL DEFAULT '[]',
          source_package_links TEXT NOT NULL DEFAULT '[]',
          evidence_links TEXT NOT NULL DEFAULT '[]',
          retrieval_plan TEXT NOT NULL DEFAULT '{}',
          package_confidence_summary TEXT NOT NULL DEFAULT '{}',
          package_precision_summary TEXT NOT NULL DEFAULT '{}',
          filtered_object_summary TEXT NOT NULL DEFAULT '{}',
          returned_object_links TEXT NOT NULL DEFAULT '[]',
          redacted_object_links TEXT NOT NULL DEFAULT '[]',
          authorization_envelope TEXT NOT NULL DEFAULT '{}',
          lifecycle_state TEXT NOT NULL DEFAULT 'assembled',
          refresh_state TEXT NOT NULL DEFAULT 'fresh',
          invalidation_reason TEXT,
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          refresh_time TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          policy_version TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"model_call_operations",
        """
        CREATE TABLE IF NOT EXISTS model_call_operations (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          function_name TEXT NOT NULL,
          model_task_type TEXT NOT NULL,
          execution_mode TEXT NOT NULL DEFAULT 'sync',
          risk_class TEXT NOT NULL DEFAULT 'low',
          model_id TEXT,
          model_version TEXT,
          prompt_template_id TEXT,
          input_schema TEXT NOT NULL DEFAULT '{}',
          output_contract TEXT NOT NULL DEFAULT '{}',
          execution_policy TEXT NOT NULL DEFAULT '{}',
          storage_target TEXT NOT NULL DEFAULT '{}',
          prompt_policy_id TEXT,
          cost_policy TEXT NOT NULL DEFAULT '{}',
          expected_confidence_behavior TEXT NOT NULL DEFAULT '{}',
          required_privileges TEXT NOT NULL DEFAULT '[]',
          allowed_partitions TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'draft',
          rollout_state TEXT NOT NULL DEFAULT 'disabled',
          suspension_reason TEXT,
          retirement_status TEXT NOT NULL DEFAULT 'active',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, function_name)
        )
        """},
       {"mcp_tool_definitions",
        """
        CREATE TABLE IF NOT EXISTS mcp_tool_definitions (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          tool_name TEXT NOT NULL,
          protocol_adapter_id TEXT NOT NULL DEFAULT 'mcp',
          implementation_type TEXT NOT NULL,
          enabled_state TEXT NOT NULL DEFAULT 'disabled',
          registration_source TEXT,
          documentation_links TEXT NOT NULL DEFAULT '[]',
          required_privileges TEXT NOT NULL DEFAULT '[]',
          allowed_partitions TEXT NOT NULL DEFAULT '[]',
          input_schema TEXT NOT NULL DEFAULT '{}',
          output_schema TEXT NOT NULL DEFAULT '{}',
          execution_policy TEXT NOT NULL DEFAULT '{}',
          routing_policy TEXT NOT NULL DEFAULT '{}',
          timeout_policy TEXT NOT NULL DEFAULT '{}',
          cost_policy TEXT NOT NULL DEFAULT '{}',
          audit_policy TEXT NOT NULL DEFAULT '{}',
          lifecycle_state TEXT NOT NULL DEFAULT 'draft',
          suspension_reason TEXT,
          retirement_status TEXT NOT NULL DEFAULT 'active',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          stale_after TEXT,
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          policy_version TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, tool_name, protocol_adapter_id)
        )
        """},
       {"idx_source_packages_workspace_hash",
        "CREATE INDEX IF NOT EXISTS idx_source_packages_workspace_hash ON source_packages(workspace_id, content_hash)"},
       {"idx_claims_source",
        "CREATE INDEX IF NOT EXISTS idx_claims_source ON claims(source_package_id)"},
       {"idx_claims_subject_action",
        "CREATE INDEX IF NOT EXISTS idx_claims_subject_action ON claims(workspace_id, subject_anchor, action_class)"},
       {"idx_facts_subject_action",
        "CREATE INDEX IF NOT EXISTS idx_facts_subject_action ON facts(workspace_id, subject_anchor, action_class)"},
       {"idx_memory_objects_subject_action",
        "CREATE INDEX IF NOT EXISTS idx_memory_objects_subject_action ON memory_objects(workspace_id, subject_anchor, action_class)"},
       {"idx_relationship_edges_from",
        "CREATE INDEX IF NOT EXISTS idx_relationship_edges_from ON relationship_edges(from_object_type, from_object_id)"},
       {"idx_relationship_edges_to",
        "CREATE INDEX IF NOT EXISTS idx_relationship_edges_to ON relationship_edges(to_object_type, to_object_id)"},
       {"idx_derivation_ledger_stage",
        "CREATE INDEX IF NOT EXISTS idx_derivation_ledger_stage ON derivation_ledger(workspace_id, derivation_stage)"},
       {"idx_active_memory_pools_state",
        "CREATE INDEX IF NOT EXISTS idx_active_memory_pools_state ON active_memory_pools(workspace_id, lifecycle_state)"},
       {"idx_context_packages_pool",
        "CREATE INDEX IF NOT EXISTS idx_context_packages_pool ON context_packages(active_memory_pool_id)"}
     ]}
  end

  # Gate 1 — workspace topology + surface/re-entry spine.
  #
  # This migration fixes the first real data-layer mismatch: old Nodes were
  # tenant/slug-scoped, but the product model is workspace -> Node. It also adds
  # the projection and topology records needed before markdown/HTML/wiki/app
  # surfaces can be treated as governed projections.
  defp migration_033_workspace_topology_surface_spine do
    {33, "workspace topology + projection surface spine",
     [
       {"nodes_v033",
        """
        CREATE TABLE IF NOT EXISTS nodes_v033 (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL DEFAULT 'default',
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          kind TEXT NOT NULL,
          node_type_id TEXT,
          parent_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
          description TEXT,
          style TEXT NOT NULL DEFAULT 'internal',
          status TEXT NOT NULL DEFAULT 'active',
          path TEXT NOT NULL DEFAULT '',
          metadata TEXT NOT NULL DEFAULT '{}',
          lifecycle_state TEXT NOT NULL DEFAULT 'active',
          valid_time_start TEXT,
          valid_time_end TEXT,
          transaction_time_start TEXT NOT NULL DEFAULT (datetime('now')),
          transaction_time_end TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, slug)
        )
        """},
       {"nodes_v033_copy",
        """
        INSERT OR IGNORE INTO nodes_v033 (
          id, tenant_id, workspace_id, slug, name, kind, node_type_id,
          parent_id, description, style, status, path, metadata, lifecycle_state,
          created_at
        )
        SELECT
          id,
          tenant_id,
          COALESCE(workspace_id, 'default'),
          slug,
          name,
          kind,
          COALESCE(workspace_id, 'default') || ':' || kind,
          parent_id,
          description,
          style,
          status,
          path,
          metadata,
          status,
          created_at
        FROM nodes
        """},
       {"nodes_drop_old", "DROP TABLE IF EXISTS nodes"},
       {"nodes_rename_v033", "ALTER TABLE nodes_v033 RENAME TO nodes"},
       {"idx_nodes_tenant_kind_v033",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_kind ON nodes(tenant_id, kind)"},
       {"idx_nodes_tenant_parent_v033",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_parent ON nodes(tenant_id, parent_id)"},
       {"idx_nodes_tenant_style_v033",
        "CREATE INDEX IF NOT EXISTS idx_nodes_tenant_style ON nodes(tenant_id, style)"},
       {"idx_nodes_workspace_kind_v033",
        "CREATE INDEX IF NOT EXISTS idx_nodes_ws_kind ON nodes(workspace_id, kind)"},
       {"idx_nodes_workspace_parent_v033",
        "CREATE INDEX IF NOT EXISTS idx_nodes_ws_parent ON nodes(workspace_id, parent_id)"},
       {"node_types",
        """
        CREATE TABLE IF NOT EXISTS node_types (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          category TEXT NOT NULL DEFAULT 'standard',
          description TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          lifecycle_state TEXT NOT NULL DEFAULT 'active',
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, slug)
        )
        """},
       {"seed_default_node_types",
        """
        INSERT OR IGNORE INTO node_types (id, tenant_id, workspace_id, slug, name, category, description)
        SELECT w.id || ':' || t.slug, w.tenant_id, w.id, t.slug, t.name, 'standard', t.description
        FROM workspaces w
        JOIN (
          SELECT 'entity' AS slug, 'Entity' AS name, 'Business, organization, institution, or operating entity.' AS description
          UNION ALL SELECT 'department', 'Department', 'Functional area inside an entity.'
          UNION ALL SELECT 'team', 'Team', 'Group of people or agents working together.'
          UNION ALL SELECT 'project', 'Project', 'Bounded initiative with a target outcome.'
          UNION ALL SELECT 'operation', 'Operation', 'Ongoing process or business function.'
          UNION ALL SELECT 'learning', 'Learning', 'Knowledge acquisition or capability development.'
          UNION ALL SELECT 'person', 'Person', 'Individual human, agent, partner, customer, or stakeholder.'
          UNION ALL SELECT 'product', 'Product', 'Product, platform, system, or offer.'
          UNION ALL SELECT 'partnership', 'Partnership', 'Collaboration, joint venture, or external working relationship.'
          UNION ALL SELECT 'context', 'Context', 'Reference context with a lifecycle and relationships.'
        ) t
        """},
       {"node_relationships",
        """
        CREATE TABLE IF NOT EXISTS node_relationships (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          source_node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
          target_node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
          relationship_type TEXT NOT NULL,
          direction TEXT NOT NULL DEFAULT 'directed',
          strength REAL NOT NULL DEFAULT 1.0,
          valid_time_start TEXT,
          valid_time_end TEXT,
          lifecycle_state TEXT NOT NULL DEFAULT 'active',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(workspace_id, source_node_id, target_node_id, relationship_type)
        )
        """},
       {"topology_change_requests",
        """
        CREATE TABLE IF NOT EXISTS topology_change_requests (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          request_type TEXT NOT NULL,
          target_object_type TEXT NOT NULL,
          target_object_id TEXT,
          proposed_payload TEXT NOT NULL DEFAULT '{}',
          reason TEXT,
          requested_by TEXT,
          review_status TEXT NOT NULL DEFAULT 'pending',
          reviewed_by TEXT,
          reviewed_at TEXT,
          lifecycle_state TEXT NOT NULL DEFAULT 'open',
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"export_records",
        """
        CREATE TABLE IF NOT EXISTS export_records (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          source_object_type TEXT NOT NULL,
          source_object_id TEXT NOT NULL,
          surface_type TEXT NOT NULL DEFAULT 'markdown',
          destination_uri TEXT NOT NULL,
          object_version TEXT,
          policy_version TEXT,
          lifecycle_state TEXT NOT NULL DEFAULT 'current',
          generated_by TEXT,
          generated_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(workspace_id, source_object_type, source_object_id, surface_type, destination_uri)
        )
        """},
       {"projection_revisions",
        """
        CREATE TABLE IF NOT EXISTS projection_revisions (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          export_record_id TEXT NOT NULL REFERENCES export_records(id) ON DELETE CASCADE,
          revision_number INTEGER NOT NULL DEFAULT 1,
          content_hash TEXT NOT NULL,
          projection_uri TEXT NOT NULL,
          source_object_links TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'current',
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(export_record_id, revision_number)
        )
        """},
       {"link_health_records",
        """
        CREATE TABLE IF NOT EXISTS link_health_records (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          export_record_id TEXT REFERENCES export_records(id) ON DELETE CASCADE,
          projection_revision_id TEXT REFERENCES projection_revisions(id) ON DELETE CASCADE,
          status TEXT NOT NULL DEFAULT 'unchecked',
          broken_links TEXT NOT NULL DEFAULT '[]',
          backlinks TEXT NOT NULL DEFAULT '[]',
          missing_references TEXT NOT NULL DEFAULT '[]',
          stale_references TEXT NOT NULL DEFAULT '[]',
          checked_at TEXT NOT NULL DEFAULT (datetime('now')),
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_node_types_workspace_slug",
        "CREATE INDEX IF NOT EXISTS idx_node_types_workspace_slug ON node_types(workspace_id, slug)"},
       {"idx_node_relationships_source",
        "CREATE INDEX IF NOT EXISTS idx_node_relationships_source ON node_relationships(workspace_id, source_node_id)"},
       {"idx_node_relationships_target",
        "CREATE INDEX IF NOT EXISTS idx_node_relationships_target ON node_relationships(workspace_id, target_node_id)"},
       {"idx_topology_change_requests_status",
        "CREATE INDEX IF NOT EXISTS idx_topology_change_requests_status ON topology_change_requests(workspace_id, review_status)"},
       {"idx_export_records_source",
        "CREATE INDEX IF NOT EXISTS idx_export_records_source ON export_records(workspace_id, source_object_type, source_object_id)"},
       {"idx_projection_revisions_export",
        "CREATE INDEX IF NOT EXISTS idx_projection_revisions_export ON projection_revisions(export_record_id, revision_number)"},
       {"idx_link_health_export",
        "CREATE INDEX IF NOT EXISTS idx_link_health_export ON link_health_records(export_record_id, checked_at)"}
     ]}
  end

  defp migration_034_tool_model_governance_runs do
    {34, "tool/model governance run records",
     [
       {"model_call_runs",
        """
        CREATE TABLE IF NOT EXISTS model_call_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          model_call_operation_id TEXT NOT NULL,
          function_name TEXT NOT NULL,
          requesting_actor_id TEXT,
          active_memory_pool_id TEXT,
          decision_state TEXT NOT NULL,
          run_status TEXT NOT NULL DEFAULT 'recorded',
          rejection_reason TEXT,
          input_payload TEXT NOT NULL DEFAULT '{}',
          output_payload TEXT NOT NULL DEFAULT '{}',
          input_hash TEXT,
          output_hash TEXT,
          required_privileges TEXT NOT NULL DEFAULT '[]',
          granted_privileges TEXT NOT NULL DEFAULT '[]',
          requested_partitions TEXT NOT NULL DEFAULT '[]',
          allowed_partitions TEXT NOT NULL DEFAULT '[]',
          policy_decision TEXT NOT NULL DEFAULT '{}',
          latency_ms INTEGER,
          cost_units REAL,
          source_package_links TEXT NOT NULL DEFAULT '[]',
          observation_links TEXT NOT NULL DEFAULT '[]',
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"tool_call_runs",
        """
        CREATE TABLE IF NOT EXISTS tool_call_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          mcp_tool_definition_id TEXT NOT NULL,
          tool_name TEXT NOT NULL,
          protocol_adapter_id TEXT NOT NULL DEFAULT 'mcp',
          requesting_actor_id TEXT,
          active_memory_pool_id TEXT,
          decision_state TEXT NOT NULL,
          run_status TEXT NOT NULL DEFAULT 'recorded',
          rejection_reason TEXT,
          input_payload TEXT NOT NULL DEFAULT '{}',
          output_payload TEXT NOT NULL DEFAULT '{}',
          input_hash TEXT,
          output_hash TEXT,
          required_privileges TEXT NOT NULL DEFAULT '[]',
          granted_privileges TEXT NOT NULL DEFAULT '[]',
          requested_partitions TEXT NOT NULL DEFAULT '[]',
          allowed_partitions TEXT NOT NULL DEFAULT '[]',
          policy_decision TEXT NOT NULL DEFAULT '{}',
          latency_ms INTEGER,
          cost_units REAL,
          source_package_links TEXT NOT NULL DEFAULT '[]',
          observation_links TEXT NOT NULL DEFAULT '[]',
          audit_event_links TEXT NOT NULL DEFAULT '[]',
          access_policy_id TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_model_call_runs_operation",
        "CREATE INDEX IF NOT EXISTS idx_model_call_runs_operation ON model_call_runs(workspace_id, model_call_operation_id)"},
       {"idx_model_call_runs_pool",
        "CREATE INDEX IF NOT EXISTS idx_model_call_runs_pool ON model_call_runs(active_memory_pool_id)"},
       {"idx_tool_call_runs_tool",
        "CREATE INDEX IF NOT EXISTS idx_tool_call_runs_tool ON tool_call_runs(workspace_id, mcp_tool_definition_id)"},
       {"idx_tool_call_runs_pool",
        "CREATE INDEX IF NOT EXISTS idx_tool_call_runs_pool ON tool_call_runs(active_memory_pool_id)"}
     ]}
  end

  defp migration_035_asset_governance do
    {35, "asset governance metadata",
     [
       {"assets.content_hash", "ALTER TABLE assets ADD COLUMN content_hash TEXT"},
       {"assets.modality", "ALTER TABLE assets ADD COLUMN modality TEXT NOT NULL DEFAULT 'binary'"},
       {"assets.source_package_id",
        "ALTER TABLE assets ADD COLUMN source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL"},
       {"assets.original_path", "ALTER TABLE assets ADD COLUMN original_path TEXT"},
       {"assets.trust_label",
        "ALTER TABLE assets ADD COLUMN trust_label TEXT NOT NULL DEFAULT 'unreviewed'"},
       {"assets.retention_class",
        "ALTER TABLE assets ADD COLUMN retention_class TEXT NOT NULL DEFAULT 'standard'"},
       {"assets.access_policy_id", "ALTER TABLE assets ADD COLUMN access_policy_id TEXT"},
       {"assets.security_labels",
        "ALTER TABLE assets ADD COLUMN security_labels TEXT NOT NULL DEFAULT '[]'"},
       {"assets.partition_ids",
        "ALTER TABLE assets ADD COLUMN partition_ids TEXT NOT NULL DEFAULT '[]'"},
       {"assets.metadata", "ALTER TABLE assets ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}'"},
       {"idx_assets_workspace_hash",
        "CREATE INDEX IF NOT EXISTS idx_assets_workspace_hash ON assets(workspace_id, content_hash)"},
       {"idx_assets_source_package",
        "CREATE INDEX IF NOT EXISTS idx_assets_source_package ON assets(source_package_id)"}
     ]}
  end

  defp migration_036_asset_adapter_runs do
    {36, "asset adapter run records",
     [
       {"asset_adapter_runs",
        """
        CREATE TABLE IF NOT EXISTS asset_adapter_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_id TEXT NOT NULL,
          adapter_role TEXT NOT NULL,
          modality TEXT NOT NULL,
          status TEXT NOT NULL,
          started_at TEXT NOT NULL,
          completed_at TEXT,
          input_hash TEXT,
          output_hash TEXT,
          output_text TEXT NOT NULL DEFAULT '',
          output_ref TEXT,
          model_id TEXT,
          model_version TEXT,
          confidence REAL,
          precision REAL,
          error_reason TEXT,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_asset_adapter_runs_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_adapter_runs_asset ON asset_adapter_runs(workspace_id, asset_id)"},
       {"idx_asset_adapter_runs_adapter",
        "CREATE INDEX IF NOT EXISTS idx_asset_adapter_runs_adapter ON asset_adapter_runs(workspace_id, adapter_id, status)"},
       {"idx_asset_adapter_runs_source_package",
        "CREATE INDEX IF NOT EXISTS idx_asset_adapter_runs_source_package ON asset_adapter_runs(source_package_id)"}
     ]}
  end

  defp migration_037_asset_extraction_projections do
    {37, "asset extraction projection records",
     [
       {"asset_extractions",
        """
        CREATE TABLE IF NOT EXISTS asset_extractions (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_run_id TEXT NOT NULL REFERENCES asset_adapter_runs(id) ON DELETE CASCADE,
          extraction_type TEXT NOT NULL,
          modality TEXT NOT NULL,
          content_text TEXT NOT NULL DEFAULT '',
          content_ref TEXT,
          content_hash TEXT,
          confidence REAL,
          precision REAL,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"asset_transcripts",
        """
        CREATE TABLE IF NOT EXISTS asset_transcripts (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_run_id TEXT NOT NULL REFERENCES asset_adapter_runs(id) ON DELETE CASCADE,
          extraction_id TEXT NOT NULL REFERENCES asset_extractions(id) ON DELETE CASCADE,
          language TEXT,
          speaker TEXT,
          transcript_text TEXT NOT NULL,
          start_ms INTEGER,
          end_ms INTEGER,
          confidence REAL,
          precision REAL,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"asset_ocr_spans",
        """
        CREATE TABLE IF NOT EXISTS asset_ocr_spans (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_run_id TEXT NOT NULL REFERENCES asset_adapter_runs(id) ON DELETE CASCADE,
          extraction_id TEXT NOT NULL REFERENCES asset_extractions(id) ON DELETE CASCADE,
          page_number INTEGER,
          span_text TEXT NOT NULL,
          bbox TEXT NOT NULL DEFAULT '{}',
          confidence REAL,
          precision REAL,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"asset_visual_observations",
        """
        CREATE TABLE IF NOT EXISTS asset_visual_observations (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_run_id TEXT NOT NULL REFERENCES asset_adapter_runs(id) ON DELETE CASCADE,
          extraction_id TEXT NOT NULL REFERENCES asset_extractions(id) ON DELETE CASCADE,
          observation_type TEXT NOT NULL,
          observation_text TEXT NOT NULL,
          region TEXT NOT NULL DEFAULT '{}',
          frame_time_ms INTEGER,
          confidence REAL,
          precision REAL,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"asset_embedding_refs",
        """
        CREATE TABLE IF NOT EXISTS asset_embedding_refs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
          source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL,
          adapter_run_id TEXT NOT NULL REFERENCES asset_adapter_runs(id) ON DELETE CASCADE,
          extraction_id TEXT NOT NULL REFERENCES asset_extractions(id) ON DELETE CASCADE,
          embedding_model_id TEXT NOT NULL,
          embedding_model_version TEXT,
          embedding_ref TEXT NOT NULL,
          embedding_dim INTEGER,
          embedding_space TEXT,
          target_ref TEXT,
          confidence REAL,
          precision REAL,
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          metadata TEXT NOT NULL DEFAULT '{}',
          derivation_ledger_id TEXT REFERENCES derivation_ledger(id) ON DELETE SET NULL,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_asset_extractions_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_extractions_asset ON asset_extractions(workspace_id, asset_id, extraction_type)"},
       {"idx_asset_extractions_adapter_run",
        "CREATE INDEX IF NOT EXISTS idx_asset_extractions_adapter_run ON asset_extractions(adapter_run_id)"},
       {"idx_asset_transcripts_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_transcripts_asset ON asset_transcripts(workspace_id, asset_id)"},
       {"idx_asset_ocr_spans_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_ocr_spans_asset ON asset_ocr_spans(workspace_id, asset_id, page_number)"},
       {"idx_asset_visual_observations_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_visual_observations_asset ON asset_visual_observations(workspace_id, asset_id)"},
       {"idx_asset_embedding_refs_asset",
        "CREATE INDEX IF NOT EXISTS idx_asset_embedding_refs_asset ON asset_embedding_refs(workspace_id, asset_id, embedding_model_id)"}
     ]}
  end

  defp migration_038_evaluation_records do
    {38, "evaluation and benchmark records",
     [
       {"evaluation_runs",
        """
        CREATE TABLE IF NOT EXISTS evaluation_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          benchmark_name TEXT NOT NULL,
          dataset_name TEXT,
          dataset_version TEXT,
          dataset_size INTEGER,
          question_count INTEGER,
          answer_model TEXT,
          judge_model TEXT,
          judge_strategy TEXT,
          retrieval_top_k INTEGER,
          run_config TEXT NOT NULL DEFAULT '{}',
          retrieval_config TEXT NOT NULL DEFAULT '{}',
          judge_config TEXT NOT NULL DEFAULT '{}',
          aggregate_scores TEXT NOT NULL DEFAULT '{}',
          status TEXT NOT NULL DEFAULT 'recorded',
          started_at TEXT,
          completed_at TEXT,
          created_by TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"evaluation_cases",
        """
        CREATE TABLE IF NOT EXISTS evaluation_cases (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          evaluation_run_id TEXT NOT NULL REFERENCES evaluation_runs(id) ON DELETE CASCADE,
          case_id TEXT NOT NULL,
          conversation_id TEXT,
          question TEXT NOT NULL,
          expected_answer TEXT,
          actual_answer TEXT,
          context_package_id TEXT REFERENCES context_packages(id) ON DELETE SET NULL,
          retrieved_object_links TEXT NOT NULL DEFAULT '[]',
          scores TEXT NOT NULL DEFAULT '{}',
          judge_output TEXT NOT NULL DEFAULT '{}',
          status TEXT NOT NULL DEFAULT 'recorded',
          error_reason TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(evaluation_run_id, case_id)
        )
        """},
       {"idx_evaluation_runs_workspace",
        "CREATE INDEX IF NOT EXISTS idx_evaluation_runs_workspace ON evaluation_runs(workspace_id, benchmark_name, created_at)"},
       {"idx_evaluation_cases_run",
        "CREATE INDEX IF NOT EXISTS idx_evaluation_cases_run ON evaluation_cases(evaluation_run_id, status)"}
     ]}
  end

  # Workspace isolation fixes:
  #
  # 1. `connectors` / `connector_runs` predate migration 026 and never got a
  #    workspace_id column, contradicting the workspace contract. Add it so
  #    connector-ingested signals can be attributed to the connector's workspace.
  #
  # 2. Legacy node renames used to run as an unconditional update on every Store
  #    init, silently rewriting node names across all workspaces. They are now a
  #    one-time migration scoped to the pre-workspace `default` workspace.
  defp migration_039_connector_workspace_and_legacy_node_renames do
    {39, "connectors.workspace_id + one-time legacy node renames (workspace isolation)",
     [
       {"connectors.workspace_id",
        "ALTER TABLE connectors ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"connector_runs.workspace_id",
        "ALTER TABLE connector_runs ADD COLUMN workspace_id TEXT NOT NULL DEFAULT 'default'"},
       {"idx_connectors_ws",
        "CREATE INDEX IF NOT EXISTS idx_connectors_ws ON connectors(workspace_id)"},
       {"idx_connector_runs_ws",
        "CREATE INDEX IF NOT EXISTS idx_connector_runs_ws ON connector_runs(workspace_id, connector_id)"},
       {"legacy rename 01-inbox to inbox (default workspace only)",
        "UPDATE contexts SET node = 'inbox' WHERE node = '01-inbox' AND workspace_id = 'default'"},
       {"legacy rename 04-products to product-customer-portal (default workspace only)",
        "UPDATE contexts SET node = 'product-customer-portal' WHERE node = '04-products' AND workspace_id = 'default'"}
     ]}
  end

  defp migration_040_asset_governance_backfill do
    {40, "asset governance metadata backfill",
     [
       {"assets.content_hash", "ALTER TABLE assets ADD COLUMN content_hash TEXT"},
       {"assets.modality", "ALTER TABLE assets ADD COLUMN modality TEXT NOT NULL DEFAULT 'binary'"},
       {"assets.source_package_id",
        "ALTER TABLE assets ADD COLUMN source_package_id TEXT REFERENCES source_packages(id) ON DELETE SET NULL"},
       {"assets.original_path", "ALTER TABLE assets ADD COLUMN original_path TEXT"},
       {"assets.trust_label",
        "ALTER TABLE assets ADD COLUMN trust_label TEXT NOT NULL DEFAULT 'unreviewed'"},
       {"assets.retention_class",
        "ALTER TABLE assets ADD COLUMN retention_class TEXT NOT NULL DEFAULT 'standard'"},
       {"assets.access_policy_id", "ALTER TABLE assets ADD COLUMN access_policy_id TEXT"},
       {"assets.security_labels",
        "ALTER TABLE assets ADD COLUMN security_labels TEXT NOT NULL DEFAULT '[]'"},
       {"assets.partition_ids",
        "ALTER TABLE assets ADD COLUMN partition_ids TEXT NOT NULL DEFAULT '[]'"},
       {"assets.metadata", "ALTER TABLE assets ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}'"},
       {"idx_assets_workspace_hash",
        "CREATE INDEX IF NOT EXISTS idx_assets_workspace_hash ON assets(workspace_id, content_hash)"},
       {"idx_assets_source_package",
        "CREATE INDEX IF NOT EXISTS idx_assets_source_package ON assets(source_package_id)"}
     ]}
  end

  # Phase 21 — Episodes: first-class episodic memory objects.
  # An episode records a discrete workspace event (transcript ingested, meeting
  # held, agent loop completed). The intake pipeline creates one row per
  # transcript/meeting source; other subsystems can also insert episodes
  # directly. `provenance` links back to source objects (source_package_id,
  # signal_id, etc.) as a JSON map so the lineage chain is explicit.
  defp migration_041_episodes do
    {41, "episodes — episodic memory primitive for transcript/meeting intake",
     [
       {"episodes",
        """
        CREATE TABLE IF NOT EXISTS episodes (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL DEFAULT 'default',
          node_id TEXT,
          kind TEXT NOT NULL DEFAULT 'event',
          occurred_at TEXT NOT NULL DEFAULT (datetime('now')),
          summary TEXT NOT NULL,
          provenance TEXT NOT NULL DEFAULT '{}',
          security_labels TEXT NOT NULL DEFAULT '[]',
          partition_ids TEXT NOT NULL DEFAULT '[]',
          lifecycle_state TEXT NOT NULL DEFAULT 'recorded',
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_episodes_workspace_kind",
        "CREATE INDEX IF NOT EXISTS idx_episodes_workspace_kind ON episodes(workspace_id, kind, occurred_at)"},
       {"idx_episodes_workspace_node",
        "CREATE INDEX IF NOT EXISTS idx_episodes_workspace_node ON episodes(workspace_id, node_id, occurred_at)"}
     ]}
  end

  # Organizations are operating entities inside a tenant. The tenant remains
  # the hard security boundary; organizations provide the ownership boundary
  # above workspaces without preventing governed cross-organization views.
  defp migration_042_organizations do
    {42, "organizations + workspace ownership",
     [
       {"organizations",
        """
        CREATE TABLE IF NOT EXISTS organizations (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          description TEXT,
          status TEXT NOT NULL DEFAULT 'active',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          archived_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(tenant_id, slug)
        )
        """},
       {"idx_organizations_tenant_status",
        "CREATE INDEX IF NOT EXISTS idx_organizations_tenant_status ON organizations(tenant_id, status)"},
       {"seed_default_organization",
        """
        INSERT OR IGNORE INTO organizations (
          id, tenant_id, slug, name, description, status
        ) VALUES (
          'default', 'default', 'default', 'Default organization',
          'Compatibility owner for workspaces created before organizations were first-class.',
          'active'
        )
        """},
       {"workspaces.organization_id",
        "ALTER TABLE workspaces ADD COLUMN organization_id TEXT REFERENCES organizations(id)"},
       {"backfill_workspaces_organization_id",
        "UPDATE workspaces SET organization_id = 'default' WHERE organization_id IS NULL"},
       {"idx_workspaces_organization_status",
        "CREATE INDEX IF NOT EXISTS idx_workspaces_organization_status ON workspaces(organization_id, status)"}
     ]}
  end

  # Repair installations where an early migration 042 attempt was recorded
  # after SQLite rejected its original non-null REFERENCES column statement.
  defp migration_043_repair_workspace_organization_ownership do
    {43, "repair workspace organization ownership",
     [
       {"workspaces.organization_id.repair",
        "ALTER TABLE workspaces ADD COLUMN organization_id TEXT REFERENCES organizations(id)"},
       {"backfill_workspaces_organization_id.repair",
        "UPDATE workspaces SET organization_id = 'default' WHERE organization_id IS NULL"},
       {"idx_workspaces_organization_status.repair",
        "CREATE INDEX IF NOT EXISTS idx_workspaces_organization_status ON workspaces(organization_id, status)"}
     ]}
  end

  # Reconcile live stores where migration 42 was recorded despite only some
  # statements being applied. Creation and backfill are intentionally ordered.
  defp migration_044_reconcile_organization_schema do
    {44, "reconcile organization schema",
     [
       {"organizations.reconcile",
        """
        CREATE TABLE IF NOT EXISTS organizations (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
          slug TEXT NOT NULL,
          name TEXT NOT NULL,
          description TEXT,
          status TEXT NOT NULL DEFAULT 'active',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          archived_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}',
          UNIQUE(tenant_id, slug)
        )
        """},
       {"idx_organizations_tenant_status.reconcile",
        "CREATE INDEX IF NOT EXISTS idx_organizations_tenant_status ON organizations(tenant_id, status)"},
       {"seed_default_organization.reconcile",
        """
        INSERT OR IGNORE INTO organizations (
          id, tenant_id, slug, name, description, status
        ) VALUES (
          'default', 'default', 'default', 'Default organization',
          'Compatibility owner for workspaces created before organizations were first-class.',
          'active'
        )
        """},
       {"workspaces.organization_id.reconcile",
        "ALTER TABLE workspaces ADD COLUMN organization_id TEXT REFERENCES organizations(id)"},
       {"backfill_workspaces_organization_id.reconcile",
        "UPDATE workspaces SET organization_id = 'default' WHERE organization_id IS NULL"},
       {"idx_workspaces_organization_status.reconcile",
        "CREATE INDEX IF NOT EXISTS idx_workspaces_organization_status ON workspaces(organization_id, status)"}
     ]}
  end

  defp migration_045_customer_node_type do
    {45, "customer node type",
     [
       {"seed_customer_node_type",
        """
        INSERT OR IGNORE INTO node_types (
          id, tenant_id, workspace_id, slug, name, category, description
        )
        SELECT
          w.id || ':customer', w.tenant_id, w.id, 'customer', 'Customer', 'standard',
          'Client account or customer organization receiving ongoing value.'
        FROM workspaces w
        """}
     ]}
  end

  defp migration_046_operational_stores do
    {46, "durable jobs, metric history, and backup catalog",
     [
       {"jobs",
        """
        CREATE TABLE IF NOT EXISTS jobs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL,
          kind TEXT NOT NULL,
          payload TEXT NOT NULL DEFAULT '{}',
          status TEXT NOT NULL DEFAULT 'available',
          priority INTEGER NOT NULL DEFAULT 0,
          run_at TEXT NOT NULL DEFAULT (datetime('now')),
          attempts INTEGER NOT NULL DEFAULT 0,
          max_attempts INTEGER NOT NULL DEFAULT 5,
          lease_owner TEXT,
          lease_expires_at TEXT,
          idempotency_key TEXT,
          last_error TEXT,
          inserted_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          completed_at TEXT,
          UNIQUE(tenant_id, workspace_id, idempotency_key)
        )
        """},
       {"idx_jobs_claim",
        "CREATE INDEX IF NOT EXISTS idx_jobs_claim ON jobs(status, run_at, priority DESC)"},
       {"idx_jobs_workspace_status",
        "CREATE INDEX IF NOT EXISTS idx_jobs_workspace_status ON jobs(workspace_id, status, run_at)"},
       {"dead_letter_jobs",
        """
        CREATE TABLE IF NOT EXISTS dead_letter_jobs (
          id TEXT PRIMARY KEY,
          original_job_id TEXT NOT NULL,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL,
          kind TEXT NOT NULL,
          payload TEXT NOT NULL DEFAULT '{}',
          attempts INTEGER NOT NULL,
          final_error TEXT NOT NULL,
          failed_at TEXT NOT NULL DEFAULT (datetime('now')),
          replayed_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_dead_letter_workspace",
        "CREATE INDEX IF NOT EXISTS idx_dead_letter_workspace ON dead_letter_jobs(workspace_id, failed_at)"},
       {"metric_samples",
        """
        CREATE TABLE IF NOT EXISTS metric_samples (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL,
          metric_name TEXT NOT NULL,
          metric_kind TEXT NOT NULL,
          value REAL NOT NULL,
          labels TEXT NOT NULL DEFAULT '{}',
          recorded_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_metric_samples_name_time",
        "CREATE INDEX IF NOT EXISTS idx_metric_samples_name_time ON metric_samples(metric_name, recorded_at)"},
       {"idx_metric_samples_workspace_time",
        "CREATE INDEX IF NOT EXISTS idx_metric_samples_workspace_time ON metric_samples(workspace_id, recorded_at)"},
       {"backup_records",
        """
        CREATE TABLE IF NOT EXISTS backup_records (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          storage_provider TEXT NOT NULL DEFAULT 'local',
          storage_uri TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'created',
          encrypted INTEGER NOT NULL DEFAULT 0,
          offsite INTEGER NOT NULL DEFAULT 0,
          size_bytes INTEGER,
          checksum_sha256 TEXT,
          integrity_status TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          verified_at TEXT,
          expires_at TEXT,
          metadata TEXT NOT NULL DEFAULT '{}'
        )
        """},
       {"idx_backup_records_status_time",
        "CREATE INDEX IF NOT EXISTS idx_backup_records_status_time ON backup_records(status, created_at)"}
     ]}
  end

  defp migration_047_model_adaptation_store do
    {47, "decomposition and workspace-scoped model adaptation records",
     [
       {"decomposition_runs",
        """
        CREATE TABLE IF NOT EXISTS decomposition_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          source_package_id TEXT NOT NULL,
          strategy TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'queued',
          fallback_strategy TEXT,
          attempt INTEGER NOT NULL DEFAULT 1,
          checkpoint_uri TEXT,
          input_bytes INTEGER,
          output_units INTEGER,
          model_calls INTEGER NOT NULL DEFAULT 0,
          iterations INTEGER NOT NULL DEFAULT 0,
          metrics TEXT NOT NULL DEFAULT '{}',
          error TEXT,
          started_at TEXT,
          completed_at TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_decomposition_runs_scope",
        "CREATE INDEX IF NOT EXISTS idx_decomposition_runs_scope ON decomposition_runs(organization_id, workspace_id, source_package_id, status)"},
       {"training_examples",
        """
        CREATE TABLE IF NOT EXISTS training_examples (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          source_package_id TEXT,
          input TEXT NOT NULL,
          expected_output TEXT NOT NULL,
          purpose TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'candidate',
          consent_basis TEXT,
          content_hash TEXT NOT NULL,
          metadata TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          approved_at TEXT,
          UNIQUE(workspace_id, content_hash)
        )
        """},
       {"idx_training_examples_scope",
        "CREATE INDEX IF NOT EXISTS idx_training_examples_scope ON training_examples(organization_id, workspace_id, status, purpose)"},
       {"training_runs",
        """
        CREATE TABLE IF NOT EXISTS training_runs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          base_model TEXT NOT NULL,
          trainer TEXT NOT NULL,
          method TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'queued',
          dataset_query TEXT NOT NULL DEFAULT '{}',
          hyperparameters TEXT NOT NULL DEFAULT '{}',
          hardware TEXT NOT NULL DEFAULT '{}',
          metrics TEXT NOT NULL DEFAULT '{}',
          error TEXT,
          started_at TEXT,
          completed_at TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_training_runs_scope",
        "CREATE INDEX IF NOT EXISTS idx_training_runs_scope ON training_runs(organization_id, workspace_id, status, created_at)"},
       {"model_adapters",
        """
        CREATE TABLE IF NOT EXISTS model_adapters (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT NOT NULL,
          workspace_id TEXT NOT NULL,
          training_run_id TEXT NOT NULL REFERENCES training_runs(id),
          base_model TEXT NOT NULL,
          adapter_type TEXT NOT NULL,
          version INTEGER NOT NULL DEFAULT 1,
          artifact_uri TEXT NOT NULL,
          checksum_sha256 TEXT NOT NULL,
          status TEXT NOT NULL DEFAULT 'candidate',
          evaluation TEXT NOT NULL DEFAULT '{}',
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          activated_at TEXT,
          retired_at TEXT,
          UNIQUE(workspace_id, id, version)
        )
        """},
       {"idx_model_adapters_scope",
        "CREATE INDEX IF NOT EXISTS idx_model_adapters_scope ON model_adapters(organization_id, workspace_id, status, base_model)"}
     ]}
  end

  defp migration_048_repair_chunk_workspace_scope do
    {48, "repair chunk and embedding workspace scope from parent contexts",
     [
       {"repair_chunks_workspace_scope",
        """
        UPDATE chunks
        SET workspace_id = (
          SELECT contexts.workspace_id
          FROM contexts
          WHERE contexts.id = chunks.signal_id
        )
        WHERE workspace_id = 'default'
          AND EXISTS (
            SELECT 1 FROM contexts
            WHERE contexts.id = chunks.signal_id
              AND contexts.workspace_id != 'default'
          )
        """},
       {"repair_chunk_embeddings_workspace_scope",
        """
        UPDATE chunk_embeddings
        SET workspace_id = (
          SELECT chunks.workspace_id
          FROM chunks
          WHERE chunks.id = chunk_embeddings.chunk_id
        )
        WHERE workspace_id = 'default'
          AND EXISTS (
            SELECT 1 FROM chunks
            WHERE chunks.id = chunk_embeddings.chunk_id
              AND chunks.workspace_id != 'default'
          )
        """}
     ]}
  end

  defp migration_049_rebuild_contexts_fts_triggers do
    {49, "rebuild contexts FTS and use content-table-safe triggers",
     [
       {"drop_contexts_fts_insert_trigger", "DROP TRIGGER IF EXISTS contexts_fts_insert"},
       {"drop_contexts_fts_update_trigger", "DROP TRIGGER IF EXISTS contexts_fts_update"},
       {"drop_contexts_fts_delete_trigger", "DROP TRIGGER IF EXISTS contexts_fts_delete"},
       {"clear_contexts_fts", "DELETE FROM contexts_fts"},
       {"rebuild_contexts_fts",
        """
        INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre)
        SELECT rowid, id, title, content, node, type, COALESCE(genre, '')
        FROM contexts
        """},
       {"contexts_fts_insert_trigger_v2",
        """
        CREATE TRIGGER contexts_fts_insert AFTER INSERT ON contexts BEGIN
          INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre)
          VALUES (new.rowid, new.id, new.title, new.content, new.node, new.type, COALESCE(new.genre, ''));
        END
        """},
       {"contexts_fts_update_trigger_v2",
        """
        CREATE TRIGGER contexts_fts_update AFTER UPDATE ON contexts BEGIN
          DELETE FROM contexts_fts WHERE rowid = old.rowid;
          INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre)
          VALUES (new.rowid, new.id, new.title, new.content, new.node, new.type, COALESCE(new.genre, ''));
        END
        """},
       {"contexts_fts_delete_trigger_v2",
        """
        CREATE TRIGGER contexts_fts_delete AFTER DELETE ON contexts BEGIN
          DELETE FROM contexts_fts WHERE rowid = old.rowid;
        END
        """}
     ]}
  end

  defp migration_050_retire_test_storage_fixtures do
    {50, "retire leaked test workspaces and stale temporary assets",
     [
       {"archive_example_secret_workspaces",
        """
        UPDATE workspaces
        SET status = 'archived'
        WHERE id GLOB 'exampleorg-*:secrets'
          AND name = 'WS secrets'
        """},
       {"delete_stale_test_assets",
        """
        DELETE FROM assets
        WHERE storage_path LIKE '/var/folders/%/T/%'
           OR storage_path GLOB '*/exercise-mm-*/assets/*'
        """}
     ]}
  end

  defp migration_051_workspace_storage_policies do
    {51, "workspace storage policies and governed replication ledger",
     [
       {"storage_provider_configs",
        """
        CREATE TABLE IF NOT EXISTS storage_provider_configs (
          id TEXT PRIMARY KEY,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT,
          workspace_id TEXT,
          provider_id TEXT NOT NULL,
          enabled INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0, 1)),
          lifecycle_state TEXT NOT NULL DEFAULT 'configured',
          config_refs TEXT NOT NULL DEFAULT '{}',
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          UNIQUE(tenant_id, workspace_id, provider_id),
          FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
        )
        """},
       {"idx_storage_provider_configs_scope",
        "CREATE INDEX IF NOT EXISTS idx_storage_provider_configs_scope ON storage_provider_configs(tenant_id, organization_id, workspace_id, enabled)"},
       {"workspace_storage_policies",
        """
        CREATE TABLE IF NOT EXISTS workspace_storage_policies (
          workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT,
          use_cases TEXT NOT NULL DEFAULT '[]',
          provider_overrides TEXT NOT NULL DEFAULT '{}',
          policy_version INTEGER NOT NULL DEFAULT 1,
          created_by TEXT,
          created_at TEXT NOT NULL DEFAULT (datetime('now')),
          updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_workspace_storage_policies_scope",
        "CREATE INDEX IF NOT EXISTS idx_workspace_storage_policies_scope ON workspace_storage_policies(tenant_id, organization_id, workspace_id)"},
       {"sync_mutations",
        """
        CREATE TABLE IF NOT EXISTS sync_mutations (
          sequence INTEGER PRIMARY KEY AUTOINCREMENT,
          id TEXT NOT NULL UNIQUE,
          tenant_id TEXT NOT NULL DEFAULT 'default',
          organization_id TEXT,
          workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
          device_id TEXT NOT NULL,
          actor_id TEXT,
          entity_type TEXT NOT NULL,
          entity_id TEXT NOT NULL,
          operation TEXT NOT NULL CHECK(operation IN ('create', 'update', 'delete')),
          payload TEXT,
          payload_hash TEXT NOT NULL,
          idempotency_key TEXT NOT NULL UNIQUE,
          occurred_at TEXT NOT NULL,
          recorded_at TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """},
       {"idx_sync_mutations_workspace_sequence",
        "CREATE INDEX IF NOT EXISTS idx_sync_mutations_workspace_sequence ON sync_mutations(tenant_id, workspace_id, sequence)"},
       {"idx_sync_mutations_entity",
        "CREATE INDEX IF NOT EXISTS idx_sync_mutations_entity ON sync_mutations(workspace_id, entity_type, entity_id, sequence)"},
       {"sync_cursors",
        """
        CREATE TABLE IF NOT EXISTS sync_cursors (
          tenant_id TEXT NOT NULL DEFAULT 'default',
          workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
          replica_id TEXT NOT NULL,
          last_sequence INTEGER NOT NULL DEFAULT 0,
          updated_at TEXT NOT NULL DEFAULT (datetime('now')),
          PRIMARY KEY(tenant_id, workspace_id, replica_id)
        )
        """}
     ]}
  end

  defp migration_052_backfill_workspace_storage_policies do
    {52, "backfill local-first policies for existing workspaces",
     [
       {"backfill_workspace_storage_policies",
        """
        INSERT OR IGNORE INTO workspace_storage_policies (
          workspace_id, tenant_id, organization_id, use_cases, created_by
        )
        SELECT id, tenant_id, organization_id, '[\"desktop_local\"]', 'migration.052'
        FROM workspaces
        WHERE status = 'active'
        """}
     ]}
  end

  # ---------------------------------------------------------------------------
  # Private — helpers
  # ---------------------------------------------------------------------------

  defp ensure_migrations_table!(db) do
    # Apply migration 001 unconditionally — it's the bootstrap.
    {1, _desc, stmts} = migration_001_schema_migrations_table()
    Enum.each(stmts, fn {label, sql} -> safe_execute(db, label, sql) end)
    :ok
  end

  defp collect_versions(db, stmt, acc) do
    case Exqlite.Sqlite3.step(db, stmt) do
      {:row, [version]} -> collect_versions(db, stmt, [version | acc])
      :done -> acc
      {:error, _} -> acc
    end
  end

  defp record_migration!(db, version, description) do
    {:ok, stmt} =
      Exqlite.Sqlite3.prepare(
        db,
        "INSERT OR IGNORE INTO schema_migrations (version, description) VALUES (?1, ?2)"
      )

    :ok = Exqlite.Sqlite3.bind(stmt, [version, description])
    :done = Exqlite.Sqlite3.step(db, stmt)
    Exqlite.Sqlite3.release(db, stmt)
    :ok
  end

  # Executes a statement and tolerates only idempotency errors. Unexpected
  # failures must stop the migration so its version is never recorded as done.
  defp safe_execute(db, label, sql) do
    case Exqlite.Sqlite3.execute(db, sql) do
      :ok ->
        :ok

      {:error, msg} when is_binary(msg) ->
        cond do
          String.contains?(msg, "duplicate column") ->
            :ok

          String.contains?(msg, "already exists") ->
            :ok

          true ->
            raise "migration statement #{label} failed: #{msg}"
        end

      {:error, reason} ->
        raise "migration statement #{label} failed: #{inspect(reason)}"
    end
  end

  defp pad(n) when is_integer(n), do: n |> to_string() |> String.pad_leading(3, "0")
end
