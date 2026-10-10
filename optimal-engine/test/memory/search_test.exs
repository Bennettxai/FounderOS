defmodule OptimalEngine.Memory.SearchTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Versioned, as: Memory

  # Each test uses an isolated workspace so rows from different tests never
  # interfere — no full DB teardown required (all tests share one SQLite file).
  defp ws, do: "search-ws-#{:erlang.unique_integer([:positive])}"

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  # Creates a memory and asserts success. Returns the struct.
  defp create!(attrs) do
    {:ok, mem} = Memory.create(attrs)
    mem
  end

  # ---------------------------------------------------------------------------
  # FTS search — basic matching
  # ---------------------------------------------------------------------------

  describe "list/1 with :q" do
    test "returns memories whose content matches the query" do
      ws = ws()

      _pricing = create!(%{content: "Our annual pricing is $500 per seat", workspace_id: ws})
      _onboard = create!(%{content: "Onboarding new employees takes three weeks", workspace_id: ws})

      _policy =
        create!(%{content: "Security policy requires MFA for all accounts", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws, q: "pricing")

      assert length(results) == 1
      assert hd(results).content =~ "pricing"
    end

    test "returns multiple memories when more than one matches" do
      ws = ws()

      _a = create!(%{content: "pricing tier one", workspace_id: ws})
      _b = create!(%{content: "pricing tier two", workspace_id: ws})
      _c = create!(%{content: "onboarding guide", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws, q: "pricing")

      assert length(results) == 2
      assert Enum.all?(results, fn m -> m.content =~ "pricing" end)
    end

    test "empty q= returns all memories (no FTS filter)" do
      ws = ws()

      create!(%{content: "alpha content", workspace_id: ws})
      create!(%{content: "beta content", workspace_id: ws})
      create!(%{content: "gamma content", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws, q: "")

      assert length(results) == 3
    end

    test "nil q is treated as absent (no FTS filter)" do
      ws = ws()

      create!(%{content: "delta content", workspace_id: ws})
      create!(%{content: "epsilon content", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws, q: nil)

      assert length(results) == 2
    end

    test "query that matches nothing returns empty list" do
      ws = ws()

      create!(%{content: "machine learning infrastructure", workspace_id: ws})
      create!(%{content: "distributed systems design", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws, q: "zxqwerty99nonexistent")

      assert results == []
    end

    test "porter stemming: searching 'price' matches 'pricing'" do
      ws = ws()

      create!(%{content: "pricing is determined quarterly", workspace_id: ws})
      create!(%{content: "unrelated topic about sports", workspace_id: ws})

      # "price" and "pricing" share the same porter stem
      {:ok, results} = Memory.list(workspace_id: ws, q: "price")

      assert length(results) == 1
      assert hd(results).content =~ "pricing"
    end
  end

  # ---------------------------------------------------------------------------
  # Workspace isolation
  # ---------------------------------------------------------------------------

  describe "workspace isolation with :q" do
    test "search does not leak results across workspaces" do
      ws_a = ws()
      ws_b = ws()

      create!(%{content: "pricing policy in workspace A", workspace_id: ws_a})
      create!(%{content: "completely different text in workspace B", workspace_id: ws_b})

      {:ok, results_a} = Memory.list(workspace_id: ws_a, q: "pricing")
      {:ok, results_b} = Memory.list(workspace_id: ws_b, q: "pricing")

      assert length(results_a) == 1
      assert results_b == []
    end

    test "memories from workspace B are not visible in workspace A search" do
      ws_a = ws()
      ws_b = ws()

      create!(%{content: "authentication flow documentation", workspace_id: ws_b})

      {:ok, results} = Memory.list(workspace_id: ws_a, q: "authentication")

      assert results == []
    end
  end

  # ---------------------------------------------------------------------------
  # Forgotten memory exclusion
  # ---------------------------------------------------------------------------

  describe "forgotten memories are excluded from search" do
    test "forgotten memories do not appear in default search results" do
      ws = ws()

      live = create!(%{content: "pricing strategy for enterprise", workspace_id: ws})
      forgotten = create!(%{content: "pricing for old plans", workspace_id: ws})

      :ok = Memory.forget(forgotten.id)

      {:ok, results} = Memory.list(workspace_id: ws, q: "pricing")

      ids = Enum.map(results, & &1.id)
      assert live.id in ids
      refute forgotten.id in ids
    end

    test "include_forgotten: true includes forgotten memories in search" do
      ws = ws()

      live = create!(%{content: "pricing model overview", workspace_id: ws})

      forgotten =
        create!(%{content: "pricing from legacy era", workspace_id: ws, dedup: "always_insert"})

      :ok = Memory.forget(forgotten.id)

      {:ok, results} = Memory.list(workspace_id: ws, q: "pricing", include_forgotten: true)

      ids = Enum.map(results, & &1.id)
      assert live.id in ids
      assert forgotten.id in ids
    end
  end

  # ---------------------------------------------------------------------------
  # Audience filter + FTS
  # ---------------------------------------------------------------------------

  describe "audience filter combined with :q" do
    test "audience filter applies on top of FTS search" do
      ws = ws()

      eng = create!(%{content: "pricing engine specs", workspace_id: ws, audience: "engineering"})
      _sales = create!(%{content: "pricing sales deck", workspace_id: ws, audience: "sales"})

      {:ok, results} = Memory.list(workspace_id: ws, q: "pricing", audience: "engineering")

      assert length(results) == 1
      assert hd(results).id == eng.id
    end
  end

  # ---------------------------------------------------------------------------
  # Non-FTS list path is unaffected
  # ---------------------------------------------------------------------------

  describe "non-FTS list path unchanged" do
    test "list without :q still works correctly" do
      ws = ws()

      m1 = create!(%{content: "first memory", workspace_id: ws})
      m2 = create!(%{content: "second memory", workspace_id: ws})

      {:ok, results} = Memory.list(workspace_id: ws)

      ids = Enum.map(results, & &1.id)
      assert m1.id in ids
      assert m2.id in ids
    end

    test "list without :q respects is_forgotten filter" do
      ws = ws()

      live = create!(%{content: "live content", workspace_id: ws})
      forgotten = create!(%{content: "forgotten content", workspace_id: ws, dedup: "always_insert"})

      :ok = Memory.forget(forgotten.id)

      {:ok, results} = Memory.list(workspace_id: ws)

      ids = Enum.map(results, & &1.id)
      assert live.id in ids
      refute forgotten.id in ids
    end
  end
end
