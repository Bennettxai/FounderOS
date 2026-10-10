defmodule OptimalEngine.MemoryCore.ContextPackage do
  @moduledoc """
  AI-usable projection of a Retrieval Package — the final Governed Recall
  output.

  A Context Package is what agents and humans receive instead of loose search
  chunks: assembled, token-budget-aware sections, Facts with source citations,
  Memory Objects, the authorization envelope that scoped the recall, freshness
  state, and the filtered-object summary. Fields map one-to-one onto the
  `context_packages` table; `:facts`, `:memory_objects`, and `:sections` are
  hydrated virtual fields (sections are also preserved in `metadata` so the
  stored package stays replayable).

  Produced by `OptimalEngine.MemoryCore.RetrievalCoordinator.retrieve/2` and
  `build_context_package/2`.
  """

  @type object_ref :: %{type: String.t(), id: String.t()}

  @type sections :: %{
          summary: String.t(),
          facts: String.t(),
          memory: String.t(),
          token_budget: non_neg_integer(),
          total_tokens: non_neg_integer(),
          truncated?: boolean()
        }

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          request_id: String.t() | nil,
          request_intent: String.t() | nil,
          requesting_actor_id: String.t() | nil,
          active_memory_pool_id: String.t() | nil,
          time_mode: String.t(),
          detail_depth: integer(),
          memory_links: [object_ref()],
          fact_links: [object_ref()],
          workflow_links: [object_ref()],
          skill_package_links: [object_ref()],
          source_package_links: [map()],
          evidence_links: [map()],
          retrieval_plan: map(),
          package_confidence_summary: map(),
          package_precision_summary: map(),
          filtered_object_summary: map(),
          returned_object_links: [object_ref()],
          redacted_object_links: [map()],
          authorization_envelope: map(),
          lifecycle_state: String.t(),
          refresh_state: String.t(),
          invalidation_reason: String.t() | nil,
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          transaction_time_start: String.t() | nil,
          transaction_time_end: String.t() | nil,
          stale_after: String.t() | nil,
          refresh_time: String.t() | nil,
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          audit_event_links: [map()],
          policy_version: String.t() | nil,
          metadata: map(),
          facts: [map()],
          memory_objects: [map()],
          asset_extractions: [map()],
          workflows: [map()],
          skill_packages: [map()],
          sections: sections() | map()
        }

  defstruct id: nil,
            tenant_id: "default",
            workspace_id: "default",
            request_id: nil,
            request_intent: nil,
            requesting_actor_id: nil,
            active_memory_pool_id: nil,
            time_mode: "current_valid",
            detail_depth: 1,
            memory_links: [],
            fact_links: [],
            workflow_links: [],
            skill_package_links: [],
            source_package_links: [],
            evidence_links: [],
            retrieval_plan: %{},
            package_confidence_summary: %{},
            package_precision_summary: %{},
            filtered_object_summary: %{},
            returned_object_links: [],
            redacted_object_links: [],
            authorization_envelope: %{},
            lifecycle_state: "assembled",
            refresh_state: "fresh",
            invalidation_reason: nil,
            valid_time_start: nil,
            valid_time_end: nil,
            transaction_time_start: nil,
            transaction_time_end: nil,
            stale_after: nil,
            refresh_time: nil,
            access_policy_id: nil,
            security_labels: [],
            partition_ids: [],
            audit_event_links: [],
            policy_version: nil,
            metadata: %{},
            facts: [],
            memory_objects: [],
            asset_extractions: [],
            workflows: [],
            skill_packages: [],
            sections: %{}

  @doc "Build a Context Package from an attribute map (atom keys)."
  @spec new(map()) :: t()
  def new(attrs) when is_map(attrs), do: struct(__MODULE__, attrs)

  @doc """
  JSON-safe plain map projection of the package (for API responses).

  Hydrated `:facts`/`:memory_objects` are already plain maps, so the result
  encodes directly with `Jason`.
  """
  @spec to_map(t()) :: map()
  def to_map(%__MODULE__{} = package), do: Map.from_struct(package)
end
