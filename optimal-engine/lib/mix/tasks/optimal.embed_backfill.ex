defmodule Mix.Tasks.Optimal.EmbedBackfill do
  @shortdoc "Backfill chunk_embeddings for chunks that have none (fixes 0% coverage)"

  @moduledoc """
  Embeds every stored chunk that is missing a `chunk_embeddings` row and
  persists the 768-dim vector, keyed to `chunk_id` in the current scope
  scheme (joinable to contexts via `chunks.signal_id`). This is the backfill
  that closes the chunk → embed → vector pipeline for data ingested before
  embedding was wired in.

  ## Behaviour

    * **Idempotent** — only processes chunks with no existing embedding, so
      re-running is safe and resumes where it left off.
    * **Batched + incremental** — embeds and commits one batch at a time, so
      a large backlog never blocks in a single huge transaction and partial
      progress survives an interrupt.
    * **Graceful** — chunks whose provider call fails (Ollama down, empty
      text, no asset) are counted as skipped and left for a later run.

  ## Usage

      mix optimal.embed_backfill
      mix optimal.embed_backfill --batch 100
      mix optimal.embed_backfill --workspace default --limit 500
      mix optimal.embed_backfill --dry-run

  ## Options

    * `--batch N`     — chunks per batch (default 50)
    * `--limit N`     — stop after N chunks total (default: all)
    * `--workspace W` — only backfill chunks in workspace W
    * `--dry-run`     — report what would be embedded; write nothing
  """

  use Mix.Task

  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}
  alias OptimalEngine.Pipeline.Embedder
  alias OptimalEngine.Store

  @default_batch 50

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, _, _} =
      OptionParser.parse(args,
        strict: [batch: :integer, limit: :integer, workspace: :string, dry_run: :boolean]
      )

    batch_size = Keyword.get(opts, :batch, @default_batch)
    limit = Keyword.get(opts, :limit)
    workspace = Keyword.get(opts, :workspace)
    dry_run = Keyword.get(opts, :dry_run, false)

    pending = pending_count(workspace)
    target = if limit, do: min(limit, pending), else: pending

    shell = Mix.shell()
    shell.info("Chunks missing embeddings: #{pending}")

    shell.info(
      "Backfilling: #{target} (batch=#{batch_size}#{if dry_run, do: ", dry-run", else: ""})"
    )

    if target == 0 do
      shell.info("Nothing to do — embedding coverage already complete.")
    else
      totals = backfill_loop(batch_size, target, workspace, dry_run, %{embedded: 0, skipped: 0})
      shell.info("")
      shell.info("Done. Embedded: #{totals.embedded}  Skipped: #{totals.skipped}")
    end
  end

  # ── batch loop ──────────────────────────────────────────────────────────

  defp backfill_loop(_batch_size, target, _ws, _dry, acc)
       when acc.embedded + acc.skipped >= target,
       do: acc

  defp backfill_loop(batch_size, target, workspace, dry_run, acc) do
    remaining = target - (acc.embedded + acc.skipped)
    take = min(batch_size, remaining)

    case fetch_chunks(take, workspace) do
      [] ->
        acc

      chunks ->
        {embedded, skipped} = process_batch(chunks, dry_run)
        acc = %{embedded: acc.embedded + embedded, skipped: acc.skipped + skipped}

        Mix.shell().info(
          "  …#{acc.embedded + acc.skipped}/#{target}  (+#{embedded} embedded, +#{skipped} skipped)"
        )

        backfill_loop(batch_size, target, workspace, dry_run, acc)
    end
  end

  defp process_batch(chunks, true) do
    # Dry run: report without embedding or writing.
    {0, length(chunks)}
  end

  defp process_batch(chunks, false) do
    structs = Enum.map(chunks, &to_chunk_struct/1)
    ws_by_id = Map.new(chunks, fn c -> {c.id, c.workspace_id} end)
    tree = %ChunkTree{root_chunk_id: nil, chunks: structs}

    {:ok, embeddings, %{errors: errors}} = Embedder.embed_tree(tree)

    rows =
      Enum.map(embeddings, fn emb ->
        emb
        |> Embedder.embedding_row(%{})
        |> Map.put(:workspace_id, Map.get(ws_by_id, emb.chunk_id, "default"))
      end)

    case Store.insert_embeddings(rows) do
      :ok ->
        {length(rows), length(errors)}

      {:error, reason} ->
        Mix.shell().error("  batch write failed: #{inspect(reason)}")
        {0, length(chunks)}
    end
  end

  # ── data access ───────────────────────────────────────────────────────────

  defp pending_count(workspace) do
    {sql, params} = pending_sql("COUNT(*)", workspace, nil)

    case Store.raw_query(sql, params) do
      {:ok, [[n] | _]} -> n
      _ -> 0
    end
  end

  # Pull the next batch of chunks lacking an embedding. Idempotent: re-running
  # never re-fetches an already-embedded chunk because the NOT EXISTS guard
  # excludes rows we just wrote in the previous batch.
  defp fetch_chunks(take, workspace) do
    cols = "c.id, c.tenant_id, c.signal_id, c.text, c.modality, c.asset_ref, c.workspace_id"
    {sql, params} = pending_sql(cols, workspace, take)

    case Store.raw_query(sql, params) do
      {:ok, rows} -> Enum.map(rows, &row_to_map/1)
      {:error, _} -> []
    end
  end

  defp pending_sql(select, nil, limit) do
    base = """
    SELECT #{select}
    FROM chunks c
    WHERE NOT EXISTS (
      SELECT 1 FROM chunk_embeddings e WHERE e.chunk_id = c.id
    )
      AND c.text <> ''
    """

    maybe_limit(base, [], limit)
  end

  defp pending_sql(select, workspace, limit) do
    base = """
    SELECT #{select}
    FROM chunks c
    WHERE NOT EXISTS (
      SELECT 1 FROM chunk_embeddings e WHERE e.chunk_id = c.id
    )
      AND c.text <> ''
      AND c.workspace_id = ?1
    """

    maybe_limit(base, [workspace], limit)
  end

  defp maybe_limit(sql, params, nil), do: {sql, params}

  defp maybe_limit(sql, params, limit),
    do: {sql <> "\nLIMIT #{limit}", params}

  defp row_to_map([id, tenant_id, signal_id, text, modality, asset_ref, workspace_id]) do
    %{
      id: id,
      tenant_id: tenant_id || "default",
      signal_id: signal_id,
      text: text || "",
      modality: modality || "text",
      asset_ref: asset_ref,
      workspace_id: workspace_id || "default"
    }
  end

  defp to_chunk_struct(c) do
    Chunk.new(
      id: c.id,
      tenant_id: c.tenant_id,
      signal_id: c.signal_id,
      text: c.text,
      modality: to_modality(c.modality),
      asset_ref: c.asset_ref
    )
  end

  defp to_modality(m) when is_atom(m), do: m

  defp to_modality(m) when is_binary(m) do
    case m do
      "text" -> :text
      "code" -> :code
      "data" -> :data
      "mixed" -> :mixed
      "image" -> :image
      "audio" -> :audio
      "video" -> :video
      _ -> :text
    end
  end
end
