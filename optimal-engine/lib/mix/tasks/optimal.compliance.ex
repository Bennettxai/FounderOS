defmodule Mix.Tasks.Optimal.Compliance do
  @shortdoc "DSAR export / erasure / retention sweep / legal holds"

  @moduledoc """
  Compliance operations — GDPR, CCPA, HIPAA, SOC 2.

  ## Usage

      mix optimal.compliance dsar <principal_id>
      mix optimal.compliance erase <principal_id>          — cascade delete
      mix optimal.compliance erase <principal_id> --preview
      mix optimal.compliance erase <principal_id> --force  — override holds
      mix optimal.compliance hold place <signal_id> --by <actor> --reason "..."
      mix optimal.compliance hold release <hold_id>
      mix optimal.compliance hold list
      mix optimal.compliance retention sweep               — apply policies
      mix optimal.compliance retention sweep --dry-run

  ## Options

    --tenant       Scope (default: default)
    --preview      Erase: show counts without deleting
    --force        Erase: proceed despite legal holds
    --by           Hold: actor placing the hold
    --reason       Hold: free-form justification
    --dry-run      Retention: compute targets without applying
  """

  use Mix.Task

  alias OptimalEngine.Compliance

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, _} =
      OptionParser.parse(args,
        strict: [
          tenant: :string,
          preview: :boolean,
          force: :boolean,
          by: :string,
          reason: :string,
          dry_run: :boolean
        ]
      )

    tenant = Keyword.get(parsed, :tenant, "default")

    case rest do
      ["dsar", principal_id | _] ->
        handle_dsar(principal_id, tenant)

      ["erase", principal_id | _] ->
        handle_erase(principal_id, tenant, parsed)

      ["hold", "place", signal_id | _] ->
        handle_hold_place(signal_id, tenant, parsed)

      ["hold", "release", hold_id | _] ->
        handle_hold_release(hold_id)

      ["hold", "list" | _] ->
        handle_hold_list(tenant)

      ["retention", "sweep" | _] ->
        handle_retention_sweep(tenant, parsed)

      _ ->
        Mix.raise(
          "Usage: mix optimal.compliance [dsar <id> | erase <id> | hold place|release|list | retention sweep]"
        )
    end
  end

  defp handle_dsar(principal_id, tenant) do
    case Compliance.dsar_export(principal_id, tenant) do
      {:ok, export} -> IO.puts(Jason.encode!(export, pretty: true))
      {:error, reason} -> Mix.raise("DSAR failed: #{inspect(reason)}")
    end
  end

  defp handle_erase(principal_id, tenant, parsed) do
    if Keyword.get(parsed, :preview, false) do
      {:ok, preview} = Compliance.erasure_preview(principal_id, tenant_id: tenant)
      IO.puts(Jason.encode!(preview, pretty: true))
    else
      opts = [tenant_id: tenant, force: Keyword.get(parsed, :force, false)]

      case Compliance.erase(principal_id, opts) do
        {:ok, report} -> IO.puts(Jason.encode!(report, pretty: true))
        {:error, reason} -> Mix.raise("Erasure failed: #{inspect(reason)}")
      end
    end
  end

  defp handle_hold_place(signal_id, tenant, parsed) do
    by = Keyword.get(parsed, :by) || Mix.raise("--by required")
    reason = Keyword.get(parsed, :reason) || Mix.raise("--reason required")

    case Compliance.place_legal_hold(signal_id, by, reason, tenant_id: tenant) do
      {:ok, id} -> IO.puts("Placed hold #{id}")
      {:error, reason} -> Mix.raise("Place hold failed: #{inspect(reason)}")
    end
  end

  defp handle_hold_release(hold_id) do
    case Compliance.release_legal_hold(String.to_integer(hold_id)) do
      :ok -> IO.puts("Released hold #{hold_id}")
      other -> Mix.raise("Release failed: #{inspect(other)}")
    end
  end

  defp handle_hold_list(tenant) do
    holds = Compliance.active_legal_holds(tenant)

    if holds == [] do
      IO.puts("No active holds.")
    else
      Enum.each(holds, fn h ->
        IO.puts("  [#{h.id}] #{h.signal_id} by #{h.held_by} — #{h.reason} (#{h.placed_at})")
      end)
    end
  end

  defp handle_retention_sweep(tenant, parsed) do
    {:ok, result} =
      Compliance.retention_sweep(
        tenant_id: tenant,
        dry_run: Keyword.get(parsed, :dry_run, false)
      )

    IO.puts("Retention sweep:")
    IO.puts("  policies_evaluated: #{result.policies_evaluated}")
    IO.puts("  actions_taken:      #{result.actions_taken}")
    IO.puts("  skipped_held:       #{result.skipped_held}")
    IO.puts("  errors:             #{result.errors}")
  end
end
