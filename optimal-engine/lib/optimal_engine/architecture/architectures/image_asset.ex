defmodule OptimalEngine.Architecture.Architectures.ImageAsset do
  @moduledoc """
  A single still image plus its caption, tags, OCR, and visual embedding.

  The primary modality is `:image`; the embedder here is CLIP-family
  (shared vector space with text so retrieval can hop modalities).
  OCR'd text is stored as a secondary `:text` field so FTS can find
  typography the visual model encodes coarsely.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "image_asset",
      version: 1,
      description: "Still image + caption + OCR + visual embedding",
      modality_primary: :image,
      granularity: [:image, :region],
      fields: [
        %Field{
          name: :image,
          modality: :image,
          required: true,
          dims: [:any, :any, 3],
          processor: :image_embedder,
          description: "Source image (path, URL, or raw bytes)"
        },
        %Field{
          name: :caption,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "Human or model-generated caption"
        },
        %Field{
          name: :ocr_text,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "OCR-extracted text (searchable via FTS)"
        },
        %Field{
          name: :tags,
          modality: :structured,
          required: false,
          description: "Classification tags from a vision model"
        },
        %Field{
          name: :exif,
          modality: :structured,
          required: false,
          description: "Captured metadata (camera, geo, timestamp)"
        }
      ]
    )
  end
end
