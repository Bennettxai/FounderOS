defmodule OptimalEngine.Pipeline.AssetStoreIntegrationTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{MemoryCore, Pipeline, Store}

  setup do
    original_root = Application.get_env(:optimal_engine, :root_path)

    tmp_dir =
      Path.join(System.tmp_dir!(), "pipeline-asset-store-#{System.unique_integer([:positive])}")

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

  test "pipeline preserves parser assets before decomposition", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "diagram.png")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "\x89PNG\r\n\x1a\npipeline-asset-store-test")

    assert {:ok, result} =
             Pipeline.run(source_path,
               workspace_id: "pipeline-assets",
               actor_id: "user:pipeline",
               security_labels: ["internal"],
               partition_ids: ["visual"],
               skip_vlm: true,
               skip_embed: true
             )

    [asset] = result.parsed.assets
    asset_id = asset.metadata.asset_id
    source_package_id = asset.metadata.source_package_id

    assert is_binary(asset_id)
    assert is_binary(source_package_id)
    assert asset.path != source_path
    assert File.exists?(asset.path)
    assert result.asset_paths[asset.hash] == asset.path
    assert result.asset_paths[asset_id] == asset.path

    document_chunk = Enum.find(result.tree.chunks, &(&1.scale == :document))
    assert document_chunk.asset_ref == asset.hash

    assert {:ok, stored_asset} = MemoryCore.get_asset(asset_id, workspace_id: "pipeline-assets")
    assert stored_asset.source_package_id == source_package_id
    assert stored_asset.security_labels == ["internal"]
    assert stored_asset.partition_ids == ["visual"]

    assert {:ok, [[ledger_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'asset_store.preserve_file'
                 AND output_object_links LIKE ?2
               """,
               ["pipeline-assets", "%#{asset_id}%"]
             )

    assert ledger_count >= 1
  end

  test "pipeline can keep parser assets ungoverned for pure parsing runs", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "ungoverned.png")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "\x89PNG\r\n\x1a\npipeline-asset-store-skip")

    assert {:ok, result} =
             Pipeline.run(source_path,
               workspace_id: "pipeline-assets",
               preserve_assets: false,
               skip_vlm: true,
               skip_embed: true
             )

    [asset] = result.parsed.assets
    refute Map.has_key?(asset.metadata, :asset_id)
    assert asset.path == source_path
    assert result.asset_paths[asset.hash] == source_path
  end
end
