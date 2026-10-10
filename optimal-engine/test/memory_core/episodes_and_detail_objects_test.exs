defmodule OptimalEngine.MemoryCore.EpisodesAndDetailObjectsTest do
  @moduledoc """
  Integration tests asserting that:

  1. `Store.insert_memory_detail_object/1` persists rows in
     `memory_detail_objects` and that count grows when called with a
     non-fixture parent_object_id.

  2. `Store.insert_episode/1` persists a row in `episodes`.

  3. A transcript intake via `Pipeline.Intake.process/2` with
     `genre: "transcript"` creates an episode row in the live database.

  4. Entities extracted from the intake signal produce detail-object rows
     in `memory_detail_objects` with the source_package as parent.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.Episode
  alias OptimalEngine.MemoryCore.MemoryDetailObject
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Store

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp unique_workspace, do: "epi-test-ws-#{System.unique_integer([:positive])}"

  defp count_detail_objects(workspace_id, parent_object_id) do
    {:ok, [[n]]} =
      Store.raw_query(
        "SELECT COUNT(*) FROM memory_detail_objects WHERE workspace_id = ?1 AND parent_object_id = ?2",
        [workspace_id, parent_object_id]
      )

    n
  end

  defp count_episodes(workspace_id) do
    {:ok, [[n]]} =
      Store.raw_query(
        "SELECT COUNT(*) FROM episodes WHERE workspace_id = ?1",
        [workspace_id]
      )

    n
  end

  defp count_episodes_by_kind(workspace_id, kind) do
    {:ok, [[n]]} =
      Store.raw_query(
        "SELECT COUNT(*) FROM episodes WHERE workspace_id = ?1 AND kind = ?2",
        [workspace_id, kind]
      )

    n
  end

  # ---------------------------------------------------------------------------
  # Unit: MemoryDetailObject
  # ---------------------------------------------------------------------------

  describe "Store.insert_memory_detail_object/1" do
    test "count grows when inserting with a non-fixture parent_object_id" do
      ws = unique_workspace()
      parent_id = "test-parent-#{System.unique_integer([:positive])}"

      assert count_detail_objects(ws, parent_id) == 0

      attrs = %{
        tenant_id: "default",
        workspace_id: ws,
        parent_object_type: "source_package",
        parent_object_id: parent_id,
        detail_type: "step",
        detail_order: 0,
        detail_text: "parsed entity: Alice",
        action_class: "entity_extraction"
      }

      assert :ok = MemoryCoreStore.insert_memory_detail_object(attrs)
      assert count_detail_objects(ws, parent_id) == 1

      attrs2 = %{attrs | detail_order: 1, detail_text: "parsed entity: Bob"}
      assert :ok = MemoryCoreStore.insert_memory_detail_object(attrs2)
      assert count_detail_objects(ws, parent_id) == 2
    end

    test "returns error when parent_object_id is missing" do
      attrs = %{
        parent_object_type: "source_package",
        detail_text: "some step"
      }

      assert {:error, :parent_object_id_required} = MemoryDetailObject.new(attrs)
    end
  end

  # ---------------------------------------------------------------------------
  # Unit: Episode
  # ---------------------------------------------------------------------------

  describe "Store.insert_episode/1" do
    test "persists a new episode row" do
      ws = unique_workspace()

      assert count_episodes(ws) == 0

      attrs = %{
        tenant_id: "default",
        workspace_id: ws,
        kind: "transcript",
        occurred_at: DateTime.utc_now() |> DateTime.to_iso8601(),
        summary: "Alice and Bob discussed the pricing model",
        provenance: %{source_package_id: "sp_abc123"},
        metadata: %{genre: "transcript"}
      }

      assert :ok = MemoryCoreStore.insert_episode(attrs)
      assert count_episodes(ws) == 1
    end

    test "insert is idempotent via OR IGNORE on same id" do
      ws = unique_workspace()

      {:ok, ep} =
        Episode.new(%{
          workspace_id: ws,
          kind: "meeting",
          summary: "Weekly standup",
          provenance: %{}
        })

      assert :ok = MemoryCoreStore.insert_episode(ep)
      assert :ok = MemoryCoreStore.insert_episode(ep)
      assert count_episodes(ws) == 1
    end

    test "Episode.new/1 returns error when kind is missing" do
      assert {:error, :kind_required} = Episode.new(%{summary: "no kind here"})
    end

    test "Episode.new/1 returns error when summary is missing" do
      assert {:error, :summary_required} = Episode.new(%{kind: "event"})
    end

    test "Episode.new/1 generates epi_ prefixed id" do
      {:ok, ep} = Episode.new(%{kind: "transcript", summary: "test episode"})
      assert String.starts_with?(ep.id, "epi_")
    end
  end

  # ---------------------------------------------------------------------------
  # Integration: transcript intake creates episode + detail objects
  # ---------------------------------------------------------------------------

  describe "Pipeline.Intake.process/2 with genre: transcript" do
    setup do
      tmp_dir =
        System.tmp_dir!()
        |> Path.join("optimal_epi_intake_test_#{:rand.uniform(99_999)}")

      File.mkdir_p!(tmp_dir)

      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      for folder <- ~w[inbox transcript] do
        File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
      end

      on_exit(fn ->
        Application.put_env(:optimal_engine, :root_path, original_root)
        File.rm_rf!(tmp_dir)
      end)

      ws = unique_workspace()
      %{ws: ws}
    end

    test "creates an episode row in episodes table on transcript intake", %{ws: ws} do
      assert count_episodes_by_kind(ws, "transcript") == 0

      raw_text = """
      Meeting with Alice and Bob on 2026-06-15.
      Alice: We need to finalize the Q3 pricing.
      Bob: Agreed, let's schedule a follow-up.
      """

      result =
        Intake.process(raw_text,
          genre: "transcript",
          title: "Q3 Pricing Discussion",
          workspace_id: ws,
          extract_claims: false
        )

      assert {:ok, _} = result
      assert count_episodes_by_kind(ws, "transcript") >= 1
    end

    test "detail object count grows when entities are extracted", %{ws: ws} do
      raw_text = "Call with Charlie about contract renewal. Charlie confirmed the terms."

      {:ok, _result} =
        Intake.process(raw_text,
          genre: "transcript",
          title: "Contract renewal call",
          entities: ["Charlie"],
          workspace_id: ws,
          extract_claims: false
        )

      # At least one detail object should exist for this workspace from the
      # entity extraction step.
      {:ok, [[total]]} =
        Store.raw_query(
          "SELECT COUNT(*) FROM memory_detail_objects WHERE workspace_id = ?1",
          [ws]
        )

      assert total >= 1
    end
  end
end
