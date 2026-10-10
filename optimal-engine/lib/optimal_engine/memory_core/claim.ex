defmodule OptimalEngine.MemoryCore.Claim do
  @moduledoc """
  Typed Claim object for the Truth Lifecycle.

  A Claim is an extracted assertion that is not yet accepted truth. Fields map
  one-to-one onto the `claims` table. The contract aliases `subject`,
  `predicate`, `object_value`, `status`, `confidence`, and `extracted_by` are
  kept in sync with the canonical columns (`subject_anchor`, `action_class`,
  `object_anchor`, `lifecycle_state`, `aggregate_confidence`, `evaluator_id`)
  so callers can rely on either vocabulary.

  Status values: `"pending"` (extracted, unreviewed), `"promoted"` (accepted
  into a Fact through review), `"rejected"` (review declined the claim).
  """

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          source_package_id: String.t() | nil,
          signal_id: String.t() | nil,
          claim_text: String.t() | nil,
          claim_type: String.t(),
          subject_anchor: String.t() | nil,
          action_class: String.t() | nil,
          object_anchor: String.t() | nil,
          semantic_frame: map(),
          source_span: map(),
          extraction_run_id: String.t() | nil,
          evaluator_id: String.t() | nil,
          aggregate_confidence: float(),
          aggregate_precision: float(),
          raw_component_scores: map(),
          calibration_dataset_version: String.t() | nil,
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          lifecycle_state: String.t(),
          review_status: String.t(),
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          transaction_time_start: String.t() | nil,
          transaction_time_end: String.t() | nil,
          stale_after: String.t() | nil,
          metadata: map(),
          created_at: String.t() | nil,
          updated_at: String.t() | nil,
          subject: String.t() | nil,
          predicate: String.t() | nil,
          object_value: String.t() | nil,
          status: String.t(),
          confidence: float(),
          extracted_by: String.t() | nil
        }

  defstruct [
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
    :semantic_frame,
    :source_span,
    :extraction_run_id,
    :evaluator_id,
    :aggregate_confidence,
    :aggregate_precision,
    :raw_component_scores,
    :calibration_dataset_version,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :review_status,
    :valid_time_start,
    :valid_time_end,
    :transaction_time_start,
    :transaction_time_end,
    :stale_after,
    :metadata,
    :created_at,
    :updated_at,
    # Contract aliases (kept in sync with canonical columns above).
    :subject,
    :predicate,
    :object_value,
    :status,
    :confidence,
    :extracted_by
  ]

  @doc """
  Build a Claim struct from an attribute map (atom keys).

  Accepts either the canonical column names or the contract alias names and
  keeps both in sync. Does not apply scoring policy; see
  `OptimalEngine.MemoryCore.ScoringPolicy`.
  """
  @spec new(map()) :: t()
  def new(%__MODULE__{} = claim), do: sync_aliases(claim)

  def new(attrs) when is_map(attrs) do
    %__MODULE__{
      id: Map.get(attrs, :id),
      tenant_id: Map.get(attrs, :tenant_id, "default"),
      workspace_id: Map.get(attrs, :workspace_id, "default"),
      source_package_id: Map.get(attrs, :source_package_id),
      signal_id: Map.get(attrs, :signal_id),
      claim_text: Map.get(attrs, :claim_text),
      claim_type: Map.get(attrs, :claim_type, "assertion"),
      subject_anchor: Map.get(attrs, :subject_anchor) || Map.get(attrs, :subject),
      action_class: Map.get(attrs, :action_class) || Map.get(attrs, :predicate),
      object_anchor: Map.get(attrs, :object_anchor) || Map.get(attrs, :object_value),
      semantic_frame: Map.get(attrs, :semantic_frame) || %{},
      source_span: Map.get(attrs, :source_span) || %{},
      extraction_run_id: Map.get(attrs, :extraction_run_id),
      evaluator_id: Map.get(attrs, :evaluator_id) || Map.get(attrs, :extracted_by),
      aggregate_confidence:
        Map.get(attrs, :aggregate_confidence) || Map.get(attrs, :confidence) || 0.5,
      aggregate_precision: Map.get(attrs, :aggregate_precision) || 0.5,
      raw_component_scores: Map.get(attrs, :raw_component_scores) || %{},
      calibration_dataset_version: Map.get(attrs, :calibration_dataset_version),
      access_policy_id: Map.get(attrs, :access_policy_id),
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state) || Map.get(attrs, :status) || "pending",
      review_status: Map.get(attrs, :review_status, "unreviewed"),
      valid_time_start: Map.get(attrs, :valid_time_start),
      valid_time_end: Map.get(attrs, :valid_time_end),
      transaction_time_start: Map.get(attrs, :transaction_time_start),
      transaction_time_end: Map.get(attrs, :transaction_time_end),
      stale_after: Map.get(attrs, :stale_after),
      metadata: Map.get(attrs, :metadata) || %{},
      created_at: Map.get(attrs, :created_at),
      updated_at: Map.get(attrs, :updated_at)
    }
    |> sync_aliases()
  end

  defp sync_aliases(%__MODULE__{} = claim) do
    %{
      claim
      | subject: claim.subject_anchor,
        predicate: claim.action_class,
        object_value: claim.object_anchor,
        status: claim.lifecycle_state,
        confidence: claim.aggregate_confidence,
        extracted_by: claim.evaluator_id
    }
  end
end
