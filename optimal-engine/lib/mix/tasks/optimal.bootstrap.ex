defmodule Mix.Tasks.Optimal.Bootstrap do
  @shortdoc "One-shot: compile, migrate, ingest sample-workspace, report status"

  @moduledoc """
  New-user onboarding in a single command. Runs every step a fresh
  checkout needs before `mix optimal.rag` will return useful results:

    1. Ensure the app is compiled.
    2. Start the supervision tree (which applies schema migrations).
    3. Ingest `sample-workspace/` if present (skipped if missing).
    4. Print a status summary + next-step hints.

  Idempotent — re-run after `git pull` to pick up new migrations or
  updated sample files.

  ## Usage

      mix optimal.bootstrap
      mix optimal.bootstrap --workspace path/to/other-workspace
      mix optimal.bootstrap --skip-seed        # compile + migrate only
  """

  use Mix.Task

  alias OptimalEngine.Health
  alias OptimalEngine.Store

  @impl Mix.Task
  def run(args) do
    {parsed, _, _} =
      OptionParser.parse(args, strict: [workspace: :string, skip_seed: :boolean])

    workspace = Keyword.get(parsed, :workspace, "sample-workspace")
    skip_seed? = Keyword.get(parsed, :skip_seed, false)

    banner("Optimal Engine — bootstrap")

    Mix.Task.run("app.start")
    step("supervision tree up", fn -> Health.live?() end)
    step("migrations applied", fn -> migrations_applied() end)

    unless skip_seed? do
      if File.dir?(workspace) do
        step("ingest #{workspace}/", fn ->
          Mix.Task.rerun("optimal.ingest_workspace", [workspace, "--reset"])
          true
        end)
      else
        IO.puts("  ⚠  #{workspace}/ not found — skipping ingest.")
      end
    end

    IO.puts("")
    summary()
    IO.puts("")
    next_steps()
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp banner(text) do
    IO.puts("")
    IO.puts("  #{text}")
    IO.puts("  " <> String.duplicate("─", 60))
  end

  defp step(label, fun) do
    t0 = System.monotonic_time(:millisecond)
    ok? = fun.() |> truthy?()
    dt = System.monotonic_time(:millisecond) - t0
    tag = if ok?, do: IO.ANSI.green() <> "OK  ", else: IO.ANSI.red() <> "FAIL"
    IO.puts("    #{tag}#{IO.ANSI.reset()}  #{String.pad_trailing(label, 40)} #{dt}ms")
  end

  defp truthy?(true), do: true
  defp truthy?(:ok), do: true
  defp truthy?({:ok, _}), do: true
  defp truthy?(_), do: false

  defp migrations_applied do
    case Store.raw_query("SELECT COUNT(*) FROM schema_migrations", []) do
      {:ok, [[n]]} when is_integer(n) and n > 0 -> true
      _ -> false
    end
  end

  defp summary do
    IO.puts("  Current state")
    IO.puts("  " <> String.duplicate("─", 60))

    rows = [
      count("nodes", "SELECT COUNT(*) FROM nodes"),
      count("contexts (signals)", "SELECT COUNT(*) FROM contexts WHERE type = 'signal'"),
      count("entities", "SELECT COUNT(*) FROM entities"),
      count("chunks", "SELECT COUNT(*) FROM chunks"),
      count("wiki pages", "SELECT COUNT(*) FROM wiki_pages"),
      count("architectures (custom)", "SELECT COUNT(*) FROM architectures"),
      count("events", "SELECT COUNT(*) FROM events")
    ]

    Enum.each(rows, fn {label, n} ->
      IO.puts("    #{String.pad_trailing(label, 30)} #{n}")
    end)
  end

  defp count(label, sql) do
    case Store.raw_query(sql, []) do
      {:ok, [[n]]} -> {label, n}
      _ -> {label, 0}
    end
  end

  defp next_steps do
    IO.puts("  Next")
    IO.puts("  " <> String.duplicate("─", 60))

    IO.puts("""
        mix optimal.rag "customer portal pricing decision" --trace
        mix optimal.search "platform"
        mix optimal.wiki list
        mix optimal.graph hubs
        mix optimal.reality_check --hard

      Launch the desktop:
        config :optimal_engine, :api, enabled: true, port: 4200   # in config/dev.exs
        iex -S mix              # engine + API
        cd desktop && npm run dev
    """)
  end
end
