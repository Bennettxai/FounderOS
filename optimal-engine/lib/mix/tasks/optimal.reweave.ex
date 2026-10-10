defmodule Mix.Tasks.Optimal.Reweave do
  @shortdoc "Find stale contexts related to a topic and suggest updates"
  @moduledoc """
  Backward pass: searches for contexts related to a topic, scores their staleness,
  and generates update suggestions.

  Usage:
      mix optimal.reweave "Alice"
      mix optimal.reweave "pricing" --days 60 --limit 5
      mix optimal.reweave "Platform Launch" --days 14 --limit 3

  Options:
    --days    Days before a context is considered fully stale (default: 30)
    --limit   Max contexts to return (default: 10)
  """

  use Mix.Task

  @separator String.duplicate("─", 60)

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, positional, _} =
      OptionParser.parse(args, strict: [days: :integer, limit: :integer])

    topic = Enum.join(positional, " ")

    if topic == "" do
      IO.puts("Usage: mix optimal.reweave \"topic\" [--days N] [--limit N]")
      System.halt(1)
    end

    opts =
      []
      |> maybe_put(:staleness_days, parsed[:days])
      |> maybe_put(:max_results, parsed[:limit])

    IO.puts("")
    IO.puts("  Reweaver — Backward Pass")
    IO.puts("  Topic: \"#{topic}\"")
    IO.puts("  #{@separator}")

    case OptimalEngine.Insight.Reweave.reweave(topic, opts) do
      {:ok, []} ->
        IO.puts("  All related contexts are up to date.")
        IO.puts("")

      {:ok, suggestions} ->
        IO.puts("  Found #{length(suggestions)} context(s) that may need updating:")
        IO.puts("")
        Enum.each(suggestions, &print_suggestion/1)
    end
  end

  defp print_suggestion(s) do
    bar_filled = round(s.staleness * 10)
    bar_empty = 10 - bar_filled
    staleness_bar = String.duplicate("█", bar_filled) <> String.duplicate("░", bar_empty)

    IO.puts("  #{s.title}")
    IO.puts("    Node:      #{s.node}")
    IO.puts("    Age:       #{s.days_old}d")
    IO.puts("    Staleness: [#{staleness_bar}] #{s.staleness}")
    IO.puts("    Suggest:   #{s.suggestion}")
    IO.puts("")
  end

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, value), do: [{key, value} | opts]
end
