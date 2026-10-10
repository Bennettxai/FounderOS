defmodule OptimalEngine.Pipeline do
  @moduledoc """
  Unified pipeline orchestrator — chains Stages 2→5 into a single call.

      file path
        │
        ▼
      [2] Parser.parse           → ParsedDoc
        │
        ▼
      [2.5] VlmEnricher.enrich  → ParsedDoc (enriched)
        │
        ▼
      [3] Decomposer.decompose  → ChunkTree
        │
        ▼
      [5] Embedder.embed_tree   → [Embedding]

  Each stage degrades gracefully. The result struct carries everything
  downstream stages need (chunks, embeddings, errors, metadata).

  ## Usage

      {:ok, result} = Pipeline.run("path/to/file.mp4")
      result.tree      # ChunkTree with all chunks
      result.embeddings # [Embedding] — 768-dim vectors
      result.parsed     # enriched ParsedDoc
  """

  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Pipeline.{Decomposer, Embedder, Parser}
  alias OptimalEngine.Pipeline.Parser.Asset
  alias OptimalEngine.Pipeline.Enricher.VlmEnricher

  require Logger

  @type result :: %{
          parsed: Parser.ParsedDoc.t(),
          tree: Decomposer.ChunkTree.t(),
          embeddings: [Embedder.Embedding.t()],
          embed_errors: [{String.t(), term()}],
          asset_paths: %{String.t() => String.t()}
        }

  @doc """
  Run the full parse → enrich → decompose → embed pipeline on a file.

  ## Options
  - `:preserve_assets` — copy parser assets into governed AssetStore (default `true`)
  - `:skip_vlm` — skip VLM enrichment (default `false`)
  - `:skip_embed` — skip embedding (default `false`)
  - `:window_bytes` — chunk window size (default 2048)
  - `:overlap_bytes` — chunk overlap (default 256)
  """
  @spec run(String.t(), keyword()) :: {:ok, result()} | {:error, term()}
  def run(path, opts \\ []) when is_binary(path) do
    with {:ok, parsed} <- Parser.parse(path, opts),
         {:ok, governed} <- maybe_preserve_assets(parsed, opts),
         {:ok, enriched} <- VlmEnricher.enrich(governed, opts),
         {:ok, tree} <- Decomposer.decompose(enriched, opts) do
      asset_paths = build_asset_paths(enriched)

      if Keyword.get(opts, :skip_embed, false) do
        {:ok,
         %{
           parsed: enriched,
           tree: tree,
           embeddings: [],
           embed_errors: [],
           asset_paths: asset_paths
         }}
      else
        embed_opts = Keyword.put(opts, :asset_paths, asset_paths)
        {:ok, embeddings, %{errors: errors}} = Embedder.embed_tree(tree, embed_opts)

        {:ok,
         %{
           parsed: enriched,
           tree: tree,
           embeddings: embeddings,
           embed_errors: errors,
           asset_paths: asset_paths
         }}
      end
    end
  end

  defp build_asset_paths(%{assets: assets}) when is_list(assets) do
    assets
    |> Enum.filter(fn a -> is_binary(a.hash) and is_binary(a.path) end)
    |> Enum.flat_map(fn asset ->
      refs =
        [asset.hash, metadata_ref(asset, :asset_id), metadata_ref(asset, "asset_id")]
        |> Enum.filter(&is_binary/1)
        |> Enum.uniq()

      Enum.map(refs, &{&1, asset.path})
    end)
    |> Map.new()
  end

  defp build_asset_paths(_), do: %{}

  defp maybe_preserve_assets(parsed, opts) do
    if Keyword.get(opts, :preserve_assets, true) do
      preserve_assets(parsed, opts)
    else
      {:ok, parsed}
    end
  end

  defp preserve_assets(%Parser.ParsedDoc{assets: assets} = parsed, opts) when is_list(assets) do
    Enum.reduce_while(assets, {:ok, []}, fn
      %Asset{} = asset, {:ok, acc} ->
        case MemoryCore.store_asset(asset, opts) do
          {:ok, %{asset: stored_asset, source_package: source_package}} ->
            {:cont, {:ok, [governed_asset(asset, stored_asset, source_package) | acc]}}

          {:error, reason} ->
            {:halt, {:error, {:asset_preservation_failed, reason}}}
        end

      asset, {:ok, acc} ->
        {:cont, {:ok, [asset | acc]}}
    end)
    |> case do
      {:ok, governed_assets} -> {:ok, %{parsed | assets: Enum.reverse(governed_assets)}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp preserve_assets(parsed, _opts), do: {:ok, parsed}

  defp governed_asset(%Asset{} = asset, stored_asset, source_package) do
    original_path = asset.path
    metadata = asset.metadata || %{}

    %{
      asset
      | path: stored_asset.storage_path,
        metadata:
          Map.merge(metadata, %{
            asset_id: stored_asset.id,
            source_package_id: source_package.id,
            original_path: original_path,
            stored_path: stored_asset.storage_path,
            content_hash: stored_asset.content_hash,
            trust_label: stored_asset.trust_label,
            retention_class: stored_asset.retention_class
          })
    }
  end

  defp metadata_ref(%Asset{metadata: metadata}, key) when is_map(metadata),
    do: Map.get(metadata, key)

  defp metadata_ref(_asset, _key), do: nil
end
