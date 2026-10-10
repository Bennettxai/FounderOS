defmodule OptimalEngine.Signal.Classifier.FailureModesTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier.FailureModes

  # --- Helpers ---

  defp base_signal(overrides \\ []) do
    Signal.new!("miosa.test", Keyword.merge([source: "/test"], overrides))
  end

  defp mode_label(failure_mode_tuple), do: elem(failure_mode_tuple, 0)
  defp detect_modes(signal), do: Enum.map(FailureModes.detect(signal), &mode_label/1)

  # --- Shannon: routing_failure ---

  describe "routing_failure" do
    test "detected when source is nil" do
      signal = Signal.new!("miosa.test")
      # source defaults to nil when not provided
      assert :routing_failure in detect_modes(signal)
    end

    test "not detected when source is set" do
      signal = base_signal()
      refute :routing_failure in detect_modes(signal)
    end

    test "description mentions origin" do
      signal = Signal.new!("miosa.test")
      {_, desc} = Enum.find(FailureModes.detect(signal), fn {m, _} -> m == :routing_failure end)
      assert String.contains?(desc, "source")
    end
  end

  # --- Shannon: bandwidth_overload ---

  describe "bandwidth_overload" do
    test "detected when data map serialized size exceeds 100KB" do
      # Map with 20 large entries generates >100KB inspect output
      large_data =
        Map.new(1..20, fn i ->
          {String.duplicate("k", 3000) <> to_string(i), String.duplicate("v", 3000)}
        end)

      signal = base_signal(data: large_data)
      assert :bandwidth_overload in detect_modes(signal)
    end

    test "not detected for small map data" do
      signal = base_signal(data: %{key: "val"})
      refute :bandwidth_overload in detect_modes(signal)
    end

    test "not detected for nil data" do
      signal = base_signal()
      refute :bandwidth_overload in detect_modes(signal)
    end

    test "description mentions bytes" do
      large_data =
        Map.new(1..20, fn i ->
          {String.duplicate("k", 3000) <> to_string(i), String.duplicate("v", 3000)}
        end)

      signal = base_signal(data: large_data)
      {_, desc} = Enum.find(FailureModes.detect(signal), fn {m, _} -> m == :bandwidth_overload end)
      assert String.contains?(desc, "bytes")
    end
  end

  # --- Shannon: fidelity_failure ---

  describe "fidelity_failure" do
    test "detected when sn_ratio is below 0.3" do
      signal = base_signal(signal_sn_ratio: 0.1)
      assert :fidelity_failure in detect_modes(signal)
    end

    test "detected when sn_ratio is exactly 0.0" do
      signal = base_signal(signal_sn_ratio: 0.0)
      assert :fidelity_failure in detect_modes(signal)
    end

    test "not detected when sn_ratio is exactly 0.3" do
      signal = base_signal(signal_sn_ratio: 0.3)
      refute :fidelity_failure in detect_modes(signal)
    end

    test "not detected when sn_ratio is nil" do
      signal = base_signal()
      # nil means unset — not a fidelity failure, just unclassified
      refute :fidelity_failure in detect_modes(signal)
    end

    test "not detected when sn_ratio is above 0.3" do
      signal = base_signal(signal_sn_ratio: 0.8)
      refute :fidelity_failure in detect_modes(signal)
    end
  end

  # --- Ashby: variety_failure ---

  describe "variety_failure" do
    test "detected when all five dimensions are nil" do
      signal = base_signal()
      assert :variety_failure in detect_modes(signal)
    end

    test "not detected when even one dimension is set" do
      signal = base_signal(signal_mode: :code)
      refute :variety_failure in detect_modes(signal)
    end
  end

  # --- Ashby: structure_failure ---

  describe "structure_failure" do
    test "detected when some but not all dimensions are set" do
      signal = base_signal(signal_mode: :code, signal_genre: :brief)
      assert :structure_failure in detect_modes(signal)
    end

    test "not detected when all dimensions are set" do
      signal =
        base_signal(
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief
        )

      refute :structure_failure in detect_modes(signal)
    end

    test "not detected when no dimensions are set (variety_failure takes over)" do
      signal = base_signal()
      refute :structure_failure in detect_modes(signal)
    end

    test "description mentions unresolved count" do
      signal = base_signal(signal_mode: :code)
      {_, desc} = Enum.find(FailureModes.detect(signal), fn {m, _} -> m == :structure_failure end)
      assert String.contains?(desc, "4 of 5")
    end
  end

  # --- Ashby: genre_mismatch ---

  describe "genre_mismatch" do
    test "detected when declared genre contradicts type-inferred genre" do
      # type has 'error' segment -> inferred :error, but declared :spec
      signal = base_signal(signal_genre: :spec)
      # signal type is "miosa.test" -> inferred :chat, so no mismatch (chat is ignored)
      refute :genre_mismatch in detect_modes(signal)
    end

    test "detected for real genre mismatch" do
      # "miosa.agent.error.occurred" infers :error, but declare :spec
      signal = Signal.new!("miosa.agent.error.occurred", source: "/test", signal_genre: :spec)
      assert :genre_mismatch in detect_modes(signal)
    end

    test "not detected when genre matches inferred genre" do
      signal = Signal.new!("miosa.agent.error.occurred", source: "/test", signal_genre: :error)
      refute :genre_mismatch in detect_modes(signal)
    end

    test "not detected when inferred genre is :chat (default)" do
      signal = base_signal(signal_genre: :report)
      # inferred genre is :chat for "miosa.test" -> no mismatch when inferred is :chat
      refute :genre_mismatch in detect_modes(signal)
    end
  end

  # --- Beer: herniation_failure ---

  describe "herniation_failure" do
    test "detected when parent_id is set without correlation_id" do
      signal = %Signal{
        id: "test-id",
        type: "miosa.test",
        specversion: "1.0.2",
        source: "/test",
        parent_id: "some-parent",
        correlation_id: nil,
        extensions: %{}
      }

      assert :herniation_failure in detect_modes(signal)
    end

    test "not detected when parent_id and correlation_id are both set" do
      signal =
        base_signal(
          parent_id: "some-parent",
          correlation_id: "corr-1"
        )

      refute :herniation_failure in detect_modes(signal)
    end

    test "not detected when parent_id is nil" do
      signal = base_signal()
      refute :herniation_failure in detect_modes(signal)
    end
  end

  # --- Beer: bridge_failure ---

  describe "bridge_failure" do
    test "detected when extensions count exceeds 20" do
      # Build a signal with 21+ extensions
      extensions = for i <- 1..21, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      assert :bridge_failure in detect_modes(signal)
    end

    test "not detected with exactly 20 extensions" do
      extensions = for i <- 1..20, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      refute :bridge_failure in detect_modes(signal)
    end

    test "not detected with empty extensions" do
      signal = base_signal()
      refute :bridge_failure in detect_modes(signal)
    end

    test "description mentions extension count" do
      extensions = for i <- 1..25, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      {_, desc} = Enum.find(FailureModes.detect(signal), fn {m, _} -> m == :bridge_failure end)
      assert String.contains?(desc, "25 extensions")
    end
  end

  # --- Beer: decay_failure ---

  describe "decay_failure" do
    test "detected when signal is older than 24 hours" do
      old_time = DateTime.add(DateTime.utc_now(), -25 * 3600, :second)

      signal = %Signal{
        id: "old-signal",
        type: "miosa.test",
        specversion: "1.0.2",
        source: "/test",
        time: old_time,
        extensions: %{}
      }

      assert :decay_failure in detect_modes(signal)
    end

    test "not detected for recent signals" do
      recent_time = DateTime.add(DateTime.utc_now(), -3600, :second)

      signal = %Signal{
        id: "recent-signal",
        type: "miosa.test",
        specversion: "1.0.2",
        source: "/test",
        time: recent_time,
        extensions: %{}
      }

      refute :decay_failure in detect_modes(signal)
    end

    test "not detected when time is nil" do
      signal = base_signal()
      refute :decay_failure in detect_modes(signal)
    end

    test "description mentions hours" do
      old_time = DateTime.add(DateTime.utc_now(), -48 * 3600, :second)

      signal = %Signal{
        id: "stale",
        type: "miosa.test",
        specversion: "1.0.2",
        source: "/test",
        time: old_time,
        extensions: %{}
      }

      {_, desc} = Enum.find(FailureModes.detect(signal), fn {m, _} -> m == :decay_failure end)
      assert String.contains?(desc, "hours")
    end
  end

  # --- Wiener: feedback_failure ---

  describe "feedback_failure" do
    test "detected for :direct type signal with no correlation_id" do
      signal = base_signal(signal_type: :direct)
      assert :feedback_failure in detect_modes(signal)
    end

    test "not detected when correlation_id is set" do
      signal = base_signal(signal_type: :direct, correlation_id: "corr-1")
      refute :feedback_failure in detect_modes(signal)
    end

    test "not detected when parent_id is set" do
      signal = base_signal(signal_type: :direct, parent_id: "p-1", correlation_id: "corr-1")
      refute :feedback_failure in detect_modes(signal)
    end

    test "not detected for :inform type signal" do
      signal = base_signal(signal_type: :inform)
      refute :feedback_failure in detect_modes(signal)
    end

    test "not detected when signal_type is nil" do
      signal = base_signal()
      refute :feedback_failure in detect_modes(signal)
    end
  end

  # --- Adversarial noise ---

  describe "adversarial_noise" do
    test "detected when extensions exceed 50" do
      extensions = for i <- 1..51, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      assert :adversarial_noise in detect_modes(signal)
    end

    test "not detected with exactly 50 extensions" do
      extensions = for i <- 1..50, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      refute :adversarial_noise in detect_modes(signal)
    end

    test "both bridge_failure and adversarial_noise detected for 51+ extensions" do
      extensions = for i <- 1..51, into: %{}, do: {"ext_#{i}", "val_#{i}"}
      signal = base_signal(extensions: extensions)
      modes = detect_modes(signal)
      assert :bridge_failure in modes
      assert :adversarial_noise in modes
    end
  end

  # --- detect/1 — general behavior ---

  describe "detect/1" do
    test "returns empty list for a well-formed fully-classified signal" do
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

      assert FailureModes.detect(signal) == []
    end

    test "can detect multiple failure modes simultaneously" do
      # nil source -> routing_failure
      # no dimensions -> variety_failure
      signal = Signal.new!("miosa.test")
      modes = detect_modes(signal)
      assert :routing_failure in modes
      assert :variety_failure in modes
    end

    test "all detected failures are tuples with atom + binary" do
      signal = Signal.new!("miosa.test")

      for {mode, description} <- FailureModes.detect(signal) do
        assert is_atom(mode)
        assert is_binary(description)
      end
    end

    test "returns failures in stable order" do
      signal = Signal.new!("miosa.test")
      first_run = FailureModes.detect(signal)
      second_run = FailureModes.detect(signal)
      assert first_run == second_run
    end
  end

  # --- Validation module integration ---

  describe "Signal.Validation integration" do
    alias OptimalEngine.Signal.Envelope.Validation

    test "shannon_check passes for small data" do
      signal = base_signal(data: "hello")
      assert Validation.shannon_check(signal, 1000) == :ok
    end

    test "shannon_check fails when data exceeds max_bytes" do
      signal = base_signal(data: String.duplicate("x", 500))
      assert {:violation, desc} = Validation.shannon_check(signal, 10)
      assert String.contains?(desc, "bytes")
    end

    test "ashby_check passes when all five dimensions are set" do
      signal =
        base_signal(
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief
        )

      assert Validation.ashby_check(signal) == :ok
    end

    test "ashby_check fails with missing dimensions" do
      signal = base_signal()
      assert {:violation, desc} = Validation.ashby_check(signal)
      assert String.contains?(desc, "unresolved dimensions")
    end

    test "beer_check passes for coherent signal" do
      signal = base_signal()
      assert Validation.beer_check(signal) == :ok
    end

    test "beer_check fails when parent_id set without correlation_id" do
      signal = %Signal{
        id: "x",
        type: "miosa.test",
        specversion: "1.0.2",
        source: "/test",
        parent_id: "parent",
        correlation_id: nil,
        extensions: %{}
      }

      assert {:violation, _} = Validation.beer_check(signal)
    end

    test "wiener_check passes when signal is a response (has parent_id)" do
      signal = base_signal(parent_id: "p-1", correlation_id: "corr-1")
      assert Validation.wiener_check(signal, []) == :ok
    end

    test "wiener_check passes when signal id is in acknowledged list" do
      signal = base_signal()
      assert Validation.wiener_check(signal, [signal.id]) == :ok
    end

    test "wiener_check fails when no acknowledgement found" do
      signal = base_signal()
      assert {:violation, desc} = Validation.wiener_check(signal, [])
      assert String.contains?(desc, "feedback loop")
    end
  end
end
