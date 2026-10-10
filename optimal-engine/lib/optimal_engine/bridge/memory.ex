defmodule OptimalEngine.Bridge.Memory do
  @moduledoc """
  Bridge to OptimalEngine.Memory — episodic events, SICA learning, Cortex synthesis, and injection.

  OptimalEngine.Memory's GenServers (Store, Episodic, Cortex, Learning) are started in
  OptimalEngine's supervision tree. This module provides a unified API.

  ## Integration points

  - `record_event/3` — Records temporal events (intake, search, mutation) via Episodic
  - `observe_mutation/1` — Feeds engine mutations to SICA learning loop
  - `record_correction/2` — Records user corrections for SICA
  - `record_error/3` — Records errors for VIGIL recovery
  - `bulletin/0` — Returns Cortex synthesis bulletin for L0 cache
  - `inject_context/2` — Uses Injector to select relevant memories for a query
  - `learning_metrics/0` — Returns SICA learning stats
  """

  require Logger

  # ---------------------------------------------------------------------------
  # Episodic Events — temporal event recording
  # ---------------------------------------------------------------------------

  @doc """
  Records a temporal event in the episodic memory.

  Event types: :intake, :search, :mutation, :decision, :error, :tool_call
  """
  @spec record_event(atom(), map(), String.t()) :: :ok
  def record_event(event_type, data, session_id \\ "optimal-engine") do
    key =
      "event_#{DateTime.utc_now() |> DateTime.to_unix(:microsecond)}_#{System.unique_integer([:positive])}"

    OptimalEngine.Memory.store(
      "events",
      key,
      %{
        type: event_type,
        data: data,
        session_id: session_id,
        created_at: DateTime.utc_now()
      },
      tags: [to_string(event_type), session_id]
    )

    :ok
  rescue
    _ -> :ok
  end

  @doc "Recalls recent events matching a query."
  @spec recall_events(String.t(), keyword()) :: [map()]
  def recall_events(query, opts \\ []) do
    limit = Keyword.get(opts, :limit, 20)

    case OptimalEngine.Memory.search("events", query) do
      {:ok, entries} ->
        entries
        |> Enum.map(& &1.value)
        |> Enum.take(limit)

      events when is_list(events) ->
        Enum.take(events, limit)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  # ---------------------------------------------------------------------------
  # SICA Learning — self-improvement
  # ---------------------------------------------------------------------------

  @doc """
  Observes an engine mutation for pattern detection.

  The interaction map should contain:
  - :tool — the operation type (e.g., "intake", "search", "index")
  - :input — what was given
  - :output — what was produced
  - :success — boolean
  """
  @spec observe_mutation(map()) :: :ok
  def observe_mutation(interaction) when is_map(interaction) do
    OptimalEngine.Memory.Learning.observe(interaction)
    :ok
  rescue
    _ -> :ok
  end

  @doc "Records a user correction for SICA learning."
  @spec record_correction(String.t(), String.t()) :: :ok
  def record_correction(original, corrected) do
    OptimalEngine.Memory.Learning.correction(original, corrected)
    :ok
  rescue
    _ -> :ok
  end

  @doc "Records an error for VIGIL recovery patterns."
  @spec record_error(String.t(), String.t(), map()) :: :ok
  def record_error(tool_name, error_message, context \\ %{}) do
    OptimalEngine.Memory.Learning.error(tool_name, error_message, context)
    :ok
  rescue
    _ -> :ok
  end

  @doc "Returns SICA learning metrics."
  @spec learning_metrics() :: map()
  def learning_metrics do
    OptimalEngine.Memory.Learning.metrics()
  rescue
    _ -> %{interactions: 0, patterns: 0, skills: 0, errors: 0}
  end

  @doc "Returns detected patterns from SICA."
  @spec patterns() :: map()
  def patterns do
    OptimalEngine.Memory.Learning.patterns()
  rescue
    _ -> %{}
  end

  @doc "Returns known error solutions from SICA."
  @spec solutions() :: map()
  def solutions do
    OptimalEngine.Memory.Learning.solutions()
  rescue
    _ -> %{}
  end

  # ---------------------------------------------------------------------------
  # Cortex Synthesis — LLM-powered bulletins
  # ---------------------------------------------------------------------------

  @doc """
  Returns the current Cortex synthesis bulletin.

  This is a 5-section summary (Focus/Pending/Decisions/Patterns/Context)
  synthesized across sessions. Cached and auto-refreshed.
  """
  @spec bulletin() :: String.t()
  def bulletin do
    case OptimalEngine.Memory.Cortex.bulletin() do
      bulletin when is_binary(bulletin) and bulletin != "" -> bulletin
      _ -> ""
    end
  rescue
    _ -> ""
  end

  @doc "Forces Cortex to re-synthesize its bulletin."
  @spec refresh_bulletin() :: :ok
  def refresh_bulletin do
    OptimalEngine.Memory.Cortex.refresh()
    :ok
  rescue
    _ -> :ok
  end

  @doc "Returns active topics detected across sessions."
  @spec active_topics() :: [String.t()]
  def active_topics do
    case OptimalEngine.Memory.Cortex.active_topics() do
      topics when is_list(topics) -> topics
      _ -> []
    end
  rescue
    _ -> []
  end

  # ---------------------------------------------------------------------------
  # Memory Injection — context-aware memory selection
  # ---------------------------------------------------------------------------

  @doc """
  Selects and formats relevant memories for a given context.

  The context map can include:
  - :files — list of file paths being worked on
  - :task — current task description
  - :error — current error message
  - :session_id — current session
  """
  @spec inject_context(list(), map()) :: String.t()
  def inject_context(entries, context) when is_list(entries) and is_map(context) do
    case OptimalEngine.Memory.Injector.inject_relevant(entries, context) do
      selected when is_list(selected) ->
        OptimalEngine.Memory.Injector.format_for_prompt(selected)

      _ ->
        ""
    end
  rescue
    _ -> ""
  end

  # ---------------------------------------------------------------------------
  # Store — long-term memory persistence
  # ---------------------------------------------------------------------------

  @doc "Saves an insight to long-term memory."
  @spec remember(String.t(), keyword()) :: :ok
  def remember(insight, opts \\ []) do
    OptimalEngine.Memory.remember(insight, opts)
    :ok
  rescue
    _ -> :ok
  end

  @doc "Recalls all long-term memories."
  @spec recall() :: String.t()
  def recall do
    case OptimalEngine.Memory.recall() do
      content when is_binary(content) -> content
      _ -> ""
    end
  rescue
    _ -> ""
  end

  @doc "Searches memories by query."
  @spec search(String.t(), keyword()) :: [map()]
  def search(query, opts \\ []) do
    case OptimalEngine.Memory.search(query, opts) do
      results when is_list(results) -> results
      _ -> []
    end
  rescue
    _ -> []
  end
end
