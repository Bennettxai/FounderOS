defmodule OptimalEngine.Retrieval.BandwidthPlannerTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Retrieval.BandwidthPlanner

  describe "plan/2" do
    test "keeps everything when budget is ample" do
      items = [
        %{content: "short one", score: 1.0},
        %{content: "short two", score: 0.9}
      ]

      plan = BandwidthPlanner.plan(items, 10_000)
      assert length(plan.kept) == 2
      assert plan.dropped == []
      refute plan.truncated?
    end

    test "orders by score descending and drops the tail when budget tight" do
      big = String.duplicate("x", 400)

      items = [
        %{content: big, score: 0.1, uri: "low"},
        %{content: big, score: 0.9, uri: "high"},
        %{content: big, score: 0.5, uri: "mid"}
      ]

      plan = BandwidthPlanner.plan(items, 200)
      assert Enum.map(plan.kept, & &1.uri) |> List.first() == "high"
      assert plan.truncated?
    end

    test "zero budget drops everything" do
      items = [%{content: "a", score: 1.0}]
      plan = BandwidthPlanner.plan(items, 0)
      assert plan.kept == []
      assert plan.dropped == items
      assert plan.truncated?
    end

    test "treats missing :score as 0 (ordered last)" do
      items = [
        %{content: "no score"},
        %{content: "high", score: 0.9}
      ]

      plan = BandwidthPlanner.plan(items, 10_000)
      assert [first | _] = plan.kept
      assert first.content == "high"
    end

    test "preserves extra keys on items" do
      items = [%{content: "a", score: 1.0, uri: "optimal://x", foo: :bar}]
      plan = BandwidthPlanner.plan(items, 10_000)
      [kept] = plan.kept
      assert kept.uri == "optimal://x"
      assert kept.foo == :bar
    end
  end

  describe "estimate_tokens/1" do
    test "4-chars-per-token heuristic" do
      assert BandwidthPlanner.estimate_tokens("abcd") == 1
      assert BandwidthPlanner.estimate_tokens("abcdefgh") == 2
    end

    test "handles nil and non-strings" do
      assert BandwidthPlanner.estimate_tokens(nil) == 0
      assert BandwidthPlanner.estimate_tokens(%{content: "abcdefgh"}) == 2
    end
  end

  describe "truncate/2" do
    test "returns full string when under budget" do
      assert BandwidthPlanner.truncate("hello", 100) == "hello"
    end

    test "trims and adds ellipsis when over" do
      big = String.duplicate("a", 100)
      out = BandwidthPlanner.truncate(big, 5)
      # 5 tokens = 20 chars
      assert byte_size(out) <= 25
      assert String.ends_with?(out, "…")
    end

    test "zero budget returns empty string" do
      assert BandwidthPlanner.truncate("anything", 0) == ""
    end
  end
end
