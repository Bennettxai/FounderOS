defmodule OptimalEngine.Pipeline.MultimodalToolRegistryTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.MultimodalToolRegistry

  test "registers the open-source multimodal capability map" do
    ids = MultimodalToolRegistry.tools() |> Enum.map(& &1.id)

    assert :docling in ids
    assert :marker in ids
    assert :olmocr in ids
    assert :whisper_cpp in ids
    assert :ffmpeg in ids
    assert :qwen_vl in ids
    assert :colpali in ids
    assert :open_clip in ids
    assert :imagebind in ids
  end

  test "groups tools by role and modality" do
    document_tools = MultimodalToolRegistry.by_role(:document_intelligence)
    audio_tools = MultimodalToolRegistry.by_role(:audio_transcription)
    image_tools = MultimodalToolRegistry.for_modality(:image)

    assert Enum.any?(document_tools, &(&1.id == :docling))
    assert Enum.any?(document_tools, &(&1.id == :marker))
    assert Enum.any?(audio_tools, &(&1.id == :whisper_cpp))
    assert Enum.any?(image_tools, &(&1.id == :qwen_vl))
    assert Enum.any?(image_tools, &(&1.id == :open_clip))
  end

  test "resolves adapter ids from JSON-safe strings" do
    assert %{id: :openai_whisper} = MultimodalToolRegistry.get("openai_whisper")
    assert %{id: :openai_whisper} = MultimodalToolRegistry.get("openai-whisper")
    assert %{id: :openai_whisper} = MultimodalToolRegistry.get("OpenAI Whisper")
    assert is_nil(MultimodalToolRegistry.get("not-a-real-adapter"))
  end

  test "returns recommended pipelines by modality" do
    assert MultimodalToolRegistry.recommended_tool_ids(:document) == [
             :docling,
             :marker,
             :olmocr,
             :unstructured,
             :tesseract,
             :colpali
           ]

    assert MultimodalToolRegistry.recommended_tool_ids(:audio) == [
             :docling,
             :whisper_cpp,
             :openai_whisper,
             :imagebind
           ]

    assert MultimodalToolRegistry.recommended_tool_ids(:video) == [
             :ffmpeg,
             :qwen_vl,
             :whisper_cpp,
             :openai_whisper,
             :imagebind
           ]
  end

  test "exposes adapter profiles for runtime execution and review policy" do
    assert %{
             primary_role: :document_intelligence,
             output_formats: output_formats,
             claimable_output: true,
             args_template: ["{asset_path}"],
             install_profile: :python_cli
           } = MultimodalToolRegistry.profile(:docling)

    assert :json in output_formats
    assert :markdown in output_formats
    assert MultimodalToolRegistry.claimable_output?(:docling)
    refute MultimodalToolRegistry.claimable_output?(:ffmpeg)
    refute MultimodalToolRegistry.claimable_output?(:colpali)
  end

  test "builds default adapter args from registry profiles" do
    assert MultimodalToolRegistry.default_args(:docling, "/tmp/source.pdf") == ["/tmp/source.pdf"]

    assert MultimodalToolRegistry.default_args(:tesseract, "/tmp/page.png") == [
             "/tmp/page.png",
             "stdout"
           ]

    assert MultimodalToolRegistry.default_args(:ffmpeg, "/tmp/source.mp4") == [
             "-i",
             "/tmp/source.mp4",
             "-f",
             "null",
             "-"
           ]
  end

  test "availability reports do not require tools to be installed" do
    availability = MultimodalToolRegistry.availability()

    assert %{available: available, command: "docling"} = availability.docling
    assert is_boolean(available)
    assert %{available: false, command: nil} = availability.qwen_vl
  end
end
