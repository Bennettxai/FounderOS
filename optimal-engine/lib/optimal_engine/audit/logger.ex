defmodule OptimalEngine.Audit.Logger do
  @moduledoc """
  Write path for audit events.

  Stateless wrapper over `OptimalEngine.Store.raw_query/2`. Phase 10 may
  introduce an async buffer for write batching; Phase 11 adds the SIEM
  exporter behavior.
  """

  alias OptimalEngine.Audit.Event
  alias OptimalEngine.Store

  require Logger

  @doc """
  Writes the event to the `events` table. Returns `:ok` on success, logs and
  returns `:ok` on failure — audit logging is best-effort and must never
  crash the caller.
  """
  @spec log(Event.t()) :: :ok
  def log(%Event{} = event) do
    sql = """
    INSERT INTO events (tenant_id, ts, principal, kind, target_uri, latency_ms, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
    """

    params = [
      event.tenant_id,
      event.ts || DateTime.utc_now() |> DateTime.to_iso8601(),
      event.principal,
      event.kind,
      event.target_uri,
      event.latency_ms,
      Jason.encode!(event.metadata || %{})
    ]

    case Store.raw_query(sql, params) do
      {:ok, _} ->
        :ok

      other ->
        Logger.warning("[Audit] Failed to log event #{event.kind}: #{inspect(other)}")
        :ok
    end
  end

  @doc "Convenience: build + log in one call."
  @spec log(String.t(), keyword()) :: :ok
  def log(kind, opts) when is_binary(kind) and is_list(opts) do
    kind |> Event.new(opts) |> log()
  end

  @doc """
  Returns events matching the filter, tenant-scoped and principal-filtered
  where applicable.

  Options:
    * `:tenant_id` — required for cross-tenant queries (defaults to default tenant)
    * `:principal` — filter by actor
    * `:kind`      — filter by event kind
    * `:since`     — ISO-8601 timestamp; events at or after
    * `:until`     — ISO-8601 timestamp; events before
    * `:limit`     — default 100
  """
  @spec query(keyword()) :: {:ok, [map()]} | {:error, term()}
  def query(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, OptimalEngine.Tenancy.Tenant.default_id())
    limit = Keyword.get(opts, :limit, 100)

    {clauses, params} =
      Enum.reduce(
        [
          {:principal, "principal = ?"},
          {:kind, "kind = ?"},
          {:since, "ts >= ?"},
          {:until, "ts < ?"}
        ],
        {["tenant_id = ?1"], [tenant_id]},
        fn {key, template}, {cs, ps} ->
          case Keyword.get(opts, key) do
            nil ->
              {cs, ps}

            value ->
              idx = length(ps) + 1
              {cs ++ [String.replace(template, "?", "?#{idx}")], ps ++ [value]}
          end
        end
      )

    where = Enum.join(clauses, " AND ")
    limit_idx = length(params) + 1

    sql = """
    SELECT id, tenant_id, ts, principal, kind, target_uri, latency_ms, metadata
    FROM events
    WHERE #{where}
    ORDER BY ts DESC
    LIMIT ?#{limit_idx}
    """

    case Store.raw_query(sql, params ++ [limit]) do
      {:ok, rows} ->
        events =
          Enum.map(rows, fn [id, tid, ts, principal, kind, target_uri, latency, meta_json] ->
            %{
              id: id,
              tenant_id: tid,
              ts: ts,
              principal: principal,
              kind: kind,
              target_uri: target_uri,
              latency_ms: latency,
              metadata: decode_json(meta_json)
            }
          end)

        {:ok, events}

      other ->
        other
    end
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(s) when is_binary(s) do
    case Jason.decode(s) do
      {:ok, m} -> m
      _ -> %{}
    end
  end
end
