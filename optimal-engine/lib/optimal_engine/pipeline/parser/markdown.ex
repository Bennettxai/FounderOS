defmodule OptimalEngine.Pipeline.Parser.Markdown do
  @moduledoc """
  Markdown parser — handles `.md`.

  Preserves heading hierarchy + paragraph + code-block boundaries.
  Lightweight native parser (regex-based); intentionally does NOT pull in a
  heavyweight AST library for Phase 2. Structure is line-offset-based so the
  Decomposer can compute chunk spans directly.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @heading_re ~r/\A(?<level>\#{1,6})\s+(?<text>.+?)\s*\z/

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, text} <- File.read(path) do
      doc = build_doc(text, Keyword.put(opts, :path, path))
      {:ok, doc}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    {:ok, build_doc(text, opts)}
  end

  defp build_doc(text, opts) do
    structure = extract_structure(text)

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :text,
      metadata: %{
        byte_size: byte_size(text),
        heading_count: Enum.count(structure, &(&1.kind == :heading)),
        code_block_count: Enum.count(structure, &(&1.kind == :code_block))
      }
    )
  end

  # Scan the text line-by-line, tracking byte offsets so StructuralElements
  # line up with the raw text. In-code-fence tracking prevents heading regex
  # from matching lines inside ``` fences.
  defp extract_structure(text) do
    {elements, _offset, _in_fence, _fence_lang, _fence_start, _fence_buf, _para_start, _para_buf,
     _para_offset} =
      text
      |> String.split("\n", trim: false)
      |> Enum.reduce(
        {[], 0, false, nil, 0, [], nil, [], nil},
        &process_line/2
      )

    Enum.reverse(elements)
  end

  defp process_line(
         line,
         {acc, offset, in_fence, fence_lang, fence_start, fence_buf, para_start, para_buf,
          para_offset}
       ) do
    next_offset = offset + byte_size(line) + 1

    cond do
      String.starts_with?(line, "```") and not in_fence ->
        # entering a code fence — flush any open paragraph first
        acc1 = flush_paragraph(acc, para_buf, para_start, para_offset)
        lang = line |> String.replace_prefix("```", "") |> String.trim()
        {acc1, next_offset, true, lang, offset, [], nil, [], nil}

      String.starts_with?(line, "```") and in_fence ->
        # leaving a code fence — emit a code_block element covering buf
        code_text = fence_buf |> Enum.reverse() |> Enum.join("\n")

        element =
          StructuralElement.new(:code_block,
            text: code_text,
            offset: fence_start,
            length: byte_size(code_text),
            metadata: %{language: fence_lang || ""}
          )

        {[element | acc], next_offset, false, nil, 0, [], nil, [], nil}

      in_fence ->
        {acc, next_offset, true, fence_lang, fence_start, [line | fence_buf], para_start, para_buf,
         para_offset}

      String.match?(line, @heading_re) ->
        # headings flush any pending paragraph and emit themselves
        acc1 = flush_paragraph(acc, para_buf, para_start, para_offset)

        %{"level" => hashes, "text" => htext} = Regex.named_captures(@heading_re, line)
        level = String.length(hashes)

        heading =
          StructuralElement.new(:heading,
            text: htext,
            offset: offset,
            length: byte_size(line),
            metadata: %{level: level}
          )

        {[heading | acc1], next_offset, false, nil, 0, [], nil, [], nil}

      String.trim(line) == "" ->
        # blank line terminates a paragraph
        acc1 = flush_paragraph(acc, para_buf, para_start, para_offset)
        {acc1, next_offset, false, nil, 0, [], nil, [], nil}

      true ->
        # content line; start or extend a paragraph
        {new_start, new_offset} =
          if para_start == nil, do: {offset, offset}, else: {para_start, para_offset}

        {acc, next_offset, false, nil, 0, [], new_start, [line | para_buf], new_offset}
    end
  end

  defp flush_paragraph(acc, [], _, _), do: acc
  defp flush_paragraph(acc, _buf, nil, _), do: acc

  defp flush_paragraph(acc, buf, start_offset, _) do
    para_text = buf |> Enum.reverse() |> Enum.join("\n")

    if String.trim(para_text) == "" do
      acc
    else
      element =
        StructuralElement.new(:paragraph,
          text: para_text,
          offset: start_offset,
          length: byte_size(para_text)
        )

      [element | acc]
    end
  end
end
