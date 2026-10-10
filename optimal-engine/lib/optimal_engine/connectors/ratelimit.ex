defmodule OptimalEngine.Connectors.RateLimit do
  @moduledoc """
  Token-bucket rate limiter for outbound connector requests.

  Each `(bucket_key, rate, burst)` triple holds its own bucket state
  in a single ETS table, so multiple connectors (or the same connector
  across tenants) can share a limiter without per-process GenServers.

  Two public calls — `check/4` and `wait/4` — cover the usual patterns:

      # Non-blocking: drop / retry elsewhere if we're over quota
      case RateLimit.check({:slack, tenant_id}, 50, 10) do
        :ok -> do_call()
        {:error, :rate_limited, retry_after_ms} -> queue_for_later()
      end

      # Blocking: sleep until a token frees up (bounded)
      :ok = RateLimit.wait({:gmail, tenant_id}, 20, 5, timeout: 5_000)

  `rate` is tokens-per-second, `burst` is the bucket's max capacity.
  """

  @table :optimal_connector_ratelimit

  @doc false
  # Called by the supervision tree. Idempotent — safe to invoke at
  # boot and in tests that start a fresh session.
  @spec ensure_table() :: :ok
  def ensure_table do
    case :ets.whereis(@table) do
      :undefined ->
        :ets.new(@table, [
          :named_table,
          :public,
          :set,
          read_concurrency: true,
          write_concurrency: true
        ])

        :ok

      _ ->
        :ok
    end
  end

  @doc """
  Non-blocking check. Returns `:ok` if a token was available,
  `{:error, :rate_limited, retry_after_ms}` otherwise.
  """
  @spec check(term(), pos_integer(), pos_integer()) ::
          :ok | {:error, :rate_limited, non_neg_integer()}
  def check(bucket_key, rate, burst) when is_integer(rate) and is_integer(burst) do
    ensure_table()
    now = System.monotonic_time(:millisecond)
    refill_rate = rate / 1000.0

    case :ets.lookup(@table, bucket_key) do
      [] ->
        :ets.insert(@table, {bucket_key, burst - 1, now})
        :ok

      [{^bucket_key, tokens, last_ts}] ->
        refilled = min(burst * 1.0, tokens + (now - last_ts) * refill_rate)

        if refilled >= 1.0 do
          :ets.insert(@table, {bucket_key, refilled - 1, now})
          :ok
        else
          :ets.insert(@table, {bucket_key, refilled, now})
          needed = 1.0 - refilled
          retry_after = ceil(needed / refill_rate)
          {:error, :rate_limited, retry_after}
        end
    end
  end

  @doc """
  Blocking variant: waits until a token frees up. Bails with
  `{:error, :timeout}` after `:timeout` ms (default 10_000).
  """
  @spec wait(term(), pos_integer(), pos_integer(), keyword()) :: :ok | {:error, :timeout}
  def wait(bucket_key, rate, burst, opts \\ []) do
    deadline = System.monotonic_time(:millisecond) + Keyword.get(opts, :timeout, 10_000)
    do_wait(bucket_key, rate, burst, deadline)
  end

  @doc "Clear the entire ratelimit table. Test-only."
  @spec reset() :: :ok
  def reset do
    ensure_table()
    :ets.delete_all_objects(@table)
    :ok
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp do_wait(bucket_key, rate, burst, deadline) do
    case check(bucket_key, rate, burst) do
      :ok ->
        :ok

      {:error, :rate_limited, sleep_ms} ->
        now = System.monotonic_time(:millisecond)

        if now + sleep_ms > deadline do
          {:error, :timeout}
        else
          Process.sleep(sleep_ms)
          do_wait(bucket_key, rate, burst, deadline)
        end
    end
  end
end
