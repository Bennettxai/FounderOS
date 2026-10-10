defmodule Mix.Tasks.Optimal.Reflect do
  @shortdoc "Find missing edges from entity co-occurrences"
  @moduledoc """
  Scans the knowledge graph for entity pairs that co-occur in contexts
  but don't have direct edges, suggesting relationships to add.

  Usage:
      mix optimal.reflect
      mix optimal.reflect --min 3 --limit 10
      mix optimal.reflect --show-contexts "Alice" "Alice"

  Options:
    --min N              Minimum co-occurrences to qualify (default: 2)
    --limit N            Max suggestions to return (default: 20)
    --show-contexts A B  Show shared contexts for a specific entity pair
  """

  use Mix.Task

  @separator String.duplicate("─", 60)
  @thin_sep String.duplicate("·", 60)

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, positional, _} =
      OptionParser.parse(args,
        strict: [min: :integer, limit: :integer, show_contexts: :boolean],
        aliases: [m: :min, l: :limit]
      )

    cond do
      parsed[:show_contexts] ->
        case positional do
          [entity_a, entity_b | _] ->
            run_shared_contexts(entity_a, entity_b)

          _ ->
            Mix.raise("Usage: mix optimal.reflect --show-contexts \"Entity A\" \"Entity B\"")
        end

      true ->
        run_reflect(parsed)
    end
  end

  # ---------------------------------------------------------------------------
  # Reflection run
  # ---------------------------------------------------------------------------

  defp run_reflect(parsed) do
    opts = build_opts(parsed)

    IO.puts("")
    IO.puts(@separator)
    IO.puts("  Reflector — Missing Edge Detection")
    IO.puts(@separator)
    IO.puts("  min_cooccurrences: #{Keyword.get(opts, :min_cooccurrences, 2)}")
    IO.puts("  limit:             #{Keyword.get(opts, :limit, 20)}")
    IO.puts("")

    case OptimalEngine.Graph.Reflector.reflect(opts) do
      {:ok, []} ->
        IO.puts("  No missing edges found. Graph looks complete!")
        IO.puts("")

      {:ok, suggestions} ->
        IO.puts("  Found #{length(suggestions)} potential missing edge(s):")
        IO.puts(@thin_sep)
        IO.puts("")
        Enum.each(suggestions, &print_suggestion/1)
        IO.puts(@separator)

        IO.puts("  Tip: mix optimal.reflect --show-contexts \"A\" \"B\" to inspect shared contexts")

        IO.puts("")
    end
  end

  # ---------------------------------------------------------------------------
  # Shared contexts run
  # ---------------------------------------------------------------------------

  defp run_shared_contexts(entity_a, entity_b) do
    IO.puts("")
    IO.puts(@separator)
    IO.puts("  Shared Contexts: #{entity_a} + #{entity_b}")
    IO.puts(@separator)
    IO.puts("")

    case OptimalEngine.Graph.Reflector.shared_contexts(entity_a, entity_b) do
      {:ok, []} ->
        IO.puts("  No shared contexts found.")
        IO.puts("")

      {:ok, contexts} ->
        IO.puts("  #{length(contexts)} shared context(s):")
        IO.puts(@thin_sep)
        IO.puts("")

        Enum.each(contexts, fn ctx ->
          node_str = "[#{ctx.node}]" |> String.pad_trailing(22)
          IO.puts("  #{node_str}  #{ctx.title}")
          IO.puts("             id: #{ctx.id}")
          IO.puts("")
        end)

      {:error, reason} ->
        IO.puts("  Error: #{inspect(reason)}")
        IO.puts("")
    end
  end

  # ---------------------------------------------------------------------------
  # Formatting helpers
  # ---------------------------------------------------------------------------

  defp print_suggestion(s) do
    confidence_pct = s.confidence |> Kernel.*(100) |> trunc()
    bar = confidence_bar(s.confidence)

    IO.puts("  #{s.source}  --#{s.suggested_relation}-->  #{s.target}")
    IO.puts("    Co-occurrences: #{s.cooccurrences}  |  Confidence: #{confidence_pct}%  #{bar}")
    IO.puts("    #{s.reason}")
    IO.puts("")
  end

  defp confidence_bar(confidence) do
    filled = min(trunc(confidence * 20), 20)
    empty = 20 - filled
    "[" <> String.duplicate("█", filled) <> String.duplicate("░", empty) <> "]"
  end

  defp build_opts(parsed) do
    []
    |> maybe_put(:min_cooccurrences, parsed[:min])
    |> maybe_put(:limit, parsed[:limit])
  end

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, val), do: Keyword.put(opts, key, val)
end
