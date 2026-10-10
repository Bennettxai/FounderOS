defmodule OptimalEngine.Store.WorkspaceIsolationMigrationTest do
  @moduledoc """
  T3 workspace isolation — schema-level guarantees:

    * migration 035 adds workspace_id to connectors + connector_runs
    * the legacy node renames are recorded as a one-time migration instead of
      running on every Store init
    * entities written through insert_context carry the context's workspace_id
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Context
  alias OptimalEngine.Store

  defp column_names(table) do
    {:ok, rows} = Store.raw_query("PRAGMA table_info(#{table})", [])
    Enum.map(rows, fn [_cid, name | _] -> name end)
  end

  test "connectors and connector_runs have a workspace_id column" do
    assert "workspace_id" in column_names("connectors")
    assert "workspace_id" in column_names("connector_runs")
  end

  test "migration 035 is recorded in schema_migrations" do
    {:ok, rows} =
      Store.raw_query("SELECT version FROM schema_migrations WHERE version = 35", [])

    assert rows == [[35]]
  end

  test "legacy node renames no longer rewrite non-default workspaces" do
    # The one-time rename migration is scoped to workspace 'default'. A
    # context using a legacy node name in another workspace must keep it —
    # the old Store-init rewrite would have renamed it globally.
    suffix = System.unique_integer([:positive])
    ws = "rename-ws-#{suffix}"
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    ctx = %Context{
      id: "rename#{suffix}",
      uri: "optimal://nodes/01-inbox/rename#{suffix}.md",
      type: :signal,
      path: "/tmp/rename#{suffix}.md",
      title: "legacy node name",
      content: "x",
      l0_abstract: "SIGNAL | note | legacy",
      l1_overview: "o",
      signal: nil,
      node: "01-inbox",
      sn_ratio: 0.5,
      entities: [],
      created_at: now,
      modified_at: now,
      routed_to: [],
      workspace_id: ws,
      metadata: %{}
    }

    :ok = Store.insert_context(ctx)

    {:ok, [[node]]} =
      Store.raw_query("SELECT node FROM contexts WHERE id = ?1", [ctx.id])

    assert node == "01-inbox"
  end

  test "entities written via insert_context carry the context's workspace_id" do
    suffix = System.unique_integer([:positive])
    ws = "entity-ws-#{suffix}"
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    ctx = %Context{
      id: "entws#{suffix}",
      uri: "optimal://nodes/n/entws#{suffix}.md",
      type: :signal,
      path: "/tmp/entws#{suffix}.md",
      title: "entity ws",
      content: "x",
      l0_abstract: "SIGNAL | note | entity ws",
      l1_overview: "o",
      signal: nil,
      node: "entity-node-#{suffix}",
      sn_ratio: 0.5,
      entities: ["EntWs#{suffix}"],
      created_at: now,
      modified_at: now,
      routed_to: [],
      workspace_id: ws,
      metadata: %{}
    }

    :ok = Store.insert_context(ctx)

    {:ok, [[entity_ws]]} =
      Store.raw_query("SELECT workspace_id FROM entities WHERE context_id = ?1", [ctx.id])

    assert entity_ws == ws
  end
end
