defmodule OptimalEngine.MemoryCore.RetrievalPackage do
  @moduledoc """
  Structured retrieval result for the Governed Recall gate.

  The gate pipeline is `query -> authorization envelope -> retrieval plan ->
  Retrieval Package -> Context Package`. A Retrieval Package is the inspectable
  intermediate stage: the authorized candidate Facts and Memory Objects with
  their evidence and source links, the retrieval plan that produced them, the
  freshness window that filtered them, and the filtered-object accounting for
  everything authorization policy excluded.

  A Retrieval Package is not persisted and is never handed to agents as final
  context. `RetrievalCoordinator.build_context_package/2` projects it into the
  persisted `OptimalEngine.MemoryCore.ContextPackage`.
  """

  @type object_ref :: %{type: String.t(), id: String.t()}
  @type redacted_ref :: %{type: String.t(), id: String.t(), reasons: [String.t()]}

  @type scope :: %{
          actor_id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          operation_class: String.t(),
          permissions: [String.t()],
          allowed_partitions: [String.t()],
          allowed_security_labels: [String.t()],
          unresolved: [atom()]
        }

  @type filtered_summary :: %{
          candidate_facts: non_neg_integer(),
          returned_facts: non_neg_integer(),
          candidate_memory_objects: non_neg_integer(),
          returned_memory_objects: non_neg_integer(),
          redacted_or_filtered_objects: non_neg_integer(),
          reason_classes: %{optional(String.t()) => non_neg_integer()}
        }

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          query: String.t() | nil,
          request_intent: String.t(),
          time_mode: String.t(),
          limit: pos_integer(),
          scope: scope() | map(),
          retrieval_plan: map(),
          facts: [map()],
          memory_objects: [map()],
          asset_extractions: [map()],
          workflows: [map()],
          skill_packages: [map()],
          source_package_links: [map()],
          evidence_links: [map()],
          redacted_object_links: [redacted_ref()],
          filtered_summary: filtered_summary() | map(),
          confidence_summary: map(),
          precision_summary: map(),
          freshness: map(),
          retrieved_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: "default",
            workspace_id: "default",
            query: nil,
            request_intent: "recall",
            time_mode: "current_valid",
            limit: 10,
            scope: %{},
            retrieval_plan: %{},
            facts: [],
            memory_objects: [],
            asset_extractions: [],
            workflows: [],
            skill_packages: [],
            source_package_links: [],
            evidence_links: [],
            redacted_object_links: [],
            filtered_summary: %{},
            confidence_summary: %{},
            precision_summary: %{},
            freshness: %{},
            retrieved_at: nil

  @doc "Build a Retrieval Package from an attribute map (atom keys)."
  @spec new(map()) :: t()
  def new(attrs) when is_map(attrs), do: struct(__MODULE__, attrs)

  @doc "Object refs for every Fact and Memory Object the package returns."
  @spec returned_object_links(t()) :: [object_ref()]
  def returned_object_links(%__MODULE__{
        facts: facts,
        memory_objects: memory_objects,
        asset_extractions: asset_extractions,
        workflows: workflows,
        skill_packages: skill_packages
      }) do
    Enum.map(facts, &%{type: "fact", id: &1.id}) ++
      Enum.map(memory_objects, &%{type: "memory_object", id: &1.id}) ++
      Enum.map(asset_extractions, &%{type: "asset_extraction", id: &1.id}) ++
      Enum.map(workflows, &%{type: "generalized_workflow", id: &1.id}) ++
      Enum.map(skill_packages, &%{type: "skill_package", id: &1.id})
  end
end
