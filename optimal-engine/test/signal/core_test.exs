defmodule OptimalEngine.Signal.CoreTest do
  use ExUnit.Case, async: true

  describe "new/2" do
    test "creates signal with type and auto-generated fields" do
      {:ok, signal} = OptimalEngine.Signal.Core.new("miosa.test.event", source: "/test")

      assert signal.type == "miosa.test.event"
      assert signal.specversion == "1.0.2"
      assert is_binary(signal.id)
      assert String.length(signal.id) == 36
      assert %DateTime{} = signal.time
      assert signal.source == "/test"
    end

    test "creates signal with data" do
      {:ok, signal} =
        OptimalEngine.Signal.Core.new("miosa.test", source: "/t", data: %{key: "value"})

      assert signal.data == %{key: "value"}
    end

    test "creates signal with all MIOSA extensions" do
      {:ok, signal} =
        OptimalEngine.Signal.Core.new("miosa.test",
          source: "/t",
          agent_id: "agent-1",
          agent_tier: :elite,
          session_id: "sess-1",
          correlation_id: "corr-1",
          extensions: %{custom: "data"}
        )

      assert signal.agent_id == "agent-1"
      assert signal.agent_tier == :elite
      assert signal.session_id == "sess-1"
      assert signal.correlation_id == "corr-1"
      assert signal.extensions == %{custom: "data"}
    end
  end

  describe "new!/2" do
    test "returns signal directly on success" do
      signal = OptimalEngine.Signal.Core.new!("miosa.test", source: "/t")
      assert signal.type == "miosa.test"
    end

    test "raises on invalid signal" do
      assert_raise ArgumentError, fn ->
        # signal_mode is invalid
        OptimalEngine.Signal.Core.new!("miosa.test", source: "/t", signal_mode: :bogus)
      end
    end
  end

  describe "classify/2" do
    test "applies Signal Theory dimensions" do
      signal = OptimalEngine.Signal.Core.new!("miosa.test", source: "/t")

      {:ok, classified} =
        OptimalEngine.Signal.Core.classify(signal,
          mode: :code,
          genre: :spec,
          type: :inform,
          format: :json,
          structure: :openapi
        )

      assert classified.signal_mode == :code
      assert classified.signal_genre == :spec
      assert classified.signal_type == :inform
      assert classified.signal_format == :json
      assert classified.signal_structure == :openapi
    end

    test "rejects invalid dimension values" do
      signal = OptimalEngine.Signal.Core.new!("miosa.test", source: "/t")
      assert {:error, _} = OptimalEngine.Signal.Core.classify(signal, mode: :invalid_mode)
    end
  end

  describe "auto_classify/1" do
    test "populates all dimensions from heuristics" do
      signal =
        OptimalEngine.Signal.Core.new!("miosa.agent.task.completed",
          source: "/agent/1",
          data: %{result: "ok"}
        )

      {:ok, classified} = OptimalEngine.Signal.Core.auto_classify(signal)

      assert classified.signal_mode != nil
      assert classified.signal_genre != nil
      assert classified.signal_type != nil
      assert classified.signal_format != nil
      assert classified.signal_structure != nil
      assert is_float(classified.signal_sn_ratio)
    end
  end

  describe "chain/3" do
    test "creates child linked to parent" do
      parent =
        OptimalEngine.Signal.Core.new!("miosa.parent", source: "/agent/1", session_id: "sess-1")

      {:ok, child} = OptimalEngine.Signal.Core.chain(parent, "miosa.child", data: %{step: 2})

      assert child.parent_id == parent.id
      assert child.correlation_id == parent.id
      assert child.source == parent.source
      assert child.session_id == "sess-1"
      assert child.data == %{step: 2}
    end

    test "preserves existing correlation_id" do
      parent =
        OptimalEngine.Signal.Core.new!("miosa.parent",
          source: "/t",
          correlation_id: "original-corr"
        )

      {:ok, child} = OptimalEngine.Signal.Core.chain(parent, "miosa.child")

      assert child.correlation_id == "original-corr"
    end
  end

  describe "to_json/1 and from_json/1" do
    test "round-trips a signal through JSON" do
      signal =
        OptimalEngine.Signal.Core.new!("miosa.agent.task.completed",
          source: "/agent/backend-1",
          data: %{"result" => "success"},
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief,
          signal_sn_ratio: 0.85,
          agent_id: "agent-1",
          agent_tier: :specialist
        )

      json = OptimalEngine.Signal.Core.to_json(signal)
      assert is_binary(json)

      {:ok, restored} = OptimalEngine.Signal.Core.from_json(json)
      assert restored.type == signal.type
      assert restored.source == signal.source
      assert restored.data == signal.data
      assert restored.signal_mode == :code
      assert restored.signal_genre == :brief
      assert restored.agent_tier == :specialist
      assert restored.signal_sn_ratio == 0.85
    end

    test "handles invalid JSON" do
      assert {:error, _} = OptimalEngine.Signal.Core.from_json("not json")
    end
  end

  describe "check_constraints/2" do
    test "returns all four constraint results" do
      signal =
        OptimalEngine.Signal.Core.new!("miosa.test",
          source: "/t",
          signal_mode: :code,
          signal_genre: :spec,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :default
        )

      results = OptimalEngine.Signal.Core.check_constraints(signal)

      assert Map.has_key?(results, :shannon)
      assert Map.has_key?(results, :ashby)
      assert Map.has_key?(results, :beer)
      assert Map.has_key?(results, :wiener)
      assert results.ashby == :ok
      assert results.shannon == :ok
    end
  end

  describe "failure_modes/1" do
    test "returns list of detected failures" do
      signal = OptimalEngine.Signal.Core.new!("miosa.test", source: "/t")
      failures = OptimalEngine.Signal.Core.failure_modes(signal)
      assert is_list(failures)
    end

    test "detects variety_failure for unclassified signal" do
      signal = OptimalEngine.Signal.Core.new!("miosa.test", source: "/t")
      failures = OptimalEngine.Signal.Core.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _desc} -> mode end)
      assert :variety_failure in modes
    end
  end
end
