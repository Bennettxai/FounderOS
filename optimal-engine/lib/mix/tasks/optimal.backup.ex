defmodule Mix.Tasks.Optimal.Backup do
  @shortdoc "Create a live backup of the engine database"

  @moduledoc """
  Online SQLite backup — runs without blocking the engine.

  ## Usage

      mix optimal.backup /var/backups/engine-$(date +%F).db
      mix optimal.backup /tmp/snap.db --verify

  ## Options

    --verify   Run PRAGMA integrity_check on the result before exit.

  ## Related

      mix optimal.restore <path>   — restore a backup (engine must be stopped)
  """

  use Mix.Task

  alias OptimalEngine.Backup

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, _} = OptionParser.parse(args, strict: [verify: :boolean])

    target =
      case rest do
        [path | _] -> path
        [] -> Mix.raise("Usage: mix optimal.backup <path>")
      end

    case Backup.create(target) do
      {:ok, info} ->
        IO.puts("Backup complete.")
        IO.puts("  target:           #{info.target}")
        IO.puts("  size_bytes:       #{info.size_bytes}")
        IO.puts("  rows_backed_up:   #{info.rows_backed_up}")
        IO.puts("  duration_ms:      #{info.duration_ms}")

        if Keyword.get(parsed, :verify, false) do
          case Backup.verify(target) do
            {:ok, :ok} ->
              IO.puts("  integrity_check:  OK")

            {:ok, issues} ->
              IO.puts("  integrity_check:  FAILED — #{inspect(issues)}")
              System.halt(1)

            {:error, reason} ->
              IO.puts("  integrity_check:  ERROR — #{inspect(reason)}")
              System.halt(1)
          end
        end

      {:error, reason} ->
        Mix.raise("Backup failed: #{inspect(reason)}")
    end
  end
end
