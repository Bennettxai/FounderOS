defmodule OptimalEngine.Pipeline.Embedder.BackfillJoinTest do
  @moduledoc """
  Verifies the chunk → embed → vector pipeline keeps embeddings in the
  *current* scope scheme: a `chunk_embeddings` row keyed by `chunk_id` JOINs
  back to its `contexts` row via `chunks.signal_id = contexts.id` — i.e. no
  orphaned vectors from ID-scheme drift.

  Does NOT require Ollama: the embed call is stubbed with a synthetic 768-dim
  vector so the persist + JOIN path is exercised deterministically in CI.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Embedder
  alias OptimalEngine.Pipeline.Embedder.Embedding
  alias OptimalEngine.Pipeline.Parser.ParsedDoc
  alias OptimalEngine.Store

  test "embedded chunk JOINs to its context via signal_id (no orphan)" do
    unique = System.unique_integer([:positive])
    signal_id = "backfill-join-#{unique}"
    workspace = "ws-backfill-#{unique}"

    # 1. A context exists, keyed by id == signal_id (the live scheme: chunks
    #    link to contexts via chunks.signal_id = contexts.id).
    {:ok, _} =
      Store.raw_query(
        "INSERT INTO contexts (id, title, workspace_id) VALUES (?1, ?2, ?3)",
        [signal_id, "Backfill join #{unique}", workspace]
      )

    # 2. A chunk is decomposed + stored for that signal. Skip the auto-embed on
    #    ingest so we drive the embed step ourselves (stubbed, no Ollama).
    doc = ParsedDoc.new(text: "Semantic search needs vectors #{unique}.", signal_id: signal_id)
    {:ok, tree} = Decomposer.decompose_and_store(doc, skip_embed: true)

    # 3. Embed step — stubbed synthetic vector, mapped through the production
    #    embedding_row/2 helper so workspace inheritance is exercised, then
    #    persisted via the store's embedding insert.
    chunk = hd(tree.chunks)

    emb =
      Embedding.new(
        chunk_id: chunk.id,
        tenant_id: chunk.tenant_id,
        model: "nomic-embed-text",
        modality: :text,
        dim: 768,
        vector: for(i <- 0..767, do: :math.sin(i / 100.0))
      )

    row =
      emb
      |> Embedder.embedding_row(%{chunk.id => chunk})
      |> Map.put(:workspace_id, workspace)

    assert :ok = Store.insert_embeddings([row])

    # 4. The JOIN chunk_embeddings → chunks → contexts resolves: the vector is
    #    reachable from the context, proving the keying is consistent.
    {:ok, rows} =
      Store.raw_query(
        """
        SELECT ctx.id, e.chunk_id, e.dim, LENGTH(e.vector), e.workspace_id
        FROM chunk_embeddings e
        JOIN chunks c   ON c.id = e.chunk_id
        JOIN contexts ctx ON ctx.id = c.signal_id
        WHERE ctx.id = ?1
        """,
        [signal_id]
      )

    assert [[ctx_id, chunk_id, dim, blob_size, ws]] = rows
    assert ctx_id == signal_id
    assert chunk_id == chunk.id
    assert dim == 768
    # 768 float32 components × 4 bytes — vector survived as a BLOB, not truncated.
    assert blob_size == 3072
    assert ws == workspace
  end
end
