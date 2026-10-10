defmodule OptimalEngine.Pipeline.Parser.MarkdownTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Markdown
  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  describe "parse_text/2" do
    test "extracts heading hierarchy" do
      md = """
      # Top
      ## Second
      ### Third
      Body.
      """

      assert {:ok, %ParsedDoc{structure: struct}} = Markdown.parse_text(md)
      headings = Enum.filter(struct, &(&1.kind == :heading))
      assert length(headings) == 3
      assert Enum.at(headings, 0).metadata.level == 1
      assert Enum.at(headings, 1).metadata.level == 2
      assert Enum.at(headings, 2).metadata.level == 3
    end

    test "extracts paragraphs between headings" do
      md = """
      # Title

      First paragraph.

      Second paragraph.
      """

      assert {:ok, doc} = Markdown.parse_text(md)
      paragraphs = Enum.filter(doc.structure, &(&1.kind == :paragraph))
      assert length(paragraphs) == 2
      assert Enum.at(paragraphs, 0).text =~ "First paragraph"
      assert Enum.at(paragraphs, 1).text =~ "Second paragraph"
    end

    test "extracts fenced code blocks with language" do
      md = """
      Some text.

      ```elixir
      IO.puts("hi")
      ```

      More text.
      """

      assert {:ok, doc} = Markdown.parse_text(md)
      code_blocks = Enum.filter(doc.structure, &(&1.kind == :code_block))
      assert length(code_blocks) == 1
      assert hd(code_blocks).metadata.language == "elixir"
      assert hd(code_blocks).text =~ "IO.puts"
    end

    test "does not treat # inside code fences as headings" do
      md = """
      ```python
      # this is a comment, not a heading
      print("hi")
      ```
      """

      assert {:ok, doc} = Markdown.parse_text(md)
      headings = Enum.filter(doc.structure, &(&1.kind == :heading))
      assert headings == []
    end

    test "sets modality and signal_id" do
      assert {:ok, doc} = Markdown.parse_text("# Hello")
      assert doc.modality == :text
      assert String.starts_with?(doc.signal_id, "sha256:")
    end
  end
end
