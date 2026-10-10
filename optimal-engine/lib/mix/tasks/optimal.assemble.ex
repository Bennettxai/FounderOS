defmodule Mix.Tasks.Optimal.Assemble do
  @moduledoc """
  Assembles tiered context for a query using RRF fusion.

  ## Usage

      mix optimal.assemble "Platform Launch pricing"
      mix optimal.assemble "Alice" --tier l0
      mix optimal.assemble "revenue" --limit 20
  """

  use Mix.Task
  require Logger

  @shortdoc "Assemble tiered context (L0/L1/L2) for a query"

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, _} =
      OptionParser.parse(args, strict: [tier: :string, limit: :integer])

    query = Enum.join(rest, " ")

    if query == "" do
      Mix.shell().error("Usage: mix optimal.assemble \"query\" [--tier l0|l1|l2] [--limit N]")
      System.halt(1)
    end

    tier = Keyword.get(parsed, :tier)

    case tier do
      "l0" ->
        {:ok, content} = OptimalEngine.Retrieval.ContextAssembler.l0()
        Mix.shell().info(content)

      _ ->
        opts = Keyword.take(parsed, [:limit])
        {:ok, result} = OptimalEngine.Retrieval.ContextAssembler.assemble(query, opts)

        Mix.shell().info("""
        === L0 (#{estimate_tokens(result.l0)} tokens) ===
        #{result.l0}

        === L1 (#{estimate_tokens(result.l1)} tokens) ===
        #{result.l1}

        === L2 (#{estimate_tokens(result.l2)} tokens) ===
        #{String.slice(result.l2, 0, 2000)}#{if String.length(result.l2) > 2000, do: "\n...[truncated]", else: ""}

        === Summary ===
        Total tokens: ~#{result.total_tokens}
        Sources: #{length(result.sources)}
        Search results: #{length(result.search_scores)}
        """)
    end
  end

  defp estimate_tokens(text) when is_binary(text), do: div(String.length(text), 4)
  defp estimate_tokens(_), do: 0
end
