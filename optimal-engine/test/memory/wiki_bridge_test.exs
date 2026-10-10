defmodule OptimalEngine.Memory.WikiBridgeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory
  alias OptimalEngine.Memory.WikiBridge
  alias OptimalEngine.Wiki.{Page, Store}

  # Unique workspace_id per test to avoid cross-test pollution.
  defp ws, do: "wb-test-#{:erlang.unique_integer([:positive])}"
  defp slug, do: "wb-page-#{:erlang.unique_integer([:positive])}"

  # Build a minimal wiki page with cite directives.
  defp build_page(body, opts \\ []) do
    suffix = :erlang.unique_integer([:positive])
    s = Keyword.get(opts, :slug, "test-page-#{suffix}")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    %Page{
      tenant_id: "default",
      workspace_id: workspace_id,
      slug: s,
      audience: Keyword.get(opts, :audience, "default"),
      version: 1,
      frontmatter: %{"slug" => s},
      body: body,
      last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
      curated_by: "test"
    }
  end

  # ---------------------------------------------------------------------------
  # extract_from_wiki_page/2
  # ---------------------------------------------------------------------------

  describe "extract_from_wiki_page/2" do
    test "returns {:ok, []} when page has no cite directives" do
      page = build_page("## Summary\n\nNo citations here.\n")
      assert {:ok, []} = WikiBridge.extract_from_wiki_page(page)
    end

    test "parses a single cite directive and creates a memory" do
      workspace_id = ws()

      page =
        build_page(
          "The engine uses ETS for caching. {{cite: optimal://chunk/abc123}}\n",
          workspace_id: workspace_id
        )

      assert {:ok, [memory_id]} =
               WikiBridge.extract_from_wiki_page(page,
                 workspace_id: workspace_id,
                 tenant_id: "default"
               )

      assert {:ok, mem} = Memory.get(memory_id)
      assert mem.is_static == true
      assert mem.citation_uri == "optimal://chunk/abc123"
      assert mem.workspace_id == workspace_id
    end

    test "parses multiple cite directives from the same page" do
      workspace_id = ws()

      body = """
      First claim about signals. {{cite: optimal://chunk/c1}}
      Second claim about memory. {{cite: optimal://chunk/c2}}
      """

      page = build_page(body, workspace_id: workspace_id)

      assert {:ok, ids} =
               WikiBridge.extract_from_wiki_page(page,
                 workspace_id: workspace_id,
                 tenant_id: "default"
               )

      assert length(ids) == 2

      for id <- ids do
        assert {:ok, mem} = Memory.get(id)
        assert mem.is_static == true
        assert mem.workspace_id == workspace_id
      end
    end

    test "stores claim_hash in metadata for deduplication" do
      workspace_id = ws()

      page =
        build_page(
          "Signal processing is core. {{cite: optimal://chunk/x99}}\n",
          workspace_id: workspace_id
        )

      {:ok, [id]} =
        WikiBridge.extract_from_wiki_page(page, workspace_id: workspace_id, tenant_id: "default")

      {:ok, mem} = Memory.get(id)
      assert is_binary(Map.get(mem.metadata, "claim_hash"))
      assert Map.get(mem.metadata, "source") == "wiki_bridge"
    end

    test "idempotency — same claim twice produces only one memory row" do
      workspace_id = ws()
      s = slug()

      body = "ETS is used for caching. {{cite: optimal://chunk/dedup1}}\n"
      page = build_page(body, slug: s, workspace_id: workspace_id)

      {:ok, first_ids} =
        WikiBridge.extract_from_wiki_page(page, workspace_id: workspace_id, tenant_id: "default")

      assert length(first_ids) == 1

      # Call again with the same page — claim already exists.
      {:ok, second_ids} =
        WikiBridge.extract_from_wiki_page(page, workspace_id: workspace_id, tenant_id: "default")

      # No new rows created — second call returns empty list.
      assert second_ids == []

      # Confirm only one memory exists in the workspace with this hash.
      all_mems = Memory.list(workspace_id: workspace_id, include_forgotten: false, limit: 1000)

      matching =
        Enum.filter(all_mems, fn m ->
          m.citation_uri == "optimal://chunk/dedup1" and m.is_static == true
        end)

      assert length(matching) == 1
    end

    test "skips lines without cite directives — no spurious memories created" do
      workspace_id = ws()

      body = """
      ## Summary
      No citation on this line.
      This one has a claim. {{cite: optimal://chunk/real1}}
      Another plain line.
      """

      page = build_page(body, workspace_id: workspace_id)

      {:ok, ids} =
        WikiBridge.extract_from_wiki_page(page, workspace_id: workspace_id, tenant_id: "default")

      # Only the cited line becomes a memory.
      assert length(ids) == 1
    end
  end

  # ---------------------------------------------------------------------------
  # promote_memory_to_wiki/3
  # ---------------------------------------------------------------------------

  describe "promote_memory_to_wiki/3" do
    test "returns :error when memory does not exist" do
      assert {:error, _} = WikiBridge.promote_memory_to_wiki("mem_does_not_exist", "some-page")
    end

    test "creates a new wiki page scaffold when the target slug does not exist" do
      workspace_id = ws()
      s = slug()

      {:ok, mem} =
        Memory.create(%{
          content: "Important architectural decision",
          workspace_id: workspace_id,
          is_static: true
        })

      assert {:ok, page} =
               WikiBridge.promote_memory_to_wiki(mem.id, s,
                 workspace_id: workspace_id,
                 tenant_id: "default",
                 audience: "default"
               )

      assert page.slug == s
      assert String.contains?(page.body, "## Memory citations")
      assert String.contains?(page.body, "optimal://memory/#{mem.id}")
      assert String.contains?(page.body, "Important architectural decision")
    end

    test "appends to an existing wiki page's Memory citations section" do
      workspace_id = ws()
      s = slug()

      # Create and store an initial page.
      initial_page = %Page{
        tenant_id: "default",
        workspace_id: workspace_id,
        slug: s,
        audience: "default",
        version: 1,
        frontmatter: %{"slug" => s},
        body:
          "## Summary\n\nBaseline content.\n\n## Memory citations\n\n- First entry {{cite: optimal://memory/mem_old}}\n",
        last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
        curated_by: "test"
      }

      :ok = Store.put(initial_page)

      {:ok, mem} =
        Memory.create(%{
          content: "New promoted insight",
          workspace_id: workspace_id,
          is_static: true
        })

      assert {:ok, page} =
               WikiBridge.promote_memory_to_wiki(mem.id, s,
                 workspace_id: workspace_id,
                 tenant_id: "default",
                 audience: "default"
               )

      # Version bumped.
      assert page.version == 2
      # Both the old and new citation are present.
      assert String.contains?(page.body, "optimal://memory/mem_old")
      assert String.contains?(page.body, "optimal://memory/#{mem.id}")
      assert String.contains?(page.body, "New promoted insight")
    end

    test "persists the promoted page to the wiki store" do
      workspace_id = ws()
      s = slug()

      {:ok, mem} =
        Memory.create(%{
          content: "Persisted via bridge",
          workspace_id: workspace_id,
          is_static: true
        })

      {:ok, _} =
        WikiBridge.promote_memory_to_wiki(mem.id, s,
          workspace_id: workspace_id,
          tenant_id: "default",
          audience: "default"
        )

      # Read back from store — should exist.
      assert {:ok, persisted} = Store.latest("default", s, "default", workspace_id)
      assert String.contains?(persisted.body, "optimal://memory/#{mem.id}")
    end
  end

  # ---------------------------------------------------------------------------
  # Failure isolation — bridge crashes must not block load-bearing paths
  # ---------------------------------------------------------------------------

  describe "failure isolation" do
    test "extract_from_wiki_page returns {:error, ...} instead of raising on bad input" do
      # Pass something that will cause internal failure gracefully.
      # A page with a nil body exercises the rescue path.
      page = %Page{
        tenant_id: "default",
        workspace_id: "default",
        slug: "test-nil-body",
        audience: "default",
        version: 1,
        frontmatter: %{},
        body: nil,
        last_curated: nil,
        curated_by: nil
      }

      # Should not raise — returns {:ok, []} or {:error, ...}.
      result = WikiBridge.extract_from_wiki_page(page)
      assert match?({:ok, _}, result) or match?({:error, _}, result)
    end

    test "promote_memory_to_wiki does not raise on missing memory" do
      result = WikiBridge.promote_memory_to_wiki("mem_nonexistent_xyz", "some-slug")
      assert match?({:error, _}, result)
    end

    test "Memory.create with is_static=false does NOT trigger wiki promotion task" do
      # This is a regression guard: non-static memories must not fire
      # WikiBridge logic. We verify this by checking the return value is
      # {:ok, mem} and the memory exists correctly (no side-effect crash).
      workspace_id = ws()

      assert {:ok, mem} =
               Memory.create(%{
                 content: "Dynamic memory — not static",
                 workspace_id: workspace_id,
                 is_static: false
               })

      assert mem.is_static == false
      assert {:ok, fetched} = Memory.get(mem.id)
      assert fetched.is_static == false
    end

    test "Memory.create with is_static=true succeeds even when auto_promote_to_wiki is false (default)" do
      # Default config has auto_promote_to_wiki: false, so no wiki page is
      # created.  The memory itself must still be created cleanly.
      workspace_id = ws()

      assert {:ok, mem} =
               Memory.create(%{
                 content: "Static memory with default config",
                 workspace_id: workspace_id,
                 is_static: true
               })

      assert mem.is_static == true
      assert {:ok, _} = Memory.get(mem.id)
    end
  end
end
