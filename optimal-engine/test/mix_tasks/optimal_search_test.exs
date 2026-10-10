defmodule Mix.Tasks.OptimalSearchTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.{Context, Store}

  defp workspace_id(label) do
    "mix-search-#{label}-#{System.unique_integer([:positive])}"
  end

  test "search forwards workspace scope to retrieval" do
    Mix.Task.run("app.start")

    workspace_a = workspace_id("a")
    workspace_b = workspace_id("b")
    token = "scopedneedle#{System.unique_integer([:positive])}"
    now = DateTime.utc_now()

    :ok =
      Store.insert_context(%Context{
        id: "ctx-a-#{System.unique_integer([:positive])}",
        uri: "optimal://nodes/a/#{token}",
        type: :memory,
        path: "a.md",
        title: "Workspace A #{token}",
        content: "#{token} belongs to workspace A only",
        l0_abstract: "#{token} A",
        l1_overview: "#{token} A overview",
        node: "inbox",
        created_at: now,
        modified_at: now,
        workspace_id: workspace_a
      })

    :ok =
      Store.insert_context(%Context{
        id: "ctx-b-#{System.unique_integer([:positive])}",
        uri: "optimal://nodes/b/#{token}",
        type: :memory,
        path: "b.md",
        title: "Workspace B #{token}",
        content: "#{token} belongs to workspace B only",
        l0_abstract: "#{token} B",
        l1_overview: "#{token} B overview",
        node: "inbox",
        created_at: now,
        modified_at: now,
        workspace_id: workspace_b
      })

    Mix.Task.reenable("optimal.search")

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Search.run([token, "--workspace", workspace_b, "--limit", "10"])
      end)

    assert output =~ "Workspace:   #{workspace_b}"
    assert output =~ "Workspace B #{token}"
    refute output =~ "Workspace A #{token}"
  end
end
