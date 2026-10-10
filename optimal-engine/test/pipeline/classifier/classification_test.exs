defmodule OptimalEngine.Pipeline.Classifier.ClassificationTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Classifier
  alias OptimalEngine.Pipeline.Classifier.Classification
  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}
  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  describe "classify_chunk/2" do
    test "returns a %Classification{} keyed to the chunk id" do
      chunk =
        Chunk.new(
          id: "test-chunk-id",
          signal_id: "test-signal",
          tenant_id: "default",
          scale: :paragraph,
          text: "## Action Items\n\nWe decided to ship on Monday.",
          modality: :text
        )

      assert {:ok, %Classification{} = c} = Classifier.classify_chunk(chunk)
      assert c.chunk_id == "test-chunk-id"
      assert c.tenant_id == "default"
      assert c.confidence >= 0.0 and c.confidence <= 1.0
    end

    test "inherits mode from chunk modality" do
      chunk = Chunk.new(id: "c1", signal_id: "s", text: "print('hi')", modality: :code)

      assert {:ok, c} = Classifier.classify_chunk(chunk)
      assert c.mode == :code
    end

    test "inherits mode from :data modality" do
      chunk = Chunk.new(id: "c1", signal_id: "s", text: "{\"k\": 1}", modality: :data)

      assert {:ok, c} = Classifier.classify_chunk(chunk)
      assert c.mode == :data
    end

    test "confidence scales with number of resolved dimensions" do
      rich =
        Chunk.new(
          id: "rich",
          signal_id: "s",
          text:
            "## Decision\n\nWe decided: use Phoenix.\n\n## Requirements\n- Must support WebSockets",
          modality: :text
        )

      sparse = Chunk.new(id: "sparse", signal_id: "s", text: "hello", modality: :text)

      {:ok, rc} = Classifier.classify_chunk(rich)
      {:ok, sc} = Classifier.classify_chunk(sparse)

      assert rc.confidence >= sc.confidence
    end

    test "sn_ratio is computed (> 0 for non-empty text)" do
      chunk = Chunk.new(id: "c1", signal_id: "s", text: "word1 word2 word3 word4")

      {:ok, c} = Classifier.classify_chunk(chunk)
      assert c.sn_ratio > 0.0
      assert c.sn_ratio <= 1.0
    end

    test "empty chunk produces sn_ratio = 0 without crashing" do
      chunk = Chunk.new(id: "c1", signal_id: "s", text: "")

      {:ok, c} = Classifier.classify_chunk(chunk)
      assert c.sn_ratio == 0.0
    end
  end

  describe "classify_tree/2" do
    test "returns one classification per chunk in the tree" do
      doc = ParsedDoc.new(text: "First paragraph.\n\nSecond paragraph of text.")
      {:ok, tree} = Decomposer.decompose(doc)

      assert {:ok, classifications} = Classifier.classify_tree(tree)
      assert length(classifications) == length(tree.chunks)

      by_chunk_id = Map.new(classifications, &{&1.chunk_id, &1})

      Enum.each(tree.chunks, fn c ->
        assert Map.has_key?(by_chunk_id, c.id),
               "no classification for chunk #{c.id}"
      end)
    end

    test "works on an empty-text doc (1 chunk, 1 classification)" do
      doc = ParsedDoc.new(text: "")
      {:ok, tree} = Decomposer.decompose(doc)
      {:ok, classifications} = Classifier.classify_tree(tree)

      assert length(tree.chunks) == length(classifications)
    end
  end
end
