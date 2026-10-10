defmodule OptimalEngine.Signal.ClassifierTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier

  describe "classify/1" do
    test "infers :code mode for map data" do
      signal = Signal.new!("miosa.test", source: "/t", data: %{key: "val"})
      c = Classifier.classify(signal)
      assert c.mode == :code
    end

    test "infers :linguistic mode for plain text data" do
      signal = Signal.new!("miosa.test", source: "/t", data: "hello world")
      c = Classifier.classify(signal)
      assert c.mode == :linguistic
    end

    test "infers :code mode for code-like text" do
      signal =
        Signal.new!("miosa.test", source: "/t", data: "defmodule Foo do\n  def bar, do: :ok\nend")

      c = Classifier.classify(signal)
      assert c.mode == :code
    end

    test "infers genre from type segments" do
      signal = Signal.new!("miosa.agent.error.occurred", source: "/t")
      c = Classifier.classify(signal)
      assert c.genre == :error
    end

    test "infers :alert genre" do
      signal = Signal.new!("miosa.system.alert.fired", source: "/t")
      c = Classifier.classify(signal)
      assert c.genre == :alert
    end

    test "infers :progress genre" do
      signal = Signal.new!("miosa.agent.progress.update", source: "/t")
      c = Classifier.classify(signal)
      assert c.genre == :progress
    end

    test "defaults genre to :chat when no match" do
      signal = Signal.new!("miosa.unknown.something", source: "/t")
      c = Classifier.classify(signal)
      assert c.genre == :chat
    end

    test "infers signal type from last segment" do
      assert Classifier.classify(Signal.new!("miosa.task.completed", source: "/t")).type ==
               :inform

      assert Classifier.classify(Signal.new!("miosa.task.created", source: "/t")).type == :inform
      assert Classifier.classify(Signal.new!("miosa.task.request", source: "/t")).type == :direct
      assert Classifier.classify(Signal.new!("miosa.task.decided", source: "/t")).type == :decide
      assert Classifier.classify(Signal.new!("miosa.task.approved", source: "/t")).type == :commit
    end

    test "infers :json format for map data" do
      signal = Signal.new!("miosa.test", source: "/t", data: %{})
      c = Classifier.classify(signal)
      assert c.format == :json
    end

    test "infers :markdown format for markdown-like text" do
      signal = Signal.new!("miosa.test", source: "/t", data: "# Header\n\n- item 1")
      c = Classifier.classify(signal)
      assert c.format == :markdown
    end

    test "returns a structure key" do
      signal = Signal.new!("miosa.agent.error.occurred", source: "/t")
      c = Classifier.classify(signal)
      assert is_atom(c.structure)
    end
  end

  describe "sn_ratio/1" do
    test "returns float between 0 and 1" do
      signal = Signal.new!("miosa.test", source: "/t")
      ratio = Classifier.sn_ratio(signal)
      assert is_float(ratio)
      assert ratio >= 0.0 and ratio <= 1.0
    end

    test "fully classified signal has higher ratio than unclassified" do
      bare = Signal.new!("miosa.test", source: "/t")

      full =
        Signal.new!("miosa.agent.task.completed",
          source: "/agent/1",
          data: %{result: "ok"},
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief,
          agent_id: "agent-1",
          session_id: "sess-1"
        )

      assert Classifier.sn_ratio(full) > Classifier.sn_ratio(bare)
    end

    test "signal with nil data has lower ratio" do
      with_data = Signal.new!("miosa.test", source: "/t", data: %{x: 1})
      without_data = Signal.new!("miosa.test", source: "/t")

      assert Classifier.sn_ratio(with_data) > Classifier.sn_ratio(without_data)
    end
  end

  describe "check_constraints/2" do
    test "returns all four constraints" do
      signal = Signal.new!("miosa.test", source: "/t")
      results = Classifier.check_constraints(signal)

      assert Map.has_key?(results, :shannon)
      assert Map.has_key?(results, :ashby)
      assert Map.has_key?(results, :beer)
      assert Map.has_key?(results, :wiener)
    end

    test "fully resolved signal passes ashby" do
      signal =
        Signal.new!("miosa.test",
          source: "/t",
          signal_mode: :code,
          signal_genre: :spec,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :default
        )

      results = Classifier.check_constraints(signal)
      assert results.ashby == :ok
    end

    test "respects max_bytes option for shannon" do
      signal = Signal.new!("miosa.test", source: "/t", data: String.duplicate("x", 500))
      results = Classifier.check_constraints(signal, max_bytes: 10)
      assert {:violation, _} = results.shannon
    end

    test "respects acknowledged_ids for wiener" do
      signal = Signal.new!("miosa.test", source: "/t")
      results = Classifier.check_constraints(signal, acknowledged_ids: [signal.id])
      assert results.wiener == :ok
    end
  end

  describe "failure_modes/1" do
    test "detects variety_failure for unclassified signal" do
      signal = Signal.new!("miosa.test", source: "/t")
      failures = Classifier.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _} -> mode end)
      assert :variety_failure in modes
    end

    test "detects structure_failure for partially classified signal" do
      signal = Signal.new!("miosa.test", source: "/t", signal_mode: :code)
      failures = Classifier.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _} -> mode end)
      assert :structure_failure in modes
    end

    test "detects herniation_failure for broken causality" do
      signal = %Signal{
        id: "a",
        type: "miosa.test",
        specversion: "1.0.2",
        parent_id: "p-1",
        correlation_id: nil,
        extensions: %{}
      }

      failures = Classifier.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _} -> mode end)
      assert :herniation_failure in modes
    end

    test "detects feedback_failure for direct signal without correlation" do
      signal = Signal.new!("miosa.test", source: "/t", signal_type: :direct)
      failures = Classifier.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _} -> mode end)
      assert :feedback_failure in modes
    end

    test "detects routing_failure when source is nil" do
      signal = Signal.new!("miosa.test")
      failures = Classifier.failure_modes(signal)
      modes = Enum.map(failures, fn {mode, _} -> mode end)
      assert :routing_failure in modes
    end

    test "returns empty list for well-formed signal" do
      signal =
        Signal.new!("miosa.agent.task.completed",
          source: "/agent/1",
          data: %{result: "ok"},
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief,
          agent_id: "agent-1",
          session_id: "sess-1",
          correlation_id: "corr-1"
        )

      failures = Classifier.failure_modes(signal)
      assert failures == []
    end
  end
end
