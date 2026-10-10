defmodule OptimalEngine.Graph.Analyzer do
  @moduledoc """
  Stateless graph analysis over the knowledge graph edge table.

  Three analyses:
  - `triangles/1`  — Find A→B, A→C pairs missing the B→C edge (synthesis opportunities)
  - `clusters/0`   — BFS connected components on the undirected edge graph
  - `hubs/0`       — Entities with degree > 2σ above the mean

  All functions return `{:ok, result}` and never raise. LLM enhancement via
  Ollama is applied when available; the module degrades gracefully without it.

  All analyses are workspace-scoped: pass `:workspace_id` (default "default").
  No cross-workspace leakage by default — see `OptimalEngine.Workspace`.
  """

  require Logger
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Detects open triangles: pairs of edges A→B and A→C where B→C does not exist.

  These represent synthesis opportunities — B and C share a common parent A but
  have not been directly linked. Each result includes an optional LLM-generated
  suggestion when Ollama is available.

  Returns `{:ok, [%{a: id, b: id, c: id, suggestion: String.t() | nil}]}`.
  """
  @spec triangles(keyword()) :: {:ok, [map()]}
  def triangles(opts \\ []) do
    limit = Keyword.get(opts, :limit, 20)
    ws = Keyword.get(opts, :workspace_id, "default")
    skip_llm = Keyword.get(opts, :skip_llm, false)

    sql = """
    SELECT DISTINCT e1.source_id, e1.target_id, e2.target_id
    FROM edges e1
    JOIN edges e2
      ON e1.source_id = e2.source_id
     AND e1.target_id != e2.target_id
     AND e2.workspace_id = ?2
    WHERE e1.workspace_id = ?2
      AND NOT EXISTS (
        SELECT 1 FROM edges e3
        WHERE e3.source_id = e1.target_id
          AND e3.target_id = e2.target_id
          AND e3.workspace_id = ?2
      )
    LIMIT ?1
    """

    case Store.raw_query(sql, [limit, ws]) do
      {:ok, rows} ->
        results = Enum.map(rows, fn [a, b, c] -> build_triangle(a, b, c, skip_llm) end)
        {:ok, results}

      {:error, reason} ->
        Logger.warning("GraphAnalyzer.triangles/1 query failed: #{inspect(reason)}")
        {:ok, []}
    end
  rescue
    err ->
      Logger.warning("GraphAnalyzer.triangles/1 failed: #{inspect(err)}")
      {:ok, []}
  end

  @doc """
  Finds connected components in the undirected knowledge graph via BFS.

  Loads all edges, builds an undirected adjacency map, then walks unvisited
  nodes to identify isolated clusters. Returns components sorted by size,
  largest first.

  Returns `{:ok, [MapSet.t()]}`.
  """
  @spec clusters(keyword()) :: {:ok, [MapSet.t()]}
  def clusters(opts \\ []) do
    ws = Keyword.get(opts, :workspace_id, "default")
    sql = "SELECT DISTINCT source_id, target_id FROM edges WHERE workspace_id = ?1"

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        adjacency = build_adjacency(rows)
        components = find_components(adjacency)
        sorted = Enum.sort_by(components, &MapSet.size/1, :desc)
        {:ok, sorted}

      {:error, reason} ->
        Logger.warning("GraphAnalyzer.clusters/0 query failed: #{inspect(reason)}")
        {:ok, []}
    end
  rescue
    err ->
      Logger.warning("GraphAnalyzer.clusters/0 failed: #{inspect(err)}")
      {:ok, []}
  end

  @doc """
  Identifies hub nodes — entities whose degree is more than 2 standard
  deviations above the mean.

  Degree is the combined in + out edge count. Returns results sorted by
  degree descending, each with the sigma distance from mean.

  Returns `{:ok, [%{id: String.t(), degree: non_neg_integer(), sigma: float()}]}`.
  """
  @spec hubs(keyword()) :: {:ok, [map()]}
  def hubs(opts \\ []) do
    ws = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, COUNT(*) as degree FROM (
      SELECT source_id as id FROM edges WHERE workspace_id = ?1
      UNION ALL
      SELECT target_id as id FROM edges WHERE workspace_id = ?1
    ) GROUP BY id
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, []} ->
        {:ok, []}

      {:ok, rows} ->
        degrees = Enum.map(rows, fn [_id, deg] -> deg end)
        mean = mean(degrees)
        stddev = stddev(degrees, mean)

        threshold = mean + 2 * stddev

        results =
          rows
          |> Enum.filter(fn [_id, deg] -> deg > threshold end)
          |> Enum.map(fn [id, deg] ->
            sigma = if stddev > 0, do: Float.round((deg - mean) / stddev, 2), else: 0.0
            %{id: id, degree: deg, sigma: sigma}
          end)
          |> Enum.sort_by(& &1.degree, :desc)

        {:ok, results}

      {:error, reason} ->
        Logger.warning("GraphAnalyzer.hubs/0 query failed: #{inspect(reason)}")
        {:ok, []}
    end
  rescue
    err ->
      Logger.warning("GraphAnalyzer.hubs/0 failed: #{inspect(err)}")
      {:ok, []}
  end

  # ---------------------------------------------------------------------------
  # Triangle helpers
  # ---------------------------------------------------------------------------

  defp build_triangle(a, b, c, skip_llm) do
    base = %{a: a, b: b, c: c, suggestion: nil}

    if !skip_llm and Ollama.available?() do
      Map.put(base, :suggestion, llm_suggestion(a, b, c))
    else
      base
    end
  end

  defp llm_suggestion(a, b, c) do
    prompt = """
    In a knowledge graph, "#{a}" links to both "#{b}" and "#{c}", but "#{b}" and "#{c}" are not directly connected.

    In one sentence, describe the most useful relationship or synthesis that could be drawn between "#{b}" and "#{c}" given their shared connection to "#{a}".
    """

    system = "You are a knowledge graph analyst. Be concise and specific. One sentence only."

    case Ollama.generate(prompt, system: system) do
      {:ok, text} -> String.trim(text)
      {:error, _} -> nil
    end
  rescue
    _ -> nil
  end

  # ---------------------------------------------------------------------------
  # BFS cluster helpers
  # ---------------------------------------------------------------------------

  defp build_adjacency(rows) do
    Enum.reduce(rows, %{}, fn [src, tgt], acc ->
      acc
      |> Map.update(src, MapSet.new([tgt]), &MapSet.put(&1, tgt))
      |> Map.update(tgt, MapSet.new([src]), &MapSet.put(&1, src))
    end)
  end

  defp find_components(adjacency) do
    all_nodes = MapSet.new(Map.keys(adjacency))
    do_bfs(adjacency, all_nodes, [])
  end

  # NOTE: a previous first clause guarded on `map_size(%{}) == 0` — a
  # constant-true guard that swallowed every call and recursed on itself,
  # looping forever for any non-empty graph. Removed; the empty-set check
  # below is the real termination condition.
  defp do_bfs(adjacency, unvisited, components) do
    if MapSet.size(unvisited) == 0 do
      components
    else
      start = unvisited |> MapSet.to_list() |> hd()
      component = bfs_visit(adjacency, [start], MapSet.new([start]))
      remaining = MapSet.difference(unvisited, component)
      do_bfs(adjacency, remaining, [component | components])
    end
  end

  defp bfs_visit(_adjacency, [], visited), do: visited

  defp bfs_visit(adjacency, [node | queue], visited) do
    neighbors =
      adjacency
      |> Map.get(node, MapSet.new())
      |> MapSet.difference(visited)
      |> MapSet.to_list()

    new_visited = Enum.reduce(neighbors, visited, &MapSet.put(&2, &1))
    bfs_visit(adjacency, queue ++ neighbors, new_visited)
  end

  # ---------------------------------------------------------------------------
  # Statistics helpers
  # ---------------------------------------------------------------------------

  defp mean([]), do: 0.0

  defp mean(values) do
    Enum.sum(values) / length(values)
  end

  defp stddev([], _mean), do: 0.0

  defp stddev(values, mean) do
    variance =
      values
      |> Enum.map(fn v -> :math.pow(v - mean, 2) end)
      |> Enum.sum()
      |> Kernel./(length(values))

    :math.sqrt(variance)
  end
end
