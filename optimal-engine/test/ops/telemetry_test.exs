defmodule OptimalEngine.TelemetryTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Telemetry

  setup do
    Telemetry.reset()
    :ok
  end

  test "snapshot/0 returns zero counters + empty histograms on a fresh reset" do
    snap = Telemetry.snapshot()
    assert snap.uptime_ms >= 0
    assert is_map(snap.counters)
    assert Enum.all?(Map.values(snap.counters), &(&1 == 0))

    assert Enum.all?(snap.histograms, fn {_k, v} ->
             v.p50 == 0 and v.count == 0
           end)
  end

  test "incr/2 increments a counter" do
    Telemetry.incr(:"optimal_engine.intake.ingested", 3)
    # Cast is async — use a sync snapshot call to flush the mailbox.
    snap = Telemetry.snapshot()
    assert snap.counters[:"optimal_engine.intake.ingested"] == 3
  end

  test "observe/2 feeds the histogram buffer" do
    for v <- [10, 20, 30, 40, 50, 60, 70, 80, 90, 100] do
      Telemetry.observe(:"optimal_engine.search.latency_ms", v)
    end

    snap = Telemetry.snapshot()
    h = snap.histograms[:"optimal_engine.search.latency_ms"]
    assert h.count == 10
    assert h.p50 >= 40 and h.p50 <= 60
    assert h.p95 >= 90
  end

  test "counter_names/0 + histogram_names/0 declare the known metrics" do
    assert :"optimal_engine.intake.ingested" in Telemetry.counter_names()
    assert :"optimal_engine.search.latency_ms" in Telemetry.histogram_names()
  end

  test "telemetry.execute events route to the aggregator" do
    :telemetry.execute([:optimal_engine, :search, :query], %{count: 1}, %{})

    # Give the cast a tick to arrive.
    Process.sleep(50)

    snap = Telemetry.snapshot()
    assert snap.counters[:"optimal_engine.search.query"] >= 1
  end
end
