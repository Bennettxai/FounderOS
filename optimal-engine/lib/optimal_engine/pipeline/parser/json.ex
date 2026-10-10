defmodule OptimalEngine.Pipeline.Parser.Json do
  @moduledoc """
  JSON parser. Uses `Jason`. Surfaces top-level keys as paragraph structural
  elements so search can hit-highlight them.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, text} <- File.read(path) do
      {:ok, build_doc(text, Keyword.put(opts, :path, path))}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    {:ok, build_doc(text, opts)}
  end

  defp build_doc(text, opts) do
    {structure, warnings, extras} =
      case Jason.decode(text) do
        {:ok, parsed} when is_map(parsed) ->
          {top_level_elements(parsed, text), [], %{top_level_keys: Map.keys(parsed)}}

        {:ok, _} ->
          {[], [], %{shape: "non-object-root"}}

        {:error, %Jason.DecodeError{} = err} ->
          {[], ["json parse failed: #{Exception.message(err)}"], %{}}
      end

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :data,
      metadata: Map.merge(%{format: "json", byte_size: byte_size(text)}, extras),
      warnings: warnings
    )
  end

  defp top_level_elements(parsed, text) do
    Enum.map(Map.keys(parsed), fn key ->
      label = "\"#{key}\""
      offset = match_offset(text, label) || 0

      StructuralElement.new(:paragraph,
        text: label,
        offset: offset,
        length: byte_size(label),
        metadata: %{json_key: key}
      )
    end)
  end

  defp match_offset(text, needle) do
    case :binary.match(text, needle) do
      {offset, _} -> offset
      :nomatch -> nil
    end
  end
end
