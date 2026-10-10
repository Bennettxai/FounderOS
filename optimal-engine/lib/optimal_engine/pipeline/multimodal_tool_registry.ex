defmodule OptimalEngine.Pipeline.MultimodalToolRegistry do
  @moduledoc """
  Registry of open-source multimodal tools the pipeline can target.

  This module does not install or execute heavyweight model stacks. It defines
  the capability map that parser adapters, connectors, retrieval, and deployment
  checks can share.

  Runtime adapters should depend on these capabilities instead of scattering
  tool names across parser modules.
  """

  @type tool_id :: atom()

  @type modality ::
          :document
          | :image
          | :audio
          | :video
          | :table
          | :text
          | :binary

  @type role ::
          :document_intelligence
          | :ocr
          | :audio_transcription
          | :video_processing
          | :visual_reasoning
          | :visual_document_retrieval
          | :multimodal_embedding

  @type output_format ::
          :plain_text
          | :markdown
          | :json
          | :media_metadata
          | :embedding_ref

  @type adapter_profile :: %{
          primary_role: role(),
          output_formats: [output_format()],
          claimable_output: boolean(),
          args_template: [String.t()],
          install_profile: atom()
        }

  @type tool :: %{
          id: tool_id(),
          name: String.t(),
          roles: [role()],
          modalities: [modality()],
          command: String.t() | nil,
          adapter: atom(),
          profile: adapter_profile(),
          source_url: String.t(),
          local_first: boolean(),
          notes: String.t()
        }

  @tools [
    %{
      id: :docling,
      name: "Docling",
      roles: [:document_intelligence, :ocr, :audio_transcription],
      modalities: [:document, :image, :audio, :table, :text],
      command: "docling",
      adapter: :document_router,
      profile: %{
        primary_role: :document_intelligence,
        output_formats: [:json, :markdown, :plain_text],
        claimable_output: true,
        args_template: ["{asset_path}"],
        install_profile: :python_cli
      },
      source_url: "https://github.com/docling-project/docling",
      local_first: true,
      notes:
        "Default document-intelligence target for PDFs, Office files, images, OCR, tables, and audio transcripts."
    },
    %{
      id: :marker,
      name: "Marker",
      roles: [:document_intelligence, :ocr],
      modalities: [:document, :image, :table, :text],
      command: "marker_single",
      adapter: :document_converter,
      profile: %{
        primary_role: :document_intelligence,
        output_formats: [:json, :markdown, :plain_text],
        claimable_output: true,
        args_template: ["{asset_path}"],
        install_profile: :python_cli
      },
      source_url: "https://github.com/datalab-to/marker",
      local_first: true,
      notes:
        "High-accuracy document-to-markdown/JSON target for PDFs, images, Office files, equations, tables, and extracted images."
    },
    %{
      id: :olmocr,
      name: "olmOCR",
      roles: [:document_intelligence, :ocr],
      modalities: [:document, :image, :text],
      command: "olmocr",
      adapter: :document_ocr,
      profile: %{
        primary_role: :ocr,
        output_formats: [:json, :plain_text],
        claimable_output: true,
        args_template: ["{asset_path}"],
        install_profile: :python_cli
      },
      source_url: "https://github.com/allenai/olmocr",
      local_first: true,
      notes:
        "Specialized OCR target for linearizing scanned and image-based documents into readable text."
    },
    %{
      id: :unstructured,
      name: "Unstructured",
      roles: [:document_intelligence],
      modalities: [:document, :image, :table, :text],
      command: nil,
      adapter: :document_partition,
      profile: %{
        primary_role: :document_intelligence,
        output_formats: [:json],
        claimable_output: true,
        args_template: [],
        install_profile: :python_library
      },
      source_url: "https://docs.unstructured.io/open-source/introduction/overview",
      local_first: true,
      notes:
        "Open-source ingestion and pre-processing toolkit for diverse file formats; useful as an adapter target, with production limits documented by the project."
    },
    %{
      id: :tesseract,
      name: "Tesseract OCR",
      roles: [:ocr],
      modalities: [:image, :document, :text],
      command: "tesseract",
      adapter: :ocr_fallback,
      profile: %{
        primary_role: :ocr,
        output_formats: [:plain_text],
        claimable_output: true,
        args_template: ["{asset_path}", "stdout"],
        install_profile: :system_package
      },
      source_url: "https://github.com/tesseract-ocr/tesseract",
      local_first: true,
      notes: "Conservative OCR fallback when document-intelligence models are unavailable."
    },
    %{
      id: :whisper_cpp,
      name: "whisper.cpp",
      roles: [:audio_transcription],
      modalities: [:audio, :video, :text],
      command: "whisper-cli",
      adapter: :speech_to_text,
      profile: %{
        primary_role: :audio_transcription,
        output_formats: [:json, :plain_text],
        claimable_output: true,
        args_template: ["-f", "{asset_path}"],
        install_profile: :native_cli
      },
      source_url: "https://github.com/ggml-org/whisper.cpp",
      local_first: true,
      notes: "Local C/C++ Whisper runtime target for speech transcription."
    },
    %{
      id: :openai_whisper,
      name: "Whisper",
      roles: [:audio_transcription],
      modalities: [:audio, :video, :text],
      command: "whisper",
      adapter: :speech_to_text,
      profile: %{
        primary_role: :audio_transcription,
        output_formats: [:json, :plain_text],
        claimable_output: true,
        args_template: ["{asset_path}"],
        install_profile: :python_cli
      },
      source_url: "https://github.com/openai/whisper",
      local_first: true,
      notes: "Reference open-source speech-recognition model target."
    },
    %{
      id: :ffmpeg,
      name: "FFmpeg",
      roles: [:video_processing],
      modalities: [:video, :audio, :image],
      command: "ffmpeg",
      adapter: :media_demux,
      profile: %{
        primary_role: :video_processing,
        output_formats: [:media_metadata],
        claimable_output: false,
        args_template: ["-i", "{asset_path}", "-f", "null", "-"],
        install_profile: :system_package
      },
      source_url: "https://ffmpeg.org/",
      local_first: true,
      notes:
        "Media demuxing, frame extraction, audio extraction, transcoding, and metadata probing target."
    },
    %{
      id: :qwen_vl,
      name: "Qwen VL",
      roles: [:visual_reasoning],
      modalities: [:image, :video, :document, :text],
      command: nil,
      adapter: :vision_language_model,
      profile: %{
        primary_role: :visual_reasoning,
        output_formats: [:json, :plain_text],
        claimable_output: true,
        args_template: [],
        install_profile: :model_runtime
      },
      source_url: "https://github.com/QwenLM/Qwen3-VL",
      local_first: true,
      notes:
        "Open vision-language model target for image/video/document question answering, visual reasoning, localization, and text reading."
    },
    %{
      id: :colpali,
      name: "ColPali / ColQwen",
      roles: [:visual_document_retrieval],
      modalities: [:document, :image, :text],
      command: nil,
      adapter: :visual_document_retrieval,
      profile: %{
        primary_role: :visual_document_retrieval,
        output_formats: [:embedding_ref],
        claimable_output: false,
        args_template: [],
        install_profile: :model_runtime
      },
      source_url: "https://github.com/illuin-tech/colpali",
      local_first: true,
      notes:
        "Visual document retrieval target for page-image search without text-only extraction as the only path."
    },
    %{
      id: :open_clip,
      name: "OpenCLIP",
      roles: [:multimodal_embedding],
      modalities: [:image, :text],
      command: nil,
      adapter: :image_text_embedding,
      profile: %{
        primary_role: :multimodal_embedding,
        output_formats: [:embedding_ref],
        claimable_output: false,
        args_template: [],
        install_profile: :model_runtime
      },
      source_url: "https://github.com/mlfoundations/open_clip",
      local_first: true,
      notes:
        "Open-source CLIP implementation target for image/text embeddings and zero-shot classification."
    },
    %{
      id: :imagebind,
      name: "ImageBind",
      roles: [:multimodal_embedding],
      modalities: [:image, :text, :audio, :video],
      command: nil,
      adapter: :cross_modal_embedding,
      profile: %{
        primary_role: :multimodal_embedding,
        output_formats: [:embedding_ref],
        claimable_output: false,
        args_template: [],
        install_profile: :model_runtime
      },
      source_url: "https://github.com/facebookresearch/ImageBind",
      local_first: true,
      notes: "Joint-embedding target across image, text, audio, and video-derived signals."
    }
  ]

  @recommended %{
    document: [:docling, :marker, :olmocr, :unstructured, :tesseract, :colpali],
    image: [:docling, :qwen_vl, :tesseract, :open_clip, :imagebind],
    audio: [:docling, :whisper_cpp, :openai_whisper, :imagebind],
    video: [:ffmpeg, :qwen_vl, :whisper_cpp, :openai_whisper, :imagebind],
    table: [:docling, :marker, :unstructured],
    text: [:docling, :marker, :unstructured],
    binary: [:docling]
  }

  @spec tools() :: [tool()]
  def tools, do: @tools

  @spec get(tool_id() | String.t()) :: tool() | nil
  def get(id) when is_atom(id), do: Enum.find(@tools, &(&1.id == id))

  def get(id) when is_binary(id) do
    normalized = normalize_id_string(id)
    Enum.find(@tools, &(normalize_id_string(Atom.to_string(&1.id)) == normalized))
  end

  @spec profile(tool_id()) :: adapter_profile() | nil
  def profile(id) when is_atom(id) do
    case get(id) do
      %{profile: profile} -> profile
      _ -> nil
    end
  end

  @spec default_args(tool_id(), String.t()) :: [String.t()]
  def default_args(id, asset_path) when is_atom(id) and is_binary(asset_path) do
    id
    |> profile()
    |> case do
      %{args_template: template} ->
        Enum.map(template, &String.replace(&1, "{asset_path}", asset_path))

      _ ->
        [asset_path]
    end
  end

  @spec claimable_output?(tool_id()) :: boolean()
  def claimable_output?(id) when is_atom(id) do
    case profile(id) do
      %{claimable_output: value} -> value
      _ -> false
    end
  end

  @spec by_role(role()) :: [tool()]
  def by_role(role) when is_atom(role),
    do: Enum.filter(@tools, &(role in &1.roles))

  @spec for_modality(modality()) :: [tool()]
  def for_modality(modality) when is_atom(modality),
    do: Enum.filter(@tools, &(modality in &1.modalities))

  @spec recommended_pipeline(modality()) :: [tool()]
  def recommended_pipeline(modality) when is_atom(modality) do
    modality
    |> recommended_tool_ids()
    |> Enum.map(&get/1)
    |> Enum.reject(&is_nil/1)
  end

  @spec recommended_tool_ids(modality()) :: [tool_id()]
  def recommended_tool_ids(modality) when is_atom(modality),
    do: Map.get(@recommended, modality, [])

  @spec availability() :: %{tool_id() => %{available: boolean(), command: String.t() | nil}}
  def availability do
    Map.new(@tools, fn tool ->
      {tool.id, %{available: available?(tool.id), command: tool.command}}
    end)
  end

  @spec available?(tool_id()) :: boolean()
  def available?(id) when is_atom(id) do
    case get(id) do
      %{command: command} when is_binary(command) ->
        System.find_executable(command) != nil

      _ ->
        false
    end
  end

  defp normalize_id_string(id) do
    id
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9]+/, "_")
    |> String.trim("_")
  end
end
