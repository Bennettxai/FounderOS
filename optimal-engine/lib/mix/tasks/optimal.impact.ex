defmodule Mix.Tasks.Optimal.Impact do
  @shortdoc "Run an impact analysis for a named entity or domain node"
  @moduledoc """
  Runs a direct impact analysis on a named person or domain node.

  Equivalent to `mix optimal.simulate "What if <name> leaves/cancels?"` but takes
  a name directly and auto-detects whether it is an entity or a node.

  Usage:
      mix optimal.impact "Dan"
      mix optimal.impact "Alice"
      mix optimal.impact "services-group"
      mix optimal.impact "Dan" --depth 4

  Options:
    --depth   BFS traversal depth (default: 3)
  """

  use Mix.Task

  @separator String.duplicate("─", 60)
  @thin_sep String.duplicate("·", 60)

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        switches: [depth: :integer],
        aliases: [d: :depth]
      )

    target =
      case positional do
        [t | _] -> t
        [] -> Mix.raise("Usage: mix optimal.impact \"entity or node name\"")
      end

    sim_opts = Keyword.take(opts, [:depth]) |> rename_key(:depth, :max_depth)

    IO.puts("")
    IO.puts(@separator)
    IO.puts("  IMPACT ANALYSIS")
    IO.puts(@separator)
    IO.puts("  Target: #{target}")
    IO.puts("")

    case OptimalEngine.Insight.Simulate.impact_analysis(target, sim_opts) do
      {:ok, report} ->
        print_report(report)

      {:error, reason} ->
        IO.puts("Impact analysis failed: #{inspect(reason)}")
    end
  end

  defp rename_key(kw, old_key, new_key) do
    case Keyword.get(kw, old_key) do
      nil -> kw
      val -> kw |> Keyword.delete(old_key) |> Keyword.put(new_key, val)
    end
  end

  defp print_report(report) do
    print_summary(report)
    print_affected_nodes(report.affected_nodes)
    print_affected_entities(report.affected_entities)
    print_critical_dependencies(report.critical_dependencies)
    print_risk_assessment(report.risk_assessment)
    print_recommendations(report.recommendations)
    print_footer(report)
  end

  defp print_summary(report) do
    IO.puts("  Mutation type:        #{format_mutation_type(report.mutation_type)}")
    IO.puts("  Scenario:             #{report.scenario}")
    IO.puts("  Contexts affected:    #{report.total_contexts_affected}")
    IO.puts("  Nodes affected:       #{length(report.affected_nodes)}")
    IO.puts("")
  end

  defp print_affected_nodes([]) do
    IO.puts("  Affected Nodes: (none)")
    IO.puts("")
  end

  defp print_affected_nodes(nodes) do
    IO.puts("  Affected Nodes  (impact score / context coverage)")
    IO.puts(@thin_sep)

    Enum.each(nodes, fn node ->
      bar = ascii_bar(node.impact_score)
      score_str = node.impact_score |> Float.round(3) |> to_string() |> String.pad_leading(5)
      name_str = String.pad_trailing(node.node, 26)
      IO.puts("  #{name_str} #{score_str}  #{bar}  #{node.description}")
    end)

    IO.puts("")
  end

  defp print_affected_entities([]) do
    :ok
  end

  defp print_affected_entities(entities) do
    IO.puts("  Affected Entities")
    IO.puts(@thin_sep)

    Enum.each(entities, fn e ->
      impact_tag = if e.impact == :direct, do: "[direct]  ", else: "[indirect]"
      IO.puts("  #{impact_tag}  #{e.name}  (#{e.contexts} context(s))")
    end)

    IO.puts("")
  end

  defp print_critical_dependencies([]) do
    IO.puts("  Critical Dependencies: none identified")
    IO.puts("")
  end

  defp print_critical_dependencies(deps) do
    IO.puts("  Critical Dependencies")
    IO.puts(@thin_sep)

    Enum.each(deps, fn dep ->
      weight_pct = trunc(dep.weight * 100)
      IO.puts("  #{dep.from}  --#{dep.relation}-->  #{dep.to}  (#{weight_pct}% coverage)")
    end)

    IO.puts("")
  end

  defp print_risk_assessment(%{severity: severity, reasoning: reasoning}) do
    label = format_severity(severity)
    IO.puts("  Risk Assessment: #{label}")
    IO.puts(@thin_sep)
    IO.puts("  #{reasoning}")
    IO.puts("")
  end

  defp print_recommendations([]) do
    IO.puts("  Recommendations: none generated")
    IO.puts("")
  end

  defp print_recommendations(recs) do
    IO.puts("  Recommendations")
    IO.puts(@thin_sep)

    recs
    |> Enum.with_index(1)
    |> Enum.each(fn {rec, i} ->
      IO.puts("  #{i}. #{rec}")
    end)

    IO.puts("")
  end

  defp print_footer(report) do
    IO.puts(@separator)
    IO.puts("  Analysis complete: #{report.scenario}")
    IO.puts(@separator)
    IO.puts("")
  end

  defp ascii_bar(score) when is_float(score) do
    filled = min(trunc(score * 2), 20)
    empty = 20 - filled
    "[" <> String.duplicate("█", filled) <> String.duplicate("░", empty) <> "]"
  end

  defp format_mutation_type(:entity_removal), do: "entity removal"
  defp format_mutation_type(:node_cancellation), do: "node cancellation"
  defp format_mutation_type(:revenue_change), do: "revenue change"
  defp format_mutation_type(:dependency_break), do: "dependency break"
  defp format_mutation_type(:general), do: "general"

  defp format_severity(:critical), do: "CRITICAL"
  defp format_severity(:high), do: "HIGH"
  defp format_severity(:medium), do: "MEDIUM"
  defp format_severity(:low), do: "LOW"
end
