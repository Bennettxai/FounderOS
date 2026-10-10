defmodule OptimalEngine.Connectors.AssetIngestTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Connectors, MemoryCore, Store}

  setup do
    original_root = Application.get_env(:optimal_engine, :root_path)

    tmp_dir =
      Path.join(System.tmp_dir!(), "connector-asset-ingest-#{System.unique_integer([:positive])}")

    Application.put_env(:optimal_engine, :root_path, tmp_dir)

    on_exit(fn ->
      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    %{tmp_dir: tmp_dir}
  end

  test "preserves connector base64 attachments as governed assets", %{tmp_dir: tmp_dir} do
    workspace_id = "connector-assets-#{System.unique_integer([:positive])}"

    payload = %{
      "attachments" => [
        %{
          "id" => "file-1",
          "filename" => "voice-note.wav",
          "content_type" => "audio/wav",
          "content_base64" => Base.encode64("RIFF....WAVE connector audio"),
          "metadata" => %{"channel" => "ops"}
        }
      ]
    }

    assert {:ok, %{assets: [%{asset: asset, source_package: source_package}], errors: []}} =
             Connectors.preserve_payload_assets(:slack, "msg-123", payload,
               connector_id: "slack-main",
               workspace_id: workspace_id,
               actor_id: "system:slack-connector",
               security_labels: ["internal"],
               partition_ids: ["ops"]
             )

    assert asset.workspace_id == workspace_id
    assert asset.modality == "audio"
    assert asset.source_package_id == source_package.id
    assert asset.security_labels == ["internal"]
    assert asset.partition_ids == ["ops"]
    assert asset.metadata["connector_kind"] == "slack"
    assert asset.metadata["connector_id"] == "slack-main"
    assert asset.metadata["external_id"] == "msg-123"
    assert asset.metadata["attachment_id"] == "file-1"
    assert asset.metadata["channel"] == "ops"
    assert File.exists?(asset.storage_path)
    assert String.starts_with?(asset.storage_path, tmp_dir)

    assert source_package.source_type == "connector_file"
    assert source_package.source_system == "slack"
    assert source_package.source_uri == "optimal://connectors/slack/msg-123/attachments/file-1"
    assert source_package.verbatim_archive_uri == asset.storage_path

    assert {:ok, stored_asset} = MemoryCore.get_asset(asset.id, workspace_id: workspace_id)
    assert stored_asset.source_package_id == source_package.id

    assert {:ok, [[ledger_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'asset_store.preserve_file'
                 AND output_object_links LIKE ?2
               """,
               [workspace_id, "%#{source_package.id}%"]
             )

    assert ledger_count >= 1
  end

  test "preserves connector path attachments and returns per-attachment errors", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "deck.pdf")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "%PDF-1.7\nconnector attachment")

    workspace_id = "connector-assets-#{System.unique_integer([:positive])}"

    payload = %{
      files: [
        %{file_id: "deck-1", name: "deck.pdf", path: source_path},
        %{file_id: "missing", name: "missing.pdf", path: Path.join(tmp_dir, "missing.pdf")}
      ]
    }

    assert {:ok, %{assets: [%{asset: asset}], errors: [%{filename: "missing.pdf"} = error]}} =
             Connectors.preserve_payload_assets("gmail", "message-9", payload,
               workspace_id: workspace_id,
               partition_ids: ["mail"]
             )

    assert asset.workspace_id == workspace_id
    assert asset.modality == "binary"
    assert asset.metadata["connector_kind"] == "gmail"
    assert asset.metadata["attachment_id"] == "deck-1"
    assert error.reason == "path_not_found"
  end

  test "payloads without attachments are a no-op" do
    assert {:ok, %{assets: [], errors: []}} =
             Connectors.preserve_payload_assets(:github, "issue-1", %{"body" => "no files"})
  end
end
