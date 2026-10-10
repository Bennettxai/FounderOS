defmodule OptimalEngine.Pipeline.Parser.Pdf do
  @moduledoc """
  PDF parser — extracts text via the system `pdftotext` binary
  (part of `poppler-utils`). If the tool isn't on `PATH`, returns a
  metadata-only `%ParsedDoc{}` with a warning so the pipeline can still
  index the file as a retrievable asset.

  Page boundaries are preserved as `:page` structural elements when
  `pdftotext -layout -f 1 -l 1` splits per page via the form-feed character.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{Asset, ParsedDoc, StructuralElement}

  @impl true
  def parse(path, _opts) when is_binary(path) do
    asset =
      case Asset.from_path(path, modality: :binary, content_type: "application/pdf") do
        {:ok, a} -> a
        _ -> nil
      end

    {text, structure, warnings, extras} =
      case System.find_executable("pdftotext") do
        nil ->
          {"", [], ["pdftotext not on PATH — install poppler-utils for full-text extraction"], %{}}

        _bin ->
          extract_with_pdftotext(path)
      end

    {:ok,
     ParsedDoc.new(
       path: path,
       text: text,
       structure: structure,
       modality: :mixed,
       metadata:
         Map.merge(
           %{format: "pdf", byte_size: byte_size(text)},
           extras
         ),
       assets: if(asset, do: [asset], else: []),
       warnings: warnings
     )}
  end

  @impl true
  def parse_text(_text, _opts) do
    {:error, :binary_format_requires_path}
  end

  # pdftotext writes form-feeds (0x0C) between pages in layout mode; split
  # on those to get page structural elements.
  defp extract_with_pdftotext(path) do
    case System.cmd("pdftotext", ["-layout", path, "-"], stderr_to_stdout: true) do
      {output, 0} ->
        pages = String.split(output, "\f", trim: false)

        {structure, _offset} =
          pages
          |> Enum.with_index(1)
          |> Enum.reduce({[], 0}, fn {page_text, page_num}, {acc, offset} ->
            trimmed = page_text
            size = byte_size(trimmed)

            element =
              StructuralElement.new(:page,
                text: trimmed,
                offset: offset,
                length: size,
                metadata: %{number: page_num}
              )

            {[element | acc], offset + size + 1}
          end)

        {output, Enum.reverse(structure), [], %{page_count: length(pages)}}

      {err, code} ->
        {"", [], ["pdftotext exited #{code}: #{String.slice(err, 0, 200)}"], %{}}
    end
  end
end
