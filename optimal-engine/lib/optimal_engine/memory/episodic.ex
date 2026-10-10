defmodule OptimalEngine.Memory.Episodic do
  @moduledoc """
  Episodic memory for patterns, solutions, and decisions.

  Records structured episodes that represent learned knowledge
  from agent interactions. Supports recall by similarity (keyword match)
  and categorized listing.

  ## Episode Types
    - `:pattern` - a recurring pattern observed across interactions
    - `:solution` - a proven solution to a known problem
    - `:decision` - an architectural or design decision with rationale
  """

  @collection "episodic"

  @type episode_type :: :pattern | :solution | :decision
  @type episode :: %{
          type: episode_type(),
          description: String.t(),
          context: String.t(),
          outcome: String.t(),
          tags: [String.t()],
          created_at: DateTime.t()
        }

  @doc """
  Record an episode (pattern, solution, or decision).

  ## Parameters
    - `type` - one of `:pattern`, `:solution`, `:decision`
    - `attrs` - map with keys `:description`, `:context`, `:outcome`, `:tags`

  ## Examples

      OptimalEngine.Memory.Episodic.record_episode(:pattern, %{
        description: "Users often ask about config",
        context: "Support chat",
        outcome: "Created FAQ section",
        tags: ["support", "config"]
      })
  """
  @spec record_episode(episode_type(), map()) :: :ok | {:error, term()}
  def record_episode(type, attrs) when type in [:pattern, :solution, :decision] do
    episode = %{
      type: type,
      description: Map.get(attrs, :description, ""),
      context: Map.get(attrs, :context, ""),
      outcome: Map.get(attrs, :outcome, ""),
      tags: Map.get(attrs, :tags, []),
      created_at: DateTime.utc_now()
    }

    key = generate_key(type)
    tags = [to_string(type)] ++ episode.tags

    OptimalEngine.Memory.store(@collection, key, episode, tags: tags)
  end

  @doc """
  Find episodes similar to the given query (keyword match).

  Returns matching episodes sorted by relevance.
  """
  @spec recall_similar(String.t()) :: {:ok, [episode()]}
  def recall_similar(query) do
    case OptimalEngine.Memory.search(@collection, query) do
      {:ok, entries} ->
        episodes = Enum.map(entries, & &1.value)
        {:ok, episodes}

      error ->
        error
    end
  end

  @doc """
  List all recorded patterns.
  """
  @spec patterns() :: {:ok, [episode()]}
  def patterns do
    list_by_type(:pattern)
  end

  @doc """
  List all proven solutions.
  """
  @spec solutions() :: {:ok, [episode()]}
  def solutions do
    list_by_type(:solution)
  end

  @doc """
  List all recorded decisions.
  """
  @spec decisions() :: {:ok, [episode()]}
  def decisions do
    list_by_type(:decision)
  end

  @doc """
  List all episodes regardless of type.
  """
  @spec all() :: {:ok, [episode()]}
  def all do
    backend = OptimalEngine.Memory.Store.backend()

    case backend.list(@collection, []) do
      {:ok, entries} ->
        {:ok, Enum.map(entries, & &1.value)}

      error ->
        error
    end
  end

  # --- Private ---

  defp list_by_type(type) do
    case OptimalEngine.Memory.search(@collection, to_string(type)) do
      {:ok, entries} ->
        episodes =
          entries
          |> Enum.map(& &1.value)
          |> Enum.filter(fn
            %{type: ^type} -> true
            ep when is_map(ep) -> Map.get(ep, :type) == type
            _ -> false
          end)

        {:ok, episodes}

      error ->
        error
    end
  end

  defp generate_key(type) do
    ts = System.system_time(:microsecond)
    "#{type}_#{ts}"
  end
end
