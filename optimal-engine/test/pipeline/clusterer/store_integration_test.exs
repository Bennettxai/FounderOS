defmodule OptimalEngine.Pipeline.Clusterer.StoreIntegrationTest do
  @moduledoc """
  Round-trips clusters + cluster_members through the Store. Uses synthetic
  embeddings to stay Ollama-free.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Clusterer
  alias OptimalEngine.Pipeline.Clusterer.Cluster
  alias OptimalEngine.Store

  test "Store.insert_clusters + insert_cluster_members round-trip" do
    unique = System.unique_integer([:positive])

    cluster =
      Cluster.new(
        id: "default:cluster-#{unique}",
        tenant_id: "default",
        theme: "test theme",
        intent_dominant: :propose_decision,
        member_count: 2,
        centroid: [0.1, 0.2, 0.3]
      )

    assert :ok = Store.insert_clusters([cluster])

    # Seed fake chunk rows so FK constraint holds
    chunk_id_a = "cl-chunk-a-#{unique}"
    chunk_id_b = "cl-chunk-b-#{unique}"

    for cid <- [chunk_id_a, chunk_id_b] do
      Store.raw_query(
        "INSERT INTO chunks (id, tenant_id, signal_id, scale, text) VALUES (?1, 'default', 'sig', 'document', 't')",
        [cid]
      )
    end

    assert :ok =
             Store.insert_cluster_members([
               %{cluster_id: cluster.id, chunk_id: chunk_id_a, tenant_id: "default", weight: 1.0},
               %{cluster_id: cluster.id, chunk_id: chunk_id_b, tenant_id: "default", weight: 0.8}
             ])

    {:ok, cluster_rows} =
      Store.raw_query(
        "SELECT theme, intent_dominant, member_count FROM clusters WHERE id = ?1",
        [cluster.id]
      )

    assert [["test theme", "propose_decision", 2]] = cluster_rows

    {:ok, member_rows} =
      Store.raw_query(
        "SELECT chunk_id, weight FROM cluster_members WHERE cluster_id = ?1 ORDER BY chunk_id",
        [cluster.id]
      )

    assert length(member_rows) == 2
    assert Enum.map(member_rows, &hd/1) |> Enum.sort() == Enum.sort([chunk_id_a, chunk_id_b])
  end

  test "Clusterer.rebuild/2 against a real tenant computes + persists clusters" do
    unique = System.unique_integer([:positive])
    tenant_id = "rebuild-#{unique}"

    # Seed default tenant row so FKs downstream stay clean
    Store.raw_query("INSERT OR IGNORE INTO tenants (id, name) VALUES (?1, ?2)", [
      tenant_id,
      "Rebuild Test"
    ])

    # Create 4 chunks with embeddings: two close pairs.
    chunks = [
      {"rc-a-#{unique}", [1.0, 0.0, 0.0]},
      {"rc-b-#{unique}", [0.98, 0.01, 0.0]},
      {"rc-c-#{unique}", [0.0, 1.0, 0.0]},
      {"rc-d-#{unique}", [0.01, 0.99, 0.0]}
    ]

    for {id, vec} <- chunks do
      # Insert context + chunk + embedding
      Store.raw_query(
        "INSERT OR IGNORE INTO contexts (id, tenant_id, title, node, type, content) VALUES (?1, ?2, ?3, ?4, 'resource', '')",
        [id, tenant_id, id, "test"]
      )

      Store.raw_query(
        "INSERT INTO chunks (id, tenant_id, signal_id, scale, text) VALUES (?1, ?2, ?1, 'document', '')",
        [id, tenant_id]
      )

      emb =
        OptimalEngine.Pipeline.Embedder.Embedding.new(
          chunk_id: id,
          tenant_id: tenant_id,
          model: "test",
          modality: :text,
          dim: 3,
          vector: vec
        )

      Store.insert_embeddings([emb])
    end

    {:ok, %{clusters: c_count, members: m_count}} = Clusterer.rebuild(tenant_id)

    assert c_count == 2
    assert m_count == 4

    {:ok, cluster_rows} =
      Store.raw_query("SELECT id, member_count FROM clusters WHERE tenant_id = ?1", [tenant_id])

    # Each cluster should have 2 members
    assert length(cluster_rows) == 2

    Enum.each(cluster_rows, fn [_id, count] ->
      assert count == 2
    end)
  end
end
