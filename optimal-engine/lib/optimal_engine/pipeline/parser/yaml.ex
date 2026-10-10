defmodule OptimalEngine.Pipeline.Parser.Yaml do
  @moduledoc """
  YAML / YML / TOML parser. Uses `yaml_elixir` for YAML; TOML is treated as
  text with a warning (no OSS Elixir TOML parser we want to ship in v0.1;
  Phase 10+ may add `:toml`).

  Produces a single `:section` structural element spanning the whole
  document, plus top-level keys surfaced as `:paragraph` elements for search
  hit-highlighting.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, text} <- File.read(path) do
      format = detect_format(path)
      doc = build_doc(text, format, Keyword.put(opts, :path, path))
      {:ok, doc}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    format = Keyword.get(opts, :format, :yaml)
    {:ok, build_doc(text, format, opts)}
  end

  defp detect_format(path) do
    case Path.extname(path) |> String.downcase() do
      ".toml" -> :toml
      _ -> :yaml
    end
  end

  defp build_doc(text, :yaml, opts) do
    {structure, warnings, metadata} =
      case YamlElixir.read_from_string(text) do
        {:ok, parsed} ->
          {top_level_elements(parsed, text), [],
           %{format: "yaml", top_level_keys: top_level_keys(parsed)}}

        {:error, reason} ->
          {[], ["yaml parse failed: #{inspect(reason)}"], %{format: "yaml"}}
      end

    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: structure,
      modality: :data,
      metadata: Map.put(metadata, :byte_size, byte_size(text)),
      warnings: warnings
    )
  end

  defp build_doc(text, :toml, opts) do
    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: [],
      modality: :data,
      metadata: %{format: "toml", byte_size: byte_size(text)},
      warnings: ["TOML full parse not implemented; indexed as raw text"]
    )
  end

  defp top_level_keys(parsed) when is_map(parsed), do: Map.keys(parsed)
  defp top_level_keys(_), do: []

  defp top_level_elements(parsed, text) when is_map(parsed) do
    Enum.map(Map.keys(parsed), fn key ->
      label = "#{key}:"
      offset = match_offset(text, label) || 0

      StructuralElement.new(:paragraph,
        text: label,
        offset: offset,
        length: byte_size(label),
        metadata: %{yaml_key: key}
      )
    end)
  end

  defp top_level_elements(_, _), do: []

  defp match_offset(text, needle) do
    case :binary.match(text, needle) do
      {offset, _} -> offset
      :nomatch -> nil
    end
  end
end
