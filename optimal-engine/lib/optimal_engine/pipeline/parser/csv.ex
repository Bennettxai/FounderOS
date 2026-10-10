defmodule OptimalEngine.Pipeline.Parser.Csv do
  @moduledoc """
  CSV / TSV parser via `NimbleCSV`.

  Produces a `:table_row` structural element per row, with the header row
  kept in the `:headers` metadata on the ParsedDoc. The text view is a plain
  tab-joined rendering for full-text search indexing.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  # Define two tiny NimbleCSV parsers (one per separator) lazily; they're
  # module-level so the definition happens at compile time exactly once.
  NimbleCSV.define(__MODULE__.Comma, separator: ",", escape: "\"")
  NimbleCSV.define(__MODULE__.Tab, separator: "\t", escape: "\"")

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, text} <- File.read(path) do
      format =
        case Path.extname(path) |> String.downcase() do
          ".tsv" -> :tsv
          _ -> :csv
        end

      {:ok, build_doc(text, format, Keyword.put(opts, :path, path))}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    format = Keyword.get(opts, :format, :csv)
    {:ok, build_doc(text, format, opts)}
  end

  defp build_doc(text, format, opts) do
    parser =
      case format do
        :tsv -> __MODULE__.Tab
        _ -> __MODULE__.Comma
      end

    {rows, warnings} =
      try do
        {parser.parse_string(text, skip_headers: false), []}
      rescue
        e -> {[], ["csv parse failed: #{Exception.message(e)}"]}
      end

    {headers, data_rows} =
      case rows do
        [head | rest] -> {head, rest}
        [] -> {[], []}
      end

    structure = table_row_elements(data_rows, text)

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :data,
      metadata: %{
        format: Atom.to_string(format),
        byte_size: byte_size(text),
        row_count: length(data_rows),
        column_count: length(headers),
        headers: headers
      },
      warnings: warnings
    )
  end

  defp table_row_elements(rows, _text) do
    # Offsets aren't meaningful for CSV post-NimbleCSV since it consumes
    # the whole buffer; set them sequentially for stable ordering.
    rows
    |> Enum.with_index()
    |> Enum.map(fn {columns, idx} ->
      rendered = Enum.join(columns, "\t")

      StructuralElement.new(:table_row,
        text: rendered,
        offset: idx,
        length: byte_size(rendered),
        metadata: %{columns: columns, row_index: idx}
      )
    end)
  end
end
