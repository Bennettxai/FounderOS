defmodule OptimalEngine.API.GraphPayload do
  @moduledoc """
  Builds graph API response payloads.

  The router owns HTTP concerns; this module owns the read-model queries and
  JSON-ready shapes for graph endpoints.
  """

  alias OptimalEngine.Graph.Analyzer, as: GraphAnalyzer
  alias OptimalEngine.Graph.Reflector
  alias OptimalEngine.Store

  @doc """
  Full graph payload: renderable contexts, visual edges, entity summary, node summary.

  Scope is workspace-bound to avoid cross-workspace leakage.
  """
  def full_graph(workspace_id \\ "default") do
    edges = fetch_visual_edges(workspace_id)
    entities = fetch_entity_summary(workspace_id)
    contexts = fetch_all_contexts(workspace_id)

    %{
      workspace_id: workspace_id,
      contexts: contexts,
      edges: edges,
      entities: format_entities(entities),
      nodes: fetch_node_summary(workspace_id),
      stats: %{
        context_count: length(contexts),
        edge_count: length(edges),
        entity_count: length(entities)
      }
    }
  end

  def hubs(workspace_id \\ "default") do
    {:ok, hubs} = GraphAnalyzer.hubs(workspace_id: workspace_id)
    %{hubs: format_hubs(hubs)}
  end

  def triangles(limit \\ 20, workspace_id \\ "default", opts \\ []) when is_list(opts) do
    {:ok, triangles} =
      GraphAnalyzer.triangles(Keyword.merge([limit: limit, workspace_id: workspace_id], opts))

    %{triangles: format_triangles(triangles)}
  end

  def clusters(workspace_id \\ "default") do
    {:ok, clusters} = GraphAnalyzer.clusters(workspace_id: workspace_id)
    %{clusters: format_clusters(clusters)}
  end

  def reflection_gaps(min_cooccurrences \\ 2, workspace_id \\ "default", opts \\ [])
      when is_list(opts) do
    skip_llm = Keyword.get(opts, :skip_llm, true)

    {:ok, gaps} =
      Reflector.reflect(
        min_cooccurrences: min_cooccurrences,
        workspace_id: workspace_id,
        skip_llm: skip_llm
      )

    %{gaps: format_gaps(gaps)}
  end

  def node(node_id, workspace_id \\ "default") do
    %{
      node: node_id,
      workspace_id: workspace_id,
      contexts: fetch_node_contexts(node_id, workspace_id),
      edges: fetch_node_edges(node_id, workspace_id) |> format_edges()
    }
  end

  # Every context in the DB; each becomes a renderable node in the graph.
  defp fetch_all_contexts(workspace_id) do
    case Store.raw_query(
           """
           SELECT id, title, node, type, genre, sn_ratio, modified_at, l0_abstract, uri
           FROM contexts
           WHERE workspace_id = ?1
           ORDER BY node, modified_at DESC
           """,
           [workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, title, node, type, genre, sn, mod, abstract, uri] ->
          %{
            id: id,
            title: title,
            node: node,
            type: type,
            genre: genre,
            sn_ratio: sn,
            modified_at: mod,
            l0_abstract: abstract,
            uri: uri
          }
        end)

      _ ->
        []
    end
  end

  defp fetch_visual_edges(workspace_id) do
    (fetch_shared_entity_edges(workspace_id) ++ fetch_cross_ref_edges(workspace_id))
    |> deduplicate_edges()
  end

  defp fetch_shared_entity_edges(workspace_id) do
    sql = """
    SELECT e1.context_id AS source, e2.context_id AS target, e1.name AS entity
    FROM entities e1
    JOIN entities e2 ON e1.name = e2.name AND e1.context_id < e2.context_id
    JOIN contexts c1 ON c1.id = e1.context_id
    JOIN contexts c2 ON c2.id = e2.context_id
    WHERE c1.workspace_id = ?1 AND c2.workspace_id = ?1
    LIMIT 2000
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        rows
        |> Enum.group_by(fn [source, target, _entity] -> {source, target} end)
        |> Enum.map(fn {{source, target}, matches} ->
          shared_count = length(matches)
          entity_names = matches |> Enum.map(fn [_, _, e] -> e end) |> Enum.uniq() |> Enum.take(3)
          weight = min(shared_count / 5.0, 1.0)

          %{
            source: source,
            target: target,
            relation: "shared_entity",
            weight: Float.round(weight, 2),
            entities: entity_names,
            shared_count: shared_count
          }
        end)

      _ ->
        []
    end
  end

  defp fetch_cross_ref_edges(workspace_id) do
    sql = """
    SELECT e.source_id AS source, c.id AS target
    FROM edges e
    JOIN contexts c ON c.node = e.target_id
    WHERE e.relation = 'cross_ref'
      AND e.workspace_id = ?1
      AND c.workspace_id = ?1
      AND EXISTS (SELECT 1 FROM contexts WHERE id = e.source_id AND workspace_id = ?1)
    LIMIT 500
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        Enum.map(rows, fn [source, target] ->
          %{source: source, target: target, relation: "cross_ref", weight: 1.0}
        end)

      _ ->
        []
    end
  end

  defp deduplicate_edges(edges) do
    edges
    |> Enum.uniq_by(fn %{source: s, target: t} ->
      if s < t, do: {s, t}, else: {t, s}
    end)
  end

  defp fetch_entity_summary(workspace_id) do
    case Store.raw_query(
           """
           SELECT e.name, e.type, COUNT(*) as count
           FROM entities e
           JOIN contexts c ON c.id = e.context_id
           WHERE c.workspace_id = ?1
           GROUP BY e.name, e.type
           ORDER BY count DESC
           LIMIT 200
           """,
           [workspace_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  defp fetch_node_summary(workspace_id) do
    case Store.raw_query(
           """
           SELECT node, COUNT(*) as count, MAX(modified_at) as last_modified
           FROM contexts
           WHERE workspace_id = ?1
           GROUP BY node
           ORDER BY node
           """,
           [workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [node, count, last_mod] ->
          %{node: node, context_count: count, last_modified: last_mod}
        end)

      _ ->
        []
    end
  end

  defp fetch_node_contexts(node_id, workspace_id) do
    case Store.raw_query(
           """
           SELECT id, title, type, genre, sn_ratio, modified_at
           FROM contexts
           WHERE node = ?1 AND workspace_id = ?2
           ORDER BY modified_at DESC
           LIMIT 50
           """,
           [node_id, workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, title, type, genre, sn, mod] ->
          %{id: id, title: title, type: type, genre: genre, sn_ratio: sn, modified_at: mod}
        end)

      _ ->
        []
    end
  end

  defp fetch_node_edges(node_id, workspace_id) do
    case Store.raw_query(
           """
           SELECT DISTINCT e.source_id, e.target_id, e.relation, e.weight
           FROM edges e
           JOIN contexts c ON (c.id = e.source_id OR c.id = e.target_id)
           WHERE c.node = ?1
           AND e.workspace_id = ?2
           LIMIT 200
           """,
           [node_id, workspace_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  defp format_edges(rows) when is_list(rows) do
    Enum.map(rows, fn
      [s, t, r, w] ->
        %{source: s, target: t, relation: r, weight: w}

      %{source_id: s, target_id: t, relation: r, weight: w} ->
        %{source: s, target: t, relation: r, weight: w}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_edges(_), do: []

  defp format_entities(rows) when is_list(rows) do
    Enum.map(rows, fn
      [name, type, count] -> %{name: name, type: type, count: count}
      other -> %{raw: inspect(other)}
    end)
  end

  defp format_entities(_), do: []

  defp format_hubs(hubs) when is_list(hubs) do
    Enum.map(hubs, fn
      %{id: id, degree: degree, sigma: sigma} ->
        %{entity: id, degree: degree, sigma: Float.round(sigma, 2)}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_hubs(_), do: []

  defp format_triangles(triangles) when is_list(triangles) do
    Enum.map(triangles, fn
      %{id: id, nodes: nodes, score: score} when is_list(nodes) ->
        %{id: id, nodes: nodes, score: Float.round(score, 2)}

      %{a: a, b: b, c: c, suggestion: suggestion} ->
        %{id: Enum.join([a, b, c], ":"), nodes: [a, b, c], score: 0.0, suggestion: suggestion}

      %{a: a, b: b, c: c} ->
        %{id: Enum.join([a, b, c], ":"), nodes: [a, b, c], score: 0.0, suggestion: nil}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_triangles(_), do: []

  defp format_clusters(clusters) when is_list(clusters) do
    Enum.map(clusters, fn
      %MapSet{} = members -> MapSet.to_list(members)
      members when is_list(members) -> Enum.map(members, &to_string/1)
      other -> [inspect(other)]
    end)
  end

  defp format_clusters(_), do: []

  defp format_gaps(gaps) when is_list(gaps) do
    Enum.map(gaps, fn
      %{entity_a: a, entity_b: b, confidence: c, relation: relation} ->
        %{entity_a: a, entity_b: b, confidence: Float.round(c, 2), relation: relation}

      %{source: a, target: b, confidence: c, suggested_relation: relation} ->
        %{entity_a: a, entity_b: b, confidence: Float.round(c, 2), relation: relation}

      %{source: a, target: b, confidence: c} ->
        %{entity_a: a, entity_b: b, confidence: Float.round(c, 2), relation: nil}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_gaps(_), do: []
end
