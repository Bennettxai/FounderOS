defmodule OptimalEngine.MemoryCore.AssetStoreTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{MemoryCore, Store}

  setup do
    original_root = Application.get_env(:optimal_engine, :root_path)
    tmp_dir = Path.join(System.tmp_dir!(), "asset-store-#{System.unique_integer([:positive])}")
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

  test "stores a raw file as governed asset evidence", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "input.png")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "\x89PNG\r\n\x1a\nasset-store-test")

    assert {:ok, %{asset: asset, source_package: source_package}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: "default",
               actor_id: "user:asset-reviewer",
               trust_label: "unreviewed",
               security_labels: ["internal"],
               partition_ids: ["media"],
               metadata: %{purpose: "asset-store-test"}
             )

    assert asset.id =~ "asset_"
    assert asset.workspace_id == "default"
    assert asset.modality == "image"
    assert asset.content_hash =~ "sha256:"
    assert asset.source_package_id == source_package.id
    assert asset.trust_label == "unreviewed"
    assert asset.security_labels == ["internal"]
    assert asset.partition_ids == ["media"]
    assert asset.metadata["purpose"] == "asset-store-test"
    assert File.exists?(asset.storage_path)
    assert asset.storage_path != source_path

    assert source_package.source_class == "image"
    assert source_package.verbatim_archive_uri == asset.storage_path
    assert source_package.metadata.asset_id == asset.id
    assert source_package.metadata.content_hash == asset.content_hash

    assert {:ok, fetched} = MemoryCore.get_asset(asset.id, workspace_id: "default")
    assert fetched.id == asset.id
    assert fetched.source_package_id == source_package.id

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM source_packages WHERE id = ?1 AND workspace_id = ?2",
               [source_package.id, "default"]
             )

    assert {:ok, [[ledger_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'asset_store.preserve_file'
                 AND output_object_links LIKE ?2
               """,
               ["default", "%#{asset.id}%"]
             )

    assert ledger_count >= 1
  end

  test "same bytes in different workspaces keep separate asset identities", %{tmp_dir: tmp_dir} do
    source_path = Path.join(tmp_dir, "shared.wav")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "RIFF....WAVEasset-store-test")

    assert {:ok, %{asset: default_asset}} =
             MemoryCore.store_asset_file(source_path, workspace_id: "default")

    assert {:ok, %{asset: other_asset}} =
             MemoryCore.store_asset_file(source_path, workspace_id: "second-workspace")

    assert default_asset.content_hash == other_asset.content_hash
    assert default_asset.id != other_asset.id
    assert default_asset.workspace_id == "default"
    assert other_asset.workspace_id == "second-workspace"
  end

  test "records multimodal adapter runs as derived asset artifacts", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("adapter-workspace")
    source_path = Path.join(tmp_dir, "document.pdf")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "%PDF-1.7\nasset-adapter-run-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               security_labels: ["internal"],
               partition_ids: ["documents"]
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :docling,
               adapter_role: :document_intelligence,
               status: "completed",
               output_text: "Extracted document text",
               model_id: "docling",
               model_version: "local",
               confidence: 0.82,
               precision: 0.77,
               metadata: %{pages: 1}
             )

    assert run.id =~ "aar_"
    assert run.asset_id == asset.id
    assert run.adapter_id == "docling"
    assert run.adapter_role == "document_intelligence"
    assert run.status == "completed"
    assert run.output_hash =~ "sha256:"

    assert {:ok, [stored_run]} =
             MemoryCore.list_asset_adapter_runs(asset.id, workspace_id: workspace_id)

    assert stored_run.id == run.id
    assert stored_run.derivation_ledger_id =~ "dl_"
    assert stored_run.metadata["pages"] == 1
    assert stored_run.security_labels == ["internal"]
    assert stored_run.partition_ids == ["documents"]

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'asset_adapter.docling'
                 AND output_object_links LIKE ?2
               """,
               [workspace_id, "%#{run.id}%"]
             )
  end

  test "turns a completed adapter run into a pending claim", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("adapter-claim-workspace")
    source_path = Path.join(tmp_dir, "claimable.pdf")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "%PDF-1.7\nasset-adapter-claim-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               security_labels: ["internal"],
               partition_ids: ["documents"]
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :docling,
               adapter_role: :document_intelligence,
               status: "completed",
               output_text: "Adapter found an approved renewal date.",
               model_id: "docling",
               confidence: 0.72,
               precision: 0.68
             )

    assert {:ok, %{source_package: source_package, pending_claim: claim}} =
             MemoryCore.claim_from_asset_adapter_run(run.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               subject_anchor: "renewal",
               action_class: "found_date"
             )

    assert source_package.source_type == "adapter_output"
    assert source_package.source_system == "docling"
    assert source_package.metadata.adapter_run_id == run.id

    assert claim.claim_type == "adapter_output"
    assert claim.claim_text == "Adapter found an approved renewal date."
    assert claim.review_status == "unreviewed"
    assert claim.lifecycle_state == "pending"
    assert claim.source_package_id == source_package.id
    assert claim.aggregate_confidence == 0.72
    assert claim.aggregate_precision == 0.68

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM claims
               WHERE workspace_id = ?1
                 AND extraction_run_id = ?2
                 AND review_status = 'unreviewed'
               """,
               [workspace_id, run.id]
             )
  end

  test "does not create a claim from unavailable adapter output", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("adapter-claim-workspace")
    source_path = Path.join(tmp_dir, "unavailable.pdf")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "%PDF-1.7\nasset-adapter-unavailable-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter"
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :docling,
               status: "unavailable",
               error_reason: "command not found"
             )

    assert {:error, {:adapter_run_not_completed, "unavailable"}} =
             MemoryCore.claim_from_asset_adapter_run(run.id,
               workspace_id: workspace_id
             )
  end

  test "stores typed extraction projections from a completed adapter run", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("asset-extraction-workspace")
    source_path = Path.join(tmp_dir, "media.mp4")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "ftypmp42asset-extraction-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               security_labels: ["internal"],
               partition_ids: ["media"]
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :ffmpeg,
               adapter_role: :video_processing,
               status: "completed",
               output_text: "adapter produced structured outputs",
               model_id: "ffmpeg",
               confidence: 0.7,
               precision: 0.66
             )

    assert {:ok, %{extraction: transcript}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :transcript,
               content_text: "Speaker said launch is approved.",
               language: "en",
               speaker: "speaker-1",
               start_ms: 0,
               end_ms: 1200
             )

    assert transcript.extraction_type == "transcript"
    assert transcript.content_hash =~ "sha256:"

    assert {:ok, %{extraction: ocr_span}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :ocr_span,
               content_text: "APPROVED",
               page_number: 2,
               bbox: %{x: 10, y: 20, w: 100, h: 40}
             )

    assert ocr_span.extraction_type == "ocr_span"

    assert {:ok, %{extraction: visual_observation}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :visual_observation,
               content_text: "Dashboard shows green deployment status.",
               observation_type: "state",
               region: %{x: 0, y: 0, w: 640, h: 360},
               frame_time_ms: 5000
             )

    assert visual_observation.extraction_type == "visual_observation"

    assert {:ok, %{extraction: embedding_ref}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :embedding_ref,
               content_text: "",
               content_ref: "optimal://embeddings/media/frame-1",
               embedding_model_id: "openclip",
               embedding_model_version: "test",
               embedding_ref: "optimal://embeddings/media/frame-1",
               embedding_dim: 768,
               embedding_space: "image_text",
               target_ref: "asset:#{asset.id}"
             )

    assert embedding_ref.extraction_type == "embedding_ref"

    assert {:ok, extractions} =
             MemoryCore.list_asset_extractions(asset.id,
               workspace_id: workspace_id
             )

    assert Enum.map(extractions, & &1.extraction_type) |> Enum.sort() ==
             ["embedding_ref", "ocr_span", "transcript", "visual_observation"]

    for table <- [
          "asset_extractions",
          "asset_transcripts",
          "asset_ocr_spans",
          "asset_visual_observations",
          "asset_embedding_refs"
        ] do
      assert {:ok, [[count]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM #{table} WHERE workspace_id = ?1 AND asset_id = ?2",
                 [workspace_id, asset.id]
               )

      assert count >= 1
    end
  end

  test "turns a text extraction into a pending claim", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("asset-extraction-claim-workspace")
    source_path = Path.join(tmp_dir, "audio.wav")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "RIFF....WAVEasset-extraction-claim-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               partition_ids: ["audio"]
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :openai_whisper,
               adapter_role: :audio_transcription,
               status: "completed",
               output_text: "transcript complete",
               confidence: 0.8,
               precision: 0.75
             )

    assert {:ok, %{extraction: extraction}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :transcript,
               content_text: "The client approved the launch window.",
               speaker: "client",
               start_ms: 250,
               end_ms: 1600
             )

    assert {:ok, %{source_package: source_package, pending_claim: claim}} =
             MemoryCore.claim_from_asset_extraction(extraction.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               subject_anchor: "client",
               action_class: "approved_launch_window"
             )

    assert source_package.source_type == "asset_extraction"
    assert source_package.metadata.extraction_id == extraction.id
    assert claim.claim_type == "transcript"
    assert claim.claim_text == "The client approved the launch window."
    assert claim.review_status == "unreviewed"
    assert claim.lifecycle_state == "pending"
    assert claim.extraction_run_id == extraction.id
  end

  test "stores reference-only embedding extractions from completed adapter runs", %{
    tmp_dir: tmp_dir
  } do
    workspace_id = unique_workspace("asset-extraction-ref-workspace")
    source_path = Path.join(tmp_dir, "image.png")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, <<137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 0>>)

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               partition_ids: ["image"]
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :open_clip,
               adapter_role: :image_embedding,
               status: "completed",
               output_text: "",
               output_ref: "optimal://adapter-runs/openclip/frame-1",
               confidence: 0.72,
               precision: 0.7
             )

    assert {:ok, %{extraction: extraction}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               extraction_type: :embedding_ref,
               content_text: "",
               content_ref: "optimal://embeddings/image-1",
               embedding_model_id: "openclip",
               embedding_model_version: "test",
               embedding_dim: 768,
               embedding_space: "image_text",
               target_ref: "asset:#{asset.id}"
             )

    assert extraction.extraction_type == "embedding_ref"
    assert extraction.content_text == ""
    assert extraction.content_ref == "optimal://embeddings/image-1"

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM asset_embedding_refs WHERE workspace_id = ?1 AND asset_id = ?2",
               [workspace_id, asset.id]
             )

    assert {:error, :asset_extraction_empty} =
             MemoryCore.claim_from_asset_extraction(extraction.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             )
  end

  test "does not create extraction projections from unavailable adapter runs", %{tmp_dir: tmp_dir} do
    workspace_id = unique_workspace("asset-extraction-failed-workspace")
    source_path = Path.join(tmp_dir, "failed.mp3")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "ID3asset-extraction-failed-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter"
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :whisper_cpp,
               status: "unavailable",
               error_reason: "command not found"
             )

    assert {:error, {:adapter_run_not_completed, "unavailable"}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               extraction_type: :transcript,
               content_text: "should not store"
             )
  end

  test "does not leave generic extraction rows for unsupported extraction types", %{
    tmp_dir: tmp_dir
  } do
    workspace_id = unique_workspace("asset-extraction-unsupported-workspace")
    source_path = Path.join(tmp_dir, "unsupported.wav")
    File.mkdir_p!(tmp_dir)
    File.write!(source_path, "RIFF....WAVEasset-extraction-unsupported-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:adapter"
             )

    assert {:ok, run} =
             MemoryCore.record_asset_adapter_run(asset.id,
               workspace_id: workspace_id,
               actor_id: "user:adapter",
               adapter_id: :openai_whisper,
               status: "completed",
               output_text: "adapter complete"
             )

    assert {:error, {:unsupported_extraction_type, "unknown_projection"}} =
             MemoryCore.record_asset_extraction(run.id,
               workspace_id: workspace_id,
               extraction_type: :unknown_projection,
               content_text: "should not store"
             )

    assert {:ok, [[0]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM asset_extractions WHERE workspace_id = ?1 AND asset_id = ?2",
               [workspace_id, asset.id]
             )
  end

  defp unique_workspace(prefix) do
    "#{prefix}-#{System.unique_integer([:positive])}"
  end
end
