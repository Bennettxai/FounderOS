defmodule OptimalEngine.Architecture.Architectures.MultimodalMedia do
  @moduledoc """
  A composite media object — video with synchronized transcript,
  thumbnail track, and scene-graph captions.

  This is the architecture that shows off the engine's
  **multi-modal alignment** promise: the same semantic query ("the
  moment Ed committed to $2K pricing") hits the transcript
  embedder, the visual-scene embedder, and the audio embedder in
  one shared vector space.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "multimodal_media",
      version: 1,
      description: "Video + transcript + thumbnail track + scene captions",
      modality_primary: :video,
      granularity: [:clip, :scene, :shot, :frame],
      fields: [
        %Field{
          name: :video,
          modality: :video,
          required: true,
          processor: :video_embedder,
          description: "Source video file"
        },
        %Field{
          name: :transcript,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "Spoken-word transcript (time-aligned)"
        },
        %Field{
          name: :scenes,
          modality: :structured,
          required: false,
          description: "[%{start_s, end_s, caption}]"
        },
        %Field{
          name: :thumbnails,
          modality: :image,
          required: false,
          processor: :image_embedder,
          description: "Representative frames at scene boundaries"
        },
        %Field{
          name: :audio_track,
          modality: :audio,
          required: false,
          processor: :audio_embedder,
          description: "Audio-only track for the audio embedder"
        },
        %Field{
          name: :duration_s,
          modality: :structured,
          required: false,
          description: "Total duration in seconds"
        }
      ]
    )
  end
end
