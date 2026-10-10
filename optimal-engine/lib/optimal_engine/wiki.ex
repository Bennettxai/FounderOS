defmodule OptimalEngine.Wiki do
  @moduledoc """
  Top-level facade for Tier 3 — the LLM-maintained wiki layer.

  A wiki page is a curated, audience-aware summary with hot citations
  back to chunks (Tier 2) and raw sources (Tier 1). Pages are the
  agent's "front door": `mix optimal.rag` consults the wiki first and
  only falls through to hybrid retrieval on wiki miss.

  Public API:

      Wiki.get(tenant_id, slug, audience)        — read latest version
      Wiki.put(page)                              — persist a new version
      Wiki.list(tenant_id)                        — list every page
      Wiki.curate(slug, citations, opts)          — run the curator
      Wiki.render(page, resolver, format)         — render directives
      Wiki.verify(page)                           — run integrity checks
      Wiki.verify_against_schema(page, schema)    — schema-enforced checks
  """

  alias OptimalEngine.Wiki.{Curator, Directives, Integrity, Page, Store}

  defdelegate from_markdown(markdown, opts), to: Page
  defdelegate to_markdown(page), to: Page

  defdelegate put(page), to: Store

  @doc "Latest version of a page, optionally scoped by workspace_id."
  defdelegate latest(tenant_id, slug, audience), to: Store
  defdelegate latest(tenant_id, slug, audience, workspace_id), to: Store

  @doc "List wiki pages for a tenant, optionally scoped to a workspace."
  defdelegate list(tenant_id), to: Store
  defdelegate list(tenant_id, workspace_id), to: Store
  defdelegate list(tenant_id, workspace_id, opts), to: Store

  @doc "Count latest wiki pages for a tenant + workspace (for pagination)."
  defdelegate count(tenant_id), to: Store
  defdelegate count(tenant_id, workspace_id), to: Store

  @doc "Render a page's body, resolving every directive via the supplied resolver."
  @spec render(Page.t(), Directives.resolver(), keyword()) :: {String.t(), [String.t()]}
  def render(%Page{} = page, resolver, opts \\ []) do
    Directives.render(page.body, resolver, opts)
  end

  @doc "Verify a page's integrity (citations, directive verbs, page size, uncited claims)."
  @spec verify(Page.t(), keyword()) :: Integrity.report()
  def verify(%Page{} = page, opts \\ []), do: Integrity.check(page, opts)

  @doc "Verify against the `.wiki/SCHEMA.md` schema rules (required sections, max size, etc.)."
  @spec verify_against_schema(Page.t(), map(), keyword()) :: Integrity.report()
  def verify_against_schema(%Page{} = page, schema, opts \\ []) do
    Integrity.against_schema(page, schema, opts)
  end

  @doc "Curate a page with new citations. See `Curator.curate/3`."
  @spec curate(Page.t(), [Curator.citation()], keyword()) :: Curator.outcome()
  def curate(%Page{} = page, citations, opts \\ []) do
    Curator.curate(page, citations, opts)
  end

  @doc """
  Curate a page AND persist the result end-to-end.

  This closes the curation loop that `curate/3` alone leaves open: it runs the
  curator, persists the new page version via `Store.put/1`, and writes one
  `citations` row per cited chunk (the hot back-link from a wiki page to its
  Tier-2 chunks). Without this step the `{{cite:}}` markers in the page body
  have no queryable provenance rows backing them.

  Returns `{:ok, %{page: Page.t(), citations_persisted: non_neg_integer(),
  outcome: Curator.outcome()}}` or `{:error, reason}`.
  """
  @spec curate_and_persist(Page.t(), [Curator.citation()], keyword()) ::
          {:ok, map()} | {:error, term()}
  def curate_and_persist(%Page{} = page, citations, opts \\ []) when is_list(citations) do
    outcome = Curator.curate(page, citations, opts)

    if outcome.ok? do
      curated = outcome.page
      audience = Map.get(outcome.metadata, :audience, curated.audience)

      with :ok <- Store.put(curated),
           :ok <- Store.clear_citations(curated.tenant_id, curated.slug, audience),
           :ok <-
             Store.put_citations(
               curated.tenant_id,
               curated.slug,
               audience,
               citations
             ) do
        {:ok,
         %{
           page: curated,
           citations_persisted: length(citations),
           outcome: outcome
         }}
      end
    else
      {:error, {:curation_failed, outcome.metadata}}
    end
  end
end
