defmodule OptimalEngine.MemoryCore.SourcePackage do
  @moduledoc """
  Preserved source evidence for the Memory Core spine.

  A Source Package is the first durable object in the memory lifecycle. It keeps
  the raw source, provenance labels, retention/security scope, and enough timing
  data for later Claim, Fact, Memory Object, workflow, and rebuild stages.
  """

  alias OptimalEngine.MemoryCore.ID

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          source_type: String.t(),
          source_class: String.t(),
          source_system: String.t() | nil,
          source_uri: String.t() | nil,
          source_time: String.t() | nil,
          received_at: String.t(),
          content_hash: String.t(),
          raw_text: String.t(),
          verbatim_archive_uri: String.t() | nil,
          trust_label: String.t(),
          retention_class: String.t(),
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          quarantine_state: String.t(),
          metadata: map(),
          created_by: String.t() | nil,
          created_at: String.t(),
          updated_at: String.t()
        }

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :source_type,
    :source_class,
    :source_system,
    :source_uri,
    :source_time,
    :received_at,
    :content_hash,
    :raw_text,
    :verbatim_archive_uri,
    :trust_label,
    :retention_class,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :quarantine_state,
    :metadata,
    :created_by,
    :created_at,
    :updated_at
  ]

  @spec from_text(String.t(), keyword()) :: t()
  def from_text(raw_text, opts \\ []) when is_binary(raw_text) do
    now = timestamp()
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = string_opt(opts, :workspace_id, "default")
    content_hash = ID.sha256(raw_text)

    %__MODULE__{
      id: ID.content_id("sp", [tenant_id, ":", workspace_id, ":", content_hash]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      source_type: string_opt(opts, :source_type, "manual"),
      source_class: string_opt(opts, :source_class, "text"),
      source_system: string_or_nil(Keyword.get(opts, :source_system)),
      source_uri: string_or_nil(Keyword.get(opts, :source_uri)),
      source_time: serialize_time(Keyword.get(opts, :source_time)),
      received_at: now,
      content_hash: content_hash,
      raw_text: raw_text,
      verbatim_archive_uri: string_or_nil(Keyword.get(opts, :verbatim_archive_uri)),
      trust_label: string_opt(opts, :trust_label, "unreviewed"),
      retention_class: string_opt(opts, :retention_class, "standard"),
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: list_opt(opts, :security_labels),
      partition_ids: list_opt(opts, :partition_ids),
      quarantine_state: string_opt(opts, :quarantine_state, "clear"),
      metadata: Keyword.get(opts, :metadata, %{}),
      created_by: string_or_nil(Keyword.get(opts, :created_by) || Keyword.get(opts, :actor_id)),
      created_at: now,
      updated_at: now
    }
  end

  @spec from_artifact(String.t(), String.t(), keyword()) :: t()
  def from_artifact(storage_uri, content_hash, opts \\ [])
      when is_binary(storage_uri) and is_binary(content_hash) do
    now = timestamp()
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = string_opt(opts, :workspace_id, "default")
    raw_text = Keyword.get(opts, :raw_text, "")

    %__MODULE__{
      id: ID.content_id("sp", [tenant_id, ":", workspace_id, ":", content_hash]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      source_type: string_opt(opts, :source_type, "file"),
      source_class: string_opt(opts, :source_class, "binary"),
      source_system: string_or_nil(Keyword.get(opts, :source_system)),
      source_uri: string_or_nil(Keyword.get(opts, :source_uri) || storage_uri),
      source_time: serialize_time(Keyword.get(opts, :source_time)),
      received_at: now,
      content_hash: content_hash,
      raw_text: raw_text,
      verbatim_archive_uri: string_or_nil(Keyword.get(opts, :verbatim_archive_uri) || storage_uri),
      trust_label: string_opt(opts, :trust_label, "unreviewed"),
      retention_class: string_opt(opts, :retention_class, "standard"),
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: list_opt(opts, :security_labels),
      partition_ids: list_opt(opts, :partition_ids),
      quarantine_state: string_opt(opts, :quarantine_state, "clear"),
      metadata: Keyword.get(opts, :metadata, %{}),
      created_by: string_or_nil(Keyword.get(opts, :created_by) || Keyword.get(opts, :actor_id)),
      created_at: now,
      updated_at: now
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
