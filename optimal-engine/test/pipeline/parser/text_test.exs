defmodule OptimalEngine.Pipeline.Parser.TextTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Text

  test "splits on blank-line boundaries into paragraphs" do
    txt = """
    First paragraph.

    Second paragraph, which is a bit longer.

    Third.
    """

    assert {:ok, doc} = Text.parse_text(txt)
    paragraphs = Enum.filter(doc.structure, &(&1.kind == :paragraph))
    assert length(paragraphs) == 3
    assert hd(paragraphs).text =~ "First"
  end

  test "handles empty input without crashing" do
    assert {:ok, doc} = Text.parse_text("")
    assert doc.structure == []
    assert doc.text == ""
  end

  test "records file size + line count metadata" do
    assert {:ok, doc} = Text.parse_text("line one\nline two\n")
    assert doc.metadata.byte_size == byte_size("line one\nline two\n")
    assert doc.metadata.line_count >= 2
  end
end
