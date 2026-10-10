defmodule OptimalEngine.Pipeline.Embedder.EmbeddingTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Embedder.Embedding

  test "new/1 computes dim from vector length" do
    e = Embedding.new(chunk_id: "c", model: "nomic-embed-text", vector: List.duplicate(0.0, 768))
    assert e.dim == 768
    assert length(e.vector) == 768
  end

  test "defaults tenant_id and modality" do
    e = Embedding.new(chunk_id: "c", model: "m", vector: [1.0, 2.0, 3.0])
    assert e.tenant_id == "default"
    assert e.modality == :text
    assert e.dim == 3
  end
end
