defmodule OptimalEngine.Store do
  @moduledoc """
  Owns the SQLite connection and an ETS hot cache.

  ## Schema

  The canonical table is `contexts` — it stores all context types (resource,
  memory, skill, signal). The `signals` view provides backward compatibility
  so that existing raw SQL queries (L0Cache, stats tasks, etc.) continue to work
  without changes.

  - `contexts`      — universal context table (replaces `signals`)
  - `signals`       — VIEW over contexts WHERE type = 'signal'
  - `contexts_fts`  — FTS5 virtual table (title + content)
  - `entities`      — named entities extracted from contexts
  - `edges`         — context-to-context relationships
  - `decisions`     — decision log entries
  - `sessions`      — session tracking

  ## ETS hot cache

  This GenServer owns an internal ETS table (`:optimal_engine_store_cache`)
  that holds recent/hot contexts keyed by ID. Entries are evicted on a simple
  LRU watermark when capacity is hit.

  All public functions return `{:ok, result}` or `{:error, reason}` — no bare
  raises outside of init (where crashing is acceptable per OTP let-it-crash).
  The one deliberate exception: `transaction/2` re-raises exceptions raised by
  the *caller's own fun* (after rolling back), so bugs inside a transaction fun
  surface in the calling process instead of being swallowed as error tuples.

  ## Transactions

  `transaction/1` runs a caller-supplied fun inside one SQLite transaction
  (`BEGIN IMMEDIATE` .. `COMMIT`/`ROLLBACK`) executed within a SINGLE GenServer
  call, so a whole read-check-write sequence is atomic and serialized against
  every other Store operation. This is the primitive that FactPromoter atomic
  promotion and ActiveMemoryPool atomic pool mutations build on.

  ### Contract

      Store.transaction(fn txn ->
        {:ok, rows} = Store.txn_query(txn, "SELECT ...", [param])
        {:ok, _changes} = Store.txn_execute(txn, "UPDATE ...", [param])
        {:ok, result}
      end)
      #=> {:ok, result} | {:error, reason}

  - The fun receives an opaque transaction handle and must run every statement
    through `txn_query/3` / `txn_execute/3`. Calling any other
    `OptimalEngine.Store` function inside the fun deadlocks (the fun runs in
    the Store GenServer process).
  - Fun returns `{:ok, result}` → `COMMIT`, `transaction/2` returns
    `{:ok, result}`. If the commit itself fails, the transaction is rolled
    back and `{:error, {:commit_failed, reason}}` is returned.
  - Fun returns `{:error, reason}` → `ROLLBACK`, `transaction/2` returns
    `{:error, reason}` unchanged.
  - Fun raises/throws/exits → `ROLLBACK`, then the exception is re-raised in
    the calling process with its original stacktrace (the Store GenServer
    itself stays alive).
  - Any other return value → `ROLLBACK` (fail closed),
    `{:error, {:invalid_transaction_return, other}}`.
  """

  use GenServer
  require Logger

  alias OptimalEngine.{Context, Graph, Signal}

  @ets_table :optimal_engine_store_cache
  @ets_max 500

  # ---------------------------------------------------------------------------
  # Schema DDL
  # ---------------------------------------------------------------------------

  @ddl_contexts """
  CREATE TABLE IF NOT EXISTS contexts (
    id TEXT PRIMARY KEY,
    uri TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'resource',
    path TEXT,
    title TEXT NOT NULL DEFAULT '',
    l0_abstract TEXT NOT NULL DEFAULT '',
    l1_overview TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    mode TEXT,
    genre TEXT,
    signal_type TEXT,
    format TEXT,
    structure TEXT,
    node TEXT NOT NULL DEFAULT 'inbox',
    sn_ratio REAL NOT NULL DEFAULT 0.5,
    entities TEXT NOT NULL DEFAULT '[]',
    created_at TEXT,
    modified_at TEXT,
    valid_from TEXT,
    valid_until TEXT,
    supersedes TEXT,
    routed_to TEXT NOT NULL DEFAULT '[]',
    metadata TEXT NOT NULL DEFAULT '{}',
    indexed_at TEXT NOT NULL DEFAULT (datetime('now'))
  )
  """

  # Backward-compat view: exposes all signal-dimension columns under the old names
  # so L0Cache and stats queries against `signals` continue to work.
  @ddl_signals_view """
  CREATE VIEW IF NOT EXISTS signals AS
  SELECT
    id,
    path,
    title,
    COALESCE(mode, 'linguistic') AS mode,
    COALESCE(genre, 'note')      AS genre,
    COALESCE(signal_type, 'inform') AS type,
    COALESCE(format, 'markdown') AS format,
    structure,
    created_at,
    modified_at,
    valid_from,
    valid_until,
    supersedes,
    node,
    sn_ratio,
    entities,
    l0_abstract     AS l0_summary,
    l1_overview     AS l1_description,
    content,
    routed_to,
    indexed_at
  FROM contexts
  WHERE type = 'signal'
  """

  @ddl_fts """
  CREATE VIRTUAL TABLE IF NOT EXISTS contexts_fts USING fts5(
    id UNINDEXED,
    title,
    content,
    node UNINDEXED,
    type UNINDEXED,
    genre UNINDEXED
  )
  """

  @ddl_fts_insert_trigger """
  CREATE TRIGGER IF NOT EXISTS contexts_fts_insert AFTER INSERT ON contexts BEGIN
    INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre)
    VALUES (new.rowid, new.id, new.title, new.content, new.node, new.type, COALESCE(new.genre, ''));
  END
  """

  @ddl_fts_update_trigger """
  CREATE TRIGGER IF NOT EXISTS contexts_fts_update AFTER UPDATE ON contexts BEGIN
    DELETE FROM contexts_fts WHERE rowid = old.rowid;
    INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre)
    VALUES (new.rowid, new.id, new.title, new.content, new.node, new.type, COALESCE(new.genre, ''));
  END
  """

  @ddl_fts_delete_trigger """
  CREATE TRIGGER IF NOT EXISTS contexts_fts_delete AFTER DELETE ON contexts BEGIN
    DELETE FROM contexts_fts WHERE rowid = old.rowid;
  END
  """

  @ddl_entities """
  CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    context_id TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'person',
    UNIQUE(context_id, name)
  )
  """

  @ddl_edges """
  CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relation TEXT NOT NULL DEFAULT 'related',
    weight REAL NOT NULL DEFAULT 1.0,
    valid_from TEXT,
    valid_until TEXT,
    UNIQUE(source_id, target_id, relation)
  )
  """

  @ddl_decisions """
  CREATE TABLE IF NOT EXISTS decisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    decision TEXT NOT NULL,
    rationale TEXT,
    decided_at TEXT NOT NULL DEFAULT (datetime('now')),
    decided_by TEXT
  )
  """

  @ddl_sessions """
  CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    committed_at TEXT,
    summary TEXT NOT NULL DEFAULT '',
    message_count INTEGER NOT NULL DEFAULT 0,
    metadata TEXT NOT NULL DEFAULT '{}'
  )
  """

  @ddl_vectors """
  CREATE TABLE IF NOT EXISTS vectors (
    context_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model TEXT NOT NULL DEFAULT 'nomic-embed-text',
    dimensions INTEGER NOT NULL DEFAULT 768,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
  )
  """

  @ddl_observations """
  CREATE TABLE IF NOT EXISTS observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    content TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 0.6,
    source TEXT NOT NULL DEFAULT 'explicit',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
  )
  """

  # ---------------------------------------------------------------------------
  # Client API
  # ---------------------------------------------------------------------------

  @doc "Starts the Store GenServer."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Inserts or replaces a Context in SQLite + ETS cache."
  @spec insert_context(Context.t()) :: :ok | {:error, term()}
  def insert_context(%Context{} = context) do
    GenServer.call(__MODULE__, {:insert_context, context}, 10_000)
  end

  @doc "Inserts multiple contexts in a single transaction."
  @spec insert_contexts([Context.t()]) :: {:ok, non_neg_integer()} | {:error, term()}
  def insert_contexts(contexts) when is_list(contexts) do
    GenServer.call(__MODULE__, {:insert_contexts, contexts}, 60_000)
  end

  @doc """
  Inserts or replaces a Signal (backward compat).
  Wraps the signal in a Context struct of type :signal before storing.
  """
  @spec insert_signal(Signal.t()) :: :ok | {:error, term()}
  def insert_signal(%Signal{} = signal) do
    insert_context(Context.from_signal(signal))
  end

  @doc "Inserts multiple signals in a single transaction (backward compat)."
  @spec insert_signals([Signal.t()]) :: {:ok, non_neg_integer()} | {:error, term()}
  def insert_signals(signals) when is_list(signals) do
    contexts = Enum.map(signals, &Context.from_signal/1)
    insert_contexts(contexts)
  end

  @doc "Retrieves a Context by ID. Checks ETS cache first, then SQLite."
  @spec get_context(String.t()) :: {:ok, Context.t()} | {:error, :not_found}
  def get_context(id) when is_binary(id) do
    case :ets.lookup(@ets_table, id) do
      [{^id, ctx}] when is_struct(ctx, Context) ->
        :telemetry.execute([:optimal_engine, :store, :cache_hit], %{count: 1}, %{})
        {:ok, ctx}

      _ ->
        GenServer.call(__MODULE__, {:get_context, id})
    end
  end

  @doc "Retrieves a Signal by ID (backward compat). Returns the embedded signal."
  @spec get_signal(String.t()) :: {:ok, Signal.t()} | {:error, :not_found}
  def get_signal(id) when is_binary(id) do
    case get_context(id) do
      {:ok, ctx} -> {:ok, Context.to_signal(ctx)}
      err -> err
    end
  end

  @doc "Returns all contexts belonging to a node."
  @spec get_by_node(String.t(), keyword()) :: {:ok, [Context.t()]} | {:error, term()}
  def get_by_node(node, opts \\ []) when is_binary(node) do
    type_filter = Keyword.get(opts, :type)
    GenServer.call(__MODULE__, {:get_by_node, node, type_filter}, 10_000)
  end

  @doc "Returns aggregated stats about the store."
  @spec stats() :: {:ok, map()} | {:error, term()}
  def stats do
    GenServer.call(__MODULE__, :stats)
  end

  @doc "Deletes a context by ID."
  @spec delete_context(String.t()) :: :ok | {:error, term()}
  def delete_context(id) when is_binary(id) do
    GenServer.call(__MODULE__, {:delete_context, id})
  end

  @doc "Deletes a signal by ID (backward compat)."
  @spec delete_signal(String.t()) :: :ok | {:error, term()}
  def delete_signal(id), do: delete_context(id)

  @doc "Executes a raw SELECT query. Returns list of rows."
  @spec raw_query(String.t(), [term()]) :: {:ok, [[term()]]} | {:error, term()}
  def raw_query(sql, params \\ []) do
    GenServer.call(__MODULE__, {:raw_query, sql, params}, 15_000)
  end

  @doc "Executes a raw non-SELECT statement with bound parameters."
  @spec raw_execute(String.t(), [term()]) :: :ok | {:error, term()}
  def raw_execute(sql, params \\ []) do
    GenServer.call(__MODULE__, {:raw_execute, sql, params}, 15_000)
  end

  @doc """
  Runs `fun` inside a single SQLite transaction (`BEGIN IMMEDIATE` ..
  `COMMIT`/`ROLLBACK`) executed within ONE Store GenServer call, so the whole
  read-check-write sequence is atomic and serialized against every other
  Store operation.

  `fun` receives an opaque transaction handle and must run every statement
  through `txn_query/3` / `txn_execute/3` (calling other `OptimalEngine.Store`
  functions inside the fun would deadlock the GenServer). Return
  `{:ok, result}` to commit or `{:error, reason}` to roll back. If the fun
  raises (or throws/exits), the transaction is rolled back and the exception
  is re-raised here in the calling process with its original stacktrace.

  See the module documentation ("Transactions") for the full contract.
  """
  @spec transaction((term() -> {:ok, term()} | {:error, term()}), timeout()) ::
          {:ok, term()} | {:error, term()}
  def transaction(fun, timeout \\ 30_000) when is_function(fun, 1) do
    case GenServer.call(__MODULE__, {:transaction, fun}, timeout) do
      {:__txn_raise__, kind, reason, stacktrace} -> :erlang.raise(kind, reason, stacktrace)
      result -> result
    end
  end

  @doc "Executes a SELECT inside a `transaction/2` fun. Returns `{:ok, rows}`."
  @spec txn_query(term(), String.t(), [term()]) :: {:ok, [[term()]]} | {:error, term()}
  def txn_query({:txn, db}, sql, params \\ []) do
    with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
         :ok <- Exqlite.Sqlite3.bind(stmt, params) do
      rows = collect_rows_raw(db, stmt, [])
      Exqlite.Sqlite3.release(db, stmt)
      {:ok, rows}
    end
  end

  @doc """
  Executes a non-SELECT statement inside a `transaction/2` fun. Returns
  `{:ok, changes}` with the number of rows affected, so callers can detect
  conditional updates that matched zero rows.
  """
  @spec txn_execute(term(), String.t(), [term()]) ::
          {:ok, non_neg_integer()} | {:error, term()}
  def txn_execute({:txn, db} = txn, sql, params \\ []) do
    with :ok <- exec_stmt(db, sql, params),
         {:ok, [[changes]]} <- txn_query(txn, "SELECT changes()") do
      {:ok, changes}
    end
  end

  @doc """
  Inserts or replaces a list of chunks in one transaction.

  Accepts anything that behaves like `OptimalEngine.Pipeline.Decomposer.Chunk`
  (struct or map with the right keys). Idempotent on `chunk.id`.
  """
  @spec insert_chunks([struct() | map()]) :: :ok | {:error, term()}
  def insert_chunks(chunks) when is_list(chunks) do
    GenServer.call(__MODULE__, {:insert_chunks, chunks}, 60_000)
  end

  @doc """
  Upserts per-chunk classifications (Phase 4).
  Accepts `%OptimalEngine.Pipeline.Classifier.Classification{}` structs
  or equivalent maps. Idempotent on `chunk_id`.
  """
  @spec insert_classifications([struct() | map()]) :: :ok | {:error, term()}
  def insert_classifications(rows) when is_list(rows) do
    GenServer.call(__MODULE__, {:insert_classifications, rows}, 60_000)
  end

  @doc """
  Upserts per-chunk intents (Phase 4).
  Accepts `%OptimalEngine.Pipeline.IntentExtractor.Intent{}` structs
  or equivalent maps. Idempotent on `chunk_id`.
  """
  @spec insert_intents([struct() | map()]) :: :ok | {:error, term()}
  def insert_intents(rows) when is_list(rows) do
    GenServer.call(__MODULE__, {:insert_intents, rows}, 60_000)
  end

  @doc """
  Upserts per-chunk embeddings into `chunk_embeddings` (Phase 5).
  Accepts `%OptimalEngine.Pipeline.Embedder.Embedding{}` structs
  (or equivalent maps). Idempotent on `chunk_id`.

  Vectors are stored as little-endian 32-bit float BLOBs — 768 × 4 = 3072
  bytes per row for nomic-class models.
  """
  @spec insert_embeddings([struct() | map()]) :: :ok | {:error, term()}
  def insert_embeddings(rows) when is_list(rows) do
    GenServer.call(__MODULE__, {:insert_embeddings, rows}, 60_000)
  end

  @doc """
  Upserts clusters (Phase 6). Idempotent on `cluster.id`.

  Accepts `%OptimalEngine.Pipeline.Clusterer.Cluster{}` structs or maps
  with the right keys. Centroid vectors are persisted as little-endian
  float32 BLOBs (same encoding as chunk_embeddings.vector).
  """
  @spec insert_clusters([struct() | map()]) :: :ok | {:error, term()}
  def insert_clusters(rows) when is_list(rows) do
    GenServer.call(__MODULE__, {:insert_clusters, rows}, 60_000)
  end

  @doc """
  Upserts cluster-membership rows into `cluster_members` (Phase 6).
  Idempotent on `(cluster_id, chunk_id)`.

  Each row: `%{cluster_id, chunk_id, tenant_id, weight}`.
  """
  @spec insert_cluster_members([map()]) :: :ok | {:error, term()}
  def insert_cluster_members(rows) when is_list(rows) do
    GenServer.call(__MODULE__, {:insert_cluster_members, rows}, 60_000)
  end

  # ---------------------------------------------------------------------------
  # GenServer Callbacks
  # ---------------------------------------------------------------------------

  @impl true
  def init(_opts) do
    db_path = Application.get_env(:optimal_engine, :db_path)

    :ets.new(@ets_table, [:named_table, :set, :public, {:read_concurrency, true}])

    case open_and_migrate(db_path) do
      {:ok, db} ->
        Logger.info("[Store] SQLite opened at #{db_path}")
        :telemetry.execute([:optimal_engine, :store, :init], %{}, %{db_path: db_path})
        {:ok, %{db: db, db_path: db_path}}

      {:error, reason} ->
        {:stop, {:db_open_failed, reason}}
    end
  end

  @impl true
  def handle_call({:insert_context, context}, _from, state) do
    result = do_insert(state.db, context)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_contexts, contexts}, _from, state) do
    result = insert_in_transaction(state.db, contexts)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_chunks, chunks}, _from, state) do
    result = insert_chunks_in_transaction(state.db, chunks)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_classifications, rows}, _from, state) do
    result = insert_classifications_in_transaction(state.db, rows)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_intents, rows}, _from, state) do
    result = insert_intents_in_transaction(state.db, rows)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_embeddings, rows}, _from, state) do
    result = insert_embeddings_in_transaction(state.db, rows)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_clusters, rows}, _from, state) do
    result = insert_clusters_in_transaction(state.db, rows)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:insert_cluster_members, rows}, _from, state) do
    result = insert_cluster_members_in_transaction(state.db, rows)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:get_context, id}, _from, state) do
    sql = "SELECT #{context_columns()} FROM contexts WHERE id = ?1"

    result =
      with {:ok, stmt} <- Exqlite.Sqlite3.prepare(state.db, sql),
           :ok <- Exqlite.Sqlite3.bind(stmt, [id]),
           {:row, row} <- Exqlite.Sqlite3.step(state.db, stmt),
           :ok <- Exqlite.Sqlite3.release(state.db, stmt) do
        ctx = Context.from_row(row)
        cache_put(id, ctx)
        {:ok, ctx}
      else
        :done -> {:error, :not_found}
        {:error, reason} -> {:error, reason}
      end

    :telemetry.execute([:optimal_engine, :store, :cache_miss], %{count: 1}, %{})
    {:reply, result, state}
  end

  @impl true
  def handle_call({:get_by_node, node, type_filter}, _from, state) do
    {sql, params} =
      if type_filter do
        {
          "SELECT #{context_columns()} FROM contexts WHERE node = ?1 AND type = ?2 ORDER BY modified_at DESC",
          [node, to_string(type_filter)]
        }
      else
        {
          "SELECT #{context_columns()} FROM contexts WHERE node = ?1 ORDER BY modified_at DESC",
          [node]
        }
      end

    result =
      with {:ok, stmt} <- Exqlite.Sqlite3.prepare(state.db, sql),
           :ok <- Exqlite.Sqlite3.bind(stmt, params) do
        contexts = collect_context_rows(state.db, stmt, [])
        Exqlite.Sqlite3.release(state.db, stmt)
        {:ok, contexts}
      end

    {:reply, result, state}
  end

  @impl true
  def handle_call(:stats, _from, state) do
    queries = [
      {"total_contexts", "SELECT COUNT(*) FROM contexts"},
      {"total_signals", "SELECT COUNT(*) FROM contexts WHERE type = 'signal'"},
      {"total_resources", "SELECT COUNT(*) FROM contexts WHERE type = 'resource'"},
      {"total_memories", "SELECT COUNT(*) FROM contexts WHERE type = 'memory'"},
      {"total_skills", "SELECT COUNT(*) FROM contexts WHERE type = 'skill'"},
      {"total_entities", "SELECT COUNT(*) FROM entities"},
      {"total_edges", "SELECT COUNT(*) FROM edges"},
      {"total_decisions", "SELECT COUNT(*) FROM decisions"}
    ]

    result =
      Enum.reduce_while(queries, %{}, fn {key, sql}, acc ->
        case single_value(state.db, sql) do
          {:ok, val} -> {:cont, Map.put(acc, key, val)}
          {:error, _} = err -> {:halt, err}
        end
      end)

    case result do
      %{} = store_stats ->
        store_stats = Map.put(store_stats, "cache_size", :ets.info(@ets_table, :size))
        {:reply, {:ok, store_stats}, state}

      err ->
        {:reply, err, state}
    end
  end

  @impl true
  def handle_call({:delete_context, id}, _from, state) do
    result = Exqlite.Sqlite3.execute(state.db, "DELETE FROM contexts WHERE id = '#{escape(id)}'")
    :ets.delete(@ets_table, id)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:raw_query, sql, params}, _from, state) do
    result =
      with {:ok, stmt} <- Exqlite.Sqlite3.prepare(state.db, sql),
           :ok <- Exqlite.Sqlite3.bind(stmt, params) do
        rows = collect_rows_raw(state.db, stmt, [])
        Exqlite.Sqlite3.release(state.db, stmt)
        {:ok, rows}
      end

    {:reply, result, state}
  end

  @impl true
  def handle_call({:raw_execute, sql, params}, _from, state) do
    result = exec_stmt(state.db, sql, params)
    {:reply, result, state}
  end

  @impl true
  def handle_call({:transaction, fun}, _from, state) do
    {:reply, run_transaction(state.db, fun), state}
  end

  @impl true
  def terminate(_reason, %{db: db}) do
    Exqlite.Sqlite3.close(db)
  end

  # ---------------------------------------------------------------------------
  # Private: Migration + Setup
  # ---------------------------------------------------------------------------

  defp open_and_migrate(db_path) do
    File.mkdir_p!(Path.dirname(db_path))

    with {:ok, db} <- Exqlite.Sqlite3.open(db_path),
         :ok <- Exqlite.Sqlite3.execute(db, "PRAGMA journal_mode=WAL"),
         :ok <- Exqlite.Sqlite3.execute(db, "PRAGMA foreign_keys=ON"),
         :ok <- Exqlite.Sqlite3.execute(db, "PRAGMA recursive_triggers=ON"),
         :ok <- Exqlite.Sqlite3.execute(db, "PRAGMA synchronous=NORMAL"),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_contexts),
         :ok <- migrate_from_signals_table(db),
         :ok <- run_contexts_column_migrations(db),
         :ok <- ensure_signals_view(db),
         :ok <- drop_legacy_triggers(db),
         :ok <- ensure_fts_schema(db),
         :ok <- run_fts_triggers(db),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_entities),
         :ok <- migrate_entities_column(db),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_edges),
         :ok <- migrate_edges_column(db),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_decisions),
         :ok <- migrate_decisions_column(db),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_sessions),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_vectors),
         :ok <- Exqlite.Sqlite3.execute(db, @ddl_observations),
         :ok <- run_index_migrations(db),
         :ok <- run_probability_column_migrations(db),
         :ok <- OptimalEngine.Store.Migrations.run(db) do
      {:ok, db}
    end
  end

  # Add probability column to edges and capacity_hours to entities.
  # ALTER TABLE fails if the column already exists — that's fine, we rescue it.
  defp run_probability_column_migrations(db) do
    migrations = [
      {"edges", "ALTER TABLE edges ADD COLUMN probability REAL NOT NULL DEFAULT 0.8"},
      {"entities_cap", "ALTER TABLE entities ADD COLUMN capacity_hours REAL DEFAULT 40.0"}
    ]

    Enum.each(migrations, fn {label, sql} ->
      case Exqlite.Sqlite3.execute(db, sql) do
        :ok ->
          Logger.info("[Store] Probability/capacity column added (#{label})")

        {:error, msg} when is_binary(msg) ->
          if String.contains?(msg, "duplicate column") do
            :ok
          else
            Logger.warning("[Store] Column migration #{label} failed: #{inspect(msg)}")
          end

        {:error, reason} ->
          Logger.warning("[Store] Column migration #{label} failed: #{inspect(reason)}")
      end
    end)

    :ok
  end

  defp run_index_migrations(db) do
    indexes = [
      "CREATE INDEX IF NOT EXISTS idx_contexts_node ON contexts(node)",
      "CREATE INDEX IF NOT EXISTS idx_contexts_type ON contexts(type)",
      "CREATE INDEX IF NOT EXISTS idx_contexts_genre ON contexts(genre)",
      "CREATE INDEX IF NOT EXISTS idx_contexts_modified ON contexts(modified_at)",
      "CREATE INDEX IF NOT EXISTS idx_entities_context_id ON entities(context_id)",
      "CREATE INDEX IF NOT EXISTS idx_entities_name ON entities(name)",
      "CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id)",
      "CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id)",
      "CREATE INDEX IF NOT EXISTS idx_edges_relation ON edges(relation)"
    ]

    Enum.each(indexes, fn sql ->
      case Exqlite.Sqlite3.execute(db, sql) do
        :ok -> :ok
        {:error, reason} -> Logger.warning("[Store] Index creation failed: #{inspect(reason)}")
      end
    end)

    :ok
  end

  # NOTE: the legacy node renames ("01-inbox" → "inbox", "04-products" →
  # "product-customer-portal") used to run here on every init, rewriting node
  # names across ALL workspaces. They now live in Migrations as a one-time,
  # workspace-scoped backfill (migration 035) — see
  # `OptimalEngine.Store.Migrations.migration_035_connector_workspace_and_legacy_node_renames/0`.

  # If an old `signals` TABLE exists (not view), migrate its data to `contexts`
  # then drop it so our VIEW can be created.
  defp migrate_from_signals_table(db) do
    case table_type(db, "signals") do
      "table" ->
        Logger.info("[Store] Migrating signals table → contexts table")
        do_migrate_signals_to_contexts(db)

      _ ->
        :ok
    end
  end

  defp do_migrate_signals_to_contexts(db) do
    # Insert existing signal rows into contexts as type='signal'
    migrate_sql = """
    INSERT OR IGNORE INTO contexts (
      id, uri, type, path, title,
      l0_abstract, l1_overview, content,
      mode, genre, signal_type, format, structure,
      node, sn_ratio, entities,
      created_at, modified_at, valid_from, valid_until, supersedes,
      routed_to, metadata, indexed_at
    )
    SELECT
      id,
      'optimal://nodes/' || COALESCE(node, 'inbox') || '/' || COALESCE(path, ''),
      'signal',
      path,
      title,
      COALESCE(l0_summary, ''),
      COALESCE(l1_description, ''),
      COALESCE(content, ''),
      mode,
      genre,
      type,
      format,
      structure,
      node,
      sn_ratio,
      entities,
      created_at,
      modified_at,
      valid_from,
      valid_until,
      supersedes,
      COALESCE(routed_to, '[]'),
      '{}',
      COALESCE(indexed_at, datetime('now'))
    FROM signals
    """

    case Exqlite.Sqlite3.execute(db, migrate_sql) do
      :ok ->
        Logger.info("[Store] Signals migrated to contexts. Dropping signals table.")

        for trigger <-
              ~w[signals_ai signals_au signals_ad signals_fts_insert signals_fts_update signals_fts_delete] do
          Exqlite.Sqlite3.execute(db, "DROP TRIGGER IF EXISTS #{trigger}")
        end

        Exqlite.Sqlite3.execute(db, "DROP TABLE IF EXISTS signals_fts")
        Exqlite.Sqlite3.execute(db, "DROP TABLE IF EXISTS signals")
        :ok

      {:error, reason} ->
        Logger.warning("[Store] Signal migration failed: #{inspect(reason)}. Proceeding anyway.")
        :ok
    end
  end

  defp table_type(db, name) do
    sql = "SELECT type FROM sqlite_master WHERE name = '#{escape(name)}'"

    case Exqlite.Sqlite3.prepare(db, sql) do
      {:ok, stmt} ->
        result = Exqlite.Sqlite3.step(db, stmt)
        Exqlite.Sqlite3.release(db, stmt)

        case result do
          {:row, [type]} -> type
          _ -> nil
        end

      _ ->
        nil
    end
  end

  defp ensure_signals_view(db) do
    # Only create if it doesn't exist yet (migration dropped the old table)
    case Exqlite.Sqlite3.execute(db, @ddl_signals_view) do
      :ok ->
        :ok

      # View already exists — that's fine
      {:error, msg} when is_binary(msg) ->
        if String.contains?(msg, "already exists"), do: :ok, else: {:error, msg}
    end
  end

  defp run_contexts_column_migrations(db) do
    existing = get_column_names(db, "contexts")

    migrations = [
      {"uri", "ALTER TABLE contexts ADD COLUMN uri TEXT NOT NULL DEFAULT ''"},
      {"metadata", "ALTER TABLE contexts ADD COLUMN metadata TEXT NOT NULL DEFAULT '{}'"},
      {"indexed_at", "ALTER TABLE contexts ADD COLUMN indexed_at TEXT"}
    ]

    Enum.each(migrations, fn {col, add_sql} ->
      unless col in existing, do: run_column_add(db, col, add_sql)
    end)

    :ok
  end

  defp run_column_add(db, col, sql) do
    case Exqlite.Sqlite3.execute(db, sql) do
      :ok ->
        :ok

      {:error, reason} ->
        Logger.warning("[Store] Column migration for #{col} failed: #{inspect(reason)}")
    end
  end

  # Migrate entities table: rename signal_id → context_id if needed
  defp migrate_entities_column(db) do
    existing = get_column_names(db, "entities")

    if "signal_id" in existing and "context_id" not in existing do
      Logger.info("[Store] Migrating entities.signal_id → context_id")

      [
        "ALTER TABLE entities RENAME TO entities_old",
        """
        CREATE TABLE entities (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          context_id TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
          name TEXT NOT NULL,
          type TEXT NOT NULL DEFAULT 'person',
          UNIQUE(context_id, name)
        )
        """,
        "INSERT OR IGNORE INTO entities (id, context_id, name, type) SELECT id, signal_id, name, type FROM entities_old",
        "DROP TABLE entities_old"
      ]
      |> Enum.each(&run_migration_sql(db, &1, "Entity"))
    end

    :ok
  end

  # Migrate edges table to the new schema (no FK constraints, relation/valid_from columns).
  # The old schema had `relationship` and FK references to contexts — recreate without them.
  defp migrate_edges_column(db) do
    existing = get_column_names(db, "edges")

    needs_migration =
      "relationship" in existing or
        ("source_id" in existing and "relation" not in existing) or
        ("source_id" in existing and "valid_from" not in existing)

    if needs_migration do
      Logger.info("[Store] Migrating edges table to new schema (no FK, relation column)")

      [
        "ALTER TABLE edges RENAME TO edges_old",
        """
        CREATE TABLE edges (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          source_id TEXT NOT NULL,
          target_id TEXT NOT NULL,
          relation TEXT NOT NULL DEFAULT 'related',
          weight REAL NOT NULL DEFAULT 1.0,
          valid_from TEXT,
          valid_until TEXT,
          UNIQUE(source_id, target_id, relation)
        )
        """,
        """
        INSERT OR IGNORE INTO edges (id, source_id, target_id, relation, weight)
        SELECT id, source_id, target_id,
               COALESCE(relationship, 'related'),
               weight
        FROM edges_old
        """,
        "DROP TABLE edges_old"
      ]
      |> Enum.each(&run_migration_sql(db, &1, "Edges"))
    end

    :ok
  end

  # Migrate decisions table: rename signal_id → context_id if needed
  defp migrate_decisions_column(db) do
    existing = get_column_names(db, "decisions")

    if "signal_id" in existing and "context_id" not in existing do
      Logger.info("[Store] Migrating decisions.signal_id → context_id")

      [
        "ALTER TABLE decisions RENAME TO decisions_old",
        """
        CREATE TABLE decisions (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          context_id TEXT REFERENCES contexts(id) ON DELETE SET NULL,
          title TEXT NOT NULL,
          decision TEXT NOT NULL,
          rationale TEXT,
          decided_at TEXT NOT NULL DEFAULT (datetime('now')),
          decided_by TEXT
        )
        """,
        "INSERT OR IGNORE INTO decisions (id, context_id, title, decision, rationale, decided_at, decided_by) SELECT id, signal_id, title, decision, rationale, decided_at, decided_by FROM decisions_old",
        "DROP TABLE decisions_old"
      ]
      |> Enum.each(&run_migration_sql(db, &1, "Decision"))
    end

    :ok
  end

  defp run_migration_sql(db, sql, label) do
    case Exqlite.Sqlite3.execute(db, String.trim(sql)) do
      :ok -> :ok
      {:error, r} -> Logger.warning("[Store] #{label} migration step failed: #{inspect(r)}")
    end
  end

  defp drop_legacy_triggers(db) do
    legacy = ~w[
      signals_ai signals_au signals_ad
      signals_fts_insert signals_fts_update signals_fts_delete
    ]

    Enum.each(legacy, fn t ->
      Exqlite.Sqlite3.execute(db, "DROP TRIGGER IF EXISTS #{t}")
    end)

    :ok
  end

  defp ensure_fts_schema(db) do
    case fts_has_correct_schema?(db) do
      true ->
        :ok

      false ->
        Logger.info("[Store] Rebuilding FTS table for contexts schema")

        for t <- ~w[contexts_fts_insert contexts_fts_update contexts_fts_delete] do
          Exqlite.Sqlite3.execute(db, "DROP TRIGGER IF EXISTS #{t}")
        end

        Exqlite.Sqlite3.execute(db, "DROP TABLE IF EXISTS contexts_fts")
        # Also drop old signals_fts if it lingered
        Exqlite.Sqlite3.execute(db, "DROP TABLE IF EXISTS signals_fts")
        :ok = Exqlite.Sqlite3.execute(db, @ddl_fts)
        backfill_fts(db)
    end
  end

  defp fts_has_correct_schema?(db) do
    # Check that contexts_fts exists and has a 'type' column
    case Exqlite.Sqlite3.prepare(db, "SELECT type FROM contexts_fts LIMIT 0") do
      {:ok, stmt} ->
        Exqlite.Sqlite3.release(db, stmt)
        true

      {:error, _} ->
        false
    end
  end

  defp backfill_fts(db) do
    Logger.info("[Store] Backfilling FTS from contexts table...")
    sql = "SELECT rowid, id, title, content, node, type, COALESCE(genre, '') FROM contexts"

    case Exqlite.Sqlite3.prepare(db, sql) do
      {:ok, stmt} ->
        :ok = Exqlite.Sqlite3.bind(stmt, [])
        count = do_backfill_fts_loop(db, stmt, 0)
        Exqlite.Sqlite3.release(db, stmt)
        Logger.info("[Store] FTS backfill complete: #{count} contexts")
        :ok

      {:error, reason} ->
        Logger.warning("[Store] FTS backfill failed to prepare: #{inspect(reason)}")
        :ok
    end
  end

  defp do_backfill_fts_loop(db, stmt, count) do
    case Exqlite.Sqlite3.step(db, stmt) do
      {:row, [rowid, id, title, content, node, type, genre]} ->
        ins =
          "INSERT INTO contexts_fts(rowid, id, title, content, node, type, genre) VALUES (?, ?, ?, ?, ?, ?, ?)"

        case Exqlite.Sqlite3.prepare(db, ins) do
          {:ok, ins_stmt} ->
            Exqlite.Sqlite3.bind(ins_stmt, [rowid, id, title, content, node, type, genre])
            Exqlite.Sqlite3.step(db, ins_stmt)
            Exqlite.Sqlite3.release(db, ins_stmt)

          _ ->
            :skip
        end

        do_backfill_fts_loop(db, stmt, count + 1)

      :done ->
        count

      {:error, _} ->
        count
    end
  end

  defp run_fts_triggers(db) do
    [@ddl_fts_insert_trigger, @ddl_fts_update_trigger, @ddl_fts_delete_trigger]
    |> Enum.reduce_while(:ok, fn ddl, :ok ->
      run_fts_trigger(db, ddl)
    end)
  end

  defp run_fts_trigger(db, ddl) do
    case Exqlite.Sqlite3.execute(db, String.trim(ddl)) do
      :ok -> {:cont, :ok}
      {:error, msg} when is_binary(msg) -> handle_trigger_error(msg)
    end
  end

  defp handle_trigger_error(msg) do
    if String.contains?(msg, "already exists") do
      {:cont, :ok}
    else
      {:halt, {:error, msg}}
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Insert
  # ---------------------------------------------------------------------------

  defp insert_in_transaction(db, contexts) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok -> commit_or_rollback(db, contexts)
      err -> err
    end
  end

  # Runs a caller fun inside BEGIN IMMEDIATE .. COMMIT/ROLLBACK on the owned
  # connection. Executes in the GenServer process, so the fun must only use
  # txn_query/3 and txn_execute/3 for database access. Exceptions raised by
  # the fun roll the transaction back and are returned as a `:__txn_raise__`
  # sentinel that `transaction/2` re-raises in the calling process — the
  # Store GenServer must survive caller bugs.
  defp run_transaction(db, fun) do
    case Exqlite.Sqlite3.execute(db, "BEGIN IMMEDIATE") do
      :ok ->
        try do
          case fun.({:txn, db}) do
            {:ok, result} ->
              case Exqlite.Sqlite3.execute(db, "COMMIT") do
                :ok ->
                  {:ok, result}

                {:error, reason} ->
                  Exqlite.Sqlite3.execute(db, "ROLLBACK")
                  {:error, {:commit_failed, reason}}
              end

            {:error, _} = err ->
              Exqlite.Sqlite3.execute(db, "ROLLBACK")
              err

            other ->
              Exqlite.Sqlite3.execute(db, "ROLLBACK")
              {:error, {:invalid_transaction_return, other}}
          end
        catch
          kind, reason ->
            stacktrace = __STACKTRACE__
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            {:__txn_raise__, kind, reason, stacktrace}
        end

      {:error, reason} ->
        {:error, {:begin_failed, reason}}
    end
  end

  # ──────────────────────────────────────────────────────────────────────────
  # Chunk inserts — Phase 1 `chunks` table; Phase 3 Decomposer writes here.
  # ──────────────────────────────────────────────────────────────────────────

  defp insert_chunks_in_transaction(_db, []), do: :ok

  defp insert_chunks_in_transaction(db, chunks) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(chunks, :ok, fn chunk, _acc ->
            case do_insert_chunk(db, chunk) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_chunk(db, chunk) do
    sql = """
    INSERT OR REPLACE INTO chunks
      (id, tenant_id, signal_id, parent_id, scale, offset_bytes, length_bytes,
       text, modality, asset_ref, classification_level, created_at, workspace_id)
    VALUES
      (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, COALESCE(?12, datetime('now')), ?13)
    """

    params = [
      chunk_field(chunk, :id),
      chunk_field(chunk, :tenant_id) || "default",
      chunk_field(chunk, :signal_id),
      chunk_field(chunk, :parent_id),
      chunk_field(chunk, :scale) |> to_string(),
      chunk_field(chunk, :offset_bytes) || 0,
      chunk_field(chunk, :length_bytes) || 0,
      chunk_field(chunk, :text) || "",
      chunk_field(chunk, :modality) |> to_string(),
      chunk_field(chunk, :asset_ref),
      chunk_field(chunk, :classification_level) || "internal",
      chunk_field(chunk, :created_at),
      chunk_field(chunk, :workspace_id) || "default"
    ]

    with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
         :ok <- Exqlite.Sqlite3.bind(stmt, params),
         :done <- Exqlite.Sqlite3.step(db, stmt),
         :ok <- Exqlite.Sqlite3.release(db, stmt) do
      :ok
    else
      {:error, reason} -> {:error, reason}
      other -> {:error, other}
    end
  end

  defp chunk_field(%{__struct__: _} = chunk, key), do: Map.get(chunk, key)

  defp chunk_field(chunk, key) when is_map(chunk),
    do: Map.get(chunk, key) || Map.get(chunk, to_string(key))

  # ──────────────────────────────────────────────────────────────────────────
  # Classifications — Phase 4 Classifier writes per-chunk classification rows.
  # ──────────────────────────────────────────────────────────────────────────

  defp insert_classifications_in_transaction(_db, []), do: :ok

  defp insert_classifications_in_transaction(db, rows) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(rows, :ok, fn row, _acc ->
            case do_insert_classification(db, row) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_classification(db, row) do
    sql = """
    INSERT INTO classifications
      (tenant_id, chunk_id, mode, genre, signal_type, format, structure,
       sn_ratio, confidence, workspace_id)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
    ON CONFLICT(chunk_id) DO UPDATE SET
      mode        = excluded.mode,
      genre       = excluded.genre,
      signal_type = excluded.signal_type,
      format      = excluded.format,
      structure   = excluded.structure,
      sn_ratio    = excluded.sn_ratio,
      confidence  = excluded.confidence,
      workspace_id = excluded.workspace_id
    """

    params = [
      chunk_field(row, :tenant_id) || "default",
      chunk_field(row, :chunk_id),
      atom_or_nil(chunk_field(row, :mode)),
      atom_or_nil(chunk_field(row, :genre)),
      atom_or_nil(chunk_field(row, :signal_type)),
      atom_or_nil(chunk_field(row, :format)),
      atom_or_nil(chunk_field(row, :structure)),
      chunk_field(row, :sn_ratio),
      chunk_field(row, :confidence),
      chunk_field(row, :workspace_id) || "default"
    ]

    exec_stmt(db, sql, params)
  end

  # ──────────────────────────────────────────────────────────────────────────
  # Intents — Phase 4 IntentExtractor writes per-chunk intent rows.
  # ──────────────────────────────────────────────────────────────────────────

  defp insert_intents_in_transaction(_db, []), do: :ok

  defp insert_intents_in_transaction(db, rows) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(rows, :ok, fn row, _acc ->
            case do_insert_intent(db, row) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_intent(db, row) do
    sql = """
    INSERT INTO intents (tenant_id, chunk_id, intent, confidence, evidence, workspace_id)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6)
    ON CONFLICT(chunk_id) DO UPDATE SET
      intent     = excluded.intent,
      confidence = excluded.confidence,
      evidence   = excluded.evidence,
      workspace_id = excluded.workspace_id
    """

    params = [
      chunk_field(row, :tenant_id) || "default",
      chunk_field(row, :chunk_id),
      atom_or_nil(chunk_field(row, :intent)) || "record_fact",
      chunk_field(row, :confidence),
      chunk_field(row, :evidence),
      chunk_field(row, :workspace_id) || "default"
    ]

    exec_stmt(db, sql, params)
  end

  defp atom_or_nil(nil), do: nil
  defp atom_or_nil(a) when is_atom(a), do: Atom.to_string(a)
  defp atom_or_nil(s) when is_binary(s), do: s

  # ──────────────────────────────────────────────────────────────────────────
  # Chunk embeddings — Phase 5 Embedder writes per-chunk aligned vectors.
  # Vectors encode as little-endian float32 BLOBs, 4 bytes per component.
  # ──────────────────────────────────────────────────────────────────────────

  defp insert_embeddings_in_transaction(_db, []), do: :ok

  defp insert_embeddings_in_transaction(db, rows) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(rows, :ok, fn row, _acc ->
            case do_insert_embedding(db, row) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_embedding(db, row) do
    vector = chunk_field(row, :vector) || []
    blob = encode_vector(vector)

    sql = """
    INSERT INTO chunk_embeddings
      (chunk_id, tenant_id, model, modality, dim, vector, workspace_id)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
    ON CONFLICT(chunk_id) DO UPDATE SET
      tenant_id    = excluded.tenant_id,
      model        = excluded.model,
      modality     = excluded.modality,
      dim          = excluded.dim,
      vector       = excluded.vector,
      workspace_id = excluded.workspace_id
    """

    params = [
      chunk_field(row, :chunk_id),
      chunk_field(row, :tenant_id) || "default",
      chunk_field(row, :model),
      atom_or_nil(chunk_field(row, :modality)) || "text",
      chunk_field(row, :dim) || length(vector),
      # Force BLOB binding — exqlite otherwise treats a raw binary as TEXT,
      # and float32 bytes frequently contain 0x00 which SQLite TEXT truncates.
      {:blob, blob},
      chunk_field(row, :workspace_id) || "default"
    ]

    exec_stmt(db, sql, params)
  end

  defp encode_vector(vector) when is_list(vector) do
    for f <- vector, into: <<>>, do: <<f::little-float-size(32)>>
  end

  # ──────────────────────────────────────────────────────────────────────────
  # Clusters — Phase 6 Clusterer writes cluster rows + membership edges.
  # ──────────────────────────────────────────────────────────────────────────

  defp insert_clusters_in_transaction(_db, []), do: :ok

  defp insert_clusters_in_transaction(db, rows) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(rows, :ok, fn row, _acc ->
            case do_insert_cluster(db, row) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_cluster(db, row) do
    centroid = chunk_field(row, :centroid) || []
    blob = encode_vector(centroid)

    sql = """
    INSERT INTO clusters
      (id, tenant_id, theme, intent_dominant, member_count, centroid, updated_at)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, COALESCE(?7, datetime('now')))
    ON CONFLICT(id) DO UPDATE SET
      theme           = excluded.theme,
      intent_dominant = excluded.intent_dominant,
      member_count    = excluded.member_count,
      centroid        = excluded.centroid,
      updated_at      = excluded.updated_at
    """

    params = [
      chunk_field(row, :id),
      chunk_field(row, :tenant_id) || "default",
      chunk_field(row, :theme) || "Unnamed cluster",
      atom_or_nil(chunk_field(row, :intent_dominant)),
      chunk_field(row, :member_count) || 0,
      {:blob, blob},
      chunk_field(row, :updated_at)
    ]

    exec_stmt(db, sql, params)
  end

  defp insert_cluster_members_in_transaction(_db, []), do: :ok

  defp insert_cluster_members_in_transaction(db, rows) do
    case Exqlite.Sqlite3.execute(db, "BEGIN") do
      :ok ->
        result =
          Enum.reduce_while(rows, :ok, fn row, _acc ->
            case do_insert_cluster_member(db, row) do
              :ok -> {:cont, :ok}
              {:error, _} = err -> {:halt, err}
            end
          end)

        case result do
          :ok ->
            Exqlite.Sqlite3.execute(db, "COMMIT")
            :ok

          {:error, _} = err ->
            Exqlite.Sqlite3.execute(db, "ROLLBACK")
            err
        end

      err ->
        err
    end
  end

  defp do_insert_cluster_member(db, row) do
    sql = """
    INSERT INTO cluster_members (tenant_id, cluster_id, chunk_id, weight)
    VALUES (?1, ?2, ?3, ?4)
    ON CONFLICT(cluster_id, chunk_id) DO UPDATE SET
      weight = excluded.weight
    """

    params = [
      chunk_field(row, :tenant_id) || "default",
      chunk_field(row, :cluster_id),
      chunk_field(row, :chunk_id),
      chunk_field(row, :weight) || 1.0
    ]

    exec_stmt(db, sql, params)
  end

  defp exec_stmt(db, sql, params) do
    with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
         :ok <- Exqlite.Sqlite3.bind(stmt, params),
         :done <- Exqlite.Sqlite3.step(db, stmt),
         :ok <- Exqlite.Sqlite3.release(db, stmt) do
      :ok
    else
      {:error, reason} -> {:error, reason}
      other -> {:error, other}
    end
  end

  defp commit_or_rollback(db, contexts) do
    count =
      Enum.reduce_while(contexts, 0, fn ctx, acc ->
        case do_insert(db, ctx) do
          :ok -> {:cont, acc + 1}
          {:error, _} = err -> {:halt, err}
        end
      end)

    case count do
      {:error, _} = err ->
        Exqlite.Sqlite3.execute(db, "ROLLBACK")
        err

      n ->
        Exqlite.Sqlite3.execute(db, "COMMIT")
        {:ok, n}
    end
  end

  defp do_insert(db, %Context{} = ctx) do
    row = Context.to_row(ctx)

    sql = """
    INSERT OR REPLACE INTO contexts (
      id, uri, type, path, title,
      l0_abstract, l1_overview, content,
      mode, genre, signal_type, format, structure,
      node, sn_ratio, entities,
      created_at, modified_at, valid_from, valid_until, supersedes,
      routed_to, metadata, workspace_id
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5,
      ?6, ?7, ?8,
      ?9, ?10, ?11, ?12, ?13,
      ?14, ?15, ?16,
      ?17, ?18, ?19, ?20, ?21,
      ?22, ?23, ?24
    )
    """

    params = [
      row.id,
      row.uri,
      row.type,
      row.path,
      row.title,
      row.l0_abstract,
      row.l1_overview,
      row.content,
      row.mode,
      row.genre,
      row.signal_type,
      row.format,
      row.structure,
      row.node,
      row.sn_ratio,
      row.entities,
      row.created_at,
      row.modified_at,
      row.valid_from,
      row.valid_until,
      row.supersedes,
      row.routed_to,
      row.metadata,
      row.workspace_id
    ]

    result =
      with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
           :ok <- Exqlite.Sqlite3.bind(stmt, params),
           :done <- Exqlite.Sqlite3.step(db, stmt),
           :ok <- Exqlite.Sqlite3.release(db, stmt) do
        insert_entities(db, ctx)
        Graph.create_edges_for_context_db(db, ctx)
        cache_put(ctx.id, ctx)
        :ok
      else
        {:error, reason} -> {:error, reason}
        unexpected -> {:error, {:unexpected_result, unexpected}}
      end

    :telemetry.execute(
      [:optimal_engine, :store, :insert],
      %{count: 1},
      %{node: ctx.node, type: ctx.type}
    )

    result
  end

  defp insert_entities(db, %Context{id: id, entities: entities} = ctx) when is_list(entities) do
    workspace_id = ctx.workspace_id || "default"

    Enum.each(entities, fn entity ->
      sql =
        "INSERT OR IGNORE INTO entities (context_id, name, type, workspace_id) VALUES ('#{escape(id)}', '#{escape(entity)}', 'person', '#{escape(workspace_id)}')"

      Exqlite.Sqlite3.execute(db, sql)
    end)
  end

  defp insert_entities(_db, _ctx), do: :ok

  # ---------------------------------------------------------------------------
  # Private: Collection helpers
  # ---------------------------------------------------------------------------

  defp collect_context_rows(db, stmt, acc) do
    case Exqlite.Sqlite3.step(db, stmt) do
      {:row, row} -> collect_context_rows(db, stmt, [Context.from_row(row) | acc])
      :done -> Enum.reverse(acc)
      {:error, _} -> Enum.reverse(acc)
    end
  end

  defp collect_rows_raw(db, stmt, acc) do
    case Exqlite.Sqlite3.step(db, stmt) do
      {:row, row} -> collect_rows_raw(db, stmt, [row | acc])
      :done -> Enum.reverse(acc)
      {:error, _} -> Enum.reverse(acc)
    end
  end

  defp single_value(db, sql) do
    with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
         {:row, [val]} <- Exqlite.Sqlite3.step(db, stmt),
         :ok <- Exqlite.Sqlite3.release(db, stmt) do
      {:ok, val}
    else
      :done -> {:ok, 0}
      {:error, _} = err -> err
    end
  end

  defp context_columns do
    """
    id, uri, type, path, title,
    l0_abstract, l1_overview, content,
    mode, genre, signal_type, format, structure,
    node, sn_ratio, entities,
    created_at, modified_at, valid_from, valid_until, supersedes,
    routed_to, metadata, workspace_id
    """
    |> String.replace("\n", " ")
    |> String.trim()
  end

  defp get_column_names(db, table) do
    case Exqlite.Sqlite3.prepare(db, "PRAGMA table_info(#{table})") do
      {:ok, stmt} ->
        :ok = Exqlite.Sqlite3.bind(stmt, [])

        names =
          Stream.repeatedly(fn -> Exqlite.Sqlite3.step(db, stmt) end)
          |> Enum.take_while(fn r -> r != :done end)
          |> Enum.map(fn {:row, [_cid, name | _]} -> name end)

        Exqlite.Sqlite3.release(db, stmt)
        names

      _ ->
        []
    end
  end

  defp cache_put(id, ctx) do
    if :ets.info(@ets_table, :size) >= @ets_max do
      case :ets.first(@ets_table) do
        :"$end_of_table" -> :ok
        old_key -> :ets.delete(@ets_table, old_key)
      end
    end

    :ets.insert(@ets_table, {id, ctx})
  end

  defp escape(str), do: String.replace(str, "'", "''")
end
