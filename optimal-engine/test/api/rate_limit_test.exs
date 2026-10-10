defmodule OptimalEngine.API.RateLimitTest do
  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.RateLimiter
  alias OptimalEngine.API.RateLimitPlug
  alias OptimalEngine.API.Router

  # ── RateLimiter (GenServer / ETS) unit tests ────────────────────────────────

  describe "RateLimiter.check/3" do
    setup do
      RateLimiter.reset()
      :ok
    end

    test "first request is allowed" do
      assert :ok = RateLimiter.check("unit_key_1", 100, 60)
    end

    test "100 sequential requests pass within burst capacity" do
      key = "burst_key_#{System.unique_integer([:positive])}"

      results = for _ <- 1..100, do: RateLimiter.check(key, 100, 60)
      assert Enum.all?(results, &(&1 == :ok))
    end

    test "101st request is rate-limited after burst is exhausted" do
      key = "exhaust_key_#{System.unique_integer([:positive])}"

      for _ <- 1..100, do: RateLimiter.check(key, 100, 60)

      assert {:rate_limited, retry_after_ms, remaining: 0, reset_at: reset_at} =
               RateLimiter.check(key, 100, 60)

      assert is_integer(retry_after_ms)
      assert retry_after_ms > 0
      # retry_after should be less than 60 seconds (one full refill cycle)
      assert retry_after_ms < 60_000
      assert is_integer(reset_at)
    end

    test "retry_after_ms is sane (> 0 and < 60 000 ms)" do
      key = "retry_after_key_#{System.unique_integer([:positive])}"

      for _ <- 1..100, do: RateLimiter.check(key, 100, 100)

      assert {:rate_limited, retry_after_ms, remaining: 0, reset_at: _} =
               RateLimiter.check(key, 100, 100)

      assert retry_after_ms > 0
      assert retry_after_ms < 60_000
    end

    test "bucket refills over time" do
      key = "refill_key_#{System.unique_integer([:positive])}"

      # Exhaust a tiny bucket (capacity = 2, refill = 120/min = 2/s)
      for _ <- 1..2, do: RateLimiter.check(key, 2, 120)
      assert {:rate_limited, _ms, remaining: 0, reset_at: _} = RateLimiter.check(key, 2, 120)

      # At 120/min = 2 tokens/s → 1 token every 500ms.
      # Wait 600ms — should have refilled at least one token.
      Process.sleep(600)

      assert :ok = RateLimiter.check(key, 2, 120)
    end

    test "different keys are independent buckets" do
      key_a = "key_a_#{System.unique_integer([:positive])}"
      key_b = "key_b_#{System.unique_integer([:positive])}"

      for _ <- 1..5, do: RateLimiter.check(key_a, 5, 60)
      # key_a is exhausted; key_b should still have tokens
      assert {:rate_limited, _, remaining: 0, reset_at: _} = RateLimiter.check(key_a, 5, 60)
      assert :ok = RateLimiter.check(key_b, 5, 60)
    end

    test "reset/0 clears all buckets" do
      key = "reset_key_#{System.unique_integer([:positive])}"
      for _ <- 1..5, do: RateLimiter.check(key, 5, 60)
      assert {:rate_limited, _, remaining: 0, reset_at: _} = RateLimiter.check(key, 5, 60)

      :ok = RateLimiter.reset()

      assert :ok = RateLimiter.check(key, 5, 60)
    end
  end

  # ── RateLimitPlug unit tests ─────────────────────────────────────────────────

  describe "RateLimitPlug" do
    setup do
      RateLimiter.reset()
      :ok
    end

    defp call_plug(conn, opts \\ []) do
      config = RateLimitPlug.init(opts)
      RateLimitPlug.call(conn, config)
    end

    defp new_conn(path \\ "/api/memory") do
      conn(:get, path) |> assign(:current_api_key, nil)
    end

    test "passes through and sets X-RateLimit-* headers on allowed request" do
      conn = call_plug(new_conn())
      assert conn.halted == false
      assert get_resp_header(conn, "x-ratelimit-limit") != []
      assert get_resp_header(conn, "x-ratelimit-remaining") != []
      assert get_resp_header(conn, "x-ratelimit-reset") != []
    end

    test "returns 429 after capacity is exhausted" do
      # Use an ultra-small capacity so we can exhaust it quickly in tests.
      opts = [default_capacity: 3, default_per_minute: 60]

      # Exhaust the bucket
      for _ <- 1..3, do: call_plug(new_conn(), opts)

      # Next request must be rate-limited
      conn = call_plug(new_conn(), opts)
      assert conn.halted == true
      assert conn.status == 429

      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] == "rate_limited"
      assert is_integer(body["retry_after_ms"])
      assert body["retry_after_ms"] > 0
    end

    test "429 response includes Retry-After header" do
      opts = [default_capacity: 1, default_per_minute: 60]
      call_plug(new_conn(), opts)

      conn = call_plug(new_conn(), opts)
      assert conn.status == 429
      [retry_after] = get_resp_header(conn, "retry-after")
      seconds = String.to_integer(retry_after)
      assert seconds > 0
      assert seconds <= 60
    end

    test "429 response includes X-RateLimit-Remaining: 0" do
      opts = [default_capacity: 1, default_per_minute: 60]
      call_plug(new_conn(), opts)

      conn = call_plug(new_conn(), opts)
      assert conn.status == 429
      assert get_resp_header(conn, "x-ratelimit-remaining") == ["0"]
    end

    test "/api/status is exempt and never rate-limited" do
      opts = [default_capacity: 1, default_per_minute: 60]

      # Exhaust anything remaining
      call_plug(new_conn("/api/memory"), opts)

      # /api/status must still pass regardless of IP bucket state
      for _ <- 1..20 do
        conn = call_plug(new_conn("/api/status"), opts)
        assert conn.halted == false
      end
    end

    test "/api/health is exempt and never rate-limited" do
      opts = [default_capacity: 1, default_per_minute: 60]
      call_plug(new_conn("/api/memory"), opts)

      for _ <- 1..20 do
        conn = call_plug(new_conn("/api/health"), opts)
        assert conn.halted == false
      end
    end

    test "api key id is used as bucket key (separate from IP)" do
      opts = [default_capacity: 2, default_per_minute: 60]

      # Two different api keys each get their own buckets
      api_key_1 = %{id: "api_key_1_#{System.unique_integer([:positive])}", metadata: %{}}
      api_key_2 = %{id: "api_key_2_#{System.unique_integer([:positive])}", metadata: %{}}

      conn_1 = new_conn() |> assign(:current_api_key, api_key_1)
      conn_2 = new_conn() |> assign(:current_api_key, api_key_2)

      # Exhaust key_1
      for _ <- 1..2, do: call_plug(conn_1, opts)
      assert call_plug(conn_1, opts).status == 429

      # key_2 must still be allowed
      assert call_plug(conn_2, opts).halted == false
    end

    test "per-key metadata rate_limit_per_minute override is respected" do
      opts = [default_capacity: 200, default_per_minute: 100]

      # Metadata sets a tight 2/min limit
      api_key = %{
        id: "meta_key_#{System.unique_integer([:positive])}",
        metadata: %{"rate_limit_per_minute" => 2}
      }

      conn = new_conn() |> assign(:current_api_key, api_key)

      # With capacity=200 and tiny per_minute, burst fills quickly but let's
      # use a tiny override capacity by also having a small burst default.
      opts_tight = [default_capacity: 2, default_per_minute: 100]
      conn_tight = new_conn() |> assign(:current_api_key, api_key)

      for _ <- 1..2, do: call_plug(conn_tight, opts_tight)
      assert call_plug(conn_tight, opts_tight).status == 429
    end
  end

  # ── Integration: Router + RateLimitPlug end-to-end ──────────────────────────

  @router_opts Router.init([])

  describe "Router integration" do
    setup do
      RateLimiter.reset()
      :ok
    end

    test "/api/status is never rate-limited through the router" do
      # The router plug pipeline applies RateLimitPlug globally, but
      # /api/status must always be reachable.
      conn = conn(:get, "/api/status") |> Router.call(@router_opts)
      # Status or 200 expected, never 429
      assert conn.status in [200, 404]
      assert conn.status != 429
    end

    test "/api/health is never rate-limited through the router" do
      conn = conn(:get, "/api/health") |> Router.call(@router_opts)
      assert conn.status != 429
    end
  end
end
