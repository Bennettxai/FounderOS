defmodule OptimalEngine.StorageAudit do
  @moduledoc "Deep, read-only health audit for every engine storage layer."

  alias OptimalEngine.{StorageCatalog, Store}
  alias OptimalEngine.Pipeline.Decomposer.RLM

  @scoped_tables ~w(contexts chunks chunk_embeddings memories claims facts memory_objects source_packages)

  @spec run() :: map()
  def run do
    checks = [
      check("sqlite_integrity", &sqlite_integrity/0),
      check("foreign_keys", &foreign_keys/0),
      check("migrations", &migrations/0),
      check("logical_stores", &logical_stores/0),
      check("fts_parity", &fts_parity/0),
      check("workspace_scope", &workspace_scope/0),
      check("vector_integrity", &vector_integrity/0),
      check("active_test_fixtures", &active_test_fixtures/0),
      check("asset_files", &asset_files/0),
      check("verified_backup", &verified_backup/0),
      check("rlm_runtime", &rlm_runtime/0),
      check("cache", &cache/0)
    ]

    failures = Enum.reject(checks, & &1.ok)
    %{ok: failures == [], checks: checks, failures: length(failures)}
  end

  defp check(name, fun) do
    case fun.() do
      {:ok, detail} -> %{name: name, ok: true, detail: detail}
      {:error, detail} -> %{name: name, ok: false, detail: inspect(detail)}
    end
  rescue
    error -> %{name: name, ok: false, detail: Exception.message(error)}
  catch
    :exit, reason -> %{name: name, ok: false, detail: inspect(reason)}
  end

  defp sqlite_integrity do
    case Store.raw_query("PRAGMA integrity_check", []) do
      {:ok, [["ok"]]} -> {:ok, "ok"}
      other -> {:error, other}
    end
  end

  defp foreign_keys do
    case Store.raw_query("PRAGMA foreign_key_check", []) do
      {:ok, []} -> {:ok, "no violations"}
      {:ok, rows} -> {:error, %{violations: length(rows)}}
      other -> {:error, other}
    end
  end

  defp migrations do
    with {:ok, [[applied]]} <- Store.raw_query("SELECT MAX(version) FROM schema_migrations", []),
         expected <- OptimalEngine.Store.Migrations.all() |> List.last() |> elem(0),
         true <- applied == expected do
      {:ok, %{applied: applied, expected: expected}}
    else
      false -> {:error, :migration_drift}
      other -> {:error, other}
    end
  end

  defp logical_stores do
    stores = StorageCatalog.list()
    degraded = Enum.filter(stores, &(&1.status != "available"))
    if degraded == [], do: {:ok, %{available: length(stores)}}, else: {:error, degraded}
  end

  defp fts_parity do
    sql = """
    SELECT
      (SELECT COUNT(*) FROM contexts),
      (SELECT COUNT(*) FROM contexts_fts),
      (SELECT COUNT(DISTINCT id) FROM contexts_fts)
    """

    case Store.raw_query(sql, []) do
      {:ok, [[contexts, fts, distinct]]} when contexts == fts and fts == distinct ->
        {:ok, %{contexts: contexts, indexed: fts}}

      {:ok, [[contexts, fts, distinct]]} ->
        {:error, %{contexts: contexts, indexed: fts, distinct_ids: distinct}}

      other ->
        {:error, other}
    end
  end

  defp workspace_scope do
    parts =
      Enum.map_join(
        @scoped_tables,
        " + ",
        &"(SELECT COUNT(*) FROM #{&1} WHERE workspace_id = 'default')"
      )

    case scalar("SELECT #{parts}") do
      {:ok, 0} -> {:ok, "no legacy-default rows"}
      {:ok, count} -> {:error, %{legacy_default_rows: count}}
      other -> other
    end
  end

  defp vector_integrity do
    sql = """
    SELECT
      (SELECT COUNT(*) FROM vectors WHERE dimensions <= 0 OR length(embedding) != dimensions * 4),
      (SELECT COUNT(*) FROM chunk_embeddings WHERE dim <= 0 OR length(vector) != dim * 4),
      (SELECT COUNT(*) FROM vectors v LEFT JOIN contexts c ON c.id = v.context_id WHERE c.id IS NULL),
      (SELECT COUNT(*) FROM chunk_embeddings e LEFT JOIN chunks c ON c.id = e.chunk_id WHERE c.id IS NULL)
    """

    case Store.raw_query(sql, []) do
      {:ok, [[0, 0, 0, 0]]} -> {:ok, "dimensions and references valid"}
      {:ok, [counts]} -> {:error, %{bad_counts: counts}}
      other -> {:error, other}
    end
  end

  defp active_test_fixtures do
    case scalar(
           "SELECT COUNT(*) FROM workspaces WHERE status = 'active' AND id GLOB 'exampleorg-*:secrets'"
         ) do
      {:ok, 0} -> {:ok, "none active"}
      {:ok, count} -> {:error, %{active_fixture_workspaces: count}}
      other -> other
    end
  end

  defp asset_files do
    case Store.raw_query("SELECT storage_path FROM assets", []) do
      {:ok, rows} ->
        missing = rows |> Enum.map(&hd/1) |> Enum.reject(&File.regular?/1)

        if missing == [],
          do: {:ok, %{cataloged: length(rows)}},
          else: {:error, %{missing: length(missing)}}

      other ->
        {:error, other}
    end
  end

  defp verified_backup do
    sql = """
    SELECT storage_uri, checksum_sha256
    FROM backup_records
    WHERE status = 'completed' AND integrity_status = 'verified'
    ORDER BY verified_at DESC LIMIT 1
    """

    case Store.raw_query(sql, []) do
      {:ok, [[path, checksum]]} when is_binary(checksum) ->
        if File.regular?(path) and String.length(checksum) == 64,
          do: {:ok, %{path: path, checksum: checksum}},
          else: {:error, :verified_backup_missing}

      {:ok, []} ->
        {:error, :no_verified_backup}

      other ->
        {:error, other}
    end
  end

  defp rlm_runtime do
    case RLM.health() do
      {:ok, status} ->
        {:ok, status}

      {:error, reason} ->
        if System.get_env("OPTIMAL_RLM_REQUIRED") == "true",
          do: {:error, reason},
          else: {:ok, %{available: false, optional: true, reason: inspect(reason)}}
    end
  end

  defp cache do
    case :ets.info(:optimal_engine_store_cache, :size) do
      :undefined -> {:error, :cache_unavailable}
      size -> {:ok, %{entries: size}}
    end
  end

  defp scalar(sql) do
    case Store.raw_query(sql, []) do
      {:ok, [[value]]} -> {:ok, value}
      other -> {:error, other}
    end
  end
end
