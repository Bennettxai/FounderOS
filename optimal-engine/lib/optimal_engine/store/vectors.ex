defmodule OptimalEngine.Store.Vectors do
  @moduledoc """
  Stateless module for storing and searching embeddings in SQLite.

  Embeddings are persisted as little-endian 32-bit float BLOBs alongside a
  `context_id` foreign key.  All database access goes through
  `OptimalEngine.Store.raw_query/2` so the GenServer connection is reused.

  ## Encoding

  768 floats × 4 bytes = 3 072 bytes per embedding (little-endian IEEE 754).

  ## Search

  Similarity search is performed in-process: all embeddings are loaded from
  SQLite and cosine similarity is computed against the query vector.  This is
  acceptable for a personal knowledge store (hundreds of thousands of entries)
  and avoids a native sqlite-vec extension dependency.  If the store grows
  beyond ~50 K vectors, revisit with an approximate-nearest-neighbour index.
  """

  require Logger

  alias OptimalEngine.Store

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Stores an embedding for a context.

  Uses INSERT OR REPLACE, so calling `store/2` again for the same `context_id`
  overwrites the previous embedding.

  Returns `:ok` or `{:error, reason}`.
  """
  @spec store(String.t(), [float()]) :: :ok | {:error, term()}
  def store(context_id, embedding) when is_binary(context_id) and is_list(embedding) do
    blob = encode_embedding(embedding)
    dims = length(embedding)

    sql = """
    INSERT OR REPLACE INTO vectors (context_id, embedding, dimensions)
    VALUES (?1, ?2, ?3)
    """

    # `{:blob, _}` forces exqlite to bind as BLOB. Raw binary otherwise
    # binds as TEXT and gets truncated at the first null byte — which is
    # catastrophic for float32 data.
    case Store.raw_query(sql, [context_id, {:blob, blob}, dims]) do
      {:ok, _} ->
        Logger.debug("[VectorStore] Stored embedding for #{context_id} (#{dims} dims)")
        :ok

      {:error, reason} = err ->
        Logger.warning(
          "[VectorStore] Failed to store embedding for #{context_id}: #{inspect(reason)}"
        )

        err
    end
  end

  @doc """
  Searches stored embeddings by cosine similarity.

  Options:
  - `:limit`          — maximum results to return (default 10)
  - `:min_similarity` — minimum cosine similarity threshold (default 0.0)
  - `:workspace_id`   — workspace scope (default `"default"`)
  - `:type_filter`    — filter by context type (string, e.g. `"signal"`)
  - `:node_filter`    — filter by node (string, e.g. `"project-platform-launch"`)

  Returns `{:ok, [{context_id, similarity_score}]}` sorted descending by score.
  """
  @spec search([float()], keyword()) :: {:ok, [{String.t(), float()}]} | {:error, term()}
  def search(query_embedding, opts \\ []) when is_list(query_embedding) do
    limit = Keyword.get(opts, :limit, 10)
    min_sim = Keyword.get(opts, :min_similarity, 0.0)
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    type_filter = Keyword.get(opts, :type_filter)
    node_filter = Keyword.get(opts, :node_filter)

    {sql, params} = build_search_query(workspace_id, type_filter, node_filter)

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        results =
          rows
          |> Enum.map(fn [cid, blob] ->
            stored = decode_embedding(blob)
            sim = cosine_similarity(query_embedding, stored)
            {cid, sim}
          end)
          |> Enum.filter(fn {_cid, sim} -> sim >= min_sim end)
          |> Enum.sort_by(fn {_cid, sim} -> sim end, :desc)
          |> Enum.take(limit)

        {:ok, results}

      {:error, reason} = err ->
        Logger.warning("[VectorStore] Search query failed: #{inspect(reason)}")
        err
    end
  end

  @doc """
  Reranks candidate contexts using their persisted chunk embeddings.

  This keeps exact local search bounded to candidates selected by FTS while
  making the richer per-chunk embedding index participate in retrieval.
  Scores use the best matching chunk for each source context.
  """
  @spec rerank_contexts([float()], [String.t()], keyword()) ::
          {:ok, [{String.t(), float()}]} | {:error, term()}
  def rerank_contexts(query_embedding, context_ids, opts \\ [])

  def rerank_contexts(_query_embedding, [], _opts), do: {:ok, []}

  def rerank_contexts(query_embedding, context_ids, opts)
      when is_list(query_embedding) and is_list(context_ids) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    limit = Keyword.get(opts, :limit, length(context_ids))
    placeholders = Enum.map_join(1..length(context_ids), ", ", &"?#{&1 + 2}")

    sql = """
    SELECT c.signal_id, ce.vector
    FROM chunk_embeddings ce
    JOIN chunks c ON c.id = ce.chunk_id
    WHERE ce.workspace_id = ?1
      AND ce.dim = ?2
      AND c.signal_id IN (#{placeholders})
    """

    params = [workspace_id, length(query_embedding) | context_ids]

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        results =
          rows
          |> Enum.reduce(%{}, fn [context_id, blob], scores ->
            similarity = cosine_similarity(query_embedding, decode_embedding(blob))
            Map.update(scores, context_id, similarity, &max(&1, similarity))
          end)
          |> Enum.sort_by(fn {_context_id, similarity} -> similarity end, :desc)
          |> Enum.take(limit)

        {:ok, results}

      {:error, reason} = error ->
        Logger.warning("[VectorStore] Chunk rerank failed: #{inspect(reason)}")
        error
    end
  end

  @doc """
  Deletes the embedding for a given context_id.

  Returns `:ok` (idempotent — no error if the row does not exist).
  """
  @spec delete(String.t()) :: :ok | {:error, term()}
  def delete(context_id) when is_binary(context_id) do
    sql = "DELETE FROM vectors WHERE context_id = ?1"

    case Store.raw_query(sql, [context_id]) do
      {:ok, _} ->
        Logger.debug("[VectorStore] Deleted embedding for #{context_id}")
        :ok

      {:error, reason} = err ->
        Logger.warning(
          "[VectorStore] Failed to delete embedding for #{context_id}: #{inspect(reason)}"
        )

        err
    end
  end

  @doc """
  Returns the total number of stored embeddings.

  Returns `{:ok, integer}` or `{:error, reason}`.
  """
  @spec count() :: {:ok, non_neg_integer()} | {:error, term()}
  def count do
    case Store.raw_query("SELECT COUNT(*) FROM vectors", []) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, _} -> {:ok, 0}
      {:error, _} = err -> err
    end
  end

  @doc """
  Retrieves the embedding for a given context_id as a list of floats.

  Returns `{:ok, [float()]}` or `{:error, :not_found}`.
  """
  @spec get(String.t()) :: {:ok, [float()]} | {:error, :not_found}
  def get(context_id) when is_binary(context_id) do
    sql = "SELECT embedding FROM vectors WHERE context_id = ?1"

    case Store.raw_query(sql, [context_id]) do
      {:ok, [[blob]]} -> {:ok, decode_embedding(blob)}
      {:ok, []} -> {:error, :not_found}
      {:ok, _} -> {:error, :not_found}
      {:error, _} -> {:error, :not_found}
    end
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp build_search_query(workspace_id, nil, nil) do
    sql = """
    SELECT v.context_id, v.embedding
    FROM vectors v
    JOIN contexts c ON c.id = v.context_id
    WHERE c.workspace_id = ?1
    """

    {sql, [workspace_id]}
  end

  defp build_search_query(workspace_id, type_filter, nil) when is_binary(type_filter) do
    sql = """
    SELECT v.context_id, v.embedding
    FROM vectors v
    JOIN contexts c ON c.id = v.context_id
    WHERE c.workspace_id = ?1 AND c.type = ?2
    """

    {sql, [workspace_id, type_filter]}
  end

  defp build_search_query(workspace_id, nil, node_filter) when is_binary(node_filter) do
    sql = """
    SELECT v.context_id, v.embedding
    FROM vectors v
    JOIN contexts c ON c.id = v.context_id
    WHERE c.workspace_id = ?1 AND c.node = ?2
    """

    {sql, [workspace_id, node_filter]}
  end

  defp build_search_query(workspace_id, type_filter, node_filter)
       when is_binary(type_filter) and is_binary(node_filter) do
    sql = """
    SELECT v.context_id, v.embedding
    FROM vectors v
    JOIN contexts c ON c.id = v.context_id
    WHERE c.workspace_id = ?1 AND c.type = ?2 AND c.node = ?3
    """

    {sql, [workspace_id, type_filter, node_filter]}
  end

  @spec encode_embedding([float()]) :: binary()
  defp encode_embedding(floats) do
    for f <- floats, into: <<>>, do: <<f::float-little-32>>
  end

  @spec decode_embedding(binary()) :: [float()]
  defp decode_embedding(binary) do
    for <<f::float-little-32 <- binary>>, do: f
  end

  @spec cosine_similarity([float()], [float()]) :: float()
  defp cosine_similarity(a, b) do
    {dot, sum_a2, sum_b2} =
      Enum.zip_reduce(a, b, {0.0, 0.0, 0.0}, fn ai, bi, {d, sa, sb} ->
        {d + ai * bi, sa + ai * ai, sb + bi * bi}
      end)

    norm_a = :math.sqrt(sum_a2)
    norm_b = :math.sqrt(sum_b2)

    if norm_a == 0.0 or norm_b == 0.0 do
      0.0
    else
      dot / (norm_a * norm_b)
    end
  end
end
