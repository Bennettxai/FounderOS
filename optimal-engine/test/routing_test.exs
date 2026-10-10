defmodule OptimalEngine.RoutingTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureLog

  alias OptimalEngine.Routing

  test "missing optional YAML files load empty defaults without warnings" do
    missing_root =
      Path.join(System.tmp_dir!(), "optimal-routing-missing-#{System.unique_integer([:positive])}")

    original_path = Application.get_env(:optimal_engine, :topology_path)
    original_full_path = Application.get_env(:optimal_engine, :topology_full_path)

    on_exit(fn ->
      Application.put_env(:optimal_engine, :topology_path, original_path)
      Application.put_env(:optimal_engine, :topology_full_path, original_full_path)
    end)

    Application.put_env(:optimal_engine, :topology_path, Path.join(missing_root, "config.yaml"))

    Application.put_env(
      :optimal_engine,
      :topology_full_path,
      Path.join(missing_root, "topology.yaml")
    )

    log =
      capture_log(fn ->
        assert {:ok, topology} = Routing.load()
        assert topology.nodes == %{}
        assert topology.routing_rules == []
      end)

    refute log =~ "Could not read YAML"
  end
end
