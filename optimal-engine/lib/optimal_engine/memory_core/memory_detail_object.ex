defmodule OptimalEngine.MemoryCore.MemoryDetailObject do
  @moduledoc """
  Recursive detail attached to a parent Memory Core object (a Memory Object,
  Episode Object, Workflow Trace, or Skill Package).

  A Memory Detail Object captures a single step, command, check, or exception
  that belongs to a coarser-grained parent. Details are ordered (`detail_order`)
  and can nest (`detail_depth`), so a workflow step can itself own sub-steps.

  This is the durable home for the "how" behind an accepted memory: the
  individual commands or parameter values that the parent summarizes. Details
  carry the same governance envelope (security labels, partitions, lifecycle)
  as every other Memory Core object and link back to the source/evidence that
  produced them.

  Construction is pure (`new/1`); persistence belongs to
  `OptimalEngine.MemoryCore.Store.insert_memory_detail_object/1`.
  """

  alias OptimalEngine.MemoryCore.ID

  @type t :: %__MODULE__{}

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :parent_object_type,
    :parent_object_id,
    :detail_type,
    :detail_order,
    :detail_depth,
    :action_class,
    :detail_text,
    :command_or_parameter_value,
    :source_package_links,
    :evidence_links,
    :aggregate_confidence,
    :aggregate_precision,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :reuse_status,
    :valid_time_start,
    :valid_time_end,
    :transaction_time_start,
    :transaction_time_end,
    :stale_after,
    :metadata,
    :created_at,
    :updated_at
  ]

  @doc """
  Build a Memory Detail Object struct from an attribute map.

  Requires `:parent_object_type`, `:parent_object_id`, and `:detail_text`.
  Defaults match the `memory_detail_objects` table DDL so a freshly built
  detail inserts cleanly.
  """
  @spec new(map()) :: {:ok, t()} | {:error, term()}
  def new(%__MODULE__{} = detail), do: {:ok, detail}

  def new(attrs) when is_map(attrs) do
    parent_type = string_or_nil(Map.get(attrs, :parent_object_type))
    parent_id = string_or_nil(Map.get(attrs, :parent_object_id))
    detail_text = string_or_nil(Map.get(attrs, :detail_text))

    cond do
      is_nil(parent_type) -> {:error, :parent_object_type_required}
      is_nil(parent_id) -> {:error, :parent_object_id_required}
      is_nil(detail_text) -> {:error, :detail_text_required}
      true -> {:ok, build(attrs, parent_type, parent_id, detail_text)}
    end
  end

  defp build(attrs, parent_type, parent_id, detail_text) do
    tenant_id = Map.get(attrs, :tenant_id, "default")
    workspace_id = Map.get(attrs, :workspace_id, "default")
    detail_order = Map.get(attrs, :detail_order, 0)

    %__MODULE__{
      id:
        Map.get(attrs, :id) ||
          ID.content_id("mdo", [
            tenant_id,
            ":",
            workspace_id,
            ":",
            parent_type,
            ":",
            parent_id,
            ":",
            Integer.to_string(detail_order),
            ":",
            detail_text
          ]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      parent_object_type: parent_type,
      parent_object_id: parent_id,
      detail_type: Map.get(attrs, :detail_type, "step"),
      detail_order: detail_order,
      detail_depth: Map.get(attrs, :detail_depth, 0),
      action_class: string_or_nil(Map.get(attrs, :action_class)),
      detail_text: detail_text,
      command_or_parameter_value: string_or_nil(Map.get(attrs, :command_or_parameter_value)),
      source_package_links: Map.get(attrs, :source_package_links) || [],
      evidence_links: Map.get(attrs, :evidence_links) || [],
      aggregate_confidence: Map.get(attrs, :aggregate_confidence) || 0.5,
      aggregate_precision: Map.get(attrs, :aggregate_precision) || 0.5,
      access_policy_id: string_or_nil(Map.get(attrs, :access_policy_id)),
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state, "candidate"),
      reuse_status: Map.get(attrs, :reuse_status, "local"),
      valid_time_start: Map.get(attrs, :valid_time_start),
      valid_time_end: Map.get(attrs, :valid_time_end),
      transaction_time_start: Map.get(attrs, :transaction_time_start),
      transaction_time_end: Map.get(attrs, :transaction_time_end),
      stale_after: Map.get(attrs, :stale_after),
      metadata: Map.get(attrs, :metadata) || %{}
    }
  end

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)
end
