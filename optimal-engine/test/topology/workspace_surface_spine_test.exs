defmodule OptimalEngine.Topology.WorkspaceSurfaceSpineTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Store

  test "migration creates Gate 1 topology and projection tables" do
    required_tables = [
      "node_types",
      "node_relationships",
      "topology_change_requests",
      "export_records",
      "projection_revisions",
      "link_health_records"
    ]

    for table <- required_tables do
      {:ok, [[^table]]} =
        Store.raw_query("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?1", [
          table
        ])
    end
  end

  test "default workspace gets standard node types" do
    {:ok, rows} =
      Store.raw_query("""
      SELECT slug
      FROM node_types
      WHERE workspace_id = 'default'
      ORDER BY slug
      """)

    slugs = Enum.map(rows, fn [slug] -> slug end)

    assert "project" in slugs
    assert "person" in slugs
    assert "product" in slugs
    assert "operation" in slugs
    assert "context" in slugs
  end
end
