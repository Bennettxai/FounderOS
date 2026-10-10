defmodule OptimalEngine.Architecture.Architectures.TextSignal do
  @moduledoc """
  The engine's default architecture: a free-text signal with metadata.

  Matches most of what humans type into the engine — notes, transcripts,
  specs, briefs, chat messages. Decomposes at 4 scales (document /
  section / paragraph / sentence) and uses a text-embedder on the body.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "text_signal",
      version: 1,
      description: "Free-text signal (note, transcript, doc, message)",
      modality_primary: :text,
      granularity: [:document, :section, :paragraph, :sentence],
      fields: [
        %Field{name: :title, modality: :text, required: true, description: "Short display title"},
        %Field{
          name: :body,
          modality: :text,
          required: true,
          processor: :text_embedder,
          description: "Primary textual content"
        },
        %Field{
          name: :genre,
          modality: :structured,
          required: false,
          description: "Speech-act genre (brief, spec, note, transcript, …)"
        },
        %Field{
          name: :authored_at,
          modality: :structured,
          required: false,
          description: "ISO-8601 timestamp"
        }
      ]
    )
  end
end
