defmodule OptimalEngine.Architecture.Processors.AudioEmbedder do
  @moduledoc """
  Audio embedding — transcribes via Whisper.cpp, then embeds the
  transcript text via `nomic-embed-text` (768-d aligned space).

  Falls back gracefully when whisper.cpp or Ollama is unreachable.
  """

  @behaviour OptimalEngine.Architecture.Processor

  alias OptimalEngine.Embed.{Ollama, Whisper}

  @impl true
  def id, do: :audio_embedder

  @impl true
  def modality, do: :audio

  @impl true
  def emits, do: [:embedding, :transcription]

  @impl true
  def init(_config), do: {:ok, %{}}

  @impl true
  def process(_field, value, _state) do
    with {:ok, path} <- resolve(value),
         {:ok, %{text: text}} when text != "" <- Whisper.transcribe(path),
         {:ok, vector} <- Ollama.embed_text(text) do
      {:ok,
       %{
         kind: :embedding,
         value: vector,
         metadata: %{
           dim: length(vector),
           model: "whisper+nomic-embed-text",
           transcript: text
         }
       }}
    else
      {:error, reason} -> {:error, reason}
      {:ok, %{text: ""}} -> {:error, :empty_transcript}
      other -> {:error, other}
    end
  rescue
    e -> {:error, Exception.message(e)}
  end

  defp resolve(path) when is_binary(path), do: {:ok, path}
  defp resolve(%{path: p}) when is_binary(p), do: {:ok, p}
  defp resolve(_), do: {:error, :unresolvable_audio_reference}
end
