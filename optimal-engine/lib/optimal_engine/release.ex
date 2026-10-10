defmodule OptimalEngine.Release do
  @moduledoc """
  Release-time entry points invoked from the release runtime.

  `mix release` builds the engine into a self-contained OTP tarball
  with its own ERTS. When that tarball is deployed, the normal
  Mix-driven boot flow (migrations, seeds, sanity checks) is gone —
  this module is what operators call instead:

      bin/optimal eval 'OptimalEngine.Release.migrate()'
      bin/optimal eval 'OptimalEngine.Release.preflight()'
      bin/optimal eval 'OptimalEngine.Release.backup("/backups/$(date +%F).db")'

  Every function here must:
    * start only the apps it actually needs (not the full supervision tree)
    * return a status / exit cleanly for scripted use
    * never assume an interactive console — no IO.gets, no prompts
  """

  require Logger

  alias OptimalEngine.{Backup, Health, Store}

  @app :optimal_engine

  @doc """
  Apply any pending schema migrations. Safe to run on every boot —
  `Store.Migrations` is idempotent.
  """
  @spec migrate() :: :ok
  def migrate do
    load_app()
    {:ok, _} = Application.ensure_all_started(@app)

    # Store.init/1 already runs migrations during supervisor boot.
    # Log the result so the release operator has a trail.
    case Store.raw_query("SELECT COUNT(*) FROM schema_migrations", []) do
      {:ok, [[n]]} -> Logger.info("[Release] migrations ok — #{n} applied")
      other -> Logger.warning("[Release] migration check returned #{inspect(other)}")
    end

    :ok
  end

  @doc """
  Run a preflight sanity check — meant to exit non-zero and fail the
  deployment if anything critical is down.
  """
  @spec preflight() :: :ok | no_return()
  def preflight do
    load_app()
    {:ok, _} = Application.ensure_all_started(@app)

    result = Health.ready(skip: [:embedder])

    Enum.each(result.checks, fn {name, status} ->
      Logger.info("[Release] preflight #{name}: #{inspect(status)}")
    end)

    if result.ok? do
      :ok
    else
      Logger.error("[Release] preflight FAILED: #{inspect(result.checks)}")
      System.halt(1)
    end
  end

  @doc "Create a backup file. See `OptimalEngine.Backup.create/1`."
  @spec backup(String.t()) :: :ok | no_return()
  def backup(path) when is_binary(path) do
    load_app()
    {:ok, _} = Application.ensure_all_started(@app)

    case Backup.create(path) do
      {:ok, result} ->
        Logger.info(
          "[Release] backup OK — #{result.size_bytes} bytes, " <>
            "#{result.rows_backed_up} rows, #{result.duration_ms}ms → #{result.target}"
        )

        :ok

      {:error, reason} ->
        Logger.error("[Release] backup failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  @doc "Current health status — exits 0 when :up, non-zero otherwise."
  @spec healthcheck() :: :ok | no_return()
  def healthcheck do
    load_app()
    {:ok, _} = Application.ensure_all_started(@app)

    case Health.status() do
      :up ->
        IO.puts("up")
        :ok

      :degraded ->
        IO.puts("degraded")
        System.halt(2)

      :down ->
        IO.puts("down")
        System.halt(1)
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  # `load_app/0` guarantees the app spec is on the path even before
  # the supervisor starts. No-op once the app is loaded.
  defp load_app do
    Application.load(@app)
  rescue
    _ -> :ok
  end
end
