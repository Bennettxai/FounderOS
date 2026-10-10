defmodule OptimalEngine.MemoryCore.DerivationLedgerEntry do
  @moduledoc """
  Auditable derivation record connecting source evidence to derived objects.
  """

  alias OptimalEngine.MemoryCore.ID

  @type object_ref :: %{type: String.t(), id: String.t()}

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          activity_type: String.t(),
          derivation_stage: String.t(),
          input_object_links: [object_ref()],
          output_object_links: [object_ref()],
          source_package_links: [object_ref()],
          evidence_links: [object_ref()],
          actor_id: String.t() | nil,
          evaluator_id: String.t() | nil,
          parser_id: String.t() | nil,
          model_id: String.t() | nil,
          model_version: String.t() | nil,
          prompt_template_id: String.t() | nil,
          tool_call_links: [object_ref()],
          confidence_delta: float() | nil,
          precision_delta: float() | nil,
          scoring_policy_version: String.t() | nil,
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          lifecycle_state: String.t(),
          replay_status: String.t(),
          activity_time: String.t(),
          transaction_time_start: String.t(),
          transaction_time_end: String.t() | nil,
          audit_event_links: [object_ref()],
          policy_version: String.t() | nil,
          metadata: map(),
          created_at: String.t()
        }

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :activity_type,
    :derivation_stage,
    :input_object_links,
    :output_object_links,
    :source_package_links,
    :evidence_links,
    :actor_id,
    :evaluator_id,
    :parser_id,
    :model_id,
    :model_version,
    :prompt_template_id,
    :tool_call_links,
    :confidence_delta,
    :precision_delta,
    :scoring_policy_version,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :replay_status,
    :activity_time,
    :transaction_time_start,
    :transaction_time_end,
    :audit_event_links,
    :policy_version,
    :metadata,
    :created_at
  ]

  @spec object_ref(atom() | String.t(), String.t()) :: object_ref()
  def object_ref(type, id), do: %{type: to_string(type), id: id}

  @spec new(String.t(), String.t(), [object_ref()], [object_ref()], keyword()) :: t()
  def new(activity_type, derivation_stage, input_links, output_links, opts \\ []) do
    now = timestamp()

    %__MODULE__{
      id: ID.random_id("dl"),
      tenant_id: string_opt(opts, :tenant_id, "default"),
      workspace_id: string_opt(opts, :workspace_id, "default"),
      activity_type: activity_type,
      derivation_stage: derivation_stage,
      input_object_links: input_links,
      output_object_links: output_links,
      source_package_links: Keyword.get(opts, :source_package_links, []),
      evidence_links: Keyword.get(opts, :evidence_links, []),
      actor_id: string_or_nil(Keyword.get(opts, :actor_id)),
      evaluator_id: string_or_nil(Keyword.get(opts, :evaluator_id)),
      parser_id: string_or_nil(Keyword.get(opts, :parser_id)),
      model_id: string_or_nil(Keyword.get(opts, :model_id)),
      model_version: string_or_nil(Keyword.get(opts, :model_version)),
      prompt_template_id: string_or_nil(Keyword.get(opts, :prompt_template_id)),
      tool_call_links: Keyword.get(opts, :tool_call_links, []),
      confidence_delta: Keyword.get(opts, :confidence_delta),
      precision_delta: Keyword.get(opts, :precision_delta),
      scoring_policy_version: string_or_nil(Keyword.get(opts, :scoring_policy_version)),
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: list_opt(opts, :security_labels),
      partition_ids: list_opt(opts, :partition_ids),
      lifecycle_state: string_opt(opts, :lifecycle_state, "recorded"),
      replay_status: string_opt(opts, :replay_status, "replayable"),
      activity_time: serialize_time(Keyword.get(opts, :activity_time)) || now,
      transaction_time_start: now,
      transaction_time_end: serialize_time(Keyword.get(opts, :transaction_time_end)),
      audit_event_links: Keyword.get(opts, :audit_event_links, []),
      policy_version: string_or_nil(Keyword.get(opts, :policy_version)),
      metadata: Keyword.get(opts, :metadata, %{}),
      created_at: now
    }
  end

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()

  defp serialize_time(nil), do: nil
  defp serialize_time(%DateTime{} = value), do: DateTime.to_iso8601(value)
  defp serialize_time(value) when is_binary(value), do: value
  defp serialize_time(value), do: to_string(value)

  defp string_opt(opts, key, default), do: Keyword.get(opts, key, default) |> to_string()

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)

  defp list_opt(opts, key) do
    opts
    |> Keyword.get(key, [])
    |> List.wrap()
    |> Enum.map(&to_string/1)
  end
end
