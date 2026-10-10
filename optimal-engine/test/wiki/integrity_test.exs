defmodule OptimalEngine.Wiki.IntegrityTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Wiki.{Integrity, Page}

  defp page(body, fm \\ %{}) do
    %Page{
      tenant_id: "default",
      slug: "test-page",
      audience: "default",
      version: 1,
      frontmatter: fm,
      body: body
    }
  end

  describe "check/2" do
    test "empty body is flagged as error" do
      report = Integrity.check(page(""))
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :empty_page))
    end

    test "body using an unknown verb is flagged" do
      report = Integrity.check(page("This has a {{bogus: arg}} directive."))
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :invalid_verb))
    end

    test "broken citations flagged via resolver" do
      body = "A fact. {{cite: optimal://missing}}"

      resolver = fn uri ->
        if String.contains?(uri, "missing"), do: {:error, :not_found}, else: :ok
      end

      report = Integrity.check(page(body), resolve_uri: resolver)
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :broken_citation))
    end

    test "large substantive paragraph without citation surfaces uncited_claim warning" do
      big_para =
        String.duplicate(
          "The quarterly performance review showed several interesting findings across verticals. ",
          5
        )

      report = Integrity.check(page(big_para))
      assert Enum.any?(report.issues, &(&1.kind == :uncited_claim))
    end

    test "short headings + short content don't trigger uncited_claim" do
      body = """
      ## Summary

      Short intro.

      ## Related
      """

      report = Integrity.check(page(body))
      refute Enum.any?(report.issues, &(&1.kind == :uncited_claim))
    end

    test "page over 50KB produces a page_too_large warning (not error)" do
      big = String.duplicate("A ", 30_000) <> " {{cite: optimal://x}}"

      resolver = fn _ -> :ok end
      report = Integrity.check(page(big), resolve_uri: resolver)

      assert Enum.any?(report.issues, &(&1.kind == :page_too_large))
      # warning, not error — ok? stays true unless there's an error
      assert report.ok?
    end
  end

  describe "against_schema/3" do
    test "flags missing required sections" do
      schema = %{"required_sections" => ["Summary", "Open threads"]}
      body = "## Summary\n\nContent. {{cite: optimal://x}}"
      resolver = fn _ -> :ok end

      report = Integrity.against_schema(page(body), schema, resolve_uri: resolver)
      refute report.ok?
      assert Enum.any?(report.issues, &(&1.kind == :missing_section))
    end

    test "honors schema size ceiling" do
      schema = %{"max_bytes" => 100}
      body = String.duplicate("x", 200)
      report = Integrity.against_schema(page(body), schema)
      assert Enum.any?(report.issues, &(&1.kind == :schema_size_exceeded))
    end

    test "flags missing required frontmatter" do
      schema = %{"required_frontmatter" => ["slug", "audience", "version", "last_curated"]}
      body = "## Summary\n\nContent."

      fm = %{"slug" => "t", "audience" => "default", "version" => 1}
      # Missing last_curated
      report = Integrity.against_schema(page(body, fm), schema)
      assert Enum.any?(report.issues, &(&1.kind == :missing_frontmatter))
    end
  end
end
