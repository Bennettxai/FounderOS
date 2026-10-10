defmodule OptimalEngine.Embed.Provider do
  @moduledoc """
  Behaviour for embedding providers.

  An embedding provider produces a fixed-dimension vector from content of a
  given modality. All providers implementing this behaviour target the same
  768-dim aligned space so the engine's retrieval layer can treat text,
  image, and audio embeddings uniformly.

  Built-in providers:

    * `OptimalEngine.Embed.Ollama`   — text (nomic-embed-text) + image (nomic-embed-vision)
    * `OptimalEngine.Embed.Whisper`  — audio → transcript → text embed

  Future providers (Cohere, Voyage, OpenAI, etc.) can be plugged in here
  without touching `OptimalEngine.Pipeline.Embedder`.
  """

  @type modality :: :text | :image | :audio | :video | :code | :data | :mixed
  @type content :: String.t() | binary()
  @type vector :: [float()]
  @type error :: atom() | term()

  @doc "Embed the given content at the given modality. Returns a 768-dim vector."
  @callback embed(content(), modality(), keyword()) :: {:ok, vector()} | {:error, error()}

  @doc "Returns `true` if the provider is reachable + ready."
  @callback available?() :: boolean()

  @doc "Returns the model identifier used for the given modality."
  @callback model_for(modality()) :: String.t() | nil

  @optional_callbacks model_for: 1
end
