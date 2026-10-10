defmodule OptimalEngine.MemoryCore do
  @moduledoc """
  Public facade for the Memory Core backend layer.

  This module names the architecture boundary explicitly. It exposes the first
  production slices for source evidence, truth promotion, governed recall, and
  task-scoped working memory while keeping low-level persistence behind typed
  Memory Core services.
  """

  alias OptimalEngine.MemoryCore.{
    ActiveMemoryPool,
    AssetStore,
    ClaimReview,
    ClaimExtractor,
    FactPromoter,
    MemoryObject,
    RetrievalCoordinator,
    SourcePackage,
    ToolModelGovernance,
    WorkflowSkill
  }

  alias OptimalEngine.Pipeline.MultimodalAdapterRunner

  @spec source_package_from_text(String.t(), keyword()) :: SourcePackage.t()
  defdelegate source_package_from_text(raw_text, opts \\ []), to: SourcePackage, as: :from_text
  defdelegate store_asset_file(path, opts \\ []), to: AssetStore, as: :store_file
  defdelegate store_asset(asset, opts \\ []), to: AssetStore, as: :store_asset
  defdelegate get_asset(asset_id, opts \\ []), to: AssetStore, as: :get

  defdelegate record_asset_adapter_run(asset_id, opts \\ []),
    to: AssetStore,
    as: :record_adapter_run

  defdelegate get_asset_adapter_run(run_id, opts \\ []), to: AssetStore, as: :get_adapter_run
  defdelegate list_asset_adapter_runs(asset_id, opts \\ []), to: AssetStore, as: :list_adapter_runs

  defdelegate claim_from_asset_adapter_run(run_id, opts \\ []),
    to: AssetStore,
    as: :claim_from_adapter_run

  defdelegate record_asset_extraction(run_id, opts \\ []), to: AssetStore
  defdelegate get_asset_extraction(extraction_id, opts \\ []), to: AssetStore
  defdelegate list_asset_extractions(asset_id, opts \\ []), to: AssetStore
  defdelegate claim_from_asset_extraction(extraction_id, opts \\ []), to: AssetStore

  defdelegate run_asset_adapter(asset_id, adapter_id, opts \\ []),
    to: MultimodalAdapterRunner,
    as: :run

  defdelegate run_recommended_asset_adapter(asset_id, modality, opts \\ []),
    to: MultimodalAdapterRunner,
    as: :run_recommended

  defdelegate extract_claim(source_package, opts \\ []),
    to: ClaimExtractor,
    as: :extract_from_source

  defdelegate pending_claims(opts \\ []), to: ClaimReview, as: :pending
  defdelegate claim_review_queue(opts \\ []), to: ClaimReview, as: :queue
  defdelegate get_claim(claim_id, opts \\ []), to: ClaimReview, as: :get
  defdelegate reject_claim(claim_id, opts \\ []), to: ClaimReview, as: :reject
  defdelegate promote_claim(claim_id, opts \\ []), to: ClaimReview, as: :promote

  def promote_claim_to_fact(claim, opts \\ []) do
    opts =
      if Keyword.has_key?(opts, :policy) or Keyword.has_key?(opts, :verifier_id) or
           Keyword.has_key?(opts, :actor_id) do
        opts
      else
        Keyword.put(opts, :policy, :auto)
      end

    FactPromoter.promote(claim, opts)
  end

  defdelegate build_memory_object(fact, opts \\ []), to: MemoryObject, as: :build_from_fact
  defdelegate retrieve(query, opts \\ []), to: RetrievalCoordinator
  defdelegate refresh_context_package(context_package_id, opts \\ []), to: RetrievalCoordinator
  defdelegate refresh_stale_context_packages(opts \\ []), to: RetrievalCoordinator
  defdelegate open_active_pool(opts \\ []), to: ActiveMemoryPool, as: :open
  defdelegate get_active_pool(pool_id), to: ActiveMemoryPool, as: :get

  defdelegate load_context_package(pool_id, context_package, opts \\ []),
    to: ActiveMemoryPool,
    as: :load_context_package

  defdelegate refresh_pool_context_packages(pool_id, opts \\ []),
    to: ActiveMemoryPool,
    as: :refresh_context_packages

  defdelegate publish_pool_observation(pool_id, observation_text, opts \\ []),
    to: ActiveMemoryPool,
    as: :publish_observation

  defdelegate close_active_pool(pool_id, opts \\ []), to: ActiveMemoryPool, as: :close

  defdelegate capture_workflow_trace(pool_id, opts \\ []),
    to: WorkflowSkill,
    as: :capture_trace_from_pool

  defdelegate generalize_workflow(trace, opts \\ []), to: WorkflowSkill
  defdelegate create_procedural_memory(workflow, opts \\ []), to: WorkflowSkill
  defdelegate package_skill(procedure, opts \\ []), to: WorkflowSkill

  defdelegate register_model_call_operation(opts), to: ToolModelGovernance
  defdelegate register_mcp_tool_definition(opts), to: ToolModelGovernance
  defdelegate record_model_call(function_name, input_payload, opts \\ []), to: ToolModelGovernance
  defdelegate record_tool_call(tool_name, input_payload, opts \\ []), to: ToolModelGovernance

  defdelegate execute_tool_call(tool_name, input_payload, executor, opts \\ []),
    to: ToolModelGovernance
end
