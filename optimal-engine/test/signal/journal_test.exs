defmodule OptimalEngine.Signal.JournalTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Journal

  setup do
    name = :"journal_#{System.unique_integer([:positive])}"
    {:ok, pid} = Journal.start_link(name: name, max_size: 100)
    %{journal: name, pid: pid}
  end

  describe "record/2 and size/1" do
    test "records a signal", %{journal: journal} do
      signal = Signal.new!("miosa.test", source: "/t")
      :ok = Journal.record(journal, signal)

      assert Journal.size(journal) == 1
    end

    test "records multiple signals", %{journal: journal} do
      for i <- 1..5 do
        signal = Signal.new!("miosa.test.#{i}", source: "/t")
        :ok = Journal.record(journal, signal)
      end

      assert Journal.size(journal) == 5
    end
  end

  describe "history/2" do
    test "returns all signals unfiltered", %{journal: journal} do
      for i <- 1..3 do
        :ok = Journal.record(journal, Signal.new!("miosa.test.#{i}", source: "/t"))
      end

      history = Journal.history(journal)
      assert length(history) == 3
    end

    test "filters by exact type", %{journal: journal} do
      :ok = Journal.record(journal, Signal.new!("miosa.agent.task", source: "/t"))
      :ok = Journal.record(journal, Signal.new!("miosa.system.event", source: "/t"))

      history = Journal.history(journal, type: "miosa.agent.task")
      assert length(history) == 1
      assert hd(history).type == "miosa.agent.task"
    end

    test "filters by type prefix with wildcard", %{journal: journal} do
      :ok = Journal.record(journal, Signal.new!("miosa.agent.task", source: "/t"))
      :ok = Journal.record(journal, Signal.new!("miosa.agent.deploy", source: "/t"))
      :ok = Journal.record(journal, Signal.new!("miosa.system.event", source: "/t"))

      history = Journal.history(journal, type: "miosa.agent.*")
      assert length(history) == 2
    end

    test "filters by source", %{journal: journal} do
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/agent/1"))
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/agent/2"))

      history = Journal.history(journal, source: "/agent/1")
      assert length(history) == 1
    end

    test "filters by agent_id", %{journal: journal} do
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/t", agent_id: "a1"))
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/t", agent_id: "a2"))

      history = Journal.history(journal, agent_id: "a1")
      assert length(history) == 1
    end

    test "filters by session_id", %{journal: journal} do
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/t", session_id: "s1"))
      :ok = Journal.record(journal, Signal.new!("miosa.test", source: "/t", session_id: "s2"))

      history = Journal.history(journal, session_id: "s1")
      assert length(history) == 1
    end

    test "filters by time range", %{journal: journal} do
      early = DateTime.add(DateTime.utc_now(), -3600, :second)
      late = DateTime.add(DateTime.utc_now(), 3600, :second)

      :ok = Journal.record(journal, Signal.new!("miosa.early", source: "/t", time: early))
      :ok = Journal.record(journal, Signal.new!("miosa.late", source: "/t", time: late))

      now = DateTime.utc_now()
      history = Journal.history(journal, since: now)
      assert length(history) == 1
      assert hd(history).type == "miosa.late"
    end

    test "respects limit", %{journal: journal} do
      for i <- 1..10 do
        :ok = Journal.record(journal, Signal.new!("miosa.test.#{i}", source: "/t"))
      end

      history = Journal.history(journal, limit: 3)
      assert length(history) == 3
    end
  end

  describe "causality_chain/2" do
    test "traces single parent-child", %{journal: journal} do
      parent = Signal.new!("miosa.parent", source: "/t")
      {:ok, child} = Signal.chain(parent, "miosa.child")

      :ok = Journal.record(journal, parent)
      :ok = Journal.record(journal, child)

      chain = Journal.causality_chain(journal, child.id)
      assert length(chain) == 2
      assert hd(chain).id == parent.id
      assert List.last(chain).id == child.id
    end

    test "traces multi-level chain", %{journal: journal} do
      root = Signal.new!("miosa.root", source: "/t")
      {:ok, mid} = Signal.chain(root, "miosa.mid")
      {:ok, leaf} = Signal.chain(mid, "miosa.leaf")

      :ok = Journal.record(journal, root)
      :ok = Journal.record(journal, mid)
      :ok = Journal.record(journal, leaf)

      chain = Journal.causality_chain(journal, leaf.id)
      assert length(chain) == 3
      assert hd(chain).id == root.id
      assert List.last(chain).id == leaf.id
    end

    test "returns single signal when no parent", %{journal: journal} do
      signal = Signal.new!("miosa.solo", source: "/t")
      :ok = Journal.record(journal, signal)

      chain = Journal.causality_chain(journal, signal.id)
      assert length(chain) == 1
      assert hd(chain).id == signal.id
    end

    test "returns empty list for unknown id", %{journal: journal} do
      chain = Journal.causality_chain(journal, "nonexistent")
      assert chain == []
    end
  end

  describe "by_correlation/2" do
    test "finds signals by correlation_id", %{journal: journal} do
      corr = "shared-correlation"
      parent = Signal.new!("miosa.parent", source: "/t", correlation_id: corr)
      {:ok, child1} = Signal.chain(parent, "miosa.child.1")
      {:ok, child2} = Signal.chain(parent, "miosa.child.2")

      :ok = Journal.record(journal, parent)
      :ok = Journal.record(journal, child1)
      :ok = Journal.record(journal, child2)

      # parent and children all share the same correlation_id
      group = Journal.by_correlation(journal, corr)
      assert length(group) == 3
    end

    test "returns empty list for unknown correlation", %{journal: journal} do
      group = Journal.by_correlation(journal, "nonexistent")
      assert group == []
    end
  end

  describe "clear/1" do
    test "removes all signals", %{journal: journal} do
      for i <- 1..5 do
        :ok = Journal.record(journal, Signal.new!("miosa.test.#{i}", source: "/t"))
      end

      assert Journal.size(journal) == 5
      :ok = Journal.clear(journal)
      assert Journal.size(journal) == 0
    end
  end

  describe "eviction" do
    test "evicts oldest when max_size reached" do
      name = :"journal_evict_#{System.unique_integer([:positive])}"
      {:ok, _pid} = Journal.start_link(name: name, max_size: 10)

      for i <- 1..15 do
        :ok = Journal.record(name, Signal.new!("miosa.test.#{i}", source: "/t"))
      end

      # After inserting 15 items with max 10, some should be evicted
      size = Journal.size(name)
      assert size <= 10
    end
  end
end
