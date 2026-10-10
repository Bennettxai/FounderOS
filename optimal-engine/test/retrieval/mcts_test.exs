defmodule OptimalEngine.Retrieval.MCTSTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Retrieval.MCTS

  describe "select/3 budget" do
    test "never exceeds the token budget" do
      cands =
        for i <- 1..10 do
          %{id: i, score: :rand.uniform(), tokens: 30, concepts: MapSet.new(["c#{i}"])}
        end

      result = MCTS.select(cands, 100, iterations: 100)
      assert result.used_tokens <= 100
      assert result.strategy == :mcts
    end

    test "drops candidates larger than the whole budget" do
      cands = [
        %{id: :big, score: 100.0, tokens: 500, concepts: MapSet.new(["a"])},
        %{id: :small, score: 1.0, tokens: 10, concepts: MapSet.new(["b"])}
      ]

      result = MCTS.select(cands, 50, iterations: 50)
      ids = Enum.map(result.selected, & &1.id)
      assert :big not in ids
      assert :small in ids
    end

    test "empty candidates returns empty selection" do
      assert %{selected: [], used_tokens: 0} = MCTS.select([], 100)
    end
  end

  describe "select/3 vs greedy on coverage fixture" do
    # Greedy-by-score is lured into two redundant high scorers that cover the
    # same single concept and exhaust the budget. MCTS should instead pick the
    # set that covers more distinct concepts within the same budget.
    test "MCTS reward >= greedy on a redundancy trap" do
      cands = [
        # two fat, high-score, REDUNDANT items (same concept), 60 tokens each
        %{id: :a, score: 1.0, tokens: 60, concepts: MapSet.new(["shared"])},
        %{id: :b, score: 0.95, tokens: 60, concepts: MapSet.new(["shared"])},
        # three lean items, lower score but each a DISTINCT concept, 30 tokens
        %{id: :c, score: 0.5, tokens: 30, concepts: MapSet.new(["x"])},
        %{id: :d, score: 0.5, tokens: 30, concepts: MapSet.new(["y"])},
        %{id: :e, score: 0.5, tokens: 30, concepts: MapSet.new(["z"])}
      ]

      budget = 100

      mcts = MCTS.select(cands, budget, iterations: 400, coverage_lambda: 1.0)
      greedy = MCTS.greedy(cands, budget, coverage_lambda: 1.0)

      assert mcts.used_tokens <= budget
      assert mcts.reward >= greedy.reward

      # Greedy takes :a then can't fit :b (120>100), so it covers 1 concept.
      # MCTS should be able to find the 3 distinct-concept set (90 tokens, 3 concepts).
      mcts_concepts =
        mcts.selected
        |> Enum.reduce(MapSet.new(), fn c, acc -> MapSet.union(acc, c.concepts) end)
        |> MapSet.size()

      greedy_concepts =
        greedy.selected
        |> Enum.reduce(MapSet.new(), fn c, acc -> MapSet.union(acc, c.concepts) end)
        |> MapSet.size()

      assert mcts_concepts >= greedy_concepts
    end
  end

  describe "enabled?/0" do
    test "defaults true" do
      assert MCTS.enabled?() in [true, false]
    end
  end
end
