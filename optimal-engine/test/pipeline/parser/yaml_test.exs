defmodule OptimalEngine.Pipeline.Parser.YamlTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.Yaml

  test "parses valid yaml and surfaces top-level keys" do
    yaml = """
    name: Optimal Engine
    version: 0.1.0
    deps:
      - jason
      - plug
    """

    assert {:ok, doc} = Yaml.parse_text(yaml, format: :yaml)
    assert doc.modality == :data
    assert doc.metadata.format == "yaml"
    assert "name" in doc.metadata.top_level_keys
    assert "version" in doc.metadata.top_level_keys
    assert "deps" in doc.metadata.top_level_keys
  end

  test "reports warning on malformed yaml" do
    malformed = """
    name: ok
      nested: bad
    """

    assert {:ok, doc} = Yaml.parse_text(malformed, format: :yaml)
    assert doc.warnings != []
  end

  test "handles toml as raw text with warning" do
    toml = """
    title = "hello"
    """

    assert {:ok, doc} = Yaml.parse_text(toml, format: :toml)
    assert doc.metadata.format == "toml"
    assert length(doc.warnings) >= 1
  end
end
