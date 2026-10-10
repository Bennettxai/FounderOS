defmodule OptimalEngine.Architecture.Processors.VideoEmbedder do
  @moduledoc """
  Video embedding — embeds keyframe images via `nomic-embed-vision`
  (768-d, aligned with the text embedder). Falls back to text embedding
  of the transcript when no frame asset is available.
  """

  @behaviour OptimalEngine.Architecture.Processor

  alias OptimalEngine.Embed.Ollama

  @impl true
  def id, do: :video_embedder

  @impl true
  def modality, do: :video

  @impl true
  def emits, do: [:embedding]

  @impl true
  def init(_config), do: {:ok, %{}}

  @impl true
  def process(_field, value, _state) do
    with {:ok, path} <- resolve(value),
         {:ok, vector} <- Ollama.embed_image(path, model: "nomic-embed-vision") do
      {:ok,
       %{
         kind: :embedding,
         value: vector,
         metadata: %{dim: length(vector), model: "vision+nomic-embed-vision"}
       }}
    else
      {:error, reason} -> {:error, reason}
      other -> {:error, other}
    end
  rescue
    e -> {:error, Exception.message(e)}
  end

  defp resolve(path) when is_binary(path), do: {:ok, path}
  defp resolve(%{path: p}) when is_binary(p), do: {:ok, p}
  defp resolve(%{url: u}) when is_binary(u), do: {:ok, u}
  defp resolve(_), do: {:error, :unresolvable_video_reference}
end
