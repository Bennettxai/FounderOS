defmodule OptimalEngine.Architecture.Processors.VlmCaptioner do
  @moduledoc """
  VLM captioning — sends images or video frames to a Vision-Language
  Model (default: Qwen2.5-VL via Ollama) and returns a text description.

  Works for both `:image` and `:video` modalities. For video, expects
  a keyframe image path as input.
  """

  @behaviour OptimalEngine.Architecture.Processor

  alias OptimalEngine.Embed.Ollama

  @prompt "Describe this image in detail. Include any text, objects, people, charts, diagrams, layout, and notable visual elements. Be precise and factual."

  @impl true
  def id, do: :vlm_captioner

  @impl true
  def modality, do: :image

  @impl true
  def emits, do: [:caption]

  @impl true
  def init(_config), do: {:ok, %{}}

  @impl true
  def process(_field, value, _state) do
    with {:ok, path} <- resolve(value),
         {:ok, caption} <- Ollama.vlm_analyze(path, @prompt) do
      cfg = Application.get_env(:optimal_engine, :ollama, [])
      model = cfg[:vlm_model] || "qwen2.5-vl"

      {:ok,
       %{
         kind: :caption,
         value: caption,
         metadata: %{model: model}
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
  defp resolve(_), do: {:error, :unresolvable_image_reference}
end
