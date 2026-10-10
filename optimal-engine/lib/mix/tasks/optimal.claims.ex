defmodule Mix.Tasks.Optimal.Claims do
  @shortdoc "List, inspect, promote, and reject Memory Core Claims"

  @moduledoc """
  Review queue CLI for Memory Core Claims.

  Claims are extracted assertions.
  They are not accepted truth until review or policy promotion creates Facts.

  ## Usage

      mix optimal.claims
      mix optimal.claims --workspace default:my-workspace
      mix optimal.claims --status unreviewed --limit 25
      mix optimal.claims get <claim-id> --workspace default:my-workspace
      mix optimal.claims promote <claim-id> --workspace default:my-workspace --actor user:reviewer
      mix optimal.claims reject <claim-id> --workspace default:my-workspace --actor user:reviewer

  ## Options

    * `--tenant` - tenant ID, defaults to `default`
    * `--workspace` - workspace ID, defaults to `default`
    * `--status` - review status filter, defaults to `unreviewed`
    * `--lifecycle` - lifecycle state filter
    * `--limit` - max rows to print, defaults to `50`
    * `--actor` - reviewer/promoter actor ID
    * `--fact-text` - fact text to use when promoting
    * `--summary` - memory-object summary when promoting
    * `--supersedes` - current fact ID superseded by the promoted claim
    * `--allow-stale` - allow stale claim promotion
  """

  use Mix.Task

  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Tenancy.Tenant

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _} =
      OptionParser.parse(args,
        strict: [
          tenant: :string,
          workspace: :string,
          status: :string,
          lifecycle: :string,
          limit: :integer,
          actor: :string,
          fact_text: :string,
          summary: :string,
          supersedes: :string,
          allow_stale: :boolean,
          json: :boolean
        ],
        aliases: [w: :workspace, n: :limit]
      )

    case rest do
      [] -> list_claims(opts)
      ["list"] -> list_claims(opts)
      ["get", claim_id] -> get_claim(claim_id, opts)
      ["promote", claim_id] -> promote_claim(claim_id, opts)
      ["reject", claim_id] -> reject_claim(claim_id, opts)
      _ -> Mix.raise("Usage: mix optimal.claims [list|get|promote|reject] [claim-id]")
    end
  end

  defp list_claims(opts) do
    tenant_id = Keyword.get(opts, :tenant, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace, "default")
    limit = Keyword.get(opts, :limit, 50)

    queue_opts =
      [
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        review_status: Keyword.get(opts, :status, "unreviewed"),
        lifecycle_state: Keyword.get(opts, :lifecycle),
        limit: limit
      ]
      |> reject_nil_values()

    case MemoryCore.claim_review_queue(queue_opts) do
      {:ok, queue} ->
        claims = Enum.take(queue.claims, limit)

        if Keyword.get(opts, :json, false) do
          IO.puts(Jason.encode!(%{queue | claims: Enum.map(claims, &claim_map/1)}, pretty: true))
        else
          IO.puts("")
          IO.puts("  Memory Core claims")
          IO.puts("  tenant:    #{queue.tenant_id}")
          IO.puts("  workspace: #{queue.workspace_id}")
          IO.puts("  count:     #{queue.count}")
          IO.puts("  " <> String.duplicate("-", 72))

          Enum.each(claims, &print_claim_row/1)

          if claims == [] do
            IO.puts("  No claims matched.")
          end

          IO.puts("")
        end

      {:error, reason} ->
        Mix.raise("optimal.claims failed: #{inspect(reason)}")
    end
  end

  defp get_claim(claim_id, opts) do
    case MemoryCore.get_claim(claim_id, scope_opts(opts)) do
      {:ok, claim} -> print_claim_detail(claim, opts)
      {:error, :not_found} -> Mix.raise("Claim not found: #{claim_id}")
      {:error, reason} -> Mix.raise("optimal.claims get failed: #{inspect(reason)}")
    end
  end

  defp promote_claim(claim_id, opts) do
    promote_opts =
      scope_opts(opts)
      |> maybe_put(:actor_id, Keyword.get(opts, :actor))
      |> maybe_put(:verifier_id, Keyword.get(opts, :actor))
      |> maybe_put(:fact_text, Keyword.get(opts, :fact_text))
      |> maybe_put(:summary, Keyword.get(opts, :summary))
      |> maybe_put(:supersedes_fact_id, Keyword.get(opts, :supersedes))
      |> maybe_put(:allow_stale, Keyword.get(opts, :allow_stale))

    case MemoryCore.promote_claim(claim_id, promote_opts) do
      {:ok, %{claim: claim, fact: fact, memory_object: memory}} ->
        IO.puts("")
        IO.puts("  Claim promoted")
        IO.puts("  " <> String.duplicate("-", 72))
        IO.puts("  Claim:  #{claim.id}  #{claim.review_status}/#{claim.lifecycle_state}")
        IO.puts("  Fact:   #{fact.id}  #{fact.lifecycle_state}/#{fact.verification_status}")
        IO.puts("  Memory: #{memory.id}")
        IO.puts("")

      {:error, reason} ->
        Mix.raise("optimal.claims promote failed: #{inspect(reason)}")
    end
  end

  defp reject_claim(claim_id, opts) do
    reject_opts =
      scope_opts(opts)
      |> maybe_put(:actor_id, Keyword.get(opts, :actor))

    case MemoryCore.reject_claim(claim_id, reject_opts) do
      {:ok, claim} ->
        IO.puts("")
        IO.puts("  Claim rejected")
        IO.puts("  " <> String.duplicate("-", 72))
        IO.puts("  Claim: #{claim.id}  #{claim.review_status}/#{claim.lifecycle_state}")
        IO.puts("")

      {:error, reason} ->
        Mix.raise("optimal.claims reject failed: #{inspect(reason)}")
    end
  end

  defp print_claim_row(claim) do
    IO.puts("  #{claim.id}")

    IO.puts(
      "    #{claim.review_status}/#{claim.lifecycle_state}  #{format_score(claim.aggregate_confidence)} confidence"
    )

    IO.puts("    #{claim.claim_text || "(no text)"}")
    IO.puts("    #{format_triple(claim.subject_anchor, claim.action_class, claim.object_anchor)}")
    IO.puts("")
  end

  defp print_claim_detail(claim, opts) do
    if Keyword.get(opts, :json, false) do
      IO.puts(Jason.encode!(claim_map(claim), pretty: true))
    else
      IO.puts("")
      IO.puts("  Claim #{claim.id}")
      IO.puts("  " <> String.duplicate("-", 72))
      IO.puts("  tenant:     #{claim.tenant_id}")
      IO.puts("  workspace:  #{claim.workspace_id}")
      IO.puts("  status:     #{claim.review_status}/#{claim.lifecycle_state}")
      IO.puts("  type:       #{claim.claim_type}")
      IO.puts("  confidence: #{format_score(claim.aggregate_confidence)}")
      IO.puts("  precision:  #{format_score(claim.aggregate_precision)}")
      IO.puts("  source:     #{claim.source_package_id || "(none)"}")
      IO.puts("  text:       #{claim.claim_text || "(no text)"}")

      IO.puts(
        "  triple:     #{format_triple(claim.subject_anchor, claim.action_class, claim.object_anchor)}"
      )

      IO.puts("")
    end
  end

  defp claim_map(claim) do
    claim
    |> Map.from_struct()
    |> Map.take([
      :id,
      :tenant_id,
      :workspace_id,
      :source_package_id,
      :signal_id,
      :claim_text,
      :claim_type,
      :subject_anchor,
      :action_class,
      :object_anchor,
      :aggregate_confidence,
      :aggregate_precision,
      :lifecycle_state,
      :review_status,
      :stale_after,
      :created_at,
      :updated_at
    ])
  end

  defp scope_opts(opts) do
    [
      tenant_id: Keyword.get(opts, :tenant, Tenant.default_id()),
      workspace_id: Keyword.get(opts, :workspace, "default")
    ]
  end

  defp reject_nil_values(opts), do: Enum.reject(opts, fn {_key, value} -> is_nil(value) end)
  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, value), do: Keyword.put(opts, key, value)

  defp format_score(nil), do: "n/a"
  defp format_score(score) when is_float(score), do: :erlang.float_to_binary(score, decimals: 2)
  defp format_score(score), do: to_string(score)

  defp format_triple(nil, nil, nil), do: "(no subject/action/object)"

  defp format_triple(subject, action, object) do
    "#{subject || "?"} / #{action || "?"} / #{object || "?"}"
  end
end
