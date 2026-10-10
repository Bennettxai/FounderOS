defmodule OptimalEngine.Pipeline.IntakeScopeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.ScopeEnvelope
  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Pipeline.Intake.Writer
  alias OptimalEngine.Signal
  alias OptimalEngine.Store
  alias OptimalEngine.Workspace
  alias OptimalEngine.WorkspaceTopology

  setup do
    tmp_dir = System.tmp_dir!() |> Path.join("optimal_intake_scope_test_#{:rand.uniform(99_999)}")
    File.mkdir_p!(tmp_dir)

    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)

    for folder <- ~w[inbox team person-operator] do
      File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
    end

    on_exit(fn ->
      Application.put_env(:optimal_engine, :root_path, original_root)
      File.rm_rf!(tmp_dir)
    end)

    %{tmp_dir: tmp_dir}
  end

  defp unique_workspace, do: "scope-test-ws-#{System.unique_integer([:positive])}"

  defp source_package_row(workspace_id) do
    Store.raw_query(
      """
      SELECT COUNT(*), MAX(quarantine_state), MAX(created_by)
      FROM source_packages
      WHERE workspace_id = ?1
      """,
      [workspace_id]
    )
  end

  defp ledger_count(workspace_id, activity_type, derivation_stage, actor_id) do
    {:ok, [[count]]} =
      Store.raw_query(
        """
        SELECT COUNT(*)
        FROM derivation_ledger
        WHERE workspace_id = ?1
          AND activity_type = ?2
          AND derivation_stage = ?3
          AND actor_id = ?4
        """,
        [workspace_id, activity_type, derivation_stage, actor_id]
      )

    count
  end

  # ─────────────────────────────────────────────────────────────
  # ScopeEnvelope
  # ─────────────────────────────────────────────────────────────

  describe "ScopeEnvelope.resolve/2" do
    test "fills best-known defaults when scope is unknown" do
      scope = ScopeEnvelope.resolve([])

      assert scope.actor == nil
      assert scope.tenant_id == "default"
      assert scope.workspace_id == "default"
      assert scope.operation_class == "intake.process"
      assert scope.node_hints == []
      assert scope.permissions == []
      assert :actor in scope.unresolved
      assert :tenant_id in scope.unresolved
      assert :workspace_id in scope.unresolved
      refute ScopeEnvelope.resolved?(scope)
    end

    test "resolves actor from actor, actor_id, and created_by aliases" do
      assert ScopeEnvelope.resolve(actor: "user:a").actor == "user:a"
      assert ScopeEnvelope.resolve(actor_id: "user:b").actor == "user:b"
      assert ScopeEnvelope.resolve(created_by: "user:c").actor == "user:c"
      # :actor wins over the aliases
      assert ScopeEnvelope.resolve(created_by: "user:c", actor: "user:a").actor == "user:a"
    end

    test "explicit scope is fully resolved" do
      scope =
        ScopeEnvelope.resolve(
          actor: "agent:osa",
          tenant_id: "default",
          workspace_id: "ws-1",
          operation_class: "batch.import",
          permissions: ["memory.write"]
        )

      assert scope.unresolved == []
      assert ScopeEnvelope.resolved?(scope)
      assert scope.operation_class == "batch.import"
      assert scope.permissions == ["memory.write"]
    end

    test "overrides win over attrs" do
      scope = ScopeEnvelope.resolve([workspace_id: "a"], workspace_id: "b")
      assert scope.workspace_id == "b"
    end

    test "collects node hints from :node and :node_hints" do
      scope = ScopeEnvelope.resolve(node: "team", node_hints: ["inbox", "team"])
      assert scope.node_hints == ["team", "inbox"]
    end

    test "source_package_opts threads scope and preserves it in metadata" do
      scope = ScopeEnvelope.resolve(workspace_id: "ws-2", actor: "user:x")
      opts = ScopeEnvelope.source_package_opts(scope, source_type: "manual")

      assert opts[:tenant_id] == "default"
      assert opts[:workspace_id] == "ws-2"
      assert opts[:created_by] == "user:x"
      assert opts[:source_type] == "manual"
      assert opts[:metadata].scope.operation_class == "intake.process"
      assert "tenant_id" in opts[:metadata].scope.unresolved
    end

    test "ledger_opts carries actor and tenant/workspace scope" do
      scope = ScopeEnvelope.resolve(workspace_id: "ws-3", actor: "user:y")
      opts = ScopeEnvelope.ledger_opts(scope)

      assert opts[:actor_id] == "user:y"
      assert opts[:tenant_id] == "default"
      assert opts[:workspace_id] == "ws-3"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Source-first intake
  # ─────────────────────────────────────────────────────────────

  describe "source-first intake" do
    test "rejected low-quality input still keeps the Source Package with rejection lineage" do
      workspace_id = unique_workspace()

      raw_text = """
      ---
      signal:
        sn_ratio: 0.1
      ---
      noisy scratch text #{System.unique_integer([:positive])}
      """

      assert {:error, :signal_too_noisy} =
               Intake.process(raw_text, workspace_id: workspace_id, actor: "user:rejector")

      # The Source Package was committed BEFORE classification rejected it
      assert {:ok, [[1, "rejected", "user:rejector"]]} = source_package_row(workspace_id)

      # The intake attempt has lineage: source, processor, actor, policy
      assert ledger_count(workspace_id, "intake.quality_gate", "source_rejected", "user:rejector") ==
               1

      {:ok, [[parser_id, policy_version]]} =
        Store.raw_query(
          """
          SELECT parser_id, policy_version
          FROM derivation_ledger
          WHERE workspace_id = ?1 AND derivation_stage = 'source_rejected'
          """,
          [workspace_id]
        )

      assert parser_id == "optimal_engine.pipeline.intake"
      assert policy_version == "intake_quality_gate_v1"
    end

    test "accepted intake threads actor into Source Package and intake ledger entry" do
      workspace_id = unique_workspace()
      raw_text = "team sync notes for scope threading #{System.unique_integer([:positive])}"

      assert {:ok, result} =
               Intake.process(raw_text,
                 node: "team",
                 workspace_id: workspace_id,
                 actor: "agent:osa",
                 title: "Scope Threading Note"
               )

      assert result.quality_action == :accepted
      assert result.source_package.workspace_id == workspace_id
      assert result.source_package.created_by == "agent:osa"

      assert {:ok, [[1, "clear", "agent:osa"]]} = source_package_row(workspace_id)

      assert ledger_count(
               workspace_id,
               "intake.process",
               "source_to_signal_context",
               "agent:osa"
             ) == 1
    end

    test "low-signal inbox fallback is quarantined but evidence and files survive", %{
      tmp_dir: tmp_dir
    } do
      workspace_id = unique_workspace()

      raw_text = """
      ---
      signal:
        sn_ratio: 0.45
      ---
      misc scratch lines without names #{System.unique_integer([:positive])}
      """

      assert {:ok, result} = Intake.process(raw_text, workspace_id: workspace_id)

      assert result.quality_action == :quarantined
      assert result.routed_to == ["inbox"]

      # File still written (to the workspace's own inbox) and Context row
      # still indexed
      [primary] = result.files_written
      assert String.starts_with?(primary, "#{workspace_id}/inbox/signals/")
      assert File.exists?(Path.join(tmp_dir, primary))
      assert {:ok, _ctx} = Store.get_context(result.signal.id)

      # Source Package survives, marked quarantined, with ledger lineage
      assert {:ok, [[1, "quarantined", nil]]} = source_package_row(workspace_id)

      {:ok, [[ledger_entries]]} =
        Store.raw_query(
          """
          SELECT COUNT(*)
          FROM derivation_ledger
          WHERE workspace_id = ?1
            AND activity_type = 'intake.quality_gate'
            AND derivation_stage = 'source_quarantined'
          """,
          [workspace_id]
        )

      assert ledger_entries == 1
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Topology-aware folder resolution
  # ─────────────────────────────────────────────────────────────

  describe "topology-aware folder resolution" do
    test "topology-created Node routes to its own folder, not inbox", %{tmp_dir: tmp_dir} do
      workspace_id = unique_workspace()
      slug = "research-lab-#{System.unique_integer([:positive])}"

      {:ok, _node} =
        WorkspaceTopology.create_node(%{
          workspace_id: workspace_id,
          slug: slug,
          name: "Research Lab",
          kind: :learning
        })

      assert {:ok, result} =
               Intake.process(
                 "research lab kickoff notes #{System.unique_integer([:positive])}",
                 node: slug,
                 workspace_id: workspace_id,
                 title: "Lab Kickoff"
               )

      [primary] = result.files_written
      assert String.starts_with?(primary, "#{workspace_id}/nodes/#{slug}/signals/")
      assert File.exists?(Path.join(tmp_dir, primary))
    end

    test "unknown node without topology row falls back to inbox without losing evidence", %{
      tmp_dir: tmp_dir
    } do
      workspace_id = unique_workspace()
      ghost_node = "ghost-node-#{System.unique_integer([:positive])}"

      assert {:ok, result} =
               Intake.process(
                 "unscoped evidence drop #{System.unique_integer([:positive])}",
                 node: ghost_node,
                 workspace_id: workspace_id
               )

      [primary] = result.files_written
      assert String.starts_with?(primary, "#{workspace_id}/inbox/signals/")
      assert File.exists?(Path.join(tmp_dir, primary))

      # Missing Node scope is a routing problem, not a reason to lose evidence
      assert {:ok, [[1, "clear", nil]]} = source_package_row(workspace_id)
    end

    test "Writer.node_to_folder/2 resolves topology first, then legacy aliases, then inbox" do
      workspace_id = unique_workspace()
      slug = "custom-node-#{System.unique_integer([:positive])}"

      {:ok, _node} =
        WorkspaceTopology.create_node(%{
          workspace_id: workspace_id,
          slug: slug,
          name: "Custom Node",
          kind: :project
        })

      scope = [tenant_id: "default", workspace_id: workspace_id]

      # 1. Topology node routes to its own folder inside the workspace dir
      assert Writer.node_to_folder(slug, scope) == "#{workspace_id}/nodes/#{slug}"
      # 2. Legacy alias fallback routes into the workspace tree too
      assert Writer.node_to_folder("operator", scope) == "#{workspace_id}/person-operator"
      # 3. Unknown slug falls back to the workspace's own inbox
      assert Writer.node_to_folder("never-created-#{System.unique_integer()}", scope) ==
               "#{workspace_id}/inbox"

      # The default workspace keeps the unprefixed single-workspace layout
      default_scope = [tenant_id: "default", workspace_id: "default"]
      assert Writer.node_to_folder("operator", default_scope) == "person-operator"
      assert Writer.node_to_folder(nil, default_scope) == "inbox"

      # Unscoped calls keep legacy behavior
      assert Writer.node_to_folder(slug) == "inbox"
      assert Writer.node_to_folder("operator") == "person-operator"
      assert Writer.node_to_folder(nil) == "inbox"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Workspace segregation of on-disk signal storage
  # ─────────────────────────────────────────────────────────────

  describe "workspace-segregated signal storage" do
    test "registered workspace routes through its provisioned slug directory" do
      ws_slug = "ws-dir-#{System.unique_integer([:positive])}"
      {:ok, workspace} = Workspace.create(%{slug: ws_slug, name: "Scoped Workspace"})

      # Non-default workspace ids carry the tenant (`default:<slug>`), but
      # files land in the directory Workspace.Filesystem provisioned.
      assert workspace.id == "default:#{ws_slug}"

      assert Writer.node_to_folder("operator", workspace_id: workspace.id) ==
               "#{ws_slug}/person-operator"

      assert Writer.node_to_folder(nil, workspace_id: workspace.id) == "#{ws_slug}/inbox"
    end

    test "same node slug in two workspaces writes segregated, non-clobbering files", %{
      tmp_dir: tmp_dir
    } do
      ws_a = unique_workspace()
      ws_b = unique_workspace()
      slug = "shared-node-#{System.unique_integer([:positive])}"

      for ws <- [ws_a, ws_b] do
        {:ok, _node} =
          WorkspaceTopology.create_node(%{
            workspace_id: ws,
            slug: slug,
            name: "Shared Slug Node",
            kind: :project
          })
      end

      signal = build_signal("Quarterly Review", slug)

      {:ok, path_a} = Writer.write_signal(signal, tenant_id: "default", workspace_id: ws_a)
      {:ok, path_b} = Writer.write_signal(signal, tenant_id: "default", workspace_id: ws_b)

      # Same node slug + same filename, yet two distinct on-disk files —
      # workspace B can never clobber (or read over) workspace A's evidence.
      refute path_a == path_b
      assert Path.basename(path_a) == Path.basename(path_b)
      assert path_a == Path.join([tmp_dir, ws_a, "nodes", slug, "signals", Path.basename(path_a)])
      assert path_b == Path.join([tmp_dir, ws_b, "nodes", slug, "signals", Path.basename(path_b)])
      assert File.exists?(path_a)
      assert File.exists?(path_b)
    end

    test "update_context and signal_uri stay inside the workspace tree", %{tmp_dir: tmp_dir} do
      workspace_id = unique_workspace()
      scope = [tenant_id: "default", workspace_id: workspace_id]

      :ok = Writer.update_context("operator", ["Scoped fact"], scope)

      context_path = Path.join([tmp_dir, workspace_id, "person-operator", "context.md"])
      assert File.exists?(context_path)
      assert File.read!(context_path) =~ "Scoped fact"
      refute File.exists?(Path.join([tmp_dir, "person-operator", "context.md"]))

      signal = build_signal("URI Check", "operator")

      assert String.starts_with?(
               Writer.relative_path(signal, scope),
               "#{workspace_id}/person-operator/signals/"
             )

      assert Writer.signal_uri(signal, scope) =~ "#{workspace_id}/person-operator/signals/"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Helpers
  # ─────────────────────────────────────────────────────────────

  defp build_signal(title, node) do
    now = DateTime.utc_now()

    %Signal{
      id: :crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower),
      title: title,
      mode: :linguistic,
      genre: "note",
      type: :inform,
      format: :markdown,
      created_at: now,
      modified_at: now,
      node: node,
      sn_ratio: 0.7,
      entities: [],
      l0_summary: "NOTE | #{node} | #{title} [S/N: 0.7]",
      l1_description: "Test signal: #{title}",
      content: "Raw content for #{title}",
      routed_to: []
    }
  end
end
