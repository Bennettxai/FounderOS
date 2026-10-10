defmodule OptimalEngine.MemoryCore.RetrievalCoordinator do
  @moduledoc """
  Governed recall for Memory Core.

  The gate pipeline runs `query -> Scope Envelope -> authorization envelope ->
  retrieval plan -> Retrieval Package -> Context Package`:

    * `retrieve_package/2` reads accepted Facts and current Memory Objects in a
      tenant + workspace scope with partition/security predicates applied
      inside the candidate SQL (authorization happens during candidate
      expansion, not as a post-filter), and returns the inspectable
      `OptimalEngine.MemoryCore.RetrievalPackage` intermediate. The
      authorization envelope fails closed: an empty/absent envelope excludes
      every security-labeled or partitioned object.
    * `build_context_package/2` projects a Retrieval Package into a persisted
      `OptimalEngine.MemoryCore.ContextPackage` with assembled sections and a
      derivation ledger entry.
    * `retrieve/2` composes both stages and returns the Context Package.

  No function here returns bare ranked chunks.
  """

  alias OptimalEngine.MemoryCore.{
    ContextPackage,
    DerivationLedgerEntry,
    ID,
    RetrievalPackage,
    ScopeEnvelope,
    Store
  }

  alias OptimalEngine.Store, as: BaseStore

  @default_limit 10
  @default_token_budget 2_000
  @max_redacted_links 25
  @policy_version "governed_recall_v1"

  @doc """
  Retrieve governed context for `query`.

  Returns `{:ok, %ContextPackage{}}` — the persisted Context Package with
  hydrated `:facts`/`:memory_objects` and assembled `:sections` — never raw
  search hits.

  The second argument is either a `%ScopeEnvelope{}` or a keyword list:

    * `:scope` — `%ScopeEnvelope{}`; wins over individual scope opts
    * `:workspace_id` / `:tenant_id` / `:actor_id` (or `:requesting_actor_id`)
    * `:allowed_partitions` / `:allowed_security_labels` — authorization
      envelope; fails closed: an empty/absent list only admits objects that
      carry no partition ids / no security labels at all. Callers must present
      an explicit envelope to read partitioned or labeled objects.
    * `:limit` (default #{@default_limit}), `:time_mode` (default
      `"current_valid"`), `:detail_depth`, `:request_id`,
      `:active_memory_pool_id`, `:token_budget` (default
      #{@default_token_budget} tokens for section assembly)
  """
  @spec retrieve(String.t(), keyword() | ScopeEnvelope.t()) ::
          {:ok, ContextPackage.t()} | {:error, term()}
  def retrieve(query, opts \\ [])

  def retrieve(query, %ScopeEnvelope{} = scope), do: retrieve(query, scope: scope)

  def retrieve(query, opts) when is_binary(query) and is_list(opts) do
    with {:ok, retrieval_package} <- retrieve_package(query, opts) do
      build_context_package(retrieval_package, opts)
    end
  end

  @doc """
  Run the retrieval stage only.

  Returns `{:ok, %RetrievalPackage{}}` — the inspectable intermediate with
  authorized candidates, plan, freshness, and filtered-object accounting. Not
  persisted; not a Context Package. Accepts the same options as `retrieve/2`.
  """
  @spec retrieve_package(String.t(), keyword() | ScopeEnvelope.t()) ::
          {:ok, RetrievalPackage.t()} | {:error, term()}
  def retrieve_package(query, opts \\ [])

  def retrieve_package(query, %ScopeEnvelope{} = scope),
    do: retrieve_package(query, scope: scope)

  def retrieve_package(query, opts) when is_binary(query) and is_list(opts) do
    scope = resolve_scope(opts)
    limit = Keyword.get(opts, :limit, @default_limit)
    time_mode = string_opt(opts, :time_mode, "current_valid")
    request_intent = string_opt(opts, :request_intent, classify_intent(query))

    authorization = %{
      allowed_partitions: string_list_opt(opts, :allowed_partitions),
      allowed_security_labels: string_list_opt(opts, :allowed_security_labels),
      allow_all?: Keyword.get(opts, :internal_refresh_allow_all, false)
    }

    retrieval_plan = build_retrieval_plan(query, scope, authorization, opts)

    with {:ok, facts, fact_accounting} <-
           fetch_candidates(:facts, scope, query, limit, authorization, retrieval_plan),
         {:ok, memories, memory_accounting} <-
           fetch_candidates(:memory_objects, scope, query, limit, authorization, retrieval_plan),
         {:ok, asset_extractions, asset_accounting} <-
           fetch_projection_candidates(
             :asset_extractions,
             scope,
             query,
             limit,
             authorization,
             retrieval_plan,
             opts
           ),
         {:ok, workflows, workflow_accounting} <-
           fetch_projection_candidates(
             :workflows,
             scope,
             query,
             limit,
             authorization,
             retrieval_plan,
             opts
           ),
         {:ok, skill_packages, skill_accounting} <-
           fetch_projection_candidates(
             :skill_packages,
             scope,
             query,
             limit,
             authorization,
             retrieval_plan,
             opts
           ) do
      now = timestamp()
      objects = facts ++ memories ++ asset_extractions ++ workflows ++ skill_packages

      redacted_links =
        fact_accounting.redacted_links ++
          memory_accounting.redacted_links ++
          asset_accounting.redacted_links ++
          workflow_accounting.redacted_links ++
          skill_accounting.redacted_links

      policy_excluded =
        fact_accounting.policy_excluded +
          memory_accounting.policy_excluded +
          asset_accounting.policy_excluded +
          workflow_accounting.policy_excluded +
          skill_accounting.policy_excluded

      {:ok,
       %RetrievalPackage{
         id: ID.random_id("rpkg"),
         tenant_id: scope.tenant_id,
         workspace_id: scope.workspace_id,
         query: query,
         request_intent: request_intent,
         time_mode: time_mode,
         limit: limit,
         scope: %{
           actor_id: scope.actor,
           tenant_id: scope.tenant_id,
           workspace_id: scope.workspace_id,
           operation_class: scope.operation_class,
           permissions: scope.permissions,
           allowed_partitions: authorization.allowed_partitions,
           allowed_security_labels: authorization.allowed_security_labels,
           unresolved: scope.unresolved
         },
         retrieval_plan: retrieval_plan,
         facts: facts,
         memory_objects: memories,
         asset_extractions: asset_extractions,
         workflows: workflows,
         skill_packages: skill_packages,
         source_package_links: collect_unique_links(objects, :source_package_links),
         evidence_links: collect_unique_links(objects, :evidence_links),
         redacted_object_links: redacted_links,
         filtered_summary: %{
           candidate_facts: fact_accounting.candidates,
           returned_facts: fact_accounting.returned,
           candidate_memory_objects: memory_accounting.candidates,
           returned_memory_objects: memory_accounting.returned,
           candidate_asset_extractions: asset_accounting.candidates,
           returned_asset_extractions: asset_accounting.returned,
           candidate_workflows: workflow_accounting.candidates,
           returned_workflows: workflow_accounting.returned,
           candidate_skill_packages: skill_accounting.candidates,
           returned_skill_packages: skill_accounting.returned,
           redacted_or_filtered_objects: policy_excluded,
           reason_classes: reason_class_counts(redacted_links)
         },
         confidence_summary: summarize_score(objects, :confidence),
         precision_summary: summarize_score(objects, :precision),
         freshness: %{
           time_mode: time_mode,
           retrieved_at: now,
           refresh_state: "fresh",
           staleness_filter: "stale_superseded_and_expired_excluded"
         },
         retrieved_at: now
       }}
    end
  end

  @doc """
  Project a Retrieval Package into a Context Package.

  Returns `{:ok, %ContextPackage{}}`. Assembles token-budget-aware sections,
  persists the `context_packages` row, and records a
  `memory_core.assemble_context_package` derivation ledger entry (source,
  processor, actor, policy).
  """
  @spec build_context_package(RetrievalPackage.t(), keyword()) ::
          {:ok, ContextPackage.t()} | {:error, term()}
  def build_context_package(%RetrievalPackage{} = retrieval_package, opts \\ []) do
    now = timestamp()

    objects =
      retrieval_package.facts ++
        retrieval_package.memory_objects ++
        retrieval_package.asset_extractions ++
        retrieval_package.workflows ++
        retrieval_package.skill_packages

    fact_links = Enum.map(retrieval_package.facts, &ref("fact", &1.id))
    memory_links = Enum.map(retrieval_package.memory_objects, &ref("memory_object", &1.id))

    asset_extraction_links =
      Enum.map(retrieval_package.asset_extractions, &ref("asset_extraction", &1.id))

    workflow_links = Enum.map(retrieval_package.workflows, &ref("generalized_workflow", &1.id))
    skill_package_links = Enum.map(retrieval_package.skill_packages, &ref("skill_package", &1.id))

    sections =
      assemble_sections(
        retrieval_package,
        Keyword.get(opts, :token_budget, @default_token_budget)
      )

    package =
      ContextPackage.new(%{
        id:
          ID.content_id("ctxpkg", [
            retrieval_package.tenant_id,
            ":",
            retrieval_package.workspace_id,
            ":",
            retrieval_package.query,
            ":",
            now
          ]),
        tenant_id: retrieval_package.tenant_id,
        workspace_id: retrieval_package.workspace_id,
        request_id: string_or_nil(Keyword.get(opts, :request_id)),
        request_intent: retrieval_package.request_intent,
        requesting_actor_id: Map.get(retrieval_package.scope, :actor_id),
        active_memory_pool_id: string_or_nil(Keyword.get(opts, :active_memory_pool_id)),
        time_mode: retrieval_package.time_mode,
        detail_depth: Keyword.get(opts, :detail_depth, 1),
        memory_links: memory_links,
        fact_links: fact_links,
        workflow_links: workflow_links,
        skill_package_links: skill_package_links,
        source_package_links: retrieval_package.source_package_links,
        evidence_links: retrieval_package.evidence_links,
        retrieval_plan: retrieval_package.retrieval_plan,
        package_confidence_summary: retrieval_package.confidence_summary,
        package_precision_summary: retrieval_package.precision_summary,
        filtered_object_summary: retrieval_package.filtered_summary,
        returned_object_links:
          fact_links ++
            memory_links ++ asset_extraction_links ++ workflow_links ++ skill_package_links,
        redacted_object_links: retrieval_package.redacted_object_links,
        authorization_envelope: %{
          actor_id: Map.get(retrieval_package.scope, :actor_id),
          permissions: Map.get(retrieval_package.scope, :permissions, []),
          allowed_partitions: Map.get(retrieval_package.scope, :allowed_partitions, []),
          allowed_security_labels: Map.get(retrieval_package.scope, :allowed_security_labels, []),
          applied_during_candidate_expansion: true,
          applied_before_package_assembly: true
        },
        lifecycle_state: "assembled",
        refresh_state: "fresh",
        refresh_time: now,
        transaction_time_start: now,
        security_labels: merge_lists(objects, :security_labels),
        partition_ids: merge_lists(objects, :partition_ids),
        policy_version: @policy_version,
        metadata:
          Map.merge(Keyword.get(opts, :metadata, %{}), %{
            query: retrieval_package.query,
            answer_surface: "context_package",
            retrieval_package_id: retrieval_package.id,
            asset_extraction_links: asset_extraction_links,
            workflow_links: workflow_links,
            skill_package_links: skill_package_links,
            sections: sections
          }),
        facts: retrieval_package.facts,
        memory_objects: retrieval_package.memory_objects,
        asset_extractions: retrieval_package.asset_extractions,
        workflows: retrieval_package.workflows,
        skill_packages: retrieval_package.skill_packages,
        sections: sections
      })

    with :ok <- Store.insert_context_package(package),
         :ok <- record_assembly(retrieval_package, package, opts) do
      {:ok, package}
    end
  end

  @spec refresh_context_package(String.t(), keyword()) ::
          {:ok, ContextPackage.t()} | {:error, term()}
  def refresh_context_package(context_package_id, opts \\ []) when is_binary(context_package_id) do
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id)

    with {:ok, stale_package} <- get_context_package(context_package_id, tenant_id, workspace_id),
         :ok <- ensure_refreshable(stale_package, opts),
         {:ok, query} <- original_query(stale_package),
         {:ok, refreshed_package} <-
           retrieve(query, refresh_opts_from_package(stale_package, opts)),
         :ok <-
           Store.mark_context_package_refreshed(stale_package.id,
             tenant_id: stale_package.tenant_id,
             workspace_id: stale_package.workspace_id
           ) do
      {:ok, refreshed_package}
    end
  end

  @spec refresh_stale_context_packages(keyword()) :: {:ok, map()} | {:error, term()}
  def refresh_stale_context_packages(opts \\ []) when is_list(opts) do
    with {:ok, stale_packages} <- list_stale_context_packages(opts) do
      refresh_opts = Keyword.drop(opts, [:batch_limit, :continue_on_error])

      stale_packages
      |> Enum.reduce_while(
        {:ok,
         %{stale_context_package_ids: Enum.map(stale_packages, & &1.id), refreshed: [], errors: []}},
        fn stale_package, {:ok, acc} ->
          case refresh_context_package(stale_package.id, refresh_opts) do
            {:ok, refreshed_package} ->
              {:cont, {:ok, put_in(acc.refreshed, acc.refreshed ++ [refreshed_package])}}

            {:error, reason} ->
              if Keyword.get(opts, :continue_on_error, true) do
                error = %{context_package_id: stale_package.id, reason: reason}
                {:cont, {:ok, put_in(acc.errors, acc.errors ++ [error])}}
              else
                {:halt, {:error, {stale_package.id, reason}}}
              end
          end
        end
      )
      |> case do
        {:ok, result} ->
          {:ok,
           %{
             stale_context_package_ids: result.stale_context_package_ids,
             refreshed_context_packages: result.refreshed,
             errors: result.errors
           }}

        other ->
          other
      end
    end
  end

  defp list_stale_context_packages(opts) do
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id)
    batch_limit = Keyword.get(opts, :batch_limit, 50)

    {workspace_clause, params} =
      case workspace_id do
        nil -> {"", [tenant_id]}
        value -> {" AND workspace_id = ?2", [tenant_id, value]}
      end

    sql = """
    #{context_package_select_sql()}
    WHERE tenant_id = ?1
      AND refresh_state = 'stale'
      #{workspace_clause}
    ORDER BY updated_at DESC
    LIMIT ?#{length(params) + 1}
    """

    case BaseStore.raw_query(sql, params ++ [batch_limit]) do
      {:ok, rows} -> {:ok, Enum.map(rows, &context_package_from_row/1)}
      other -> other
    end
  end

  defp get_context_package(context_package_id, tenant_id, workspace_id) do
    {workspace_clause, params} =
      case workspace_id do
        nil -> {"", [tenant_id, context_package_id]}
        value -> {" AND workspace_id = ?3", [tenant_id, context_package_id, value]}
      end

    sql = """
    #{context_package_select_sql()}
    WHERE tenant_id = ?1
      AND id = ?2
      #{workspace_clause}
    LIMIT 1
    """

    case BaseStore.raw_query(sql, params) do
      {:ok, [row]} -> {:ok, context_package_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  defp ensure_refreshable(%{refresh_state: "stale"}, _opts), do: :ok

  defp ensure_refreshable(_package, opts) do
    if Keyword.get(opts, :force, false), do: :ok, else: {:error, :context_package_not_stale}
  end

  defp original_query(package) do
    query =
      map_get(package.metadata, :query) ||
        map_get(package.retrieval_plan, :query) ||
        ""

    case query do
      "" -> {:error, :missing_original_query}
      query when is_binary(query) -> {:ok, query}
      query -> {:ok, to_string(query)}
    end
  end

  defp refresh_opts_from_package(package, opts) do
    auth = package.authorization_envelope

    [
      tenant_id: Keyword.get(opts, :tenant_id, package.tenant_id),
      workspace_id: Keyword.get(opts, :workspace_id, package.workspace_id),
      request_id: Keyword.get(opts, :request_id, package.request_id),
      request_intent: Keyword.get(opts, :request_intent, package.request_intent),
      actor_id:
        Keyword.get(opts, :actor_id) ||
          Keyword.get(opts, :requesting_actor_id) ||
          package.requesting_actor_id,
      active_memory_pool_id:
        Keyword.get(opts, :active_memory_pool_id, package.active_memory_pool_id),
      time_mode: Keyword.get(opts, :time_mode, package.time_mode),
      detail_depth: Keyword.get(opts, :detail_depth, package.detail_depth),
      allowed_partitions:
        Keyword.get(opts, :allowed_partitions) ||
          map_get(auth, :allowed_partitions) ||
          package.partition_ids,
      allowed_security_labels:
        Keyword.get(opts, :allowed_security_labels) ||
          map_get(auth, :allowed_security_labels) ||
          package.security_labels,
      limit: Keyword.get(opts, :limit) || map_get(package.retrieval_plan, :limit) || @default_limit,
      metadata:
        Map.merge(Keyword.get(opts, :metadata, %{}), %{
          refreshed_from_context_package_id: package.id,
          refreshed_from_invalidation_reason: package.invalidation_reason
        })
    ]
    |> Keyword.put(:internal_refresh_allow_all, true)
  end

  # ---------------------------------------------------------------------------
  # Private: scope resolution
  # ---------------------------------------------------------------------------

  defp resolve_scope(opts) do
    case Keyword.get(opts, :scope) do
      %ScopeEnvelope{} = scope ->
        scope

      _ ->
        opts
        |> normalize_actor_opt()
        |> Keyword.put_new(:operation_class, "recall.retrieve")
        |> ScopeEnvelope.resolve()
    end
  end

  defp normalize_actor_opt(opts) do
    case Keyword.get(opts, :requesting_actor_id) do
      nil -> opts
      actor -> Keyword.put_new(opts, :actor_id, actor)
    end
  end

  # ---------------------------------------------------------------------------
  # Private: candidate expansion (authorization inside the SQL)
  # ---------------------------------------------------------------------------

  defp fetch_candidates(kind, %ScopeEnvelope{} = scope, query, limit, authorization, retrieval_plan) do
    parts = sql_parts(kind)
    pattern = "%#{query}%"
    filters = Map.get(retrieval_plan, :structured_filters, %{})

    base_params = [
      scope.tenant_id,
      scope.workspace_id,
      pattern,
      query,
      Map.get(filters, :subject_anchor),
      Map.get(filters, :action_class),
      Map.get(filters, :object_anchor)
    ]

    where = base_where(parts)

    {auth_clauses, auth_params} = authorization_clauses(parts.table, authorization, 7)
    limit_placeholder = "?#{8 + length(auth_params)}"

    authorized_sql = """
    #{parts.select}
    FROM #{parts.table}
    #{where}#{and_all(auth_clauses)}
    #{parts.order}
    LIMIT #{limit_placeholder}
    """

    authorized_count_sql = """
    SELECT COUNT(*) FROM #{parts.table}
    #{where}#{and_all(auth_clauses)}
    """

    with {:ok, rows} <-
           BaseStore.raw_query(authorized_sql, base_params ++ auth_params ++ [limit]),
         {:ok, [[authorized_count]]} <-
           BaseStore.raw_query(authorized_count_sql, base_params ++ auth_params),
         {:ok, redacted_links, excluded_count} <-
           fetch_excluded(parts, where, auth_clauses, base_params, auth_params, authorization) do
      objects = Enum.map(rows, &object_from_row(kind, &1))

      {:ok, objects,
       %{
         candidates: authorized_count + excluded_count,
         returned: length(objects),
         policy_excluded: excluded_count,
         redacted_links: redacted_links
       }}
    end
  end

  # The authorization envelope always yields both predicates (fail-closed),
  # so the excluded-object accounting runs on every retrieval.
  defp fetch_excluded(parts, where, auth_clauses, base_params, auth_params, authorization) do
    excluded_where = "#{where}  AND NOT (#{Enum.join(auth_clauses, " AND ")})\n"

    excluded_sql = """
    SELECT id, security_labels, partition_ids FROM #{parts.table}
    #{excluded_where}
    LIMIT #{@max_redacted_links}
    """

    excluded_count_sql = """
    SELECT COUNT(*) FROM #{parts.table}
    #{excluded_where}
    """

    with {:ok, rows} <- BaseStore.raw_query(excluded_sql, base_params ++ auth_params),
         {:ok, [[count]]} <-
           BaseStore.raw_query(excluded_count_sql, base_params ++ auth_params) do
      {:ok, Enum.map(rows, &redacted_ref(parts.object_type, &1, authorization)), count}
    end
  end

  defp fetch_projection_candidates(
         kind,
         %ScopeEnvelope{} = scope,
         query,
         limit,
         authorization,
         retrieval_plan,
         opts
       ) do
    with {:ok, candidates} <-
           fetch_projection_rows(kind, scope, query, limit * 2, retrieval_plan, opts) do
      {authorized, redacted} =
        Enum.split_with(candidates, &authorized_projection?(&1, authorization))

      returned = Enum.take(authorized, limit)

      {:ok, returned,
       %{
         candidates: length(candidates),
         returned: length(returned),
         policy_excluded: length(redacted),
         redacted_links:
           redacted
           |> Enum.take(@max_redacted_links)
           |> Enum.map(&redacted_projection_ref(kind, &1, authorization))
       }}
    end
  end

  defp fetch_projection_rows(:asset_extractions, scope, query, limit, retrieval_plan, _opts) do
    pattern = "%#{query}%"
    filters = Map.get(retrieval_plan, :asset_filters, %{})

    sql = """
    SELECT id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
           extraction_type, modality, content_text, content_ref, content_hash,
           confidence, precision, security_labels, partition_ids, metadata,
           derivation_ledger_id, created_by, created_at
    FROM asset_extractions
    WHERE tenant_id = ?1
      AND workspace_id = ?2
      AND (
        content_text LIKE ?3 OR content_ref LIKE ?3 OR extraction_type LIKE ?3
        OR modality LIKE ?3 OR ?4 = ''
      )
      AND (?5 IS NULL OR modality = ?5)
      AND (?6 IS NULL OR extraction_type = ?6)
    ORDER BY confidence DESC, created_at DESC
    LIMIT ?7
    """

    params = [
      scope.tenant_id,
      scope.workspace_id,
      pattern,
      query,
      Map.get(filters, :modality),
      Map.get(filters, :extraction_type),
      limit
    ]

    with {:ok, rows} <- BaseStore.raw_query(sql, params) do
      {:ok, Enum.map(rows, &asset_extraction_from_row/1)}
    end
  end

  defp fetch_projection_rows(:workflows, scope, query, limit, retrieval_plan, _opts) do
    pattern = "%#{query}%"
    filters = Map.get(retrieval_plan, :workflow_filters, %{})

    sql = """
    SELECT id, tenant_id, workspace_id, workflow_family, scope,
           applicability_conditions, outcome_class, workflow_trace_links,
           supporting_episode_links, contradicting_trace_links, step_pattern_links,
           aggregate_confidence, aggregate_precision, validation_score,
           lifecycle_state, validation_status, supersession_status,
           valid_time_start, valid_time_end, stale_after, access_policy_id,
           security_labels, partition_ids, metadata
    FROM generalized_workflows
    WHERE tenant_id = ?1
      AND workspace_id = ?2
      AND lifecycle_state IN ('candidate', 'current', 'approved')
      AND validation_status != 'rejected'
      AND supersession_status = 'none'
      AND (valid_time_end IS NULL OR datetime(valid_time_end) >= datetime('now'))
      AND (stale_after IS NULL OR datetime(stale_after) >= datetime('now'))
      AND (
        workflow_family LIKE ?3 OR outcome_class LIKE ?3 OR metadata LIKE ?3
        OR ?4 = ''
      )
      AND (?5 IS NULL OR workflow_family = ?5)
    ORDER BY validation_score DESC, aggregate_confidence DESC, updated_at DESC
    LIMIT ?6
    """

    params = [
      scope.tenant_id,
      scope.workspace_id,
      pattern,
      query,
      Map.get(filters, :workflow_family),
      limit
    ]

    with {:ok, rows} <- BaseStore.raw_query(sql, params) do
      {:ok, Enum.map(rows, &workflow_from_row/1)}
    end
  end

  defp fetch_projection_rows(:skill_packages, scope, query, limit, retrieval_plan, opts) do
    pattern = "%#{query}%"
    filters = Map.get(retrieval_plan, :skill_filters, %{})

    eligibility_sql =
      if Keyword.get(opts, :include_draft_skills, false) do
        "AND retirement_status = 'active'"
      else
        """
        AND review_status = 'approved'
        AND enabled_state = 'enabled'
        AND retirement_status = 'active'
        AND suspension_reason IS NULL
        """
      end

    sql = """
    SELECT id, tenant_id, workspace_id, version, skill_package_name, task_family,
           competency_links, risk_class, procedural_memory_links, workflow_links,
           validation_links, evidence_links, input_contract, output_contract,
           execution_policy, required_privileges, tool_requirements, model_policy_id,
           aggregate_confidence, aggregate_precision, review_status, enabled_state,
           suspension_reason, retirement_status, valid_time_start, valid_time_end,
           stale_after, access_policy_id, security_labels, partition_ids, metadata
    FROM skill_packages
    WHERE tenant_id = ?1
      AND workspace_id = ?2
      #{eligibility_sql}
      AND (valid_time_end IS NULL OR datetime(valid_time_end) >= datetime('now'))
      AND (stale_after IS NULL OR datetime(stale_after) >= datetime('now'))
      AND (
        skill_package_name LIKE ?3 OR task_family LIKE ?3 OR metadata LIKE ?3
        OR ?4 = ''
      )
      AND (?5 IS NULL OR task_family = ?5)
      AND (?6 IS NULL OR skill_package_name = ?6)
    ORDER BY aggregate_confidence DESC, updated_at DESC
    LIMIT ?7
    """

    params = [
      scope.tenant_id,
      scope.workspace_id,
      pattern,
      query,
      Map.get(filters, :task_family),
      Map.get(filters, :skill_package_name),
      limit
    ]

    with {:ok, rows} <- BaseStore.raw_query(sql, params) do
      {:ok, Enum.map(rows, &skill_package_from_row/1)}
    end
  end

  # Tenant + workspace predicates together form the scope key: rows from
  # another tenant that share a workspace id (e.g. the literal "default")
  # never enter the candidate set.
  defp base_where(parts) do
    """
    WHERE tenant_id = ?1
      AND workspace_id = ?2
    #{parts.lifecycle_filter}  AND (valid_time_end IS NULL OR datetime(valid_time_end) >= datetime('now'))
      AND (stale_after IS NULL OR datetime(stale_after) >= datetime('now'))
    #{parts.match_filter}
    #{parts.structured_filter}
    """
  end

  defp and_all(clauses), do: Enum.map_join(clauses, fn clause -> "  AND #{clause}\n" end)

  # Partition access requires intersection with the allowed partitions;
  # security access requires the object's labels to be a subset of the allowed
  # labels. The envelope fails closed: an empty/absent allowed list only
  # admits objects that carry no partition ids / no security labels at all —
  # it never grants access to partitioned or labeled objects. Predicates run
  # inside the candidate SQL so unauthorized rows never enter (or crowd out)
  # the candidate window.
  defp authorization_clauses(table, authorization, param_offset) do
    if Map.get(authorization, :allow_all?, false) do
      {["1 = 1", "1 = 1"], []}
    else
      authorization_clauses_scoped(table, authorization, param_offset)
    end
  end

  defp authorization_clauses_scoped(table, authorization, param_offset) do
    {partition_clause, partition_params} =
      case authorization.allowed_partitions do
        [] ->
          {"NOT EXISTS (SELECT 1 FROM json_each(COALESCE(NULLIF(#{table}.partition_ids, ''), '[]')))",
           []}

        partitions ->
          marks = placeholders(param_offset + 1, length(partitions))

          {"EXISTS (SELECT 1 FROM json_each(COALESCE(NULLIF(#{table}.partition_ids, ''), '[]')) AS object_partition WHERE object_partition.value IN (#{marks}))",
           partitions}
      end

    security_offset = param_offset + length(partition_params)

    {security_clause, security_params} =
      case authorization.allowed_security_labels do
        [] ->
          {"NOT EXISTS (SELECT 1 FROM json_each(COALESCE(NULLIF(#{table}.security_labels, ''), '[]')))",
           []}

        labels ->
          marks = placeholders(security_offset + 1, length(labels))

          {"NOT EXISTS (SELECT 1 FROM json_each(COALESCE(NULLIF(#{table}.security_labels, ''), '[]')) AS object_label WHERE object_label.value NOT IN (#{marks}))",
           labels}
      end

    {[partition_clause, security_clause], partition_params ++ security_params}
  end

  defp placeholders(start, count) do
    start..(start + count - 1)
    |> Enum.map_join(", ", &"?#{&1}")
  end

  defp sql_parts(:facts) do
    %{
      table: "facts",
      object_type: "fact",
      select: """
      SELECT id, tenant_id, workspace_id, fact_text, fact_type, subject_anchor,
             action_class, object_anchor, accepted_claim_ids, supporting_evidence_links,
             aggregate_confidence, aggregate_precision, access_policy_id,
             security_labels, partition_ids, lifecycle_state, contradiction_status, valid_time_start,
             valid_time_end, stale_after
      """,
      lifecycle_filter: """
        AND lifecycle_state = 'accepted'
        AND contradiction_status = 'none'
      """,
      match_filter: """
        AND (
          fact_text LIKE ?3 OR subject_anchor LIKE ?3 OR action_class LIKE ?3
          OR object_anchor LIKE ?3 OR ?4 = ''
        )
      """,
      structured_filter: """
        AND (?5 IS NULL OR subject_anchor = ?5)
        AND (?6 IS NULL OR action_class = ?6)
        AND (?7 IS NULL OR object_anchor = ?7)
      """,
      order: "ORDER BY aggregate_confidence DESC, updated_at DESC"
    }
  end

  defp sql_parts(:memory_objects) do
    %{
      table: "memory_objects",
      object_type: "memory_object",
      select: """
      SELECT id, tenant_id, workspace_id, memory_type, summary, subject_anchor,
             action_class, fact_links, claim_links, source_package_links, evidence_links,
             aggregate_confidence, aggregate_precision, access_policy_id,
             security_labels, partition_ids, lifecycle_state, staleness_status,
             supersession_status, valid_time_start, valid_time_end, stale_after
      """,
      lifecycle_filter: """
        AND lifecycle_state = 'current'
        AND staleness_status = 'current'
        AND supersession_status = 'none'
      """,
      match_filter: """
        AND (
          summary LIKE ?3 OR subject_anchor LIKE ?3 OR action_class LIKE ?3
          OR ?4 = ''
        )
      """,
      structured_filter: """
        AND (?5 IS NULL OR subject_anchor = ?5)
        AND (?6 IS NULL OR action_class = ?6)
        AND (?7 IS NULL OR ?7 IS NOT NULL)
      """,
      order: "ORDER BY salience DESC, aggregate_confidence DESC, updated_at DESC"
    }
  end

  defp object_from_row(:facts, row), do: fact_from_row(row)
  defp object_from_row(:memory_objects, row), do: memory_from_row(row)

  defp fact_from_row([
         id,
         tenant_id,
         workspace_id,
         fact_text,
         fact_type,
         subject_anchor,
         action_class,
         object_anchor,
         accepted_claim_ids,
         supporting_evidence_links,
         aggregate_confidence,
         aggregate_precision,
         access_policy_id,
         security_labels,
         partition_ids,
         lifecycle_state,
         contradiction_status,
         valid_time_start,
         valid_time_end,
         stale_after
       ]) do
    source_links =
      supporting_evidence_links
      |> decode_list()
      |> Enum.filter(&link_type?(&1, "source_package"))

    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      text: fact_text,
      type: fact_type,
      subject_anchor: subject_anchor,
      action_class: action_class,
      object_anchor: object_anchor,
      accepted_claim_ids: decode_list(accepted_claim_ids),
      evidence_links: decode_list(supporting_evidence_links),
      source_package_links: source_links,
      confidence: aggregate_confidence,
      precision: aggregate_precision,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      lifecycle_state: lifecycle_state,
      contradiction_status: contradiction_status,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      stale_after: stale_after
    }
  end

  defp memory_from_row([
         id,
         tenant_id,
         workspace_id,
         memory_type,
         summary,
         subject_anchor,
         action_class,
         fact_links,
         claim_links,
         source_package_links,
         evidence_links,
         aggregate_confidence,
         aggregate_precision,
         access_policy_id,
         security_labels,
         partition_ids,
         lifecycle_state,
         staleness_status,
         supersession_status,
         valid_time_start,
         valid_time_end,
         stale_after
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      text: summary,
      type: memory_type,
      subject_anchor: subject_anchor,
      action_class: action_class,
      fact_links: decode_list(fact_links),
      claim_links: decode_list(claim_links),
      source_package_links: decode_list(source_package_links),
      evidence_links: decode_list(evidence_links),
      confidence: aggregate_confidence,
      precision: aggregate_precision,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      lifecycle_state: lifecycle_state,
      staleness_status: staleness_status,
      supersession_status: supersession_status,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      stale_after: stale_after
    }
  end

  defp asset_extraction_from_row([
         id,
         tenant_id,
         workspace_id,
         asset_id,
         source_package_id,
         adapter_run_id,
         extraction_type,
         modality,
         content_text,
         content_ref,
         content_hash,
         confidence,
         precision,
         security_labels,
         partition_ids,
         metadata,
         derivation_ledger_id,
         created_by,
         created_at
       ]) do
    source_links =
      if is_binary(source_package_id) and source_package_id != "" do
        [ref("source_package", source_package_id)]
      else
        []
      end

    evidence_links =
      [
        ref("asset", asset_id),
        ref("asset_adapter_run", adapter_run_id),
        ref("asset_extraction", id)
        | source_links
      ]
      |> Enum.reject(&(is_nil(&1.id) or &1.id == ""))

    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      text: if(content_text in [nil, ""], do: content_ref, else: content_text),
      type: extraction_type,
      asset_id: asset_id,
      source_package_id: source_package_id,
      adapter_run_id: adapter_run_id,
      extraction_type: extraction_type,
      modality: modality,
      content_text: content_text,
      content_ref: content_ref,
      content_hash: content_hash,
      source_package_links: source_links,
      evidence_links: evidence_links,
      confidence: confidence || 0.0,
      precision: precision || 0.0,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      metadata: decode_map(metadata),
      derivation_ledger_id: derivation_ledger_id,
      created_by: created_by,
      created_at: created_at
    }
  end

  defp workflow_from_row([
         id,
         tenant_id,
         workspace_id,
         workflow_family,
         scope,
         applicability_conditions,
         outcome_class,
         workflow_trace_links,
         supporting_episode_links,
         contradicting_trace_links,
         step_pattern_links,
         aggregate_confidence,
         aggregate_precision,
         validation_score,
         lifecycle_state,
         validation_status,
         supersession_status,
         valid_time_start,
         valid_time_end,
         stale_after,
         access_policy_id,
         security_labels,
         partition_ids,
         metadata
       ]) do
    workflow_trace_links = decode_list(workflow_trace_links)
    supporting_episode_links = decode_list(supporting_episode_links)
    contradicting_trace_links = decode_list(contradicting_trace_links)

    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      text: workflow_family,
      type: "generalized_workflow",
      workflow_family: workflow_family,
      scope: decode_map(scope),
      applicability_conditions: decode_map(applicability_conditions),
      outcome_class: outcome_class,
      workflow_trace_links: workflow_trace_links,
      supporting_episode_links: supporting_episode_links,
      contradicting_trace_links: contradicting_trace_links,
      step_pattern_links: decode_list(step_pattern_links),
      source_package_links: [],
      evidence_links: workflow_trace_links ++ supporting_episode_links ++ contradicting_trace_links,
      confidence: aggregate_confidence || 0.0,
      precision: aggregate_precision || 0.0,
      validation_score: validation_score,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      lifecycle_state: lifecycle_state,
      validation_status: validation_status,
      supersession_status: supersession_status,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      stale_after: stale_after,
      metadata: decode_map(metadata)
    }
  end

  defp skill_package_from_row([
         id,
         tenant_id,
         workspace_id,
         version,
         skill_package_name,
         task_family,
         competency_links,
         risk_class,
         procedural_memory_links,
         workflow_links,
         validation_links,
         evidence_links,
         input_contract,
         output_contract,
         execution_policy,
         required_privileges,
         tool_requirements,
         model_policy_id,
         aggregate_confidence,
         aggregate_precision,
         review_status,
         enabled_state,
         suspension_reason,
         retirement_status,
         valid_time_start,
         valid_time_end,
         stale_after,
         access_policy_id,
         security_labels,
         partition_ids,
         metadata
       ]) do
    procedural_memory_links = decode_list(procedural_memory_links)
    workflow_links = decode_list(workflow_links)
    validation_links = decode_list(validation_links)
    evidence_links = decode_list(evidence_links)

    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      text: skill_package_name,
      type: "skill_package",
      version: version,
      skill_package_name: skill_package_name,
      task_family: task_family,
      competency_links: decode_list(competency_links),
      risk_class: risk_class,
      procedural_memory_links: procedural_memory_links,
      workflow_links: workflow_links,
      validation_links: validation_links,
      source_package_links: [],
      evidence_links:
        evidence_links ++ workflow_links ++ procedural_memory_links ++ validation_links,
      input_contract: decode_map(input_contract),
      output_contract: decode_map(output_contract),
      execution_policy: decode_map(execution_policy),
      required_privileges: decode_list(required_privileges),
      tool_requirements: decode_list(tool_requirements),
      model_policy_id: model_policy_id,
      confidence: aggregate_confidence || 0.0,
      precision: aggregate_precision || 0.0,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      review_status: review_status,
      enabled_state: enabled_state,
      suspension_reason: suspension_reason,
      retirement_status: retirement_status,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      stale_after: stale_after,
      metadata: decode_map(metadata)
    }
  end

  defp context_package_select_sql do
    """
    SELECT id, tenant_id, workspace_id, request_id, request_intent,
           requesting_actor_id, active_memory_pool_id, time_mode, detail_depth,
           memory_links, fact_links, workflow_links, skill_package_links,
           source_package_links, evidence_links, retrieval_plan,
           package_confidence_summary, package_precision_summary,
           filtered_object_summary, returned_object_links, redacted_object_links,
           authorization_envelope, lifecycle_state, refresh_state,
           invalidation_reason, valid_time_start, valid_time_end,
           transaction_time_start, transaction_time_end, stale_after,
           refresh_time, access_policy_id, security_labels, partition_ids,
           audit_event_links, policy_version, metadata
    FROM context_packages
    """
  end

  defp context_package_from_row([
         id,
         tenant_id,
         workspace_id,
         request_id,
         request_intent,
         requesting_actor_id,
         active_memory_pool_id,
         time_mode,
         detail_depth,
         memory_links,
         fact_links,
         workflow_links,
         skill_package_links,
         source_package_links,
         evidence_links,
         retrieval_plan,
         package_confidence_summary,
         package_precision_summary,
         filtered_object_summary,
         returned_object_links,
         redacted_object_links,
         authorization_envelope,
         lifecycle_state,
         refresh_state,
         invalidation_reason,
         valid_time_start,
         valid_time_end,
         transaction_time_start,
         transaction_time_end,
         stale_after,
         refresh_time,
         access_policy_id,
         security_labels,
         partition_ids,
         audit_event_links,
         policy_version,
         metadata
       ]) do
    ContextPackage.new(%{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      request_id: request_id,
      request_intent: request_intent,
      requesting_actor_id: requesting_actor_id,
      active_memory_pool_id: active_memory_pool_id,
      time_mode: time_mode,
      detail_depth: detail_depth,
      memory_links: decode_list(memory_links),
      fact_links: decode_list(fact_links),
      workflow_links: decode_list(workflow_links),
      skill_package_links: decode_list(skill_package_links),
      source_package_links: decode_list(source_package_links),
      evidence_links: decode_list(evidence_links),
      retrieval_plan: decode_map(retrieval_plan),
      package_confidence_summary: decode_map(package_confidence_summary),
      package_precision_summary: decode_map(package_precision_summary),
      filtered_object_summary: decode_map(filtered_object_summary),
      returned_object_links: decode_list(returned_object_links),
      redacted_object_links: decode_list(redacted_object_links),
      authorization_envelope: decode_map(authorization_envelope),
      lifecycle_state: lifecycle_state,
      refresh_state: refresh_state,
      invalidation_reason: invalidation_reason,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      transaction_time_start: transaction_time_start,
      transaction_time_end: transaction_time_end,
      stale_after: stale_after,
      refresh_time: refresh_time,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      audit_event_links: decode_list(audit_event_links),
      policy_version: policy_version,
      metadata: decode_map(metadata)
    })
  end

  # ---------------------------------------------------------------------------
  # Private: filtered-object accounting
  # ---------------------------------------------------------------------------

  defp redacted_ref(object_type, [id, security_labels, partition_ids], authorization) do
    object = %{
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids)
    }

    reasons =
      []
      |> add_reason(
        "partition_filtered",
        not partition_allowed?(object, authorization.allowed_partitions)
      )
      |> add_reason(
        "security_label_filtered",
        not security_allowed?(object, authorization.allowed_security_labels)
      )
      |> Enum.reverse()

    %{type: object_type, id: id, reasons: reasons}
  end

  defp add_reason(reasons, reason, true), do: [reason | reasons]
  defp add_reason(reasons, _reason, false), do: reasons

  # Mirrors the fail-closed SQL predicates: an empty envelope only allows
  # objects with no partitions / no labels.
  defp partition_allowed?(object, []), do: object.partition_ids == []

  defp partition_allowed?(object, allowed_partitions) do
    MapSet.intersection(MapSet.new(object.partition_ids), MapSet.new(allowed_partitions))
    |> MapSet.size()
    |> Kernel.>(0)
  end

  defp security_allowed?(object, []), do: object.security_labels == []

  defp security_allowed?(object, allowed_security_labels) do
    MapSet.subset?(MapSet.new(object.security_labels), MapSet.new(allowed_security_labels))
  end

  defp reason_class_counts(redacted_links) do
    redacted_links
    |> Enum.flat_map(& &1.reasons)
    |> Enum.frequencies()
  end

  # ---------------------------------------------------------------------------
  # Private: section assembly (token-budget-aware tiering)
  # ---------------------------------------------------------------------------

  defp assemble_sections(retrieval_package, token_budget) do
    summary = summary_section(retrieval_package)
    remaining = max(token_budget - estimate_tokens(summary), 0)
    fact_budget = div(remaining * 6, 10)

    {facts_section, fact_tokens, facts_truncated?} =
      object_section("## Facts", retrieval_package.facts, fact_budget, &fact_entry/1)

    {memory_section, memory_tokens, memory_truncated?} =
      object_section(
        "## Memory",
        retrieval_package.memory_objects,
        remaining - fact_tokens,
        &memory_entry/1
      )

    %{
      summary: summary,
      facts: facts_section,
      memory: memory_section,
      token_budget: token_budget,
      total_tokens: estimate_tokens(summary) + fact_tokens + memory_tokens,
      truncated?: facts_truncated? or memory_truncated?
    }
  end

  defp summary_section(retrieval_package) do
    filtered = retrieval_package.filtered_summary

    """
    # Context Package
    Query: #{retrieval_package.query}
    Workspace: #{retrieval_package.workspace_id}
    Facts: #{filtered.returned_facts} returned of #{filtered.candidate_facts} candidates
    Memory objects: #{filtered.returned_memory_objects} returned of #{filtered.candidate_memory_objects} candidates
    Policy-excluded objects: #{filtered.redacted_or_filtered_objects}
    Freshness: #{retrieval_package.time_mode} as of #{retrieval_package.retrieved_at}
    Actor: #{Map.get(retrieval_package.scope, :actor_id) || "unscoped"}
    """
  end

  defp object_section(_header, [], _budget, _entry_fun), do: {"", 0, false}

  defp object_section(header, objects, budget, entry_fun) do
    opening = header <> "\n"

    {body, tokens, kept} =
      Enum.reduce(objects, {"", estimate_tokens(opening), 0}, fn object, {acc, acc_tokens, kept} ->
        entry = entry_fun.(object)
        entry_tokens = estimate_tokens(entry)

        if acc_tokens + entry_tokens <= budget do
          {acc <> entry, acc_tokens + entry_tokens, kept + 1}
        else
          {acc, acc_tokens, kept}
        end
      end)

    {opening <> body, tokens, kept < length(objects)}
  end

  defp fact_entry(fact) do
    "- #{fact.text} (confidence: #{fact.confidence})#{citations(fact.source_package_links)}\n"
  end

  defp memory_entry(memory) do
    fact_ids = Enum.map(memory.fact_links, &link_id/1) |> Enum.reject(&is_nil/1)
    fact_note = if fact_ids == [], do: "", else: " [facts: #{Enum.join(fact_ids, ", ")}]"
    "- #{memory.text}#{fact_note}#{citations(memory.source_package_links)}\n"
  end

  defp citations([]), do: ""

  defp citations(source_package_links) do
    ids = source_package_links |> Enum.map(&link_id/1) |> Enum.reject(&is_nil/1)
    if ids == [], do: "", else: " [sources: #{Enum.join(ids, ", ")}]"
  end

  defp estimate_tokens(text) when is_binary(text), do: div(String.length(text), 4)
  defp estimate_tokens(_), do: 0

  # ---------------------------------------------------------------------------
  # Private: derivation ledger
  # ---------------------------------------------------------------------------

  defp record_assembly(retrieval_package, package, opts) do
    entry =
      DerivationLedgerEntry.new(
        "memory_core.assemble_context_package",
        "retrieval_package_to_context_package",
        [
          %{type: "retrieval_package", id: retrieval_package.id}
          | package.returned_object_links
        ],
        [%{type: "context_package", id: package.id}],
        tenant_id: package.tenant_id,
        workspace_id: package.workspace_id,
        actor_id:
          Map.get(retrieval_package.scope, :actor_id) ||
            string_or_nil(Keyword.get(opts, :actor_id)),
        parser_id: "optimal_engine.memory_core.retrieval_coordinator",
        source_package_links: package.source_package_links,
        evidence_links: package.evidence_links,
        security_labels: package.security_labels,
        partition_ids: package.partition_ids,
        policy_version: @policy_version,
        metadata: %{
          query: retrieval_package.query,
          retrieval_path: retrieval_package.retrieval_plan[:path],
          filtered_object_summary: retrieval_package.filtered_summary
        }
      )

    Store.insert_derivation_entry(entry)
  end

  # ---------------------------------------------------------------------------
  # Private: helpers
  # ---------------------------------------------------------------------------

  defp collect_unique_links(objects, key) do
    objects
    |> Enum.flat_map(&Map.get(&1, key, []))
    |> Enum.uniq()
  end

  defp build_retrieval_plan(query, %ScopeEnvelope{} = scope, authorization, opts) do
    structured_filters =
      %{
        subject_anchor: string_opt_or_nil(opts, :subject_anchor),
        action_class: string_opt_or_nil(opts, :action_class),
        object_anchor: string_opt_or_nil(opts, :object_anchor)
      }
      |> reject_nil_values()

    asset_filters =
      %{
        modality: string_opt_or_nil(opts, :modality),
        extraction_type: string_opt_or_nil(opts, :extraction_type)
      }
      |> reject_nil_values()

    workflow_filters =
      %{workflow_family: string_opt_or_nil(opts, :workflow_family)}
      |> reject_nil_values()

    skill_filters =
      %{
        task_family: string_opt_or_nil(opts, :task_family),
        skill_package_name: string_opt_or_nil(opts, :skill_package_name)
      }
      |> reject_nil_values()

    %{
      query: query,
      path: retrieval_path(structured_filters, asset_filters, workflow_filters, skill_filters),
      intent: string_opt(opts, :request_intent, classify_intent(query)),
      executed_paths: [
        "facts.sql_like",
        "memory_objects.sql_like",
        "asset_extractions.sql_like",
        "generalized_workflows.sql_like",
        "skill_packages.sql_like",
        "workflow_skill.eligibility_filter",
        "authorization.pre_assembly_filter"
      ],
      filters: %{
        tenant_id: scope.tenant_id,
        workspace_id: scope.workspace_id,
        lifecycle: [
          "facts.accepted",
          "memory_objects.current",
          "asset_extractions.projected",
          "generalized_workflows.candidate_or_current",
          skill_package_lifecycle_filter(opts)
        ],
        time_mode: string_opt(opts, :time_mode, "current_valid"),
        authorization: %{
          allowed_partitions: authorization.allowed_partitions,
          allowed_security_labels: authorization.allowed_security_labels,
          applied: "during_candidate_expansion",
          empty_envelope_behavior: "fail_closed"
        }
      },
      structured_filters: structured_filters,
      asset_filters: asset_filters,
      workflow_filters: workflow_filters,
      skill_filters: skill_filters,
      limit: Keyword.get(opts, :limit, @default_limit)
    }
  end

  defp retrieval_path(structured_filters, asset_filters, workflow_filters, skill_filters)
       when map_size(structured_filters) > 0 or map_size(asset_filters) > 0 or
              map_size(workflow_filters) > 0 or map_size(skill_filters) > 0 do
    "memory_core_structured_lookup"
  end

  defp retrieval_path(_structured_filters, _asset_filters, _workflow_filters, _skill_filters) do
    "memory_core_fact_memory_workflow_skill_lookup"
  end

  defp skill_package_lifecycle_filter(opts) do
    if Keyword.get(opts, :include_draft_skills, false) do
      "skill_packages.active_including_drafts"
    else
      "skill_packages.approved_enabled_active"
    end
  end

  defp reject_nil_values(map) do
    map
    |> Enum.reject(fn {_key, value} -> is_nil(value) end)
    |> Map.new()
  end

  defp authorized_projection?(object, authorization) do
    if Map.get(authorization, :allow_all?, false) do
      true
    else
      authorized_projection_scoped?(object, authorization)
    end
  end

  defp authorized_projection_scoped?(object, authorization) do
    partition_allowed? =
      case authorization.allowed_partitions do
        [] ->
          Map.get(object, :partition_ids, []) == []

        allowed ->
          MapSet.size(
            MapSet.intersection(
              MapSet.new(Map.get(object, :partition_ids, [])),
              MapSet.new(allowed)
            )
          ) > 0
      end

    security_allowed? =
      case authorization.allowed_security_labels do
        [] ->
          Map.get(object, :security_labels, []) == []

        allowed ->
          MapSet.subset?(MapSet.new(Map.get(object, :security_labels, [])), MapSet.new(allowed))
      end

    partition_allowed? and security_allowed?
  end

  defp redacted_projection_ref(kind, object, authorization) do
    reasons =
      []
      |> maybe_reason(
        "partition_not_allowed",
        not authorized_projection?(Map.put(object, :security_labels, []), authorization)
      )
      |> maybe_reason(
        "security_label_not_allowed",
        not authorized_projection?(Map.put(object, :partition_ids, []), authorization)
      )

    %{type: projection_ref_type(kind), id: object.id, reasons: reasons}
  end

  defp maybe_reason(reasons, _reason, false), do: reasons
  defp maybe_reason(reasons, reason, true), do: [reason | reasons]

  defp projection_ref_type(:asset_extractions), do: "asset_extraction"
  defp projection_ref_type(:workflows), do: "generalized_workflow"
  defp projection_ref_type(:skill_packages), do: "skill_package"

  defp merge_lists(objects, key) do
    objects
    |> Enum.flat_map(&Map.get(&1, key, []))
    |> Enum.uniq()
  end

  defp summarize_score([], _field), do: %{count: 0, min: nil, max: nil, average: nil}

  defp summarize_score(objects, field) do
    values = Enum.map(objects, &Map.get(&1, field, 0.0))
    count = length(values)

    %{
      count: count,
      min: Enum.min(values),
      max: Enum.max(values),
      average: Enum.sum(values) / count
    }
  end

  defp classify_intent(""), do: "recall"
  defp classify_intent(_query), do: "recall"

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, list} when is_list(list) -> list
      {:ok, other} -> [other]
      _ -> []
    end
  end

  defp decode_list(value) when is_list(value), do: value
  defp decode_list(value), do: [value]

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

  defp map_get(map, key) when is_map(map) do
    Map.get(map, key) || Map.get(map, to_string(key))
  end

  defp map_get(_map, _key), do: nil

  defp link_type?(%{type: type}, expected), do: type == expected
  defp link_type?(%{"type" => type}, expected), do: type == expected
  defp link_type?(_, _expected), do: false

  defp link_id(%{"id" => id}), do: id
  defp link_id(%{id: id}), do: id
  defp link_id(id) when is_binary(id), do: id
  defp link_id(_), do: nil

  defp ref(type, id), do: %{type: type, id: id}

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()

  defp string_opt(opts, key, default), do: Keyword.get(opts, key, default) |> to_string()

  defp string_opt_or_nil(opts, key) do
    case Keyword.get(opts, key) do
      nil -> nil
      "" -> nil
      value -> to_string(value)
    end
  end

  defp string_list_opt(opts, key) do
    opts
    |> Keyword.get(key, [])
    |> List.wrap()
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.map(&to_string/1)
  end

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)
end
