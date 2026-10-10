defmodule OptimalEngine.Routing do
  @moduledoc """
  Loads and parses topology configuration from YAML files.

  Two YAML files are consumed:
  - `.system/config.yaml` — paths, node mapping, tiers, half-lives, routing rules
  - `topology.yaml` — full entity/endpoint/agent definitions with genre competence

  This module is stateless — call `load!/0` at startup and pass the result around,
  or call it within a GenServer init. The topology rarely changes during runtime.
  """

  require Logger

  @type node_entry :: %{
          id: String.t(),
          type: String.t(),
          folder: String.t()
        }

  @type endpoint :: %{
          id: String.t(),
          name: String.t(),
          role: String.t(),
          genre_competence: [String.t()],
          channels: [String.t()],
          notes: String.t() | nil
        }

  @type routing_rule :: %{
          name: String.t(),
          condition: map(),
          destination: String.t() | [String.t()],
          priority: String.t()
        }

  @type t :: %{
          root_path: String.t(),
          db_path: String.t(),
          cache_path: String.t(),
          nodes: %{String.t() => node_entry()},
          half_lives: %{String.t() => non_neg_integer()},
          routing_rules: [routing_rule()],
          endpoints: %{String.t() => endpoint()},
          tier_budgets: %{String.t() => non_neg_integer() | nil}
        }

  @doc """
  Loads topology from both config YAML files.
  Returns `{:ok, topology}` or `{:error, reason}`.
  """
  @spec load() :: {:ok, t()} | {:error, term()}
  def load do
    config_path = Application.get_env(:optimal_engine, :topology_path)
    full_path = Application.get_env(:optimal_engine, :topology_full_path)
    root = Application.get_env(:optimal_engine, :root_path, File.cwd!())
    db = Application.get_env(:optimal_engine, :db_path)
    cache = Application.get_env(:optimal_engine, :cache_path)

    with {:ok, config} <- load_yaml(config_path),
         {:ok, full} <- load_yaml(full_path) do
      topology = %{
        root_path: root,
        db_path: db,
        cache_path: cache,
        nodes: parse_nodes(config),
        half_lives: parse_half_lives(config),
        routing_rules: parse_routing_rules(full),
        endpoints: parse_endpoints(full),
        tier_budgets: parse_tiers(config)
      }

      {:ok, topology}
    end
  end

  @doc "Same as `load/0` but raises on failure."
  @spec load!() :: t()
  def load! do
    case load() do
      {:ok, topology} -> topology
      {:error, reason} -> raise "Failed to load topology: #{inspect(reason)}"
    end
  end

  @doc "Returns the half-life in hours for a given genre, defaulting to 720h (30 days)."
  @spec half_life_for(t(), String.t()) :: non_neg_integer()
  def half_life_for(topology, genre) when is_binary(genre) do
    Map.get(topology.half_lives, genre, Map.get(topology.half_lives, "default", 720))
  end

  @doc "Returns the endpoint profile for a receiver ID, or nil."
  @spec endpoint_for(t(), String.t()) :: endpoint() | nil
  def endpoint_for(topology, receiver_id) do
    Map.get(topology.endpoints, receiver_id)
  end

  @doc "Returns the primary genre competence for a receiver (first in list)."
  @spec primary_genre_for(t(), String.t()) :: String.t()
  def primary_genre_for(topology, receiver_id) do
    case endpoint_for(topology, receiver_id) do
      %{genre_competence: [primary | _]} -> primary
      _ -> "note"
    end
  end

  @doc """
  Returns all node folder names (e.g. \"person-operator\", \"product-customer-portal\").
  """
  @spec node_folders(t()) :: [String.t()]
  def node_folders(topology) do
    Map.keys(topology.nodes)
  end

  # --- Parsing helpers ---

  defp load_yaml(path) when is_binary(path) do
    case File.read(path) do
      {:ok, content} ->
        case YamlElixir.read_from_string(content) do
          {:ok, data} -> {:ok, data}
          {:error, _} = err -> err
        end

      {:error, :enoent} ->
        {:ok, %{}}

      {:error, reason} ->
        Logger.warning("Could not read YAML at #{path}: #{inspect(reason)}")
        {:ok, %{}}
    end
  end

  defp load_yaml(nil), do: {:ok, %{}}

  defp parse_nodes(config) do
    nodes = Map.get(config, "nodes", %{})

    Enum.into(nodes, %{}, fn {folder, attrs} ->
      entry = %{
        id: Map.get(attrs, "id", folder),
        type: Map.get(attrs, "type", "unknown"),
        folder: folder
      }

      {folder, entry}
    end)
  end

  defp parse_half_lives(config) do
    Map.get(config, "half_lives", %{})
    |> Enum.into(%{}, fn {k, v} -> {to_string(k), v} end)
  end

  defp parse_routing_rules(full) do
    rules = Map.get(full, "routing_rules", [])

    Enum.map(rules, fn rule ->
      %{
        name: Map.get(rule, "name", ""),
        condition: Map.get(rule, "condition", %{}),
        destination: normalize_destination(Map.get(rule, "destination", "inbox")),
        priority: Map.get(rule, "priority", "normal")
      }
    end)
  end

  defp normalize_destination(dest) when is_list(dest), do: dest
  defp normalize_destination(dest) when is_binary(dest), do: [dest]
  defp normalize_destination(_), do: ["inbox"]

  defp parse_endpoints(full) do
    endpoints = Map.get(full, "endpoints", %{})

    Enum.into(endpoints, %{}, fn {id, attrs} ->
      competence =
        Map.get(attrs, "genre_competence", [])
        |> Enum.map(&to_string/1)

      channels =
        Map.get(attrs, "channels", [])
        |> Enum.map(&to_string/1)

      entry = %{
        id: to_string(id),
        name: Map.get(attrs, "name", id),
        role: Map.get(attrs, "role", ""),
        genre_competence: competence,
        channels: channels,
        notes: Map.get(attrs, "notes")
      }

      {to_string(id), entry}
    end)
  end

  defp parse_tiers(config) do
    tiers = Map.get(config, "tiers", %{})

    Enum.into(tiers, %{}, fn {tier, attrs} ->
      {to_string(tier), Map.get(attrs, "max_tokens")}
    end)
  end
end
