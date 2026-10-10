defmodule Mix.Tasks.Optimal.Pull do
  @shortdoc "Run all enabled connectors once (manual/cron intake feed)"

  @moduledoc """
  Runs one pull cycle across every enabled connector and drives the
  emitted signals through the intake pipeline.

  This is the manual / cron entry point for the same loop the
  `OptimalEngine.Connectors.PullScheduler` runs on an interval. Idempotent:
  files already ingested (tracked per-connector via its cursor) are skipped.

  ## Usage

      mix optimal.pull                       — run every enabled connector once
      mix optimal.pull --tenant default      — scope to a tenant
      mix optimal.pull --connector sources_folder
      mix optimal.pull --max-retries 3

  ## Options

    --tenant       <id>      tenant scope (default: "default")
    --connector    <kind>    run only this connector kind (repeatable)
    --max-retries  <n>       runner retry cap for transient errors
  """

  use Mix.Task

  alias OptimalEngine.Connectors.PullScheduler

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _rest, _} =
      OptionParser.parse(args,
        strict: [tenant: :string, connector: :keep, max_retries: :integer]
      )

    opts =
      [tenant_id: Keyword.get(parsed, :tenant, "default")]
      |> maybe_connectors(parsed)
      |> maybe_max_retries(parsed)

    {:ok, summary} = PullScheduler.run_once(opts)
    report(summary)
  end

  defp maybe_connectors(opts, parsed) do
    case Keyword.get_values(parsed, :connector) do
      [] -> opts
      kinds -> Keyword.put(opts, :connectors, Enum.map(kinds, &String.to_atom/1))
    end
  end

  defp maybe_max_retries(opts, parsed) do
    case Keyword.get(parsed, :max_retries) do
      nil -> opts
      n -> Keyword.put(opts, :max_retries, n)
    end
  end

  defp report(summary) do
    Mix.shell().info("Pull complete (tenant=#{summary.tenant_id})")
    Mix.shell().info("  signals ingested: #{summary.ingested_count}")
    Mix.shell().info("  connectors with errors: #{summary.error_count}")

    Enum.each(summary.connector_results, fn r ->
      detail = if r.reason, do: " (#{inspect(r.reason)})", else: ""
      Mix.shell().info("  - #{r.kind} [#{r.status}] signals=#{r.signals}#{detail}")
    end)
  end
end
