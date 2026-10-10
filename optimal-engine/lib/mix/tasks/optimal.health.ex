defmodule Mix.Tasks.Optimal.Health do
  @shortdoc "Run health diagnostics on the Optimal Engine store"
  @moduledoc """
  Runs 10 diagnostic checks on the knowledge base and reports issues.

  Checks:
    1. orphaned_contexts       — contexts with no edges
    2. stale_signals           — contexts not modified in 30+ days
    3. missing_cross_refs      — routed_to nodes missing cross_ref edges
    4. fts_drift               — mismatch between contexts and contexts_fts counts
    5. entity_merge_candidates — duplicate entity names (case-insensitive)
    6. node_imbalance          — nodes with >3x mean context count
    7. duplicate_detection     — identical titles within the same node
    8. broken_references       — supersedes pointing to nonexistent IDs
    9. embedding_coverage      — ratio of vectors to total contexts
   10. quality_distribution    — flag if >20% of contexts have S/N ratio < 0.4

  Exit codes:
    0 — all checks passed or warnings only
    1 — one or more critical issues found

  Usage:
      mix optimal.health
  """

  use Mix.Task

  @impl Mix.Task
  def run(_args) do
    Mix.Task.run("app.start")

    IO.puts(IO.ANSI.bright() <> "\nOptimal Engine Health Diagnostics\n" <> IO.ANSI.reset())

    {:ok, diagnostics} = OptimalEngine.Insight.Health.run()
    Enum.each(diagnostics, &print_check/1)
    print_summary(diagnostics)
    maybe_exit_nonzero(diagnostics)
  end

  # ---------------------------------------------------------------------------
  # Printing
  # ---------------------------------------------------------------------------

  defp print_check(%{
         name: name,
         severity: severity,
         message: message,
         details: details,
         fix: fix
       }) do
    {icon, color} = severity_format(severity)
    label = name |> to_string() |> String.replace("_", " ")

    IO.puts(color <> "  #{icon} #{label}" <> IO.ANSI.reset())
    IO.puts("    #{message}")

    if details != [] and length(details) <= 5 do
      Enum.each(details, fn item ->
        IO.puts("    " <> IO.ANSI.faint() <> format_detail(item) <> IO.ANSI.reset())
      end)
    end

    if details != [] and length(details) > 5 do
      first_five = Enum.take(details, 5)

      Enum.each(first_five, fn item ->
        IO.puts("    " <> IO.ANSI.faint() <> format_detail(item) <> IO.ANSI.reset())
      end)

      IO.puts("    " <> IO.ANSI.faint() <> "... and #{length(details) - 5} more" <> IO.ANSI.reset())
    end

    if fix && severity != :ok do
      IO.puts("    " <> IO.ANSI.cyan() <> "fix: #{fix}" <> IO.ANSI.reset())
    end

    IO.puts("")
  end

  defp print_summary(diagnostics) do
    %{ok: ok, warning: warning, critical: critical, total: total} =
      OptimalEngine.Insight.Health.summary(diagnostics)

    IO.puts(IO.ANSI.bright() <> "Summary" <> IO.ANSI.reset())
    IO.puts("  Total checks : #{total}")
    IO.puts(IO.ANSI.green() <> "  Passed       : #{ok}" <> IO.ANSI.reset())

    if warning > 0 do
      IO.puts(IO.ANSI.yellow() <> "  Warnings     : #{warning}" <> IO.ANSI.reset())
    else
      IO.puts("  Warnings     : #{warning}")
    end

    if critical > 0 do
      IO.puts(IO.ANSI.red() <> "  Critical     : #{critical}" <> IO.ANSI.reset())
    else
      IO.puts("  Critical     : #{critical}")
    end

    IO.puts("")
  end

  defp maybe_exit_nonzero(diagnostics) do
    has_critical = Enum.any?(diagnostics, &(&1.severity == :critical))
    if has_critical, do: System.halt(1)
  end

  # ---------------------------------------------------------------------------
  # Formatting helpers
  # ---------------------------------------------------------------------------

  defp severity_format(:ok), do: {"✓", IO.ANSI.green()}
  defp severity_format(:warning), do: {"!", IO.ANSI.yellow()}
  defp severity_format(:critical), do: {"✗", IO.ANSI.red()}

  defp format_detail(%{id: id, title: title}) when is_binary(id) and is_binary(title) do
    short_id = String.slice(id, 0, 12)
    "#{short_id}… — #{truncate(title, 60)}"
  end

  defp format_detail(%{node: node, title: title, count: count}) do
    "#{node} / \"#{truncate(title, 50)}\" (#{count}x)"
  end

  defp format_detail(%{node: node, count: count, mean: mean, threshold: threshold}) do
    "#{node}: #{count} contexts (mean=#{mean}, threshold=#{threshold})"
  end

  defp format_detail(%{canonical: name, count: count, variants: variants}) do
    "\"#{name}\" — #{count} variants: #{Enum.join(variants, ", ")}"
  end

  defp format_detail(%{contexts: ctx, fts: fts, drift: drift}) do
    "contexts=#{ctx}, fts=#{fts}, drift=#{drift}"
  end

  defp format_detail(%{vectors: vec, contexts: ctx, coverage_pct: pct}) do
    "#{vec}/#{ctx} contexts embedded (#{pct}%)"
  end

  defp format_detail(%{
         low_quality_count: low,
         total: total,
         ratio_pct: pct,
         threshold_pct: thresh
       }) do
    "#{low}/#{total} below S/N 0.4 (#{pct}%, threshold #{thresh}%)"
  end

  defp format_detail(other), do: inspect(other, limit: 5)

  defp truncate(str, max) when byte_size(str) <= max, do: str
  defp truncate(str, max), do: String.slice(str, 0, max) <> "…"
end
