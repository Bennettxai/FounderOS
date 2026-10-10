defmodule OptimalEngine.Pipeline.IntakeEnrichmentTest do
  @moduledoc """
  Integration tests verifying that a real ingest call populates:
  (a) a non-'seed-%' chunk row with chunk.signal_id matching the context id
  (b) at least one non-'lives_in' edge predicate
  (c) a memory_detail_object with a real parent_object_id

  These cover the three behaviors wired in the intake enrichment pass:
  decompose+embed, semantic edges, and skeleton detail objects.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Store

  # Give async tasks time to complete before we query.
  @async_settle_ms 300

  setup do
    tmp_dir =
      System.tmp_dir!()
      |> Path.join("optimal_intake_enrichment_test_#{:rand.uniform(999_999)}")

    File.mkdir_p!(tmp_dir)
    File.mkdir_p!(Path.join(tmp_dir, "inbox/signals"))

    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)

    on_exit(fn ->
      Application.put_env(:optimal_engine, :root_path, original_root)
      File.rm_rf!(tmp_dir)
    end)

    :ok
  end

  defp unique_workspace,
    do: "enrichment-test-ws-#{System.unique_integer([:positive])}"

  # ── helpers ──────────────────────────────────────────────────────────────────

  defp non_seed_chunks_for(context_id) do
    case Store.raw_query(
           "SELECT id, signal_id FROM chunks WHERE signal_id = ?1 AND id NOT LIKE 'seed-%'",
           [context_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  defp non_lives_in_edges_for(workspace_id) do
    case Store.raw_query(
           "SELECT relation FROM edges WHERE workspace_id = ?1 AND relation <> 'lives_in'",
           [workspace_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  defp mdo_rows_for(source_package_id) do
    case Store.raw_query(
           """
           SELECT id, parent_object_id FROM memory_detail_objects
           WHERE parent_object_id = ?1
             AND parent_object_id NOT LIKE 'fixture-%'
           """,
           [source_package_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  # ── tests ─────────────────────────────────────────────────────────────────────

  describe "intake enrichment" do
    test "chunk row with non-seed id and signal_id matching context id" do
      ws = unique_workspace()

      text = """
      Alice and Bob discussed the Q3 roadmap during their planning meeting.
      They agreed to ship the new feature by end of quarter. Carol will
      review the spec before Friday.
      """

      assert {:ok, result} =
               Intake.process(text,
                 genre: "note",
                 workspace_id: ws,
                 entities: ["Alice", "Bob", "Carol"]
               )

      context_id = result.context.id

      # Allow async decompose task to finish.
      Process.sleep(@async_settle_ms)

      chunk_rows = non_seed_chunks_for(context_id)

      assert chunk_rows != [],
             "Expected at least one non-'seed-%' chunk row with signal_id=#{context_id}, got none"

      # Every returned row must have signal_id == context_id.
      Enum.each(chunk_rows, fn [_id, signal_id] ->
        assert signal_id == context_id,
               "chunk.signal_id #{inspect(signal_id)} does not match context.id #{inspect(context_id)}"
      end)
    end

    test "non-lives_in edge predicate exists after ingest" do
      ws = unique_workspace()

      text = """
      Roberto met with Jordan and Michael to discuss the platform strategy.
      They decided to move forward with the Fractal architecture.
      """

      assert {:ok, _result} =
               Intake.process(text,
                 genre: "note",
                 workspace_id: ws,
                 entities: ["Roberto", "Jordan", "Michael"]
               )

      # Allow async tasks and synchronous assert_edge calls to land.
      Process.sleep(@async_settle_ms)

      edge_rows = non_lives_in_edges_for(ws)

      relations = Enum.map(edge_rows, fn [relation] -> relation end) |> Enum.uniq()

      assert relations != [],
             "Expected at least one non-'lives_in' edge relation in workspace #{ws}, got none"

      assert Enum.any?(relations, &(&1 != "lives_in")),
             "All edge relations were 'lives_in'; wanted at least one semantic predicate. Got: #{inspect(relations)}"
    end

    test "memory_detail_object created with real parent_object_id for transcript genre" do
      ws = unique_workspace()

      text = """
      ## Participants
      Alice, Bob

      ## Key Points
      - Agreed on Q3 roadmap
      - Feature delivery by end of quarter

      ## Decisions Made
      - Ship new feature by September 30

      ## Action Items
      - Bob: write spec by Friday

      ## Open Questions
      - Budget approval still pending
      """

      assert {:ok, result} =
               Intake.process(text,
                 genre: "transcript",
                 workspace_id: ws,
                 entities: ["Alice", "Bob"]
               )

      source_package_id = result.source_package.id

      # source_package_id must not be a fixture placeholder.
      refute String.starts_with?(source_package_id, "fixture-"),
             "source_package.id should be a real id, got: #{source_package_id}"

      mdo_rows = mdo_rows_for(source_package_id)

      assert mdo_rows != [],
             "Expected at least one memory_detail_object with parent_object_id=#{source_package_id}, got none"

      # All returned MDO rows must have the correct parent_object_id.
      Enum.each(mdo_rows, fn [_id, parent_id] ->
        assert parent_id == source_package_id,
               "MDO parent_object_id #{inspect(parent_id)} does not match source_package.id #{inspect(source_package_id)}"
      end)
    end
  end
end
