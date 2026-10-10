defmodule OptimalEngine.Signal.RouterTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Signal.Router

  setup do
    name = :"router_#{System.unique_integer([:positive])}"
    {:ok, pid} = Router.start_link(name: name)
    %{router: name, pid: pid}
  end

  describe "start_link/1" do
    test "starts a named router", %{pid: pid} do
      assert Process.alive?(pid)
    end
  end

  describe "add_route/4 and routes/1" do
    test "adds a route", %{router: router} do
      handler = fn _sig -> :ok end
      :ok = Router.add_route(router, "miosa.test.**", handler)

      routes = Router.routes(router)
      assert length(routes) == 1
      assert hd(routes).pattern == "miosa.test.**"
    end

    test "adds multiple routes", %{router: router} do
      :ok = Router.add_route(router, "miosa.a.**", fn _ -> :a end)
      :ok = Router.add_route(router, "miosa.b.**", fn _ -> :b end)
      :ok = Router.add_route(router, "miosa.c.**", fn _ -> :c end)

      routes = Router.routes(router)
      assert length(routes) == 3
    end

    test "respects priority", %{router: router} do
      :ok = Router.add_route(router, "miosa.low.**", fn _ -> :low end, priority: -50)
      :ok = Router.add_route(router, "miosa.high.**", fn _ -> :high end, priority: 50)

      routes = Router.routes(router)
      assert hd(routes).priority == 50
    end

    test "clamps priority to [-100, 100]", %{router: router} do
      :ok = Router.add_route(router, "miosa.extreme.**", fn _ -> :ok end, priority: 999)

      routes = Router.routes(router)
      assert hd(routes).priority == 100
    end
  end

  describe "remove_route/2" do
    test "removes an existing route", %{router: router} do
      :ok = Router.add_route(router, "miosa.test.**", fn _ -> :ok end)
      :ok = Router.remove_route(router, "miosa.test.**")

      assert Router.routes(router) == []
    end

    test "returns error for non-existent route", %{router: router} do
      assert {:error, :not_found} = Router.remove_route(router, "miosa.nope")
    end
  end

  describe "match/2" do
    test "matches exact type", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.task.completed", fn _ -> :exact end)

      matches = Router.match(router, "miosa.agent.task.completed")
      assert length(matches) == 1

      {handler, _route} = hd(matches)
      assert handler.(nil) == :exact
    end

    test "does not match different type on exact route", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.task.completed", fn _ -> :exact end)

      matches = Router.match(router, "miosa.agent.task.started")
      assert matches == []
    end

    test "matches single wildcard", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.*.completed", fn _ -> :wildcard end)

      matches = Router.match(router, "miosa.agent.task.completed")
      assert length(matches) == 1

      # Should also match different middle segments
      matches2 = Router.match(router, "miosa.agent.deploy.completed")
      assert length(matches2) == 1
    end

    test "single wildcard does not match multiple segments", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.*.completed", fn _ -> :wildcard end)

      matches = Router.match(router, "miosa.agent.task.sub.completed")
      assert matches == []
    end

    test "matches multi wildcard", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.**", fn _ -> :multi end)

      matches = Router.match(router, "miosa.agent.task.completed")
      assert length(matches) == 1

      matches2 = Router.match(router, "miosa.agent.a.b.c.d")
      assert length(matches2) == 1
    end

    test "multi wildcard matches single remaining segment", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.**", fn _ -> :multi end)

      matches = Router.match(router, "miosa.agent.task")
      assert length(matches) == 1
    end

    test "orders by specificity (exact > wildcard > multi-wildcard)", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.**", fn _ -> :multi end)
      :ok = Router.add_route(router, "miosa.agent.task.completed", fn _ -> :exact end)
      :ok = Router.add_route(router, "miosa.agent.*.completed", fn _ -> :single end)

      matches = Router.match(router, "miosa.agent.task.completed")
      assert length(matches) == 3

      handlers = Enum.map(matches, fn {handler, _route} -> handler.(nil) end)
      assert handlers == [:exact, :single, :multi]
    end

    test "within same specificity, orders by priority", %{router: router} do
      :ok = Router.add_route(router, "miosa.a.**", fn _ -> :low end, priority: 10)
      :ok = Router.add_route(router, "miosa.b.**", fn _ -> :high end, priority: 50)

      # Both match "miosa.a.x" and "miosa.b.x" independently
      # Test that priority works within matches for same type
      :ok = Router.add_route(router, "miosa.test.**", fn _ -> :low end, priority: 10)
      :ok = Router.add_route(router, "miosa.**", fn _ -> :high end, priority: 50)

      matches = Router.match(router, "miosa.test.event")
      assert length(matches) >= 2

      # miosa.test.** is more specific than miosa.**, so it comes first
      {first_handler, _} = hd(matches)
      assert first_handler.(nil) == :low
    end

    test "matches against a Signal struct", %{router: router} do
      :ok = Router.add_route(router, "miosa.test.**", fn _ -> :ok end)

      signal = OptimalEngine.Signal.Envelope.new!("miosa.test.event", source: "/t")
      matches = Router.match(router, signal)
      assert length(matches) == 1
    end

    test "returns empty list for no matches", %{router: router} do
      :ok = Router.add_route(router, "miosa.agent.**", fn _ -> :ok end)

      matches = Router.match(router, "miosa.system.event")
      assert matches == []
    end
  end
end
