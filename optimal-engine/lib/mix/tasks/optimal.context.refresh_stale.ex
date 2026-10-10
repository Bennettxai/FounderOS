defmodule Mix.Tasks.Optimal.Context.RefreshStale do
  @shortdoc "Refresh stale Memory Core Context Packages"

  @moduledoc """
  Refresh stale Memory Core Context Packages.

  This task is intended for manual runs, cron, or scheduler wrappers. It does not
  implement separate retrieval logic; it calls the same governed batch refresh
  service used by the API.

  ## Usage

      mix optimal.context.refresh_stale --workspace default
      mix optimal.context.refresh_stale --workspace default --batch-limit 25
      mix optimal.context.refresh_stale --workspace default --json
  """

  use Mix.Task

  alias OptimalEngine.MemoryCore

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _rest, _invalid} =
      OptionParser.parse(args,
        strict: [
          workspace: :string,
          tenant: :string,
          actor: :string,
          batch_limit: :integer,
          limit: :integer,
          continue_on_error: :boolean,
          json: :boolean
        ],
        aliases: [w: :workspace, t: :tenant]
      )

    opts =
      [
        workspace_id: Keyword.get(parsed, :workspace, "default"),
        tenant_id: Keyword.get(parsed, :tenant, "default"),
        actor_id: Keyword.get(parsed, :actor, "system:context-refresh"),
        batch_limit: Keyword.get(parsed, :batch_limit, 50),
        limit: Keyword.get(parsed, :limit),
        continue_on_error: Keyword.get(parsed, :continue_on_error, true)
      ]
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    case MemoryCore.refresh_stale_context_packages(opts) do
      {:ok, result} ->
        payload = %{
          workspace_id: Keyword.fetch!(opts, :workspace_id),
          tenant_id: Keyword.fetch!(opts, :tenant_id),
          stale_context_package_ids: result.stale_context_package_ids,
          refreshed_count: length(result.refreshed_context_packages),
          error_count: length(result.errors),
          refreshed_context_package_ids: Enum.map(result.refreshed_context_packages, & &1.id),
          errors: result.errors
        }

        if Keyword.get(parsed, :json, false) do
          Mix.shell().info(Jason.encode!(payload))
        else
          Mix.shell().info(
            "Refreshed #{payload.refreshed_count} stale Context Package(s) " <>
              "for workspace #{payload.workspace_id}; errors=#{payload.error_count}"
          )
        end

      {:error, reason} ->
        Mix.shell().error("Context refresh failed: #{inspect(reason)}")
        System.halt(1)
    end
  end
end
