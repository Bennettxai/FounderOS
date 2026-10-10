defmodule OptimalEngine.Pipeline.IndexerAssetStoreTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Pipeline.Indexer, Store}

  setup do
    original_root = Application.get_env(:optimal_engine, :root_path)

    tmp_dir =
      Path.join(System.tmp_dir!(), "indexer-asset-store-#{System.unique_integer([:positive])}")

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

  test "binary indexing preserves assets in the requested workspace", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "indexed-image.png")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "\x89PNG\r\n\x1a\nindexer-asset-store-test")

    assert {:ok, context} =
             Indexer.index_file(source_path,
               workspace_id: "indexed-assets",
               actor_id: "user:indexer",
               security_labels: ["internal"],
               partition_ids: ["indexed-media"],
               skip_vlm: true
             )

    assert context.type == :resource
    assert context.path == source_path

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM assets
               WHERE workspace_id = ?1
                 AND original_path = ?2
                 AND modality = 'image'
               """,
               ["indexed-assets", source_path]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM source_packages
               WHERE workspace_id = ?1
                 AND source_uri = ?2
                 AND source_class = 'image'
               """,
               ["indexed-assets", source_path]
             )
  end
end
