defmodule OptimalEngine.Memory.WikiBridge do
  @moduledoc """
  Bridge between the Memory primitive (Phase 17.1) and the Wiki tier (Phase 7).

  ## Direction 1: Wiki → Memory (extract_from_wiki_page/2)

  When a wiki page is written and the workspace config has
  `memory.extract_from_wiki: true`, this module parses every
  `{{cite: <uri>}}` directive in the page body into a factual-claim
  candidate, deduplicates by SHA-256 claim hash, and calls
  `Memory.create/1` for each novel claim.  Memories get:

    - `is_static: true`
    - `citation_uri:` the chunk URI from the cite directive
    - `content:` the sentence/line that carried the cite directive
    - idempotent id derived from the claim hash (same claim → same memory)

  ## Direction 2: Memory → Wiki (promote_memory_to_wiki/3)

  Given a memory id, appends the memory content as a new entry under a
  `## Memory citations` section in the named wiki page, referencing the
  memory via an `optimal://memory/<id>` URI.  If the target page does not
  exist it is created with a minimal scaffold.

  ## Failure contract

  All public functions are safe to call from fire-and-forget Task.start
  contexts.  They log warnings on partial failure and return structured
  results — they never raise.
  """

  require Logger

  alias OptimalEngine.Memory
  alias OptimalEngine.Wiki.{Page, Store}

  @type extract_result :: {:ok, [String.t()]} | {:error, term()}
  @type promote_result :: {:ok, Page.t()} | {:error, term()}

  # Matches `{{cite: <uri>}}` anywhere in a line.
  @cite_re ~r/\{\{cite:\s*([^\}]+)\}\}/

  # ---------------------------------------------------------------------------
  # Direction 1: Wiki → Memory
  # ---------------------------------------------------------------------------

  @doc """
  Extract factual claims from a `%Wiki.Page{}` body and persist each as a
  static `Memory` row.

  Each line containing a `{{cite: <uri>}}` directive is treated as a claim.
  The claim text is the line with the directive stripped.  Claims are
  deduplicated by a SHA-256 hash of `{page_slug}:{claim_text}` so that
  re-curating the same page does not produce duplicate memories.

  Options:
    - `:workspace_id` — scope for the created memories (default: `"default"`)
    - `:tenant_id`    — tenant scope (default: `"default"`)

  Returns `{:ok, [memory_id]}` — the list of *newly* created memory ids
  (existing duplicates are not included).
  """
  @spec extract_from_wiki_page(Page.t(), keyword()) :: extract_result()
  def extract_from_wiki_page(%Page{} = page, opts \\ []) do
    workspace_id = Keyword.get(opts, :workspace_id, page.workspace_id || "default")
    tenant_id = Keyword.get(opts, :tenant_id, page.tenant_id || "default")

    candidates = parse_cite_candidates(page)

    created_ids =
      candidates
      |> Enum.reduce([], fn candidate, acc ->
        case create_memory_idempotent(candidate, workspace_id, tenant_id, page.audience) do
          {:created, mem_id} ->
            [mem_id | acc]

          {:existing, _} ->
            acc

          {:error, reason} ->
            Logger.warning(
              "[WikiBridge] Failed to create memory for claim from #{page.slug}: #{inspect(reason)}"
            )

            acc
        end
      end)
      |> Enum.reverse()

    {:ok, created_ids}
  rescue
    e ->
      Logger.warning("[WikiBridge] extract_from_wiki_page failed: #{inspect(e)}")
      {:error, e}
  end

  # ---------------------------------------------------------------------------
  # Direction 2: Memory → Wiki
  # ---------------------------------------------------------------------------

  @doc """
  Promote a memory to a wiki page by appending it as a citation entry.

  Fetches the memory by `memory_id`, then appends an entry under a
  `## Memory citations` section in the wiki page identified by `slug`.
  If the page does not exist for the given `workspace_id` / `tenant_id`,
  it is created fresh with a minimal scaffold.

  The memory is referenced via `optimal://memory/<id>`.

  Options:
    - `:workspace_id` — (default `"default"`)
    - `:tenant_id`    — (default `"default"`)
    - `:audience`     — page audience (default `"default"`)

  Returns `{:ok, page}` on success or `{:error, reason}` on failure.
  """
  @spec promote_memory_to_wiki(String.t(), String.t(), keyword()) :: promote_result()
  def promote_memory_to_wiki(memory_id, slug, opts \\ [])
      when is_binary(memory_id) and is_binary(slug) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    audience = Keyword.get(opts, :audience, "default")

    with {:ok, mem} <- Memory.get(memory_id) do
      page = fetch_or_scaffold(tenant_id, workspace_id, slug, audience)
      updated_page = append_memory_citation(page, mem)

      case Store.put(updated_page) do
        :ok -> {:ok, updated_page}
        error -> error
      end
    end
  rescue
    e ->
      Logger.warning("[WikiBridge] promote_memory_to_wiki failed for #{memory_id}: #{inspect(e)}")
      {:error, e}
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  # Parse every line in the page body that contains a {{cite: …}} directive.
  # Returns a list of %{claim_text, citation_uri, claim_hash} maps.
  defp parse_cite_candidates(%Page{slug: slug, body: body}) do
    (body || "")
    |> String.split("\n")
    |> Enum.flat_map(fn line ->
      case Regex.scan(@cite_re, line) do
        [] ->
          []

        matches ->
          # A line may have multiple cite directives; emit one candidate per URI.
          Enum.map(matches, fn [_full_match, uri] ->
            uri = String.trim(uri)
            claim_text = Regex.replace(@cite_re, line, "") |> String.trim()

            # strip leading list bullets / markdown formatting from claim
            claim_text = Regex.replace(~r/^[-*>]+\s*/, claim_text, "") |> String.trim()

            hash = claim_hash(slug, claim_text, uri)
            %{claim_text: claim_text, citation_uri: uri, claim_hash: hash}
          end)
      end
    end)
    |> Enum.reject(fn c -> c.claim_text == "" end)
  end

  # Idempotent create: if a memory with the same claim_hash metadata key
  # already exists in this workspace, skip it.
  defp create_memory_idempotent(
         %{claim_text: text, citation_uri: uri, claim_hash: hash},
         workspace_id,
         tenant_id,
         audience
       ) do
    # Check existing memories for this claim hash via metadata.
    existing =
      Memory.list(
        workspace_id: workspace_id,
        include_forgotten: false,
        include_old_versions: false,
        limit: 1000
      )
      |> Enum.find(fn m ->
        Map.get(m.metadata, "claim_hash") == hash
      end)

    if existing do
      {:existing, existing.id}
    else
      attrs = %{
        content: text,
        workspace_id: workspace_id,
        tenant_id: tenant_id,
        is_static: true,
        citation_uri: uri,
        audience: audience || "default",
        metadata: %{"claim_hash" => hash, "source" => "wiki_bridge"}
      }

      case Memory.create(attrs) do
        {:ok, mem} -> {:created, mem.id}
        {:error, reason} -> {:error, reason}
      end
    end
  end

  # Fetch the latest version of a page or build a scaffold if none exists.
  defp fetch_or_scaffold(tenant_id, workspace_id, slug, audience) do
    case Store.latest(tenant_id, slug, audience, workspace_id) do
      {:ok, page} ->
        page

      {:error, :not_found} ->
        %Page{
          tenant_id: tenant_id,
          workspace_id: workspace_id,
          slug: slug,
          audience: audience,
          version: 1,
          frontmatter: %{"slug" => slug, "audience" => audience},
          body: "# #{slug}\n\n",
          last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
          curated_by: "wiki_bridge"
        }
    end
  end

  # Append a memory citation under the `## Memory citations` section.
  # If the section already exists, appends within it. Otherwise creates it.
  defp append_memory_citation(%Page{} = page, mem) do
    uri = "optimal://memory/#{mem.id}"
    preview = mem.content |> String.slice(0, 200) |> String.replace(~r/\s+/, " ") |> String.trim()
    entry = "- #{preview} {{cite: #{uri}}}"

    updated_body =
      if String.contains?(page.body, "## Memory citations") do
        # Append before the next `##` heading or at the end of the section.
        Regex.replace(
          ~r/(## Memory citations\n)(.*?)(\n##|\z)/s,
          page.body,
          fn _, header, existing, tail ->
            existing_trimmed = String.trim_trailing(existing)
            "#{header}#{existing_trimmed}\n#{entry}\n#{tail}"
          end
        )
      else
        trimmed = String.trim_trailing(page.body)
        "#{trimmed}\n\n## Memory citations\n\n#{entry}\n"
      end

    %{
      page
      | body: updated_body,
        version: page.version + 1,
        last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
        curated_by: "wiki_bridge"
    }
  end

  # Deterministic SHA-256 hash of slug + claim_text + uri → 16-char hex prefix.
  defp claim_hash(slug, claim_text, uri) do
    :crypto.hash(:sha256, "#{slug}:#{claim_text}:#{uri}")
    |> Base.encode16(case: :lower)
    |> String.slice(0, 16)
  end
end
