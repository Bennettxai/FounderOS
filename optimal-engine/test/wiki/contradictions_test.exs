defmodule OptimalEngine.Wiki.ContradictionsTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Wiki.{Integrity, Page}

  # ── Helpers ────────────────────────────────────────────────────────────────

  defp page(body, workspace_id \\ "default") do
    %Page{
      tenant_id: "default",
      workspace_id: workspace_id,
      slug: "test-contradictions",
      audience: "default",
      version: 1,
      frontmatter: %{},
      body: body
    }
  end

  # ── check_contradictions/2 ─────────────────────────────────────────────────

  describe "check_contradictions/2" do
    test "same entity with two different price values → contradiction flagged" do
      body = """
      **Example Widget** costs $2K in the enterprise tier. {{cite: optimal://a#c1}}

      According to the updated sheet, **Example Widget** is priced at $2.4K. {{cite: optimal://b#c2}}
      """

      result = Integrity.check_contradictions(page(body))

      assert length(result) == 1
      [clash] = result
      assert clash.type == :entity_attr_clash
      assert clash.entity == "example widget"
      assert clash.attr == :numeric
      assert length(clash.claims) == 2

      values = Enum.map(clash.claims, & &1.value) |> Enum.sort()
      assert "$2.4K" in values
      assert "$2K" in values
    end

    test "same entity with consistent values → no contradiction" do
      body = """
      **Example Widget** costs $2K. {{cite: optimal://a#c1}}

      The **Example Widget** enterprise tier is $2K per seat. {{cite: optimal://b#c2}}
      """

      result = Integrity.check_contradictions(page(body))
      assert result == []
    end

    test "different entities with different values → no contradiction" do
      body = """
      **Widget A** is priced at $1K. {{cite: optimal://a#c1}}

      **Widget B** costs $5K. {{cite: optimal://b#c2}}
      """

      result = Integrity.check_contradictions(page(body))
      assert result == []
    end

    test "same entity with two different dates → date contradiction flagged" do
      body = """
      **Project Alpha** launches on 2026-03-01. {{cite: optimal://a#c1}}

      The **Project Alpha** launch is scheduled for 2026-06-15. {{cite: optimal://b#c2}}
      """

      result = Integrity.check_contradictions(page(body))

      assert length(result) == 1
      [clash] = result
      assert clash.attr == :date
      assert clash.entity == "project alpha"
    end

    test "no cited passages → no contradictions" do
      body = """
      **Widget X** costs $2K.

      **Widget X** costs $3K.
      """

      result = Integrity.check_contradictions(page(body))
      # No {{cite:...}} directives — no claimed pairs extracted.
      assert result == []
    end

    test "each claim has a non-empty hash and the citation URI" do
      body = """
      **Bolt** is $500. {{cite: optimal://source-a#c1}}

      **Bolt** costs $750. {{cite: optimal://source-b#c2}}
      """

      [clash] = Integrity.check_contradictions(page(body))

      Enum.each(clash.claims, fn claim ->
        assert is_binary(claim.hash) and byte_size(claim.hash) == 12
        assert String.starts_with?(claim.citation, "optimal://")
      end)
    end
  end

  # ── Integrity.check/2 with contradiction opts ──────────────────────────────

  describe "Integrity.check/2 — contradiction integration" do
    test "report includes :contradictions key even when none found" do
      body = "Summary. {{cite: optimal://x#c1}}"
      report = Integrity.check(page(body))
      assert Map.has_key?(report, :contradictions)
      assert report.contradictions == []
    end

    test "contradiction flagged → report has non-empty :contradictions" do
      body = """
      **Turbo** costs $2K here. {{cite: optimal://a#c1}}

      **Turbo** is $4K according to new data. {{cite: optimal://b#c2}}
      """

      report = Integrity.check(page(body))
      assert length(report.contradictions) == 1
    end

    test "detect_contradictions: false → contradictions not checked" do
      body = """
      **Turbo** costs $2K here. {{cite: optimal://a#c1}}

      **Turbo** is $4K according to new data. {{cite: optimal://b#c2}}
      """

      report = Integrity.check(page(body), detect_contradictions: false)
      assert report.contradictions == []
    end

    test "policy=flag_for_review → ok? remains true despite contradictions" do
      body = """
      **Turbo** costs $2K here. {{cite: optimal://a#c1}}

      **Turbo** is $4K according to new data. {{cite: optimal://b#c2}}
      """

      # Default policy (no config file) is flag_for_review.
      report = Integrity.check(page(body))
      # ok? is not invalidated by contradictions under flag_for_review
      assert report.ok?
      assert length(report.contradictions) == 1
    end

    test "policy=reject → ok? is false when contradictions present" do
      body = """
      **Turbo** costs $2K here. {{cite: optimal://a#c1}}

      **Turbo** is $4K according to new data. {{cite: optimal://b#c2}}
      """

      # Config.get_section("default", :contradictions, ..., root) looks in
      # <root>/.optimal/config.yaml (because "default" slug → root itself).
      tmp_dir = System.tmp_dir!() |> Path.join("oe_test_#{:rand.uniform(99999)}")
      optimal_dir = Path.join(tmp_dir, ".optimal")
      File.mkdir_p!(optimal_dir)
      File.write!(Path.join(optimal_dir, "config.yaml"), "contradictions:\n  policy: reject\n")

      report =
        Integrity.check(
          page(body, "default"),
          workspace_slug: "default",
          config_root: tmp_dir
        )

      refute report.ok?
      assert length(report.contradictions) == 1

      File.rm_rf!(tmp_dir)
    end

    test "policy=silent_resolve → each contradiction collapses to one claim" do
      body = """
      **Turbo** costs $2K here. {{cite: optimal://a#c1}}

      **Turbo** is $4K according to new data. {{cite: optimal://b#c2}}
      """

      tmp_dir = System.tmp_dir!() |> Path.join("oe_test_#{:rand.uniform(99999)}")
      optimal_dir = Path.join(tmp_dir, ".optimal")
      File.mkdir_p!(optimal_dir)

      File.write!(
        Path.join(optimal_dir, "config.yaml"),
        "contradictions:\n  policy: silent_resolve\n"
      )

      report =
        Integrity.check(
          page(body, "default"),
          workspace_slug: "default",
          config_root: tmp_dir
        )

      # silent_resolve collapses each contradiction to the last (newest) claim.
      assert report.ok?
      [resolved] = report.contradictions
      assert length(resolved.claims) == 1

      File.rm_rf!(tmp_dir)
    end
  end

  # ── existing check/2 regression guard ─────────────────────────────────────
  # These mirror the original tests to ensure we didn't break anything.

  describe "regression — existing Integrity.check/2 behaviour" do
    test "empty body still flagged as error" do
      report = Integrity.check(page(""))
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :empty_page))
    end

    test "unknown verb still flagged" do
      report = Integrity.check(page("A {{bogus: arg}} directive."))
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :invalid_verb))
    end

    test "broken citation still flagged" do
      body = "A fact. {{cite: optimal://missing}}"

      resolver = fn uri ->
        if String.contains?(uri, "missing"), do: {:error, :not_found}, else: :ok
      end

      report = Integrity.check(page(body), resolve_uri: resolver)
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :broken_citation))
    end
  end
end
