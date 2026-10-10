defmodule OptimalEngine.Pipeline.ClustererTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Clusterer
  alias OptimalEngine.Pipeline.Clusterer.Cluster

  defp feature(id, embedding, opts \\ []) do
    %{
      chunk_id: id,
      tenant_id: Keyword.get(opts, :tenant_id, "default"),
      intent: Keyword.get(opts, :intent),
      feature: %{
        embedding: embedding,
        entities: Keyword.get(opts, :entities, []),
        intent: Keyword.get(opts, :intent),
        node_id: Keyword.get(opts, :node_id),
        node_ancestors: Keyword.get(opts, :node_ancestors, [])
      }
    }
  end

  describe "assign/2 — incremental greedy" do
    test "single feature seeds one cluster" do
      f = feature("c1", [1.0, 0.0, 0.0])

      assert {:ok, [%Cluster{} = cluster], [%{chunk_id: "c1", cluster_id: cid}]} =
               Clusterer.assign([f], existing: [])

      assert cluster.id == cid
      assert cluster.member_count == 1
      assert cluster.centroid == [1.0, 0.0, 0.0]
    end

    test "similar features join the same cluster" do
      f1 = feature("c1", [1.0, 0.0, 0.0], intent: :propose_decision)
      f2 = feature("c2", [0.98, 0.01, 0.0], intent: :propose_decision)

      assert {:ok, clusters, memberships} = Clusterer.assign([f1, f2], existing: [])

      assert length(clusters) == 1
      assert hd(clusters).member_count == 2
      # Both memberships point at the same cluster
      cluster_ids = Enum.map(memberships, & &1.cluster_id) |> Enum.uniq()
      assert length(cluster_ids) == 1
    end

    test "dissimilar features seed distinct clusters" do
      f1 = feature("c1", [1.0, 0.0, 0.0])
      f2 = feature("c2", [-1.0, 0.0, 0.0])

      assert {:ok, clusters, memberships} = Clusterer.assign([f1, f2], existing: [])

      # Dissimilar enough that default threshold (0.65) rejects joining
      assert length(clusters) == 2
      cluster_ids = Enum.map(memberships, & &1.cluster_id) |> Enum.uniq()
      assert length(cluster_ids) == 2
    end

    test "custom threshold can force everything into one cluster" do
      f1 = feature("c1", [1.0, 0.0, 0.0])
      f2 = feature("c2", [-1.0, 0.0, 0.0])

      assert {:ok, clusters, _} = Clusterer.assign([f1, f2], existing: [], threshold: 0.0)
      assert length(clusters) == 1
    end

    test "centroid updates as running mean" do
      f1 = feature("c1", [2.0, 0.0])
      f2 = feature("c2", [2.01, 0.01])

      assert {:ok, [cluster], _} = Clusterer.assign([f1, f2], existing: [], threshold: 0.0)
      # Mean of [2.0, 0.0] and [2.01, 0.01] = [2.005, 0.005]
      [x, y] = cluster.centroid
      assert_in_delta x, 2.005, 0.01
      assert_in_delta y, 0.005, 0.01
    end

    test "intent_dominant captures the first-seen intent for new clusters" do
      f = feature("c1", [1.0, 0.0], intent: :propose_decision)
      assert {:ok, [cluster], _} = Clusterer.assign([f], existing: [])
      assert cluster.intent_dominant == :propose_decision
    end
  end

  describe "assign/2 with existing clusters" do
    test "new feature absorbed into pre-existing cluster if similar" do
      existing =
        Cluster.new(
          tenant_id: "default",
          theme: "pre-existing",
          intent_dominant: :propose_decision,
          member_count: 3,
          centroid: [1.0, 0.0, 0.0]
        )

      f = feature("c-new", [0.99, 0.01, 0.0])

      {:ok, [result], [%{cluster_id: cid}]} = Clusterer.assign([f], existing: [existing])
      assert cid == existing.id
      assert result.member_count == 4
    end

    test "unrelated new feature seeds a new cluster alongside existing" do
      existing =
        Cluster.new(
          tenant_id: "default",
          theme: "pre-existing",
          member_count: 2,
          centroid: [1.0, 0.0, 0.0]
        )

      f = feature("c-new", [-1.0, 0.0, 0.0])

      {:ok, clusters, [%{cluster_id: new_cid}]} = Clusterer.assign([f], existing: [existing])
      refute new_cid == existing.id
      assert length(clusters) == 2
    end
  end

  describe "determinism" do
    test "same input ordering → same assignments" do
      fs = [
        feature("c1", [1.0, 0.0]),
        feature("c2", [0.99, 0.01]),
        feature("c3", [-1.0, 0.0])
      ]

      {:ok, clusters1, memberships1} = Clusterer.assign(fs, existing: [])
      {:ok, clusters2, memberships2} = Clusterer.assign(fs, existing: [])

      # Cluster IDs are random per-run, but the shape should be the same
      assert length(clusters1) == length(clusters2)
      assert length(memberships1) == length(memberships2)
    end
  end
end
