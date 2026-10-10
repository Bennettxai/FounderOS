defmodule OptimalEngine.MemoryCore.ActiveMemoryPool do
  @moduledoc """
  Task-scoped working memory for humans, agents, and tools.

  Pool observations are not accepted truth. Publishing an observation preserves
  it as a Source Package and creates a pending Claim (status `"pending"`,
  `extracted_by` recording the agent or tool identity) through
  `OptimalEngine.MemoryCore.ClaimExtractor`. Review happens through
  `promote_observation/3` and `reject_observation/3`, which delegate the
  decision to `OptimalEngine.MemoryCore.FactPromoter` — the pool never writes
  the facts table directly. Review is still allowed after a pool closes;
  publishing new observations or context into a closed pool is not.

  Pools are shared working memory for multiple humans, agents, and tools, so
  every pool mutation runs its full read-modify-write cycle inside a single
  `OptimalEngine.Store.transaction/1` call (see `mutate_pool/2`): the row is
  re-read fresh, re-validated, and updated atomically, so concurrent
  mutations serialize instead of last-write-winning the whole row.
  """

  alias OptimalEngine.MemoryCore.{
    Claim,
    ClaimExtractor,
    Fact,
    FactPromoter,
    ID,
    JSON,
    RetrievalCoordinator,
    SourcePackage
  }

  alias OptimalEngine.MemoryCore.Store
  alias OptimalEngine.Store, as: BaseStore

  @pool_select_sql """
  SELECT id, tenant_id, workspace_id, pool_scope, task_type, subject_anchor,
         time_mode, loaded_context_links, source_package_links, evidence_links,
         context_package_links, promotion_candidate_links,
         context_confidence_summary, context_precision_summary, member_links,
         agent_links, tool_links, membership_policy, delegation_chain_links,
         lifecycle_state, refresh_state, archive_state, opened_at, closed_at,
         valid_time_start, valid_time_end, transaction_time_start,
         transaction_time_end, stale_after, access_policy_id, security_labels,
         partition_ids, audit_event_links, policy_version, metadata
  FROM active_memory_pools
  WHERE id = ?1
  """

  # Only the columns pool mutations are allowed to change. Scope and
  # membership columns (tenant/workspace, member/agent/tool links, policies)
  # are fixed at open time and never rewritten by mutations.
  @pool_mutate_sql """
  UPDATE active_memory_pools
  SET loaded_context_links = ?2,
      source_package_links = ?3,
      evidence_links = ?4,
      context_package_links = ?5,
      promotion_candidate_links = ?6,
      context_confidence_summary = ?7,
      context_precision_summary = ?8,
      lifecycle_state = ?9,
      refresh_state = ?10,
      archive_state = ?11,
      closed_at = ?12,
      transaction_time_end = ?13,
      metadata = ?14,
      updated_at = datetime('now')
  WHERE id = ?1
  """

  @spec open(keyword()) :: {:ok, map()} | {:error, term()}
  def open(opts \\ []) do
    now = timestamp()
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = string_opt(opts, :workspace_id, "default")
    task_type = string_or_nil(Keyword.get(opts, :task_type))
    subject_anchor = string_or_nil(Keyword.get(opts, :subject_anchor))

    pool = %{
      id:
        ID.content_id("amp", [
          tenant_id,
          ":",
          workspace_id,
          ":",
          task_type || "task",
          ":",
          subject_anchor || "general",
          ":",
          now
        ]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      pool_scope: Keyword.get(opts, :pool_scope, %{}),
      task_type: task_type,
      subject_anchor: subject_anchor,
      time_mode: string_opt(opts, :time_mode, "current_valid"),
      loaded_context_links: [],
      source_package_links: [],
      evidence_links: [],
      context_package_links: [],
      promotion_candidate_links: [],
      context_confidence_summary: %{},
      context_precision_summary: %{},
      member_links: Keyword.get(opts, :member_links, []),
      agent_links: Keyword.get(opts, :agent_links, []),
      tool_links: Keyword.get(opts, :tool_links, []),
      membership_policy: Keyword.get(opts, :membership_policy, %{}),
      delegation_chain_links: Keyword.get(opts, :delegation_chain_links, []),
      lifecycle_state: "open",
      refresh_state: "fresh",
      archive_state: "active",
      opened_at: now,
      closed_at: nil,
      transaction_time_start: now,
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: Keyword.get(opts, :security_labels, []),
      partition_ids: Keyword.get(opts, :partition_ids, []),
      audit_event_links: Keyword.get(opts, :audit_event_links, []),
      policy_version: string_or_nil(Keyword.get(opts, :policy_version)),
      metadata: Keyword.get(opts, :metadata, %{})
    }

    with :ok <- Store.insert_active_memory_pool(pool) do
      {:ok, pool}
    end
  end

  @spec get(String.t()) :: {:ok, map()} | {:error, term()}
  def get(pool_id) when is_binary(pool_id) do
    case BaseStore.raw_query(@pool_select_sql, [pool_id]) do
      {:ok, [row]} -> {:ok, pool_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Load a Context Package into the pool.

  Accepts the map returned by `RetrievalCoordinator.retrieve/2` today and any
  future `ContextPackage` struct (both are maps with the same field names).
  """
  @spec load_context_package(String.t(), map() | struct(), keyword()) ::
          {:ok, map()} | {:error, term()}
  def load_context_package(pool_id, context_package, _opts \\ [])
      when is_binary(pool_id) and is_map(context_package) do
    with {:ok, package_id} <- context_package_id(context_package) do
      package_ref = ref("context_package", package_id)

      mutate_pool(pool_id, fn pool ->
        with :ok <- ensure_open(pool) do
          {:ok,
           pool
           |> put_unique(:context_package_links, package_ref)
           |> put_unique(:loaded_context_links, package_ref)
           |> merge_unique(
             :source_package_links,
             Map.get(context_package, :source_package_links, [])
           )
           |> merge_unique(:evidence_links, Map.get(context_package, :evidence_links, []))
           |> Map.put(
             :context_confidence_summary,
             Map.get(context_package, :package_confidence_summary, %{})
           )
           |> Map.put(
             :context_precision_summary,
             Map.get(context_package, :package_precision_summary, %{})
           )
           |> Map.put(:refresh_state, "fresh")}
        end
      end)
    end
  end

  @spec refresh_context_packages(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def refresh_context_packages(pool_id, opts \\ []) when is_binary(pool_id) and is_list(opts) do
    with {:ok, pool} <- get(pool_id) do
      base_opts = [
        tenant_id: pool.tenant_id,
        workspace_id: pool.workspace_id,
        active_memory_pool_id: pool.id
      ]

      refresh_opts = Keyword.merge(base_opts, opts)

      pool.context_package_links
      |> Enum.reduce_while({:ok, %{refreshed: [], skipped: [], errors: []}}, fn link, {:ok, acc} ->
        case refresh_pool_context_link(pool.id, link, refresh_opts) do
          {:ok, :skipped, package_id} ->
            {:cont, {:ok, put_in(acc.skipped, acc.skipped ++ [package_id])}}

          {:ok, refreshed_package} ->
            {:cont, {:ok, put_in(acc.refreshed, acc.refreshed ++ [refreshed_package])}}

          {:error, reason} ->
            if Keyword.get(opts, :continue_on_error, true) do
              {:cont, {:ok, put_in(acc.errors, acc.errors ++ [%{link: link, reason: reason}])}}
            else
              {:halt, {:error, {link, reason}}}
            end
        end
      end)
      |> case do
        {:ok, result} ->
          with {:ok, refreshed_pool} <- get(pool_id) do
            {:ok,
             %{
               pool: refreshed_pool,
               refreshed_context_packages: result.refreshed,
               skipped_context_package_ids: result.skipped,
               errors: result.errors
             }}
          end

        other ->
          other
      end
    end
  end

  defp refresh_pool_context_link(pool_id, link, refresh_opts) do
    package_id = link_id(link)

    case RetrievalCoordinator.refresh_context_package(package_id, refresh_opts) do
      {:ok, refreshed_package} ->
        with {:ok, _pool} <- load_context_package(pool_id, refreshed_package) do
          {:ok, refreshed_package}
        end

      {:error, :context_package_not_stale} ->
        {:ok, :skipped, package_id}

      other ->
        other
    end
  end

  @doc """
  Publish an observation into the pool.

  The raw observation text is preserved as a Source Package and extracted into
  a pending Claim (`status: "pending"`, `review_status: "unreviewed"`). The
  Claim's `extracted_by` records the publishing identity: `:extracted_by`
  (e.g. `"tool:calendar.read"` for tool output) wins over `:actor_id`.

  `:source_metadata` / `:claim_metadata` let callers stamp lineage (such as a
  governed run ID) onto the preserved evidence and the pending Claim.

  Returns `{:ok, %{pool: pool, source_package: source, pending_claim: claim}}`.
  """
  @spec publish_observation(String.t(), String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def publish_observation(pool_id, observation_text, opts \\ [])
      when is_binary(pool_id) and is_binary(observation_text) do
    with {:ok, pool} <- get(pool_id),
         :ok <- ensure_open(pool) do
      source =
        SourcePackage.from_text(observation_text,
          tenant_id: pool.tenant_id,
          workspace_id: pool.workspace_id,
          source_type: string_opt(opts, :source_type, "pool_observation"),
          source_class: "text",
          source_system: "active_memory_pool",
          source_uri: "optimal://active-memory-pools/#{pool.id}/observations",
          trust_label: string_opt(opts, :trust_label, "unreviewed"),
          access_policy_id: pool.access_policy_id,
          security_labels: pool.security_labels,
          partition_ids: pool.partition_ids,
          created_by: Keyword.get(opts, :actor_id),
          metadata:
            Map.merge(Keyword.get(opts, :source_metadata, %{}), %{
              active_memory_pool_id: pool.id,
              observation_kind: string_opt(opts, :observation_kind, "observation")
            })
        )

      with {:ok, claim} <-
             ClaimExtractor.extract_from_source(source,
               claim_text: string_opt(opts, :claim_text, observation_text),
               claim_type: "observation",
               subject_anchor: Keyword.get(opts, :subject_anchor) || pool.subject_anchor,
               action_class: Keyword.get(opts, :action_class),
               object_anchor: Keyword.get(opts, :object_anchor),
               aggregate_confidence: Keyword.get(opts, :aggregate_confidence, 0.5),
               aggregate_precision: Keyword.get(opts, :aggregate_precision, 0.5),
               actor_id: Keyword.get(opts, :actor_id),
               extracted_by: Keyword.get(opts, :extracted_by),
               evaluator_id: Keyword.get(opts, :evaluator_id),
               metadata:
                 Map.merge(Keyword.get(opts, :claim_metadata, %{}), %{
                   active_memory_pool_id: pool.id
                 })
             ),
           {:ok, updated} <-
             mutate_pool(pool_id, fn fresh ->
               with :ok <- ensure_open(fresh) do
                 {:ok,
                  fresh
                  |> put_unique(:source_package_links, ref("source_package", source.id))
                  |> put_unique(:evidence_links, ref("source_package", source.id))
                  |> put_unique(:promotion_candidate_links, ref("claim", claim.id))
                  |> Map.put(:refresh_state, "dirty")}
               end
             end) do
        {:ok, %{pool: updated, source_package: source, pending_claim: claim}}
      end
    end
  end

  @doc """
  Promote one of the pool's pending observation Claims into a Fact.

  The Claim must be in the pool's `promotion_candidate_links`. The promotion
  decision (reviewer approval via `:verifier_id`/`:actor_id`, or
  `policy: :auto`), the Fact insert, the claim row transition, and the
  derivation ledger entry are all owned by
  `OptimalEngine.MemoryCore.FactPromoter` — this function only resolves the
  pool scope and records the review outcome on the pool.

  Returns `{:ok, %{pool: pool, fact: %Fact{}}}`.
  """
  @spec promote_observation(String.t(), String.t(), keyword()) ::
          {:ok, %{pool: map(), fact: Fact.t()}} | {:error, term()}
  def promote_observation(pool_id, claim_id, opts \\ [])
      when is_binary(pool_id) and is_binary(claim_id) do
    with {:ok, pool} <- get(pool_id),
         :ok <- ensure_promotion_candidate(pool, claim_id),
         {:ok, fact} <-
           FactPromoter.promote(claim_id, Keyword.put(opts, :workspace_id, pool.workspace_id)),
         {:ok, updated} <-
           mutate_pool(pool_id, fn fresh ->
             {:ok,
              fresh
              |> drop_link(:promotion_candidate_links, "claim", claim_id)
              |> put_unique(:evidence_links, ref("fact", fact.id))
              |> Map.put(:refresh_state, "dirty")
              |> record_review_decision(claim_id, "promoted", %{fact_id: fact.id}, opts)}
           end) do
      {:ok, %{pool: updated, fact: fact}}
    end
  end

  @doc """
  Reject one of the pool's pending observation Claims.

  Delegates to `OptimalEngine.MemoryCore.FactPromoter.reject/2` (requires
  `:verifier_id` or `:actor_id`; optional `:reason`), which transitions the
  claim row to `status: "rejected"` and writes the `memory_core.reject_claim`
  ledger entry. No Fact is ever created on this path.

  Returns `{:ok, %{pool: pool, rejected_claim: %Claim{}}}`.
  """
  @spec reject_observation(String.t(), String.t(), keyword()) ::
          {:ok, %{pool: map(), rejected_claim: Claim.t()}} | {:error, term()}
  def reject_observation(pool_id, claim_id, opts \\ [])
      when is_binary(pool_id) and is_binary(claim_id) do
    with {:ok, pool} <- get(pool_id),
         :ok <- ensure_promotion_candidate(pool, claim_id),
         {:ok, claim} <-
           FactPromoter.reject(claim_id, Keyword.put(opts, :workspace_id, pool.workspace_id)),
         {:ok, updated} <-
           mutate_pool(pool_id, fn fresh ->
             {:ok,
              fresh
              |> drop_link(:promotion_candidate_links, "claim", claim_id)
              |> record_review_decision(
                claim_id,
                "rejected",
                %{reason: Keyword.get(opts, :reason)},
                opts
              )}
           end) do
      {:ok, %{pool: updated, rejected_claim: claim}}
    end
  end

  @spec close(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def close(pool_id, opts \\ []) when is_binary(pool_id) do
    now = timestamp()

    mutate_pool(pool_id, fn pool ->
      {:ok,
       pool
       |> Map.put(:lifecycle_state, string_opt(opts, :lifecycle_state, "closed"))
       |> Map.put(:archive_state, string_opt(opts, :archive_state, "archived"))
       |> Map.put(:closed_at, now)
       |> Map.put(:transaction_time_end, now)
       |> Map.put(:metadata, Map.merge(pool.metadata, %{close_reason: Keyword.get(opts, :reason)}))}
    end)
  end

  defp pool_from_row([
         id,
         tenant_id,
         workspace_id,
         pool_scope,
         task_type,
         subject_anchor,
         time_mode,
         loaded_context_links,
         source_package_links,
         evidence_links,
         context_package_links,
         promotion_candidate_links,
         context_confidence_summary,
         context_precision_summary,
         member_links,
         agent_links,
         tool_links,
         membership_policy,
         delegation_chain_links,
         lifecycle_state,
         refresh_state,
         archive_state,
         opened_at,
         closed_at,
         valid_time_start,
         valid_time_end,
         transaction_time_start,
         transaction_time_end,
         stale_after,
         access_policy_id,
         security_labels,
         partition_ids,
         audit_event_links,
         policy_version,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      pool_scope: decode_map(pool_scope),
      task_type: task_type,
      subject_anchor: subject_anchor,
      time_mode: time_mode,
      loaded_context_links: decode_list(loaded_context_links),
      source_package_links: decode_list(source_package_links),
      evidence_links: decode_list(evidence_links),
      context_package_links: decode_list(context_package_links),
      promotion_candidate_links: decode_list(promotion_candidate_links),
      context_confidence_summary: decode_map(context_confidence_summary),
      context_precision_summary: decode_map(context_precision_summary),
      member_links: decode_list(member_links),
      agent_links: decode_list(agent_links),
      tool_links: decode_list(tool_links),
      membership_policy: decode_map(membership_policy),
      delegation_chain_links: decode_list(delegation_chain_links),
      lifecycle_state: lifecycle_state,
      refresh_state: refresh_state,
      archive_state: archive_state,
      opened_at: opened_at,
      closed_at: closed_at,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      transaction_time_start: transaction_time_start,
      transaction_time_end: transaction_time_end,
      stale_after: stale_after,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      audit_event_links: decode_list(audit_event_links),
      policy_version: policy_version,
      metadata: decode_map(metadata)
    }
  end

  # Atomic pool mutation: re-reads the row, applies `mutate_fun`, and writes
  # the mutable columns back, all inside one `OptimalEngine.Store.transaction/1`
  # call (`BEGIN IMMEDIATE` .. `COMMIT` in a single Store GenServer call), so
  # concurrent read-modify-write cycles on the same pool serialize instead of
  # last-write-winning each other's link and metadata updates.
  #
  # `mutate_fun` receives the freshly-read pool and returns `{:ok, updated}`
  # to commit or `{:error, reason}` to roll back. It must be pure (no Store
  # calls): persistence side effects such as Source Package / Claim writes or
  # FactPromoter decisions happen before entering the transaction.
  defp mutate_pool(pool_id, mutate_fun) when is_binary(pool_id) and is_function(mutate_fun, 1) do
    BaseStore.transaction(fn txn ->
      with {:ok, pool} <- txn_get_pool(txn, pool_id),
           {:ok, updated} <- mutate_fun.(pool) do
        txn_update_pool(txn, updated)
      end
    end)
  end

  defp txn_get_pool(txn, pool_id) do
    case BaseStore.txn_query(txn, @pool_select_sql, [pool_id]) do
      {:ok, [row]} -> {:ok, pool_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  defp txn_update_pool(txn, pool) do
    params = [
      pool.id,
      JSON.list(pool.loaded_context_links),
      JSON.list(pool.source_package_links),
      JSON.list(pool.evidence_links),
      JSON.list(pool.context_package_links),
      JSON.list(pool.promotion_candidate_links),
      JSON.map(pool.context_confidence_summary),
      JSON.map(pool.context_precision_summary),
      pool.lifecycle_state,
      pool.refresh_state,
      pool.archive_state,
      pool.closed_at,
      pool.transaction_time_end,
      JSON.map(pool.metadata)
    ]

    case BaseStore.txn_execute(txn, @pool_mutate_sql, params) do
      {:ok, 1} -> {:ok, pool}
      {:ok, 0} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  defp ensure_open(%{lifecycle_state: "open"}), do: :ok
  defp ensure_open(pool), do: {:error, {:pool_not_open, pool.lifecycle_state}}

  defp ensure_promotion_candidate(pool, claim_id) do
    if link?(pool.promotion_candidate_links, "claim", claim_id) do
      :ok
    else
      {:error, :not_a_promotion_candidate}
    end
  end

  defp context_package_id(context_package) do
    case Map.get(context_package, :id) || Map.get(context_package, "id") do
      nil -> {:error, :context_package_id_required}
      id -> {:ok, to_string(id)}
    end
  end

  defp record_review_decision(pool, claim_id, decision, extra, opts) do
    entry =
      %{
        claim_id: claim_id,
        decision: decision,
        actor_id: string_or_nil(Keyword.get(opts, :verifier_id) || Keyword.get(opts, :actor_id)),
        policy: Keyword.get(opts, :policy),
        decided_at: timestamp()
      }
      |> Map.merge(extra)
      |> Map.reject(fn {_key, value} -> is_nil(value) end)

    decisions = decode_list(Map.get(pool.metadata, "review_decisions"))
    Map.put(pool, :metadata, Map.put(pool.metadata, "review_decisions", decisions ++ [entry]))
  end

  defp link?(links, type, id) do
    Enum.any?(links, fn link ->
      link_field(link, :type) == type and link_field(link, :id) == id
    end)
  end

  defp drop_link(pool, key, type, id) do
    Map.update!(pool, key, fn links ->
      Enum.reject(links, fn link ->
        link_field(link, :type) == type and link_field(link, :id) == id
      end)
    end)
  end

  defp link_field(link, key) when is_map(link),
    do: Map.get(link, key) || Map.get(link, to_string(key))

  defp link_field(_link, _key), do: nil

  defp put_unique(pool, key, value), do: merge_unique(pool, key, [value])

  defp merge_unique(pool, key, values) do
    Map.update(pool, key, Enum.uniq(values), fn existing ->
      (existing ++ values) |> Enum.uniq()
    end)
  end

  defp decode_map(nil), do: %{}
  defp decode_map(""), do: %{}

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, map} when is_map(map) -> map
      _ -> %{}
    end
  end

  defp decode_map(value) when is_map(value), do: value
  defp decode_map(_), do: %{}

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, list} when is_list(list) -> Enum.map(list, &normalize_link/1)
      {:ok, other} -> [normalize_link(other)]
      _ -> []
    end
  end

  defp decode_list(value) when is_list(value), do: Enum.map(value, &normalize_link/1)
  defp decode_list(value), do: [normalize_link(value)]

  defp normalize_link(%{"type" => type, "id" => id}), do: %{type: type, id: id}
  defp normalize_link(%{type: _type, id: _id} = link), do: link
  defp normalize_link(value), do: value

  defp link_id(%{id: id}), do: id
  defp link_id(%{"id" => id}), do: id
  defp link_id(id) when is_binary(id), do: id

  defp ref(type, id), do: %{type: type, id: id}

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()

  defp string_opt(opts, key, default), do: Keyword.get(opts, key, default) |> to_string()

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)
end
