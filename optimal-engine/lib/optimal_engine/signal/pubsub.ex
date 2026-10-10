defmodule OptimalEngine.Signal.PubSub do
  @moduledoc """
  Built-in ETS-backed PubSub broker for the MIOSA Signal ecosystem.

  Provides topic-based publish/subscribe without requiring Phoenix.PubSub.
  Supports exact-match topics and two wildcard forms:

  - `*`  — matches exactly one path segment
  - `**` — matches one or more trailing segments

  ## Examples

      {:ok, _} = OptimalEngine.Signal.PubSub.start_link(name: MyApp.PubSub)

      OptimalEngine.Signal.PubSub.subscribe(MyApp.PubSub, "signals.orders.*", self())
      OptimalEngine.Signal.PubSub.broadcast(MyApp.PubSub, "signals.orders.created", %{id: 1})
      # subscriber receives: {:pubsub, "signals.orders.created", %{id: 1}}

      OptimalEngine.Signal.PubSub.unsubscribe(MyApp.PubSub, "signals.orders.*", self())

  Process monitoring is automatic — when a subscriber exits, all its subscriptions
  are cleaned up without any explicit call.
  """

  use GenServer

  require Logger

  @type server :: GenServer.server()
  @type topic :: String.t()

  # ── Public API ──────────────────────────────────────────────────

  @doc """
  Starts the PubSub broker.

  ## Options

  - `:name` — registered name for the server (required for named access)
  """
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    {name, init_opts} = Keyword.pop(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, init_opts, name: name)
  end

  @doc """
  Subscribes `pid` to messages broadcast on `topic`.

  The topic pattern may contain:
  - `*`  — wildcard matching a single dot-delimited segment
  - `**` — wildcard matching all remaining segments

  Subscribing the same `{topic, pid}` pair more than once is idempotent.
  """
  @spec subscribe(server(), topic(), pid()) :: :ok
  def subscribe(server \\ __MODULE__, topic, pid) when is_binary(topic) and is_pid(pid) do
    GenServer.call(server, {:subscribe, topic, pid})
  end

  @doc """
  Removes `pid`'s subscription to `topic`.

  Safe to call even if the subscription does not exist.
  """
  @spec unsubscribe(server(), topic(), pid()) :: :ok
  def unsubscribe(server \\ __MODULE__, topic, pid) when is_binary(topic) and is_pid(pid) do
    GenServer.call(server, {:unsubscribe, topic, pid})
  end

  @doc """
  Broadcasts `message` to all subscribers whose topic pattern matches `topic`.

  Subscribers receive `{:pubsub, topic, message}`.

  Returns `:ok` regardless of subscriber count.
  """
  @spec broadcast(server(), topic(), term()) :: :ok
  def broadcast(server \\ __MODULE__, topic, message) when is_binary(topic) do
    GenServer.call(server, {:broadcast, topic, message})
  end

  @doc """
  Returns the list of `{topic_pattern, pid}` subscriptions currently registered.
  Useful for inspection and testing.
  """
  @spec list_subscriptions(server()) :: [{topic(), pid()}]
  def list_subscriptions(server \\ __MODULE__) do
    GenServer.call(server, :list_subscriptions)
  end

  # ── GenServer Callbacks ──────────────────────────────────────────

  @impl GenServer
  def init(_opts) do
    table = :ets.new(:optimal_engine_pubsub_subs, [:bag, :private])
    # monitors table: ref -> {topic, pid}
    monitors = :ets.new(:optimal_engine_pubsub_monitors, [:set, :private])
    {:ok, %{table: table, monitors: monitors}}
  end

  @impl GenServer
  def handle_call({:subscribe, topic, pid}, _from, state) do
    %{table: table, monitors: monitors} = state

    # Idempotent: only insert if not already present
    existing = :ets.match(table, {topic, pid})

    if existing == [] do
      :ets.insert(table, {topic, pid})
      ensure_monitored(monitors, pid)
    end

    {:reply, :ok, state}
  end

  def handle_call({:unsubscribe, topic, pid}, _from, state) do
    %{table: table, monitors: monitors} = state
    :ets.delete_object(table, {topic, pid})
    maybe_demonitor(table, monitors, pid)
    {:reply, :ok, state}
  end

  def handle_call({:broadcast, topic, message}, _from, state) do
    %{table: table} = state
    all_subs = :ets.tab2list(table)

    Enum.each(all_subs, fn {pattern, pid} ->
      if topic_matches?(pattern, topic) do
        send(pid, {:pubsub, topic, message})
      end
    end)

    {:reply, :ok, state}
  end

  def handle_call(:list_subscriptions, _from, state) do
    subs = :ets.tab2list(state.table)
    {:reply, subs, state}
  end

  @impl GenServer
  def handle_info({:DOWN, ref, :process, pid, _reason}, state) do
    %{table: table, monitors: monitors} = state

    # Remove all subscriptions for the dead process
    :ets.match_delete(table, {:_, pid})

    # Clean up the monitor reference
    :ets.match_delete(monitors, {ref, pid})

    {:noreply, state}
  end

  # ── Private Helpers ──────────────────────────────────────────────

  # Monitors the pid if it has not been monitored yet.
  defp ensure_monitored(monitors, pid) do
    case :ets.match(monitors, {:"$1", pid}) do
      [] ->
        ref = Process.monitor(pid)
        :ets.insert(monitors, {ref, pid})

      _ ->
        :ok
    end
  end

  # Demonitors when the pid has no remaining subscriptions.
  defp maybe_demonitor(table, monitors, pid) do
    remaining = :ets.match(table, {:_, pid})

    if remaining == [] do
      case :ets.match(monitors, {:"$1", pid}) do
        [[ref] | _] ->
          Process.demonitor(ref, [:flush])
          :ets.match_delete(monitors, {:"$1", pid})

        _ ->
          :ok
      end
    end
  end

  @doc false
  # Visible for testing.
  @spec topic_matches?(topic(), topic()) :: boolean()
  def topic_matches?(pattern, topic) do
    pattern_segs = String.split(pattern, ".")
    topic_segs = String.split(topic, ".")
    match_segments(pattern_segs, topic_segs)
  end

  defp match_segments([], []), do: true
  defp match_segments(["**"], [_ | _]), do: true
  defp match_segments(["**" | _rest], [_ | _]), do: true
  defp match_segments(["*" | p_rest], [_ | t_rest]), do: match_segments(p_rest, t_rest)
  defp match_segments([seg | p_rest], [seg | t_rest]), do: match_segments(p_rest, t_rest)
  defp match_segments(_, _), do: false
end
