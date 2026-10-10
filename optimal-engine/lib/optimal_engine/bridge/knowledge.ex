defmodule OptimalEngine.Bridge.Knowledge do
  @moduledoc """
  Bridge to OptimalEngine.Knowledge — stateless graph queries via SQLite, on-demand OWL reasoning.

  ## Design

  Graph queries (`graph_boost/2`, `context_for/1`) query SQLite edges directly —
  no GenServer hop, no ETS copy. Fast and simple.

  OWL reasoning (`materialize/0`) is on-demand only: lazily opens a OptimalEngine.Knowledge
  store, syncs edges, runs the reasoner, then returns. This is the ONE time we
  duplicate data into ETS — because OWL 2 RL needs a triple store to reason over.

  ## Integration points

  - `graph_boost/2` — Boosts search results by graph connectivity (queries SQLite directly)
  - `context_for/1` — Builds markdown context for a topic (queries SQLite directly)
  - `materialize/0` — Syncs edges to OptimalEngine.Knowledge, runs OWL 2 RL, returns inferred count
  - `sync_and_materialize/0` — Same as materialize, explicit name for indexer use
  """

  require Logger

  alias OptimalEngine.Store

  @store_name :optimal_knowledge

  # ---------------------------------------------------------------------------
  # Graph Boost — queries SQLite edges directly
  # ---------------------------------------------------------------------------

  @doc """
  Boosts search results that are connected in the knowledge graph.

  Queries SQLite `edges` table directly — no GenServer, no ETS copy.
  For each result, checks if its entities appear connected to the query terms.
  Connected results get a score boost.
  """
  @spec graph_boost([map()], String.t()) :: [map()]
  def graph_boost(results, query) when is_list(results) and is_binary(query) do
    terms = query |> String.downcase() |> String.split(~r/\s+/, trim: true)

    connected_entities = find_connected_entities(terms)

    if MapSet.size(connected_entities) == 0 do
      results
    else
      Enum.map(results, fn result ->
        entities = Map.get(result, :entities, []) || []

        boost =
          entities
          |> Enum.count(&MapSet.member?(connected_entities, &1))
          |> then(fn count -> 1.0 + count * 0.1 end)

        case Map.get(result, :score) do
          score when is_number(score) -> %{result | score: Float.round(score * boost, 4)}
          _ -> result
        end
      end)
    end
  rescue
    _ -> results
  end

  # ---------------------------------------------------------------------------
  # Context For — queries SQLite edges directly
  # ---------------------------------------------------------------------------

  @doc """
  Builds a structured context snapshot for a given entity/topic.
  Returns a markdown-formatted string ready for LLM injection.

  Queries SQLite edges directly for relationships, then formats as markdown.
  """
  @spec context_for(String.t()) :: {:ok, String.t()} | {:error, term()}
  def context_for(entity) when is_binary(entity) do
    # Find all edges where entity is source or target
    outgoing_sql = "SELECT source_id, relation, target_id FROM edges WHERE source_id LIKE ?1"
    incoming_sql = "SELECT source_id, relation, target_id FROM edges WHERE target_id LIKE ?1"
    pattern = "%#{entity}%"

    with {:ok, out_rows} <- Store.raw_query(outgoing_sql, [pattern]),
         {:ok, in_rows} <- Store.raw_query(incoming_sql, [pattern]) do
      all_edges = out_rows ++ in_rows

      if all_edges == [] do
        {:ok, "No knowledge graph context found for: #{entity}"}
      else
        markdown = format_context_markdown(entity, all_edges)
        {:ok, markdown}
      end
    end
  rescue
    e -> {:error, e}
  end

  # ---------------------------------------------------------------------------
  # OWL Reasoning — on-demand only
  # ---------------------------------------------------------------------------

  @doc """
  Syncs SQLite edges into a temporary OptimalEngine.Knowledge store and runs OWL 2 RL
  materialization. This is the ONE time we sync — after full reindex.

  Returns `{:ok, inferred_count}` or `{:error, reason}`.
  """
  @spec materialize() :: {:ok, non_neg_integer()} | {:error, term()}
  def materialize do
    sync_and_materialize()
  end

  @doc """
  Syncs edges from SQLite to OptimalEngine.Knowledge ETS store, then runs OWL reasoning.
  Called after full index to make the knowledge graph smarter.
  """
  @spec sync_and_materialize() :: {:ok, non_neg_integer()} | {:error, term()}
  def sync_and_materialize do
    Logger.info("[Bridge.Knowledge] Starting on-demand OWL materialization...")

    with {:ok, store} <- open_knowledge_store(),
         {:ok, synced} <- sync_edges_to_store(store),
         result <- run_materialization(store) do
      Logger.info(
        "[Bridge.Knowledge] OWL materialization complete: synced #{synced} edges, result: #{inspect(result)}"
      )

      result
    end
  rescue
    e ->
      Logger.error("[Bridge.Knowledge] Materialization failed: #{inspect(e)}")
      {:error, e}
  end

  @doc "Returns the triple count by counting edges in SQLite."
  @spec count() :: {:ok, non_neg_integer()} | {:error, term()}
  def count do
    case Store.raw_query("SELECT COUNT(*) FROM edges", []) do
      {:ok, [[count]]} -> {:ok, count}
      {:error, _} = err -> err
    end
  end

  # ---------------------------------------------------------------------------
  # Private: SQLite-based graph queries
  # ---------------------------------------------------------------------------

  defp find_connected_entities(terms) do
    terms
    |> Enum.flat_map(fn term ->
      pattern = "%#{term}%"

      # Find entities connected as objects (term is subject)
      objects =
        case Store.raw_query(
               "SELECT DISTINCT target_id FROM edges WHERE LOWER(source_id) LIKE ?1",
               [pattern]
             ) do
          {:ok, rows} -> Enum.map(rows, fn [target] -> target end)
          _ -> []
        end

      # Find entities connected as subjects (term is object)
      subjects =
        case Store.raw_query(
               "SELECT DISTINCT source_id FROM edges WHERE LOWER(target_id) LIKE ?1",
               [pattern]
             ) do
          {:ok, rows} -> Enum.map(rows, fn [source] -> source end)
          _ -> []
        end

      objects ++ subjects
    end)
    |> MapSet.new()
  end

  defp format_context_markdown(entity, edges) do
    grouped =
      Enum.group_by(edges, fn [_s, relation, _t] -> relation end)

    sections =
      Enum.map(grouped, fn {relation, rows} ->
        items =
          Enum.map(rows, fn [source, _rel, target] ->
            if String.contains?(String.downcase(source), String.downcase(entity)) do
              "  - → #{target}"
            else
              "  - ← #{source}"
            end
          end)

        "### #{relation}\n#{Enum.join(items, "\n")}"
      end)

    "## Knowledge Graph: #{entity}\n\n#{Enum.join(sections, "\n\n")}"
  end

  # ---------------------------------------------------------------------------
  # Private: On-demand OptimalEngine.Knowledge store
  # ---------------------------------------------------------------------------

  defp open_knowledge_store do
    OptimalEngine.Knowledge.open(@store_name, backend: OptimalEngine.Knowledge.Backend.ETS)
  end

  defp sync_edges_to_store(store) do
    case Store.raw_query("SELECT source_id, relation, target_id FROM edges", []) do
      {:ok, rows} ->
        triples =
          Enum.map(rows, fn [source, relation, target] ->
            [source, relation, target]
          end)

        OptimalEngine.Knowledge.assert_many(store, triples)
        Logger.info("[Bridge.Knowledge] Synced #{length(triples)} edges as triples")
        {:ok, length(triples)}

      {:error, _} = err ->
        err
    end
  end

  defp run_materialization(store) do
    OptimalEngine.Knowledge.Reasoner.materialize(store, store)
  end
end
