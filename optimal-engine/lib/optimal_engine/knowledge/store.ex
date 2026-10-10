defmodule OptimalEngine.Knowledge.Store do
  @moduledoc """
  GenServer managing a single knowledge store instance.

  Wraps a backend module and emits telemetry on all mutations.
  """

  use GenServer

  require Logger

  @default_backend OptimalEngine.Knowledge.Backend.ETS

  defstruct [:store_id, :backend, :backend_state]

  # --- Client API ---

  def start_link(opts) do
    store_id = Keyword.fetch!(opts, :store_id)
    name = Keyword.get(opts, :name, via_name(store_id))
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  def stop(store), do: GenServer.stop(store)

  def assert(store, s, p, o), do: GenServer.call(store, {:assert, s, p, o})

  def assert(store, g, s, p, o), do: GenServer.call(store, {:assert, g, s, p, o})

  def assert_many(store, triples), do: GenServer.call(store, {:assert_many, triples})

  def retract(store, s, p, o), do: GenServer.call(store, {:retract, s, p, o})

  def retract(store, g, s, p, o), do: GenServer.call(store, {:retract, g, s, p, o})

  def query(store, pattern), do: GenServer.call(store, {:query, pattern})

  def sparql(store, query_string), do: GenServer.call(store, {:sparql, query_string})

  def count(store), do: GenServer.call(store, :count)

  # --- Server Callbacks ---

  @impl true
  def init(opts) do
    store_id = Keyword.fetch!(opts, :store_id)
    backend = Keyword.get(opts, :backend, @default_backend)
    backend_opts = Keyword.get(opts, :backend_opts, [])

    case backend.init(store_id, backend_opts) do
      {:ok, backend_state} ->
        emit_telemetry(:open, %{store_id: store_id, backend: backend})

        {:ok,
         %__MODULE__{
           store_id: store_id,
           backend: backend,
           backend_state: backend_state
         }}

      {:error, reason} ->
        {:stop, reason}
    end
  end

  @impl true
  def handle_call({:assert, s, p, o}, _from, state) do
    case state.backend.assert(state.backend_state, s, p, o) do
      {:ok, new_backend_state} ->
        emit_telemetry(:assert, %{store_id: state.store_id, triple: {s, p, o}})
        {:reply, :ok, %{state | backend_state: new_backend_state}}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  def handle_call({:assert, g, s, p, o}, _from, state) do
    result =
      if function_exported?(state.backend, :assert, 5) do
        state.backend.assert(state.backend_state, g, s, p, o)
      else
        state.backend.assert(state.backend_state, s, p, o)
      end

    case result do
      {:ok, new_backend_state} ->
        emit_telemetry(:assert, %{store_id: state.store_id, triple: {s, p, o}, graph: g})
        {:reply, :ok, %{state | backend_state: new_backend_state}}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  def handle_call({:assert_many, triples}, _from, state) do
    case state.backend.assert_many(state.backend_state, triples) do
      {:ok, new_backend_state} ->
        emit_telemetry(:assert_many, %{store_id: state.store_id, count: length(triples)})
        {:reply, :ok, %{state | backend_state: new_backend_state}}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  def handle_call({:retract, s, p, o}, _from, state) do
    case state.backend.retract(state.backend_state, s, p, o) do
      {:ok, new_backend_state} ->
        emit_telemetry(:retract, %{store_id: state.store_id, triple: {s, p, o}})
        {:reply, :ok, %{state | backend_state: new_backend_state}}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  def handle_call({:retract, g, s, p, o}, _from, state) do
    result =
      if function_exported?(state.backend, :retract, 5) do
        state.backend.retract(state.backend_state, g, s, p, o)
      else
        state.backend.retract(state.backend_state, s, p, o)
      end

    case result do
      {:ok, new_backend_state} ->
        emit_telemetry(:retract, %{store_id: state.store_id, triple: {s, p, o}, graph: g})
        {:reply, :ok, %{state | backend_state: new_backend_state}}

      {:error, _} = error ->
        {:reply, error, state}
    end
  end

  def handle_call({:query, pattern}, _from, state) do
    result = state.backend.query(state.backend_state, pattern)
    {:reply, result, state}
  end

  def handle_call({:sparql, query_string}, _from, state) do
    if function_exported?(state.backend, :sparql, 2) do
      # Backend implements native SPARQL (optional callback)
      result = state.backend.sparql(state.backend_state, query_string)
      {:reply, result, state}
    else
      # Use MIOSA's native pure-Elixir SPARQL engine
      result =
        OptimalEngine.Knowledge.SPARQL.query(query_string, state.backend, state.backend_state)

      {:reply, result, state}
    end
  end

  def handle_call(:count, _from, state) do
    result = state.backend.count(state.backend_state)
    {:reply, result, state}
  end

  @impl true
  def terminate(_reason, state) do
    state.backend.terminate(state.backend_state)
    :ok
  end

  # --- Private ---

  defp via_name(store_id) do
    {:via, Registry, {OptimalEngine.Knowledge.Registry, store_id}}
  end

  defp emit_telemetry(event, metadata) do
    :telemetry.execute(
      [:optimal_engine, event],
      %{system_time: System.system_time()},
      metadata
    )
  end
end
