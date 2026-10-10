defmodule Mix.Tasks.Optimal.Migrate do
  @shortdoc "Apply pending database migrations"

  @moduledoc """
  Applies any pending schema migrations.

  The engine's `Store` runs migrations automatically at boot, so this task is
  primarily useful for:

    * explicitly running migrations against a detached database file
    * verifying the migration state of a store
    * listing the applied versions

  ## Usage

      mix optimal.migrate           # apply pending
      mix optimal.migrate --status  # show version state, no writes

  """

  use Mix.Task

  alias OptimalEngine.Store.Migrations

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, _, _} = OptionParser.parse(args, strict: [status: :boolean])

    if opts[:status] do
      status()
    else
      apply_pending()
    end
  end

  # ── Apply ────────────────────────────────────────────────────────────────

  defp apply_pending do
    db = open_db!()

    Migrations.run(db)

    applied = Migrations.applied_versions(db) |> MapSet.to_list() |> Enum.sort()

    IO.puts("")
    IO.puts("  Migration state")
    IO.puts("  " <> String.duplicate("─", 40))
    IO.puts("  Applied versions: #{Enum.join(applied, ", ")}")
    IO.puts("")

    Exqlite.Sqlite3.close(db)
  end

  # ── Status ───────────────────────────────────────────────────────────────

  defp status do
    db = open_db!()

    applied = Migrations.applied_versions(db)

    all =
      Migrations.all()
      |> Enum.map(fn {v, d, _s} -> {v, d, MapSet.member?(applied, v)} end)

    IO.puts("")
    IO.puts("  Migration status (#{length(all)} total)")
    IO.puts("  " <> String.duplicate("─", 60))

    Enum.each(all, fn {v, desc, applied?} ->
      mark = if applied?, do: "✓", else: "·"
      IO.puts("  #{mark} #{String.pad_leading(Integer.to_string(v), 3)}  #{desc}")
    end)

    IO.puts("")

    pending = Enum.reject(all, fn {_, _, a?} -> a? end) |> length()

    if pending == 0 do
      IO.puts("  All migrations applied.")
    else
      IO.puts("  #{pending} pending — run `mix optimal.migrate` to apply.")
    end

    IO.puts("")

    Exqlite.Sqlite3.close(db)
  end

  # ── Helpers ──────────────────────────────────────────────────────────────

  defp open_db! do
    db_path = Application.fetch_env!(:optimal_engine, :db_path)
    File.mkdir_p!(Path.dirname(db_path))
    {:ok, db} = Exqlite.Sqlite3.open(db_path)
    db
  end
end
