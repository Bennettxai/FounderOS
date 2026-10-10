defmodule Mix.Tasks.Optimal.Grep do
  @shortdoc "Hybrid semantic + literal grep over a workspace (signal-trace aware)"

  @moduledoc """
  Grep the indexed workspace using the full retrieval pipeline — hybrid BM25 +
  vector search with intent / scale / modality filtering.

  Unlike a file-system grep, every match carries the complete signal trace:
  slug, scale, intent, sn_ratio, modality. Use this to do typed cued recall
  ("find all `commit_action` chunks mentioning pricing in the academy node")
  as well as plain literal substring search.

  ## Usage

      mix optimal.grep "query"
      mix optimal.grep "query" [path_prefix] [options]

  ## Arguments

      query         The search term (required)
      path_prefix   Optional: restrict to a node slug or slug prefix
                    e.g. "operation-weekly-review" or "operation-weekly-review/" or "operation-weekly-review/pricing"

  ## Options

      -w, --workspace   Workspace slug (default: "default")
      -n, --node        Alias for path_prefix — restrict to this node slug
      -i, --intent      Filter by intent (one of 10 canonical values, see below)
      -s, --scale       Filter by scale: document | section | paragraph | chunk
      -m, --modality    Filter by modality, e.g. text | image | audio
      -l, --limit       Max results (default 25)
      -L, --literal     Force literal FTS match — skip semantic/vector search
          --json        Output full JSON array instead of human-readable text

  ## Intent values

      request_info    propose_decision    record_fact    express_concern
      commit_action   reference           narrate        reflect
      specify         measure

  ## Examples

      # Semantic grep for "pricing" across the default workspace
      mix optimal.grep "pricing"

      # Restrict to the academy node, section scale only
      mix optimal.grep "pricing" operation-weekly-review/ --scale section

      # All commit_action chunks mentioning "launch"
      mix optimal.grep "launch" --intent commit_action

      # Force literal match, output JSON
      mix optimal.grep "Academy pricing decision" --literal --json

      # Search a specific workspace
      mix optimal.grep "infrastructure cost" --workspace engineering --limit 10

  ## Output format (default)

      <slug>:<scale>  intent=<intent>  sn=<sn>  modality=<modality>
        <snippet>

  ## Output format (--json)

      [
        {
          "slug": "operation-weekly-review",
          "scale": "section",
          "intent": "record_fact",
          "sn_ratio": 0.82,
          "modality": "text",
          "snippet": "…",
          "score": 0.943
        },
        …
      ]
  """

  use Mix.Task

  alias OptimalEngine.Retrieval.Grep

  @default_limit 25

  @valid_intents ~w(
    request_info propose_decision record_fact express_concern commit_action
    reference narrate reflect specify measure
  )

  @valid_scales ~w(document section paragraph chunk)

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        strict: [
          workspace: :string,
          node: :string,
          intent: :string,
          scale: :string,
          modality: :string,
          limit: :integer,
          literal: :boolean,
          json: :boolean
        ],
        aliases: [
          w: :workspace,
          n: :node,
          i: :intent,
          s: :scale,
          m: :modality,
          l: :limit,
          L: :literal
        ]
      )

    {query, path_prefix} =
      case positional do
        [] ->
          Mix.raise("Usage: mix optimal.grep \"query\" [path_prefix] [options]")

        [q] ->
          {q, Keyword.get(opts, :node)}

        [q, path | _] ->
          # Allow --node to override positional path if given
          {q, Keyword.get(opts, :node, path)}
      end

    workspace_slug = Keyword.get(opts, :workspace, "default")
    workspace_id = resolve_workspace_id(workspace_slug)

    {default_limit, default_scale} = workspace_defaults(workspace_slug)

    limit = Keyword.get(opts, :limit, default_limit)
    scale = Keyword.get(opts, :scale) || default_scale
    intent = Keyword.get(opts, :intent)
    modality = Keyword.get(opts, :modality)
    literal? = Keyword.get(opts, :literal, false)
    json? = Keyword.get(opts, :json, false)

    # Validate intent / scale before hitting the engine
    with :ok <- validate_intent_opt(intent),
         :ok <- validate_scale_opt(scale) do
      grep_opts =
        [
          workspace_id: workspace_id,
          limit: limit,
          literal: literal?
        ]
        |> maybe_put(:intent, intent && String.to_atom(intent))
        |> maybe_put(:scale, scale && String.to_atom(scale))
        |> maybe_put(:modality, modality && safe_atom(modality))
        |> maybe_put(:path_prefix, path_prefix)

      unless json?, do: print_header(query, workspace_slug, opts)

      case Grep.grep(query, grep_opts) do
        {:ok, []} ->
          if json? do
            IO.puts("[]")
          else
            IO.puts(IO.ANSI.yellow() <> "  No matches found." <> IO.ANSI.reset())
          end

        {:ok, matches} ->
          if json? do
            print_json(matches)
          else
            Enum.each(matches, &print_match/1)

            IO.puts(
              "\n" <>
                IO.ANSI.bright() <>
                "#{length(matches)} match(es)" <>
                IO.ANSI.reset()
            )
          end

        {:error, reason} ->
          Mix.shell().error("grep failed: #{inspect(reason)}")
          System.halt(1)
      end
    else
      {:error, msg} ->
        Mix.shell().error(msg)
        System.halt(1)
    end
  end

  # ---------------------------------------------------------------------------
  # Workspace resolution
  # ---------------------------------------------------------------------------

  # Resolve a workspace slug to its stored id. Falls back to using the slug
  # directly (backwards-compatible with pre-Phase-1.5 single-workspace setups).
  defp resolve_workspace_id("default"), do: "default"

  defp resolve_workspace_id(slug) do
    case OptimalEngine.Workspace.get_by_slug(
           slug,
           OptimalEngine.Tenancy.Tenant.default_id()
         ) do
      {:ok, ws} -> ws.id
      _ -> slug
    end
  end

  # Read scale + limit defaults from Workspace.Config if available.
  # Workspace.Config is built by a parallel agent — call it defensively.
  defp workspace_defaults(slug) do
    defaults = %{limit: @default_limit, scale: nil}

    config =
      try do
        apply(OptimalEngine.Workspace.Config, :get_section, [slug, :grep, defaults])
      rescue
        UndefinedFunctionError -> defaults
        _ -> defaults
      catch
        :exit, _ -> defaults
      end

    limit = Map.get(config, :limit, @default_limit)
    scale = Map.get(config, :scale)
    {limit, scale}
  end

  # ---------------------------------------------------------------------------
  # Output: text
  # ---------------------------------------------------------------------------

  defp print_header(query, workspace_slug, opts) do
    IO.puts("")

    IO.puts(
      IO.ANSI.bright() <>
        IO.ANSI.cyan() <>
        "[optimal.grep]" <>
        IO.ANSI.reset() <>
        " Query: " <>
        IO.ANSI.bright() <>
        "\"#{query}\"" <>
        IO.ANSI.reset()
    )

    IO.puts("  workspace: #{workspace_slug}")

    if p = Keyword.get(opts, :node), do: IO.puts("  path:      #{p}")
    if i = Keyword.get(opts, :intent), do: IO.puts("  intent:    #{i}")
    if s = Keyword.get(opts, :scale), do: IO.puts("  scale:     #{s}")
    if m = Keyword.get(opts, :modality), do: IO.puts("  modality:  #{m}")
    if Keyword.get(opts, :literal, false), do: IO.puts("  mode:      literal")

    IO.puts("")
  end

  defp print_match(%{
         slug: slug,
         scale: scale,
         intent: intent,
         sn_ratio: sn,
         modality: modality,
         snippet: snippet,
         score: score
       }) do
    sn_str = if sn, do: Float.round(sn * 1.0, 2) |> to_string(), else: "n/a"
    intent_str = if intent, do: to_string(intent), else: "—"
    modality_str = if modality, do: to_string(modality), else: "—"
    score_str = Float.round(score * 1.0, 3) |> to_string()

    # Header line: slug:scale  intent=X  sn=Y  modality=Z  score=S
    header =
      IO.ANSI.green() <>
        "#{slug}:#{scale}" <>
        IO.ANSI.reset() <>
        "  intent=" <>
        IO.ANSI.yellow() <>
        intent_str <>
        IO.ANSI.reset() <>
        "  sn=" <>
        sn_str <>
        "  modality=" <>
        modality_str <>
        "  score=" <>
        score_str

    IO.puts("  #{header}")
    IO.puts("    #{snippet}")
    IO.puts("")
  end

  # ---------------------------------------------------------------------------
  # Output: JSON
  # ---------------------------------------------------------------------------

  defp print_json(matches) do
    serializable =
      Enum.map(matches, fn m ->
        %{
          slug: m.slug,
          scale: to_string(m.scale),
          intent: if(m.intent, do: to_string(m.intent), else: nil),
          sn_ratio: m.sn_ratio,
          modality: if(m.modality, do: to_string(m.modality), else: nil),
          snippet: m.snippet,
          score: m.score
        }
      end)

    IO.puts(Jason.encode!(serializable, pretty: true))
  end

  # ---------------------------------------------------------------------------
  # Validation
  # ---------------------------------------------------------------------------

  defp validate_intent_opt(nil), do: :ok

  defp validate_intent_opt(intent) do
    if intent in @valid_intents do
      :ok
    else
      {:error, "Unknown intent '#{intent}'. Valid values: #{Enum.join(@valid_intents, " | ")}"}
    end
  end

  defp validate_scale_opt(nil), do: :ok

  defp validate_scale_opt(scale) do
    if scale in @valid_scales do
      :ok
    else
      {:error, "Unknown scale '#{scale}'. Valid values: #{Enum.join(@valid_scales, " | ")}"}
    end
  end

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, val), do: Keyword.put(opts, key, val)

  defp safe_atom(str) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      ArgumentError -> String.to_atom(str)
    end
  end
end
