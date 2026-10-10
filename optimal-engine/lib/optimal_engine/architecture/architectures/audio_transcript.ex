defmodule OptimalEngine.Architecture.Architectures.AudioTranscript do
  @moduledoc """
  An audio clip plus its transcript, speaker diarization, and aligned
  word-level timings. Voice memos, calls, podcast segments.

  The engine stores the raw audio (for waveform-level models) AND the
  transcript (for text retrieval). Alignment between the two is a
  first-class field so downstream consumers can jump from a sentence
  back to the exact second in the waveform.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "audio_transcript",
      version: 1,
      description: "Audio clip with transcript, diarization, and word-level alignment",
      modality_primary: :audio,
      granularity: [:clip, :utterance, :word],
      fields: [
        %Field{
          name: :audio,
          modality: :audio,
          required: true,
          processor: :audio_embedder,
          description: "Source waveform (path, URL, or raw bytes)"
        },
        %Field{
          name: :transcript,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "Whisper-style transcription"
        },
        %Field{
          name: :speakers,
          modality: :structured,
          required: false,
          description: "Diarization: [%{speaker, start, end}]"
        },
        %Field{
          name: :alignment,
          modality: :time_series,
          required: false,
          description: "Word-level timings: [[word, t_start, t_end]]"
        },
        %Field{
          name: :duration_s,
          modality: :structured,
          required: false,
          description: "Clip duration in seconds"
        }
      ]
    )
  end
end
