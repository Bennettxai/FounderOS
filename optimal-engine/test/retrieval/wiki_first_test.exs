defmodule OptimalEngine.Retrieval.WikiFirstTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Retrieval.WikiFirst
  alias OptimalEngine.Wiki.{Page, Store}

  defp seed_page(slug, audience, opts \\ []) do
    page = %Page{
      tenant_id: "default",
      slug: slug,
      audience: audience,
      version: Keyword.get(opts, :version, 1),
      frontmatter: Keyword.get(opts, :frontmatter, %{"slug" => slug}),
      body: Keyword.get(opts, :body, "## Summary\n\nHello {{cite: optimal://x}}.")
    }

    :ok = Store.put(page)
    page
  end

  describe "lookup/3" do
    test "returns :hit on exact slug match" do
      suffix = System.unique_integer([:positive])
      slug = "wikifirst-exact-#{suffix}"
      seed_page(slug, "default")

      assert {:hit, page, :slug_exact} = WikiFirst.lookup(slug, "default")
      assert page.slug == slug
    end

    test "slugifies the query — spaces and casing normalize away" do
      suffix = System.unique_integer([:positive])
      slug = "wikifirst-slugified-#{suffix}"
      seed_page(slug, "default")

      # The query exactly slugifies to the slug (when we replace - with space)
      query = String.replace(slug, "-", " ") |> String.upcase()
      assert {:hit, page, _} = WikiFirst.lookup(query, "default")
      assert page.slug == slug
    end

    test "returns :miss when nothing matches" do
      assert :miss = WikiFirst.lookup("totally-absent-#{System.unique_integer([:positive])}")
    end

    test "audience miss falls through to default audience" do
      suffix = System.unique_integer([:positive])
      slug = "wikifirst-audience-#{suffix}"
      seed_page(slug, "default")

      # No "sales" version exists, so default fallback wins.
      assert {:hit, page, _} = WikiFirst.lookup(slug, "sales")
      assert page.audience == "default"
    end

    test "audience-specific page wins over default when both exist" do
      suffix = System.unique_integer([:positive])
      slug = "wikifirst-priority-#{suffix}"
      seed_page(slug, "default", body: "DEFAULT BODY")
      seed_page(slug, "sales", body: "SALES BODY")

      assert {:hit, page, _} = WikiFirst.lookup(slug, "sales")
      assert page.audience == "sales"
      assert page.body == "SALES BODY"
    end

    test "title substring match on frontmatter title" do
      suffix = System.unique_integer([:positive])
      slug = "wikifirst-opaque-id-#{suffix}"

      seed_page(slug, "default", frontmatter: %{"slug" => slug, "title" => "Pricing Strategy Q4"})

      assert {:hit, page, _} = WikiFirst.lookup("pricing strategy", "default")
      assert page.slug == slug
    end
  end

  describe "lookup_many/3" do
    test "returns multiple hits ranked best-first" do
      suffix = System.unique_integer([:positive])
      prefix = "wikifirst-many-#{suffix}"

      seed_page("#{prefix}-a", "default")
      seed_page("#{prefix}-b", "default")

      hits = WikiFirst.lookup_many(prefix, "default", limit: 5)
      assert length(hits) >= 2
      assert Enum.all?(hits, &String.starts_with?(&1.slug, prefix))
    end

    test "respects :limit" do
      suffix = System.unique_integer([:positive])
      prefix = "wikifirst-limit-#{suffix}"

      for i <- 1..5, do: seed_page("#{prefix}-#{i}", "default")

      hits = WikiFirst.lookup_many(prefix, "default", limit: 2)
      assert length(hits) == 2
    end
  end
end
