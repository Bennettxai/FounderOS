defmodule OptimalEngine.Wiki.Store do
  @moduledoc """
  Store helpers for Tier-3 wiki pages and their citations.

  Persistence lives in two Phase 1 tables:

    * `wiki_pages` — one row per `(tenant_id, slug, audience, version)`
    * `citations` — one row per `(tenant_id, wiki_slug, wiki_audience, chunk_id, claim_hash)`

  This module converts between the `%Page{}` struct and the row tuples,
  handles version bumps, and provides read helpers for the Curator + CLI.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Wiki.Page

  @doc "Upsert a wiki page (new version row)."
  @spec put(Page.t()) :: :ok | {:error, term()}
  def put(%Page{} = page) do
    frontmatter_json = Jason.encode!(page.frontmatter || %{})

    # `last_curated` is NOT NULL in the schema, but callers may pass nil for
    # a brand-new page. COALESCE to `datetime('now')` so the DEFAULT still
    # applies when the caller doesn't set one explicitly. Also writes
    # `workspace_id` so Phase 1.6 isolation queries find this page.
    sql = """
    INSERT INTO wiki_pages (tenant_id, workspace_id, slug, audience, version, frontmatter, body, last_curated, curated_by)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, COALESCE(?8, datetime('now')), ?9)
    ON CONFLICT(tenant_id, slug, audience, version) DO UPDATE SET
      workspace_id = excluded.workspace_id,
      frontmatter  = excluded.frontmatter,
      body         = excluded.body,
      last_curated = COALESCE(excluded.last_curated, datetime('now')),
      curated_by   = excluded.curated_by
    """

    workspace_id = Map.get(page, :workspace_id) || "default"

    params = [
      page.tenant_id,
      workspace_id,
      page.slug,
      page.audience,
      page.version,
      frontmatter_json,
      page.body,
      page.last_curated,
      page.curated_by
    ]

    case Store.raw_query(sql, params) do
      {:ok, _} ->
        notify_surfacer(page)
        :ok

      other ->
        other
    end
  end

  # Phase 15 hook — every successful wiki write fans out to the Surfacer
  # so any matching subscription gets a proactive push. Workspace_id will
  # come off the Page struct once the wiki tier is workspace-aware; for
  # now every write is attributed to the default workspace.
  defp notify_surfacer(%Page{} = page) do
    workspace_id = Map.get(page, :workspace_id) || OptimalEngine.Workspace.default_id()
    body_preview = page.body |> to_string() |> String.slice(0, 500)

    OptimalEngine.Memory.Surfacer.notify_wiki_updated(
      workspace_id,
      page.slug,
      audience: page.audience || "default",
      body_preview: body_preview,
      entities: extract_entities(page.frontmatter)
    )

    maybe_extract_memories(page, workspace_id)
  rescue
    # Never let a Surfacer failure block a curator write — curation is
    # the load-bearing path; surfacing is opportunistic.
    e ->
      require Logger
      Logger.warning("[Wiki.Store] Surfacer notify failed: #{inspect(e)}")
  end

  # Phase 17.1/7 bridge — if `memory.extract_from_wiki: true` in workspace
  # config, fire WikiBridge.extract_from_wiki_page/2 in a detached task so
  # it cannot block the wiki write path.
  defp maybe_extract_memories(%Page{} = page, workspace_id) do
    require Logger
    alias OptimalEngine.Workspace.Config

    # Resolve the workspace slug for config lookup.  `workspace_id` here is
    # the raw value from the Page struct which may be an id or a slug; Config
    # accepts either form via get_section/2.
    mem_cfg = Config.get_section(workspace_id, :memory, %{extract_from_wiki: false})

    if Map.get(mem_cfg, :extract_from_wiki, false) do
      Task.start(fn ->
        opts = [
          workspace_id: workspace_id,
          tenant_id: page.tenant_id || "default"
        ]

        case OptimalEngine.Memory.WikiBridge.extract_from_wiki_page(page, opts) do
          {:ok, ids} when ids != [] ->
            Logger.info(
              "[Wiki.Store] WikiBridge extracted #{length(ids)} memories from #{page.slug}"
            )

          {:ok, _} ->
            :ok

          {:error, reason} ->
            Logger.warning(
              "[Wiki.Store] WikiBridge.extract_from_wiki_page failed: #{inspect(reason)}"
            )
        end
      end)
    end

    :ok
  rescue
    e ->
      require Logger
      Logger.warning("[Wiki.Store] maybe_extract_memories error: #{inspect(e)}")
      :ok
  end

  defp extract_entities(%{entities: ents}) when is_list(ents) do
    Enum.map(ents, fn
      %{name: n} -> n
      n when is_binary(n) -> n
      _ -> nil
    end)
    |> Enum.reject(&is_nil/1)
  end

  defp extract_entities(_), do: []

  @doc """
  Fetch the latest version of a page by `(tenant_id, slug, audience)`.

  Optional `workspace_id` (defaults to `"default"`) scopes the lookup
  to a single workspace's wiki — Phase 1.6 isolation.
  """
  @spec latest(String.t(), String.t(), String.t(), String.t()) ::
          {:ok, Page.t()} | {:error, :not_found}
  def latest(tenant_id, slug, audience \\ "default", workspace_id \\ "default") do
    sql = """
    SELECT tenant_id, workspace_id, slug, audience, version, frontmatter, body, last_curated, curated_by
    FROM wiki_pages
    WHERE tenant_id = ?1 AND workspace_id = ?2 AND slug = ?3 AND audience = ?4
    ORDER BY version DESC
    LIMIT 1
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, slug, audience]) do
      {:ok, [row]} -> {:ok, row_to_page(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  List every page for a tenant + workspace (latest version of each
  (slug, audience) pair). Defaults to the `default` workspace for
  backwards compat with pre-Phase-1.6 callers.

  Options:
    - `:limit`  — max rows to return (default 50)
    - `:offset` — row offset for pagination (default 0)
  """
  @spec list(String.t(), String.t(), keyword()) :: {:ok, [Page.t()]}
  def list(tenant_id, workspace_id \\ "default", opts \\ []) do
    limit = Keyword.get(opts, :limit, 50)
    offset = Keyword.get(opts, :offset, 0)

    sql = """
    SELECT w.tenant_id, w.workspace_id, w.slug, w.audience, w.version, w.frontmatter, w.body, w.last_curated, w.curated_by
    FROM wiki_pages w
    INNER JOIN (
      SELECT tenant_id, workspace_id, slug, audience, MAX(version) AS max_version
      FROM wiki_pages
      WHERE tenant_id = ?1 AND workspace_id = ?2
      GROUP BY tenant_id, workspace_id, slug, audience
    ) latest
      ON w.tenant_id    = latest.tenant_id
     AND w.workspace_id = latest.workspace_id
     AND w.slug         = latest.slug
     AND w.audience     = latest.audience
     AND w.version      = latest.max_version
    ORDER BY w.slug, w.audience
    LIMIT ?3 OFFSET ?4
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, limit, offset]) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_page/1)}
      other -> other
    end
  end

  @doc """
  Counts the total number of unique (slug, audience) latest wiki pages
  for a tenant + workspace. Used for API pagination totals.
  """
  @spec count(String.t(), String.t()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(tenant_id, workspace_id \\ "default") do
    sql = """
    SELECT COUNT(*)
    FROM wiki_pages w
    INNER JOIN (
      SELECT tenant_id, workspace_id, slug, audience, MAX(version) AS max_version
      FROM wiki_pages
      WHERE tenant_id = ?1 AND workspace_id = ?2
      GROUP BY tenant_id, workspace_id, slug, audience
    ) latest
      ON w.tenant_id    = latest.tenant_id
     AND w.workspace_id = latest.workspace_id
     AND w.slug         = latest.slug
     AND w.audience     = latest.audience
     AND w.version      = latest.max_version
    """

    case Store.raw_query(sql, [tenant_id, workspace_id]) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, []} -> {:ok, 0}
      other -> other
    end
  end

  @doc "Upsert a batch of citations (bulk, atomic)."
  @spec put_citations(String.t(), String.t(), String.t(), [map()]) :: :ok | {:error, term()}
  def put_citations(tenant_id, wiki_slug, wiki_audience, citations)
      when is_binary(tenant_id) and is_binary(wiki_slug) and is_list(citations) do
    sql = """
    INSERT INTO citations
      (tenant_id, wiki_slug, wiki_audience, chunk_id, claim_hash, last_verified)
    VALUES (?1, ?2, ?3, ?4, ?5, datetime('now'))
    """

    Enum.reduce_while(citations, :ok, fn c, _acc ->
      case Store.raw_query(sql, [
             tenant_id,
             wiki_slug,
             wiki_audience,
             Map.get(c, :chunk_id) || Map.get(c, "chunk_id"),
             Map.get(c, :claim_hash) || Map.get(c, "claim_hash") || derive_claim_hash(c)
           ]) do
        {:ok, _} -> {:cont, :ok}
        {:error, _} = err -> {:halt, err}
      end
    end)
  end

  @doc "Delete all citations tied to a page (used before re-curating)."
  @spec clear_citations(String.t(), String.t(), String.t()) :: :ok
  def clear_citations(tenant_id, slug, audience) do
    Store.raw_query(
      "DELETE FROM citations WHERE tenant_id = ?1 AND wiki_slug = ?2 AND wiki_audience = ?3",
      [tenant_id, slug, audience]
    )

    :ok
  end

  defp row_to_page([
         tenant_id,
         workspace_id,
         slug,
         audience,
         version,
         fm_json,
         body,
         last_curated,
         curated_by
       ]) do
    frontmatter =
      case Jason.decode(fm_json || "{}") do
        {:ok, m} -> m
        _ -> %{}
      end

    %Page{
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      slug: slug,
      audience: audience,
      version: version,
      frontmatter: frontmatter,
      body: body,
      last_curated: last_curated,
      curated_by: curated_by
    }
  end

  defp derive_claim_hash(%{text: text}) when is_binary(text) do
    :crypto.hash(:sha256, text) |> Base.encode16(case: :lower) |> String.slice(0, 16)
  end

  defp derive_claim_hash(_), do: "unknown"
end
