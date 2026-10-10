defmodule OptimalEngine.Knowledge.Backend.ETS do
  @moduledoc """
  ETS-backed knowledge store. Zero-config, in-memory, fast.

  Uses four ETS tables for GSPO/GPOS/GOSP/POSG quad indexing with named graph support:
  - GSPO index: `{graph, subject, predicate, object}` — graph+subject-first lookups
  - GPOS index: `{graph, predicate, object, subject}` — graph+predicate-first lookups
  - GOSP index: `{graph, object, subject, predicate}` — graph+object-first lookups
  - POSG index: `{predicate, object, subject, graph}` — cross-graph predicate-first lookups

  Triples asserted without a graph are stored in the `"default"` named graph.
  Query results are always returned as triples `{subject, predicate, object}` for
  backward compatibility; the graph is query metadata, not part of the result tuple.

  All pattern queries select the optimal index automatically.
  """

  @behaviour OptimalEngine.Knowledge.Backend

  @default_graph "default"

  defstruct [:gspo, :gpos, :gosp, :posg, :store_id]

  @impl true
  def init(store_id, _opts) do
    gspo = :ets.new(:"mk_gspo_#{store_id}", [:ordered_set, :public])
    gpos = :ets.new(:"mk_gpos_#{store_id}", [:ordered_set, :public])
    gosp = :ets.new(:"mk_gosp_#{store_id}", [:ordered_set, :public])
    posg = :ets.new(:"mk_posg_#{store_id}", [:ordered_set, :public])

    {:ok, %__MODULE__{gspo: gspo, gpos: gpos, gosp: gosp, posg: posg, store_id: store_id}}
  end

  # Triple assert — normalize to default graph and delegate
  @impl true
  def assert(state, s, p, o) do
    assert(state, @default_graph, s, p, o)
  end

  # Quad assert — graph-aware
  @impl true
  def assert(state, g, s, p, o) do
    :ets.insert(state.gspo, {{g, s, p, o}})
    :ets.insert(state.gpos, {{g, p, o, s}})
    :ets.insert(state.gosp, {{g, o, s, p}})
    :ets.insert(state.posg, {{p, o, s, g}})
    {:ok, state}
  end

  @impl true
  def assert_many(state, triples) do
    gspo_entries = Enum.map(triples, fn {s, p, o} -> {{@default_graph, s, p, o}} end)
    gpos_entries = Enum.map(triples, fn {s, p, o} -> {{@default_graph, p, o, s}} end)
    gosp_entries = Enum.map(triples, fn {s, p, o} -> {{@default_graph, o, s, p}} end)
    posg_entries = Enum.map(triples, fn {s, p, o} -> {{p, o, s, @default_graph}} end)

    :ets.insert(state.gspo, gspo_entries)
    :ets.insert(state.gpos, gpos_entries)
    :ets.insert(state.gosp, gosp_entries)
    :ets.insert(state.posg, posg_entries)

    {:ok, state}
  end

  # Triple retract — normalize to default graph and delegate
  @impl true
  def retract(state, s, p, o) do
    retract(state, @default_graph, s, p, o)
  end

  # Quad retract — graph-aware
  @impl true
  def retract(state, g, s, p, o) do
    :ets.delete(state.gspo, {g, s, p, o})
    :ets.delete(state.gpos, {g, p, o, s})
    :ets.delete(state.gosp, {g, o, s, p})
    :ets.delete(state.posg, {p, o, s, g})
    {:ok, state}
  end

  @impl true
  def query(state, pattern) do
    g = Keyword.get(pattern, :graph)
    s = Keyword.get(pattern, :subject)
    p = Keyword.get(pattern, :predicate)
    o = Keyword.get(pattern, :object)

    results = do_query(state, g, s, p, o)
    {:ok, results}
  end

  @impl true
  def count(state) do
    {:ok, :ets.info(state.gspo, :size)}
  end

  @impl true
  def sparql(_state, _query) do
    {:error, :sparql_not_supported}
  end

  @impl true
  def terminate(state) do
    try do
      :ets.delete(state.gspo)
      :ets.delete(state.gpos)
      :ets.delete(state.gosp)
      :ets.delete(state.posg)
    rescue
      ArgumentError -> :ok
    end

    :ok
  end

  # --- Pattern matching: select optimal index ---
  # Results are always returned as triples {s, p, o} — graph is query metadata.

  # All four bound — exact lookup on GSPO
  defp do_query(state, g, s, p, o)
       when not is_nil(g) and not is_nil(s) and not is_nil(p) and not is_nil(o) do
    case :ets.lookup(state.gspo, {g, s, p, o}) do
      [_] -> [{s, p, o}]
      [] -> []
    end
  end

  # g + s + p — GSPO prefix scan
  defp do_query(state, g, s, p, nil) when not is_nil(g) and not is_nil(s) and not is_nil(p) do
    ms = [{{{g, s, p, :"$1"}}, [], [{{s, p, :"$1"}}]}]
    :ets.select(state.gspo, ms)
  end

  # g + s — GSPO prefix scan
  defp do_query(state, g, s, nil, nil) when not is_nil(g) and not is_nil(s) do
    ms = [{{{g, s, :"$1", :"$2"}}, [], [{{s, :"$1", :"$2"}}]}]
    :ets.select(state.gspo, ms)
  end

  # g + p + o — GPOS prefix scan
  defp do_query(state, g, nil, p, o) when not is_nil(g) and not is_nil(p) and not is_nil(o) do
    ms = [{{{g, p, o, :"$1"}}, [], [{{:"$1", p, o}}]}]
    :ets.select(state.gpos, ms)
  end

  # g + p — GPOS prefix scan
  defp do_query(state, g, nil, p, nil) when not is_nil(g) and not is_nil(p) do
    ms = [{{{g, p, :"$1", :"$2"}}, [], [{{:"$2", p, :"$1"}}]}]
    :ets.select(state.gpos, ms)
  end

  # g + o + s — GOSP: lookup by graph+object+subject
  defp do_query(state, g, s, nil, o) when not is_nil(g) and not is_nil(s) and not is_nil(o) do
    ms = [{{{g, o, s, :"$1"}}, [], [{{s, :"$1", o}}]}]
    :ets.select(state.gosp, ms)
  end

  # g + o — GOSP prefix scan
  defp do_query(state, g, nil, nil, o) when not is_nil(g) and not is_nil(o) do
    ms = [{{{g, o, :"$1", :"$2"}}, [], [{{:"$1", :"$2", o}}]}]
    :ets.select(state.gosp, ms)
  end

  # g only — GSPO prefix scan for whole graph
  defp do_query(state, g, nil, nil, nil) when not is_nil(g) do
    ms = [{{{g, :"$1", :"$2", :"$3"}}, [], [{{:"$1", :"$2", :"$3"}}]}]
    :ets.select(state.gspo, ms)
  end

  # s + p + o (no graph) — cross-graph exact via GSPO match spec
  defp do_query(state, nil, s, p, o) when not is_nil(s) and not is_nil(p) and not is_nil(o) do
    ms = [{{{:"$1", s, p, o}}, [], [{{s, p, o}}]}]
    :ets.select(state.gspo, ms)
  end

  # s + p (no graph) — cross-graph GSPO match spec
  defp do_query(state, nil, s, p, nil) when not is_nil(s) and not is_nil(p) do
    ms = [{{{:_, s, p, :"$1"}}, [], [{{s, p, :"$1"}}]}]
    :ets.select(state.gspo, ms)
  end

  # s only (no graph) — cross-graph GSPO match spec
  defp do_query(state, nil, s, nil, nil) when not is_nil(s) do
    ms = [{{{:_, s, :"$1", :"$2"}}, [], [{{s, :"$1", :"$2"}}]}]
    :ets.select(state.gspo, ms)
  end

  # p + o (no graph) — cross-graph POSG prefix scan
  defp do_query(state, nil, nil, p, o) when not is_nil(p) and not is_nil(o) do
    ms = [{{{p, o, :"$1", :_}}, [], [{{:"$1", p, o}}]}]
    :ets.select(state.posg, ms)
  end

  # p only (no graph) — cross-graph POSG prefix scan
  defp do_query(state, nil, nil, p, nil) when not is_nil(p) do
    ms = [{{{p, :"$1", :"$2", :_}}, [], [{{:"$2", p, :"$1"}}]}]
    :ets.select(state.posg, ms)
  end

  # o only (no graph) — cross-graph GOSP match spec
  defp do_query(state, nil, nil, nil, o) when not is_nil(o) do
    ms = [{{{:_, o, :"$1", :"$2"}}, [], [{{:"$1", :"$2", o}}]}]
    :ets.select(state.gosp, ms)
  end

  # s + o (no graph) — cross-graph GOSP match spec
  defp do_query(state, nil, s, nil, o) when not is_nil(s) and not is_nil(o) do
    ms = [{{{:_, o, s, :"$1"}}, [], [{{s, :"$1", o}}]}]
    :ets.select(state.gosp, ms)
  end

  # Wildcard (no constraints) — full scan on GSPO
  defp do_query(state, nil, nil, nil, nil) do
    ms = [{{{:_, :"$1", :"$2", :"$3"}}, [], [{{:"$1", :"$2", :"$3"}}]}]
    :ets.select(state.gspo, ms)
  end
end
