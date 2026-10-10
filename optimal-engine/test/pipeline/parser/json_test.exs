defmodule OptimalEngine.Pipeline.Parser.JsonTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Json

  test "parses a valid JSON object and surfaces top-level keys" do
    json = ~s({"name": "Optimal", "version": "0.1.0", "plan": "default"})
    assert {:ok, doc} = Json.parse_text(json)
    assert doc.modality == :data
    assert doc.metadata.format == "json"
    assert "name" in doc.metadata.top_level_keys
    assert length(doc.structure) == 3
  end

  test "records a warning on malformed json" do
    assert {:ok, doc} = Json.parse_text(~s({bad: not_quoted}))
    assert doc.warnings != []
  end

  test "handles non-object roots" do
    assert {:ok, doc} = Json.parse_text("[1, 2, 3]")
    assert doc.structure == []
    assert doc.metadata.shape == "non-object-root"
  end
end
