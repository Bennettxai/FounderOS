defmodule OptimalEngine.API.RateLimitPlug do
  @moduledoc """
  Plug that enforces per-key token-bucket rate limiting on the HTTP API.

  ## Bucket key resolution

  1. `conn.assigns[:current_api_key].id` when an authenticated API key is present.
  2. The request's peer IP address as a string (anonymous fallback).

  ## Rate limit values

  Priority (highest → lowest):

  1. `api_key.metadata["rate_limit_per_minute"]` — per-key override.
  2. `Workspace.Config.get_section(workspace_slug, :rate_limit)` — per-workspace config.
  3. Module default: 100 req/min, 200 burst capacity.

  ## Response headers

  On every allowed request:
    - `X-RateLimit-Limit`     — capacity (burst size)
    - `X-RateLimit-Remaining` — tokens remaining (truncated to integer)
    - `X-RateLimit-Reset`     — Unix timestamp (seconds) when bucket will be full

  On a rate-limited request the plug halts and returns:
    - HTTP 429 with `Content-Type: application/json`
    - Body: `{"error":"rate_limited","retry_after_ms":N}`
    - `Retry-After: <seconds>` header

  ## Exempt paths

  Paths listed in `exempt_paths` (default `["/api/status", "/api/health"]`) are
  passed through without consuming a token.

  ## Usage

      plug OptimalEngine.API.RateLimitPlug,
        default_capacity: 200,
        default_per_minute: 100,
        exempt_paths: ["/api/status", "/api/health"]
  """

  @behaviour Plug

  import Plug.Conn

  alias OptimalEngine.API.RateLimiter
  alias OptimalEngine.Workspace.Config, as: WorkspaceConfig

  @default_per_minute 100
  @default_capacity 200
  @default_exempt ["/api/status", "/api/health"]

  # ── Plug callbacks ───────────────────────────────────────────────────────────

  @impl Plug
  def init(opts) do
    app_defaults = Application.get_env(:optimal_engine, :api_rate_limit, [])

    %{
      default_capacity:
        Keyword.get(
          opts,
          :default_capacity,
          Keyword.get(app_defaults, :default_capacity, @default_capacity)
        ),
      default_per_minute:
        Keyword.get(
          opts,
          :default_per_minute,
          Keyword.get(app_defaults, :default_per_minute, @default_per_minute)
        ),
      exempt_paths: Keyword.get(opts, :exempt_paths, @default_exempt)
    }
  end

  @impl Plug
  def call(conn, config) do
    if exempt?(conn, config.exempt_paths) do
      conn
    else
      enforce(conn, config)
    end
  end

  # ── Private ─────────────────────────────────────────────────────────────────

  defp exempt?(conn, exempt_paths) do
    conn.request_path in exempt_paths
  end

  defp enforce(conn, config) do
    {bucket_key, capacity, per_minute} = resolve_bucket(conn, config)

    case RateLimiter.check(bucket_key, capacity, per_minute) do
      :ok ->
        put_success_headers(conn, capacity, per_minute)

      {:rate_limited, retry_after_ms, remaining: 0, reset_at: reset_at} ->
        conn
        |> put_resp_header("retry-after", to_string(div(retry_after_ms, 1_000) + 1))
        |> put_resp_header("x-ratelimit-limit", to_string(capacity))
        |> put_resp_header("x-ratelimit-remaining", "0")
        |> put_resp_header("x-ratelimit-reset", monotonic_to_unix(reset_at))
        |> send_resp(429, Jason.encode!(%{error: "rate_limited", retry_after_ms: retry_after_ms}))
        |> halt()
    end
  end

  # Returns `{bucket_key, capacity, per_minute}`.
  defp resolve_bucket(conn, config) do
    api_key = conn.assigns[:current_api_key]

    bucket_key =
      if api_key && api_key.id do
        "key:#{api_key.id}"
      else
        "ip:#{peer_ip(conn)}"
      end

    {capacity, per_minute} = resolve_limits(api_key, config)
    {bucket_key, capacity, per_minute}
  end

  defp resolve_limits(api_key, config) do
    # 1. Per-key metadata override.
    key_limit = api_key && get_in(api_key, [Access.key(:metadata, %{}), "rate_limit_per_minute"])

    if is_integer(key_limit) and key_limit > 0 do
      # Per-key override — use it with the default capacity.
      {config.default_capacity, key_limit}
    else
      # 2. Workspace config (best-effort, no workspace slug available at plug
      #    level so we fall through to defaults if it can't be determined).
      workspace_limit = fetch_workspace_limit(api_key)

      case workspace_limit do
        {cap, rpm} when is_integer(cap) and is_integer(rpm) -> {cap, rpm}
        _ -> {config.default_capacity, config.default_per_minute}
      end
    end
  end

  # Attempt to read workspace rate-limit config from the API key's workspace
  # assignment. Returns `{capacity, per_minute}` or `nil`.
  defp fetch_workspace_limit(nil), do: nil

  defp fetch_workspace_limit(api_key) do
    workspace_slug = api_key[:workspace_slug] || api_key[:workspace_id]

    if is_binary(workspace_slug) and workspace_slug != "" do
      case WorkspaceConfig.get_section(workspace_slug, :rate_limit, nil) do
        %{} = rl ->
          capacity = Map.get(rl, :burst_capacity) || Map.get(rl, :requests_per_minute)
          rpm = Map.get(rl, :requests_per_minute)

          if is_integer(capacity) and is_integer(rpm), do: {capacity, rpm}, else: nil

        _ ->
          nil
      end
    else
      nil
    end
  end

  defp peer_ip(conn) do
    case Plug.Conn.get_peer_data(conn) do
      %{address: addr} when is_tuple(addr) ->
        addr |> :inet.ntoa() |> to_string()

      _ ->
        "unknown"
    end
  end

  defp put_success_headers(conn, capacity, per_minute) do
    # Estimate remaining tokens without a second ETS read; the caller already
    # consumed one, so we report capacity - 1 as a conservative bound.
    # Accurate remaining counts would need a second lookup — not worth it.
    remaining = max(0, capacity - 1)

    # Monotonic time when the bucket would be full if empty right now.
    now_ms = System.monotonic_time(:millisecond)
    full_in_ms = trunc(capacity / (per_minute / 60_000.0))
    reset_unix = monotonic_to_unix(now_ms + full_in_ms)

    conn
    |> put_resp_header("x-ratelimit-limit", to_string(capacity))
    |> put_resp_header("x-ratelimit-remaining", to_string(remaining))
    |> put_resp_header("x-ratelimit-reset", reset_unix)
  end

  # Convert a monotonic timestamp (ms) to a Unix epoch second string.
  # Monotonic time has no absolute meaning; we offset it against the
  # current wall/monotonic delta to get a wall-clock value.
  defp monotonic_to_unix(monotonic_ms) do
    wall_now_s = System.system_time(:second)
    mono_now_ms = System.monotonic_time(:millisecond)
    delta_s = div(monotonic_ms - mono_now_ms, 1_000)
    to_string(wall_now_s + delta_s)
  end
end
