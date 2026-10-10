defmodule OptimalEngine.MemoryCore.ClaimReview do
  @moduledoc """
  Review boundary for pending Claims.

  Claims are not truth. This module is the controlled lifecycle boundary for
  accepting or rejecting them after evidence/policy review. Promotion creates a
  Fact and Memory Object through the governed lifecycle modules; rejection only updates the
  Claim state.
  """

  alias OptimalEngine.MemoryCore.{FactPromoter, MemoryObject, Store}

  @spec pending(keyword()) :: {:ok, [map()]} | {:error, term()}
  def pending(opts \\ []) do
    opts
    |> Keyword.put(:review_status, "unreviewed")
    |> Keyword.put(:lifecycle_state, "pending")
    |> Store.list_claims()
  end

  @spec queue(keyword()) ::
          {:ok,
           %{
             tenant_id: String.t(),
             workspace_id: String.t(),
             count: non_neg_integer(),
             review_counts: map(),
             lifecycle_counts: map(),
             claims: [map()]
           }}
          | {:error, term()}
  def queue(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    with {:ok, claims} <-
           Store.list_claims(
             tenant_id: tenant_id,
             workspace_id: workspace_id,
             review_status: Keyword.get(opts, :review_status),
             lifecycle_state: Keyword.get(opts, :lifecycle_state),
             limit: Keyword.get(opts, :limit, 100)
           ) do
      {:ok,
       %{
         tenant_id: tenant_id,
         workspace_id: workspace_id,
         count: length(claims),
         review_counts: count_by(claims, :review_status),
         lifecycle_counts: count_by(claims, :lifecycle_state),
         claims: claims
       }}
    end
  end

  @spec get(String.t(), keyword()) :: {:ok, map()} | {:error, :not_found}
  def get(claim_id, opts \\ []), do: Store.get_claim(claim_id, opts)

  @spec reject(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def reject(claim_id, opts \\ []) when is_binary(claim_id) do
    workspace_id = Keyword.get(opts, :workspace_id)
    tenant_id = Keyword.get(opts, :tenant_id, "default")

    with {:ok, claim} <-
           Store.get_claim(claim_id, tenant_id: tenant_id, workspace_id: workspace_id),
         :ok <-
           Store.update_claim_review(claim_id,
             tenant_id: tenant_id,
             workspace_id: workspace_id || claim.workspace_id,
             review_status: "rejected",
             lifecycle_state: "rejected",
             actor_id: Keyword.get(opts, :actor_id)
           ) do
      Store.get_claim(claim_id,
        tenant_id: tenant_id,
        workspace_id: workspace_id || claim.workspace_id
      )
    end
  end

  @spec promote(String.t(), keyword()) ::
          {:ok, %{claim: map(), fact: map(), memory_object: map()}} | {:error, term()}
  def promote(claim_id, opts \\ []) when is_binary(claim_id) do
    workspace_id = Keyword.get(opts, :workspace_id)
    tenant_id = Keyword.get(opts, :tenant_id, "default")

    with {:ok, claim} <-
           Store.get_claim(claim_id, tenant_id: tenant_id, workspace_id: workspace_id),
         :ok <- ensure_promotable(claim),
         {:ok, policy} <- promotion_policy(claim, opts),
         {:ok, fact} <- FactPromoter.promote(claim, fact_opts(opts, policy)),
         :ok <- invalidate_explicit_supersession(claim, opts),
         {:ok, memory} <- MemoryObject.build_from_fact(fact, memory_opts(opts)),
         :ok <- apply_promotion_policy(policy, fact, opts),
         {:ok, reviewed_claim} <-
           Store.get_claim(claim.id, tenant_id: claim.tenant_id, workspace_id: claim.workspace_id) do
      accepted_claim =
        reviewed_claim
        |> Map.put(:review_status, "accepted")
        |> Map.put(:lifecycle_state, "accepted")

      {:ok, %{claim: accepted_claim, fact: fact, memory_object: memory}}
    end
  end

  defp ensure_promotable(%{review_status: "rejected"}), do: {:error, :claim_rejected}
  defp ensure_promotable(%{lifecycle_state: "rejected"}), do: {:error, :claim_rejected}
  defp ensure_promotable(_claim), do: :ok

  defp promotion_policy(claim, opts) do
    with :ok <- ensure_not_stale(claim, opts),
         {:ok, current_facts} <- Store.list_current_facts_for_claim(claim),
         {:ok, superseded_fact} <- resolve_superseded_fact(claim, current_facts, opts),
         :ok <- ensure_no_unresolved_conflicts(claim, current_facts, superseded_fact, opts) do
      {:ok,
       %{
         stale_checked: true,
         superseded_fact: superseded_fact,
         current_fact_count: length(current_facts)
       }}
    end
  end

  defp ensure_not_stale(%{stale_after: stale_after}, opts)
       when is_binary(stale_after) and stale_after != "" do
    if Keyword.get(opts, :allow_stale, false) do
      :ok
    else
      case parse_datetime(stale_after) do
        {:ok, stale_time} ->
          case DateTime.compare(stale_time, DateTime.utc_now()) do
            :lt -> {:error, {:claim_stale, stale_after}}
            _ -> :ok
          end

        :error ->
          :ok
      end
    end
  end

  defp ensure_not_stale(_claim, _opts), do: :ok

  defp resolve_superseded_fact(claim, current_facts, opts) do
    case Keyword.get(opts, :supersedes_fact_id) do
      nil ->
        {:ok, nil}

      fact_id ->
        current_match = Enum.find(current_facts, &(&1.id == fact_id))

        cond do
          current_match ->
            {:ok, current_match}

          true ->
            with {:ok, fact} <-
                   Store.get_fact(fact_id,
                     tenant_id: claim.tenant_id,
                     workspace_id: claim.workspace_id
                   ),
                 :ok <- ensure_current_fact(fact) do
              {:ok, fact}
            else
              {:error, :not_found} -> {:error, {:superseded_fact_not_found, fact_id}}
              {:error, reason} -> {:error, reason}
            end
        end
    end
  end

  defp ensure_current_fact(%{lifecycle_state: "accepted", contradiction_status: "none"} = fact) do
    ensure_fact_not_stale(fact)
  end

  defp ensure_current_fact(fact), do: {:error, {:superseded_fact_not_current, fact.id}}

  defp ensure_fact_not_stale(%{stale_after: stale_after} = fact)
       when is_binary(stale_after) and stale_after != "" do
    case parse_datetime(stale_after) do
      {:ok, stale_time} ->
        case DateTime.compare(stale_time, DateTime.utc_now()) do
          :lt -> {:error, {:superseded_fact_stale, fact.id}}
          _ -> :ok
        end

      :error ->
        :ok
    end
  end

  defp ensure_fact_not_stale(_fact), do: :ok

  defp ensure_no_unresolved_conflicts(claim, current_facts, superseded_fact, opts) do
    fact_text = normalize_text(Keyword.get(opts, :fact_text) || claim.claim_text)
    superseded_id = if superseded_fact, do: superseded_fact.id

    conflicts =
      current_facts
      |> Enum.reject(&(&1.id == superseded_id))
      |> Enum.reject(&(normalize_text(&1.fact_text) == fact_text))

    if conflicts == [] do
      :ok
    else
      {:error, {:contradicts_current_facts, Enum.map(conflicts, & &1.id)}}
    end
  end

  defp apply_promotion_policy(%{superseded_fact: nil}, _fact, _opts), do: :ok

  defp apply_promotion_policy(%{superseded_fact: old_fact}, _fact, _opts) do
    with {:ok, _count} <-
           Store.invalidate_context_packages_for_object("fact", old_fact.id,
             tenant_id: old_fact.tenant_id,
             workspace_id: old_fact.workspace_id,
             reason: "fact_superseded:#{old_fact.id}"
           ) do
      :ok
    end
  end

  defp invalidate_explicit_supersession(claim, opts) do
    case Keyword.get(opts, :supersedes_fact_id) do
      nil ->
        :ok

      fact_id ->
        case Store.invalidate_context_packages_for_object("fact", fact_id,
               tenant_id: claim.tenant_id,
               workspace_id: claim.workspace_id,
               reason: "fact_superseded:#{fact_id}"
             ) do
          {:ok, _count} -> :ok
          :ok -> :ok
          {:error, _} = error -> error
        end
    end
  end

  defp count_by(claims, field) do
    claims
    |> Enum.map(&Map.get(&1, field))
    |> Enum.reject(&is_nil/1)
    |> Enum.frequencies()
  end

  defp fact_opts(opts, policy) do
    [
      actor_id: Keyword.get(opts, :actor_id),
      verifier_id: Keyword.get(opts, :verifier_id) || Keyword.get(opts, :actor_id),
      fact_text: Keyword.get(opts, :fact_text),
      fact_type: Keyword.get(opts, :fact_type),
      verification_status: Keyword.get(opts, :verification_status, "verified"),
      aggregate_confidence: Keyword.get(opts, :aggregate_confidence),
      aggregate_precision: Keyword.get(opts, :aggregate_precision),
      valid_time_start: Keyword.get(opts, :valid_time_start),
      valid_time_end: Keyword.get(opts, :valid_time_end),
      stale_after: Keyword.get(opts, :stale_after),
      metadata: promotion_metadata(opts, policy)
    ]
    |> compact_nil()
  end

  defp memory_opts(opts) do
    [
      actor_id: Keyword.get(opts, :actor_id),
      summary: Keyword.get(opts, :summary),
      memory_type: Keyword.get(opts, :memory_type, "general"),
      salience: Keyword.get(opts, :salience),
      metadata: Keyword.get(opts, :memory_metadata, %{})
    ]
    |> compact_nil()
  end

  defp compact_nil(opts) do
    Enum.reject(opts, fn {_key, value} -> is_nil(value) end)
  end

  defp promotion_metadata(opts, policy) do
    superseded_fact = Map.get(policy, :superseded_fact)

    Keyword.get(opts, :fact_metadata, %{})
    |> Map.merge(%{
      promotion_policy: %{
        stale_checked: Map.get(policy, :stale_checked, false),
        current_fact_count: Map.get(policy, :current_fact_count, 0),
        supersedes_fact_id: if(superseded_fact, do: superseded_fact.id)
      }
    })
  end

  defp normalize_text(value) when is_binary(value) do
    value
    |> String.trim()
    |> String.downcase()
  end

  defp normalize_text(value), do: value |> to_string() |> normalize_text()

  defp parse_datetime(value) when is_binary(value) do
    case DateTime.from_iso8601(value) do
      {:ok, datetime, _offset} ->
        {:ok, datetime}

      _ ->
        case NaiveDateTime.from_iso8601(value) do
          {:ok, naive} -> {:ok, DateTime.from_naive!(naive, "Etc/UTC")}
          _ -> :error
        end
    end
  end
end
