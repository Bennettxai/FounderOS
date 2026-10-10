defmodule OptimalEngine.Pipeline.DecomposerTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}
  alias OptimalEngine.Pipeline.Parser.{ParsedDoc, StructuralElement}

  describe "decompose/2 — the minimum viable doc" do
    test "empty doc produces a single :document chunk" do
      doc = ParsedDoc.new(text: "")

      assert {:ok, %ChunkTree{chunks: [root]}} = Decomposer.decompose(doc)
      assert root.scale == :document
      assert root.text == ""
    end

    test "text with no parser structure → doc + 1 section + paragraphs + chunks" do
      doc = ParsedDoc.new(text: "First paragraph.\n\nSecond paragraph with more words.")

      assert {:ok, tree} = Decomposer.decompose(doc)
      counts = ChunkTree.counts(tree)

      assert counts.document == 1
      assert counts.section == 1
      assert counts.paragraph == 2
      assert counts.chunk == 2
    end
  end

  describe "hierarchy invariants" do
    setup do
      text =
        "Section A intro.\n\nA-1.\n\nA-2.\n\nSection B intro.\n\nB-1.\n\nB-2."

      structure = [
        StructuralElement.new(:heading,
          text: "Section A",
          offset: 0,
          length: 16,
          metadata: %{level: 1}
        ),
        StructuralElement.new(:heading,
          text: "Section B",
          offset: 36,
          length: 16,
          metadata: %{level: 1}
        )
      ]

      doc = ParsedDoc.new(text: text, structure: structure)
      {:ok, tree} = Decomposer.decompose(doc)
      {:ok, tree: tree, doc: doc}
    end

    test "every non-document chunk has a parent_id", %{tree: tree} do
      tree.chunks
      |> Enum.reject(&(&1.scale == :document))
      |> Enum.each(fn c -> assert c.parent_id, "missing parent on #{inspect(c)}" end)
    end

    test "every parent_id resolves to a real chunk one scale above", %{tree: tree} do
      scale_order = %{document: 0, section: 1, paragraph: 2, chunk: 3}
      ids = Map.new(tree.chunks, &{&1.id, &1})

      tree.chunks
      |> Enum.reject(&(&1.scale == :document))
      |> Enum.each(fn c ->
        parent = Map.fetch!(ids, c.parent_id)

        assert scale_order[parent.scale] == scale_order[c.scale] - 1,
               "chunk #{c.id} at #{c.scale} has parent at #{parent.scale}"
      end)
    end

    test "exactly one :document chunk, which is the root", %{tree: tree} do
      docs = ChunkTree.at_scale(tree, :document)
      assert length(docs) == 1
      assert hd(docs).id == tree.root_chunk_id
    end

    test "sections populated from heading elements", %{tree: tree} do
      sections = ChunkTree.at_scale(tree, :section)
      assert length(sections) >= 2
    end
  end

  describe "paragraph scale" do
    test "uses parser-reported paragraph elements when present" do
      text = "Alpha paragraph text.\n\nBravo paragraph text."

      structure = [
        StructuralElement.new(:paragraph, text: "Alpha paragraph text.", offset: 0, length: 21),
        StructuralElement.new(:paragraph, text: "Bravo paragraph text.", offset: 23, length: 21)
      ]

      doc = ParsedDoc.new(text: text, structure: structure)
      {:ok, tree} = Decomposer.decompose(doc)

      paragraphs = ChunkTree.at_scale(tree, :paragraph)
      assert length(paragraphs) == 2
      assert hd(paragraphs).text == "Alpha paragraph text."
    end

    test "falls back to blank-line split when parser reported no paragraphs" do
      text = "Line one.\n\nLine two.\n\nLine three."
      doc = ParsedDoc.new(text: text)
      {:ok, tree} = Decomposer.decompose(doc)

      paragraphs = ChunkTree.at_scale(tree, :paragraph)
      assert length(paragraphs) == 3
      assert hd(paragraphs).text =~ "Line one"
    end
  end

  describe "chunk scale — sliding window" do
    test "paragraphs shorter than window become exactly one chunk" do
      doc = ParsedDoc.new(text: "short paragraph.")
      {:ok, tree} = Decomposer.decompose(doc, window_bytes: 2048, overlap_bytes: 256)
      chunks = ChunkTree.at_scale(tree, :chunk)
      assert length(chunks) == 1
      assert hd(chunks).text == "short paragraph."
    end

    test "paragraphs longer than window are split with overlap" do
      # 20 chars of repeating content → 200 with a 50-byte window, 10-byte overlap
      text = String.duplicate("ABCDEFGHIJ", 20)
      doc = ParsedDoc.new(text: text)

      {:ok, tree} = Decomposer.decompose(doc, window_bytes: 50, overlap_bytes: 10)
      chunks = ChunkTree.at_scale(tree, :chunk)

      # Step = 50 - 10 = 40. Expect windows at offsets 0, 40, 80, 120, 160 (5 windows)
      assert length(chunks) == 5

      assert Enum.all?(chunks, fn c -> c.length_bytes <= 50 end)
      # Last chunk may be shorter than window
      first = hd(chunks)
      assert first.offset_bytes == 0
      assert first.length_bytes == 50
    end

    test ":chunk-scale chunks never span a paragraph boundary" do
      text = "First.\n\nSecond paragraph has lots more text so it spans the window.\n\nThird."

      doc = ParsedDoc.new(text: text)
      {:ok, tree} = Decomposer.decompose(doc, window_bytes: 30, overlap_bytes: 5)

      tree.chunks
      |> Enum.filter(&(&1.scale == :chunk))
      |> Enum.each(fn chunk ->
        parent_para = Enum.find(tree.chunks, &(&1.id == chunk.parent_id))
        assert parent_para, "chunk #{chunk.id} has no parent paragraph"

        assert chunk.offset_bytes >= parent_para.offset_bytes

        assert chunk.offset_bytes + chunk.length_bytes <=
                 parent_para.offset_bytes + parent_para.length_bytes
      end)
    end
  end

  describe "reassembly guarantee" do
    test "root :document chunk text is byte-identical to the source" do
      text = "Mixed content.\n\nWith paragraphs.\n\nAnd edge cases: áéíóú, emoji 🙂, quotes \"\"."

      doc = ParsedDoc.new(text: text)
      {:ok, tree} = Decomposer.decompose(doc)

      assert ChunkTree.reassemble(tree) == text
    end

    test "reassembly works for a PDF-shaped doc with page structural elements" do
      text = "Page one content.\fPage two content.\fPage three content."

      structure = [
        StructuralElement.new(:page,
          text: "Page one content.",
          offset: 0,
          length: 17,
          metadata: %{number: 1}
        ),
        StructuralElement.new(:page,
          text: "Page two content.",
          offset: 18,
          length: 17,
          metadata: %{number: 2}
        ),
        StructuralElement.new(:page,
          text: "Page three content.",
          offset: 36,
          length: 19,
          metadata: %{number: 3}
        )
      ]

      doc = ParsedDoc.new(text: text, structure: structure, modality: :mixed)

      {:ok, tree} = Decomposer.decompose(doc)
      counts = ChunkTree.counts(tree)

      assert counts.document == 1
      # 3 page-derived sections
      assert counts.section >= 3
      assert ChunkTree.reassemble(tree) == text
    end
  end

  describe "determinism" do
    test "same input produces same chunk ids" do
      doc = ParsedDoc.new(text: "Deterministic content.\n\nSecond para.")
      {:ok, tree1} = Decomposer.decompose(doc)
      {:ok, tree2} = Decomposer.decompose(doc)

      ids1 = Enum.map(tree1.chunks, & &1.id) |> Enum.sort()
      ids2 = Enum.map(tree2.chunks, & &1.id) |> Enum.sort()

      assert ids1 == ids2
    end

    test "chunk ids follow the {signal_id}:{prefix}-{index} convention" do
      doc = ParsedDoc.new(text: "one\n\ntwo")
      {:ok, tree} = Decomposer.decompose(doc)

      signal_id = hd(tree.chunks).signal_id
      document_chunk = ChunkTree.at_scale(tree, :document) |> hd()

      assert document_chunk.id == "#{signal_id}:doc-0"
      assert Chunk.build_id(signal_id, :document, 0) == document_chunk.id
    end
  end

  describe "tenant_id propagation" do
    test "all chunks carry the tenant_id from opts" do
      doc = ParsedDoc.new(text: "anything")
      {:ok, tree} = Decomposer.decompose(doc, tenant_id: "sample-corp")

      assert Enum.all?(tree.chunks, &(&1.tenant_id == "sample-corp"))
    end

    test "defaults to `default` tenant" do
      doc = ParsedDoc.new(text: "anything")
      {:ok, tree} = Decomposer.decompose(doc)

      assert Enum.all?(tree.chunks, &(&1.tenant_id == "default"))
    end
  end

  describe "modality propagation" do
    test "chunks inherit modality from ParsedDoc" do
      doc = ParsedDoc.new(text: "print('hi')", modality: :code)
      {:ok, tree} = Decomposer.decompose(doc)

      assert Enum.all?(tree.chunks, &(&1.modality == :code))
    end
  end
end
