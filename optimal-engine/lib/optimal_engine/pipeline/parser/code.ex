defmodule OptimalEngine.Pipeline.Parser.Code do
  @moduledoc """
  Source-code parser. Reads the file as text with `modality: :code` and
  extracts simple line-based function / class structure via language-
  agnostic heuristics.

  A richer tree-sitter-based parse is deferred to Phase 10+ — for now we
  preserve the text verbatim (so BM25 + vector retrieval work) and surface
  a small amount of structure for the Decomposer to respect function
  boundaries when it chunks.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  # Very simple def/function/class heuristics across common languages.
  # Intentionally coarse — we'd rather under-structure than over-split.
  @def_re ~r/^\s*(?:def|defp|function|func|fn|class|module|struct|interface|public\s+(?:class|interface|static)|private\s+def|impl)\s+([\w.:]+)/

  @impl true
  def parse(path, opts) when is_binary(path) do
    with {:ok, text} <- File.read(path) do
      language = detect_language(path)
      {:ok, build_doc(text, language, Keyword.put(opts, :path, path))}
    end
  end

  @impl true
  def parse_text(text, opts \\ []) when is_binary(text) do
    language = Keyword.get(opts, :language, "text")
    {:ok, build_doc(text, language, opts)}
  end

  defp build_doc(text, language, opts) do
    ParsedDoc.new(
      path: Keyword.get(opts, :path),
      text: text,
      structure: definitions(text, language),
      modality: :code,
      metadata: %{
        language: language,
        byte_size: byte_size(text),
        line_count: text |> String.split("\n") |> length()
      }
    )
  end

  defp definitions(text, language) do
    text
    |> String.split("\n", trim: false)
    |> Enum.reduce({[], 0}, fn line, {acc, offset} ->
      case Regex.run(@def_re, line, capture: :all_but_first) do
        [name] ->
          element =
            StructuralElement.new(:code_block,
              text: String.trim(line),
              offset: offset,
              length: byte_size(line),
              metadata: %{language: language, definition: name}
            )

          {[element | acc], offset + byte_size(line) + 1}

        _ ->
          {acc, offset + byte_size(line) + 1}
      end
    end)
    |> elem(0)
    |> Enum.reverse()
  end

  defp detect_language(path) do
    case Path.extname(path) |> String.downcase() do
      ".ex" -> "elixir"
      ".exs" -> "elixir"
      ".py" -> "python"
      ".js" -> "javascript"
      ".jsx" -> "javascript"
      ".ts" -> "typescript"
      ".tsx" -> "typescript"
      ".go" -> "go"
      ".rs" -> "rust"
      ".rb" -> "ruby"
      ".java" -> "java"
      ".c" -> "c"
      ".cpp" -> "cpp"
      ".cc" -> "cpp"
      ".hpp" -> "cpp"
      ".h" -> "c"
      ".cs" -> "csharp"
      ".kt" -> "kotlin"
      ".swift" -> "swift"
      ".php" -> "php"
      ".sh" -> "bash"
      ".bash" -> "bash"
      ".zsh" -> "bash"
      ".lua" -> "lua"
      ".pl" -> "perl"
      ".sql" -> "sql"
      ".scala" -> "scala"
      ".clj" -> "clojure"
      ".hs" -> "haskell"
      ".ml" -> "ocaml"
      ".fs" -> "fsharp"
      ".dart" -> "dart"
      ".nim" -> "nim"
      ".zig" -> "zig"
      ".vim" -> "vim"
      ".el" -> "elisp"
      _ -> "text"
    end
  end
end
