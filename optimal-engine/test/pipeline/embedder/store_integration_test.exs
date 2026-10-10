defmodule OptimalEngine.Pipeline.Embedder.StoreIntegrationTest do
  @moduledoc """
  Verifies Store.insert_embeddings/1 round-trips into the chunk_embeddings
  table created by Phase 5 migration 023. Does NOT require Ollama — uses
  a synthetic embedding to exercise the Store write path.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Embedder.Embedding
  alias OptimalEngine.Pipeline.Parser.ParsedDoc
  alias OptimalEngine.Store

  test "insert_embeddings persists one row per chunk keyed by chunk_id" do
    unique = System.unique_integer([:positive])
    text = "Embedder integration test #{unique}."
    signal_id = "sha256:embedder-#{unique}"
    doc = ParsedDoc.new(text: text, signal_id: signal_id)

    {:ok, tree} = Decomposer.decompose_and_store(doc)

    embeddings =
      Enum.map(tree.chunks, fn c ->
        # Synthetic 768-dim vector (deterministic so tests are repeatable).
        vector = for i <- 0..767, do: :math.sin(i / 100.0)

        Embedding.new(
          chunk_id: c.id,
          tenant_id: c.tenant_id,
          model: "nomic-embed-text",
          modality: c.modality,
          dim: 768,
          vector: vector
        )
      end)

    assert :ok = Store.insert_embeddings(embeddings)

    {:ok, rows} =
      Store.raw_query(
        """
        SELECT chunk_id, model, modality, dim, LENGTH(vector)
        FROM chunk_embeddings
        WHERE chunk_id IN (SELECT id FROM chunks WHERE signal_id = ?1)
        """,
        [signal_id]
      )

    assert length(rows) == length(embeddings)

    # Every row has dim=768 and a 3072-byte vector (768 × 4 bytes for float32).
    Enum.each(rows, fn [_id, model, modality, dim, blob_size] ->
      assert model == "nomic-embed-text"
      assert is_binary(modality)
      assert dim == 768
      assert blob_size == 3072
    end)
  end

  test "re-inserting the same chunk_id overwrites (INSERT OR REPLACE)" do
    unique = System.unique_integer([:positive])
    signal_id = "sha256:embedder-idem-#{unique}"
    doc = ParsedDoc.new(text: "idempotent #{unique}", signal_id: signal_id)
    {:ok, tree} = Decomposer.decompose_and_store(doc)
    [chunk | _] = tree.chunks

    e1 =
      Embedding.new(
        chunk_id: chunk.id,
        tenant_id: chunk.tenant_id,
        model: "first",
        modality: :text,
        dim: 3,
        vector: [0.0, 0.0, 0.0]
      )

    e2 = %{e1 | model: "second", vector: [1.0, 1.0, 1.0], dim: 3}

    assert :ok = Store.insert_embeddings([e1])
    assert :ok = Store.insert_embeddings([e2])

    {:ok, [[model]]} =
      Store.raw_query("SELECT model FROM chunk_embeddings WHERE chunk_id = ?1", [chunk.id])

    assert model == "second"
  end
end
