defmodule OptimalEngine.Pipeline.Enricher.VlmEnricher do
  @moduledoc """
  Post-parse enrichment via Vision-Language Model.

  Runs between Parse (Stage 2) and Decompose (Stage 3). For any
  `ParsedDoc` with modality `:image` or `:video`, calls the configured
  VLM to generate a rich text description that gets prepended to the
  document's text. This makes downstream stages (decompose, classify,
  embed) work with richer content.

  Graceful degradation: if the VLM is unavailable or the model isn't
  pulled, returns the ParsedDoc unchanged.
  """

  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  require Logger

  @image_prompt """
  Describe this image in detail. Include any text, objects, people, charts, \
  diagrams, layout, and notable visual elements. Be precise and factual.\
  """

  @frame_prompt_template """
  This is frame %{index} at timestamp %{timestamp}s from a video. \
  Describe what you see. Include any text, diagrams, people, objects, \
  and notable visual elements. Be precise and factual.\
  """

  @doc """
  Enrich a ParsedDoc with VLM-generated descriptions.

  Only processes `:image` and `:video` modalities. Returns
  `{:ok, enriched_doc}` always — failures degrade gracefully.
  """
  @spec enrich(ParsedDoc.t(), keyword()) :: {:ok, ParsedDoc.t()}
  def enrich(%ParsedDoc{} = doc, opts \\ []) do
    if vlm_enabled?(opts) do
      do_enrich(doc, opts)
    else
      {:ok, doc}
    end
  end

  defp do_enrich(%ParsedDoc{modality: :image} = doc, opts) do
    case find_image_asset(doc) do
      nil ->
        {:ok, doc}

      asset ->
        case Ollama.vlm_analyze(asset.path, @image_prompt, opts) do
          {:ok, description} ->
            enriched_text = build_enriched_text(description, doc.text, "VLM Description")
            metadata = Map.put(doc.metadata, :ocr_text, doc.text)
            metadata = Map.put(metadata, :vlm_description, description)
            {:ok, %{doc | text: enriched_text, metadata: metadata}}

          {:error, reason} ->
            Logger.debug("VLM enrichment skipped for image: #{inspect(reason)}")
            {:ok, doc}
        end
    end
  end

  defp do_enrich(%ParsedDoc{modality: :video} = doc, opts) do
    frame_assets =
      Enum.filter(doc.assets, fn a ->
        a.modality == :image and is_binary(a.path) and File.exists?(a.path)
      end)

    case frame_assets do
      [] ->
        {:ok, doc}

      frames ->
        frame_descriptions =
          frames
          |> Enum.with_index(1)
          |> Enum.map(fn {asset, idx} ->
            timestamp = get_in(asset.metadata, [:timestamp_s]) || 0.0

            prompt =
              String.replace(@frame_prompt_template, "%{index}", to_string(idx))
              |> String.replace("%{timestamp}", to_string(timestamp))

            case Ollama.vlm_analyze(asset.path, prompt, opts) do
              {:ok, desc} -> {idx, timestamp, desc}
              {:error, _} -> nil
            end
          end)
          |> Enum.reject(&is_nil/1)

        case frame_descriptions do
          [] ->
            {:ok, doc}

          descs ->
            vlm_section =
              descs
              |> Enum.map_join("\n\n", fn {idx, ts, desc} ->
                "### Frame #{idx} (#{ts}s)\n#{desc}"
              end)

            enriched_text = build_enriched_text(vlm_section, doc.text, "Visual Analysis")

            metadata =
              doc.metadata
              |> Map.put(
                :frame_descriptions,
                Enum.map(descs, fn {i, t, d} -> %{index: i, timestamp_s: t, description: d} end)
              )

            {:ok, %{doc | text: enriched_text, metadata: metadata}}
        end
    end
  end

  defp do_enrich(doc, _opts), do: {:ok, doc}

  defp build_enriched_text(vlm_text, original_text, header) do
    parts =
      [
        "## #{header}\n\n#{vlm_text}",
        if(original_text != "" and not is_nil(original_text), do: original_text)
      ]
      |> Enum.reject(&is_nil/1)
      |> Enum.join("\n\n---\n\n")

    parts
  end

  defp find_image_asset(%ParsedDoc{assets: assets}) do
    Enum.find(assets, fn a ->
      a.modality == :image and is_binary(a.path) and File.exists?(a.path)
    end)
  end

  defp vlm_enabled?(opts) do
    if Keyword.get(opts, :skip_vlm, false) do
      false
    else
      Ollama.vlm_available?()
    end
  end
end
