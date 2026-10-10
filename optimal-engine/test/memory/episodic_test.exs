defmodule OptimalEngine.Memory.EpisodicTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Episodic

  setup do
    :ets.delete_all_objects(:optimal_engine_memory_store)
    :ets.delete_all_objects(:optimal_engine_memory_collections)
    :ok
  end

  describe "record_episode/2" do
    test "records a pattern episode" do
      assert :ok =
               Episodic.record_episode(:pattern, %{
                 description: "Users ask about config frequently",
                 context: "Support chat",
                 outcome: "Created FAQ",
                 tags: ["support"]
               })

      {:ok, patterns} = Episodic.patterns()
      assert length(patterns) == 1
      assert hd(patterns).type == :pattern
      assert hd(patterns).description == "Users ask about config frequently"
    end

    test "records a solution episode" do
      assert :ok =
               Episodic.record_episode(:solution, %{
                 description: "Use ETS for fast local cache",
                 context: "Performance optimization",
                 outcome: "10x speedup",
                 tags: ["performance", "ets"]
               })

      {:ok, solutions} = Episodic.solutions()
      assert length(solutions) == 1
      assert hd(solutions).outcome == "10x speedup"
    end

    test "records a decision episode" do
      assert :ok =
               Episodic.record_episode(:decision, %{
                 description: "Use GenServer per session",
                 context: "Architecture design",
                 outcome: "Clean isolation"
               })

      {:ok, decisions} = Episodic.decisions()
      assert length(decisions) == 1
    end

    test "sets created_at timestamp" do
      Episodic.record_episode(:pattern, %{description: "test"})
      {:ok, [ep]} = Episodic.patterns()
      assert %DateTime{} = ep.created_at
    end
  end

  describe "recall_similar/1" do
    test "finds episodes by keyword" do
      Episodic.record_episode(:pattern, %{
        description: "Elixir GenServer pattern",
        tags: ["elixir"]
      })

      Episodic.record_episode(:solution, %{
        description: "Use Rust for speed",
        tags: ["rust"]
      })

      {:ok, matches} = Episodic.recall_similar("elixir")
      assert length(matches) >= 1
      assert Enum.any?(matches, &String.contains?(&1.description, "Elixir"))
    end

    test "returns empty for no matches" do
      Episodic.record_episode(:pattern, %{description: "test"})
      {:ok, matches} = Episodic.recall_similar("zzzznothing")
      assert matches == []
    end
  end

  describe "patterns/0, solutions/0, decisions/0" do
    test "filters by type" do
      Episodic.record_episode(:pattern, %{description: "P1"})
      Episodic.record_episode(:solution, %{description: "S1"})
      Episodic.record_episode(:decision, %{description: "D1"})

      {:ok, patterns} = Episodic.patterns()
      {:ok, solutions} = Episodic.solutions()
      {:ok, decisions} = Episodic.decisions()

      assert Enum.all?(patterns, &(&1.type == :pattern))
      assert Enum.all?(solutions, &(&1.type == :solution))
      assert Enum.all?(decisions, &(&1.type == :decision))
    end
  end

  describe "all/0" do
    test "returns all episodes" do
      Episodic.record_episode(:pattern, %{description: "P"})
      Episodic.record_episode(:solution, %{description: "S"})

      {:ok, all} = Episodic.all()
      assert length(all) == 2
    end

    test "returns empty when no episodes" do
      {:ok, all} = Episodic.all()
      assert all == []
    end
  end
end
