defmodule OptimalEngine.Retrieval.WikiFirst do
  @moduledoc """
  Tier-3 pre-check that runs *before* hybrid retrieval.

  The wiki is the curated front door: one well-maintained page can
  obviate thousands of chunk retrievals. `lookup/3` asks one question —
  "does a curated page already answer this query for this audience?" —
  and returns either the matching page or `:miss` so the orchestrator
  (`OptimalEngine.Retrieval.RAG`) can fall through to Tier 2.

  ## Match strategy

  1. **Slug-exact**: the query (slugified) matches a page `slug`.
  2. **Slug-contains**: a page slug contains the slugified query, or
     vice versa. Useful when the user types a shorter variant.
  3. **Title-substring**: frontmatter `title` contains the query
     (case-insensitive). Handy when slugs are opaque ids.

  Audience fallback: we look for `audience` first, then fall back to
  `"default"` when no audience-specific page exists.

  ## Cost

  This is a SQL scan bounded by `list/1` on `wiki_pages` (~hundreds
  of rows per tenant in practice). For large tenants we would index
  slug + title in FTS5 — deferred until the wiki grows past ~1K pages.
  """

  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Page

  @type lookup_result ::
          {:hit, Page.t(), match_reason :: atom()} | :miss

  @doc """
  Look up a curated wiki page for `query` in the given `audience`.

  Options:
    * `:tenant_id` — defaults to `Tenancy.Tenant.default_id/0`
    * `:workspace_id` — scope to a workspace (default `"default"`)
    * `:fallback_audience` — audience to try on miss (default `"default"`)
  """
  @spec lookup(String.t(), String.t(), keyword()) :: lookup_result()
  def lookup(query, audience \\ "default", opts \\ []) when is_binary(query) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    fallback = Keyword.get(opts, :fallback_audience, "default")
    normalized = slugify(query)

    with {:ok, pages} <- Wiki.list(tenant_id, workspace_id) do
      case match(Enum.filter(pages, &(&1.audience == audience)), query, normalized) do
        :miss when audience != fallback ->
          match(Enum.filter(pages, &(&1.audience == fallback)), query, normalized)

        result ->
          result
      end
    else
      _ -> :miss
    end
  end

  @doc """
  Same as `lookup/3` but returns a list of hits ranked best-first,
  capped by `:limit`. Useful when multiple curated pages overlap.
  """
  @spec lookup_many(String.t(), String.t(), keyword()) :: [Page.t()]
  def lookup_many(query, audience \\ "default", opts \\ []) when is_binary(query) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    fallback = Keyword.get(opts, :fallback_audience, "default")
    limit = Keyword.get(opts, :limit, 5)
    normalized = slugify(query)

    case Wiki.list(tenant_id, workspace_id) do
      {:ok, pages} ->
        primary = score_and_rank(pages, audience, query, normalized)

        cond do
          primary != [] ->
            Enum.take(primary, limit)

          audience != fallback ->
            pages
            |> score_and_rank(fallback, query, normalized)
            |> Enum.take(limit)

          true ->
            []
        end

      _ ->
        []
    end
  end

  defp score_and_rank(pages, audience, query, normalized) do
    pages
    |> Enum.filter(&(&1.audience == audience))
    |> Enum.map(fn p -> {score(p, query, normalized), p} end)
    |> Enum.filter(fn {score, _} -> score > 0 end)
    |> Enum.sort_by(fn {score, _} -> score end, :desc)
    |> Enum.map(fn {_, p} -> p end)
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp match([], _query, _normalized), do: :miss

  defp match(pages, query, normalized) do
    pages
    |> Enum.map(fn p -> {score(p, query, normalized), p} end)
    |> Enum.max_by(fn {s, _} -> s end, fn -> {0, nil} end)
    |> case do
      {0, _} -> :miss
      {score, page} -> {:hit, page, reason_for(score)}
    end
  end

  # Minimum normalized-query length before a substring match is allowed to
  # score. `"ai"` would otherwise hit `"project-platform-launch-pricing"`, `"ai-strategy"`,
  # `"ai-workflow"`, …; any of them could short-circuit hybrid retrieval
  # under the front-door contract. Four chars rules out single acronyms +
  # stop-word sized queries while still catching real slug stems.
  @min_substring_len 4

  # Scoring: higher is better. Exact slug = 100. Prefix = 50. Contains = 10
  # (only when both sides are long enough to be discriminative). Title
  # substring = 10. Missing = 0. A prefix match on a different slug always
  # beats a contains match on any slug.
  defp score(%Page{slug: slug} = page, query, normalized) do
    slug_score =
      cond do
        slug == normalized ->
          100

        String.starts_with?(slug, normalized) or String.starts_with?(normalized, slug) ->
          50

        byte_size(normalized) >= @min_substring_len and
          byte_size(slug) >= @min_substring_len and
            (String.contains?(slug, normalized) or String.contains?(normalized, slug)) ->
          10

        true ->
          0
      end

    title_score =
      case title_of(page) do
        nil ->
          0

        title ->
          t_lower = String.downcase(title)
          q_lower = String.downcase(query)

          cond do
            t_lower == q_lower -> 40
            byte_size(q_lower) >= @min_substring_len and String.contains?(t_lower, q_lower) -> 10
            true -> 0
          end
      end

    slug_score + title_score
  end

  defp title_of(%Page{frontmatter: fm}) when is_map(fm) do
    fm["title"] || fm[:title]
  end

  defp title_of(_), do: nil

  defp reason_for(s) when s >= 100, do: :slug_exact
  defp reason_for(s) when s >= 50, do: :slug_prefix
  defp reason_for(s) when s >= 10, do: :slug_contains
  defp reason_for(_), do: :title_substring

  # Lowercase, collapse non-alphanumerics to hyphens, trim, dedupe.
  defp slugify(text) do
    text
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9]+/, "-")
    |> String.trim("-")
  end
end
