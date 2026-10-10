defmodule OptimalEngine.Wiki.CuratorTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Wiki.{Curator, Page}

  defp base_page do
    %Page{
      tenant_id: "default",
      slug: "test",
      audience: "default",
      version: 1,
      frontmatter: %{"slug" => "test"},
      body: """
      # Test page

      Summary text here.
      """
    }
  end

  describe "curate/3 — deterministic path" do
    test "no new citations → no-op outcome" do
      outcome = Curator.curate(base_page(), [], deterministic: true)
      assert outcome.ok?
      assert outcome.metadata.reason == :no_new_citations
      # Page is unchanged when there's nothing to integrate
      assert outcome.page.body == base_page().body
    end

    test "appends a `## New signals` section with citation directives" do
      cites = [
        %{chunk_id: "c1", text: "New fact one.", uri: "optimal://a#c1"},
        %{chunk_id: "c2", text: "New fact two.", uri: "optimal://b#c2"}
      ]

      outcome = Curator.curate(base_page(), cites, deterministic: true)

      assert outcome.ok?
      assert String.contains?(outcome.page.body, "## New signals")
      assert String.contains?(outcome.page.body, "{{cite: optimal://a#c1}}")
      assert String.contains?(outcome.page.body, "{{cite: optimal://b#c2}}")
    end

    test "bumps version + records curated_by + last_curated" do
      outcome =
        Curator.curate(
          base_page(),
          [%{chunk_id: "c1", text: "fact", uri: "optimal://x"}],
          deterministic: true
        )

      assert outcome.page.version == 2
      assert String.starts_with?(outcome.page.curated_by, "deterministic:")
      assert is_binary(outcome.page.last_curated)
    end

    test "replaces existing `## New signals` section rather than appending twice" do
      page_with_existing = %{
        base_page()
        | body: """
          # Test page

          Summary.

          ## New signals

          - Old fact {{cite: optimal://old}}
          """
      }

      cites = [%{chunk_id: "c1", text: "New fact.", uri: "optimal://new"}]
      outcome = Curator.curate(page_with_existing, cites, deterministic: true)

      # Old citation removed, new citation present
      refute String.contains?(outcome.page.body, "optimal://old")
      assert String.contains?(outcome.page.body, "optimal://new")

      # Only one "New signals" header survives
      matches = Regex.scan(~r/## New signals/, outcome.page.body)
      assert length(matches) == 1
    end
  end
end
