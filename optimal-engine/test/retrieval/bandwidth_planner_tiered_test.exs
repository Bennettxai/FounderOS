defmodule OptimalEngine.Retrieval.BandwidthPlannerTieredTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Retrieval.BandwidthPlanner

  describe "plan_tiered/2" do
    test "downgrades a high-value item to a leaner tier instead of dropping it" do
      # full content is too big for the remaining budget, but l0_abstract fits.
      big = String.duplicate("x", 4_000)

      items = [
        %{
          score: 1.0,
          content: big,
          l1_overview: String.duplicate("y", 800),
          l0_abstract: "tiny abstract"
        }
      ]

      # Budget large enough only for the L0 abstract (+overhead), not full content.
      plan = BandwidthPlanner.plan_tiered(items, 100)

      assert [kept] = plan.kept
      assert kept.tier == :l0
      assert plan.dropped == []
      assert plan.downgrades == 1
      assert plan.used_tokens <= 100
    end

    test "keeps richest tier (l3) when budget is ample" do
      items = [%{score: 1.0, content: "full text here", l0_abstract: "abs"}]
      plan = BandwidthPlanner.plan_tiered(items, 10_000)
      assert [kept] = plan.kept
      assert kept.tier == :l3
      assert plan.downgrades == 0
    end

    test "drops an item only when even L0 will not fit" do
      items = [%{score: 1.0, content: "x", l0_abstract: String.duplicate("z", 4_000)}]
      plan = BandwidthPlanner.plan_tiered(items, 1)
      assert plan.kept == []
      assert length(plan.dropped) == 1
      assert plan.truncated?
    end

    test "render_tier picks the right field per tier" do
      item = %{content: "FULL", l1_overview: "OVERVIEW", l0_abstract: "ABS"}
      assert BandwidthPlanner.render_tier(item, :l3) == "FULL"
      assert BandwidthPlanner.render_tier(item, :l1) == "OVERVIEW"
      assert BandwidthPlanner.render_tier(item, :l0) == "ABS"
    end

    test "L3 is verbatim passthrough of content" do
      item = %{content: "verbatim payload"}
      assert BandwidthPlanner.render_tier(item, :l3) == "verbatim payload"
    end
  end
end
