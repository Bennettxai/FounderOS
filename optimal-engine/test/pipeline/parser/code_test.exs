defmodule OptimalEngine.Pipeline.Parser.CodeTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Code

  test "preserves source text verbatim and detects language" do
    src = """
    defmodule Hello do
      def greet(name), do: "hi " <> name
    end
    """

    tmp = Path.join(System.tmp_dir!(), "code_test_#{System.unique_integer([:positive])}.ex")
    File.write!(tmp, src)

    try do
      assert {:ok, doc} = Code.parse(tmp, [])
      assert doc.modality == :code
      assert doc.metadata.language == "elixir"
      assert doc.text == src

      defs = Enum.filter(doc.structure, &(&1.kind == :code_block))
      assert length(defs) >= 1

      assert Enum.any?(defs, fn e ->
               String.contains?(e.text, "defmodule Hello") or
                 String.contains?(e.text, "def greet")
             end)
    after
      File.rm(tmp)
    end
  end

  test "parse_text handles inline code with language hint" do
    assert {:ok, doc} = Code.parse_text("print('hi')", language: "python")
    assert doc.modality == :code
    assert doc.metadata.language == "python"
  end
end
