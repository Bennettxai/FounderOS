defmodule OptimalEngine.Pipeline.Parser.Text do
  @moduledoc """
  Plain-text parser — handles `.txt`, `.rst`, `.adoc`, `.log`, and the
  "unknown extension" fallback from the Parser dispatcher.

  Splits on blank-line boundaries to produce `:paragraph` structural
  elements so the Decomposer can respect paragraph seams without needing
  heuristics.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @impl true
  def parse(path, opts) when is_binary(path) do
    case File.read(path) do
      {:ok, text} -> {:ok, build_doc(text, Keyword.put(opts, :path, path))}
      {:error, reason} -> {:error, reason}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    {:ok, build_doc(text, opts)}
  end

  defp build_doc(text, opts) do
    structure = paragraphs(text)

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :text,
      metadata: %{
        byte_size: byte_size(text),
        line_count: text |> String.split("\n", trim: false) |> length(),
        paragraph_count: length(structure)
      }
    )
  end

  defp paragraphs(""), do: []

  defp paragraphs(text) do
    text
    |> split_preserving_offsets()
    |> Enum.reject(fn {chunk, _offset} -> String.trim(chunk) == "" end)
    |> Enum.map(fn {chunk, offset} ->
      StructuralElement.new(:paragraph,
        text: chunk,
        offset: offset,
        length: byte_size(chunk)
      )
    end)
  end

  # Split on blank-line boundaries while tracking byte offsets back into the
  # original text, so the Decomposer gets real offsets (not reindexed ones).
  defp split_preserving_offsets(text) do
    text
    |> String.split(~r/\n{2,}/, include_captures: true, trim: false)
    |> Enum.chunk_every(2, 2, [""])
    |> Enum.reduce({[], 0}, fn [chunk, separator], {acc, offset} ->
      {[{chunk, offset} | acc], offset + byte_size(chunk) + byte_size(separator || "")}
    end)
    |> elem(0)
    |> Enum.reverse()
  end
end
