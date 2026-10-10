defmodule OptimalEngine.Signal.EnvelopeTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Signal.Envelope, as: Signal

  describe "new/2" do
    test "generates UUID v4 id" do
      {:ok, s} = Signal.new("miosa.test", source: "/t")
      # UUID v4 format: 8-4-4-4-12 hex chars
      assert Regex.match?(
               ~r/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
               s.id
             )
    end

    test "defaults specversion to 1.0.2" do
      {:ok, s} = Signal.new("miosa.test", source: "/t")
      assert s.specversion == "1.0.2"
    end

    test "defaults datacontenttype to application/json" do
      {:ok, s} = Signal.new("miosa.test", source: "/t")
      assert s.datacontenttype == "application/json"
    end

    test "defaults extensions to empty map" do
      {:ok, s} = Signal.new("miosa.test", source: "/t")
      assert s.extensions == %{}
    end

    test "accepts all Signal Theory dimensions" do
      {:ok, s} =
        Signal.new("miosa.test",
          source: "/t",
          signal_mode: :linguistic,
          signal_genre: :chat,
          signal_type: :express,
          signal_format: :markdown,
          signal_structure: :conversation,
          signal_sn_ratio: 0.75
        )

      assert s.signal_mode == :linguistic
      assert s.signal_genre == :chat
      assert s.signal_type == :express
      assert s.signal_format == :markdown
      assert s.signal_structure == :conversation
      assert s.signal_sn_ratio == 0.75
    end
  end

  describe "validate/1" do
    test "passes for valid signal with required fields" do
      assert :ok =
               Signal.validate(%Signal{
                 id: "abc",
                 source: "/s",
                 type: "t",
                 specversion: "1.0.2"
               })
    end

    test "fails when id is nil" do
      {:error, reasons} = Signal.validate(%Signal{type: "t", specversion: "1.0.2"})
      assert "id is required" in reasons
    end

    test "fails when type is nil" do
      {:error, reasons} = Signal.validate(%Signal{id: "a", specversion: "1.0.2"})
      assert "type is required" in reasons
    end

    test "fails when specversion is nil" do
      {:error, reasons} = Signal.validate(%Signal{id: "a", type: "t", specversion: nil})
      assert "specversion is required" in reasons
    end

    test "rejects invalid signal_mode" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", signal_mode: :bad})

      assert Enum.any?(reasons, &String.contains?(&1, "signal_mode"))
    end

    test "rejects invalid signal_genre" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", signal_genre: :bad})

      assert Enum.any?(reasons, &String.contains?(&1, "signal_genre"))
    end

    test "rejects invalid signal_type" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", signal_type: :bad})

      assert Enum.any?(reasons, &String.contains?(&1, "signal_type"))
    end

    test "rejects invalid signal_format" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", signal_format: :bad})

      assert Enum.any?(reasons, &String.contains?(&1, "signal_format"))
    end

    test "rejects invalid agent_tier" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", agent_tier: :bad})

      assert Enum.any?(reasons, &String.contains?(&1, "agent_tier"))
    end

    test "rejects sn_ratio out of range" do
      {:error, reasons} =
        Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2", signal_sn_ratio: 1.5})

      assert Enum.any?(reasons, &String.contains?(&1, "signal_sn_ratio"))
    end

    test "accepts sn_ratio at boundaries" do
      assert :ok =
               Signal.validate(%Signal{
                 id: "a",
                 type: "t",
                 specversion: "1.0.2",
                 signal_sn_ratio: 0.0
               })

      assert :ok =
               Signal.validate(%Signal{
                 id: "a",
                 type: "t",
                 specversion: "1.0.2",
                 signal_sn_ratio: 1.0
               })
    end

    test "accepts nil for all optional fields" do
      assert :ok = Signal.validate(%Signal{id: "a", type: "t", specversion: "1.0.2"})
    end
  end

  describe "classify/2" do
    test "preserves existing dimensions when not overridden" do
      {:ok, s} = Signal.new("miosa.test", source: "/t", signal_mode: :code)
      {:ok, classified} = Signal.classify(s, genre: :spec)

      assert classified.signal_mode == :code
      assert classified.signal_genre == :spec
    end

    test "overrides existing dimensions" do
      {:ok, s} = Signal.new("miosa.test", source: "/t", signal_mode: :code)
      {:ok, classified} = Signal.classify(s, mode: :linguistic)

      assert classified.signal_mode == :linguistic
    end

    test "sets sn_ratio via classify" do
      {:ok, s} = Signal.new("miosa.test", source: "/t")
      {:ok, classified} = Signal.classify(s, sn_ratio: 0.92)

      assert classified.signal_sn_ratio == 0.92
    end
  end

  describe "chain/3" do
    test "child gets new id" do
      parent = Signal.new!("miosa.parent", source: "/t")
      {:ok, child} = Signal.chain(parent, "miosa.child")

      assert child.id != parent.id
    end

    test "child type is the provided type" do
      parent = Signal.new!("miosa.parent", source: "/t")
      {:ok, child} = Signal.chain(parent, "miosa.child.step")

      assert child.type == "miosa.child.step"
    end

    test "child inherits agent context" do
      parent =
        Signal.new!("miosa.parent",
          source: "/agent/1",
          agent_id: "agent-1",
          agent_tier: :elite,
          session_id: "sess-1"
        )

      {:ok, child} = Signal.chain(parent, "miosa.child")

      assert child.agent_id == "agent-1"
      assert child.agent_tier == :elite
      assert child.session_id == "sess-1"
    end

    test "opts override inherited values" do
      parent = Signal.new!("miosa.parent", source: "/agent/1", agent_id: "agent-1")
      {:ok, child} = Signal.chain(parent, "miosa.child", agent_id: "agent-2")

      assert child.agent_id == "agent-2"
    end
  end

  describe "to_cloud_event/1" do
    test "includes all required CloudEvents fields" do
      signal = Signal.new!("miosa.test.event", source: "/test")
      ce = Signal.to_cloud_event(signal)

      assert ce["specversion"] == "1.0.2"
      assert ce["id"] == signal.id
      assert ce["source"] == "/test"
      assert ce["type"] == "miosa.test.event"
    end

    test "serializes Signal Theory extensions with miosa_ prefix" do
      signal =
        Signal.new!("miosa.test",
          source: "/t",
          signal_mode: :code,
          signal_genre: :spec,
          agent_tier: :elite
        )

      ce = Signal.to_cloud_event(signal)

      assert ce["miosa_signal_mode"] == "code"
      assert ce["miosa_signal_genre"] == "spec"
      assert ce["miosa_agent_tier"] == "elite"
    end

    test "serializes custom extensions with miosa_ext_ prefix" do
      signal = Signal.new!("miosa.test", source: "/t", extensions: %{custom_key: "value"})
      ce = Signal.to_cloud_event(signal)

      assert ce["miosa_ext_custom_key"] == "value"
    end

    test "omits nil values" do
      signal = Signal.new!("miosa.test", source: "/t")
      ce = Signal.to_cloud_event(signal)

      refute Map.has_key?(ce, "miosa_signal_mode")
      refute Map.has_key?(ce, "subject")
      refute Map.has_key?(ce, "data")
    end

    test "formats time as ISO 8601" do
      {:ok, time, _} = DateTime.from_iso8601("2026-03-07T12:00:00Z")
      signal = Signal.new!("miosa.test", source: "/t", time: time)
      ce = Signal.to_cloud_event(signal)

      assert ce["time"] == "2026-03-07T12:00:00Z"
    end
  end

  describe "from_cloud_event/1" do
    test "deserializes required fields" do
      ce = %{
        "specversion" => "1.0.2",
        "id" => "test-id",
        "source" => "/test",
        "type" => "miosa.test"
      }

      {:ok, signal} = Signal.from_cloud_event(ce)

      assert signal.id == "test-id"
      assert signal.source == "/test"
      assert signal.type == "miosa.test"
      assert signal.specversion == "1.0.2"
    end

    test "deserializes Signal Theory extensions" do
      ce = %{
        "specversion" => "1.0.2",
        "id" => "test-id",
        "source" => "/test",
        "type" => "miosa.test",
        "miosa_signal_mode" => "code",
        "miosa_signal_genre" => "spec",
        "miosa_signal_type" => "inform",
        "miosa_signal_format" => "json",
        "miosa_agent_tier" => "elite"
      }

      {:ok, signal} = Signal.from_cloud_event(ce)

      assert signal.signal_mode == :code
      assert signal.signal_genre == :spec
      assert signal.signal_type == :inform
      assert signal.signal_format == :json
      assert signal.agent_tier == :elite
    end

    test "deserializes custom extensions" do
      ce = %{
        "specversion" => "1.0.2",
        "id" => "test-id",
        "source" => "/test",
        "type" => "miosa.test",
        "miosa_ext_custom" => "value"
      }

      {:ok, signal} = Signal.from_cloud_event(ce)
      assert signal.extensions == %{"custom" => "value"}
    end

    test "rejects invalid cloud event" do
      ce = %{"specversion" => "1.0.2"}
      {:error, reasons} = Signal.from_cloud_event(ce)
      assert "id is required" in reasons
    end
  end

  describe "shannon_check/2" do
    test "passes for small data" do
      signal = Signal.new!("miosa.test", source: "/t", data: "small")
      assert :ok = Signal.shannon_check(signal, 10_000)
    end

    test "fails for data exceeding budget" do
      big_data = String.duplicate("x", 10_000)
      signal = Signal.new!("miosa.test", source: "/t", data: big_data)
      assert {:violation, msg} = Signal.shannon_check(signal, 100)
      assert String.contains?(msg, "exceeds channel capacity")
    end
  end

  describe "ashby_check/1" do
    test "passes when all dimensions resolved" do
      signal =
        Signal.new!("miosa.test",
          source: "/t",
          signal_mode: :code,
          signal_genre: :spec,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :openapi
        )

      assert :ok = Signal.ashby_check(signal)
    end

    test "fails when dimensions missing" do
      signal = Signal.new!("miosa.test", source: "/t")
      assert {:violation, msg} = Signal.ashby_check(signal)
      assert String.contains?(msg, "unresolved dimensions")
    end

    test "fails with partial dimensions" do
      signal = Signal.new!("miosa.test", source: "/t", signal_mode: :code)
      assert {:violation, msg} = Signal.ashby_check(signal)
      assert String.contains?(msg, "genre")
    end
  end

  describe "beer_check/1" do
    test "passes for well-structured signal" do
      signal =
        Signal.new!("miosa.agent.task.completed",
          source: "/t",
          data: %{result: "ok"}
        )

      assert :ok = Signal.beer_check(signal)
    end

    test "fails when parent_id set without correlation_id" do
      signal = %Signal{
        id: "a",
        type: "miosa.test",
        specversion: "1.0.2",
        parent_id: "parent-1",
        correlation_id: nil,
        extensions: %{}
      }

      assert {:violation, msg} = Signal.beer_check(signal)
      assert String.contains?(msg, "causality chain")
    end
  end

  describe "wiener_check/2" do
    test "passes when signal is acknowledged" do
      signal = Signal.new!("miosa.test", source: "/t")
      assert :ok = Signal.wiener_check(signal, [signal.id])
    end

    test "passes when signal is a response (has parent_id)" do
      signal =
        Signal.new!("miosa.response",
          source: "/t",
          parent_id: "parent-1",
          correlation_id: "corr-1"
        )

      assert :ok = Signal.wiener_check(signal, [])
    end

    test "fails when feedback loop is open" do
      signal = Signal.new!("miosa.request", source: "/t")
      assert {:violation, msg} = Signal.wiener_check(signal, [])
      assert String.contains?(msg, "feedback loop open")
    end
  end
end
