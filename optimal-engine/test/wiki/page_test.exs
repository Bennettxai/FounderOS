defmodule OptimalEngine.Wiki.PageTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Wiki.Page

  describe "from_markdown/2" do
    test "parses YAML frontmatter and body" do
      md = """
      ---
      slug: ed-honour-pricing
      audience: sales
      version: 3
      last_curated: 2026-04-17T10:00:00Z
      ---

      # Alice pricing

      Body text here.
      """

      assert {:ok, %Page{} = p} = Page.from_markdown(md)
      assert p.slug == "ed-honour-pricing"
      assert p.audience == "sales"
      assert p.version == 3
      assert String.contains?(p.body, "Body text here.")
    end

    test "errors when slug is missing and none in opts" do
      md = "---\naudience: sales\n---\n\nBody"
      assert {:error, :missing_slug} = Page.from_markdown(md)
    end

    test "falls back to default values" do
      md = "---\nslug: test\n---\n\nBody"
      assert {:ok, p} = Page.from_markdown(md)
      assert p.audience == "default"
      assert p.version == 1
      assert p.tenant_id == "default"
    end

    test "handles markdown with no frontmatter when slug in opts" do
      assert {:ok, p} = Page.from_markdown("Just body", slug: "fallback")
      assert p.slug == "fallback"
      assert p.body == "Just body"
    end
  end

  describe "to_markdown/1" do
    test "serializes frontmatter and body round-trip" do
      page = %Page{
        slug: "test-slug",
        audience: "sales",
        version: 2,
        tenant_id: "default",
        frontmatter: %{"custom_key" => "value"},
        body: "# Body\n\nContent."
      }

      rendered = Page.to_markdown(page)
      assert {:ok, parsed} = Page.from_markdown(rendered)

      assert parsed.slug == "test-slug"
      assert parsed.audience == "sales"
      assert parsed.version == 2
      assert parsed.body =~ "Content."
    end
  end
end
