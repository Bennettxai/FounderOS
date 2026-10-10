defmodule OptimalEngine.MemoryCore.Fact do
  @moduledoc """
  Typed Fact object for the Truth Lifecycle.

  A Fact is an accepted assertion promoted from one or more Claims through
  evidence review. Fields map one-to-one onto the `facts` table. Facts are
  bitemporal: `valid_time_start`/`valid_time_end` describe when the fact is
  true in the world, `transaction_time_start`/`transaction_time_end` describe
  when the engine recorded and (if superseded) closed this version.

  Supersession lineage is carried in `supersedes`/`superseded_by` (virtual
  fields persisted in `metadata` plus a `supersedes` relationship edge); a
  superseded fact keeps `lifecycle_state: "superseded"` and a closed
  transaction time rather than being deleted.
  """

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          fact_text: String.t() | nil,
          fact_type: String.t(),
          subject_anchor: String.t() | nil,
          action_class: String.t() | nil,
          object_anchor: String.t() | nil,
          scope: map(),
          accepted_claim_ids: [String.t()],
          supporting_evidence_links: [map()],
          contradicting_evidence_links: [map()],
          verifier_id: String.t() | nil,
          verification_status: String.t(),
          aggregate_confidence: float(),
          aggregate_precision: float(),
          raw_component_scores: map(),
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          lifecycle_state: String.t(),
          contradiction_status: String.t(),
          event_time: String.t() | nil,
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          transaction_time_start: String.t() | nil,
          transaction_time_end: String.t() | nil,
          verification_time: String.t() | nil,
          stale_after: String.t() | nil,
          metadata: map(),
          created_at: String.t() | nil,
          updated_at: String.t() | nil,
          supersedes: [String.t()],
          superseded_by: String.t() | nil
        }

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :fact_text,
    :fact_type,
    :subject_anchor,
    :action_class,
    :object_anchor,
    :scope,
    :accepted_claim_ids,
    :supporting_evidence_links,
    :contradicting_evidence_links,
    :verifier_id,
    :verification_status,
    :aggregate_confidence,
    :aggregate_precision,
    :raw_component_scores,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :contradiction_status,
    :event_time,
    :valid_time_start,
    :valid_time_end,
    :transaction_time_start,
    :transaction_time_end,
    :verification_time,
    :stale_after,
    :metadata,
    :created_at,
    :updated_at,
    # Supersession lineage (persisted via metadata + relationship edges).
    :supersedes,
    :superseded_by
  ]

  @doc """
  Build a Fact struct from an attribute map (atom keys).

  `supersedes`/`superseded_by` may be given directly or read back out of
  `metadata`; either way both representations are kept in sync.
  """
  @spec new(map()) :: t()
  def new(%__MODULE__{} = fact), do: sync_supersession(fact)

  def new(attrs) when is_map(attrs) do
    metadata = Map.get(attrs, :metadata) || %{}

    supersedes =
      Map.get(attrs, :supersedes) ||
        Map.get(metadata, "supersedes") || Map.get(metadata, :supersedes) || []

    superseded_by =
      Map.get(attrs, :superseded_by) ||
        Map.get(metadata, "superseded_by") || Map.get(metadata, :superseded_by)

    %__MODULE__{
      id: Map.get(attrs, :id),
      tenant_id: Map.get(attrs, :tenant_id, "default"),
      workspace_id: Map.get(attrs, :workspace_id, "default"),
      fact_text: Map.get(attrs, :fact_text),
      fact_type: Map.get(attrs, :fact_type, "assertion"),
      subject_anchor: Map.get(attrs, :subject_anchor),
      action_class: Map.get(attrs, :action_class),
      object_anchor: Map.get(attrs, :object_anchor),
      scope: Map.get(attrs, :scope) || %{},
      accepted_claim_ids: Map.get(attrs, :accepted_claim_ids) || [],
      supporting_evidence_links: Map.get(attrs, :supporting_evidence_links) || [],
      contradicting_evidence_links: Map.get(attrs, :contradicting_evidence_links) || [],
      verifier_id: Map.get(attrs, :verifier_id),
      verification_status: Map.get(attrs, :verification_status, "unverified"),
      aggregate_confidence: Map.get(attrs, :aggregate_confidence) || 0.5,
      aggregate_precision: Map.get(attrs, :aggregate_precision) || 0.5,
      raw_component_scores: Map.get(attrs, :raw_component_scores) || %{},
      access_policy_id: Map.get(attrs, :access_policy_id),
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state, "candidate"),
      contradiction_status: Map.get(attrs, :contradiction_status, "none"),
      event_time: Map.get(attrs, :event_time),
      valid_time_start: Map.get(attrs, :valid_time_start),
      valid_time_end: Map.get(attrs, :valid_time_end),
      transaction_time_start: Map.get(attrs, :transaction_time_start),
      transaction_time_end: Map.get(attrs, :transaction_time_end),
      verification_time: Map.get(attrs, :verification_time),
      stale_after: Map.get(attrs, :stale_after),
      metadata: metadata,
      created_at: Map.get(attrs, :created_at),
      updated_at: Map.get(attrs, :updated_at),
      supersedes: List.wrap(supersedes),
      superseded_by: superseded_by
    }
    |> sync_supersession()
  end

  defp sync_supersession(%__MODULE__{} = fact) do
    metadata =
      fact.metadata
      |> maybe_put("supersedes", if(fact.supersedes != [], do: fact.supersedes))
      |> maybe_put("superseded_by", fact.superseded_by)

    %{fact | metadata: metadata, supersedes: List.wrap(fact.supersedes)}
  end

  defp maybe_put(map, _key, nil), do: map
  defp maybe_put(map, key, value), do: Map.put(map, key, value)
end
