defmodule Mix.Tasks.Optimal.Rethink do
  @shortdoc "Synthesize accumulated observations into actionable knowledge"
  @moduledoc """
  When enough observations have accumulated on a topic (confidence >= 1.5),
  generates a synthesis report with evidence and proposed updates.

  Usage:
      mix optimal.rethink "process"
      mix optimal.rethink "people" --force

  Options:
    --force    Bypass the confidence threshold and synthesize regardless
  """

  use Mix.Task

  @separator String.duplicate("─", 60)

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, positional, _} = OptionParser.parse(args, strict: [force: :boolean])

    topic = Enum.join(positional, " ")

    if topic == "" do
      IO.puts("\n#{@separator}")
      IO.puts("  RethinkEngine — Evidence Synthesis")
      IO.puts(@separator)
      IO.puts("")
      show_candidates()
    else
      opts = if parsed[:force], do: [force: true], else: []

      IO.puts("\n#{@separator}")
      IO.puts("  RethinkEngine — Evidence Synthesis for \"#{topic}\"")
      IO.puts(@separator)
      IO.puts("")

      case OptimalEngine.Insight.Rethink.rethink(topic, opts) do
        {:ok, %{status: :insufficient_evidence} = result} ->
          IO.puts("  Status:       Insufficient evidence")
          IO.puts("  Observations: #{result.observation_count}")
          IO.puts("  Confidence:   #{result.total_confidence} / #{result.threshold}")
          IO.puts("")
          IO.puts("  #{result.message}")
          IO.puts("")

        {:ok, %{status: :synthesized} = result} ->
          IO.puts("  Status:     Synthesized")

          IO.puts(
            "  Evidence:   #{result.observation_count} observations, #{result.related_context_count} related contexts"
          )

          IO.puts("  Confidence: #{result.total_confidence}")
          IO.puts("")
          IO.puts("  Synthesis (#{result.synthesis.method}):")
          IO.puts("  #{result.synthesis.summary}")
          IO.puts("")

          unless result.synthesis.patterns == [] do
            IO.puts("  Patterns identified:")

            Enum.each(result.synthesis.patterns, fn p ->
              IO.puts("    - #{p}")
            end)

            IO.puts("")
          end

          if result.proposed_updates == [] do
            IO.puts("  No specific file updates proposed.")
          else
            IO.puts("  Proposed updates:")

            Enum.each(result.proposed_updates, fn u ->
              IO.puts("    [#{u.action}] #{u.file}")
              IO.puts("      #{String.slice(u.content || "", 0, 80)}")
            end)
          end

          IO.puts("")

        {:ok, %{status: :error, message: msg}} ->
          IO.puts("  Error: #{msg}")
          IO.puts("")
      end
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Show escalation candidates when no topic given
  # ---------------------------------------------------------------------------

  defp show_candidates do
    case OptimalEngine.Insight.Remember.escalation_candidates() do
      {:ok, []} ->
        IO.puts("  No categories have reached escalation threshold yet.")
        IO.puts("  Use 'mix optimal.remember' to accumulate observations.")
        IO.puts("")

      {:ok, candidates} ->
        IO.puts("  Categories ready for rethink:\n")

        Enum.each(candidates, fn c ->
          marker = if c.ready_for_rethink, do: ">>>", else: "   "

          IO.puts(
            "  #{marker} [#{c.category}] #{c.count} observations, confidence: #{Float.round(c.total_confidence * 1.0, 2)}"
          )
        end)

        IO.puts("")
        IO.puts("  Run: mix optimal.rethink \"category_name\"")
        IO.puts("")
    end
  rescue
    _ ->
      IO.puts(
        "  Could not load escalation candidates. Run 'mix optimal.remember --escalations' instead."
      )

      IO.puts("")
  end
end
