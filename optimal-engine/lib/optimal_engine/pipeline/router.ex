defmodule OptimalEngine.Pipeline.Router do
  @moduledoc """
  Routes signals to destination nodes based on topology routing rules.

  Rules are loaded from `topology.yaml` at startup and cached in GenServer state.
  Rule evaluation is deterministic — all matching rules fire (not first-match).

  Priority levels: `:critical` > `:high` > `:normal` > `:low`

  Cross-cutting rules from `config.yaml` are also applied:
  - Financial genres always copy to `operation-revenue`
  - Any signal mentioning a known person copies to `team`
  """

  use GenServer
  require Logger

  alias OptimalEngine.{Signal, Routing}

  @financial_genres ~w[
    invoice profit-loss balance-sheet budget
    expense-report revenue-report financial-forecast financial-model
  ]

  # --- Client API ---

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Routes a Signal and returns the list of destination node IDs.
  Always returns at least `[\"inbox\"]` as a fallback.
  """
  @spec route(Signal.t()) :: {:ok, [String.t()]} | {:error, term()}
  def route(%Signal{} = signal) do
    GenServer.call(__MODULE__, {:route, signal})
  end

  @doc "Reloads routing rules from topology files."
  @spec reload() :: :ok
  def reload do
    GenServer.call(__MODULE__, :reload)
  end

  # --- GenServer Callbacks ---

  @impl true
  def init(_opts) do
    state = load_state()
    Logger.info("[Router] Loaded #{length(state.rules)} routing rules")
    {:ok, state}
  end

  @impl true
  def handle_call({:route, signal}, _from, state) do
    destinations =
      signal
      |> apply_rules(state.rules)
      |> apply_cross_cutting(signal)
      |> Enum.uniq()
      |> normalize_destinations()

    result = if destinations == [], do: ["inbox"], else: destinations

    :telemetry.execute(
      [:optimal_engine, :router, :route],
      %{destinations: length(result)},
      %{node: signal.node, genre: signal.genre}
    )

    {:reply, {:ok, result}, state}
  end

  @impl true
  def handle_call(:reload, _from, _state) do
    new_state = load_state()
    Logger.info("[Router] Reloaded #{length(new_state.rules)} routing rules")
    {:reply, :ok, new_state}
  end

  # --- Private: Rule Evaluation ---

  defp apply_rules(signal, rules) do
    rules
    |> Enum.filter(&rule_matches?(signal, &1))
    |> Enum.flat_map(& &1.destination)
  end

  defp rule_matches?(signal, rule) do
    condition = rule.condition

    cond do
      Map.has_key?(condition, "default") ->
        false

      Map.has_key?(condition, "signal_type") ->
        to_string(signal.type) == to_string(condition["signal_type"])

      Map.has_key?(condition, "source_node") ->
        signal.node == condition["source_node"]

      Map.has_key?(condition, "entity_mention") ->
        mentions = condition["entity_mention"]

        Enum.any?(mentions, fn entity ->
          Regex.match?(~r/#{Regex.escape(entity)}/i, signal.content || "") ||
            entity in (signal.entities || [])
        end)

      Map.has_key?(condition, "content_pattern") ->
        patterns = condition["content_pattern"]

        Enum.any?(patterns, fn pattern ->
          case Regex.compile(pattern) do
            {:ok, re} -> Regex.match?(re, signal.content || "")
            _ -> false
          end
        end)

      true ->
        false
    end
  end

  defp apply_cross_cutting(destinations, signal) do
    extra =
      []
      |> maybe_add_financial(signal)
      |> maybe_add_team(signal)

    destinations ++ extra
  end

  defp maybe_add_financial(acc, signal) do
    if signal.genre in @financial_genres do
      ["operation-revenue" | acc]
    else
      acc
    end
  end

  defp maybe_add_team(acc, signal) do
    if length(signal.entities || []) > 0 do
      ["team" | acc]
    else
      acc
    end
  end

  # Legacy destination aliases. Routing carries Node slugs through unchanged —
  # folder resolution is topology-aware and owned by `Intake.Writer` — so this
  # is NOT a list of known nodes, only renames for pre-topology rule targets.
  @legacy_destination_aliases %{
    "governance/l7-bypass" => "person-operator",
    "optimal-system" => "entity-company"
  }

  # Normalize: strip entity-path prefixes, apply legacy aliases, pass all other
  # Node slugs through untouched (unknown slugs are a Writer/topology concern).
  defp normalize_destinations(destinations) do
    destinations
    |> Enum.map(fn dest ->
      last = dest |> String.split("/") |> List.last()
      Map.get(@legacy_destination_aliases, last, last)
    end)
    |> Enum.reject(&is_nil/1)
    |> Enum.uniq()
  end

  defp load_state do
    topology =
      case Routing.load() do
        {:ok, t} -> t
        {:error, _} -> %{routing_rules: []}
      end

    %{rules: topology.routing_rules}
  end
end
