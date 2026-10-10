defmodule OptimalEngine.Pipeline.Parser.Office do
  @moduledoc """
  Microsoft Office OOXML parser — `.docx`, `.pptx`, `.xlsx`.

  Native Elixir: unzip the archive with Erlang's `:zip`, then walk the
  well-known XML entries to extract text. No external executables required.

  - `.docx` — reads `word/document.xml`, concatenates text in document order.
  - `.pptx` — reads `ppt/slides/slide*.xml` in order, emits one `:slide`
    structural element per slide.
  - `.xlsx` — reads `xl/sharedStrings.xml` (string table) plus
    `xl/worksheets/sheet*.xml`, renders each sheet as TSV text, emits
    `:section` per sheet.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @impl true
  def parse(path, _opts) when is_binary(path) do
    ext = Path.extname(path) |> String.downcase()

    case :zip.unzip(String.to_charlist(path), [:memory]) do
      {:ok, files} ->
        entries =
          Map.new(files, fn {name, data} -> {to_string(name), data} end)

        doc =
          case ext do
            ".docx" -> parse_docx(path, entries)
            ".pptx" -> parse_pptx(path, entries)
            ".xlsx" -> parse_xlsx(path, entries)
            _ -> parse_unknown(path, entries, ext)
          end

        {:ok, doc}

      {:error, reason} ->
        {:ok,
         ParsedDoc.new(
           path: path,
           text: "",
           modality: :text,
           metadata: %{format: office_format(ext)},
           warnings: ["failed to unzip #{ext}: #{inspect(reason)}"]
         )}
    end
  end

  @impl true
  def parse_text(_text, _opts), do: {:error, :binary_format_requires_path}

  # ── .docx ────────────────────────────────────────────────────────────────

  defp parse_docx(path, entries) do
    body = Map.get(entries, "word/document.xml", "")
    text = extract_text_runs(body)

    ParsedDoc.new(
      path: path,
      text: text,
      structure: paragraphs_from_text(text),
      modality: :text,
      metadata: %{format: "docx", byte_size: byte_size(text)}
    )
  end

  # ── .pptx ────────────────────────────────────────────────────────────────

  defp parse_pptx(path, entries) do
    slide_files =
      entries
      |> Map.keys()
      |> Enum.filter(&Regex.match?(~r|^ppt/slides/slide\d+\.xml$|, &1))
      |> Enum.sort_by(&slide_number/1)

    {text, structure, _offset} =
      slide_files
      |> Enum.with_index(1)
      |> Enum.reduce({"", [], 0}, fn {file, idx}, {acc_text, acc_struct, offset} ->
        slide_text = extract_text_runs(Map.get(entries, file, ""))
        new_text = acc_text <> slide_text <> "\n\n"

        element =
          StructuralElement.new(:slide,
            text: slide_text,
            offset: offset,
            length: byte_size(slide_text),
            metadata: %{number: idx}
          )

        {new_text, [element | acc_struct], offset + byte_size(slide_text) + 2}
      end)

    ParsedDoc.new(
      path: path,
      text: text,
      structure: Enum.reverse(structure),
      modality: :text,
      metadata: %{format: "pptx", slide_count: length(slide_files), byte_size: byte_size(text)}
    )
  end

  # ── .xlsx ────────────────────────────────────────────────────────────────

  defp parse_xlsx(path, entries) do
    shared_strings = parse_shared_strings(Map.get(entries, "xl/sharedStrings.xml", ""))

    sheet_files =
      entries
      |> Map.keys()
      |> Enum.filter(&Regex.match?(~r|^xl/worksheets/sheet\d+\.xml$|, &1))
      |> Enum.sort_by(&sheet_number/1)

    {text, structure, _offset} =
      sheet_files
      |> Enum.with_index(1)
      |> Enum.reduce({"", [], 0}, fn {file, idx}, {acc_text, acc_struct, offset} ->
        sheet_xml = Map.get(entries, file, "")
        rendered = render_sheet(sheet_xml, shared_strings)
        new_text = acc_text <> rendered <> "\n\n"

        element =
          StructuralElement.new(:section,
            text: rendered,
            offset: offset,
            length: byte_size(rendered),
            metadata: %{title: "Sheet #{idx}", number: idx}
          )

        {new_text, [element | acc_struct], offset + byte_size(rendered) + 2}
      end)

    ParsedDoc.new(
      path: path,
      text: text,
      structure: Enum.reverse(structure),
      modality: :data,
      metadata: %{format: "xlsx", sheet_count: length(sheet_files), byte_size: byte_size(text)}
    )
  end

  defp parse_unknown(path, _entries, ext) do
    ParsedDoc.new(
      path: path,
      text: "",
      modality: :text,
      metadata: %{format: office_format(ext)},
      warnings: ["unknown office format: #{ext}"]
    )
  end

  # ── XML text extraction (shared) ─────────────────────────────────────────

  # Extract text by pulling every <w:t>…</w:t> (docx) / <a:t>…</a:t> (pptx)
  # / <t>…</t> (xlsx strings) element. Newlines between paragraphs/slides/cells.
  defp extract_text_runs(xml) when is_binary(xml) do
    Regex.scan(~r|<[^>]*:t[^/>]*>([^<]*)</[^>]*:t>|, xml, capture: :all_but_first)
    |> Enum.concat(Regex.scan(~r|<t[^/>]*>([^<]*)</t>|, xml, capture: :all_but_first))
    |> Enum.map(fn [chunk] -> unescape_xml(chunk) end)
    |> Enum.join("\n")
  end

  defp parse_shared_strings(xml) do
    Regex.scan(~r|<si[^>]*>(.*?)</si>|s, xml, capture: :all_but_first)
    |> Enum.map(fn [si_fragment] ->
      si_fragment
      |> then(&Regex.scan(~r|<t[^/>]*>([^<]*)</t>|, &1, capture: :all_but_first))
      |> Enum.map(fn [t] -> unescape_xml(t) end)
      |> Enum.join("")
    end)
  end

  defp render_sheet(sheet_xml, shared_strings) do
    Regex.scan(~r|<row[^>]*>(.*?)</row>|s, sheet_xml, capture: :all_but_first)
    |> Enum.map(fn [row_xml] ->
      Regex.scan(~r|<c([^>]*)>(.*?)</c>|s, row_xml)
      |> Enum.map(fn [_full, attrs, cell_body] ->
        cond do
          String.contains?(attrs, ~s(t="s")) ->
            [idx_str] =
              Regex.run(~r|<v>([^<]+)</v>|, cell_body, capture: :all_but_first) || ["0"]

            idx = String.to_integer(idx_str)
            Enum.at(shared_strings, idx, "")

          true ->
            case Regex.run(~r|<v>([^<]+)</v>|, cell_body, capture: :all_but_first) do
              [value] -> unescape_xml(value)
              _ -> ""
            end
        end
      end)
      |> Enum.join("\t")
    end)
    |> Enum.join("\n")
  end

  defp unescape_xml(text) do
    text
    |> String.replace("&amp;", "&")
    |> String.replace("&lt;", "<")
    |> String.replace("&gt;", ">")
    |> String.replace("&quot;", "\"")
    |> String.replace("&apos;", "'")
  end

  # ── helpers ──────────────────────────────────────────────────────────────

  defp slide_number(path) do
    case Regex.run(~r|slide(\d+)\.xml$|, path, capture: :all_but_first) do
      [n] -> String.to_integer(n)
      _ -> 0
    end
  end

  defp sheet_number(path) do
    case Regex.run(~r|sheet(\d+)\.xml$|, path, capture: :all_but_first) do
      [n] -> String.to_integer(n)
      _ -> 0
    end
  end

  defp office_format(".docx"), do: "docx"
  defp office_format(".pptx"), do: "pptx"
  defp office_format(".xlsx"), do: "xlsx"
  defp office_format(ext), do: String.trim_leading(ext, ".")

  defp paragraphs_from_text(text) do
    text
    |> String.split(~r/\n{2,}/, trim: true)
    |> Enum.with_index()
    |> Enum.map(fn {para, idx} ->
      StructuralElement.new(:paragraph,
        text: para,
        offset: idx,
        length: byte_size(para)
      )
    end)
  end
end
