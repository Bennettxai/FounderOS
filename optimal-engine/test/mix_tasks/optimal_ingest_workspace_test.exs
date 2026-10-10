defmodule Mix.Tasks.Optimal.IngestWorkspaceTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Store

  test "ingests duplicate node slugs, entities, and wiki pages within workspace scope" do
    suffix = System.unique_integer([:positive])
    root = Path.join(System.tmp_dir!(), "ingest_workspace_scope_#{suffix}")
    workspace_a = "ingest-a-#{suffix}"
    workspace_b = "ingest-b-#{suffix}"

    on_exit(fn -> File.rm_rf(root) end)

    for {workspace, context} <- [
          {workspace_a, "---\nname: Shared Person\nkind: person\n---\n"},
          {workspace_b,
           "# Shared Person\n\n## Identity\n\n- Node type: `person`\n- Status: `active`\n"}
        ] do
      workspace_root = Path.join(root, workspace)
      signal_dir = Path.join([workspace_root, "nodes", "shared-person", "signals"])
      wiki_dir = Path.join(workspace_root, ".wiki")
      File.mkdir_p!(signal_dir)
      File.mkdir_p!(wiki_dir)

      File.write!(
        Path.join([workspace_root, "nodes", "shared-person", "context.md"]),
        context
      )

      File.write!(
        Path.join(signal_dir, "2026-07-12-update.md"),
        "---\ntitle: Scoped update\nentities:\n  - Shared Company\n---\n\nScoped signal.\n"
      )

      File.write!(
        Path.join(wiki_dir, "overview.md"),
        "---\nslug: overview-#{workspace}\n---\n\nScoped wiki.\n"
      )

      Mix.Task.reenable("optimal.ingest_workspace")

      Mix.Tasks.Optimal.IngestWorkspace.run([
        workspace_root,
        "--workspace",
        workspace
      ])
    end

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM nodes WHERE slug = 'shared-person' AND workspace_id IN (?1, ?2)",
               [workspace_a, workspace_b]
             )

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM nodes WHERE slug = 'shared-person' AND kind = 'person' AND workspace_id IN (?1, ?2)",
               [workspace_a, workspace_b]
             )

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM entities WHERE name = 'Shared Company' AND workspace_id IN (?1, ?2)",
               [workspace_a, workspace_b]
             )

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM wiki_pages WHERE workspace_id IN (?1, ?2) AND slug LIKE 'overview-%'",
               [workspace_a, workspace_b]
             )
  end
end
