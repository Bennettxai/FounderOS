defmodule OptimalEngine.Pipeline.Decomposer.StoreIntegrationTest do
  @moduledoc """
  Round-trip: decompose → Store.insert_chunks → SELECT FROM chunks.
  Verifies Phase 3 persistence against the Phase 1 schema.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Parser.ParsedDoc
  alias OptimalEngine.Store

  test "decompose_and_store/2 persists every chunk to the chunks table" do
    unique = System.unique_integer([:positive])
    text = "Integration test #{unique}.\n\nSecond paragraph #{unique}."
    signal_id = "sha256:integration-#{unique}"
    doc = ParsedDoc.new(text: text, signal_id: signal_id)

    assert {:ok, tree} = Decomposer.decompose_and_store(doc)

    # Expect: 1 doc + 1 section + 2 paragraphs + 2 chunks = 6 total
    assert length(tree.chunks) >= 6

    {:ok, rows} =
      Store.raw_query(
        "SELECT id, scale, parent_id, text FROM chunks WHERE signal_id = ?1 ORDER BY scale, offset_bytes",
        [signal_id]
      )

    assert length(rows) == length(tree.chunks)

    # Root chunk round-trips byte-identical text
    [{^signal_id, "document-scale-text", _parent, stored_text}] =
      rows
      |> Enum.filter(fn [_, scale, _, _] -> scale == "document" end)
      |> Enum.map(fn [id, _, parent, stored_text] ->
        {signal_id_from_id(id), "document-scale-text", parent, stored_text}
      end)

    assert stored_text == text
  end

  test "re-inserting the same decomposition is idempotent (INSERT OR REPLACE)" do
    unique = System.unique_integer([:positive])
    text = "Idempotence test #{unique}."
    signal_id = "sha256:idem-#{unique}"
    doc = ParsedDoc.new(text: text, signal_id: signal_id)

    assert {:ok, tree1} = Decomposer.decompose_and_store(doc)
    assert {:ok, tree2} = Decomposer.decompose_and_store(doc)

    assert length(tree1.chunks) == length(tree2.chunks)

    {:ok, [[count]]} =
      Store.raw_query("SELECT COUNT(*) FROM chunks WHERE signal_id = ?1", [signal_id])

    assert count == length(tree1.chunks)
  end

  test "persists workspace scope on chunks" do
    unique = System.unique_integer([:positive])
    signal_id = "sha256:workspace-#{unique}"
    workspace_id = "workspace-#{unique}"
    doc = ParsedDoc.new(text: "Workspace-scoped content", signal_id: signal_id)

    assert {:ok, tree} =
             Decomposer.decompose_and_store(doc,
               workspace_id: workspace_id,
               skip_embed: true
             )

    assert Enum.all?(tree.chunks, &(&1.workspace_id == workspace_id))

    assert {:ok, [[^workspace_id]]} =
             Store.raw_query(
               "SELECT DISTINCT workspace_id FROM chunks WHERE signal_id = ?1",
               [signal_id]
             )
  end

  defp signal_id_from_id(id), do: id |> String.split(":doc-") |> hd()
end
