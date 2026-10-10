defmodule OptimalEngine.BatchTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Batch
  alias OptimalEngine.Memory
  alias OptimalEngine.Pipeline.Intake

  # Use a unique workspace per test to avoid cross-test pollution.
  defp ws, do: "batch-test-#{:erlang.unique_integer([:positive])}"

  # Node-folder setup required by Intake.process/2.
  defp setup_node_folders(tmp_dir) do
    for folder <- ~w[
      person-operator product-customer-portal entity-company project-platform-launch entity-company
      operation-delivery team learning-research-library
      inbox team operation-revenue operation-delivery
    ] do
      File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
    end
  end

  setup do
    tmp_dir = System.tmp_dir!() |> Path.join("batch_test_#{:rand.uniform(99_999)}")
    File.mkdir_p!(tmp_dir)
    setup_node_folders(tmp_dir)

    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)

    on_exit(fn ->
      Application.put_env(:optimal_engine, :root_path, original_root)
      File.rm_rf!(tmp_dir)
    end)

    :ok
  end

  # ---------------------------------------------------------------------------
  # import_signals/2
  # ---------------------------------------------------------------------------

  describe "import_signals/2" do
    test "imports 3 distinct signals successfully" do
      workspace_id = ws()

      items = [
        %{"content" => "First signal about Q4 planning", "genre" => "note", "node" => "operator"},
        %{
          "content" => "Second signal about product roadmap",
          "genre" => "note",
          "node" => "operator"
        },
        %{"content" => "Third signal about team velocity", "genre" => "note", "node" => "team"}
      ]

      assert {:ok, summary} = Batch.import_signals(items, workspace_id: workspace_id)
      assert summary.imported == 3
      assert summary.skipped == 0
      assert summary.errors == 0
    end

    test "counts items missing content as errors, does not abort" do
      workspace_id = ws()

      items = [
        %{"content" => "Valid signal content here", "node" => "operator"},
        %{"title" => "No content key at all"},
        %{"content" => "Another valid signal", "node" => "operator"}
      ]

      assert {:ok, summary} = Batch.import_signals(items, workspace_id: workspace_id)
      assert summary.imported == 2
      assert summary.errors == 1
      assert summary.skipped == 0
    end

    test "partial batch: 2 imported, 1 error on bad item" do
      workspace_id = ws()

      items = [
        %{"content" => "Good signal alpha", "node" => "operator"},
        %{"content" => ""},
        %{"content" => "Good signal beta", "node" => "operator"}
      ]

      assert {:ok, summary} = Batch.import_signals(items, workspace_id: workspace_id)
      assert summary.imported == 2
      assert summary.errors == 1
      assert summary.skipped == 0
    end

    test "empty list returns zero counts" do
      assert {:ok, summary} = Batch.import_signals([], workspace_id: ws())
      assert summary == %{imported: 0, skipped: 0, errors: 0}
    end

    test "accepts both string and atom keys in signal maps" do
      workspace_id = ws()

      items = [
        %{content: "Atom-keyed signal content", genre: "note", node: "operator"}
      ]

      assert {:ok, summary} = Batch.import_signals(items, workspace_id: workspace_id)
      assert summary.imported == 1
    end
  end

  # ---------------------------------------------------------------------------
  # import_memories/2
  # ---------------------------------------------------------------------------

  describe "import_memories/2" do
    test "imports 3 distinct memories successfully" do
      workspace_id = ws()

      items = [
        %{"content" => "Team velocity is measured weekly"},
        %{"content" => "Pricing decisions require VP approval"},
        %{"content" => "All deploys happen on Friday afternoons"}
      ]

      assert {:ok, summary} = Batch.import_memories(items, workspace_id: workspace_id)
      assert summary.imported == 3
      assert summary.skipped == 0
      assert summary.errors == 0
    end

    test "duplicate memory (same content, same workspace) is skipped" do
      workspace_id = ws()
      content = "Deployment policy: no deploys on Fridays"

      # First import
      {:ok, _} = Memory.create(%{content: content, workspace_id: workspace_id})

      items = [%{"content" => content}]

      assert {:ok, summary} = Batch.import_memories(items, workspace_id: workspace_id)
      assert summary.skipped == 1
      assert summary.imported == 0
      assert summary.errors == 0
    end

    test "missing content is counted as error, batch continues" do
      workspace_id = ws()

      items = [
        %{"content" => "Valid memory entry"},
        %{"is_static" => true},
        %{"content" => "Another valid entry"}
      ]

      assert {:ok, summary} = Batch.import_memories(items, workspace_id: workspace_id)
      assert summary.imported == 2
      assert summary.errors == 1
      assert summary.skipped == 0
    end

    test "is_static flag is forwarded correctly" do
      workspace_id = ws()

      items = [
        %{"content" => "Static fact: the sky is blue", "is_static" => true}
      ]

      assert {:ok, summary} = Batch.import_memories(items, workspace_id: workspace_id)
      assert summary.imported == 1

      # Verify the memory was stored with is_static=true
      mems = Memory.list(workspace_id: workspace_id)
      assert Enum.any?(mems, & &1.is_static)
    end

    test "empty list returns zero counts" do
      assert {:ok, summary} = Batch.import_memories([], workspace_id: ws())
      assert summary == %{imported: 0, skipped: 0, errors: 0}
    end
  end

  # ---------------------------------------------------------------------------
  # export_signals/1
  # ---------------------------------------------------------------------------

  describe "export_signals/1" do
    test "returns signals for the workspace" do
      workspace_id = ws()

      # Ingest two signals into this workspace
      {:ok, _} =
        Intake.process("Signal export test content A", workspace_id: workspace_id, node: "operator")

      {:ok, _} =
        Intake.process("Signal export test content B", workspace_id: workspace_id, node: "operator")

      assert {:ok, signals} = Batch.export_signals(workspace_id: workspace_id)
      assert length(signals) >= 2

      Enum.each(signals, fn s ->
        assert Map.has_key?(s, :id)
        assert Map.has_key?(s, :content)
        assert Map.has_key?(s, :workspace_id)
        assert s.workspace_id == workspace_id
      end)
    end

    test "does not return signals from other workspaces" do
      ws_a = ws()
      ws_b = ws()

      {:ok, _} = Intake.process("Workspace A signal", workspace_id: ws_a, node: "operator")

      assert {:ok, signals_b} = Batch.export_signals(workspace_id: ws_b)
      refute Enum.any?(signals_b, &(&1.workspace_id == ws_a))
    end

    test "empty workspace returns empty list" do
      assert {:ok, signals} =
               Batch.export_signals(workspace_id: "batch-empty-ws-#{:erlang.unique_integer()}")

      assert signals == []
    end
  end

  # ---------------------------------------------------------------------------
  # export_memories/1
  # ---------------------------------------------------------------------------

  describe "export_memories/1" do
    test "returns only active (non-forgotten) latest memories" do
      workspace_id = ws()

      {:ok, mem_keep} = Memory.create(%{content: "Keep this memory", workspace_id: workspace_id})

      {:ok, mem_forget} =
        Memory.create(%{content: "Forget this memory", workspace_id: workspace_id})

      # Soft-forget the second memory
      :ok = Memory.forget(mem_forget.id, reason: "test cleanup")

      assert {:ok, memories} = Batch.export_memories(workspace_id: workspace_id)

      ids = Enum.map(memories, & &1.id)
      assert mem_keep.id in ids
      refute mem_forget.id in ids
    end

    test "returns correct count matching the workspace" do
      workspace_id = ws()

      {:ok, _} = Memory.create(%{content: "Memory one for export", workspace_id: workspace_id})
      {:ok, _} = Memory.create(%{content: "Memory two for export", workspace_id: workspace_id})
      {:ok, _} = Memory.create(%{content: "Memory three for export", workspace_id: workspace_id})

      assert {:ok, memories} = Batch.export_memories(workspace_id: workspace_id)
      assert length(memories) == 3
    end

    test "exported memories contain required fields" do
      workspace_id = ws()
      {:ok, _} = Memory.create(%{content: "Structured memory", workspace_id: workspace_id})

      assert {:ok, [mem | _]} = Batch.export_memories(workspace_id: workspace_id)
      assert Map.has_key?(mem, :id)
      assert Map.has_key?(mem, :content)
      assert Map.has_key?(mem, :workspace_id)
      assert Map.has_key?(mem, :is_static)
      assert Map.has_key?(mem, :is_forgotten)
      assert mem.is_forgotten == false
    end

    test "empty workspace returns empty list" do
      assert {:ok, mems} =
               Batch.export_memories(workspace_id: "batch-no-mems-#{:erlang.unique_integer()}")

      assert mems == []
    end
  end

  # ---------------------------------------------------------------------------
  # export_workspace/2
  # ---------------------------------------------------------------------------

  describe "export_workspace/2" do
    test "snapshot contains all 4 required sections" do
      workspace_id = ws()

      # Seed some data
      {:ok, _} =
        Intake.process("Workspace snapshot signal", workspace_id: workspace_id, node: "operator")

      {:ok, _} = Memory.create(%{content: "Workspace snapshot memory", workspace_id: workspace_id})

      assert {:ok, snapshot} = Batch.export_workspace(workspace_id)

      assert Map.has_key?(snapshot, :signals), "snapshot missing :signals"
      assert Map.has_key?(snapshot, :memories), "snapshot missing :memories"
      assert Map.has_key?(snapshot, :wiki), "snapshot missing :wiki"
      assert Map.has_key?(snapshot, :config), "snapshot missing :config"
    end

    test "snapshot metadata fields are present" do
      workspace_id = ws()

      assert {:ok, snapshot} = Batch.export_workspace(workspace_id)

      assert snapshot.workspace_id == workspace_id
      assert is_binary(snapshot.exported_at)
      assert is_list(snapshot.signals)
      assert is_list(snapshot.memories)
      assert is_list(snapshot.wiki)
    end

    test "signals and memories in snapshot match standalone export counts" do
      workspace_id = ws()

      {:ok, _} = Intake.process("Count check signal", workspace_id: workspace_id, node: "operator")
      {:ok, _} = Memory.create(%{content: "Count check memory", workspace_id: workspace_id})

      {:ok, signals} = Batch.export_signals(workspace_id: workspace_id)
      {:ok, memories} = Batch.export_memories(workspace_id: workspace_id)
      {:ok, snapshot} = Batch.export_workspace(workspace_id)

      assert length(snapshot.signals) == length(signals)
      assert length(snapshot.memories) == length(memories)
    end
  end
end
