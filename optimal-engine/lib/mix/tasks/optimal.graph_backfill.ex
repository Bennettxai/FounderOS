defmodule Mix.Tasks.Optimal.GraphBackfill do
  @shortdoc "Backfill knowledge-graph edges from existing contexts and claims"

  @moduledoc """
  Builds the relationship-edge graph from the already-indexed contexts and the
  extracted claims, attributing every edge to the SIGNAL's real workspace.

  For each context this writes (all idempotent via `INSERT OR IGNORE`):

    * `entity --mentioned_in--> context`
    * `entity --co_occurs--> entity`  (both directions, for each shared pair)
    * `context --lives_in--> node`
    * `context --cross_ref--> node`   (extra routed-to destinations)
    * `context --supersedes--> context`

  For each claim it writes:

    * `claim --about--> entity`  (entities taken from the claim's source context)

  ## Behaviour

    * **Idempotent** — every insert is `INSERT OR IGNORE`, so re-running adds no
      duplicate edges. Safe to run repeatedly.
    * **Batched** — contexts are streamed in batches so a large backlog never
      builds one huge transaction.
    * **Workspace-correct** — each edge carries the workspace_id of the context
      it was derived from; no cross-workspace leakage.

  ## Usage

      mix optimal.graph_backfill
      mix optimal.graph_backfill --batch 100
      mix optimal.graph_backfill --workspace default
      mix optimal.graph_backfill --no-claims

  ## Options

    * `--batch N`     — contexts per batch (default 200)
    * `--workspace W` — only backfill edges for contexts in workspace W
    * `--no-claims`   — skip the claim->about->entity pass
  """

  use Mix.Task

  alias OptimalEngine.Graph
  alias OptimalEngine.Store

  @default_batch 200

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, _, _} =
      OptionParser.parse(args,
        strict: [batch: :integer, workspace: :string, claims: :boolean]
      )

    batch_size = Keyword.get(opts, :batch, @default_batch)
    workspace = Keyword.get(opts, :workspace)
    do_claims = Keyword.get(opts, :claims, true)

    shell = Mix.shell()

    before = edge_count(workspace)
    shell.info("Edges before: #{before}")

    ctx_total = backfill_contexts(batch_size, workspace, shell)
    shell.info("Context rows processed: #{ctx_total}")

    claim_total =
      if do_claims do
        n = backfill_claims(batch_size, workspace, shell)
        shell.info("Claim rows processed: #{n}")
        n
      else
        shell.info("Claim pass skipped (--no-claims)")
        0
      end

    after_count = edge_count(workspace)
    shell.info("")
    shell.info("Done. Edges after: #{after_count} (+#{after_count - before})")
    shell.info("Contexts: #{ctx_total}  Claims: #{claim_total}")
  end

  # ── context edges ─────────────────────────────────────────────────────────

  defp backfill_contexts(batch_size, workspace, shell) do
    {sql, params} = context_select(workspace)

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        rows
        |> Enum.chunk_every(batch_size)
        |> Enum.reduce(0, fn batch, acc ->
          Enum.each(batch, &edges_for_context_row/1)
          shell.info("  contexts: #{acc + length(batch)}")
          acc + length(batch)
        end)

      _ ->
        0
    end
  end

  defp context_select(nil) do
    {"SELECT id, node, entities, routed_to, supersedes, workspace_id FROM contexts", []}
  end

  defp context_select(ws) do
    {"SELECT id, node, entities, routed_to, supersedes, workspace_id FROM contexts WHERE workspace_id = ?1",
     [ws]}
  end

  defp edges_for_context_row([id, node, entities_json, routed_json, supersedes, ws]) do
    context = %{
      id: id,
      node: node,
      entities: decode_list(entities_json),
      routed_to: decode_list(routed_json),
      supersedes: supersedes,
      workspace_id: ws || "default"
    }

    Graph.create_edges_for_context(context)
  end

  # ── claim edges ───────────────────────────────────────────────────────────

  # Claims link to their source context via signal_id; the entities come from
  # that context. We join so each claim gets about-edges to its context's entities.
  defp backfill_claims(batch_size, workspace, shell) do
    {sql, params} = claim_select(workspace)

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        rows
        |> Enum.chunk_every(batch_size)
        |> Enum.reduce(0, fn batch, acc ->
          Enum.each(batch, &edges_for_claim_row/1)
          shell.info("  claims: #{acc + length(batch)}")
          acc + length(batch)
        end)

      _ ->
        0
    end
  end

  defp claim_select(nil) do
    {"""
     SELECT cl.id, c.entities, cl.workspace_id
     FROM claims cl
     JOIN contexts c ON c.id = cl.signal_id
     WHERE cl.signal_id IS NOT NULL
     """, []}
  end

  defp claim_select(ws) do
    {"""
     SELECT cl.id, c.entities, cl.workspace_id
     FROM claims cl
     JOIN contexts c ON c.id = cl.signal_id
     WHERE cl.signal_id IS NOT NULL AND cl.workspace_id = ?1
     """, [ws]}
  end

  defp edges_for_claim_row([claim_id, entities_json, ws]) do
    entities = decode_list(entities_json)

    if entities != [] do
      Graph.create_claim_edges(claim_id, entities, workspace_id: ws || "default")
    end

    :ok
  end

  # ── helpers ───────────────────────────────────────────────────────────────

  defp edge_count(nil) do
    case Store.raw_query("SELECT COUNT(*) FROM edges", []) do
      {:ok, [[n]]} -> n
      _ -> 0
    end
  end

  defp edge_count(ws) do
    case Store.raw_query("SELECT COUNT(*) FROM edges WHERE workspace_id = ?1", [ws]) do
      {:ok, [[n]]} -> n
      _ -> 0
    end
  end

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, list} when is_list(list) -> list
      _ -> []
    end
  end

  defp decode_list(_), do: []
end
