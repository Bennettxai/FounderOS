defmodule Mix.Tasks.OptimalRememberTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.Store

  defp workspace_id do
    "mix-remember-#{System.unique_integer([:positive])}"
  end

  test "explicit remember uses governed Memory Core intake scoped to workspace" do
    workspace_id = workspace_id()
    content = "Decision: governed remember owns CLI memory #{System.unique_integer([:positive])}"

    Mix.Task.reenable("optimal.remember")

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Remember.run([
          content,
          "--workspace",
          workspace_id,
          "--force",
          "--actor",
          "user:test"
        ])
      end)

    assert output =~ "Optimal Memory"
    assert output =~ "Workspace: #{workspace_id}"
    assert output =~ "Source:"
    assert output =~ "Claim:"

    assert {:ok, [[source_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM source_packages
               WHERE workspace_id = ?1
                 AND raw_text = ?2
                 AND source_type = 'memory_remember'
               """,
               [workspace_id, content]
             )

    assert source_count == 1

    assert {:ok, [[claim_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM claims
               WHERE workspace_id = ?1
                 AND claim_text = ?2
                 AND lifecycle_state = 'pending'
               """,
               [workspace_id, content]
             )

    assert claim_count == 1

    assert {:ok, [[fact_count]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [
               workspace_id
             ])

    assert fact_count == 0
  end
end
