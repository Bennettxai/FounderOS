defmodule OptimalEngine.Connectors.AssetIngest do
  @moduledoc """
  Preserves connector-delivered file attachments through Memory Core.

  Connectors should not treat downloaded files as loose side effects. A file
  from Slack, Gmail, Drive, Zoom, or another source first becomes a governed
  Source Package plus asset row, then later extraction/adapters can derive
  transcripts, OCR spans, visual observations, embeddings, Claims, and Facts.
  """

  alias OptimalEngine.MemoryCore

  @type attachment :: map()
  @type result :: %{
          assets: [map()],
          errors: [map()]
        }

  @attachment_keys [:attachments, "attachments", :files, "files"]

  @doc """
  Preserve all attachment-like payloads found in a connector raw payload.

  Supported attachment shapes:

      %{"path" => "/tmp/report.pdf", "filename" => "report.pdf"}
      %{"content_base64" => "...", "filename" => "call.wav"}
      %{path: "/tmp/image.png", content_type: "image/png"}

  The returned asset rows are governed Memory Core assets. Errors are returned
  per attachment so a connector can keep processing the rest of the batch.
  """
  @spec preserve_payload_assets(atom() | String.t(), String.t(), map(), keyword()) ::
          {:ok, result()}
  def preserve_payload_assets(connector_kind, external_id, payload, opts \\ [])
      when is_binary(external_id) and is_map(payload) do
    attachments = attachments_from_payload(payload)

    attachments
    |> Enum.with_index()
    |> Enum.reduce(%{assets: [], errors: []}, fn {attachment, index}, acc ->
      case preserve_attachment(connector_kind, external_id, attachment, index, opts) do
        {:ok, asset_result} ->
          %{acc | assets: [asset_result | acc.assets]}

        {:error, reason} ->
          error = %{
            index: index,
            filename: attachment_filename(attachment),
            reason: reason_to_string(reason)
          }

          %{acc | errors: [error | acc.errors]}
      end
    end)
    |> then(fn result ->
      {:ok,
       %{
         assets: Enum.reverse(result.assets),
         errors: Enum.reverse(result.errors)
       }}
    end)
  end

  @doc """
  Preserve one connector attachment through Memory Core.
  """
  @spec preserve_attachment(
          atom() | String.t(),
          String.t(),
          attachment(),
          non_neg_integer(),
          keyword()
        ) ::
          {:ok, map()} | {:error, term()}
  def preserve_attachment(connector_kind, external_id, attachment, index \\ 0, opts \\ [])
      when is_binary(external_id) and is_map(attachment) do
    with {:ok, source_path, cleanup} <- attachment_source_path(attachment) do
      try do
        MemoryCore.store_asset_file(
          source_path,
          store_opts(connector_kind, external_id, attachment, index, opts)
        )
      after
        cleanup.()
      end
    end
  end

  defp attachments_from_payload(payload) do
    Enum.find_value(@attachment_keys, [], fn key ->
      case Map.get(payload, key) do
        values when is_list(values) -> values
        value when is_map(value) -> [value]
        _ -> nil
      end
    end)
    |> Enum.filter(&is_map/1)
  end

  defp attachment_source_path(attachment) do
    cond do
      is_binary(field(attachment, :path)) and String.trim(field(attachment, :path)) != "" ->
        path = field(attachment, :path)

        if File.exists?(path) do
          {:ok, path, fn -> :ok end}
        else
          {:error, :path_not_found}
        end

      is_binary(field(attachment, :file_path)) and String.trim(field(attachment, :file_path)) != "" ->
        path = field(attachment, :file_path)

        if File.exists?(path) do
          {:ok, path, fn -> :ok end}
        else
          {:error, :path_not_found}
        end

      is_binary(field(attachment, :content_base64)) and
          String.trim(field(attachment, :content_base64)) != "" ->
        write_temp_base64(attachment, field(attachment, :content_base64))

      is_binary(field(attachment, :data_base64)) and
          String.trim(field(attachment, :data_base64)) != "" ->
        write_temp_base64(attachment, field(attachment, :data_base64))

      true ->
        {:error, :attachment_source_required}
    end
  end

  defp write_temp_base64(attachment, encoded) do
    case Base.decode64(encoded) do
      {:ok, bytes} ->
        safe_name =
          attachment
          |> attachment_filename()
          |> Path.basename()
          |> String.replace(~r/[^A-Za-z0-9._-]/, "_")
          |> case do
            "" -> "attachment.bin"
            value -> value
          end

        path =
          Path.join(
            System.tmp_dir!(),
            "optimal-connector-asset-#{System.unique_integer([:positive])}-#{safe_name}"
          )

        case File.write(path, bytes) do
          :ok -> {:ok, path, fn -> File.rm(path) end}
          {:error, reason} -> {:error, {:write_failed, reason}}
        end

      :error ->
        {:error, :invalid_base64}
    end
  end

  defp store_opts(connector_kind, external_id, attachment, index, opts) do
    kind = connector_kind |> to_string()
    attachment_id = attachment_id(attachment, index)
    source_uri = source_uri(kind, external_id, attachment_id)

    metadata =
      opts
      |> Keyword.get(:metadata, %{})
      |> coerce_metadata()
      |> Map.merge(coerce_metadata(field(attachment, :metadata)))
      |> Map.merge(%{
        connector_kind: kind,
        connector_id: Keyword.get(opts, :connector_id),
        external_id: external_id,
        attachment_id: attachment_id,
        filename: attachment_filename(attachment),
        source_uri: source_uri
      })
      |> reject_nil_values()

    [
      tenant_id: Keyword.get(opts, :tenant_id, "default"),
      workspace_id: Keyword.get(opts, :workspace_id, "default"),
      actor_id: Keyword.get(opts, :actor_id, "system:connector-asset-ingest"),
      source_type: Keyword.get(opts, :source_type, "connector_file"),
      source_system: Keyword.get(opts, :source_system, kind),
      source_uri: source_uri,
      content_type: field(attachment, :content_type) || field(attachment, :mime_type),
      modality: normalize_modality(field(attachment, :modality)),
      trust_label: Keyword.get(opts, :trust_label, "connector_unreviewed"),
      retention_class: Keyword.get(opts, :retention_class, "standard"),
      access_policy_id: Keyword.get(opts, :access_policy_id),
      security_labels: Keyword.get(opts, :security_labels, []),
      partition_ids: Keyword.get(opts, :partition_ids, []),
      metadata: metadata
    ]
    |> Enum.reject(fn {_key, value} -> is_nil(value) end)
  end

  defp attachment_id(attachment, index) do
    field(attachment, :id) ||
      field(attachment, :attachment_id) ||
      field(attachment, :file_id) ||
      "attachment-#{index}"
  end

  defp attachment_filename(attachment) do
    field(attachment, :filename) ||
      field(attachment, :name) ||
      field(attachment, :title) ||
      "attachment.bin"
  end

  defp source_uri(kind, external_id, attachment_id) do
    "optimal://connectors/#{kind}/#{external_id}/attachments/#{attachment_id}"
  end

  defp field(map, key) when is_atom(key) do
    Map.get(map, key) || Map.get(map, Atom.to_string(key))
  end

  defp normalize_modality(nil), do: nil
  defp normalize_modality(value) when is_atom(value), do: value

  defp normalize_modality(value) when is_binary(value) do
    case String.downcase(value) do
      "image" -> :image
      "audio" -> :audio
      "video" -> :video
      "binary" -> :binary
      _ -> nil
    end
  end

  defp coerce_metadata(value) when is_map(value), do: stringify_keys(value)
  defp coerce_metadata(_value), do: %{}

  defp stringify_keys(map) do
    Map.new(map, fn {key, value} -> {to_string(key), value} end)
  end

  defp reject_nil_values(map) do
    Map.reject(map, fn {_key, value} -> is_nil(value) end)
  end

  defp reason_to_string(reason) when is_atom(reason), do: Atom.to_string(reason)
  defp reason_to_string(reason), do: inspect(reason)
end
