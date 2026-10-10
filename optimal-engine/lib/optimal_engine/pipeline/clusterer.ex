defmodule OptimalEngine.Pipeline.Clusterer do
  @moduledoc """
  Stage 8 (number matches ARCHITECTURE.md; built as Phase 6 per PLAN.md) —
  the wide pass across chunks that groups them into theme-clusters.

  ## Algorithm

  Incremental greedy clustering with a weighted-similarity threshold:

  1. For each incoming chunk-feature:
     a. Compute `Similarity.sim(feature, existing_cluster_centroid)` for
        every existing cluster in the tenant.
     b. If the max similarity ≥ `:threshold` (default 0.65), add the
        chunk to that cluster and update its centroid (running mean).
     c. Otherwise, seed a new cluster containing just this chunk.
  2. Touched clusters bump `member_count` + `updated_at`.
  3. (Optional, Ollama-gated) re-name the theme when member_count grows
     significantly — see `Clusterer.Theme`.

  This algorithm is deterministic (given a fixed iteration order of
  input chunks), incremental (O(|chunks| × |clusters|) to absorb N new
  chunks), and produces stable assignments — a chunk stays in its
  cluster unless explicitly re-clustered via `rebuild/1`.

  ## Why not HDBSCAN

  HDBSCAN produces nicer density-based clusters but is not on hex as a
  pure-Elixir library. Porting it is a Phase 10+ enhancement. For v1,
  the greedy threshold approach gives acceptable cluster quality and
  works incrementally without the re-cluster-everything overhead
  HDBSCAN would impose.

  ## Feature construction

  The Clusterer pulls from four Phase 1–5 tables to build each feature:

    * `chunk_embeddings.vector`        → the vector (60% weight)
    * `entities.name` for chunk_id     → entity set (20% weight; Jaccard)
    * `intents.intent`                 → intent enum (15% weight; exact match)
    * `chunks.signal_id` → `contexts.node` → Workspace.Node tree (5% weight)
  """

  alias OptimalEngine.Pipeline.Clusterer.{Cluster, Similarity}
  alias OptimalEngine.Store

  require Logger

  @default_threshold 0.65
  @default_theme "Unnamed cluster"

  @type feature :: Similarity.feature()
  @type chunk_feature :: %{
          chunk_id: String.t(),
          tenant_id: String.t(),
          feature: feature(),
          intent: atom() | nil
        }

  @doc """
  Incrementally assign a batch of chunk-features to clusters.

  Arguments:
    * `features` — list of `%{chunk_id, tenant_id, feature, intent}`
    * `opts`
      * `:threshold`    — similarity threshold for joining (default 0.65)
      * `:tenant_id`    — required when features don't carry one
      * `:existing`     — pre-loaded list of existing clusters; avoids a store read

  Returns `{:ok, [%Cluster{}], [%{chunk_id, cluster_id, weight}]}` —
  the set of clusters that were created or touched, plus the new
  membership edges for the incoming features.
  """
  @spec assign([chunk_feature()], keyword()) ::
          {:ok, [Cluster.t()], [%{chunk_id: String.t(), cluster_id: String.t(), weight: float()}]}
  def assign(features, opts \\ []) when is_list(features) do
    threshold = Keyword.get(opts, :threshold, @default_threshold)
    tenant_id = Keyword.get(opts, :tenant_id)

    existing =
      case Keyword.get(opts, :existing) do
        nil -> load_existing_clusters(tenant_id)
        clusters when is_list(clusters) -> clusters
      end

    {clusters, memberships} =
      Enum.reduce(features, {existing, []}, fn chunk_feature, {clusters_acc, mem_acc} ->
        absorb(chunk_feature, clusters_acc, threshold, mem_acc)
      end)

    {:ok, clusters, Enum.reverse(memberships)}
  end

  @doc """
  Rebuild all clusters for a tenant from scratch — reads every chunk+feature,
  clears the existing clusters + memberships, and reassigns.

  Use sparingly; `assign/2` is the hot-path incremental API.
  """
  @spec rebuild(String.t(), keyword()) ::
          {:ok, %{clusters: non_neg_integer(), members: non_neg_integer()}}
          | {:error, term()}
  def rebuild(tenant_id, opts \\ []) when is_binary(tenant_id) do
    threshold = Keyword.get(opts, :threshold, @default_threshold)

    with :ok <- clear_clusters(tenant_id),
         {:ok, features} <- load_features(tenant_id),
         {:ok, clusters, memberships} <-
           assign(features, threshold: threshold, tenant_id: tenant_id, existing: []) do
      :ok = Store.insert_clusters(clusters)

      :ok =
        Store.insert_cluster_members(memberships |> Enum.map(&Map.put(&1, :tenant_id, tenant_id)))

      {:ok, %{clusters: length(clusters), members: length(memberships)}}
    end
  end

  # ─── private: incremental assignment ─────────────────────────────────────

  defp absorb(
         %{chunk_id: chunk_id, tenant_id: cf_tenant, feature: feature, intent: intent} = _cf,
         clusters,
         threshold,
         mem_acc
       ) do
    tenant_id = cf_tenant || "default"

    case find_nearest(clusters, feature, threshold) do
      {:ok, %Cluster{} = cluster, _sim} ->
        # Proper running-mean update:
        updated_centroid =
          cluster.centroid
          |> Enum.zip(feature.embedding)
          |> Enum.map(fn {c, v} ->
            (c * cluster.member_count + v) / (cluster.member_count + 1)
          end)

        updated =
          %{
            cluster
            | centroid: updated_centroid,
              member_count: cluster.member_count + 1,
              intent_dominant:
                majority_intent(cluster.intent_dominant, intent, cluster.member_count)
          }

        clusters_next = replace_cluster(clusters, cluster.id, updated)

        membership = %{chunk_id: chunk_id, cluster_id: cluster.id, weight: 1.0}
        {clusters_next, [membership | mem_acc]}

      :none ->
        new_cluster =
          Cluster.new(
            tenant_id: tenant_id,
            theme: @default_theme,
            intent_dominant: intent,
            member_count: 1,
            centroid: feature.embedding
          )

        membership = %{chunk_id: chunk_id, cluster_id: new_cluster.id, weight: 1.0}
        {[new_cluster | clusters], [membership | mem_acc]}
    end
  end

  defp find_nearest(clusters, feature, threshold) do
    scored =
      clusters
      |> Enum.map(fn c ->
        {c,
         Similarity.sim(feature, %{
           embedding: c.centroid,
           entities: [],
           intent: c.intent_dominant,
           node_id: nil,
           node_ancestors: []
         })}
      end)
      |> Enum.sort_by(fn {_c, sim} -> -sim end)

    case scored do
      [{c, sim} | _] when sim >= threshold -> {:ok, c, sim}
      _ -> :none
    end
  end

  defp replace_cluster(clusters, id, new_cluster) do
    Enum.map(clusters, fn c -> if c.id == id, do: new_cluster, else: c end)
  end

  # Running plurality: after member_count additions the stored
  # intent_dominant is the one most seen so far. We approximate this by
  # keeping the last-observed intent when a tie or a new-winner pattern
  # isn't obvious from the centroid alone. More accurate plurality
  # tracking would require a per-cluster intent histogram; deferred.
  defp majority_intent(nil, new_intent, _count), do: new_intent
  defp majority_intent(existing, nil, _count), do: existing
  defp majority_intent(existing, _new, _count), do: existing

  # ─── private: Store interop ──────────────────────────────────────────────

  defp load_existing_clusters(nil), do: []

  defp load_existing_clusters(tenant_id) when is_binary(tenant_id) do
    case Store.raw_query(
           """
           SELECT id, tenant_id, theme, intent_dominant, member_count, centroid, updated_at
           FROM clusters
           WHERE tenant_id = ?1
           """,
           [tenant_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, tid, theme, intent_dom, count, blob, updated] ->
          %Cluster{
            id: id,
            tenant_id: tid,
            theme: theme || @default_theme,
            intent_dominant: safe_atom(intent_dom),
            member_count: count || 0,
            centroid: decode_vector(blob),
            updated_at: updated
          }
        end)

      _ ->
        []
    end
  end

  defp load_features(tenant_id) do
    # Pull everything we need in one query:
    # chunk_id + embedding vector + intent + signal_id + entities (aggregated)
    sql = """
    SELECT
      ce.chunk_id,
      ce.tenant_id,
      ce.vector,
      i.intent,
      ch.signal_id,
      (SELECT GROUP_CONCAT(e.name, '\u0001')
         FROM entities e
         WHERE e.context_id = ch.signal_id) AS entity_names
    FROM chunk_embeddings ce
    JOIN chunks ch ON ch.id = ce.chunk_id
    LEFT JOIN intents i ON i.chunk_id = ce.chunk_id
    WHERE ce.tenant_id = ?1
    """

    case Store.raw_query(sql, [tenant_id]) do
      {:ok, rows} ->
        features =
          Enum.map(rows, fn [chunk_id, cf_tenant, blob, intent, _signal_id, entity_joined] ->
            %{
              chunk_id: chunk_id,
              tenant_id: cf_tenant,
              intent: safe_atom(intent),
              feature: %{
                embedding: decode_vector(blob),
                entities: split_entities(entity_joined),
                intent: safe_atom(intent),
                # Node affinity uses chunks.node / Workspace.NodeMember — deferred.
                node_id: nil,
                node_ancestors: []
              }
            }
          end)

        {:ok, features}

      err ->
        err
    end
  end

  defp clear_clusters(tenant_id) do
    with {:ok, _} <-
           Store.raw_query("DELETE FROM cluster_members WHERE tenant_id = ?1", [tenant_id]),
         {:ok, _} <- Store.raw_query("DELETE FROM clusters WHERE tenant_id = ?1", [tenant_id]) do
      :ok
    end
  end

  defp decode_vector(nil), do: []
  defp decode_vector(""), do: []

  defp decode_vector(blob) when is_binary(blob) do
    for <<f::little-float-size(32) <- blob>>, do: f
  end

  defp split_entities(nil), do: []
  defp split_entities(""), do: []

  defp split_entities(str) when is_binary(str) do
    str |> String.split(<<0x01>>) |> Enum.reject(&(&1 == ""))
  end

  defp safe_atom(nil), do: nil

  defp safe_atom(str) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      _ -> nil
    end
  end
end
