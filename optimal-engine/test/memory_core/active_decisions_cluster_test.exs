defmodule OptimalEngine.MemoryCore.ActiveDecisionsClusterTest do
  @moduledoc """
  End-to-end proof that the Active-pools / observations / decisions / memory
  graph cluster is wired and every store goes from 0 -> >=1 row via a REAL
  flow (public API), with correct shape + provenance.

  Stores exercised:
    * active_memory_pools     — ActiveMemoryPool.open + publish_observation + load_context_package
    * observations            — Insight.Remember.remember/2
    * decisions               — Insight.Remember.record_decision/2 (newly wired caller)
    * memory_relations        — Memory.Versioned.derive/2 (writes the typed relation)
    * memory_detail_objects   — MemoryCore.Store.insert_memory_detail_object/1 (newly wired)
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Insight.Remember
  alias OptimalEngine.Memory.Versioned, as: Memory
  alias OptimalEngine.MemoryCore.ActiveMemoryPool
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Store

  defp ws, do: "test-ac-#{:erlang.unique_integer([:positive])}"

  defp count(sql, params) do
    {:ok, [[n]]} = Store.raw_query(sql, params)
    n
  end

  describe "active_memory_pools" do
    test "opening a pool and publishing an observation populates the pool row" do
      workspace_id = ws()

      before =
        count("SELECT COUNT(*) FROM active_memory_pools WHERE workspace_id = ?1", [workspace_id])

      assert before == 0

      {:ok, pool} =
        ActiveMemoryPool.open(
          workspace_id: workspace_id,
          task_type: "launch_review",
          subject_anchor: "project_launch"
        )

      # Real flow: load a context package + publish an observation into the pool.
      {:ok, _} = ActiveMemoryPool.load_context_package(pool.id, %{id: "ctx-#{workspace_id}"})

      {:ok, obs} =
        ActiveMemoryPool.publish_observation(pool.id, "Pool saw a launch review event.",
          actor_id: "agent:loop",
          subject_anchor: "project_launch",
          action_class: "observed"
        )

      assert obs.pending_claim.status == "pending"

      assert count("SELECT COUNT(*) FROM active_memory_pools WHERE workspace_id = ?1", [
               workspace_id
             ]) == 1

      # Shape + provenance: the persisted row carries the pool's scope, links,
      # and lifecycle state from the real flow.
      {:ok, [[lifecycle, pkg_links, candidate_links]]} =
        Store.raw_query(
          "SELECT lifecycle_state, context_package_links, promotion_candidate_links FROM active_memory_pools WHERE id = ?1",
          [pool.id]
        )

      assert lifecycle == "open"

      assert Jason.decode!(pkg_links) == [
               %{"type" => "context_package", "id" => "ctx-#{workspace_id}"}
             ]

      assert [%{"type" => "claim", "id" => _}] = Jason.decode!(candidate_links)
    end
  end

  describe "observations" do
    test "Remember.remember/2 stores a workspace-scoped observation" do
      workspace_id = ws()

      assert count("SELECT COUNT(*) FROM observations WHERE workspace_id = ?1", [workspace_id]) ==
               0

      {:ok, result} =
        Remember.remember("Always check duplicates before inserting",
          category: "process",
          workspace_id: workspace_id
        )

      assert result.category == "process"

      assert count("SELECT COUNT(*) FROM observations WHERE workspace_id = ?1", [workspace_id]) ==
               1

      {:ok, [[category, content, source]]} =
        Store.raw_query(
          "SELECT category, content, source FROM observations WHERE workspace_id = ?1",
          [workspace_id]
        )

      assert category == "process"
      assert content == "Always check duplicates before inserting"
      assert source == "explicit"
    end
  end

  describe "decisions" do
    test "Remember.record_decision/2 writes a durable decision row" do
      workspace_id = ws()

      assert count("SELECT COUNT(*) FROM decisions WHERE workspace_id = ?1", [workspace_id]) == 0

      {:ok, decision} =
        Remember.record_decision("Price AI Masters at $2K per seat",
          rationale: "Matches Ed's call; keeps margin above 60%",
          decided_by: "roberto",
          workspace_id: workspace_id
        )

      assert decision.decided_by == "roberto"

      assert count("SELECT COUNT(*) FROM decisions WHERE workspace_id = ?1", [workspace_id]) == 1

      {:ok, [[title, dec, rationale, decided_by]]} =
        Store.raw_query(
          "SELECT title, decision, rationale, decided_by FROM decisions WHERE workspace_id = ?1",
          [workspace_id]
        )

      assert title == "Price AI Masters at $2K per seat"
      assert dec == "Price AI Masters at $2K per seat"
      assert rationale == "Matches Ed's call; keeps margin above 60%"
      assert decided_by == "roberto"
    end
  end

  describe "memory_relations" do
    test "Versioned.derive/2 creates a typed relation between two memories" do
      workspace_id = ws()

      {:ok, parent} = Memory.create(%{content: "Original launch plan", workspace_id: workspace_id})

      before =
        count(
          "SELECT COUNT(*) FROM memory_relations WHERE workspace_id = ?1",
          [workspace_id]
        )

      assert before == 0

      {:ok, child} = Memory.derive(parent.id, %{content: "Revised launch plan"})

      assert count("SELECT COUNT(*) FROM memory_relations WHERE workspace_id = ?1", [
               workspace_id
             ]) == 1

      # Shape + provenance: child --derives--> parent.
      {:ok, [[source, target, relation]]} =
        Store.raw_query(
          "SELECT source_memory_id, target_memory_id, relation FROM memory_relations WHERE workspace_id = ?1",
          [workspace_id]
        )

      assert source == child.id
      assert target == parent.id
      assert relation == "derives"

      # Public read path agrees.
      {:ok, rels} = Memory.relations(child.id)
      assert [%{relation: :derives, direction: :outbound, target_memory_id: parent_id}] = rels
      assert parent_id == parent.id
    end
  end

  describe "memory_detail_objects" do
    test "a detail object attaches to a parent memory object via the store flow" do
      workspace_id = ws()

      # Build + persist a parent Memory Object (real Memory Core object).
      parent =
        MemoryObject.new(%{
          id: "mo-#{workspace_id}",
          workspace_id: workspace_id,
          summary: "Deploy runbook",
          subject_anchor: "deploy"
        })

      :ok = MemoryCoreStore.insert_memory_object(parent)

      before =
        count("SELECT COUNT(*) FROM memory_detail_objects WHERE workspace_id = ?1", [workspace_id])

      assert before == 0

      # Real flow: record a recursive detail (a step/command) on the parent.
      :ok =
        MemoryCoreStore.insert_memory_detail_object(%{
          workspace_id: workspace_id,
          parent_object_type: "memory_object",
          parent_object_id: parent.id,
          detail_type: "command",
          detail_order: 0,
          action_class: "execute",
          detail_text: "Run the migration before flipping traffic",
          command_or_parameter_value: "mix ecto.migrate"
        })

      assert count("SELECT COUNT(*) FROM memory_detail_objects WHERE workspace_id = ?1", [
               workspace_id
             ]) == 1

      # Shape + provenance: detail points back at the parent memory object.
      {:ok, [[ptype, pid, dtype, dtext, cmd]]} =
        Store.raw_query(
          "SELECT parent_object_type, parent_object_id, detail_type, detail_text, command_or_parameter_value FROM memory_detail_objects WHERE workspace_id = ?1",
          [workspace_id]
        )

      assert ptype == "memory_object"
      assert pid == parent.id
      assert dtype == "command"
      assert dtext == "Run the migration before flipping traffic"
      assert cmd == "mix ecto.migrate"

      # Query helper round-trips it.
      {:ok, [[_id, _pt, ^pid | _]]} =
        MemoryCoreStore.get_memory_detail_objects(workspace_id, "memory_object", parent.id)
    end
  end
end
