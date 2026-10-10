defmodule OptimalEngine.Pipeline.Parser.CsvTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Csv

  test "parses a CSV with header and rows" do
    csv = """
    name,role,tenure
    Ada,Engineer,5
    Bob,Designer,3
    """

    assert {:ok, doc} = Csv.parse_text(csv, format: :csv)
    assert doc.modality == :data
    assert doc.metadata.format == "csv"
    assert doc.metadata.row_count == 2
    assert doc.metadata.column_count == 3
    assert doc.metadata.headers == ["name", "role", "tenure"]
    assert length(doc.structure) == 2
    first_row = hd(doc.structure)
    assert first_row.kind == :table_row
    assert first_row.metadata.columns == ["Ada", "Engineer", "5"]
  end

  test "parses TSV with tab separator" do
    tsv = "a\tb\nc\td\n"
    assert {:ok, doc} = Csv.parse_text(tsv, format: :tsv)
    assert doc.metadata.format == "tsv"
    assert length(doc.structure) == 1
    assert hd(doc.structure).metadata.columns == ["c", "d"]
  end
end
