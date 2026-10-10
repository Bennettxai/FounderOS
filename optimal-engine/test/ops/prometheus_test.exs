defmodule OptimalEngine.Telemetry.PrometheusTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Telemetry
  alias OptimalEngine.Telemetry.Prometheus

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router

  @opts Router.init([])

  setup do
    Telemetry.reset()
    :ok
  end

  # ---------------------------------------------------------------------------
  # Prometheus.render/0 unit tests
  # ---------------------------------------------------------------------------

  describe "Prometheus.render/0" do
    test "returns a string" do
      assert is_binary(Prometheus.render())
    end

    test "contains # HELP and # TYPE lines" do
      output = Prometheus.render()
      assert output =~ "# HELP"
      assert output =~ "# TYPE"
    end

    test "contains uptime_seconds gauge" do
      output = Prometheus.render()
      assert output =~ "optimal_engine_uptime_seconds"
      assert output =~ "# TYPE optimal_engine_uptime_seconds gauge"
    end

    test "contains BEAM system metrics" do
      output = Prometheus.render()
      assert output =~ "optimal_engine_memory_heap_bytes"
      assert output =~ "optimal_engine_process_count"
      assert output =~ "# TYPE optimal_engine_memory_heap_bytes gauge"
      assert output =~ "# TYPE optimal_engine_process_count gauge"
    end

    test "counter metrics end with _total suffix" do
      output = Prometheus.render()
      assert output =~ "optimal_engine_intake_ingested_total"
      assert output =~ "optimal_engine_retrieval_rag_total"
    end

    test "counter TYPE lines declare counter type" do
      output = Prometheus.render()
      assert output =~ "# TYPE optimal_engine_intake_ingested_total counter"
    end

    test "histogram metrics emit _count and _sum lines" do
      Telemetry.observe(:"optimal_engine.search.latency_ms", 100)
      Telemetry.observe(:"optimal_engine.search.latency_ms", 200)
      # sync via snapshot
      Telemetry.snapshot()

      output = Prometheus.render()
      assert output =~ "optimal_engine_search_latency_ms_count"
      assert output =~ "optimal_engine_search_latency_ms_sum"
    end

    test "histogram TYPE lines declare histogram type" do
      output = Prometheus.render()
      assert output =~ "# TYPE optimal_engine_search_latency_ms histogram"
    end

    test "counter values reflect incr/2 calls" do
      Telemetry.incr(:"optimal_engine.retrieval.rag", 7)
      # sync
      Telemetry.snapshot()

      output = Prometheus.render()

      # The line should show the value 7
      assert output =~ ~r/optimal_engine_retrieval_rag_total\s+7/
    end

    test "uptime_seconds value is a positive number" do
      output = Prometheus.render()

      # Extract the value after the metric name
      case Regex.run(~r/optimal_engine_uptime_seconds\s+([0-9.]+)/, output) do
        [_, value_str] ->
          {value, _} = Float.parse(value_str)
          assert value >= 0

        nil ->
          flunk("uptime_seconds line not found in output:\n#{output}")
      end
    end

    test "DB count metrics are present" do
      output = Prometheus.render()
      # These may be 0 in test but must be present.
      assert output =~ "optimal_engine_contexts_total"
      assert output =~ "optimal_engine_memories_total"
      assert output =~ "optimal_engine_wiki_pages_total"
      assert output =~ "optimal_engine_workspaces_total"
    end

    test "each metric block has exactly one HELP and one TYPE" do
      output = Prometheus.render()
      help_count = output |> String.split("\n") |> Enum.count(&String.starts_with?(&1, "# HELP"))
      type_count = output |> String.split("\n") |> Enum.count(&String.starts_with?(&1, "# TYPE"))
      # They should match (one HELP + one TYPE per metric family).
      assert help_count == type_count
      assert help_count > 0
    end

    test "output ends with a newline" do
      output = Prometheus.render()
      assert String.ends_with?(output, "\n")
    end
  end

  # ---------------------------------------------------------------------------
  # HTTP endpoint tests
  # ---------------------------------------------------------------------------

  describe "GET /api/metrics/prometheus" do
    test "returns 200" do
      conn = conn(:get, "/api/metrics/prometheus") |> Router.call(@opts)
      assert conn.status == 200
    end

    test "sets correct Prometheus content-type" do
      conn = conn(:get, "/api/metrics/prometheus") |> Router.call(@opts)
      content_type = get_resp_header(conn, "content-type") |> List.first()
      assert content_type =~ "text/plain"
      assert content_type =~ "version=0.0.4"
    end

    test "body contains # TYPE lines" do
      conn = conn(:get, "/api/metrics/prometheus") |> Router.call(@opts)
      assert conn.resp_body =~ "# TYPE"
    end

    test "body contains uptime_seconds" do
      conn = conn(:get, "/api/metrics/prometheus") |> Router.call(@opts)
      assert conn.resp_body =~ "optimal_engine_uptime_seconds"
    end

    test "existing /api/metrics JSON endpoint is not affected" do
      conn = conn(:get, "/api/metrics") |> Router.call(@opts)
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "counters")
      assert Map.has_key?(body, "histograms")
      assert Map.has_key?(body, "uptime_ms")
    end
  end
end
