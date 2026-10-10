defmodule Mix.Tasks.Optimal.Verify do
  @shortdoc "Cold-read test of L0 abstract fidelity"
  @moduledoc """
  Samples contexts and evaluates how well their L0 abstracts represent
  their full content. Reports per-context scores and an aggregate grade.

  Usage:
      mix optimal.verify
      mix optimal.verify --sample 20
      mix optimal.verify --sample 10 --node project-platform-launch
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _, _} = OptionParser.parse(args, strict: [sample: :integer, node: :string])

    opts = []
    opts = if s = parsed[:sample], do: [{:sample, s} | opts], else: opts
    opts = if n = parsed[:node], do: [{:node, n} | opts], else: opts

    IO.puts("\nVerify Engine — L0 Fidelity Test\n")

    case OptimalEngine.Insight.Verify.verify(opts) do
      {:ok, %{scores: [], message: msg}} ->
        IO.puts("  #{msg}")

      {:ok, result} ->
        IO.puts("  Sample size:      #{result.sample_size}")
        IO.puts("  Aggregate score:  #{result.aggregate}")
        IO.puts("  Assessment:       #{result.message}\n")

        IO.puts("  Per-context scores:")
        IO.puts("  " <> String.duplicate("-", 72))

        Enum.each(result.scores, fn s ->
          grade_color =
            case s.grade do
              "A" -> IO.ANSI.green()
              "B" -> IO.ANSI.cyan()
              "C" -> IO.ANSI.yellow()
              _ -> IO.ANSI.red()
            end

          title = String.slice(s.title || "(untitled)", 0, 40)
          score_str = if s.score, do: :erlang.float_to_binary(s.score, decimals: 2), else: "N/A"

          IO.puts(
            "  #{grade_color}[#{s.grade}]#{IO.ANSI.reset()} " <>
              String.pad_trailing(title, 42) <>
              String.pad_leading(score_str, 6) <>
              "  (#{s.node})"
          )
        end)

        IO.puts("")
    end
  end
end
