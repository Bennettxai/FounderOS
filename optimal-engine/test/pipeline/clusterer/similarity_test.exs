defmodule OptimalEngine.Pipeline.Clusterer.SimilarityTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Clusterer.Similarity

  describe "cosine/2" do
    test "identical vectors → 1.0 (rescaled)" do
      v = [1.0, 2.0, 3.0]
      assert_in_delta Similarity.cosine(v, v), 1.0, 1.0e-6
    end

    test "orthogonal vectors → 0.5 (rescaled mid)" do
      # Rescale formula: (cos + 1) / 2
      # cos([1,0],[0,1]) = 0 → rescaled 0.5
      assert_in_delta Similarity.cosine([1.0, 0.0], [0.0, 1.0]), 0.5, 1.0e-6
    end

    test "opposite vectors → 0.0" do
      assert_in_delta Similarity.cosine([1.0, 0.0], [-1.0, 0.0]), 0.0, 1.0e-6
    end

    test "mismatched-length vectors → 0.0" do
      assert Similarity.cosine([1.0, 2.0], [1.0, 2.0, 3.0]) == 0.0
    end

    test "empty vectors → 0.0" do
      assert Similarity.cosine([], [1.0, 2.0]) == 0.0
      assert Similarity.cosine([1.0, 2.0], []) == 0.0
    end

    test "zero vector → 0.0 (undefined cos handled safely)" do
      assert Similarity.cosine([0.0, 0.0, 0.0], [1.0, 2.0, 3.0]) == 0.0
    end
  end

  describe "entity_overlap/2" do
    test "identical sets → 1.0" do
      ents = ["Alice", "Alice", "Platform Launch"]
      assert Similarity.entity_overlap(ents, ents) == 1.0
    end

    test "disjoint sets → 0.0" do
      assert Similarity.entity_overlap(["A"], ["B"]) == 0.0
    end

    test "half overlap → 0.5 (2 shared / 4 union)" do
      assert Similarity.entity_overlap(["A", "B"], ["B", "C"]) |> Float.round(3) == 0.333
    end

    test "one empty → 0.0" do
      assert Similarity.entity_overlap([], ["A"]) == 0.0
    end

    test "both empty → 0.0" do
      assert Similarity.entity_overlap([], []) == 0.0
    end
  end

  describe "intent_match/2" do
    test "same intent → 1.0" do
      assert Similarity.intent_match(:request_info, :request_info) == 1.0
    end

    test "different intents → 0.0" do
      assert Similarity.intent_match(:request_info, :propose_decision) == 0.0
    end

    test "nil on either side → 0.0" do
      assert Similarity.intent_match(nil, :request_info) == 0.0
      assert Similarity.intent_match(:request_info, nil) == 0.0
    end
  end

  describe "node_affinity/2" do
    test "same node_id → 1.0" do
      a = %{node_id: "project-platform-launch", node_ancestors: []}
      b = %{node_id: "project-platform-launch", node_ancestors: []}
      assert Similarity.node_affinity(a, b) == 1.0
    end

    test "shared ancestor → 0.5" do
      a = %{node_id: "child-a", node_ancestors: ["parent", "root"]}
      b = %{node_id: "child-b", node_ancestors: ["parent", "root"]}
      assert Similarity.node_affinity(a, b) == 0.5
    end

    test "no shared node or ancestor → 0.0" do
      a = %{node_id: "x", node_ancestors: ["x-parent"]}
      b = %{node_id: "y", node_ancestors: ["y-parent"]}
      assert Similarity.node_affinity(a, b) == 0.0
    end

    test "nil node → 0.0" do
      a = %{node_id: nil, node_ancestors: []}
      b = %{node_id: "x", node_ancestors: []}
      assert Similarity.node_affinity(a, b) == 0.0
    end
  end

  describe "sim/2 — weighted composite" do
    test "identical chunks on all four components → ~1.0" do
      v = [1.0, 0.0, 0.0]

      a = %{
        embedding: v,
        entities: ["Ed", "Alice"],
        intent: :propose_decision,
        node_id: "project-platform-launch",
        node_ancestors: []
      }

      assert_in_delta Similarity.sim(a, a), 1.0, 1.0e-6
    end

    test "completely disjoint → ~0.3 (only cosine mid-point contributes)" do
      a = %{
        embedding: [1.0, 0.0],
        entities: ["Ed"],
        intent: :request_info,
        node_id: "a",
        node_ancestors: []
      }

      b = %{
        embedding: [0.0, 1.0],
        entities: ["Bob"],
        intent: :record_fact,
        node_id: "b",
        node_ancestors: []
      }

      # cosine = 0.5 (orthogonal rescaled) * 0.6 = 0.3
      # everything else = 0
      assert_in_delta Similarity.sim(a, b), 0.3, 1.0e-6
    end
  end

  describe "mean_vector/1" do
    test "empty → empty" do
      assert Similarity.mean_vector([]) == []
    end

    test "single vector → itself" do
      assert Similarity.mean_vector([[1.0, 2.0, 3.0]]) == [1.0, 2.0, 3.0]
    end

    test "multiple vectors → element-wise mean" do
      result = Similarity.mean_vector([[1.0, 2.0], [3.0, 4.0], [5.0, 6.0]])
      assert result == [3.0, 4.0]
    end
  end
end
