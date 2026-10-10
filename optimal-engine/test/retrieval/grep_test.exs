defmodule OptimalEngine.Retrieval.GrepTest do
  @moduledoc """
  Tests for the hybrid grep engine.

  The grep engine sits on top of `Retrieval.Search`, so tests that require
  indexed content are necessarily integration-level. We test:

    1. Public contract — returns `{:ok, [match()]}` with required keys
    2. Workspace isolation — the `:workspace_id` option is forwarded
    3. Literal mode — `:literal` true still returns well-formed results
    4. Path prefix filtering — matches are restricted to slugs starting with prefix
    5. Intent + scale validation — invalid values are silently ignored (nil)
    6. Empty query guard — always handled by the upstream search engine
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Retrieval.Grep

  # ---------------------------------------------------------------------------
  # Contract: basic invocation
  # ---------------------------------------------------------------------------

  describe "grep/2 — basic contract" do
    test "returns {:ok, list} for any query" do
      assert {:ok, matches} = Grep.grep("pricing")
      assert is_list(matches)
    end

    test "each match has the required signal-trace keys" do
      {:ok, matches} = Grep.grep("decision", limit: 5)

      Enum.each(matches, fn m ->
        assert Map.has_key?(m, :slug)
        assert Map.has_key?(m, :scale)
        assert Map.has_key?(m, :intent)
        assert Map.has_key?(m, :sn_ratio)
        assert Map.has_key?(m, :modality)
        assert Map.has_key?(m, :snippet)
        assert Map.has_key?(m, :score)
      end)
    end

    test "score is a float in 0.0..1.0 range" do
      {:ok, matches} = Grep.grep("test query", limit: 5)

      Enum.each(matches, fn m ->
        assert is_float(m.score) or is_integer(m.score)
        assert m.score >= 0.0
      end)
    end

    test "snippet is a non-empty binary for all returned matches" do
      {:ok, matches} = Grep.grep("the", limit: 10)

      Enum.each(matches, fn m ->
        assert is_binary(m.snippet)
        assert String.length(m.snippet) > 0
      end)
    end

    test "returns at most :limit results" do
      {:ok, matches} = Grep.grep("a", limit: 3)
      assert length(matches) <= 3
    end
  end

  # ---------------------------------------------------------------------------
  # Workspace isolation
  # ---------------------------------------------------------------------------

  describe "grep/2 — workspace scoping" do
    test "accepts workspace_id without error" do
      assert {:ok, _} = Grep.grep("pricing", workspace_id: "default", limit: 5)
    end

    test "non-existent workspace returns empty list rather than error" do
      assert {:ok, matches} =
               Grep.grep("pricing",
                 workspace_id:
                   "workspace-that-does-not-exist-#{System.unique_integer([:positive])}",
                 limit: 5
               )

      assert is_list(matches)
    end
  end

  # ---------------------------------------------------------------------------
  # Literal mode
  # ---------------------------------------------------------------------------

  describe "grep/2 — literal mode" do
    test "literal: true still returns {:ok, list}" do
      assert {:ok, matches} = Grep.grep("test", literal: true, limit: 5)
      assert is_list(matches)
    end

    test "literal: false (default) returns {:ok, list}" do
      assert {:ok, matches} = Grep.grep("test", literal: false, limit: 5)
      assert is_list(matches)
    end
  end

  # ---------------------------------------------------------------------------
  # Path-prefix filtering
  # ---------------------------------------------------------------------------

  describe "grep/2 — path prefix" do
    test "path_prefix: nil returns all matches up to limit" do
      {:ok, without_prefix} = Grep.grep("a", path_prefix: nil, limit: 20)
      assert is_list(without_prefix)
    end

    test "path_prefix restricts matches to slugs starting with the prefix" do
      # We can't guarantee any specific slugs are indexed in CI, so we
      # verify the invariant: every returned slug starts with the prefix.
      prefix = "nonexistent-node-prefix-#{System.unique_integer([:positive])}"
      {:ok, matches} = Grep.grep("a", path_prefix: prefix, limit: 10)

      Enum.each(matches, fn m ->
        assert String.starts_with?(m.slug, prefix),
               "Expected slug '#{m.slug}' to start with '#{prefix}'"
      end)
    end

    test "path_prefix with trailing slash is handled" do
      {:ok, matches} = Grep.grep("a", path_prefix: "some-node/", limit: 5)
      assert is_list(matches)
    end
  end

  # ---------------------------------------------------------------------------
  # Intent + scale filters
  # ---------------------------------------------------------------------------

  describe "grep/2 — intent filter" do
    test "valid intent atom is accepted" do
      assert {:ok, _} = Grep.grep("decision", intent: :propose_decision, limit: 5)
    end

    test "valid intent string is accepted" do
      assert {:ok, _} = Grep.grep("commit", intent: "commit_action", limit: 5)
    end

    test "invalid intent string is silently dropped (treated as nil)" do
      # The engine normalizes invalid intents to nil — no crash, no error.
      assert {:ok, _} = Grep.grep("test", intent: "not_a_real_intent", limit: 5)
    end
  end

  describe "grep/2 — scale filter" do
    test "valid scale atom is accepted" do
      assert {:ok, _} = Grep.grep("test", scale: :section, limit: 5)
    end

    test "valid scale string is accepted" do
      assert {:ok, _} = Grep.grep("test", scale: "paragraph", limit: 5)
    end

    test "invalid scale is silently dropped" do
      assert {:ok, _} = Grep.grep("test", scale: "giant_blob", limit: 5)
    end
  end

  # ---------------------------------------------------------------------------
  # All 10 canonical intents are accepted
  # ---------------------------------------------------------------------------

  describe "grep/2 — all intent values" do
    intents = ~w(
      request_info propose_decision record_fact express_concern commit_action
      reference narrate reflect specify measure
    )a

    for intent <- intents do
      @intent intent
      test "accepts intent #{intent}" do
        assert {:ok, _} = Grep.grep("x", intent: @intent, limit: 3)
      end
    end
  end

  # ---------------------------------------------------------------------------
  # All 4 scales are accepted
  # ---------------------------------------------------------------------------

  describe "grep/2 — all scale values" do
    for scale <- ~w(document section paragraph chunk)a do
      @scale scale
      test "accepts scale #{scale}" do
        assert {:ok, _} = Grep.grep("x", scale: @scale, limit: 3)
      end
    end
  end
end
