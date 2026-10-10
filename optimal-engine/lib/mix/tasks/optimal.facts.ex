defmodule Mix.Tasks.Optimal.Facts do
  @shortdoc "List and inspect accepted Memory Core Facts"

  @moduledoc """
  CLI for accepted Memory Core Facts.

  Facts are reviewed or policy-accepted assertions.
  Agents should create or review Claims first, then promote them into Facts.

  ## Usage

      mix optimal.facts
      mix optimal.facts --workspace default:my-workspace
      mix optimal.facts --subject customer_portal --current
      mix optimal.facts get <fact-id> --workspace default:my-workspace

  ## Options

    * `--tenant` - tenant ID, defaults to `default`
    * `--workspace` - workspace ID, defaults to `default`
    * `--subject` - subject anchor filter
    * `--action` - action class filter
    * `--lifecycle` - lifecycle state filter, defaults to `accepted`
    * `--current` - only facts with open transaction time
    * `--limit` - max rows to print, defaults to `50`
  """

  use Mix.Task

  alias OptimalEngine.MemoryCore.Store
  alias OptimalEngine.Tenancy.Tenant

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _} =
      OptionParser.parse(args,
        strict: [
          tenant: :string,
          workspace: :string,
          subject: :string,
          action: :string,
          lifecycle: :string,
          current: :boolean,
          limit: :integer,
          json: :boolean
        ],
        aliases: [w: :workspace, n: :limit]
      )

    case rest do
      [] -> list_facts(opts)
      ["list"] -> list_facts(opts)
      ["get", fact_id] -> get_fact(fact_id, opts)
      _ -> Mix.raise("Usage: mix optimal.facts [list|get] [fact-id]")
    end
  end

  defp list_facts(opts) do
    tenant_id = Keyword.get(opts, :tenant, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace, "default")
    limit = Keyword.get(opts, :limit, 50)

    list_opts =
      [
        tenant_id: tenant_id,
        subject_anchor: Keyword.get(opts, :subject),
        action_class: Keyword.get(opts, :action),
        lifecycle_state: Keyword.get(opts, :lifecycle, "accepted"),
        open_transaction_only: Keyword.get(opts, :current, false)
      ]
      |> reject_nil_values()

    case Store.list_facts(workspace_id, list_opts) do
      {:ok, facts} ->
        facts = Enum.take(facts, limit)

        if Keyword.get(opts, :json, false) do
          IO.puts(Jason.encode!(Enum.map(facts, &fact_map/1), pretty: true))
        else
          IO.puts("")
          IO.puts("  Memory Core facts")
          IO.puts("  tenant:    #{tenant_id}")
          IO.puts("  workspace: #{workspace_id}")
          IO.puts("  " <> String.duplicate("-", 72))

          Enum.each(facts, &print_fact_row/1)

          if facts == [] do
            IO.puts("  No facts matched.")
          end

          IO.puts("")
        end

      {:error, reason} ->
        Mix.raise("optimal.facts failed: #{inspect(reason)}")
    end
  end

  defp get_fact(fact_id, opts) do
    workspace_id = Keyword.get(opts, :workspace, "default")
    tenant_id = Keyword.get(opts, :tenant, Tenant.default_id())

    case Store.get_fact(workspace_id, fact_id, tenant_id: tenant_id) do
      {:ok, fact} -> print_fact_detail(fact, opts)
      {:error, :not_found} -> Mix.raise("Fact not found: #{fact_id}")
      {:error, reason} -> Mix.raise("optimal.facts get failed: #{inspect(reason)}")
    end
  end

  defp print_fact_row(fact) do
    IO.puts("  #{fact.id}")

    IO.puts(
      "    #{fact.lifecycle_state}/#{fact.verification_status}  #{format_score(fact.aggregate_confidence)} confidence"
    )

    IO.puts("    #{fact.fact_text || "(no text)"}")
    IO.puts("    #{format_triple(fact.subject_anchor, fact.action_class, fact.object_anchor)}")
    IO.puts("")
  end

  defp print_fact_detail(fact, opts) do
    if Keyword.get(opts, :json, false) do
      IO.puts(Jason.encode!(fact_map(fact), pretty: true))
    else
      IO.puts("")
      IO.puts("  Fact #{fact.id}")
      IO.puts("  " <> String.duplicate("-", 72))
      IO.puts("  tenant:       #{fact.tenant_id}")
      IO.puts("  workspace:    #{fact.workspace_id}")
      IO.puts("  status:       #{fact.lifecycle_state}/#{fact.verification_status}")
      IO.puts("  type:         #{fact.fact_type}")
      IO.puts("  confidence:   #{format_score(fact.aggregate_confidence)}")
      IO.puts("  precision:    #{format_score(fact.aggregate_precision)}")
      IO.puts("  verifier:     #{fact.verifier_id || "(none)"}")
      IO.puts("  claims:       #{Enum.join(fact.accepted_claim_ids || [], ", ")}")
      IO.puts("  text:         #{fact.fact_text || "(no text)"}")

      IO.puts(
        "  triple:       #{format_triple(fact.subject_anchor, fact.action_class, fact.object_anchor)}"
      )

      IO.puts("")
    end
  end

  defp fact_map(fact) do
    fact
    |> Map.from_struct()
    |> Map.take([
      :id,
      :tenant_id,
      :workspace_id,
      :fact_text,
      :fact_type,
      :subject_anchor,
      :action_class,
      :object_anchor,
      :accepted_claim_ids,
      :verifier_id,
      :verification_status,
      :aggregate_confidence,
      :aggregate_precision,
      :lifecycle_state,
      :contradiction_status,
      :valid_time_start,
      :valid_time_end,
      :transaction_time_start,
      :transaction_time_end,
      :stale_after,
      :created_at,
      :updated_at,
      :supersedes,
      :superseded_by
    ])
  end

  defp reject_nil_values(opts), do: Enum.reject(opts, fn {_key, value} -> is_nil(value) end)

  defp format_score(nil), do: "n/a"
  defp format_score(score) when is_float(score), do: :erlang.float_to_binary(score, decimals: 2)
  defp format_score(score), do: to_string(score)

  defp format_triple(nil, nil, nil), do: "(no subject/action/object)"

  defp format_triple(subject, action, object) do
    "#{subject || "?"} / #{action || "?"} / #{object || "?"}"
  end
end
