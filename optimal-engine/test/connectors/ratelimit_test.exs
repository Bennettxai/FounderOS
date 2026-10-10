defmodule OptimalEngine.Connectors.RateLimitTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Connectors.RateLimit

  setup do
    RateLimit.reset()
    :ok
  end

  test "check/4 grants the first `burst` tokens immediately" do
    key = {:test, :burst}
    for _ <- 1..5, do: assert(:ok = RateLimit.check(key, 10, 5))
  end

  test "check/4 rejects with retry_after when the bucket is empty" do
    key = {:test, :exhaust}
    for _ <- 1..3, do: :ok = RateLimit.check(key, 1, 3)
    assert {:error, :rate_limited, ms} = RateLimit.check(key, 1, 3)
    assert is_integer(ms)
    assert ms > 0
  end

  test "wait/4 sleeps and returns :ok when the bucket refills in time" do
    key = {:test, :wait_ok}
    # Burn the burst
    for _ <- 1..2, do: :ok = RateLimit.check(key, 100, 2)
    # Next call should refill within ~10ms at rate=100/sec
    assert :ok = RateLimit.wait(key, 100, 2, timeout: 500)
  end

  test "wait/4 times out when the bucket can't refill in time" do
    key = {:test, :wait_to}
    # Burn the burst
    for _ <- 1..2, do: :ok = RateLimit.check(key, 1, 2)
    # Rate=1/sec, need 1 token → timeout=50ms → we'll give up
    assert {:error, :timeout} = RateLimit.wait(key, 1, 2, timeout: 50)
  end
end
