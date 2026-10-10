defmodule OptimalEngine.Pipeline.IntakeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Intake, as: Intake
  alias OptimalEngine.Pipeline.Intake.Skeleton, as: Skeleton
  alias OptimalEngine.Pipeline.Intake.Writer, as: Writer
  alias OptimalEngine.Signal

  # ─────────────────────────────────────────────────────────────
  # Skeleton
  # ─────────────────────────────────────────────────────────────

  describe "Skeleton.sections_for/1" do
    test "returns correct sections for transcript" do
      sections = Skeleton.sections_for("transcript")
      names = Enum.map(sections, & &1.name)

      assert names == [
               "Participants",
               "Key Points",
               "Decisions Made",
               "Action Items",
               "Open Questions"
             ]
    end

    test "returns correct sections for brief" do
      sections = Skeleton.sections_for("brief")
      names = Enum.map(sections, & &1.name)
      assert names == ["Objective", "Key Messages", "Call to Action", "Supporting Materials"]
    end

    test "returns correct sections for spec" do
      sections = Skeleton.sections_for("spec")
      names = Enum.map(sections, & &1.name)

      assert names == [
               "Goal",
               "Requirements",
               "Constraints",
               "Architecture",
               "Acceptance Criteria"
             ]
    end

    test "returns correct sections for plan" do
      sections = Skeleton.sections_for("plan")
      names = Enum.map(sections, & &1.name)

      assert names == [
               "Objective",
               "Non-Negotiables",
               "Time Blocks",
               "Dependencies",
               "Success Criteria"
             ]
    end

    test "returns correct sections for note" do
      sections = Skeleton.sections_for("note")
      names = Enum.map(sections, & &1.name)
      assert names == ["Context", "Content", "Route"]
    end

    test "returns correct sections for decision-log" do
      sections = Skeleton.sections_for("decision-log")
      names = Enum.map(sections, & &1.name)
      assert names == ["Decision", "Context", "Options Considered", "Rationale", "Implications"]
    end

    test "returns correct sections for standup" do
      sections = Skeleton.sections_for("standup")
      names = Enum.map(sections, & &1.name)
      assert names == ["Status", "Priorities This Week", "Blockers", "Fidelity Check"]
    end

    test "returns correct sections for review" do
      sections = Skeleton.sections_for("review")
      names = Enum.map(sections, & &1.name)
      assert names == ["Single-Loop", "Double-Loop", "Drift Scores", "Next Week"]
    end

    test "returns correct sections for report" do
      sections = Skeleton.sections_for("report")
      names = Enum.map(sections, & &1.name)
      assert names == ["Executive Summary", "Findings", "Analysis", "Recommendations"]
    end

    test "returns correct sections for pitch" do
      sections = Skeleton.sections_for("pitch")
      names = Enum.map(sections, & &1.name)
      assert names == ["Hook", "Problem", "Solution", "Proof", "Ask"]
    end

    test "falls back to note skeleton for unknown genre" do
      sections = Skeleton.sections_for("nonexistent-genre")
      names = Enum.map(sections, & &1.name)
      assert names == ["Context", "Content", "Route"]
    end

    test "supported_genres/0 returns all 10 genres" do
      genres = Skeleton.supported_genres()
      assert length(genres) == 10

      expected = ~w[transcript brief spec plan note decision-log standup review report pitch]
      Enum.each(expected, fn g -> assert g in genres, "Missing genre: #{g}" end)
    end

    test "supported?/1 returns true for known genre" do
      assert Skeleton.supported?("transcript") == true
      assert Skeleton.supported?("brief") == true
    end

    test "supported?/1 returns false for unknown genre" do
      assert Skeleton.supported?("mystery") == false
    end
  end

  describe "Skeleton.apply_skeleton/2" do
    test "places raw content under the first section" do
      result = Skeleton.apply_skeleton("note", "Alice said pricing is $99/mo")
      assert String.starts_with?(result, "## Context\n\nAlice said pricing is $99/mo")
    end

    test "generates empty section headers for remaining sections" do
      result = Skeleton.apply_skeleton("note", "some content")
      assert String.contains?(result, "## Content\n\n")
      assert String.contains?(result, "## Route\n\n")
    end

    test "includes hints as HTML comments" do
      result = Skeleton.apply_skeleton("note", "content")
      assert String.contains?(result, "<!-- ")
    end

    test "handles empty content gracefully" do
      result = Skeleton.apply_skeleton("brief", "")
      assert String.contains?(result, "## Objective")
    end

    test "transcript skeleton includes all 5 sections" do
      result = Skeleton.apply_skeleton("transcript", "Alice, Alice")
      assert String.contains?(result, "## Participants")
      assert String.contains?(result, "## Key Points")
      assert String.contains?(result, "## Decisions Made")
      assert String.contains?(result, "## Action Items")
      assert String.contains?(result, "## Open Questions")
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Writer
  # ─────────────────────────────────────────────────────────────

  describe "Writer" do
    setup do
      tmp_dir = System.tmp_dir!() |> Path.join("optimal_writer_test_#{:rand.uniform(99_999)}")
      File.mkdir_p!(tmp_dir)

      # Override root_path for the duration of the test
      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      # Create node folder structure
      for folder <- ~w[project-platform-launch operation-revenue inbox person-operator] do
        File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
      end

      on_exit(fn ->
        Application.put_env(:optimal_engine, :root_path, original_root)
        File.rm_rf!(tmp_dir)
      end)

      %{tmp_dir: tmp_dir}
    end

    test "write_signal/1 creates file with YAML frontmatter", %{tmp_dir: tmp_dir} do
      signal = build_signal("Requirements Review Call", "project-platform-launch", "transcript")

      {:ok, path} = Writer.write_signal(signal)

      assert File.exists?(path)
      assert String.starts_with?(path, tmp_dir)
      assert String.contains?(path, "project-platform-launch/signals/")
      assert String.ends_with?(path, ".md")

      content = File.read!(path)
      assert String.starts_with?(content, "---\n")
      assert String.contains?(content, "node: project-platform-launch")
      assert String.contains?(content, "genre: transcript")
      assert String.contains?(content, "title: Requirements Review Call")
    end

    test "write_signal/1 filename is date-slug formatted", %{} do
      signal =
        build_signal("Requirements Review Call 2026", "project-platform-launch", "transcript")

      {:ok, path} = Writer.write_signal(signal)

      filename = Path.basename(path)

      assert Regex.match?(
               ~r/^\d{4}-\d{2}-\d{2}-requirements-review-call-2026\.md$/,
               filename
             )
    end

    test "write_signal/1 creates signals/ directory if missing", %{tmp_dir: tmp_dir} do
      # Remove the signals dir for operator
      signals_dir = Path.join([tmp_dir, "person-operator", "signals"])
      File.rm_rf!(signals_dir)

      signal = build_signal("My Note", "operator", "note")
      {:ok, path} = Writer.write_signal(signal)

      assert File.exists?(path)
    end

    test "write_signal/1 body contains genre skeleton sections", %{} do
      signal = build_signal("Weekly Standup", "operator", "standup")

      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      assert String.contains?(content, "## Status")
      assert String.contains?(content, "## Priorities This Week")
      assert String.contains?(content, "## Blockers")
    end

    test "write_signal/1 includes entities in frontmatter", %{} do
      signal = build_signal("Ed Call", "project-platform-launch", "transcript")
      signal = %{signal | entities: ["Alice", "Alice"]}

      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      assert String.contains?(content, "- Alice")
      assert String.contains?(content, "- Alice")
    end

    test "write_signal/1 includes routed_to in frontmatter", %{} do
      signal = build_signal("Plan", "operator", "plan")
      signal = %{signal | routed_to: ["person-operator", "operation-revenue"]}

      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      assert String.contains?(content, "- person-operator")
      assert String.contains?(content, "- operation-revenue")
    end

    test "write_signal/1 includes tiers l0 and l1 in frontmatter", %{} do
      signal = build_signal("Platform Spec", "operator", "spec")

      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      assert String.contains?(content, "tiers:")
      assert String.contains?(content, "l0:")
      assert String.contains?(content, "l1:")
    end

    test "write_cross_references/2 writes to additional nodes", %{tmp_dir: tmp_dir} do
      signal = build_signal("Revenue Note", "project-platform-launch", "note")
      signal = %{signal | routed_to: ["project-platform-launch", "operation-revenue"]}

      {:ok, paths} = Writer.write_cross_references(signal, ["operation-revenue"])

      assert length(paths) == 1
      [cross_path] = paths
      assert String.contains?(cross_path, "operation-revenue/signals/")
      assert File.exists?(cross_path)

      # Verify cross-ref frontmatter
      content = File.read!(cross_path)
      assert String.contains?(content, "cross_ref_from: project-platform-launch")

      _ = tmp_dir
    end

    test "write_cross_references/2 skips nodes that map to the same folder as primary", %{} do
      signal = build_signal("Note", "inbox", "note")
      # "inbox" and "inbox" both map to inbox
      {:ok, paths} = Writer.write_cross_references(signal, ["inbox"])
      assert paths == []
    end

    test "update_context/2 appends to existing context.md", %{tmp_dir: tmp_dir} do
      context_path = Path.join([tmp_dir, "project-platform-launch", "context.md"])
      File.write!(context_path, "# Platform Launch\n\nExisting content.\n")

      :ok =
        Writer.update_context("project-platform-launch", [
          "Pricing is $2K/seat",
          "Course launches Q2"
        ])

      content = File.read!(context_path)
      assert String.contains?(content, "Pricing is $2K/seat")
      assert String.contains?(content, "Course launches Q2")
      assert String.contains?(content, "## Facts Updated")
    end

    test "update_context/2 creates context.md if missing", %{tmp_dir: tmp_dir} do
      context_path = Path.join([tmp_dir, "project-platform-launch", "context.md"])
      File.rm(context_path)

      :ok = Writer.update_context("project-platform-launch", ["Fact one"])

      assert File.exists?(context_path)
      content = File.read!(context_path)
      assert String.contains?(content, "Fact one")
    end

    test "node_to_folder/1 maps all 12 nodes correctly" do
      assert Writer.node_to_folder("operator") == "person-operator"
      assert Writer.node_to_folder("product-customer-portal") == "product-customer-portal"
      assert Writer.node_to_folder("entity-company") == "entity-company"
      assert Writer.node_to_folder("project-platform-launch") == "project-platform-launch"
      assert Writer.node_to_folder("entity-company") == "entity-company"
      assert Writer.node_to_folder("operation-delivery") == "operation-delivery"
      assert Writer.node_to_folder("team") == "team"
      assert Writer.node_to_folder("learning-research-library") == "learning-research-library"
      assert Writer.node_to_folder("inbox") == "inbox"
      assert Writer.node_to_folder("team") == "team"
      assert Writer.node_to_folder("operation-revenue") == "operation-revenue"
      assert Writer.node_to_folder("operation-delivery") == "operation-delivery"
    end

    test "node_to_folder/1 accepts folder names passthrough" do
      assert Writer.node_to_folder("project-platform-launch") == "project-platform-launch"
      assert Writer.node_to_folder("operation-revenue") == "operation-revenue"
    end

    test "node_to_folder/1 defaults to inbox for unknown node" do
      assert Writer.node_to_folder("unknown-node") == "inbox"
      assert Writer.node_to_folder(nil) == "inbox"
    end

    test "relative_path/1 returns node_folder/signals/filename" do
      signal = build_signal("Ed Call", "project-platform-launch", "transcript")
      rel = Writer.relative_path(signal)
      assert String.starts_with?(rel, "project-platform-launch/signals/")
      assert String.ends_with?(rel, ".md")
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Intake GenServer
  # ─────────────────────────────────────────────────────────────

  describe "Intake.process/2" do
    setup do
      tmp_dir = System.tmp_dir!() |> Path.join("optimal_intake_test_#{:rand.uniform(99_999)}")
      File.mkdir_p!(tmp_dir)

      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      # Create all 12 node folder structures
      for folder <- ~w[
        person-operator product-customer-portal entity-company project-platform-launch entity-company
        operation-delivery team learning-research-library
        inbox team operation-revenue operation-delivery
      ] do
        File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
      end

      on_exit(fn ->
        Application.put_env(:optimal_engine, :root_path, original_root)
        File.rm_rf!(tmp_dir)
      end)

      %{tmp_dir: tmp_dir}
    end

    test "returns {:ok, result} with required fields", %{} do
      {:ok, result} =
        Intake.process("Customer called about pricing. We decided on $99/mo.",
          genre: "note",
          node: "project-platform-launch"
        )

      assert is_struct(result.signal, Signal)
      assert is_struct(result.context, OptimalEngine.Context)
      assert is_list(result.files_written)
      assert is_list(result.routed_to)
      assert is_list(result.cross_references)
      assert is_binary(result.uri)
    end

    test "extracts a pending claim and ledgers signal_to_claim", %{} do
      workspace_id = "test-intake-claims-#{:erlang.unique_integer([:positive])}"

      {:ok, result} =
        Intake.process("Decision: Finance lead owns pricing renewal for Q4 at $2000.",
          genre: "decision-log",
          node: "project-platform-launch",
          workspace_id: workspace_id
        )

      # A pending claim is produced from the signal/source.
      assert result.pending_claim != nil
      claim = result.pending_claim
      assert claim.status == "pending" or claim.lifecycle_state == "pending"
      assert claim.signal_id == result.signal.id
      assert claim.workspace_id == workspace_id

      # The claim row exists in the store, scoped to the workspace.
      assert {:ok, [[1]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT COUNT(*) FROM claims WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, claim.id]
               )

      # The signal -> claim hop is ledgered (the new lineage entry).
      assert {:ok, [[count]]} =
               OptimalEngine.Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'intake.extract_claim'
                   AND derivation_stage = 'signal_to_claim'
                 """,
                 [workspace_id]
               )

      assert count >= 1
    end

    test "honors extract_claims: false to skip claim extraction", %{} do
      workspace_id = "test-intake-noclaims-#{:erlang.unique_integer([:positive])}"

      {:ok, result} =
        Intake.process("Decision: Finance lead owns pricing renewal for Q4 at $2000.",
          genre: "decision-log",
          node: "project-platform-launch",
          workspace_id: workspace_id,
          extract_claims: false
        )

      assert result.pending_claim == nil

      assert {:ok, [[0]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT COUNT(*) FROM claims WHERE workspace_id = ?1",
                 [workspace_id]
               )
    end

    test "writes primary signal file to disk", %{tmp_dir: tmp_dir} do
      {:ok, result} =
        Intake.process("Alice wants platform V2 by Q3.",
          genre: "plan",
          node: "operator",
          title: "V2 Plan"
        )

      assert length(result.files_written) == 1
      [relative_path] = result.files_written

      abs_path = Path.join(tmp_dir, relative_path)
      assert File.exists?(abs_path)
      assert String.contains?(relative_path, "person-operator/signals/")
    end

    test "written file has valid YAML frontmatter", %{tmp_dir: tmp_dir} do
      {:ok, result} =
        Intake.process("Spec for new feature.",
          genre: "spec",
          node: "operator",
          title: "Feature Spec"
        )

      [relative_path] = result.files_written
      content = File.read!(Path.join(tmp_dir, relative_path))

      {frontmatter, _body} = OptimalEngine.Pipeline.Classifier.parse_frontmatter(content)
      assert Map.has_key?(frontmatter, "node")
      assert Map.has_key?(frontmatter, "signal")
      assert get_in(frontmatter, ["signal", "genre"]) == "spec"
    end

    test "genre override is respected", %{tmp_dir: tmp_dir} do
      {:ok, result} =
        Intake.process("Alice, Alice. Pricing decided.",
          genre: "transcript",
          node: "project-platform-launch",
          title: "Ed Call"
        )

      [relative_path] = result.files_written
      content = File.read!(Path.join(tmp_dir, relative_path))

      assert String.contains?(content, "genre: transcript")
      assert String.contains?(content, "## Participants")
      assert String.contains?(content, "## Action Items")
    end

    test "title override is respected", %{} do
      {:ok, result} =
        Intake.process("some content", title: "My Custom Title", node: "operator")

      assert result.signal.title == "My Custom Title"
    end

    test "node override sets primary node on the signal", %{} do
      {:ok, result} =
        Intake.process("agency work update", node: "operation-delivery")

      assert result.signal.node == "operation-delivery"
    end

    test "node override writes file to correct folder", %{tmp_dir: tmp_dir} do
      {:ok, result} =
        Intake.process("agency work update", node: "operation-delivery")

      [primary] = result.files_written
      assert String.starts_with?(primary, "operation-delivery/signals/")

      assert File.exists?(Path.join(tmp_dir, primary))
    end

    test "entity override merges with auto-extracted", %{} do
      {:ok, result} =
        Intake.process(
          "Alice called. Pricing discussion.",
          entities: ["Alice"],
          node: "project-platform-launch"
        )

      assert "Alice" in result.signal.entities
    end

    test "uri is a valid optimal:// URI", %{} do
      {:ok, result} = Intake.process("test content", node: "operator")
      assert String.starts_with?(result.uri, "optimal://nodes/")
    end

    test "signal is indexed in SQLite store", %{} do
      {:ok, result} = Intake.process("indexed content", node: "operator", title: "Index Test")

      {:ok, ctx} = OptimalEngine.Store.get_context(result.signal.id)
      assert ctx.title == "Index Test"
    end

    test "auto-detects decision genre from content", %{} do
      {:ok, result} =
        Intake.process(
          "We decided to use Elixir. Decision: use Phoenix for the API layer.",
          node: "operator"
        )

      assert result.signal.type == :decide
    end

    test "auto-detects transcript genre from Participants heading", %{} do
      content = """
      ## Participants
      - Alice
      - Alice

      ## Key Points
      Discussed pricing.

      ## Action Items
      - Follow up next week
      """

      {:ok, result} = Intake.process(content, node: "project-platform-launch")
      assert result.signal.genre == "transcript"
    end

    test "cross-references are written for financial content", %{tmp_dir: tmp_dir} do
      # invoice genre triggers operation-revenue cross-cutting rule
      {:ok, result} =
        Intake.process(
          "Invoice #1234. Amount: $5,000. Payment due: 2026-04-01.",
          genre: "invoice",
          node: "entity-company",
          title: "Invoice 1234"
        )

      # Should have written cross-ref to operation-revenue
      all_paths = result.files_written ++ result.cross_references
      has_money = Enum.any?(all_paths, &String.contains?(&1, "operation-revenue"))
      assert has_money, "Expected cross-ref to operation-revenue, got: #{inspect(all_paths)}"

      _ = tmp_dir
    end

    test "cross-reference contexts carry the intake workspace, not \"default\"", %{
      tmp_dir: tmp_dir
    } do
      workspace_id = "ws-cross-#{System.unique_integer([:positive])}"

      {:ok, result} =
        Intake.process(
          "Invoice #4321. Amount: $7,500. Payment due: 2026-05-01.",
          genre: "invoice",
          node: "entity-company",
          title: "Invoice 4321",
          workspace_id: workspace_id
        )

      cross_refs = result.cross_references
      assert cross_refs != [], "Expected at least one cross-reference to be written"

      assert Enum.any?(cross_refs, &String.contains?(&1, "operation-revenue")),
             "Expected cross-ref to operation-revenue, got: #{inspect(cross_refs)}"

      # Every cross-ref Context row must be stamped with the intake workspace —
      # regression guard against leaking workspace content into "default".
      for rel <- cross_refs do
        abs_path = Path.join(tmp_dir, rel)

        assert {:ok, [[stored_workspace]]} =
                 OptimalEngine.Store.raw_query(
                   "SELECT workspace_id FROM contexts WHERE path = ?1",
                   [abs_path]
                 )

        assert stored_workspace == workspace_id,
               "Cross-ref context #{rel} carries workspace #{inspect(stored_workspace)}, " <>
                 "expected #{inspect(workspace_id)}"
      end

      # Primary context carries the workspace as well
      assert result.context.workspace_id == workspace_id

      # Nothing from this intake landed in the default workspace
      assert {:ok, [[0]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT COUNT(*) FROM contexts WHERE workspace_id = 'default' AND path LIKE ?1",
                 [tmp_dir <> "%"]
               )
    end

    test "handles empty entities gracefully", %{} do
      {:ok, result} = Intake.process("No people mentioned here.", node: "operator")
      assert is_list(result.signal.entities)
    end

    test "handles very short input", %{} do
      {:ok, result} = Intake.process("ok", node: "inbox")
      assert is_binary(result.uri)
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Writer file format validation (end-to-end parse check)
  # ─────────────────────────────────────────────────────────────

  describe "written file parseable by Classifier" do
    setup do
      tmp_dir = System.tmp_dir!() |> Path.join("optimal_parse_test_#{:rand.uniform(99_999)}")
      File.mkdir_p!(tmp_dir)
      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      for folder <- ~w[project-platform-launch] do
        File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
      end

      on_exit(fn ->
        Application.put_env(:optimal_engine, :root_path, original_root)
        File.rm_rf!(tmp_dir)
      end)

      %{tmp_dir: tmp_dir}
    end

    test "written file is detected as :signal type", %{} do
      signal = build_signal("Platform Spec", "project-platform-launch", "spec")
      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      ctx_type = OptimalEngine.Pipeline.Classifier.detect_type(content, path: path)
      assert ctx_type == :signal
    end

    test "written file classifies with correct genre", %{} do
      signal = build_signal("Requirements Review", "project-platform-launch", "transcript")
      {:ok, path} = Writer.write_signal(signal)

      content = File.read!(path)
      ctx = OptimalEngine.Pipeline.Classifier.classify_context(content, path: path)
      assert ctx.signal.genre == "transcript"
    end

    test "written file has parseable YAML frontmatter", %{} do
      signal = build_signal("My Plan", "project-platform-launch", "plan")
      signal = %{signal | entities: ["Alice", "Alice"], routed_to: ["project-platform-launch"]}

      {:ok, path} = Writer.write_signal(signal)
      content = File.read!(path)

      {fm, _body} = OptimalEngine.Pipeline.Classifier.parse_frontmatter(content)
      assert Map.get(fm, "node") == "project-platform-launch"
      assert get_in(fm, ["signal", "genre"]) == "plan"
      assert get_in(fm, ["signal", "mode"]) == "linguistic"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Helpers
  # ─────────────────────────────────────────────────────────────

  defp build_signal(title, node, genre) do
    now = DateTime.utc_now()

    %Signal{
      id: :crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower),
      path: nil,
      title: title,
      mode: :linguistic,
      genre: genre,
      type: :inform,
      format: :markdown,
      structure: "",
      created_at: now,
      modified_at: now,
      valid_from: nil,
      valid_until: nil,
      supersedes: nil,
      node: node,
      sn_ratio: 0.7,
      entities: [],
      l0_summary: "#{String.upcase(genre)} | #{node} | #{title} [S/N: 0.7]",
      l1_description: "Test signal: #{title}",
      content: "Raw content for #{title}",
      routed_to: [],
      score: nil
    }
  end
end
