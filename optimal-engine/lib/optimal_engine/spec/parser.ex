defmodule OptimalEngine.Spec.Parser do
  @moduledoc """
  Parses `.spec.md` files — markdown with reserved fenced YAML blocks.

  Extracts five block types:
  - `spec-meta`          — subject identity (id, kind, status, surface, min strength)
  - `spec-requirements`  — list of requirements (id, statement, priority, stability)
  - `spec-scenarios`     — BDD given-when-then scenarios covering requirements
  - `spec-verification`  — verification claims (kind, target, covers)
  - `spec-exceptions`    — documented exceptions to requirements

  Errors are collected, not raised. A file with a malformed block still returns
  the blocks that parsed successfully, with errors attached.
  """

  @type requirement :: %{
          id: String.t(),
          statement: String.t(),
          priority: String.t(),
          stability: String.t()
        }

  @type scenario :: %{
          id: String.t(),
          covers: [String.t()],
          given: [String.t()],
          when_clause: [String.t()],
          then_clause: [String.t()]
        }

  @type verification :: %{
          kind: String.t(),
          target: String.t(),
          covers: [String.t()]
        }

  @type exception :: %{
          id: String.t(),
          statement: String.t()
        }

  @type meta :: %{
          id: String.t(),
          kind: String.t(),
          status: String.t(),
          surface: [String.t()],
          verification_minimum_strength: String.t()
        }

  @type spec :: %{
          file: String.t() | nil,
          title: String.t(),
          meta: meta(),
          requirements: [requirement()],
          scenarios: [scenario()],
          verifications: [verification()],
          exceptions: [exception()],
          errors: [{String.t(), String.t()}]
        }

  @block_types ~w[spec-meta spec-requirements spec-scenarios spec-verification spec-exceptions]

  # Match fenced code blocks with spec-* language tags
  @block_regex ~r/```(spec-(?:meta|requirements|scenarios|verification|exceptions))\n(.*?)```/s

  @doc """
  Parses a `.spec.md` file from disk.

  Returns `{:ok, spec}` with all extracted blocks and any parse errors,
  or `{:error, reason}` if the file cannot be read.
  """
  @spec parse_file(String.t()) :: {:ok, spec()} | {:error, term()}
  def parse_file(path) do
    case File.read(path) do
      {:ok, content} ->
        {:ok, spec} = parse_string(content)
        {:ok, %{spec | file: path}}

      {:error, reason} ->
        {:error, {:file_read, path, reason}}
    end
  end

  @doc """
  Parses a spec from a raw markdown string.
  """
  @spec parse_string(String.t()) :: {:ok, spec()}
  def parse_string(content) when is_binary(content) do
    title = extract_title(content)
    blocks = extract_blocks(content)

    spec = %{
      file: nil,
      title: title,
      meta: parse_meta(blocks["spec-meta"]),
      requirements:
        parse_list(blocks["spec-requirements"], &normalize_requirement/1, "spec-requirements"),
      scenarios: parse_list(blocks["spec-scenarios"], &normalize_scenario/1, "spec-scenarios"),
      verifications:
        parse_list(blocks["spec-verification"], &normalize_verification/1, "spec-verification"),
      exceptions: parse_list(blocks["spec-exceptions"], &normalize_exception/1, "spec-exceptions"),
      errors: collect_errors(blocks)
    }

    {:ok, spec}
  end

  @doc """
  Parses all `.spec.md` files under a directory.
  """
  @spec parse_all(String.t()) :: {:ok, [spec()]}
  def parse_all(dir) do
    specs =
      dir
      |> Path.join("**/*.spec.md")
      |> Path.wildcard()
      |> Enum.sort()
      |> Enum.map(fn path ->
        case parse_file(path) do
          {:ok, spec} -> spec
          {:error, _} -> nil
        end
      end)
      |> Enum.reject(&is_nil/1)

    {:ok, specs}
  end

  # -- Private: Block Extraction -----------------------------------------------

  defp extract_title(content) do
    case Regex.run(~r/^#\s+(.+)$/m, content) do
      [_, title] -> String.trim(title)
      _ -> "(untitled)"
    end
  end

  defp extract_blocks(content) do
    @block_regex
    |> Regex.scan(content)
    |> Enum.reduce(%{}, fn
      [_full, type, yaml], acc ->
        case decode_yaml(yaml, type) do
          {:ok, decoded} -> Map.put(acc, type, {:ok, decoded})
          {:error, reason} -> Map.put(acc, type, {:error, reason})
        end
    end)
  end

  defp decode_yaml(yaml_string, block_type) do
    case YamlElixir.read_from_string(yaml_string) do
      {:ok, decoded} -> {:ok, decoded}
      {:error, reason} -> {:error, "#{block_type}: YAML parse error — #{inspect(reason)}"}
    end
  end

  # -- Private: Meta -----------------------------------------------------------

  defp parse_meta(nil),
    do: %{
      id: nil,
      kind: nil,
      status: "draft",
      surface: [],
      verification_minimum_strength: "claimed"
    }

  defp parse_meta({:error, _}),
    do: %{
      id: nil,
      kind: nil,
      status: "draft",
      surface: [],
      verification_minimum_strength: "claimed"
    }

  defp parse_meta({:ok, map}) when is_map(map) do
    %{
      id: map["id"],
      kind: map["kind"],
      status: map["status"] || "draft",
      node: map["node"],
      surface: normalize_string_list(map["surface"]),
      decisions: normalize_string_list(map["decisions"]),
      verification_minimum_strength: map["verification_minimum_strength"] || "claimed"
    }
  end

  # -- Private: List Blocks ----------------------------------------------------

  defp parse_list(nil, _normalizer, _block_type), do: []
  defp parse_list({:error, _}, _normalizer, _block_type), do: []

  defp parse_list({:ok, items}, normalizer, _block_type) when is_list(items) do
    Enum.map(items, normalizer)
  end

  defp parse_list({:ok, _}, _normalizer, _block_type), do: []

  defp normalize_requirement(map) when is_map(map) do
    %{
      id: map["id"],
      statement: map["statement"] || "",
      priority: map["priority"] || "should",
      stability: map["stability"] || "evolving"
    }
  end

  defp normalize_requirement(_),
    do: %{id: nil, statement: "", priority: "should", stability: "evolving"}

  defp normalize_scenario(map) when is_map(map) do
    %{
      id: map["id"],
      covers: normalize_string_list(map["covers"]),
      given: normalize_string_list(map["given"]),
      when_clause: normalize_string_list(map["when"]),
      then_clause: normalize_string_list(map["then"])
    }
  end

  defp normalize_scenario(_),
    do: %{id: nil, covers: [], given: [], when_clause: [], then_clause: []}

  defp normalize_verification(map) when is_map(map) do
    %{
      kind: map["kind"] || "source_file",
      target: map["target"] || "",
      covers: normalize_string_list(map["covers"])
    }
  end

  defp normalize_verification(_), do: %{kind: "source_file", target: "", covers: []}

  defp normalize_exception(map) when is_map(map) do
    %{
      id: map["id"],
      statement: map["statement"] || ""
    }
  end

  defp normalize_exception(_), do: %{id: nil, statement: ""}

  # -- Private: Errors ---------------------------------------------------------

  defp collect_errors(blocks) do
    @block_types
    |> Enum.flat_map(fn type ->
      case Map.get(blocks, type) do
        {:error, reason} -> [{type, reason}]
        _ -> []
      end
    end)
  end

  # -- Private: Helpers --------------------------------------------------------

  defp normalize_string_list(nil), do: []
  defp normalize_string_list(list) when is_list(list), do: Enum.map(list, &to_string/1)
  defp normalize_string_list(str) when is_binary(str), do: [str]
  defp normalize_string_list(_), do: []
end
