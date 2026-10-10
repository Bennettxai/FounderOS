defmodule OptimalEngine.Signal.Journal do
  @moduledoc """
  Signal history and causality tracker — records signals, traces chains, and queries correlations.

  The Journal is an ETS-backed GenServer that provides:

  - **Recording** — store signals as they flow through the system
  - **History** — query signals with filters (type, source, time range, agent)
  - **Causality chains** — trace parent_id links to reconstruct signal lineage
  - **Correlation groups** — find all signals sharing a correlation_id

  ## Usage

      {:ok, journal} = OptimalEngine.Signal.Journal.start_link(name: :my_journal)
      :ok = OptimalEngine.Signal.Journal.record(journal, signal)
      history = OptimalEngine.Signal.Journal.history(journal)
      chain = OptimalEngine.Signal.Journal.causality_chain(journal, signal.id)
  """

  use GenServer

  alias OptimalEngine.Signal.Envelope, as: Signal

  @type filter :: keyword()

  # ── Client API ──────────────────────────────────────────────────

  @doc """
  Starts the journal as a named GenServer.

  ## Options

  - `:name` — registration name (default: `__MODULE__`)
  - `:max_size` — maximum number of signals to retain (default: 10_000). Oldest are evicted first.
  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    name = Keyword.get(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  @doc """
  Records a signal in the journal.

  ## Examples

      :ok = OptimalEngine.Signal.Journal.record(journal, signal)
  """
  @spec record(GenServer.server(), Signal.t()) :: :ok
  def record(journal, %Signal{} = signal) do
    GenServer.call(journal, {:record, signal})
  end

  @doc """
  Retrieves signal history with optional filters.

  ## Filters

  - `:type` — exact type match or prefix match with trailing `*`
  - `:source` — exact source match
  - `:agent_id` — filter by agent
  - `:session_id` — filter by session
  - `:since` — DateTime, only signals after this time
  - `:until` — DateTime, only signals before this time
  - `:limit` — max results (default: 100)

  ## Examples

      signals = OptimalEngine.Signal.Journal.history(journal, type: "miosa.agent.*", limit: 50)
  """
  @spec history(GenServer.server(), filter()) :: [Signal.t()]
  def history(journal, filters \\ []) do
    GenServer.call(journal, {:history, filters})
  end

  @doc """
  Traces the causality chain for a signal by following parent_id links.

  Returns a list of signals from the given signal back to the root (no parent),
  ordered oldest-first (root first).

  ## Examples

      chain = OptimalEngine.Signal.Journal.causality_chain(journal, signal_id)
  """
  @spec causality_chain(GenServer.server(), String.t()) :: [Signal.t()]
  def causality_chain(journal, signal_id) when is_binary(signal_id) do
    GenServer.call(journal, {:causality_chain, signal_id})
  end

  @doc """
  Finds all signals sharing a correlation_id.

  Returns signals ordered by time (oldest first).

  ## Examples

      group = OptimalEngine.Signal.Journal.by_correlation(journal, correlation_id)
  """
  @spec by_correlation(GenServer.server(), String.t()) :: [Signal.t()]
  def by_correlation(journal, correlation_id) when is_binary(correlation_id) do
    GenServer.call(journal, {:by_correlation, correlation_id})
  end

  @doc """
  Returns the total number of signals recorded.
  """
  @spec size(GenServer.server()) :: non_neg_integer()
  def size(journal) do
    GenServer.call(journal, :size)
  end

  @doc """
  Clears all recorded signals.
  """
  @spec clear(GenServer.server()) :: :ok
  def clear(journal) do
    GenServer.call(journal, :clear)
  end

  # ── Server Callbacks ────────────────────────────────────────────

  @impl true
  def init(opts) do
    max_size = Keyword.get(opts, :max_size, 10_000)

    # Primary index: id → signal
    signals_table = :ets.new(:journal_signals, [:set, :private])
    # Secondary index: correlation_id → [id]
    correlation_table = :ets.new(:journal_correlations, [:bag, :private])
    # Secondary index: parent_id → [id]
    parent_table = :ets.new(:journal_parents, [:bag, :private])
    # Ordered insertion tracking
    order_table = :ets.new(:journal_order, [:ordered_set, :private])

    state = %{
      signals: signals_table,
      correlations: correlation_table,
      parents: parent_table,
      order: order_table,
      max_size: max_size,
      counter: 0
    }

    {:ok, state}
  end

  @impl true
  def handle_call({:record, signal}, _from, state) do
    state = maybe_evict(state)

    # Insert into primary index
    :ets.insert(state.signals, {signal.id, signal})

    # Insert into order tracking
    :ets.insert(state.order, {state.counter, signal.id})

    # Insert into correlation index
    if signal.correlation_id do
      :ets.insert(state.correlations, {signal.correlation_id, signal.id})
    end

    # Insert into parent index
    if signal.parent_id do
      :ets.insert(state.parents, {signal.parent_id, signal.id})
    end

    {:reply, :ok, %{state | counter: state.counter + 1}}
  end

  @impl true
  def handle_call({:history, filters}, _from, state) do
    limit = Keyword.get(filters, :limit, 100)

    signals =
      :ets.tab2list(state.signals)
      |> Enum.map(fn {_id, signal} -> signal end)
      |> apply_filters(filters)
      |> Enum.sort_by(fn s -> s.time end, {:asc, DateTime})
      |> Enum.take(limit)

    {:reply, signals, state}
  end

  @impl true
  def handle_call({:causality_chain, signal_id}, _from, state) do
    chain = trace_chain(state.signals, signal_id, [])
    {:reply, chain, state}
  end

  @impl true
  def handle_call({:by_correlation, correlation_id}, _from, state) do
    ids =
      :ets.lookup(state.correlations, correlation_id)
      |> Enum.map(fn {_corr, id} -> id end)

    signals =
      ids
      |> Enum.flat_map(fn id ->
        case :ets.lookup(state.signals, id) do
          [{_id, signal}] -> [signal]
          [] -> []
        end
      end)
      |> Enum.sort_by(fn s -> s.time end, {:asc, DateTime})

    {:reply, signals, state}
  end

  @impl true
  def handle_call(:size, _from, state) do
    {:reply, :ets.info(state.signals, :size), state}
  end

  @impl true
  def handle_call(:clear, _from, state) do
    :ets.delete_all_objects(state.signals)
    :ets.delete_all_objects(state.correlations)
    :ets.delete_all_objects(state.parents)
    :ets.delete_all_objects(state.order)
    {:reply, :ok, %{state | counter: 0}}
  end

  @impl true
  def terminate(_reason, state) do
    :ets.delete(state.signals)
    :ets.delete(state.correlations)
    :ets.delete(state.parents)
    :ets.delete(state.order)
    :ok
  end

  # ── Private: Filtering ──────────────────────────────────────────

  defp apply_filters(signals, filters) do
    Enum.reduce(filters, signals, fn
      {:type, pattern}, sigs -> filter_by_type(sigs, pattern)
      {:source, source}, sigs -> Enum.filter(sigs, &(&1.source == source))
      {:agent_id, aid}, sigs -> Enum.filter(sigs, &(&1.agent_id == aid))
      {:session_id, sid}, sigs -> Enum.filter(sigs, &(&1.session_id == sid))
      {:since, dt}, sigs -> Enum.filter(sigs, &(DateTime.compare(&1.time, dt) != :lt))
      {:until, dt}, sigs -> Enum.filter(sigs, &(DateTime.compare(&1.time, dt) != :gt))
      {:limit, _}, sigs -> sigs
      _, sigs -> sigs
    end)
  end

  defp filter_by_type(signals, pattern) when is_binary(pattern) do
    if String.ends_with?(pattern, "*") do
      prefix = String.trim_trailing(pattern, "*")
      Enum.filter(signals, &String.starts_with?(&1.type, prefix))
    else
      Enum.filter(signals, &(&1.type == pattern))
    end
  end

  # ── Private: Causality Tracing ──────────────────────────────────

  defp trace_chain(table, signal_id, acc) do
    case :ets.lookup(table, signal_id) do
      [{_id, signal}] ->
        if signal.parent_id do
          trace_chain(table, signal.parent_id, [signal | acc])
        else
          [signal | acc]
        end

      [] ->
        acc
    end
  end

  # ── Private: Eviction ──────────────────────────────────────────

  defp maybe_evict(state) do
    current_size = :ets.info(state.signals, :size)

    if current_size >= state.max_size do
      # Evict oldest 10%
      evict_count = max(1, div(state.max_size, 10))
      evict_oldest(state, evict_count)
    else
      state
    end
  end

  defp evict_oldest(state, count) do
    keys =
      :ets.tab2list(state.order)
      |> Enum.sort_by(fn {order, _id} -> order end)
      |> Enum.take(count)

    Enum.each(keys, fn {order_key, signal_id} ->
      :ets.delete(state.signals, signal_id)
      :ets.delete(state.order, order_key)
      # Note: we don't clean up correlation/parent indexes for performance.
      # Lookups handle missing signals gracefully.
    end)

    state
  end
end
