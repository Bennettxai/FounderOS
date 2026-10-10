defmodule OptimalEngine.Retrieval do
  @moduledoc """
  Top-level facade for the retrieval layer.

  This is the module you reach for when you want an answer: one call,
  wiki-first, principal-filtered, bandwidth-aware.

      {:ok, result} = OptimalEngine.Retrieval.ask("Q4 pricing decision")

      result.source       # :wiki | :chunks | :empty
      result.envelope.body
      result.trace.wiki_hit?

  ## When to use which sub-module

  | You want                                 | Call                                    |
  |-----------------------------------------|-----------------------------------------|
  | An answer for a receiver                | `Retrieval.ask/2`                       |
  | Structural inventory (always-loaded L0) | `Retrieval.ContextAssembler.l0/0`       |
  | Raw ranked chunks                       | `Retrieval.Search.search/2`             |
  | Check if a curated wiki page exists     | `Retrieval.WikiFirst.lookup/3`          |
  | Build a receiver profile                | `Retrieval.Receiver.from_principal/2`   |
  | Pack chunks into a token budget         | `Retrieval.BandwidthPlanner.plan/2`     |

  See `docs/architecture/RETRIEVAL.md` for the full flow diagram.
  """

  alias OptimalEngine.Retrieval.{
    BandwidthPlanner,
    Deliver,
    RAG,
    Receiver,
    Search,
    WikiFirst
  }

  @doc "Ask the engine a question. Wiki-first, hybrid-fallback. See `RAG.ask/2`."
  def ask(query, opts \\ []), do: RAG.ask(query, opts)

  @doc "Build a receiver profile. See `Receiver.new/1`."
  def receiver(attrs), do: Receiver.new(attrs)

  @doc "Default receiver for anonymous callers (CLI, batch)."
  def anonymous_receiver(opts \\ []), do: Receiver.anonymous(opts)

  @doc "Hydrate a receiver from a stored principal id."
  def receiver_from_principal(principal_id, opts \\ []) do
    Receiver.from_principal(principal_id, opts)
  end

  @doc "Probe Tier 3 for a curated wiki hit."
  def wiki_lookup(query, audience \\ "default", opts \\ []) do
    WikiFirst.lookup(query, audience, opts)
  end

  @doc "Raw hybrid search (FTS5 + vectors + graph). See `Search.search/2`."
  def search(query, opts \\ []), do: Search.search(query, opts)

  @doc "Pack items under a token budget. See `BandwidthPlanner.plan/2`."
  def plan(items, budget), do: BandwidthPlanner.plan(items, budget)

  @doc "Render a list of chunks in the receiver's format."
  def render_chunks(chunks, receiver), do: Deliver.render_chunks(chunks, receiver)
end
