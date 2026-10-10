defmodule OptimalEngine.Pipeline.ParserTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser

  describe "dispatch/1" do
    test "routes by extension (text)" do
      assert {:ok, OptimalEngine.Pipeline.Parser.Markdown} = Parser.dispatch("x.md")
      assert {:ok, OptimalEngine.Pipeline.Parser.Text} = Parser.dispatch("x.txt")
      assert {:ok, OptimalEngine.Pipeline.Parser.Yaml} = Parser.dispatch("x.yaml")
      assert {:ok, OptimalEngine.Pipeline.Parser.Json} = Parser.dispatch("x.json")
      assert {:ok, OptimalEngine.Pipeline.Parser.Csv} = Parser.dispatch("x.csv")
      assert {:ok, OptimalEngine.Pipeline.Parser.Html} = Parser.dispatch("x.html")
    end

    test "routes by extension (code)" do
      assert {:ok, OptimalEngine.Pipeline.Parser.Code} = Parser.dispatch("x.ex")
      assert {:ok, OptimalEngine.Pipeline.Parser.Code} = Parser.dispatch("x.py")
      assert {:ok, OptimalEngine.Pipeline.Parser.Code} = Parser.dispatch("x.rs")
    end

    test "routes by extension (binary)" do
      assert {:ok, OptimalEngine.Pipeline.Parser.Pdf} = Parser.dispatch("x.pdf")
      assert {:ok, OptimalEngine.Pipeline.Parser.Office} = Parser.dispatch("x.docx")
      assert {:ok, OptimalEngine.Pipeline.Parser.Image} = Parser.dispatch("x.png")
      assert {:ok, OptimalEngine.Pipeline.Parser.Audio} = Parser.dispatch("x.wav")
      assert {:ok, OptimalEngine.Pipeline.Parser.Video} = Parser.dispatch("x.mp4")
    end

    test "returns :unknown for unrecognized extensions" do
      assert :unknown = Parser.dispatch("x.unknown")
      assert :unknown = Parser.dispatch("x")
    end

    test "case-insensitive on extension" do
      assert {:ok, OptimalEngine.Pipeline.Parser.Markdown} = Parser.dispatch("X.MD")
      assert {:ok, OptimalEngine.Pipeline.Parser.Pdf} = Parser.dispatch("X.PDF")
    end
  end

  describe "backends/0" do
    test "returns the registry of {extensions, module} tuples" do
      registry = Parser.backends()
      assert is_list(registry)
      assert Enum.all?(registry, fn {exts, mod} -> is_list(exts) and is_atom(mod) end)
    end
  end
end
