defmodule Mix.Tasks.Optimal.Graph do
  @shortdoc "Show knowledge graph stats and analysis"
  @moduledoc """
  Displays statistics and analysis from the Optimal Engine graph projection.

  Usage:
      mix optimal.graph                # Show graph stats and sample edges
      mix optimal.graph triangles      # Detect synthesis opportunities (open triangles)
      mix optimal.graph triangles --limit 10  # Limit results
      mix optimal.graph clusters       # Find isolated connected components
      mix optimal.graph hubs           # Identify high-degree hub nodes

  Subcommands:
    triangles   A→B and A→C edges where B→C is missing. Shows synthesis opportunities.
                Includes LLM-generated suggestions when Ollama is running.
    clusters    BFS connected components. Shows which nodes are isolated from each other.
    hubs        Entities with degree > 2σ above mean. These are the most connected nodes.
  """

  use Mix.Task

  @impl Mix.Task
  def run([]) do
    Mix.Task.run("app.start")

    IO.puts("\nKnowledge Graph\n")

    with {:ok, stats} <- OptimalEngine.Store.stats(),
         {:ok, graph_stats} <- OptimalEngine.Graph.stats(),
         {:ok, top_entities} <- OptimalEngine.Graph.top_entities(10),
         {:ok, samples} <- OptimalEngine.Graph.sample_edges(8) do
      print_summary(stats, graph_stats)
      print_top_entities(top_entities)
      print_samples(samples)
    else
      {:error, reason} ->
        IO.puts("Graph stats failed: #{inspect(reason)}")
    end
  end

  def run(["triangles" | rest]) do
    Mix.Task.run("app.start")
    limit = parse_limit(rest, 20)

    IO.puts("\nOpen Triangles (synthesis opportunities)\n")

    case OptimalEngine.Graph.Analyzer.triangles(limit: limit) do
      {:ok, []} ->
        IO.puts("  No open triangles found.")

      {:ok, triangles} ->
        print_triangles(triangles)
    end
  end

  def run(["clusters"]) do
    Mix.Task.run("app.start")

    IO.puts("\nConnected Components (isolated clusters)\n")

    case OptimalEngine.Graph.Analyzer.clusters() do
      {:ok, []} ->
        IO.puts("  No edges found in the graph.")

      {:ok, clusters} ->
        print_clusters(clusters)
    end
  end

  def run(["hubs"]) do
    Mix.Task.run("app.start")

    IO.puts("\nHub Nodes (degree > 2σ above mean)\n")

    case OptimalEngine.Graph.Analyzer.hubs() do
      {:ok, []} ->
        IO.puts("  No hubs found (graph may be too sparse).")

      {:ok, hubs} ->
        print_hubs(hubs)
    end
  end

  def run(args) do
    IO.puts("Unknown subcommand: #{Enum.join(args, " ")}")
    IO.puts("Usage: mix optimal.graph [triangles|clusters|hubs]")
  end

  # ---------------------------------------------------------------------------
  # Stats printers (original behavior)
  # ---------------------------------------------------------------------------

  defp print_summary(stats, graph_stats) do
    IO.puts("  Entities: #{stats["total_entities"]}")
    IO.puts("  Edges:    #{graph_stats["total_edges"]}")
    IO.puts("")
    IO.puts("  Edge types:")
    IO.puts("    mentioned_in:  #{graph_stats["mentioned_in"]}")
    IO.puts("    lives_in:      #{graph_stats["lives_in"]}")
    IO.puts("    works_on:      #{graph_stats["works_on"]}")
    IO.puts("    cross_ref:     #{graph_stats["cross_ref"]}")
    IO.puts("    supersedes:    #{graph_stats["supersedes"]}")
  end

  defp print_top_entities([]), do: :ok

  defp print_top_entities(entities) do
    IO.puts("")
    IO.puts("  Top connected entities:")

    Enum.each(entities, fn {name, count} ->
      IO.puts("    #{String.pad_trailing(name, 25)} #{count} connections")
    end)
  end

  defp print_samples([]), do: :ok

  defp print_samples(samples) do
    IO.puts("")
    IO.puts("  Sample edges:")

    Enum.each(samples, fn %{source_id: src, relation: rel, target_id: tgt} ->
      IO.puts("    #{truncate_id(src)} --#{rel}--> #{truncate_id(tgt)}")
    end)
  end

  # ---------------------------------------------------------------------------
  # Triangle printer
  # ---------------------------------------------------------------------------

  defp print_triangles(triangles) do
    IO.puts("  Found #{length(triangles)} open triangle(s):\n")

    triangles
    |> Enum.with_index(1)
    |> Enum.each(fn {%{a: a, b: b, c: c, suggestion: suggestion}, idx} ->
      IO.puts("  #{idx}. #{truncate_id(a)}")
      IO.puts("       ---> #{truncate_id(b)}")
      IO.puts("       ---> #{truncate_id(c)}")
      IO.puts("       missing: #{truncate_id(b)} --> #{truncate_id(c)}")

      if suggestion do
        IO.puts("       suggestion: #{suggestion}")
      end

      IO.puts("")
    end)
  end

  # ---------------------------------------------------------------------------
  # Cluster printer
  # ---------------------------------------------------------------------------

  defp print_clusters(clusters) do
    total = length(clusters)
    IO.puts("  #{total} connected component(s):\n")

    clusters
    |> Enum.with_index(1)
    |> Enum.each(fn {component, idx} ->
      size = MapSet.size(component)
      members = component |> MapSet.to_list() |> Enum.map(&truncate_id/1) |> Enum.take(5)
      overflow = max(0, size - 5)

      preview =
        if overflow > 0 do
          Enum.join(members, ", ") <> ", +#{overflow} more"
        else
          Enum.join(members, ", ")
        end

      IO.puts("  #{idx}. #{size} node(s): #{preview}")
    end)

    IO.puts("")
  end

  # ---------------------------------------------------------------------------
  # Hub printer
  # ---------------------------------------------------------------------------

  defp print_hubs(hubs) do
    IO.puts("  #{length(hubs)} hub(s) found:\n")

    Enum.each(hubs, fn %{id: id, degree: degree, sigma: sigma} ->
      bar = String.duplicate("=", min(degree, 40))

      IO.puts(
        "  #{String.pad_trailing(truncate_id(id), 28)} #{String.pad_leading(Integer.to_string(degree), 4)} edges  #{sigma}σ  #{bar}"
      )
    end)

    IO.puts("")
  end

  # ---------------------------------------------------------------------------
  # Shared helpers
  # ---------------------------------------------------------------------------

  defp parse_limit(["--limit", n | _], _default) do
    case Integer.parse(n) do
      {int, ""} when int > 0 -> int
      _ -> 20
    end
  end

  defp parse_limit(_, default), do: default

  # Truncate long SHA256 context IDs for display; leave short names as-is
  defp truncate_id(id) when is_binary(id) and byte_size(id) == 32 do
    String.slice(id, 0, 12) <> "..."
  end

  defp truncate_id(id), do: id
end
