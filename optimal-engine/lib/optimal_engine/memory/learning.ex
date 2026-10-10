defmodule OptimalEngine.Memory.Learning do
  @moduledoc """
  Self-learning engine based on SICA, VIGIL, and memory-layer systems patterns.

  Continuous improvement loop:
    OBSERVE  → Capture every tool interaction, error, user correction
    REFLECT  → Identify patterns across recent interactions
    PROPOSE  → Generate new patterns/skills when repetition detected
    TEST     → Validate proposed patterns against past data
    INTEGRATE → Merge validated patterns into long-term memory

  Three-tier memory:
    Working  → ETS, 15min TTL, fast access
    Episodic → JSONL files, 30-day retention
    Semantic → MEMORY.md, permanent knowledge

  Consolidation triggers:
    - Every 5 interactions (incremental)
    - Every 50 interactions (full consolidation)
    - On session end (cleanup)
    - On user correction (immediate capture)
  """

  use GenServer
  require Logger

  @consolidation_interval 5
  @full_consolidation_interval 50
  @working_memory_ttl 900_000
  @max_episodes 1000
  @pattern_threshold 3
  @skill_generation_threshold 5

  defstruct interaction_count: 0,
            working_memory: %{},
            episodes: [],
            patterns: %{},
            solutions: %{},
            metrics: %{
              total_interactions: 0,
              patterns_captured: 0,
              skills_generated: 0,
              errors_recovered: 0,
              consolidations: 0
            }

  # ── Client API ───────────────────────────────────────────────────────

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Record a tool interaction for learning."
  @spec observe(map()) :: :ok
  def observe(interaction) do
    GenServer.cast(__MODULE__, {:observe, interaction})
  end

  @doc "Record a user correction (immediate learning)."
  @spec correction(String.t(), String.t()) :: :ok
  def correction(what_was_wrong, what_is_right) do
    GenServer.cast(__MODULE__, {:correction, what_was_wrong, what_is_right})
  end

  @doc "Record an error for VIGIL recovery."
  @spec error(String.t(), String.t(), String.t()) :: :ok
  def error(tool_name, error_message, context) do
    GenServer.cast(__MODULE__, {:error, tool_name, error_message, context})
  end

  @doc "Get current learning metrics."
  @spec metrics() :: map()
  def metrics, do: GenServer.call(__MODULE__, :metrics)

  @doc "Get detected patterns."
  @spec patterns() :: map()
  def patterns, do: GenServer.call(__MODULE__, :patterns)

  @doc "Get known solutions."
  @spec solutions() :: map()
  def solutions, do: GenServer.call(__MODULE__, :solutions)

  @doc "Force a consolidation cycle."
  @spec consolidate() :: :ok
  def consolidate, do: GenServer.cast(__MODULE__, :consolidate)

  # ── GenServer Callbacks ───────────────────────────────────────────────

  @impl true
  def init(_opts) do
    table = :ets.new(:optimal_engine_learning_working_memory, [:set, :public])
    Process.send_after(self(), :cleanup_working_memory, @working_memory_ttl)
    state = %__MODULE__{working_memory: table}
    state = load_persisted(state)

    Logger.info(
      "[OptimalEngine.Memory.Learning] SICA engine started (patterns: #{map_size(state.patterns)}, solutions: #{map_size(state.solutions)})"
    )

    {:ok, state}
  end

  @impl true
  def handle_cast({:observe, interaction}, state) do
    episode = %{
      timestamp: DateTime.utc_now(),
      type: :tool_use,
      tool_name: interaction[:tool_name],
      duration_ms: interaction[:duration_ms],
      success: interaction[:success] != false,
      result_length: interaction[:result_length] || 0,
      context: interaction[:context]
    }

    key = "ep_#{state.interaction_count}"
    :ets.insert(state.working_memory, {key, episode, System.monotonic_time(:millisecond)})

    episodes = [episode | state.episodes] |> Enum.take(@max_episodes)

    state = %{
      state
      | interaction_count: state.interaction_count + 1,
        episodes: episodes,
        metrics: %{state.metrics | total_interactions: state.metrics.total_interactions + 1}
    }

    state =
      cond do
        rem(state.interaction_count, @full_consolidation_interval) == 0 ->
          full_consolidation(state)

        rem(state.interaction_count, @consolidation_interval) == 0 ->
          incremental_consolidation(state)

        true ->
          state
      end

    {:noreply, state}
  end

  @impl true
  def handle_cast({:correction, what_was_wrong, what_is_right}, state) do
    Logger.info(
      "[OptimalEngine.Memory.Learning] User correction: #{String.slice(what_was_wrong, 0, 50)} → #{String.slice(what_is_right, 0, 50)}"
    )

    key = normalize_key(what_was_wrong)

    solutions =
      Map.put(state.solutions, key, %{
        correction: what_is_right,
        timestamp: DateTime.utc_now(),
        source: :user_correction
      })

    state = %{
      state
      | solutions: solutions,
        metrics: %{state.metrics | patterns_captured: state.metrics.patterns_captured + 1}
    }

    persist_solutions(state)

    emitter().emit(:system_event, %{
      event: :learning_correction,
      what: what_was_wrong,
      correction: what_is_right
    })

    {:noreply, state}
  end

  @impl true
  def handle_cast({:error, tool_name, error_message, context}, state) do
    error_type = classify_error(error_message)

    episode = %{
      timestamp: DateTime.utc_now(),
      type: :error,
      tool_name: tool_name,
      error_type: error_type,
      error_message: String.slice(error_message, 0, 500),
      context: context
    }

    episodes = [episode | state.episodes] |> Enum.take(@max_episodes)

    case Map.get(state.solutions, error_type) do
      nil ->
        Logger.debug("[OptimalEngine.Memory.Learning] New error type: #{error_type}")

      solution ->
        Logger.info(
          "[OptimalEngine.Memory.Learning] Known solution for #{error_type}: #{inspect(solution.correction)}"
        )

        emitter().emit(:system_event, %{
          event: :learning_recovery_available,
          error_type: error_type,
          solution: solution.correction
        })
    end

    state = %{
      state
      | episodes: episodes,
        metrics: %{state.metrics | errors_recovered: state.metrics.errors_recovered + 1}
    }

    {:noreply, state}
  end

  @impl true
  def handle_cast(:consolidate, state), do: {:noreply, full_consolidation(state)}

  @impl true
  def handle_call(:metrics, _from, state), do: {:reply, state.metrics, state}

  @impl true
  def handle_call(:patterns, _from, state), do: {:reply, state.patterns, state}

  @impl true
  def handle_call(:solutions, _from, state), do: {:reply, state.solutions, state}

  @impl true
  def handle_info(:cleanup_working_memory, state) do
    now = System.monotonic_time(:millisecond)
    cutoff = now - @working_memory_ttl

    :ets.foldl(
      fn {key, _ep, ts}, acc ->
        if ts < cutoff, do: :ets.delete(state.working_memory, key)
        acc
      end,
      :ok,
      state.working_memory
    )

    Process.send_after(self(), :cleanup_working_memory, @working_memory_ttl)
    {:noreply, state}
  end

  @impl true
  def handle_info(_msg, state), do: {:noreply, state}

  # ── Consolidation ────────────────────────────────────────────────────

  defp incremental_consolidation(state) do
    recent = Enum.take(state.episodes, @consolidation_interval)

    tool_patterns =
      recent
      |> Enum.filter(&(&1.type == :tool_use))
      |> Enum.group_by(& &1.tool_name)
      |> Enum.map(fn {tool, uses} ->
        avg_duration =
          uses
          |> Enum.map(&(&1.duration_ms || 0))
          |> then(fn ds -> if ds == [], do: 0, else: div(Enum.sum(ds), length(ds)) end)

        success_rate = Enum.count(uses, & &1.success) / max(length(uses), 1)

        {"tool:#{tool}",
         %{count: length(uses), avg_duration: avg_duration, success_rate: success_rate}}
      end)

    patterns =
      Enum.reduce(tool_patterns, state.patterns, fn {key, info}, acc ->
        existing = Map.get(acc, key, %{count: 0})

        Map.put(acc, key, %{
          count: existing.count + info.count,
          avg_duration: info.avg_duration,
          success_rate: info.success_rate,
          last_seen: DateTime.utc_now()
        })
      end)

    state = check_skill_generation(patterns, state)

    state = %{
      state
      | patterns: patterns,
        metrics: %{state.metrics | consolidations: state.metrics.consolidations + 1}
    }

    persist_patterns(state)
    persist_solutions(state)
    state
  end

  defp full_consolidation(state) do
    Logger.info(
      "[OptimalEngine.Memory.Learning] Full consolidation (#{length(state.episodes)} episodes, #{map_size(state.patterns)} patterns)"
    )

    error_episodes = Enum.filter(state.episodes, &(&1.type == :error))
    error_groups = Enum.group_by(error_episodes, & &1.error_type)

    solutions =
      Enum.reduce(error_groups, state.solutions, fn {error_type, errors}, acc ->
        if length(errors) >= @pattern_threshold and not Map.has_key?(acc, error_type) do
          Map.put(acc, error_type, %{
            correction: "Recurring error (#{length(errors)}x). Check: #{hd(errors).context}",
            timestamp: DateTime.utc_now(),
            source: :auto_detected
          })
        else
          acc
        end
      end)

    patterns =
      state.patterns
      |> Enum.filter(fn {_key, info} ->
        info.count >= 2 or
          DateTime.diff(DateTime.utc_now(), info[:last_seen] || DateTime.utc_now(), :hour) < 24
      end)
      |> Map.new()

    state = %{
      state
      | patterns: patterns,
        solutions: solutions,
        metrics: %{
          state.metrics
          | consolidations: state.metrics.consolidations + 1,
            patterns_captured: map_size(patterns)
        }
    }

    persist_patterns(state)
    persist_solutions(state)

    emitter().emit(:system_event, %{
      event: :learning_consolidation,
      patterns: map_size(patterns),
      solutions: map_size(solutions),
      episodes: length(state.episodes)
    })

    state
  end

  defp check_skill_generation(patterns, state) do
    candidates =
      patterns
      |> Enum.filter(fn {_key, info} -> info.count >= @skill_generation_threshold end)
      |> Enum.reject(fn {key, _info} -> String.starts_with?(key, "tool:") end)

    if candidates != [] do
      Logger.info(
        "[OptimalEngine.Memory.Learning] #{length(candidates)} skill generation candidates detected"
      )

      emitter().emit(:system_event, %{
        event: :learning_skill_candidates,
        count: length(candidates),
        candidates: Enum.map(candidates, fn {key, _} -> key end)
      })

      %{
        state
        | metrics: %{
            state.metrics
            | skills_generated: state.metrics.skills_generated + length(candidates)
          }
      }
    else
      state
    end
  end

  # ── Error Classification ──────────────────────────────────────────────

  @error_taxonomy [
    {~r/file.*not found|no such file/i, "file_not_found"},
    {~r/permission denied/i, "permission_denied"},
    {~r/syntax error/i, "syntax_error"},
    {~r/import.*error|module.*not found/i, "import_error"},
    {~r/type.*error|expected.*got/i, "type_error"},
    {~r/network|connection.*refused/i, "network_error"},
    {~r/out of memory|memory/i, "memory_error"},
    {~r/timeout/i, "timeout_error"},
    {~r/compilation.*error|compile.*fail/i, "compilation_error"},
    {~r/test.*fail/i, "test_failure"}
  ]

  defp classify_error(error_message) do
    Enum.find_value(@error_taxonomy, "unknown_error", fn {pattern, type} ->
      if Regex.match?(pattern, error_message), do: type
    end)
  end

  # ── Persistence ──────────────────────────────────────────────────────

  defp learning_dir do
    dir = Path.expand(Application.get_env(:optimal_engine, :learning_dir, "~/.osa/learning"))
    File.mkdir_p!(dir)
    dir
  end

  defp persist_patterns(state) do
    path = Path.join(learning_dir(), "patterns.json")

    data =
      Enum.map(state.patterns, fn {key, info} ->
        %{key: key, count: info.count, last_seen: to_string(info[:last_seen] || DateTime.utc_now())}
      end)

    File.write!(path, Jason.encode!(data, pretty: true))
  rescue
    e ->
      Logger.warning(
        "[OptimalEngine.Memory.Learning] Failed to persist patterns: #{Exception.message(e)}"
      )
  end

  defp persist_solutions(state) do
    path = Path.join(learning_dir(), "solutions.json")

    data =
      Enum.map(state.solutions, fn {key, info} ->
        %{key: key, correction: info.correction, source: to_string(info.source)}
      end)

    File.write!(path, Jason.encode!(data, pretty: true))
  rescue
    e ->
      Logger.warning(
        "[OptimalEngine.Memory.Learning] Failed to persist solutions: #{Exception.message(e)}"
      )
  end

  defp load_persisted(state) do
    patterns = load_json("patterns.json", [])
    solutions = load_json("solutions.json", [])

    patterns_map =
      Enum.reduce(patterns, %{}, fn entry, acc ->
        Map.put(acc, entry["key"], %{count: entry["count"] || 0, last_seen: DateTime.utc_now()})
      end)

    solutions_map =
      Enum.reduce(solutions, %{}, fn entry, acc ->
        source =
          try do
            String.to_existing_atom(entry["source"] || "loaded")
          rescue
            ArgumentError -> :loaded
          end

        Map.put(acc, entry["key"], %{
          correction: entry["correction"],
          source: source,
          timestamp: DateTime.utc_now()
        })
      end)

    %{state | patterns: patterns_map, solutions: solutions_map}
  end

  defp load_json(filename, default) do
    path = Path.join(learning_dir(), filename)

    if File.exists?(path) do
      case File.read!(path) |> Jason.decode() do
        {:ok, data} when is_list(data) -> data
        _ -> default
      end
    else
      default
    end
  rescue
    _ -> default
  end

  defp normalize_key(text) do
    text
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9\s]/, "")
    |> String.split()
    |> Enum.take(5)
    |> Enum.join("_")
  end

  defp emitter do
    Application.get_env(:optimal_engine, :event_emitter, OptimalEngine.Memory.NullEmitter)
  end
end
