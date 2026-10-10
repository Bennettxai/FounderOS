defmodule OptimalEngine.Batch do
  @moduledoc """
  Batch import/export operations for workspace migration and backup.

  Provides bulk ingestion of signals and memories into a workspace, and
  bulk export of signals, memories, wiki pages, and workspace config as
  JSON-serializable maps suitable for migration or API backup endpoints.

  ## Import behaviour

  Every item in an import batch is processed independently. A failure on
  one item never aborts the batch — errors are accumulated and returned
  in the summary counters. Callers receive `{:ok, %{imported, skipped,
  errors}}` regardless of per-item failures.

  ## Deduplication

  - Signals: `Intake.process/2` content-hashes each signal and returns
    `{:error, :duplicate}` for already-known content. Duplicates are
    counted as `:skipped`.
  - Memories: `Memory.create/1` uses a content hash per workspace; an
    existing memory returns `was_existing: true` which is counted as
    `:skipped`.

  ## Workspace_id flow

  The `:workspace_id` option is threaded into every downstream call so
  imported data lands in the correct workspace. Defaults to `"default"`.
  """

  require Logger

  alias OptimalEngine.Memory
  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Wiki
  alias OptimalEngine.Workspace

  @type import_summary :: %{
          imported: non_neg_integer(),
          skipped: non_neg_integer(),
          errors: non_neg_integer()
        }

  # ---------------------------------------------------------------------------
  # Import
  # ---------------------------------------------------------------------------

  @doc """
  Import a list of signal maps into a workspace.

  Each map may contain: `content` (required), `title`, `genre`, `node`,
  `entities`. Unknown keys are ignored.

  Uses `Intake.process/2` for each item. Duplicate content (as detected by
  the pipeline's content hash) is counted as `:skipped`. Any other error
  (including `:signal_too_noisy`) is counted as `:errors`.

  Options:
    - `:workspace_id` — target workspace (default: `"default"`)
  """
  @spec import_signals(list(map()), keyword()) ::
          {:ok, import_summary()} | {:error, term()}
  def import_signals(items, opts \\ []) when is_list(items) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    summary =
      Enum.reduce(items, %{imported: 0, skipped: 0, errors: 0}, fn item, acc ->
        case process_signal_item(item, workspace_id) do
          :imported -> Map.update!(acc, :imported, &(&1 + 1))
          :skipped -> Map.update!(acc, :skipped, &(&1 + 1))
          :error -> Map.update!(acc, :errors, &(&1 + 1))
        end
      end)

    {:ok, summary}
  end

  @doc """
  Import a list of memory maps into a workspace.

  Each map may contain: `content` (required), `is_static`, `audience`,
  `citation_uri`, `source_chunk_id`, `metadata`.

  Uses `Memory.create/1` for each item. Content-hash dedup is handled
  transparently — an existing memory returns `was_existing: true` which
  is counted as `:skipped`.

  Options:
    - `:workspace_id` — target workspace (default: `"default"`)
  """
  @spec import_memories(list(map()), keyword()) ::
          {:ok, import_summary()} | {:error, term()}
  def import_memories(items, opts \\ []) when is_list(items) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    summary =
      Enum.reduce(items, %{imported: 0, skipped: 0, errors: 0}, fn item, acc ->
        case process_memory_item(item, workspace_id) do
          :imported -> Map.update!(acc, :imported, &(&1 + 1))
          :skipped -> Map.update!(acc, :skipped, &(&1 + 1))
          :error -> Map.update!(acc, :errors, &(&1 + 1))
        end
      end)

    {:ok, summary}
  end

  # ---------------------------------------------------------------------------
  # Export
  # ---------------------------------------------------------------------------

  @doc """
  Export all signals (contexts of type 'signal') from a workspace as a
  list of JSON-serializable maps.

  Options:
    - `:workspace_id` — source workspace (default: `"default"`)
  """
  @spec export_signals(keyword()) :: {:ok, list(map())}
  def export_signals(opts \\ []) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, uri, title, genre, mode, signal_type, format, structure,
           node, sn_ratio, content, l0_abstract, l1_overview,
           modified_at, created_at, workspace_id
    FROM contexts
    WHERE workspace_id = ?1 AND type = 'signal'
    ORDER BY created_at ASC
    """

    signals =
      case Store.raw_query(sql, [workspace_id]) do
        {:ok, rows} -> Enum.map(rows, &signal_row_to_map/1)
        _ -> []
      end

    {:ok, signals}
  end

  @doc """
  Export all active memories (latest, non-forgotten) from a workspace as
  a list of JSON-serializable maps.

  Options:
    - `:workspace_id` — source workspace (default: `"default"`)
  """
  @spec export_memories(keyword()) :: {:ok, list(map())}
  def export_memories(opts \\ []) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, content, content_hash,
           is_static, is_forgotten, forget_after, forget_reason,
           version, parent_memory_id, root_memory_id, is_latest,
           citation_uri, source_chunk_id, audience, metadata,
           created_at, updated_at
    FROM memories
    WHERE workspace_id = ?1 AND is_latest = 1 AND is_forgotten = 0
    ORDER BY created_at ASC
    """

    memories =
      case Store.raw_query(sql, [workspace_id]) do
        {:ok, rows} -> Enum.map(rows, &memory_row_to_map/1)
        _ -> []
      end

    {:ok, memories}
  end

  @doc """
  Export a full workspace snapshot combining signals, memories, wiki pages,
  and workspace config into a single JSON-serializable map.

  Options:
    - `:tenant_id` — tenant id (default: the platform default tenant)
  """
  @spec export_workspace(String.t(), keyword()) :: {:ok, map()}
  def export_workspace(workspace_id, opts \\ []) when is_binary(workspace_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())

    {:ok, signals} = export_signals(workspace_id: workspace_id)
    {:ok, memories} = export_memories(workspace_id: workspace_id)
    wiki_pages = export_wiki_pages(workspace_id, tenant_id)
    config = export_workspace_config(workspace_id)

    snapshot = %{
      workspace_id: workspace_id,
      tenant_id: tenant_id,
      exported_at: DateTime.utc_now() |> DateTime.to_iso8601(),
      signals: signals,
      memories: memories,
      wiki: wiki_pages,
      config: config
    }

    {:ok, snapshot}
  end

  # ---------------------------------------------------------------------------
  # Private: import helpers
  # ---------------------------------------------------------------------------

  defp process_signal_item(item, workspace_id) when is_map(item) do
    content = Map.get(item, "content") || Map.get(item, :content)

    if not (is_binary(content) and content != "") do
      Logger.warning("[Batch] Skipping signal item — missing content: #{inspect(Map.keys(item))}")
      :error
    else
      intake_opts =
        [workspace_id: workspace_id]
        |> maybe_kw_put(:genre, item["genre"] || item[:genre])
        |> maybe_kw_put(:node, item["node"] || item[:node])
        |> maybe_kw_put(:title, item["title"] || item[:title])
        |> maybe_kw_put(:entities, item["entities"] || item[:entities])

      case Intake.process(content, intake_opts) do
        {:ok, _result} ->
          :imported

        {:error, :duplicate} ->
          :skipped

        {:error, reason} ->
          Logger.warning("[Batch] Signal import error: #{inspect(reason)}")
          :error
      end
    end
  rescue
    e ->
      Logger.warning("[Batch] Signal import exception: #{inspect(e)}")
      :error
  end

  defp process_signal_item(_item, _workspace_id), do: :error

  defp process_memory_item(item, workspace_id) when is_map(item) do
    content = Map.get(item, "content") || Map.get(item, :content)

    if not (is_binary(content) and content != "") do
      Logger.warning("[Batch] Skipping memory item — missing content")
      :error
    else
      attrs =
        %{content: content, workspace_id: workspace_id}
        |> maybe_map_put(:is_static, item["is_static"] || item[:is_static])
        |> maybe_map_put(:audience, item["audience"] || item[:audience])
        |> maybe_map_put(:citation_uri, item["citation_uri"] || item[:citation_uri])
        |> maybe_map_put(:source_chunk_id, item["source_chunk_id"] || item[:source_chunk_id])
        |> maybe_map_put(:metadata, item["metadata"] || item[:metadata])

      case Memory.create(attrs) do
        {:ok, %{was_existing: true}} ->
          :skipped

        {:ok, _mem} ->
          :imported

        {:error, reason} ->
          Logger.warning("[Batch] Memory import error: #{inspect(reason)}")
          :error
      end
    end
  rescue
    e ->
      Logger.warning("[Batch] Memory import exception: #{inspect(e)}")
      :error
  end

  defp process_memory_item(_item, _workspace_id), do: :error

  # ---------------------------------------------------------------------------
  # Private: export helpers
  # ---------------------------------------------------------------------------

  defp export_wiki_pages(workspace_id, tenant_id) do
    case Wiki.list(tenant_id, workspace_id) do
      {:ok, pages} ->
        Enum.map(pages, fn p ->
          %{
            slug: p.slug,
            audience: p.audience,
            version: p.version,
            body: p.body,
            last_curated: p.last_curated,
            curated_by: p.curated_by,
            frontmatter: p.frontmatter,
            workspace_id: p.workspace_id,
            tenant_id: p.tenant_id
          }
        end)

      _ ->
        []
    end
  end

  defp export_workspace_config(workspace_id) do
    with {:ok, ws} <- Workspace.get(workspace_id),
         {:ok, cfg} <- OptimalEngine.Workspace.Config.get(ws.slug) do
      cfg
    else
      _ -> %{}
    end
  end

  defp signal_row_to_map([
         id,
         uri,
         title,
         genre,
         mode,
         signal_type,
         format,
         structure,
         node,
         sn_ratio,
         content,
         l0_abstract,
         l1_overview,
         modified_at,
         created_at,
         workspace_id
       ]) do
    %{
      id: id,
      uri: uri,
      title: title,
      genre: genre,
      mode: mode,
      signal_type: signal_type,
      format: format,
      structure: structure,
      node: node,
      sn_ratio: sn_ratio,
      content: content,
      l0_abstract: l0_abstract,
      l1_overview: l1_overview,
      modified_at: modified_at,
      created_at: created_at,
      workspace_id: workspace_id
    }
  end

  defp memory_row_to_map([
         id,
         tenant_id,
         workspace_id,
         content,
         content_hash,
         is_static,
         is_forgotten,
         forget_after,
         forget_reason,
         version,
         parent_memory_id,
         root_memory_id,
         is_latest,
         citation_uri,
         source_chunk_id,
         audience,
         metadata_json,
         created_at,
         updated_at
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      content: content,
      content_hash: content_hash,
      is_static: is_static == 1,
      is_forgotten: is_forgotten == 1,
      forget_after: forget_after,
      forget_reason: forget_reason,
      version: version,
      parent_memory_id: parent_memory_id,
      root_memory_id: root_memory_id,
      is_latest: is_latest == 1,
      citation_uri: citation_uri,
      source_chunk_id: source_chunk_id,
      audience: audience,
      metadata: decode_json(metadata_json),
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(s) when is_binary(s) do
    case Jason.decode(s) do
      {:ok, m} -> m
      _ -> %{}
    end
  end

  defp maybe_kw_put(kw, _key, nil), do: kw
  defp maybe_kw_put(kw, _key, []), do: kw
  defp maybe_kw_put(kw, key, val), do: Keyword.put(kw, key, val)

  defp maybe_map_put(map, _key, nil), do: map
  defp maybe_map_put(map, key, val), do: Map.put(map, key, val)
end
