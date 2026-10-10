defmodule OptimalEngine.Pipeline.Decomposer.ChunkTreeTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}

  setup do
    doc = Chunk.new(id: "s:doc-0", scale: :document, signal_id: "s", text: "full document text")

    sec =
      Chunk.new(
        id: "s:sec-0",
        scale: :section,
        parent_id: doc.id,
        signal_id: "s",
        text: "section"
      )

    par =
      Chunk.new(
        id: "s:par-0",
        scale: :paragraph,
        parent_id: sec.id,
        signal_id: "s",
        text: "paragraph"
      )

    chk = Chunk.new(id: "s:chk-0", scale: :chunk, parent_id: par.id, signal_id: "s", text: "chunk")

    tree = %ChunkTree{root_chunk_id: doc.id, chunks: [doc, sec, par, chk]}

    {:ok, tree: tree, doc: doc, sec: sec, par: par, chk: chk}
  end

  test "at_scale/2 filters by scale", %{tree: tree} do
    assert [%{scale: :document}] = ChunkTree.at_scale(tree, :document)
    assert [%{scale: :section}] = ChunkTree.at_scale(tree, :section)
    assert [%{scale: :paragraph}] = ChunkTree.at_scale(tree, :paragraph)
    assert [%{scale: :chunk}] = ChunkTree.at_scale(tree, :chunk)
  end

  test "children_of/2 walks one level down", %{tree: tree, doc: doc, sec: sec, par: par, chk: chk} do
    assert [^sec] = ChunkTree.children_of(tree, doc.id)
    assert [^par] = ChunkTree.children_of(tree, sec.id)
    assert [^chk] = ChunkTree.children_of(tree, par.id)
    assert [] = ChunkTree.children_of(tree, chk.id)
  end

  test "parent_of/2 walks one level up", %{tree: tree, doc: doc, sec: sec, par: par, chk: chk} do
    assert ^par = ChunkTree.parent_of(tree, chk.id)
    assert ^sec = ChunkTree.parent_of(tree, par.id)
    assert ^doc = ChunkTree.parent_of(tree, sec.id)
    assert nil == ChunkTree.parent_of(tree, doc.id)
  end

  test "counts/1 returns per-scale counts", %{tree: tree} do
    assert %{document: 1, section: 1, paragraph: 1, chunk: 1} = ChunkTree.counts(tree)
  end

  test "reassemble/1 returns the root document chunk text verbatim", %{tree: tree} do
    assert "full document text" = ChunkTree.reassemble(tree)
  end
end
