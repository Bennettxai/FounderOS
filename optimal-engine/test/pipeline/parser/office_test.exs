defmodule OptimalEngine.Pipeline.Parser.OfficeTest do
  @moduledoc """
  Office format (docx/pptx/xlsx) parsing via native :zip + OOXML.

  Tests construct minimal OOXML-shaped ZIPs on the fly rather than shipping
  fixture binaries. The goal is to verify the parser's happy path and its
  graceful-degradation behavior without requiring external authoring tools.
  """
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Office
  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  describe ".docx" do
    test "extracts <w:t> runs as text" do
      docx_xml = """
      <?xml version="1.0"?>
      <w:document xmlns:w="urn:x">
        <w:body>
          <w:p><w:r><w:t>Hello</w:t></w:r></w:p>
          <w:p><w:r><w:t>World</w:t></w:r></w:p>
        </w:body>
      </w:document>
      """

      path = make_zip("test.docx", [{"word/document.xml", docx_xml}])

      try do
        assert {:ok, %ParsedDoc{modality: :text} = doc} = Office.parse(path, [])
        assert String.contains?(doc.text, "Hello")
        assert String.contains?(doc.text, "World")
        assert doc.metadata.format == "docx"
      after
        File.rm(path)
      end
    end
  end

  describe ".pptx" do
    test "emits one :slide element per slide" do
      slide1 = ~s(<?xml version="1.0"?><p:sld xmlns:a="x"><a:t>Slide one</a:t></p:sld>)
      slide2 = ~s(<?xml version="1.0"?><p:sld xmlns:a="x"><a:t>Slide two</a:t></p:sld>)

      path =
        make_zip("test.pptx", [
          {"ppt/slides/slide1.xml", slide1},
          {"ppt/slides/slide2.xml", slide2}
        ])

      try do
        assert {:ok, doc} = Office.parse(path, [])
        slides = Enum.filter(doc.structure, &(&1.kind == :slide))
        assert length(slides) == 2
        assert doc.metadata.slide_count == 2
      after
        File.rm(path)
      end
    end
  end

  describe ".xlsx" do
    test "expands shared-string references and emits :section per sheet" do
      shared = ~s(<?xml version="1.0"?><sst><si><t>Ada</t></si><si><t>5</t></si></sst>)

      sheet =
        ~s(<?xml version="1.0"?><worksheet><sheetData><row><c t="s"><v>0</v></c><c t="s"><v>1</v></c></row></sheetData></worksheet>)

      path =
        make_zip("test.xlsx", [
          {"xl/sharedStrings.xml", shared},
          {"xl/worksheets/sheet1.xml", sheet}
        ])

      try do
        assert {:ok, doc} = Office.parse(path, [])
        sections = Enum.filter(doc.structure, &(&1.kind == :section))
        assert length(sections) == 1
        assert String.contains?(doc.text, "Ada")
      after
        File.rm(path)
      end
    end
  end

  defp make_zip(filename, entries) do
    path = Path.join(System.tmp_dir!(), "#{System.unique_integer([:positive])}_#{filename}")

    zip_entries =
      Enum.map(entries, fn {name, data} ->
        {String.to_charlist(name), data}
      end)

    {:ok, _} = :zip.create(String.to_charlist(path), zip_entries)
    path
  end
end
