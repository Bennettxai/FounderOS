defmodule OptimalEngine.Pipeline.Classifier do
  @moduledoc """
  Stateless classification for all content types. No GenServer needed — pure functions.

  ## Context type detection

  Before signal classification, the classifier determines WHAT KIND of content
  it is dealing with:

  - `:signal`   — markdown with YAML frontmatter OR content matching signal patterns
  - `:resource` — static knowledge (docs, code, PDFs, specs, data files)
  - `:memory`   — dynamic learned facts (path-based: `_memories/`)
  - `:skill`    — callable tools (path-based: `_skills/`)

  ## Signal classification pipeline (only for :signal type)

  1. Parse YAML frontmatter from the `signal:` block
  2. Fall back to content-pattern detection when frontmatter is absent/incomplete
  3. Return a fully-classified Signal struct

  ## Resource/Memory/Skill pipeline

  For non-signal types:
  1. Detect file format from extension
  2. Extract title from content
  3. Generate L0/L1/L2 summaries

  Entity extraction uses the known people list from the topology — regex-based
  scanning for proper nouns from the roster.
  """

  alias OptimalEngine.{Context, Signal}
  alias OptimalEngine.Signal.GenreRegistry
  alias OptimalEngine.Pipeline.Classifier.Classification
  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}

  # Genre keywords used for auto-detection (ordered by specificity)
  @genre_patterns [
    {"invoice", ~r/invoice|billing|payment due/i},
    {"profit-loss", ~r/P&L|profit.loss|revenue vs|net income/i},
    {"budget", ~r/\bbudget\b|quarterly spend|annual spend/i},
    {"decision-log", ~r/decision:|decided:|we decided|key decisions/i},
    {"adr", ~r/Architecture Decision Record|ADR\s+\d+|## Status\n.*## Context/ims},
    {"spec", ~r/## Requirements|## Acceptance Criteria|## Constraints/i},
    {"runbook", ~r/## Steps|## Procedure|runbook|step-by-step/i},
    {"postmortem", ~r/postmortem|incident report|root cause analysis/i},
    {"standup", ~r/yesterday|today|blockers|standup|weekly signal/i},
    {"transcript", ~r/## Participants|## Action Items|## Key Points/i},
    {"plan", ~r/## Non-Negotiables|## Time Blocks|week plan|execution plan/i},
    {"brief", ~r/## Key Messages|## Call to Action|## Objective/i},
    {"proposal", ~r/## Proposed|## Investment|## Timeline|## Deliverables/i},
    {"pitch", ~r/pitch|value proposition|problem.solution/i},
    {"changelog", ~r/CHANGELOG|## Added|## Changed|## Fixed|## Removed/i},
    {"readme", ~r/## Installation|## Usage|## Getting Started|## Contributing/i},
    {"note", ~r/.*/}
  ]

  @mode_patterns [
    {:code, ~r/```[a-z]+\n|defmodule|def |class |function |import |require /},
    {:data, ~r/^\s*[\[\{]/m},
    {:visual, ~r/!\[.*\]\(.*\)|<img |<svg /},
    {:mixed, ~r/```/},
    {:linguistic, ~r/.*/}
  ]

  @type_patterns [
    {:decide, ~r/decided:|we decided|decision:|approved:|rejected:/i},
    {:commit, ~r/committing to|we will|I will|action item:|committed:/i},
    {:direct, ~r/please |you need to|you must|do this|call to action|next step:|immediately/i},
    {:express, ~r/feeling|frustrated|excited|worried|concerned|hope|proud/i},
    {:inform, ~r/.*/}
  ]

  # File extensions → context mode and format hints
  @code_extensions ~w[.ex .exs .py .js .ts .jsx .tsx .go .rs .rb .java .c .cpp .h .sh .bash .zsh]
  @data_extensions ~w[.json .yaml .yml .toml .csv .xml]
  @doc_extensions ~w[.md .txt .rst .adoc]
  @binary_extensions ~w[.pdf .docx .pptx .xlsx .png .jpg .jpeg .gif .svg .mp4 .mp3]

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Detects the context type for content, optionally using path hints.

  Returns one of: `:signal`, `:resource`, `:memory`, `:skill`.

  Detection priority:
  1. Path-based: `_memories/` → :memory, `_skills/` → :skill
  2. YAML frontmatter present → :signal
  3. Markdown content matching signal patterns → :signal
  4. Default → :resource
  """
  @spec detect_type(String.t(), keyword()) :: Context.context_type()
  def detect_type(content, opts \\ []) do
    path = Keyword.get(opts, :path, "")

    cond do
      path_is_memory?(path) -> :memory
      path_is_skill?(path) -> :skill
      has_signal_frontmatter?(content) -> :signal
      looks_like_signal?(content, path) -> :signal
      true -> :resource
    end
  end

  @doc """
  Classifies content and returns a Context struct.

  This is the main entry point. It:
  1. Detects the context type
  2. For :signal types, runs the full Mode + Genre + Type + Format + Structure signal classification
  3. For all types, generates L0/L1/L2 summaries
  4. Returns a fully populated Context (caller must set: id, uri, created_at, modified_at)
  """
  @spec classify_context(String.t(), keyword()) :: Context.t()
  def classify_context(content, opts \\ []) when is_binary(content) do
    path = Keyword.get(opts, :path, "")
    ctx_type = Keyword.get(opts, :type) || detect_type(content, opts)
    known_entities = Keyword.get(opts, :known_entities, [])

    {title, l0_abstract, l1_overview, signal} =
      case ctx_type do
        :signal ->
          sig = classify(content, opts)
          {sig.title, sig.l0_summary, sig.l1_description, sig}

        _ ->
          ext = Path.extname(path)
          t = extract_title_generic(content, path)
          l0 = generate_resource_l0(t, ctx_type, path)
          l1 = generate_l1_description(content, ext)
          {t, l0, l1, nil}
      end

    node = if signal, do: signal.node, else: infer_node_from_path(path)
    entities = extract_entities(content, known_entities)

    %Context{
      id: nil,
      uri: nil,
      type: ctx_type,
      path: path,
      title: title,
      content: content,
      l0_abstract: l0_abstract,
      l1_overview: l1_overview,
      signal: signal,
      node: node,
      sn_ratio: (signal && signal.sn_ratio) || 0.5,
      entities: entities,
      created_at: nil,
      modified_at: nil,
      routed_to: (signal && signal.routed_to) || [],
      metadata: build_file_metadata(path)
    }
  end

  @doc """
  Classifies raw markdown content into signal dimensions (Mode + Genre + Type + Format + Structure).

  Returns a partial Signal struct with all classification fields populated.
  The caller must set: id, path, content, created_at, modified_at.

  This function is kept as the primary signal classification entry point for
  backward compatibility.
  """
  @spec classify(String.t(), keyword()) :: Signal.t()
  def classify(content, opts \\ []) when is_binary(content) do
    {frontmatter, body} = parse_frontmatter(content)
    known_entities = Keyword.get(opts, :known_entities, [])

    mode = extract_mode(frontmatter, body)
    genre = extract_genre(frontmatter, body)
    type = extract_type(frontmatter, body)
    format = extract_format(frontmatter)
    structure = extract_structure(frontmatter)
    node = extract_node(frontmatter)
    sn_ratio = extract_sn_ratio(frontmatter)
    entities = extract_entities(body, known_entities)
    valid_from = extract_date(frontmatter, "valid_from")
    valid_until = extract_date(frontmatter, "valid_until")
    supersedes = get_in(frontmatter, ["signal", "supersedes"])

    title = extract_title(body, frontmatter)
    l0_summary = generate_l0_summary(title, genre, node, sn_ratio)
    l1_description = generate_l1_description(body, "")

    %Signal{
      id: nil,
      path: nil,
      title: title,
      mode: mode,
      genre: genre,
      type: type,
      format: format,
      structure: structure,
      created_at: nil,
      modified_at: nil,
      valid_from: valid_from,
      valid_until: valid_until,
      supersedes: supersedes,
      node: node,
      sn_ratio: sn_ratio,
      entities: entities,
      l0_summary: l0_summary,
      l1_description: l1_description,
      content: content,
      routed_to: [],
      score: nil
    }
  end

  # ─── Per-chunk classification (Phase 4) ──────────────────────────────────

  @doc """
  Classify a single chunk from Phase 3's Decomposer.

  Unlike `classify/2` which builds a full `%Signal{}` from a whole document,
  this runs on chunk-scale text and produces a `%Classification{}` row
  keyed by `chunk.id`. Frontmatter is not re-parsed (the parent doc's
  frontmatter already set the modality / format); chunk-level classification
  focuses on what varies across chunks: genre, signal_type, structure.

  Options:
    * `:mode_default`   — used when the chunk offers no mode signal (default: inherit from chunk.modality)
    * `:format_default` — fallback format (default: inferred from chunk.modality)
  """
  @spec classify_chunk(Chunk.t(), keyword()) :: {:ok, Classification.t()}
  def classify_chunk(%Chunk{} = chunk, opts \\ []) do
    text = chunk.text || ""
    frontmatter = %{}

    mode = classify_chunk_mode(chunk, opts) || extract_mode(frontmatter, text)
    genre = extract_genre(frontmatter, text)
    signal_type = extract_type(frontmatter, text)
    format = classify_chunk_format(chunk, opts) || extract_format(frontmatter)
    structure = extract_structure(frontmatter)
    sn_ratio = chunk_sn_ratio(text)
    confidence = chunk_confidence(mode, genre, signal_type, format, structure)

    {:ok,
     Classification.new(
       chunk_id: chunk.id,
       tenant_id: chunk.tenant_id,
       mode: mode,
       genre: genre,
       signal_type: signal_type,
       format: format,
       structure: structure,
       sn_ratio: sn_ratio,
       confidence: confidence
     )}
  end

  @doc "Classify every chunk in a ChunkTree. Returns the list in tree order."
  @spec classify_tree(ChunkTree.t(), keyword()) :: {:ok, [Classification.t()]}
  def classify_tree(%ChunkTree{chunks: chunks}, opts \\ []) do
    classifications = Enum.map(chunks, fn c -> elem(classify_chunk(c, opts), 1) end)
    {:ok, classifications}
  end

  # Prefer the chunk's modality (set by the Parser) — it's the ground truth for
  # mode. Fall back to heuristics on body text.
  defp classify_chunk_mode(%Chunk{modality: :code}, _opts), do: :code
  defp classify_chunk_mode(%Chunk{modality: :image}, _opts), do: :visual
  defp classify_chunk_mode(%Chunk{modality: :audio}, _opts), do: :linguistic
  defp classify_chunk_mode(%Chunk{modality: :video}, _opts), do: :mixed
  defp classify_chunk_mode(%Chunk{modality: :data}, _opts), do: :data
  defp classify_chunk_mode(%Chunk{modality: :mixed}, _opts), do: :mixed
  defp classify_chunk_mode(_, opts), do: Keyword.get(opts, :mode_default)

  defp classify_chunk_format(%Chunk{modality: :code}, _opts), do: :code
  defp classify_chunk_format(%Chunk{modality: :data}, _opts), do: :json
  defp classify_chunk_format(%Chunk{modality: :image}, _opts), do: :unknown
  defp classify_chunk_format(%Chunk{modality: :audio}, _opts), do: :unknown
  defp classify_chunk_format(%Chunk{modality: :video}, _opts), do: :unknown
  defp classify_chunk_format(_, opts), do: Keyword.get(opts, :format_default)

  # Cheap sn_ratio heuristic: higher when the chunk is dense + structured.
  defp chunk_sn_ratio(text) do
    len = byte_size(text)

    cond do
      len == 0 -> 0.0
      len < 40 -> 0.3
      true -> entropy_like_density(text)
    end
  end

  defp entropy_like_density(text) do
    # Rough measure: ratio of unique tokens to total tokens, mapped to 0.0..1.0.
    tokens =
      text
      |> String.downcase()
      |> String.split(~r/[^a-z0-9]+/, trim: true)

    total = length(tokens)

    if total == 0 do
      0.0
    else
      unique = tokens |> Enum.uniq() |> length()
      ratio = unique / total
      # Squash to 0.4..0.9 so chunks don't all land at the extremes
      Float.round(0.4 + ratio * 0.5, 3)
    end
  end

  # Confidence rises with how many of the five dimensions were resolved.
  defp chunk_confidence(mode, genre, signal_type, format, structure) do
    resolved =
      [mode, genre, signal_type, format, structure]
      |> Enum.count(&(not is_nil(&1)))

    Float.round(resolved / 5.0, 2)
  end

  @doc """
  Parses YAML frontmatter from a markdown document.
  Returns `{frontmatter_map, body_string}`.
  """
  @spec parse_frontmatter(String.t()) :: {map(), String.t()}
  def parse_frontmatter(content) when is_binary(content) do
    case Regex.run(~r/\A---\r?\n(.*?)\r?\n---\r?\n?(.*)\z/s, content) do
      [_, yaml_block, body] ->
        case YamlElixir.read_from_string(yaml_block) do
          {:ok, data} when is_map(data) -> {data, body}
          _ -> {%{}, content}
        end

      _ ->
        {%{}, content}
    end
  end

  @doc "Returns the file format category for a given extension."
  @spec file_format(String.t()) :: :code | :data | :document | :binary | :unknown
  def file_format(ext) when is_binary(ext) do
    cond do
      ext in @code_extensions -> :code
      ext in @data_extensions -> :data
      ext in @doc_extensions -> :document
      ext in @binary_extensions -> :binary
      true -> :unknown
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Context type detection
  # ---------------------------------------------------------------------------

  defp path_is_memory?(path), do: String.contains?(path, "_memories/")
  defp path_is_skill?(path), do: String.contains?(path, "_skills/")

  defp has_signal_frontmatter?(content) do
    case parse_frontmatter(content) do
      {fm, _body} when map_size(fm) > 0 ->
        # Frontmatter with "signal:" block or "node:" key is a signal
        Map.has_key?(fm, "signal") or Map.has_key?(fm, "node")

      _ ->
        false
    end
  end

  defp looks_like_signal?(_content, path) do
    ext = Path.extname(path)

    # Markdown files in org folders look like signals
    is_org_markdown =
      ext == ".md" and
        Enum.any?(
          ~w[entity-company person-operator product-customer-portal
             project-platform-launch operation-weekly-review operation-revenue
             operation-delivery learning-research-library inbox team],
          &String.contains?(path, &1)
        )

    is_org_markdown
  end

  # ---------------------------------------------------------------------------
  # Private: Resource/Generic extraction
  # ---------------------------------------------------------------------------

  defp extract_title_generic(content, path) do
    ext = Path.extname(path)

    cond do
      ext in @doc_extensions ->
        # Try H1 for markdown/text
        case Regex.run(~r/^#\s+(.+)$/m, content) do
          [_, title] -> String.trim(title)
          _ -> Path.basename(path, ext) |> humanize_filename()
        end

      ext in @code_extensions ->
        # Use filename as title for code files
        Path.basename(path) |> humanize_filename()

      ext in @data_extensions ->
        Path.basename(path) |> humanize_filename()

      path != "" ->
        Path.basename(path) |> humanize_filename()

      true ->
        "Untitled"
    end
  end

  defp humanize_filename(name) do
    name
    |> String.replace(~r/[-_]/, " ")
    |> String.split(".")
    |> List.first("")
    |> String.trim()
  end

  defp generate_resource_l0(title, ctx_type, path) do
    ext = Path.extname(path)
    fmt = file_format(ext)
    type_str = ctx_type |> to_string() |> String.upcase()
    "#{type_str} | #{fmt} | #{title}"
  end

  defp generate_l1_description(content, ext) when is_binary(ext) do
    # For code files, grab comments and function signatures
    if ext in @code_extensions do
      extract_code_overview(content)
    else
      content
      |> String.replace(~r/^#+\s+.+$/m, "")
      |> String.replace(~r/\n{3,}/, "\n\n")
      |> String.replace(~r/---\r?\n.*?---\r?\n/s, "")
      |> String.trim()
      |> truncate(500)
    end
  end

  defp extract_code_overview(content) do
    # Extract module/function definitions and top-level comments
    lines = String.split(content, "\n")

    overview_lines =
      lines
      |> Enum.filter(fn line ->
        trimmed = String.trim(line)

        String.starts_with?(trimmed, "#") or
          String.starts_with?(trimmed, "//") or
          String.starts_with?(trimmed, "/*") or
          Regex.match?(~r/^(def|defmodule|defp|class|function|export|module)\b/, trimmed)
      end)
      |> Enum.take(20)

    if overview_lines == [] do
      truncate(content, 500)
    else
      Enum.join(overview_lines, "\n") |> truncate(500)
    end
  end

  defp build_file_metadata(path) when is_binary(path) and path != "" do
    %{
      "extension" => Path.extname(path),
      "filename" => Path.basename(path),
      "format" => file_format(Path.extname(path)) |> to_string()
    }
  end

  defp build_file_metadata(_), do: %{}

  defp infer_node_from_path(path) when is_binary(path) do
    root = Application.get_env(:optimal_engine, :root_path, "")
    relative = String.replace_prefix(path, root <> "/", "")
    top = relative |> String.split("/") |> List.first("")
    folder_to_node(top)
  end

  # Derive the node from the path.
  defp folder_to_node("entity-company"), do: "entity-company"
  defp folder_to_node("person-operator"), do: "person-operator"
  defp folder_to_node("product-customer-portal"), do: "product-customer-portal"
  defp folder_to_node("project-platform-launch"), do: "project-platform-launch"
  defp folder_to_node("operation-weekly-review"), do: "operation-weekly-review"
  defp folder_to_node("operation-revenue"), do: "operation-revenue"
  defp folder_to_node("operation-delivery"), do: "operation-delivery"
  defp folder_to_node("learning-research-library"), do: "learning-research-library"
  defp folder_to_node("team"), do: "team"
  defp folder_to_node("inbox"), do: "inbox"
  defp folder_to_node("docs"), do: "resources"
  defp folder_to_node(_), do: "inbox"

  # ---------------------------------------------------------------------------
  # Private: Signal extraction helpers
  # ---------------------------------------------------------------------------

  defp extract_mode(fm, body) do
    raw =
      get_in(fm, ["signal", "mode"]) ||
        Map.get(fm, "mode")

    if raw do
      parse_mode(raw)
    else
      detect_mode(body)
    end
  end

  defp parse_mode("linguistic"), do: :linguistic
  defp parse_mode("visual"), do: :visual
  defp parse_mode("code"), do: :code
  defp parse_mode("data"), do: :data
  defp parse_mode("mixed"), do: :mixed
  defp parse_mode(_), do: :linguistic

  defp detect_mode(body) do
    Enum.find_value(@mode_patterns, :linguistic, fn {mode, pattern} ->
      if Regex.match?(pattern, body), do: mode
    end)
  end

  defp extract_genre(fm, body) do
    raw =
      get_in(fm, ["signal", "genre"]) ||
        Map.get(fm, "genre")

    cond do
      raw && is_binary(raw) && GenreRegistry.valid?(raw) -> raw
      raw && is_binary(raw) -> raw
      true -> detect_genre(body)
    end
  end

  # Two-stage detection against the SINGLE canonical genre registry:
  # 1. high-precision regex cues (@genre_patterns) for the most common genres
  # 2. fall back to keyword scoring across all 142 registry genres
  # 3. graceful default of "note" when nothing matches
  defp detect_genre(body) do
    precise =
      Enum.find_value(@genre_patterns, nil, fn {genre, pattern} ->
        if Regex.match?(pattern, body), do: genre
      end)

    case precise do
      nil -> GenreRegistry.detect(body)
      "note" -> GenreRegistry.detect(body)
      genre -> genre
    end
  end

  defp extract_type(fm, body) do
    raw =
      get_in(fm, ["signal", "type"]) ||
        Map.get(fm, "type")

    if raw do
      parse_type(raw)
    else
      detect_type_from_content(body)
    end
  end

  defp parse_type("direct"), do: :direct
  defp parse_type("inform"), do: :inform
  defp parse_type("commit"), do: :commit
  defp parse_type("decide"), do: :decide
  defp parse_type("express"), do: :express
  defp parse_type(_), do: :inform

  defp detect_type_from_content(body) do
    Enum.find_value(@type_patterns, :inform, fn {type, pattern} ->
      if Regex.match?(pattern, body), do: type
    end)
  end

  defp extract_format(fm) do
    raw =
      get_in(fm, ["signal", "format"]) ||
        Map.get(fm, "format")

    case raw do
      "markdown" -> :markdown
      "code" -> :code
      "json" -> :json
      "yaml" -> :yaml
      _ -> :markdown
    end
  end

  defp extract_structure(fm) do
    get_in(fm, ["signal", "structure"]) ||
      Map.get(fm, "structure") ||
      ""
  end

  defp extract_node(fm) do
    raw = Map.get(fm, "node") || get_in(fm, ["signal", "node"])
    if is_binary(raw), do: raw, else: "inbox"
  end

  defp extract_sn_ratio(fm) do
    raw =
      get_in(fm, ["signal", "sn_ratio"]) ||
        Map.get(fm, "sn_ratio")

    case raw do
      v when is_float(v) -> v |> max(0.0) |> min(1.0)
      v when is_integer(v) -> (v / 1.0) |> max(0.0) |> min(1.0)
      _ -> 0.6
    end
  end

  defp extract_date(fm, field) do
    raw = Map.get(fm, field) || get_in(fm, ["signal", field])

    case raw do
      str when is_binary(str) ->
        case DateTime.from_iso8601(str <> "T00:00:00Z") do
          {:ok, dt, _} -> dt
          _ -> try_date_parse(str)
        end

      _ ->
        nil
    end
  end

  defp try_date_parse(str) do
    case Date.from_iso8601(str) do
      {:ok, date} -> DateTime.new!(date, ~T[00:00:00], "Etc/UTC")
      _ -> nil
    end
  end

  defp extract_title(body, fm) do
    fm_title = Map.get(fm, "title") || get_in(fm, ["signal", "title"])

    if is_binary(fm_title) && String.length(fm_title) > 0 do
      fm_title
    else
      case Regex.run(~r/^#\s+(.+)$/m, body) do
        [_, title] -> String.trim(title)
        _ -> "Untitled"
      end
    end
  end

  defp extract_entities(body, known_entities) when is_list(known_entities) do
    known_entities
    |> Enum.filter(fn name ->
      String.length(name) > 2 && Regex.match?(~r/\b#{Regex.escape(name)}\b/i, body)
    end)
    |> Enum.uniq()
  end

  defp generate_l0_summary(title, genre, node, sn_ratio) do
    "#{String.upcase(genre)} | #{node} | #{title} [S/N: #{Float.round(sn_ratio, 1)}]"
  end

  defp truncate(str, max_len) do
    if String.length(str) > max_len do
      str
      |> String.slice(0, max_len)
      |> String.replace(~r/\s+\S+$/, "")
      |> Kernel.<>("...")
    else
      str
    end
  end
end
