defmodule OptimalEngine.CLI do
  @moduledoc """
  Command-line entry point for the Optimal Engine.

  Exposes the supported `mix optimal.*` task surface as subcommands of a single
  `optimal` binary, so humans, shell scripts, and agent runtimes can use one
  command name.

  ## Usage

      optimal <subcommand> [args...]
      optimal --help
      optimal --version

  ## Subcommands

  The CLI proxies to the same `mix optimal.*` task surface used by developers.
  This keeps the local `optimal` binary as a thin command router while native
  dependencies such as SQLite load through the normal Mix build.
  """

  @version Mix.Project.config()[:version] || "0.1.0"

  @subcommands %{
    # Setup
    "setup" => Mix.Tasks.Optimal.Setup,
    "initiate" => Mix.Tasks.Optimal.Initiate,
    "init" => Mix.Tasks.Optimal.Init,
    "bootstrap" => Mix.Tasks.Optimal.Bootstrap,
    "migrate" => Mix.Tasks.Optimal.Migrate,
    # Intake and signal pipeline
    "ingest" => Mix.Tasks.Optimal.Ingest,
    "capture" => Mix.Tasks.Optimal.Ingest,
    "ingest-workspace" => Mix.Tasks.Optimal.IngestWorkspace,
    "intake" => Mix.Tasks.Optimal.Intake,
    "index" => Mix.Tasks.Optimal.Index,
    "classify" => Mix.Tasks.Optimal.Classify,
    "parse" => Mix.Tasks.Optimal.Parse,
    "decompose" => Mix.Tasks.Optimal.Decompose,
    "embed" => Mix.Tasks.Optimal.Embed,
    "cluster" => Mix.Tasks.Optimal.Cluster,
    "architectures" => Mix.Tasks.Optimal.Architectures,
    # Retrieval
    "search" => Mix.Tasks.Optimal.Search,
    "grep" => Mix.Tasks.Optimal.Grep,
    "read" => Mix.Tasks.Optimal.Read,
    "ls" => Mix.Tasks.Optimal.Ls,
    "l0" => Mix.Tasks.Optimal.L0,
    "assemble" => Mix.Tasks.Optimal.Assemble,
    "rag" => Mix.Tasks.Optimal.Rag,
    "context.refresh-stale" => Mix.Tasks.Optimal.Context.RefreshStale,
    # Topology and projections
    "topology" => Mix.Tasks.Optimal.Topology,
    "wiki" => Mix.Tasks.Optimal.Wiki,
    # Graph analysis
    "graph" => Mix.Tasks.Optimal.Graph,
    "reflect" => Mix.Tasks.Optimal.Reflect,
    "reweave" => Mix.Tasks.Optimal.Reweave,
    "simulate" => Mix.Tasks.Optimal.Simulate,
    "impact" => Mix.Tasks.Optimal.Impact,
    # Learning loop
    "remember" => Mix.Tasks.Optimal.Remember,
    "rethink" => Mix.Tasks.Optimal.Rethink,
    "knowledge" => Mix.Tasks.Optimal.Knowledge,
    # Truth lifecycle
    "claims" => Mix.Tasks.Optimal.Claims,
    "facts" => Mix.Tasks.Optimal.Facts,
    # Connectors, governance, and operations
    "connector" => Mix.Tasks.Optimal.Connector,
    "auth" => Mix.Tasks.Optimal.Auth,
    "compliance" => Mix.Tasks.Optimal.Compliance,
    "backup" => Mix.Tasks.Optimal.Backup,
    # Health & verification
    "health" => Mix.Tasks.Optimal.Health,
    "verify" => Mix.Tasks.Optimal.Verify,
    "stats" => Mix.Tasks.Optimal.Stats,
    "status" => Mix.Tasks.Optimal.Status,
    "reality-check" => Mix.Tasks.Optimal.RealityCheck,
    "eval.run" => Mix.Tasks.Optimal.Eval.Run,
    # HTTP API & visualizer
    "api" => Mix.Tasks.Optimal.Api,
    "graph-ui" => Mix.Tasks.Optimal.GraphUi,
    # Spec tooling
    "spec.drift" => Mix.Tasks.Optimal.Spec.Drift,
    "spec.init" => Mix.Tasks.Optimal.Spec.Init,
    "spec.report" => Mix.Tasks.Optimal.Spec.Report
  }

  @aliases %{
    "ingest_workspace" => "ingest-workspace",
    "find" => "search",
    "signal" => "capture",
    "context.refresh_stale" => "context.refresh-stale",
    "context-refresh-stale" => "context.refresh-stale",
    "context" => "context.refresh-stale",
    "reality_check" => "reality-check",
    "reality" => "reality-check",
    "eval" => "eval.run",
    "graph_ui" => "graph-ui"
  }

  @ordered_groups [
    {"Setup", ["setup", "initiate", "init", "bootstrap", "migrate"]},
    {"Intake and signals",
     [
       "ingest",
       "ingest-workspace",
       "intake",
       "index",
       "classify",
       "parse",
       "decompose",
       "embed",
       "cluster",
       "architectures"
     ]},
    {"Retrieval and context",
     ["search", "grep", "read", "ls", "l0", "assemble", "rag", "context.refresh-stale"]},
    {"Topology and projections", ["topology", "wiki"]},
    {"Graph analysis", ["graph", "reflect", "reweave", "simulate", "impact"]},
    {"Learning loop", ["remember", "rethink", "knowledge"]},
    {"Truth lifecycle", ["claims", "facts"]},
    {"Connectors, governance, and operations", ["connector", "auth", "compliance", "backup"]},
    {"Health, verification, and evaluation",
     ["health", "verify", "stats", "status", "reality-check", "eval.run"]},
    {"Agent convenience loop",
     ["doctor", "boot", "find", "capture", "aware", "note", "lesson", "decision", "task", "close"]},
    {"HTTP API & visualizer", ["api", "graph-ui"]},
    {"Spec tooling", ["spec.drift", "spec.init", "spec.report"]}
  ]

  # Shortdocs are hardcoded here because escripts strip @shortdoc metadata.
  @shortdocs %{
    "setup" => "Create a workspace with starter Nodes, rhythm, projections, and agent SOP",
    "initiate" => "Start a workspace from messy context and apply conservative Node proposals",
    "init" => "Scaffold a markdown workspace directory from the sample template",
    "bootstrap" => "Compile, migrate, ingest sample workspace, and report status",
    "migrate" => "Apply or inspect store migrations",
    "ingest" => "Classify, route, write signal files, and index content",
    "ingest-workspace" => "Ingest a markdown workspace into the engine store",
    "intake" => "Interactive multi-line intake from stdin",
    "index" => "Full reindex of all markdown files",
    "classify" => "Classify content without writing final truth",
    "parse" => "Parse supported files or content",
    "decompose" => "Decompose parsed documents into chunks",
    "embed" => "Generate/store embeddings where configured",
    "cluster" => "Cluster indexed chunks and signals",
    "architectures" => "List registered multimodal architectures and processors",
    "search" => "Hybrid BM25 + temporal + graph-boosted search",
    "grep" => "Literal/content grep over indexed chunks",
    "read" => "Read a context by optimal:// URI at a given tier",
    "ls" => "List contexts under an optimal:// URI",
    "l0" => "Print the always-loaded L0 context (~100 tokens per node)",
    "assemble" => "Build a tiered (L0/L1/L2) context bundle for a topic",
    "rag" => "Return LLM-ready retrieved chunks for a query",
    "context.refresh-stale" => "Refresh stale stored Context Packages",
    "topology" => "Inspect workspaces, Nodes, Node Types, memberships, and skills",
    "wiki" => "List, render, view, and verify wiki/export projections",
    "graph" => "Knowledge graph stats, triangles, clusters, hubs",
    "reflect" => "Find missing edges from entity co-occurrences",
    "reweave" => "Find stale contexts on a topic and suggest updates",
    "simulate" => "Run a 'what if' scenario through the graph",
    "impact" => "Impact analysis for an entity or node",
    "remember" => "Store observations; mine friction patterns",
    "rethink" => "Synthesize observations into actionable knowledge",
    "knowledge" => "Knowledge graph + SICA learning operations",
    "claims" => "List, inspect, promote, and reject Memory Core Claims",
    "facts" => "List and inspect accepted Memory Core Facts",
    "connector" => "Connector registry and sync operations",
    "auth" => "Mint, list, revoke, and delete API keys for apps and remote agents",
    "compliance" => "DSAR, erasure, legal hold, and retention workflows",
    "backup" => "Backup runtime state",
    "health" => "Diagnostic checks on the knowledge base",
    "verify" => "Cold-read test of L0 abstract fidelity",
    "stats" => "Store statistics (counts, sizes, token budgets)",
    "status" => "Runtime status",
    "reality-check" =>
      "Broad backend spine check across store, topology, memory, retrieval, and governance",
    "eval.run" => "Run evaluation datasets",
    "doctor" => "Print local runtime diagnostics and quick status",
    "boot" => "Run doctor, then print always-loaded L0 context",
    "find" => "Alias for search",
    "capture" => "Capture raw text through intake with claim extraction",
    "aware" => "Capture a signal, search related context, and remember a summary",
    "note" => "Remember a lightweight note",
    "lesson" => "Remember a reusable lesson",
    "decision" => "Remember a decision",
    "task" => "Remember an action item",
    "close" => "Remember a close-loop summary",
    "api" => "Start the HTTP API on port 4200",
    "graph-ui" => "Launch the graph visualizer against a running API",
    "spec.drift" => "Detect code changes without spec updates",
    "spec.init" => "Scaffold the .spec/ directory",
    "spec.report" => "Spec coverage and verification summary"
  }

  @doc """
  Main entry point for the escript.

  Takes the raw argv from the OS, parses the subcommand, and delegates to the
  matching `mix optimal.*` task. The escript is intentionally a local wrapper,
  not the production release format.
  """
  @spec main([String.t()]) :: any()
  def main(argv) do
    case argv do
      [] -> print_help_and_exit(0)
      ["--help"] -> print_help_and_exit(0)
      ["-h"] -> print_help_and_exit(0)
      ["help"] -> print_help_and_exit(0)
      ["--version"] -> print_version_and_exit()
      ["-v"] -> print_version_and_exit()
      [sub | rest] -> dispatch(normalize_subcommand(sub), rest)
    end
  end

  @doc false
  @spec subcommands() :: %{String.t() => module()}
  def subcommands, do: @subcommands

  @doc false
  @spec command_names() :: [String.t()]
  def command_names do
    @ordered_groups
    |> Enum.flat_map(fn {_group, names} -> names end)
    |> Enum.uniq()
    |> Enum.sort()
  end

  @doc false
  @spec resolve_subcommand(String.t()) :: {:ok, String.t(), module()} | :error
  def resolve_subcommand(sub) when is_binary(sub) do
    canonical = normalize_subcommand(sub)

    case Map.fetch(@subcommands, canonical) do
      {:ok, module} -> {:ok, canonical, module}
      :error -> :error
    end
  end

  @doc false
  @spec help_text() :: String.t()
  def help_text do
    [
      "Optimal Engine CLI — backend runtime for human and AI workspaces.",
      "",
      "  usage: optimal <subcommand> [args...]",
      "         optimal --help",
      "         optimal --version",
      "",
      "Common first-run flow:",
      "",
      "  optimal doctor",
      "  optimal boot",
      "  optimal reality-check",
      "  optimal setup my-workspace --name \"My Workspace\"",
      "  optimal topology --workspace default:my-workspace",
      "  optimal initiate my-workspace --name \"My Workspace\" --dump setup.md",
      "  optimal find \"query\" --workspace default:my-workspace",
      "  optimal capture \"raw signal\" --workspace default:my-workspace",
      "  optimal aware \"important correction\" --workspace default:my-workspace",
      "  optimal close \"what changed and how verified\"",
      "",
      "Subcommands by group:",
      grouped_help_text(),
      "",
      "Run 'optimal <subcommand> --help' for per-command detail where supported."
    ]
    |> List.flatten()
    |> Enum.join("\n")
  end

  # ── Dispatch ─────────────────────────────────────────────────────────────

  defp dispatch(sub, rest) do
    case sub do
      "doctor" ->
        run_sequence([{"status", ["--quick"]}], 0)

      "boot" ->
        run_sequence([{"status", ["--quick"]}, {"l0", rest}], 0)

      "capture" ->
        run_mix_task("ingest", ensure_text_arg!("signal text", rest))

      "aware" ->
        args = ensure_text_arg!("signal text", rest)
        text = Enum.join(args, " ")
        run_sequence([{"ingest", args}, {"search", [text]}, {"remember", ["Signal: " <> text]}], 0)

      "note" ->
        remember_with_prefix("Note: ", rest)

      "lesson" ->
        remember_with_prefix("Lesson: ", rest)

      "decision" ->
        remember_with_prefix("Decision: ", rest)

      "task" ->
        remember_with_prefix("Task: ", rest)

      "close" ->
        remember_with_prefix("Close-loop: ", rest)

      _ ->
        dispatch_mix_task(sub, rest)
    end
  end

  defp dispatch_mix_task(sub, rest) do
    case resolve_subcommand(sub) do
      {:ok, canonical, _module} ->
        run_mix_task(canonical, rest)

      :error ->
        IO.puts(:stderr, "optimal: unknown subcommand '#{sub}'")
        IO.puts(:stderr, "")
        print_help(:stderr)
        System.halt(64)
    end
  end

  # ── Mix Delegation ───────────────────────────────────────────────────────

  defp run_mix_task(canonical, rest) do
    task = mix_task_name(canonical)

    case System.find_executable("mix") do
      nil ->
        IO.puts(:stderr, "optimal: cannot find 'mix' on PATH")
        IO.puts(:stderr, "Install Elixir/Mix or run the OTP release/API deployment instead.")
        System.halt(69)

      _mix ->
        {_output, status} =
          System.cmd("mix", [task | rest],
            into: IO.stream(:stdio, :line),
            stderr_to_stdout: true
          )

        System.halt(status)
    end
  end

  defp run_sequence([], status), do: System.halt(status)

  defp run_sequence([{canonical, rest} | remaining], _status) do
    case run_mix_task_inline(canonical, rest) do
      0 -> run_sequence(remaining, 0)
      status -> System.halt(status)
    end
  end

  defp run_mix_task_inline(canonical, rest) do
    task = mix_task_name(canonical)

    case System.find_executable("mix") do
      nil ->
        IO.puts(:stderr, "optimal: cannot find 'mix' on PATH")
        IO.puts(:stderr, "Install Elixir/Mix or run the OTP release/API deployment instead.")
        69

      _mix ->
        {_output, status} =
          System.cmd("mix", [task | rest],
            into: IO.stream(:stdio, :line),
            stderr_to_stdout: true
          )

        status
    end
  end

  defp remember_with_prefix(prefix, rest) do
    args = ensure_text_arg!("memory text", rest)
    run_mix_task("remember", [prefix <> Enum.join(args, " ")])
  end

  defp ensure_text_arg!(label, []) do
    IO.puts(:stderr, "optimal: missing #{label}")
    System.halt(64)
  end

  defp ensure_text_arg!(_label, args), do: args

  defp mix_task_name(canonical) do
    mix_name =
      canonical
      |> String.replace("-", "_")

    "optimal.#{mix_name}"
  end

  # ── Help ─────────────────────────────────────────────────────────────────

  defp print_help_and_exit(code) do
    print_help(:stdio)
    System.halt(code)
  end

  defp print_version_and_exit do
    IO.puts("optimal #{@version}")
    System.halt(0)
  end

  defp print_help(io) do
    IO.puts(io, help_text())
  end

  defp grouped_help_text do
    Enum.flat_map(@ordered_groups, fn {group, names} ->
      ["", "  #{group}:"] ++
        Enum.map(names, fn name ->
          short = Map.get(@shortdocs, name, "")
          "    #{String.pad_trailing(name, 22)} #{short}"
        end)
    end)
  end

  defp normalize_subcommand(sub) do
    trimmed = String.trim(sub)
    Map.get(@aliases, trimmed, trimmed)
  end
end
