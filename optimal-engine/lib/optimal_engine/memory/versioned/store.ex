defmodule OptimalEngine.Memory.Versioned.Store do
  @moduledoc """
  Raw SQL operations for the versioned memory primitive.

  This module is internal — only `OptimalEngine.Memory.Versioned` should
  call it. All SQL lives here; the facade owns business logic.

  Column layout mirrors the `memories` table created in migration 028.
  Boolean fields (`is_static`, `is_forgotten`, `is_latest`) are stored as
  SQLite INTEGER (0/1) and deserialized to Elixir booleans here.
  """

  alias OptimalEngine.Store

  @select_cols """
  id, tenant_id, workspace_id, content, is_static, is_forgotten,
  forget_after, forget_reason, version, parent_memory_id, root_memory_id,
  is_latest, citation_uri, source_chunk_id, audience, metadata,
  created_at, updated_at
  """

  # Columns used in the dedup lookup — must match @select_cols order exactly.
  @dedup_select_cols @select_cols

  # ---------------------------------------------------------------------------
  # Write operations
  # ---------------------------------------------------------------------------

  @doc "Inserts a new memory row. Returns the raw row list on success."
  @spec insert(map()) :: {:ok, list()} | {:error, term()}
  def insert(attrs) do
    sql = """
    INSERT INTO memories (
      id, tenant_id, workspace_id, content, is_static, is_forgotten,
      version, parent_memory_id, root_memory_id, is_latest,
      citation_uri, source_chunk_id, audience, metadata, content_hash
    ) VALUES (?1, ?2, ?3, ?4, ?5, 0, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14)
    """

    params = [
      attrs.id,
      attrs.tenant_id,
      attrs.workspace_id,
      attrs.content,
      bool_to_int(attrs[:is_static] || false),
      attrs[:version] || 1,
      attrs[:parent_memory_id],
      attrs[:root_memory_id],
      bool_to_int(attrs[:is_latest] != false),
      attrs[:citation_uri],
      attrs[:source_chunk_id],
      attrs[:audience] || "default",
      Jason.encode!(attrs[:metadata] || %{}),
      attrs[:content_hash]
    ]

    Store.raw_query(sql, params)
  end

  @doc """
  Looks up an existing live memory by workspace_id, audience, and content_hash.

  "Live" means `is_forgotten = 0` AND `is_latest = 1`. Returns `{:ok, row}`
  when found, `{:error, :not_found}` when absent.
  """
  @spec find_by_content_hash(String.t(), String.t(), String.t()) ::
          {:ok, list()} | {:error, :not_found}
  def find_by_content_hash(workspace_id, audience, content_hash)
      when is_binary(workspace_id) and is_binary(audience) and is_binary(content_hash) do
    sql = """
    SELECT #{@dedup_select_cols}
    FROM memories
    WHERE workspace_id = ?1
      AND audience = ?2
      AND content_hash = ?3
      AND is_forgotten = 0
      AND is_latest = 1
    LIMIT 1
    """

    case Store.raw_query(sql, [workspace_id, audience, content_hash]) do
      {:ok, [row]} -> {:ok, row}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Fetches a single memory row by id."
  @spec get(String.t()) :: {:ok, list()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    sql = "SELECT #{@select_cols} FROM memories WHERE id = ?1"

    case Store.raw_query(sql, [id]) do
      {:ok, [row]} -> {:ok, row}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Lists memories filtered by opts:
    - `:workspace_id` (required) — defaults to "default"
    - `:audience` — filter by audience string
    - `:include_forgotten` — default false
    - `:include_old_versions` — default false (only is_latest=1)
    - `:limit` — default 50
    - `:offset` — default 0 (for pagination)
    - `:q` — full-text search query; when present, results are ordered by
              BM25 rank (best match first) instead of `created_at DESC`.
              Uses the `memories_fts` FTS5 virtual table created in
              migration 031. An empty string or nil is treated as absent
              (no FTS filter applied).
  """
  @spec list(keyword()) :: {:ok, [list()]} | {:error, term()}
  def list(opts) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    audience = Keyword.get(opts, :audience)
    include_forgotten = Keyword.get(opts, :include_forgotten, false)
    include_old = Keyword.get(opts, :include_old_versions, false)
    limit = Keyword.get(opts, :limit, 50)
    offset = Keyword.get(opts, :offset, 0)
    q = opts |> Keyword.get(:q) |> normalize_fts_query()

    if q do
      list_with_fts(workspace_id, audience, include_forgotten, include_old, limit, offset, q)
    else
      list_without_fts(workspace_id, audience, include_forgotten, include_old, limit, offset)
    end
  end

  @doc """
  Counts total memories matching the same filter opts as `list/1`.
  Accepts `:workspace_id`, `:audience`, `:include_forgotten`, `:include_old_versions`.
  Returns `{:ok, non_neg_integer()}`.
  """
  @spec count(keyword()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(opts) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    audience = Keyword.get(opts, :audience)
    include_forgotten = Keyword.get(opts, :include_forgotten, false)
    include_old = Keyword.get(opts, :include_old_versions, false)

    {conditions, params} =
      build_filter_conditions(workspace_id, audience, include_forgotten, include_old, "m")

    where = Enum.join(conditions, " AND ")

    sql = "SELECT COUNT(*) FROM memories m WHERE #{where}"

    case Store.raw_query(sql, params) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, []} -> {:ok, 0}
      other -> other
    end
  end

  # ---------------------------------------------------------------------------
  # Private — list helpers
  # ---------------------------------------------------------------------------

  # Plain list path (no FTS). Builds a dynamic WHERE using positional params.
  defp list_without_fts(workspace_id, audience, include_forgotten, include_old, limit, offset) do
    {conditions, params} =
      build_filter_conditions(workspace_id, audience, include_forgotten, include_old, "m")

    param_idx = length(params) + 1
    where = Enum.join(conditions, " AND ")

    sql = """
    SELECT #{prefixed_select_cols("m")} FROM memories m
    WHERE #{where}
    ORDER BY m.created_at DESC
    LIMIT ?#{param_idx} OFFSET ?#{param_idx + 1}
    """

    Store.raw_query(sql, params ++ [limit, offset])
  end

  # FTS path: JOIN memories against memories_fts on rowid, filter with MATCH,
  # apply the same workspace/audience/forgotten/latest constraints, and order
  # by BM25 rank (negated — lower raw value = better match in SQLite).
  defp list_with_fts(
         workspace_id,
         audience,
         include_forgotten,
         include_old,
         limit,
         offset,
         fts_query
       ) do
    # ?1 = fts_query; build the rest of the conditions starting at ?2.
    {conditions, base_params} =
      build_filter_conditions(workspace_id, audience, include_forgotten, include_old, "m", 2)

    params = [fts_query | base_params]
    param_idx = length(params) + 1
    where = Enum.join(conditions, " AND ")

    sql = """
    SELECT #{prefixed_select_cols("m")} FROM memories_fts
    JOIN memories m ON m.rowid = memories_fts.rowid
    WHERE memories_fts MATCH ?1
      AND #{where}
    ORDER BY bm25(memories_fts) ASC
    LIMIT ?#{param_idx} OFFSET ?#{param_idx + 1}
    """

    Store.raw_query(sql, params ++ [limit, offset])
  end

  # Builds {conditions_list, params_list} for the WHERE clause.
  # `table_alias` is the SQL alias prefix (e.g. "m").
  # `start_idx` is the first positional param index (default 1 for non-FTS,
  # 2 for FTS where ?1 is the FTS query term).
  defp build_filter_conditions(
         workspace_id,
         audience,
         include_forgotten,
         include_old,
         table_alias,
         start_idx \\ 1
       ) do
    conditions = ["#{table_alias}.workspace_id = ?#{start_idx}"]
    params = [workspace_id]
    param_idx = start_idx + 1

    {conditions, params, param_idx} =
      if audience do
        {conditions ++ ["#{table_alias}.audience = ?#{param_idx}"], params ++ [audience],
         param_idx + 1}
      else
        {conditions, params, param_idx}
      end

    _ = param_idx

    conditions =
      if not include_forgotten,
        do: conditions ++ ["#{table_alias}.is_forgotten = 0"],
        else: conditions

    conditions =
      if not include_old,
        do: conditions ++ ["#{table_alias}.is_latest = 1"],
        else: conditions

    {conditions, params}
  end

  # Returns a comma-separated column list with the given table alias prefix.
  # Must match the column order of @select_cols exactly so row_to_struct/1
  # in Versioned continues to work without modification.
  defp prefixed_select_cols(alias) do
    Enum.map_join(
      ~w[id tenant_id workspace_id content is_static is_forgotten
         forget_after forget_reason version parent_memory_id root_memory_id
         is_latest citation_uri source_chunk_id audience metadata
         created_at updated_at],
      ", ",
      fn col -> "#{alias}.#{col}" end
    )
  end

  # Normalises the caller-supplied FTS query.
  # Returns nil (no FTS) for blank/nil; otherwise escapes SQLite FTS5 special
  # characters and wraps bare terms so the query is MATCH-safe.
  defp normalize_fts_query(nil), do: nil
  defp normalize_fts_query(""), do: nil

  defp normalize_fts_query(q) when is_binary(q) do
    trimmed = String.trim(q)

    if trimmed == "" do
      nil
    else
      # Escape double-quotes (used for phrase queries) by doubling them,
      # then strip FTS5 operator chars that could cause parse errors.
      trimmed
      |> String.replace("\"", "\"\"")
      |> String.replace(~r/[*^()]/u, " ")
      |> String.trim()
    end
  end

  @doc """
  Marks an old version as no longer latest. Called during update versioning.
  """
  @spec demote_latest(String.t()) :: {:ok, term()} | {:error, term()}
  def demote_latest(id) when is_binary(id) do
    Store.raw_query(
      "UPDATE memories SET is_latest = 0, updated_at = datetime('now') WHERE id = ?1",
      [id]
    )
  end

  @doc "Marks a memory as forgotten (soft delete)."
  @spec mark_forgotten(String.t(), String.t() | nil, String.t() | nil) ::
          {:ok, term()} | {:error, term()}
  def mark_forgotten(id, reason, forget_after) when is_binary(id) do
    Store.raw_query(
      """
      UPDATE memories
      SET is_forgotten = 1, forget_reason = ?2, forget_after = ?3,
          updated_at = datetime('now')
      WHERE id = ?1
      """,
      [id, reason, forget_after]
    )
  end

  @doc "Hard deletes a memory row (cascades to memory_relations)."
  @spec delete(String.t()) :: {:ok, term()} | {:error, term()}
  def delete(id) when is_binary(id) do
    Store.raw_query("DELETE FROM memories WHERE id = ?1", [id])
  end

  # ---------------------------------------------------------------------------
  # Relation operations
  # ---------------------------------------------------------------------------

  @doc "Inserts a typed relation between two memories."
  @spec add_relation(String.t(), String.t(), String.t(), String.t(), String.t()) ::
          {:ok, term()} | {:error, term()}
  def add_relation(source_id, target_id, relation, workspace_id, tenant_id) do
    sql = """
    INSERT OR IGNORE INTO memory_relations
      (tenant_id, workspace_id, source_memory_id, target_memory_id, relation)
    VALUES (?1, ?2, ?3, ?4, ?5)
    """

    Store.raw_query(sql, [tenant_id, workspace_id, source_id, target_id, relation])
  end

  @doc """
  Returns all relations touching `memory_id` — both outbound (source) and
  inbound (target). Each row: [id, source_memory_id, target_memory_id, relation, created_at].
  """
  @spec get_relations(String.t()) :: {:ok, [list()]} | {:error, term()}
  def get_relations(memory_id) when is_binary(memory_id) do
    sql = """
    SELECT id, source_memory_id, target_memory_id, relation, created_at
    FROM memory_relations
    WHERE source_memory_id = ?1 OR target_memory_id = ?1
    ORDER BY created_at ASC
    """

    Store.raw_query(sql, [memory_id])
  end

  @doc """
  Returns all memories in the version chain rooted at `root_id`, ordered
  by version ascending.
  """
  @spec get_version_chain(String.t(), String.t()) :: {:ok, [list()]} | {:error, term()}
  def get_version_chain(root_id, workspace_id) when is_binary(root_id) do
    sql = """
    SELECT #{@select_cols} FROM memories
    WHERE workspace_id = ?1 AND (root_memory_id = ?2 OR id = ?2)
    ORDER BY version ASC
    """

    Store.raw_query(sql, [workspace_id, root_id])
  end

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp bool_to_int(true), do: 1
  defp bool_to_int(false), do: 0
  defp bool_to_int(1), do: 1
  defp bool_to_int(0), do: 0
end
