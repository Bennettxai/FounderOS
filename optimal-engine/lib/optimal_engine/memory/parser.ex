defmodule OptimalEngine.Memory.Parser do
  @moduledoc """
  Parses MEMORY.md formatted content into structured entry maps.

  Entry format:
      ## [category] 2026-02-27T10:30:00Z
      Content spanning multiple lines...
  """

  @category_importance %{
    "decision" => 1.0,
    "preference" => 0.9,
    "architecture" => 0.95,
    "bug" => 0.8,
    "insight" => 0.85,
    "contact" => 0.7,
    "workflow" => 0.75,
    "general" => 0.5,
    "note" => 0.4
  }

  @doc """
  Parse MEMORY.md content into a list of `{entry_id, entry_map}` tuples.
  """
  @spec parse(String.t()) :: [{String.t(), map()}]
  def parse(content) when is_binary(content) do
    entry_regex = ~r/^## \[([^\]]+)\]\s+(.+)$/m
    parts = Regex.split(entry_regex, content, include_captures: true, trim: true)
    parse_parts(parts, [])
  end

  def parse(_), do: []

  @doc "Compute an importance score for a given category and content."
  @spec importance(String.t(), String.t()) :: float()
  def importance(category, content) do
    base = Map.get(@category_importance, String.downcase(category), 0.5)

    length_boost =
      cond do
        byte_size(content) > 500 -> 0.1
        byte_size(content) > 200 -> 0.05
        true -> 0.0
      end

    technical_boost =
      if Regex.match?(~r/[A-Z][a-z]+[A-Z]|[a-z]+_[a-z]+|->|=>|\(\)|def |fn /, content) do
        0.05
      else
        0.0
      end

    min(base + length_boost + technical_boost, 1.0)
  end

  @doc "Generate a deterministic entry ID from category, timestamp, and content."
  @spec entry_id(String.t(), String.t(), String.t()) :: String.t()
  def entry_id(category, timestamp, content) do
    data = "#{category}:#{timestamp}:#{content}"
    :crypto.hash(:sha256, data) |> Base.encode16(case: :lower) |> String.slice(0, 16)
  end

  @doc "Returns the importance weight for a given category string."
  @spec category_importance(String.t()) :: float()
  def category_importance(category) do
    Map.get(@category_importance, String.downcase(category || "general"), 0.5)
  end

  # ── Private ──────────────────────────────────────────────────────────

  defp parse_parts([], acc), do: Enum.reverse(acc)

  defp parse_parts([potential_header | rest], acc) do
    case Regex.run(~r/^## \[([^\]]+)\]\s+(.+)$/, String.trim(potential_header)) do
      [_full, category, timestamp_str] ->
        {content, remaining} =
          case rest do
            [body | tail] ->
              if Regex.match?(~r/^## \[/, String.trim(body)) do
                {"", rest}
              else
                {String.trim(body), tail}
              end

            [] ->
              {"", []}
          end

        id = entry_id(category, timestamp_str, content)

        entry = %{
          id: id,
          category: category,
          timestamp: timestamp_str,
          content: content,
          importance: importance(category, content)
        }

        parse_parts(remaining, [{id, entry} | acc])

      nil ->
        parse_parts(rest, acc)
    end
  end
end
