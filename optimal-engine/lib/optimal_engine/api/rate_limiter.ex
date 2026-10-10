defmodule OptimalEngine.API.RateLimiter do
  @moduledoc """
  Token-bucket rate limiter for the HTTP API layer.

  A single GenServer owns the ETS table and runs a periodic GC pass to evict
  buckets that have been idle for more than one hour.  All hot-path bucket
  checks go directly to ETS — no GenServer round-trip required in normal
  operation.

  ## Bucket record layout

      {key, tokens :: float(), last_refill_at :: integer()}

  `last_refill_at` is `System.monotonic_time(:millisecond)`.

  ## Public API

      # Succeeds if a token was available; rate-limited otherwise.
      :ok = RateLimiter.check("api_key_abc", 200, 100)

      {:rate_limited, retry_after_ms, remaining: 0, reset_at: ts} =
        RateLimiter.check("anon_1.2.3.4", 200, 100)

  `capacity` is the burst capacity (max tokens in the bucket).
  `refill_per_minute` is the steady-state refill rate.
  """

  use GenServer
  require Logger

  @table :optimal_api_rate_limit_buckets

  # Buckets idle for longer than this are collected.
  @gc_idle_threshold_ms 60 * 60 * 1_000

  # GC runs every 5 minutes.
  @gc_interval_ms 5 * 60 * 1_000

  # ── Child spec ──────────────────────────────────────────────────────────────

  @doc false
  def child_spec(opts) do
    %{
      id: __MODULE__,
      start: {__MODULE__, :start_link, [opts]},
      type: :worker,
      restart: :permanent
    }
  end

  # ── Client API ──────────────────────────────────────────────────────────────

  @doc """
  Starts the rate limiter and creates its ETS table.
  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Check (and consume) one token from the bucket identified by `key`.

  - `capacity` — burst capacity; the maximum number of tokens the bucket may
    hold at any point in time.
  - `refill_per_minute` — steady-state token replenishment rate.

  Returns `:ok` on success (token consumed) or
  `{:rate_limited, retry_after_ms, remaining: 0, reset_at: monotonic_ms}`
  when the bucket is empty.

  The check is a single `:ets.lookup` + `:ets.insert` — O(1), no GenServer
  message in the success path.
  """
  @spec check(term(), pos_integer(), pos_integer()) ::
          :ok
          | {:rate_limited, non_neg_integer(), [remaining: 0, reset_at: integer()]}
  def check(key, capacity, refill_per_minute)
      when is_integer(capacity) and capacity > 0 and
             is_integer(refill_per_minute) and refill_per_minute > 0 do
    ensure_table()
    now = System.monotonic_time(:millisecond)

    # Tokens replenished per millisecond.
    refill_rate_per_ms = refill_per_minute / 60_000.0

    case :ets.lookup(@table, key) do
      [] ->
        # First request from this key — open a full bucket and consume one.
        :ets.insert(@table, {key, capacity - 1.0, now})
        :ok

      [{^key, tokens, last_refill_at}] ->
        elapsed_ms = now - last_refill_at
        tokens_now = min(capacity * 1.0, tokens + elapsed_ms * refill_rate_per_ms)

        if tokens_now >= 1.0 do
          :ets.insert(@table, {key, tokens_now - 1.0, now})
          :ok
        else
          # Leave token count unchanged (still accumulating) and report wait.
          :ets.insert(@table, {key, tokens_now, last_refill_at})

          needed = 1.0 - tokens_now
          retry_after_ms = ceil(needed / refill_rate_per_ms)

          # When the bucket will next have ≥1 token (monotonic ms).
          reset_at = last_refill_at + retry_after_ms

          {:rate_limited, retry_after_ms, remaining: 0, reset_at: reset_at}
        end
    end
  end

  @doc "Delete all bucket state. Intended for tests only."
  @spec reset() :: :ok
  def reset do
    ensure_table()
    :ets.delete_all_objects(@table)
    :ok
  end

  # ── GenServer callbacks ──────────────────────────────────────────────────────

  @impl true
  def init(_opts) do
    create_table()
    schedule_gc()
    Logger.debug("[RateLimiter] Started; ETS table #{@table} ready")
    {:ok, %{}}
  end

  @impl true
  def handle_info(:gc, state) do
    gc_stale_buckets()
    schedule_gc()
    {:noreply, state}
  end

  # ── Private ─────────────────────────────────────────────────────────────────

  defp create_table do
    case :ets.whereis(@table) do
      :undefined ->
        :ets.new(@table, [
          :named_table,
          :public,
          :set,
          read_concurrency: true,
          write_concurrency: true
        ])

      _ ->
        :ok
    end
  end

  # Called in the hot path before any operation in case the table was lost
  # (e.g., process crash before supervisor restart).  Idempotent.
  defp ensure_table do
    case :ets.whereis(@table) do
      :undefined -> create_table()
      _ -> :ok
    end
  end

  defp schedule_gc do
    Process.send_after(self(), :gc, @gc_interval_ms)
  end

  defp gc_stale_buckets do
    now = System.monotonic_time(:millisecond)
    cutoff = now - @gc_idle_threshold_ms

    # Select all rows whose last_refill_at is older than the cutoff.
    match_spec = [{{:_, :_, :"$1"}, [{:<, :"$1", cutoff}], [true]}]
    deleted = :ets.select_delete(@table, match_spec)

    if deleted > 0 do
      Logger.debug("[RateLimiter] GC evicted #{deleted} stale buckets")
    end
  end
end
