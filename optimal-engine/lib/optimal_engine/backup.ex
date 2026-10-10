defmodule OptimalEngine.Backup do
  @moduledoc """
  Backup + restore for the engine's SQLite database.

  SQLite's online backup uses `VACUUM INTO 'target.db'` — atomic,
  safe to run while the engine keeps serving, and produces a
  self-contained single file with no WAL trail attached. That one
  file plus any migration metadata is the complete engine state.

  ## Usage

      # Live backup while the engine runs
      OptimalEngine.Backup.create("/var/backups/engine-2026-01-05.db")

      # Restore from a backup (requires supervisor shutdown first —
      # we fail loudly if it's still up)
      OptimalEngine.Backup.restore("/var/backups/engine-2026-01-05.db")

  The backup is not encrypted. Treat the file as sensitive —
  connector credentials live in it (encrypted envelope, but the
  envelope key lives outside the DB by design).
  """

  alias OptimalEngine.Store

  @type backup_result :: %{
          id: String.t(),
          target: String.t(),
          size_bytes: non_neg_integer(),
          rows_backed_up: non_neg_integer(),
          duration_ms: non_neg_integer(),
          checksum_sha256: String.t(),
          integrity_status: String.t()
        }

  @doc """
  Create a backup at `target_path`. Runs online — the engine keeps
  serving throughout. Parent directory must exist.
  """
  @spec create(String.t()) :: {:ok, backup_result()} | {:error, term()}
  def create(target_path) when is_binary(target_path) do
    parent = Path.dirname(target_path)

    with :ok <- ensure_parent_dir(parent),
         :ok <- ensure_no_existing(target_path),
         started <- System.monotonic_time(:millisecond),
         :ok <- vacuum_into(target_path),
         size <- File.stat!(target_path).size,
         rows <- count_backup_rows(target_path),
         checksum <- checksum_sha256(target_path),
         {:ok, integrity} <- verify(target_path),
         integrity_status <- if(integrity == :ok, do: "verified", else: "failed"),
         id <- backup_id(),
         :ok <- catalog_backup(id, target_path, size, checksum, integrity_status) do
      {:ok,
       %{
         id: id,
         target: target_path,
         size_bytes: size,
         rows_backed_up: rows,
         duration_ms: System.monotonic_time(:millisecond) - started,
         checksum_sha256: checksum,
         integrity_status: integrity_status
       }}
    end
  end

  @doc """
  Restore from `source_path`. Refuses to run while the supervisor is
  up — restore must happen pre-boot.

  The target is whatever `:optimal_engine, :db_path` resolves to. We
  don't overwrite without a prior copy of the existing DB at
  `<db_path>.pre-restore-<epoch>` so the operator has a rollback.
  """
  @spec restore(String.t()) :: {:ok, String.t()} | {:error, term()}
  def restore(source_path) when is_binary(source_path) do
    db_path = Application.get_env(:optimal_engine, :db_path)

    with :ok <- refuse_if_supervisor_up(),
         :ok <- ensure_source_exists(source_path),
         {:ok, backup_of_current} <- archive_current(db_path) do
      File.cp!(source_path, db_path)
      {:ok, backup_of_current}
    end
  end

  @doc """
  Quick integrity check on a backup file — opens it as a read-only
  SQLite, runs `PRAGMA integrity_check`, and closes. Returns
  `{:ok, :ok}` for clean DBs or `{:ok, issues}` for anything else.
  """
  @spec verify(String.t()) :: {:ok, term()} | {:error, term()}
  def verify(path) when is_binary(path) do
    with {:ok, db} <- Exqlite.Sqlite3.open(path, mode: :readonly),
         {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, "PRAGMA integrity_check"),
         :ok <- :ok,
         rows <- drain(db, stmt, []),
         _ <- Exqlite.Sqlite3.release(db, stmt),
         :ok <- Exqlite.Sqlite3.close(db) do
      case rows do
        [["ok"]] -> {:ok, :ok}
        issues -> {:ok, issues}
      end
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp ensure_parent_dir(parent) do
    if File.dir?(parent), do: :ok, else: {:error, {:parent_missing, parent}}
  end

  defp ensure_no_existing(path) do
    if File.exists?(path), do: {:error, {:target_exists, path}}, else: :ok
  end

  defp ensure_source_exists(path) do
    if File.exists?(path), do: :ok, else: {:error, {:source_missing, path}}
  end

  # Use an independent SQLite connection so a large online backup does not
  # monopolize the Store GenServer and stall every request behind it.
  defp vacuum_into(target_path) do
    source_path = Application.fetch_env!(:optimal_engine, :db_path)

    with {:ok, db} <- Exqlite.Sqlite3.open(source_path) do
      try do
        with {:ok, statement} <- Exqlite.Sqlite3.prepare(db, "VACUUM INTO ?1"),
             :ok <- Exqlite.Sqlite3.bind(statement, [target_path]) do
          result =
            case Exqlite.Sqlite3.step(db, statement) do
              :done -> :ok
              {:error, reason} -> {:error, reason}
              other -> {:error, {:unexpected_vacuum_result, other}}
            end

          Exqlite.Sqlite3.release(db, statement)
          result
        end
      after
        Exqlite.Sqlite3.close(db)
      end
    end
  end

  defp catalog_backup(id, path, size, checksum, integrity_status) do
    sql = """
    INSERT INTO backup_records (
      id, tenant_id, storage_provider, storage_uri, status, encrypted, offsite,
      size_bytes, checksum_sha256, integrity_status, verified_at
    ) VALUES (?1, 'default', 'local', ?2, 'completed', 0, 0, ?3, ?4, ?5, datetime('now'))
    """

    case Store.raw_query(sql, [id, Path.expand(path), size, checksum, integrity_status]) do
      {:ok, _} -> :ok
      {:error, reason} -> {:error, {:catalog_failed, reason}}
    end
  end

  defp backup_id do
    "backup_" <> (:crypto.strong_rand_bytes(16) |> Base.encode16(case: :lower))
  end

  defp checksum_sha256(path) do
    digest =
      path
      |> File.stream!([], 1_048_576)
      |> Enum.reduce(:crypto.hash_init(:sha256), &:crypto.hash_update(&2, &1))
      |> :crypto.hash_final()

    Base.encode16(digest, case: :lower)
  end

  defp refuse_if_supervisor_up do
    case Process.whereis(OptimalEngine.Supervisor) do
      nil -> :ok
      _ -> {:error, :supervisor_still_running}
    end
  end

  defp archive_current(db_path) do
    if File.exists?(db_path) do
      epoch = System.os_time(:second)
      dest = "#{db_path}.pre-restore-#{epoch}"
      File.cp!(db_path, dest)
      {:ok, dest}
    else
      {:ok, nil}
    end
  end

  # Sum of `COUNT(*)` across the primary engine tables — gives a
  # rough "how much data is in this backup" number without opening
  # every table.
  defp count_backup_rows(path) do
    try do
      {:ok, db} = Exqlite.Sqlite3.open(path, mode: :readonly)

      total =
        ~w(contexts signals wiki_pages connectors chunks citations)
        |> Enum.reduce(0, fn table, acc ->
          case exec(db, "SELECT COUNT(*) FROM #{table}") do
            [[n]] when is_integer(n) -> acc + n
            _ -> acc
          end
        end)

      Exqlite.Sqlite3.close(db)
      total
    rescue
      _ -> 0
    end
  end

  defp exec(db, sql) do
    with {:ok, stmt} <- Exqlite.Sqlite3.prepare(db, sql),
         rows <- drain(db, stmt, []) do
      Exqlite.Sqlite3.release(db, stmt)
      rows
    else
      _ -> []
    end
  end

  defp drain(db, stmt, acc) do
    case Exqlite.Sqlite3.step(db, stmt) do
      {:row, row} -> drain(db, stmt, [row | acc])
      :done -> Enum.reverse(acc)
      _ -> Enum.reverse(acc)
    end
  end
end
