defmodule OptimalEngine.Insight.Health do
  @moduledoc """
  Stateless health diagnostics for the OptimalOS knowledge base.

  Runs 10 checks against the SQLite store via `Store.raw_query/2` and returns
  a structured list of diagnostic results, each with a severity level.

  This module has no process state — no GenServer, no supervision tree entry.
  All failures are caught; every check always returns a result map.

  All checks are workspace-scoped (`:workspace_id` option, default
  `"default"`): details may include concrete context ids and titles, so a
  check must never aggregate across workspaces. Entity and vector membership
  is resolved through the contexts table — the source of truth for
  workspace_id — rather than the (legacy-backfilled) columns on those tables.

  Severities:
  - `:ok`       — check passed, no action needed
  - `:warning`  — issue found, should be addressed
  - `:critical` — serious integrity problem requiring immediate attention
  """

  alias OptimalEngine.Store
  require Logger

  @stale_days 30
  @low_sn_threshold 0.4
  @low_quality_ratio 0.20
  @node_imbalance_multiplier 3.0

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Runs all 10 diagnostic checks and returns their results.

  Options:
  - `:workspace_id` — workspace to diagnose (default: "default")

  Always returns `{:ok, [map()]}`. Individual check failures are caught and
  reported as `:warning` severity rather than propagated.
  """
  @spec run(keyword()) :: {:ok, [map()]}
  def run(opts \\ []) do
    ws = Keyword.get(opts, :workspace_id, "default")

    checks = [
      check_orphaned_contexts(ws),
      check_stale_signals(ws),
      check_missing_cross_refs(ws),
      check_fts_drift(ws),
      check_entity_merge_candidates(ws),
      check_node_imbalance(ws),
      check_duplicate_detection(ws),
      check_broken_references(ws),
      check_embedding_coverage(ws),
      check_quality_distribution(ws)
    ]

    {:ok, checks}
  end

  @doc """
  Summarises a list of diagnostic results by severity count.

  Returns a map with keys `:ok`, `:warning`, `:critical`, and `:total`.
  """
  @spec summary([map()]) :: map()
  def summary(diagnostics) when is_list(diagnostics) do
    counts = Enum.frequencies_by(diagnostics, & &1.severity)

    %{
      ok: Map.get(counts, :ok, 0),
      warning: Map.get(counts, :warning, 0),
      critical: Map.get(counts, :critical, 0),
      total: length(diagnostics)
    }
  end

  # ---------------------------------------------------------------------------
  # Check 1 — Orphaned contexts (no edges)
  # ---------------------------------------------------------------------------

  @spec check_orphaned_contexts(String.t()) :: map()
  def check_orphaned_contexts(ws \\ "default") do
    sql = """
    SELECT c.id, c.title FROM contexts c
    LEFT JOIN edges e ON e.source_id = c.id OR e.target_id = c.id
    WHERE e.id IS NULL AND c.workspace_id = ?1
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        items = Enum.map(rows, fn [id, title] -> %{id: id, title: title} end)
        count = length(items)

        severity = if count == 0, do: :ok, else: :warning

        %{
          name: :orphaned_contexts,
          severity: severity,
          message: "Found #{count} orphaned context(s) with no edges",
          details: items,
          fix: "Run `mix optimal.index` to rebuild edges from context content"
        }

      {:error, reason} ->
        error_result(:orphaned_contexts, reason)
    end
  rescue
    err -> rescue_result(:orphaned_contexts, err)
  end

  # ---------------------------------------------------------------------------
  # Check 2 — Stale signals (not modified in >30 days)
  # ---------------------------------------------------------------------------

  @spec check_stale_signals(String.t()) :: map()
  def check_stale_signals(ws \\ "default") do
    sql = """
    SELECT id, title, modified_at FROM contexts
    WHERE modified_at < datetime('now', '-#{@stale_days} days')
    AND modified_at IS NOT NULL
    AND workspace_id = ?1
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        items =
          Enum.map(rows, fn [id, title, modified_at] ->
            %{id: id, title: title, modified_at: modified_at}
          end)

        count = length(items)

        severity =
          cond do
            count == 0 -> :ok
            count < 10 -> :warning
            true -> :warning
          end

        %{
          name: :stale_signals,
          severity: severity,
          message: "Found #{count} context(s) not modified in #{@stale_days}+ days",
          details: items,
          fix: "Review stale contexts — archive, update, or mark as permanent reference"
        }

      {:error, reason} ->
        error_result(:stale_signals, reason)
    end
  rescue
    err -> rescue_result(:stale_signals, err)
  end

  # ---------------------------------------------------------------------------
  # Check 3 — Missing cross-references
  # ---------------------------------------------------------------------------

  @spec check_missing_cross_refs(String.t()) :: map()
  def check_missing_cross_refs(ws \\ "default") do
    ctx_sql = """
    SELECT c.id, c.title, c.routed_to, c.node FROM contexts c
    WHERE c.routed_to != '[]' AND c.routed_to != '' AND c.routed_to IS NOT NULL
    AND c.workspace_id = ?1
    """

    edge_sql = """
    SELECT source_id, target_id FROM edges
    WHERE relation = 'cross_ref' AND workspace_id = ?1
    """

    with {:ok, ctx_rows} <- Store.raw_query(ctx_sql, [ws]),
         {:ok, edge_rows} <- Store.raw_query(edge_sql, [ws]) do
      existing_pairs =
        MapSet.new(edge_rows, fn [src, tgt] -> {src, tgt} end)

      missing =
        Enum.flat_map(ctx_rows, fn [id, title, routed_to_raw, node] ->
          destinations = parse_json_list(routed_to_raw)

          missing_dests =
            Enum.reject(destinations, fn dest ->
              dest == node or MapSet.member?(existing_pairs, {id, dest})
            end)

          if missing_dests == [] do
            []
          else
            [%{id: id, title: title, missing_cross_refs: missing_dests}]
          end
        end)

      count = length(missing)
      severity = if count == 0, do: :ok, else: :warning

      %{
        name: :missing_cross_refs,
        severity: severity,
        message: "Found #{count} context(s) with missing cross_ref edges",
        details: missing,
        fix: "Run `mix optimal.index` to regenerate cross-reference edges"
      }
    else
      {:error, reason} -> error_result(:missing_cross_refs, reason)
    end
  rescue
    err -> rescue_result(:missing_cross_refs, err)
  end

  # ---------------------------------------------------------------------------
  # Check 4 — FTS/index drift (count mismatch)
  # ---------------------------------------------------------------------------

  @spec check_fts_drift(String.t()) :: map()
  def check_fts_drift(ws \\ "default") do
    sql = """
    SELECT (SELECT COUNT(*) FROM contexts WHERE workspace_id = ?1) as ctx_count,
           (SELECT COUNT(*) FROM contexts_fts f
              JOIN contexts c ON c.id = f.id
             WHERE c.workspace_id = ?1) as fts_count
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, [[ctx_count, fts_count]]} ->
        drift = abs(ctx_count - fts_count)

        severity =
          cond do
            drift == 0 -> :ok
            drift < 5 -> :warning
            true -> :critical
          end

        %{
          name: :fts_drift,
          severity: severity,
          message: "contexts=#{ctx_count}, contexts_fts=#{fts_count} (drift=#{drift})",
          details: [%{contexts: ctx_count, fts: fts_count, drift: drift}],
          fix: "Run `mix optimal.index` to rebuild the FTS index"
        }

      {:ok, _unexpected} ->
        error_result(:fts_drift, :unexpected_row_shape)

      {:error, reason} ->
        error_result(:fts_drift, reason)
    end
  rescue
    err -> rescue_result(:fts_drift, err)
  end

  # ---------------------------------------------------------------------------
  # Check 5 — Entity merge candidates (duplicate names, case-insensitive)
  # ---------------------------------------------------------------------------

  @spec check_entity_merge_candidates(String.t()) :: map()
  def check_entity_merge_candidates(ws \\ "default") do
    sql = """
    SELECT LOWER(e.name) as lname, COUNT(*) as cnt, GROUP_CONCAT(DISTINCT e.name) as variants
    FROM entities e
    JOIN contexts c ON c.id = e.context_id AND c.workspace_id = ?1
    GROUP BY LOWER(e.name)
    HAVING COUNT(*) > 1
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        items =
          Enum.map(rows, fn [lname, cnt, variants] ->
            %{
              canonical: lname,
              count: cnt,
              variants: String.split(variants, ",")
            }
          end)

        count = length(items)
        severity = if count == 0, do: :ok, else: :warning

        %{
          name: :entity_merge_candidates,
          severity: severity,
          message: "Found #{count} entity name collision(s) across case variants",
          details: items,
          fix: "Manually normalise entity names or run entity deduplication"
        }

      {:error, reason} ->
        error_result(:entity_merge_candidates, reason)
    end
  rescue
    err -> rescue_result(:entity_merge_candidates, err)
  end

  # ---------------------------------------------------------------------------
  # Check 6 — Node imbalance (any node > 3x mean context count)
  # ---------------------------------------------------------------------------

  @spec check_node_imbalance(String.t()) :: map()
  def check_node_imbalance(ws \\ "default") do
    sql = """
    SELECT node, COUNT(*) as cnt FROM contexts WHERE workspace_id = ?1 GROUP BY node
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, []} ->
        ok_result(:node_imbalance, "No contexts in store — nothing to compare")

      {:ok, rows} ->
        node_counts = Enum.map(rows, fn [node, cnt] -> {node, cnt} end)
        total = Enum.sum(Enum.map(node_counts, &elem(&1, 1)))
        mean = total / length(node_counts)
        threshold = mean * @node_imbalance_multiplier

        overloaded =
          Enum.filter(node_counts, fn {_node, cnt} -> cnt > threshold end)
          |> Enum.map(fn {node, cnt} ->
            %{
              node: node,
              count: cnt,
              mean: Float.round(mean, 1),
              threshold: Float.round(threshold, 1)
            }
          end)

        count = length(overloaded)
        severity = if count == 0, do: :ok, else: :warning

        %{
          name: :node_imbalance,
          severity: severity,
          message:
            "Found #{count} node(s) with >#{@node_imbalance_multiplier}x mean context count (mean=#{Float.round(mean, 1)})",
          details: overloaded,
          fix: "Review overloaded nodes — consider splitting or reclassifying contexts"
        }

      {:error, reason} ->
        error_result(:node_imbalance, reason)
    end
  rescue
    err -> rescue_result(:node_imbalance, err)
  end

  # ---------------------------------------------------------------------------
  # Check 7 — Duplicate detection (identical titles within same node)
  # ---------------------------------------------------------------------------

  @spec check_duplicate_detection(String.t()) :: map()
  def check_duplicate_detection(ws \\ "default") do
    sql = """
    SELECT node, title, COUNT(*) as cnt FROM contexts
    WHERE workspace_id = ?1
    GROUP BY node, title
    HAVING COUNT(*) > 1
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        items =
          Enum.map(rows, fn [node, title, cnt] ->
            %{node: node, title: title, count: cnt}
          end)

        count = length(items)
        severity = if count == 0, do: :ok, else: :warning

        %{
          name: :duplicate_detection,
          severity: severity,
          message: "Found #{count} duplicate title(s) within the same node",
          details: items,
          fix: "Review duplicates and merge or disambiguate context titles"
        }

      {:error, reason} ->
        error_result(:duplicate_detection, reason)
    end
  rescue
    err -> rescue_result(:duplicate_detection, err)
  end

  # ---------------------------------------------------------------------------
  # Check 8 — Broken references (supersedes pointing to nonexistent IDs)
  # ---------------------------------------------------------------------------

  @spec check_broken_references(String.t()) :: map()
  def check_broken_references(ws \\ "default") do
    sql = """
    SELECT c.id, c.title, c.supersedes FROM contexts c
    WHERE c.supersedes IS NOT NULL AND c.supersedes != ''
    AND c.workspace_id = ?1
    AND c.supersedes NOT IN (SELECT id FROM contexts WHERE workspace_id = ?1)
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        items =
          Enum.map(rows, fn [id, title, supersedes] ->
            %{id: id, title: title, missing_ref: supersedes}
          end)

        count = length(items)
        severity = if count == 0, do: :ok, else: :critical

        %{
          name: :broken_references,
          severity: severity,
          message: "Found #{count} broken `supersedes` reference(s) to nonexistent context IDs",
          details: items,
          fix: "Clear or correct the `supersedes` field for the listed contexts"
        }

      {:error, reason} ->
        error_result(:broken_references, reason)
    end
  rescue
    err -> rescue_result(:broken_references, err)
  end

  # ---------------------------------------------------------------------------
  # Check 9 — Embedding coverage (ratio of vectors to total contexts)
  # ---------------------------------------------------------------------------

  @spec check_embedding_coverage(String.t()) :: map()
  def check_embedding_coverage(ws \\ "default") do
    sql = """
    SELECT (SELECT COUNT(*) FROM vectors v
              JOIN contexts c ON c.id = v.context_id
             WHERE c.workspace_id = ?1) as vec_count,
           (SELECT COUNT(*) FROM contexts WHERE workspace_id = ?1) as ctx_count
    """

    case Store.raw_query(sql, [ws]) do
      {:ok, [[vec_count, ctx_count]]} when ctx_count > 0 ->
        coverage = Float.round(vec_count / ctx_count * 100, 1)

        severity =
          cond do
            coverage >= 80.0 -> :ok
            coverage >= 50.0 -> :warning
            true -> :critical
          end

        %{
          name: :embedding_coverage,
          severity: severity,
          message: "#{coverage}% of contexts have embeddings (#{vec_count}/#{ctx_count})",
          details: [%{vectors: vec_count, contexts: ctx_count, coverage_pct: coverage}],
          fix:
            "Embeddings require Ollama with `nomic-embed-text`. Run `mix optimal.index` to generate."
        }

      {:ok, [[_vec_count, 0]]} ->
        ok_result(:embedding_coverage, "No contexts in store — coverage check skipped")

      {:ok, _unexpected} ->
        error_result(:embedding_coverage, :unexpected_row_shape)

      {:error, reason} ->
        error_result(:embedding_coverage, reason)
    end
  rescue
    err -> rescue_result(:embedding_coverage, err)
  end

  # ---------------------------------------------------------------------------
  # Check 10 — Quality distribution (flag if >20% have sn_ratio < 0.4)
  # ---------------------------------------------------------------------------

  @spec check_quality_distribution(String.t()) :: map()
  def check_quality_distribution(ws \\ "default") do
    low_sql =
      "SELECT COUNT(*) as low_quality FROM contexts WHERE sn_ratio < #{@low_sn_threshold} AND workspace_id = ?1"

    total_sql = "SELECT COUNT(*) FROM contexts WHERE workspace_id = ?1"

    with {:ok, [[low_count]]} <- Store.raw_query(low_sql, [ws]),
         {:ok, [[total_count]]} <- Store.raw_query(total_sql, [ws]) do
      if total_count == 0 do
        ok_result(:quality_distribution, "No contexts in store — quality check skipped")
      else
        ratio = low_count / total_count
        pct = Float.round(ratio * 100, 1)

        severity =
          cond do
            ratio <= @low_quality_ratio -> :ok
            ratio <= 0.40 -> :warning
            true -> :critical
          end

        threshold_pct = round(@low_quality_ratio * 100)

        %{
          name: :quality_distribution,
          severity: severity,
          message:
            "#{pct}% of contexts have S/N < #{@low_sn_threshold} (#{low_count}/#{total_count}, threshold #{threshold_pct}%)",
          details: [
            %{
              low_quality_count: low_count,
              total: total_count,
              ratio_pct: pct,
              threshold_pct: threshold_pct
            }
          ],
          fix: "Review low S/N contexts — improve content or remove noise"
        }
      end
    else
      {:error, reason} -> error_result(:quality_distribution, reason)
    end
  rescue
    err -> rescue_result(:quality_distribution, err)
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp ok_result(name, message) do
    %{
      name: name,
      severity: :ok,
      message: message,
      details: [],
      fix: nil
    }
  end

  defp error_result(name, reason) do
    Logger.warning("[HealthDiagnostics] Check #{name} failed: #{inspect(reason)}")

    %{
      name: name,
      severity: :warning,
      message: "Check could not run: #{inspect(reason)}",
      details: [],
      fix: "Ensure the Store is running and the database is accessible"
    }
  end

  defp rescue_result(name, err) do
    Logger.warning("[HealthDiagnostics] Check #{name} raised: #{inspect(err)}")

    %{
      name: name,
      severity: :warning,
      message: "Check raised an exception: #{inspect(err)}",
      details: [],
      fix: "Check engine logs for details"
    }
  end

  defp parse_json_list(nil), do: []
  defp parse_json_list(""), do: []
  defp parse_json_list("[]"), do: []

  defp parse_json_list(raw) when is_binary(raw) do
    case Jason.decode(raw) do
      {:ok, list} when is_list(list) -> Enum.filter(list, &is_binary/1)
      _ -> []
    end
  rescue
    _ -> []
  end
end
