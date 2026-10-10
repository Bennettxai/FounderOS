defmodule OptimalEngine.Connectors.Adapters.SourcesFolderTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Connectors.Adapters.SourcesFolder
  alias OptimalEngine.Connectors.PullScheduler
  alias OptimalEngine.Store

  setup do
    root = Path.join(System.tmp_dir!(), "oe-sources-#{System.unique_integer([:positive])}")
    File.mkdir_p!(Path.join(root, "meetings"))
    File.mkdir_p!(Path.join(root, "highlight"))

    File.write!(Path.join(root, "meetings/2026-04-01-call.md"), "# Pricing call\n\nEd wants $2K.")
    File.write!(Path.join(root, "highlight/note-1.md"), "# Quick note\n\nFollow up the operator.")
    File.write!(Path.join(root, "loose.md"), "# Loose note\n\nNo subfolder.")
    # Non-markdown should be ignored.
    File.write!(Path.join(root, "ignore.txt"), "not markdown")

    on_exit(fn -> File.rm_rf!(root) end)

    {:ok, root: root}
  end

  describe "sync/2" do
    test "discovers all markdown files and routes genres by subfolder", %{root: root} do
      {:ok, state} = SourcesFolder.init(%{"root" => root})

      {:ok, %{signals: signals, cursor: cursor, payloads: payloads}} =
        SourcesFolder.sync(state, nil)

      assert length(signals) == 3
      assert length(payloads) == 3

      genres = Enum.into(payloads, %{}, fn p -> {p["rel"], p["genre"]} end)
      assert genres["meetings/2026-04-01-call.md"] == "transcript"
      assert genres["highlight/note-1.md"] == "transcript"
      assert genres["loose.md"] == "note"

      # Cursor records every ingested file (the dedupe ledger).
      assert {:ok, %{"ingested" => ingested}} = Jason.decode(cursor)
      assert "meetings/2026-04-01-call.md" in ingested
      assert "loose.md" in ingested
    end

    test "is idempotent — re-running with the prior cursor yields no new files", %{root: root} do
      {:ok, state} = SourcesFolder.init(%{"root" => root})
      {:ok, %{cursor: cursor}} = SourcesFolder.sync(state, nil)

      {:ok, %{signals: signals2, cursor: cursor2, payloads: payloads2}} =
        SourcesFolder.sync(state, cursor)

      assert signals2 == []
      assert payloads2 == []
      # Cursor is stable across the idempotent re-run.
      assert Jason.decode!(cursor2) == Jason.decode!(cursor)
    end

    test "picks up only newly-added files on a second run", %{root: root} do
      {:ok, state} = SourcesFolder.init(%{"root" => root})
      {:ok, %{cursor: cursor}} = SourcesFolder.sync(state, nil)

      File.write!(Path.join(root, "meetings/2026-04-02-new.md"), "# New\n\nFresh.")

      {:ok, %{signals: signals, payloads: payloads}} = SourcesFolder.sync(state, cursor)

      assert length(signals) == 1
      assert [%{"rel" => "meetings/2026-04-02-new.md"}] = payloads
    end

    test "missing root yields a structured error" do
      {:ok, state} = SourcesFolder.init(%{"root" => "/no/such/dir/oe-#{System.unique_integer()}"})
      assert {:error, {:root_not_found, _}} = SourcesFolder.sync(state, nil)
    end
  end

  describe "PullScheduler.run_once/1 end-to-end" do
    test "ingests N files into the store and is idempotent on re-run", %{root: root} do
      id = "pull-test-#{System.unique_integer([:positive])}"

      opts = [
        connectors: [%{kind: :sources_folder, id: id, config: %{"root" => root}}],
        tenant_id: "default"
      ]

      before = signal_count()

      {:ok, summary} = PullScheduler.run_once(opts)

      assert summary.ingested_count == 3
      assert summary.error_count == 0
      assert [%{kind: :sources_folder, status: :ok, signals: 3}] = summary.connector_results

      assert signal_count() >= before + 3

      # Re-run: cursor (persisted by the Runner) skips already-ingested files.
      {:ok, summary2} = PullScheduler.run_once(opts)
      assert summary2.ingested_count == 0
    end
  end

  defp signal_count do
    case Store.raw_query("SELECT COUNT(*) FROM contexts", []) do
      {:ok, [[n]]} -> n
      _ -> 0
    end
  end
end
