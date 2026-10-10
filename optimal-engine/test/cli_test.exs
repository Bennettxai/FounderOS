defmodule OptimalEngine.CLITest do
  use ExUnit.Case, async: true

  alias OptimalEngine.CLI

  test "exposes first-run and backend-readiness commands" do
    names = CLI.command_names()

    for command <- [
          "setup",
          "initiate",
          "topology",
          "auth",
          "migrate",
          "reality-check",
          "context.refresh-stale",
          "eval.run",
          "wiki",
          "connector",
          "claims",
          "facts",
          "doctor",
          "boot",
          "find",
          "capture",
          "aware",
          "note",
          "lesson",
          "decision",
          "task",
          "close"
        ] do
      assert command in names
    end
  end

  test "resolves compatibility aliases" do
    assert {:ok, "reality-check", Mix.Tasks.Optimal.RealityCheck} =
             CLI.resolve_subcommand("reality_check")

    assert {:ok, "context.refresh-stale", Mix.Tasks.Optimal.Context.RefreshStale} =
             CLI.resolve_subcommand("context.refresh_stale")

    assert {:ok, "ingest-workspace", Mix.Tasks.Optimal.IngestWorkspace} =
             CLI.resolve_subcommand("ingest_workspace")

    assert {:ok, "search", Mix.Tasks.Optimal.Search} =
             CLI.resolve_subcommand("find")

    assert {:ok, "capture", Mix.Tasks.Optimal.Ingest} =
             CLI.resolve_subcommand("signal")
  end

  test "help text includes local setup and auth guidance" do
    help = CLI.help_text()

    assert help =~ "optimal doctor"
    assert help =~ "optimal boot"
    assert help =~ "optimal find"
    assert help =~ "optimal capture"
    assert help =~ "optimal aware"
    assert help =~ "optimal close"
    assert help =~ "optimal setup my-workspace"
    assert help =~ "optimal initiate my-workspace"
    assert help =~ "auth"
    assert help =~ "reality-check"
  end
end
