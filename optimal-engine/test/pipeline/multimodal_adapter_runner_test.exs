defmodule OptimalEngine.Pipeline.MultimodalAdapterRunnerTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{MemoryCore, Store}

  setup do
    original_root = Application.get_env(:optimal_engine, :root_path)

    tmp_dir =
      Path.join(
        System.tmp_dir!(),
        "multimodal-adapter-runner-#{System.unique_integer([:positive])}"
      )

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

  test "executes a configured adapter command and records a completed run", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "doc.pdf", "%PDF-1.7\nrunner-test")

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :docling,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: ["Docling extracted text"],
               model_version: "test",
               metadata: %{test_case: "completed"}
             )

    assert run.status == "completed"
    assert run.adapter_id == "docling"
    assert run.output_text == "Docling extracted text"
    assert run.output_hash =~ "sha256:"
    assert run.metadata["command"] == "printf"
    assert run.metadata["args"] == ["Docling extracted text"]
    assert run.metadata["test_case"] == "completed"

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM asset_adapter_runs
               WHERE workspace_id = ?1
                 AND asset_id = ?2
                 AND status = 'completed'
               """,
               ["runner-workspace", asset.id]
             )

    assert {:ok, [extraction]} =
             MemoryCore.list_asset_extractions(asset.id, workspace_id: "runner-workspace")

    assert extraction.adapter_run_id == run.id
    assert extraction.extraction_type == "ocr_span"
    assert extraction.content_text == "Docling extracted text"
    assert extraction.metadata["auto_projected"] == true

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM asset_ocr_spans
               WHERE workspace_id = ?1
                 AND asset_id = ?2
                 AND adapter_run_id = ?3
               """,
               ["runner-workspace", asset.id, run.id]
             )
  end

  test "can disable automatic extraction projection", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "manual.pdf", "%PDF-1.7\nmanual-projection-test")

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :docling,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: ["Manual extraction only"],
               auto_extract: false
             )

    assert run.status == "completed"

    assert {:ok, []} =
             MemoryCore.list_asset_extractions(asset.id, workspace_id: "runner-workspace")
  end

  test "projects transcript JSON into segment-level transcript rows", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "segments.wav", "RIFF....WAVEsegments")

    transcript_json =
      Jason.encode!(%{
        language: "en",
        segments: [
          %{id: 1, start: 0.0, end: 1.25, speaker: "speaker-a", text: "First segment"},
          %{id: 2, start: 1.25, end: 2.0, speaker: "speaker-b", text: "Second segment"}
        ]
      })

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :openai_whisper,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: [transcript_json],
               adapter_role: :audio_transcription,
               confidence: 0.88,
               precision: 0.77
             )

    assert run.status == "completed"

    assert {:ok, extractions} =
             MemoryCore.list_asset_extractions(asset.id, workspace_id: "runner-workspace")

    assert Enum.count(extractions, &(&1.extraction_type == "transcript")) == 2
    assert Enum.any?(extractions, &(&1.content_text == "First segment"))
    assert Enum.any?(extractions, &(&1.content_text == "Second segment"))

    assert {:ok, rows} =
             Store.raw_query(
               """
               SELECT transcript_text, language, speaker, start_ms, end_ms
               FROM asset_transcripts
               WHERE workspace_id = ?1 AND asset_id = ?2
               ORDER BY start_ms ASC
               """,
               ["runner-workspace", asset.id]
             )

    assert [
             ["First segment", "en", "speaker-a", 0, 1250],
             ["Second segment", "en", "speaker-b", 1250, 2000]
           ] = rows
  end

  test "projects document JSON into page-level OCR spans", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "pages.pdf", "%PDF-1.7\npage-json-test")

    page_json =
      Jason.encode!(%{
        pages: [
          %{page_number: 1, text: "Page one extracted text", bbox: %{x: 1, y: 2}},
          %{page_number: 2, text: "Page two extracted text", bbox: %{x: 3, y: 4}}
        ]
      })

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :docling,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: [page_json],
               adapter_role: :document_intelligence
             )

    assert run.status == "completed"

    assert {:ok, extractions} =
             MemoryCore.list_asset_extractions(asset.id, workspace_id: "runner-workspace")

    assert Enum.count(extractions, &(&1.extraction_type == "ocr_span")) == 2

    assert {:ok, rows} =
             Store.raw_query(
               """
               SELECT page_number, span_text, bbox
               FROM asset_ocr_spans
               WHERE workspace_id = ?1 AND asset_id = ?2
               ORDER BY page_number ASC
               """,
               ["runner-workspace", asset.id]
             )

    assert [[1, "Page one extracted text", bbox_1], [2, "Page two extracted text", bbox_2]] = rows
    assert Jason.decode!(bbox_1) == %{"x" => 1, "y" => 2}
    assert Jason.decode!(bbox_2) == %{"x" => 3, "y" => 4}
  end

  test "projects nested document elements and tables into OCR spans", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "structured.pdf", "%PDF-1.7\nstructured-doc-test")

    structured_json =
      Jason.encode!(%{
        pages: [
          %{
            page_number: 1,
            text: "Page summary",
            elements: [
              %{id: "h1", type: "heading", text: "Executive Summary", bbox: %{x: 1, y: 2}},
              %{id: "p1", type: "paragraph", text: "Revenue improved."}
            ],
            tables: [
              %{
                id: "t1",
                rows: [
                  ["Metric", "Value"],
                  ["ARR", "$1M"]
                ],
                bbox: %{x: 10, y: 20}
              }
            ]
          }
        ]
      })

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :docling,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: [structured_json],
               adapter_role: :document_intelligence
             )

    assert run.status == "completed"

    assert {:ok, rows} =
             Store.raw_query(
               """
               SELECT page_number, span_text, metadata
               FROM asset_ocr_spans
               WHERE workspace_id = ?1 AND asset_id = ?2
               ORDER BY span_text ASC
               """,
               ["runner-workspace", asset.id]
             )

    texts = Enum.map(rows, fn [_page, text, _metadata] -> text end)
    assert "Executive Summary" in texts
    assert "Page summary" in texts
    assert "| Metric | Value |\n| --- | --- |\n| ARR | $1M |" in texts

    table_row = Enum.find(rows, fn [_page, text, _metadata] -> String.contains?(text, "ARR") end)
    assert [1, _table_text, table_metadata_json] = table_row
    table_metadata = Jason.decode!(table_metadata_json)
    assert table_metadata["element_type"] == "table"
    assert table_metadata["table_id"] == "t1"
    assert table_metadata["row_count"] == 2
    assert table_metadata["column_count"] == 2
  end

  test "projects nested video frame observations and detections", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "frames.mp4", "ftypmp4 frame-json-test")

    frame_json =
      Jason.encode!(%{
        frames: [
          %{
            id: "frame-1",
            time: 1.5,
            caption: "Operator opens the dashboard.",
            detections: [
              %{id: "det-1", label: "dashboard", confidence: 0.91, bbox: %{x: 5, y: 6}}
            ],
            observations: [
              %{id: "obs-1", type: "risk", text: "Warning banner visible."}
            ]
          }
        ]
      })

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :ffmpeg,
               workspace_id: "runner-workspace",
               actor_id: "user:runner",
               command: "printf",
               args: [frame_json],
               adapter_role: :visual_reasoning
             )

    assert run.status == "completed"

    assert {:ok, rows} =
             Store.raw_query(
               """
               SELECT observation_type, observation_text, frame_time_ms, region, metadata
               FROM asset_visual_observations
               WHERE workspace_id = ?1 AND asset_id = ?2
               ORDER BY observation_text ASC
               """,
               ["runner-workspace", asset.id]
             )

    texts = Enum.map(rows, fn [_type, text, _time, _region, _metadata] -> text end)
    assert "Operator opens the dashboard." in texts
    assert "Warning banner visible." in texts
    assert "dashboard" in texts

    detection_row =
      Enum.find(rows, fn [_type, text, _time, _region, _metadata] -> text == "dashboard" end)

    assert ["object_detection", "dashboard", 1500, region_json, metadata_json] = detection_row
    assert Jason.decode!(region_json) == %{"x" => 5, "y" => 6}

    metadata = Jason.decode!(metadata_json)
    assert metadata["observation_id"] == "det-1"
    assert metadata["object_label"] == "dashboard"
    assert metadata["object_confidence"] == 0.91
  end

  test "records unavailable run when adapter command is not configured or missing", %{
    tmp_dir: tmp_dir
  } do
    asset = store_test_asset!(tmp_dir, "image.png", "\x89PNG\r\n\x1a\nrunner-test")

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :qwen_vl,
               workspace_id: "runner-workspace",
               actor_id: "user:runner"
             )

    assert run.status == "unavailable"
    assert run.adapter_id == "qwen_vl"
    assert run.error_reason == "adapter has no local command configured"

    assert {:ok, [stored_run]} =
             MemoryCore.list_asset_adapter_runs(asset.id, workspace_id: "runner-workspace")

    assert stored_run.id == run.id
    assert stored_run.derivation_ledger_id =~ "dl_"
  end

  test "unknown adapter is rejected before an adapter run is written", %{tmp_dir: tmp_dir} do
    asset = store_test_asset!(tmp_dir, "unknown.bin", "runner-test")

    assert {:error, {:unknown_adapter, :not_a_real_adapter}} =
             MemoryCore.run_asset_adapter(asset.id, :not_a_real_adapter,
               workspace_id: "runner-workspace"
             )

    assert {:ok, []} =
             MemoryCore.list_asset_adapter_runs(asset.id, workspace_id: "runner-workspace")
  end

  defp store_test_asset!(tmp_dir, filename, bytes) do
    path = Path.join(tmp_dir, filename)
    File.mkdir_p!(tmp_dir)
    File.write!(path, bytes)

    {:ok, %{asset: asset}} =
      MemoryCore.store_asset_file(path,
        workspace_id: "runner-workspace",
        actor_id: "user:runner",
        security_labels: ["internal"],
        partition_ids: ["runner-assets"]
      )

    asset
  end
end
