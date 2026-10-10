defmodule OptimalEngine.Store.FTSIntegrityTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Context
  alias OptimalEngine.Store

  test "context updates and deletes do not leave duplicate FTS rows" do
    suffix = System.unique_integer([:positive])
    id = "fts-integrity-#{suffix}"
    context = context(id, "Original title")

    assert :ok = Store.insert_context(context)
    assert :ok = Store.insert_context(%{context | title: "Updated title"})
    assert :ok = Store.insert_context(%{context | title: "Final title"})

    assert {:ok, [[1, "Final title"]]} =
             Store.raw_query(
               "SELECT COUNT(*), MAX(title) FROM contexts_fts WHERE id = ?1",
               [id]
             )

    assert :ok = Store.delete_context(id)
    assert {:ok, [[0]]} = Store.raw_query("SELECT COUNT(*) FROM contexts_fts WHERE id = ?1", [id])
  end

  defp context(id, title) do
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    %Context{
      id: id,
      uri: "optimal://inbox/#{id}.md",
      type: :signal,
      path: "/tmp/#{id}.md",
      title: title,
      content: title,
      l0_abstract: title,
      l1_overview: title,
      node: "inbox",
      sn_ratio: 0.8,
      entities: [],
      created_at: now,
      modified_at: now,
      routed_to: [],
      workspace_id: "default:knowledge-intake",
      metadata: %{}
    }
  end
end
