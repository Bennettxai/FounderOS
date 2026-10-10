defmodule OptimalEngine.Audit.EventTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Audit.{Event, Logger}

  describe "Event.new/2" do
    test "builds an Event with the kind and sensible defaults" do
      event = Event.new("retrieval.executed", principal: "user:ada@sample.test")

      assert event.kind == "retrieval.executed"
      assert event.principal == "user:ada@sample.test"
      assert event.tenant_id == "default"
      assert is_binary(event.ts)
    end
  end

  describe "Logger.log/1 and query/1" do
    test "writes an event and retrieves it by kind" do
      marker = "unit-test-marker-#{System.unique_integer([:positive])}"

      :ok =
        Logger.log("unit.test.event",
          principal: marker,
          target_uri: "optimal://test/#{marker}",
          latency_ms: 12,
          metadata: %{"probe" => marker}
        )

      assert {:ok, events} = Logger.query(principal: marker)
      assert [event | _] = events
      assert event.kind == "unit.test.event"
      assert event.principal == marker
      assert event.target_uri == "optimal://test/#{marker}"
      assert event.latency_ms == 12
      assert event.metadata["probe"] == marker
    end

    test "principal filter restricts results" do
      marker_a = "principal-a-#{System.unique_integer([:positive])}"
      marker_b = "principal-b-#{System.unique_integer([:positive])}"

      :ok = Logger.log("unit.ab.event", principal: marker_a)
      :ok = Logger.log("unit.ab.event", principal: marker_b)

      assert {:ok, events_a} = Logger.query(principal: marker_a, kind: "unit.ab.event")
      assert length(events_a) == 1
      assert hd(events_a).principal == marker_a
    end
  end
end
