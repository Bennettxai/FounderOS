defmodule OptimalEngine.Signal.Router do
  @moduledoc """
  Trie-based signal router that matches signal types to handlers using patterns.

  Supports three match strategies:

  - **Exact match** — `"miosa.agent.task.completed"` matches only that type
  - **Single wildcard** — `"miosa.agent.*.completed"` matches any single segment
  - **Multi wildcard** — `"miosa.agent.**"` matches any remaining segments

  Routes are ordered by specificity: exact > single wildcard > multi wildcard.
  Within the same specificity, routes are ordered by priority (higher first).

  ## Examples

      {:ok, router} = OptimalEngine.Signal.Router.start_link(name: :my_router)
      :ok = OptimalEngine.Signal.Router.add_route(:my_router, "miosa.agent.**", &handler/1, priority: 10)
      handlers = OptimalEngine.Signal.Router.match(:my_router, "miosa.agent.task.completed")
  """

  use GenServer

  @type pattern :: String.t()
  @type handler :: (OptimalEngine.Signal.Envelope.t() -> term()) | {module(), atom(), [term()]}
  @type priority :: integer()

  @type route :: %{
          pattern: pattern(),
          handler: handler(),
          priority: priority(),
          id: String.t()
        }

  # ── Client API ──────────────────────────────────────────────────

  @doc """
  Starts the router as a named GenServer.

  ## Options

  - `:name` — registration name (default: `__MODULE__`)

  ## Examples

      {:ok, pid} = OptimalEngine.Signal.Router.start_link(name: :signal_router)
  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    name = Keyword.get(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  @doc """
  Creates a new router (alias for `start_link/1` for API consistency).
  """
  @spec new(keyword()) :: GenServer.on_start()
  def new(opts \\ []), do: start_link(opts)

  @doc """
  Adds a route pattern that maps to a handler.

  ## Options

  - `:priority` — route priority, -100 to +100 (default: 0). Higher priority routes match first
    within the same specificity level.

  ## Pattern Syntax

  - Exact: `"miosa.agent.task.completed"` — matches only this exact type
  - Single wildcard: `"miosa.agent.*.completed"` — `*` matches one segment
  - Multi wildcard: `"miosa.agent.**"` — `**` matches one or more remaining segments

  ## Examples

      :ok = OptimalEngine.Signal.Router.add_route(router, "miosa.agent.**", fn sig -> IO.inspect(sig) end)
      :ok = OptimalEngine.Signal.Router.add_route(router, "miosa.agent.task.*", &handle_task/1, priority: 50)
  """
  @spec add_route(GenServer.server(), pattern(), handler(), keyword()) :: :ok
  def add_route(router, pattern, handler, opts \\ []) do
    GenServer.call(router, {:add_route, pattern, handler, opts})
  end

  @doc """
  Removes a route by its pattern string.

  ## Examples

      :ok = OptimalEngine.Signal.Router.remove_route(router, "miosa.agent.**")
  """
  @spec remove_route(GenServer.server(), pattern()) :: :ok | {:error, :not_found}
  def remove_route(router, pattern) do
    GenServer.call(router, {:remove_route, pattern})
  end

  @doc """
  Finds all matching handlers for a signal type, ordered by specificity then priority.

  Returns a list of `{handler, route}` tuples, most specific first.

  ## Examples

      matches = OptimalEngine.Signal.Router.match(router, "miosa.agent.task.completed")
  """
  @spec match(GenServer.server(), String.t() | OptimalEngine.Signal.Envelope.t()) :: [
          {handler(), route()}
        ]
  def match(router, %OptimalEngine.Signal.Envelope{type: type}), do: match(router, type)

  def match(router, type) when is_binary(type) do
    GenServer.call(router, {:match, type})
  end

  @doc """
  Lists all registered routes.

  ## Examples

      routes = OptimalEngine.Signal.Router.routes(router)
  """
  @spec routes(GenServer.server()) :: [route()]
  def routes(router) do
    GenServer.call(router, :routes)
  end

  # ── Server Callbacks ────────────────────────────────────────────

  @impl true
  def init(opts) do
    table = :ets.new(:signal_routes, [:set, :private])
    name = Keyword.get(opts, :name, __MODULE__)
    {:ok, %{table: table, name: name, counter: 0}}
  end

  @impl true
  def handle_call({:add_route, pattern, handler, opts}, _from, state) do
    priority = Keyword.get(opts, :priority, 0)
    priority = max(-100, min(100, priority))

    route = %{
      pattern: pattern,
      handler: handler,
      priority: priority,
      id: "route_#{state.counter}",
      segments: String.split(pattern, "."),
      specificity: compute_specificity(pattern)
    }

    :ets.insert(state.table, {pattern, route})
    {:reply, :ok, %{state | counter: state.counter + 1}}
  end

  @impl true
  def handle_call({:remove_route, pattern}, _from, state) do
    case :ets.lookup(state.table, pattern) do
      [] ->
        {:reply, {:error, :not_found}, state}

      _ ->
        :ets.delete(state.table, pattern)
        {:reply, :ok, state}
    end
  end

  @impl true
  def handle_call({:match, type}, _from, state) do
    type_segments = String.split(type, ".")

    matches =
      :ets.tab2list(state.table)
      |> Enum.map(fn {_key, route} -> route end)
      |> Enum.filter(fn route -> pattern_matches?(route.segments, type_segments) end)
      |> Enum.sort_by(fn route -> {-route.specificity, -route.priority} end)
      |> Enum.map(fn route ->
        public_route = Map.drop(route, [:segments, :specificity])
        {route.handler, public_route}
      end)

    {:reply, matches, state}
  end

  @impl true
  def handle_call(:routes, _from, state) do
    routes =
      :ets.tab2list(state.table)
      |> Enum.map(fn {_key, route} -> Map.drop(route, [:segments, :specificity]) end)
      |> Enum.sort_by(fn r -> {-r.priority} end)

    {:reply, routes, state}
  end

  @impl true
  def terminate(_reason, state) do
    :ets.delete(state.table)
    :ok
  end

  # ── Private: Pattern Matching ───────────────────────────────────

  # Both exhausted — match
  defp pattern_matches?([], []), do: true

  # Pattern exhausted, type has more — no match
  defp pattern_matches?([], _type_rest), do: false

  # Type exhausted, pattern has more — only match if remaining is **
  defp pattern_matches?(["**"], _), do: true
  defp pattern_matches?(_pattern_rest, []), do: false

  # Multi wildcard matches everything remaining
  defp pattern_matches?(["**" | _], _type_rest), do: true

  # Single wildcard matches exactly one segment
  defp pattern_matches?(["*" | p_rest], [_type_seg | t_rest]) do
    pattern_matches?(p_rest, t_rest)
  end

  # Exact segment match
  defp pattern_matches?([seg | p_rest], [seg | t_rest]) do
    pattern_matches?(p_rest, t_rest)
  end

  # No match
  defp pattern_matches?(_, _), do: false

  # ── Private: Specificity ────────────────────────────────────────

  # Higher = more specific. Exact segments worth 3, * worth 1, ** worth 0.
  defp compute_specificity(pattern) do
    pattern
    |> String.split(".")
    |> Enum.reduce(0, fn
      "**", acc -> acc
      "*", acc -> acc + 1
      _, acc -> acc + 3
    end)
  end
end
