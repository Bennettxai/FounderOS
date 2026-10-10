defmodule OptimalEngine.Signal.DispatcherTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.{Dispatcher, Router}
  alias OptimalEngine.MemoryCore.ToolModelGovernance
  alias OptimalEngine.Store

  setup do
    name = :"dispatcher_router_#{System.unique_integer([:positive])}"
    {:ok, _pid} = Router.start_link(name: name)
    %{router: name}
  end

  describe "dispatch/2" do
    test "dispatches to matched handlers via router", %{router: router} do
      test_pid = self()

      :ok =
        Router.add_route(router, "miosa.test.**", fn sig ->
          send(test_pid, {:got, sig.type})
          :handled
        end)

      signal = Signal.new!("miosa.test.event", source: "/t")
      {:ok, results} = Dispatcher.dispatch(signal, router: router)

      assert results == [:handled]
      assert_receive {:got, "miosa.test.event"}
    end

    test "dispatches to multiple handlers", %{router: router} do
      test_pid = self()

      :ok =
        Router.add_route(router, "miosa.test.**", fn _ ->
          send(test_pid, :handler_1)
          :one
        end)

      :ok =
        Router.add_route(router, "miosa.**", fn _ ->
          send(test_pid, :handler_2)
          :two
        end)

      signal = Signal.new!("miosa.test.event", source: "/t")
      {:ok, results} = Dispatcher.dispatch(signal, router: router)

      assert length(results) == 2
      assert_receive :handler_1
      assert_receive :handler_2
    end

    test "dispatches with explicit handlers list (no router)" do
      test_pid = self()

      handlers = [
        fn sig ->
          send(test_pid, {:h1, sig.type})
          :first
        end,
        fn _sig -> :second end
      ]

      signal = Signal.new!("miosa.test", source: "/t")
      {:ok, results} = Dispatcher.dispatch(signal, handlers: handlers)

      assert results == [:first, :second]
      assert_receive {:h1, "miosa.test"}
    end

    test "captures handler errors" do
      handlers = [fn _sig -> raise "boom" end]

      signal = Signal.new!("miosa.test", source: "/t")
      {:error, errors} = Dispatcher.dispatch(signal, handlers: handlers)

      assert length(errors) == 1
    end

    test "delivers via :pid adapter" do
      signal = Signal.new!("miosa.test", source: "/t")

      {:ok, _} =
        Dispatcher.dispatch(signal,
          handlers: [],
          adapter: :pid,
          adapter_opts: [pid: self()]
        )

      assert_receive {:signal, ^signal}
    end

    test "delivers via :named adapter" do
      Process.register(self(), :dispatch_test_named)

      signal = Signal.new!("miosa.test", source: "/t")

      {:ok, _} =
        Dispatcher.dispatch(signal,
          handlers: [],
          adapter: :named,
          adapter_opts: [name: :dispatch_test_named]
        )

      assert_receive {:signal, ^signal}
    after
      try do
        Process.unregister(:dispatch_test_named)
      rescue
        _ -> :ok
      end
    end

    test "noop adapter does nothing" do
      signal = Signal.new!("miosa.test", source: "/t")

      {:ok, []} =
        Dispatcher.dispatch(signal, handlers: [], adapter: :noop)

      refute_receive {:signal, _}
    end

    test "governed adapter blocks unauthorized delivery and records the attempt" do
      workspace_id = "dispatcher-governed-reject-#{System.unique_integer([:positive])}"

      {:ok, _definition} =
        ToolModelGovernance.register_mcp_tool_definition(
          workspace_id: workspace_id,
          tool_name: "signal.dispatch",
          implementation_type: "signal_adapter",
          required_privileges: ["signal:deliver"],
          allowed_partitions: ["ops"],
          input_schema: %{required: ["signal_type"]},
          output_schema: %{required: ["delivered"]}
        )

      signal = Signal.new!("optimal.test.governed", source: "/t")

      assert {:error, [{:error, {:adapter, {:governance_rejected, reason}}}]} =
               Dispatcher.dispatch(signal,
                 handlers: [],
                 adapter: :governed,
                 adapter_opts: [
                   tool_name: "signal.dispatch",
                   inner_adapter: :pid,
                   inner_adapter_opts: [pid: self()],
                   input_payload: %{signal_type: signal.type},
                   governance_opts: [
                     workspace_id: workspace_id,
                     actor_id: "agent:test",
                     granted_privileges: [],
                     requested_partitions: ["ops"]
                   ]
                 ]
               )

      assert reason =~ "missing:signal:deliver"
      refute_receive {:signal, ^signal}

      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM tool_call_runs WHERE workspace_id = ?1 AND decision_state = 'rejected'",
                 [workspace_id]
               )
    end

    test "governed adapter delivers authorized signals and records the completed run" do
      workspace_id = "dispatcher-governed-allow-#{System.unique_integer([:positive])}"

      {:ok, _definition} =
        ToolModelGovernance.register_mcp_tool_definition(
          workspace_id: workspace_id,
          tool_name: "signal.dispatch",
          implementation_type: "signal_adapter",
          required_privileges: ["signal:deliver"],
          allowed_partitions: ["ops"],
          input_schema: %{required: ["signal_type"]},
          output_schema: %{required: ["delivered"]}
        )

      signal = Signal.new!("optimal.test.governed", source: "/t")

      assert {:ok, []} =
               Dispatcher.dispatch(signal,
                 handlers: [],
                 adapter: :governed,
                 adapter_opts: [
                   tool_name: "signal.dispatch",
                   inner_adapter: :pid,
                   inner_adapter_opts: [pid: self()],
                   input_payload: %{signal_type: signal.type},
                   governance_opts: [
                     workspace_id: workspace_id,
                     actor_id: "agent:test",
                     granted_privileges: ["signal:deliver"],
                     requested_partitions: ["ops"]
                   ]
                 ]
               )

      assert_receive {:signal, ^signal}

      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM tool_call_runs WHERE workspace_id = ?1 AND decision_state = 'allowed' AND run_status = 'completed'",
                 [workspace_id]
               )
    end
  end

  describe "dispatch_async/2" do
    test "returns a Task that resolves to dispatch result" do
      handlers = [fn _sig -> :async_result end]

      signal = Signal.new!("miosa.test", source: "/t")
      task = Dispatcher.dispatch_async(signal, handlers: handlers)

      assert %Task{} = task
      {:ok, results} = Task.await(task)
      assert results == [:async_result]
    end
  end

  describe "dispatch_batch/2" do
    test "dispatches multiple signals concurrently" do
      test_pid = self()

      handlers = [
        fn sig ->
          send(test_pid, {:batch, sig.type})
          :ok
        end
      ]

      signals =
        for i <- 1..5 do
          Signal.new!("miosa.batch.#{i}", source: "/t")
        end

      results = Dispatcher.dispatch_batch(signals, handlers: handlers, max_concurrency: 3)

      assert length(results) == 5
      assert Enum.all?(results, &match?({:ok, [:ok]}, &1))

      received =
        for _ <- 1..5 do
          assert_receive {:batch, type}
          type
        end

      for i <- 1..5 do
        assert "miosa.batch.#{i}" in received
      end
    end

    test "handles empty batch" do
      results = Dispatcher.dispatch_batch([], handlers: [fn _ -> :ok end])
      assert results == []
    end
  end
end
