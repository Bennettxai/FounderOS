defmodule OptimalEngine.Pipeline.Parser.Html do
  @moduledoc """
  HTML parser via `Floki`. Strips scripts + styles, extracts visible text,
  and preserves heading structure (`h1`..`h6`) as `:heading` elements.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, html} <- File.read(path) do
      {:ok, build_doc(html, Keyword.put(opts, :path, path))}
    end
  end

  @impl true
  def parse_text(html, opts \\ []) when is_binary(html) do
    {:ok, build_doc(html, opts)}
  end

  defp build_doc(html, opts) do
    {text, structure, metadata, warnings} =
      case Floki.parse_document(html) do
        {:ok, doc} ->
          cleaned = doc |> strip(["script", "style", "noscript"])
          plain = cleaned |> Floki.text(sep: "\n") |> normalize_whitespace()

          {
            plain,
            extract_headings(cleaned, plain),
            %{
              title: find_text(cleaned, "title"),
              byte_size_html: byte_size(html),
              byte_size_text: byte_size(plain)
            },
            []
          }

        {:error, reason} ->
          {html, [], %{parse_error: inspect(reason)}, ["floki parse failed"]}
      end

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :text,
      metadata: metadata,
      warnings: warnings
    )
  end

  defp strip(tree, selectors) do
    Enum.reduce(selectors, tree, fn selector, acc -> Floki.filter_out(acc, selector) end)
  end

  defp find_text(tree, selector) do
    case Floki.find(tree, selector) do
      [] -> nil
      nodes -> nodes |> Floki.text() |> normalize_whitespace() |> nil_if_empty()
    end
  end

  defp nil_if_empty(""), do: nil
  defp nil_if_empty(s), do: s

  defp normalize_whitespace(text) do
    text
    |> String.replace(~r/\r\n|\r/, "\n")
    |> String.replace(~r/[\t ]+/, " ")
    |> String.replace(~r/\n{3,}/, "\n\n")
    |> String.trim()
  end

  defp extract_headings(tree, plain_text) do
    1..6
    |> Enum.flat_map(fn level ->
      tree
      |> Floki.find("h#{level}")
      |> Enum.map(fn node ->
        heading_text = node |> Floki.text() |> normalize_whitespace()
        offset = match_offset(plain_text, heading_text) || 0

        StructuralElement.new(:heading,
          text: heading_text,
          offset: offset,
          length: byte_size(heading_text),
          metadata: %{level: level}
        )
      end)
    end)
    |> Enum.sort_by(& &1.offset)
  end

  defp match_offset(_text, ""), do: nil

  defp match_offset(text, needle) do
    case :binary.match(text, needle) do
      {offset, _} -> offset
      :nomatch -> nil
    end
  end
end
