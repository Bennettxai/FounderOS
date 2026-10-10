defmodule OptimalEngine.Memory.Cortex do
  @moduledoc """
  Knowledge synthesis engine — produces targeted memory bulletins and tracks cross-session patterns.

  Instead of summarizing ALL of memory into a 200-word blob, the Cortex:

  1. **Tracks active topics** across recent sessions using keyword frequency analysis
  2. **Generates targeted bulletins** with structured sections:
     - Current Focus, Pending Items, Key Decisions, Patterns, Context
  3. **Detects cross-session patterns** (recurring topics, communication style shifts)
  4. **Maintains per-session summaries** for fast cross-session awareness

  The synthesis runs on a configurable interval (default 5 minutes).
  `bulletin/0` is a fast read of cached state — safe to call on every
  context build. `refresh/0` forces an immediate re-synthesis.

  ## LLM Provider

  Configure via application env:

      config :optimal_engine, cortex_provider: MyApp.LLMBridge

  The provider must implement `OptimalEngine.Memory.Cortex.Provider` (a single
  `chat/2` callback). Defaults to no-op (no bulletin synthesis).
  """

  use GenServer
  require Logger

  alias OptimalEngine.Memory.{Store, Index}

  @default_refresh_interval 300_000
  @boot_delay 300_000
  @max_recent_sessions 5
  @topic_ets_table :optimal_engine_cortex_topics

  @synthesis_prompt_template """
  You are a knowledge synthesis engine for an AI agent. Based on the following context, produce a structured bulletin that will be injected into the agent's system prompt.

  ## Active Sessions (most recent)
  %SESSION_SECTION%

  ## Long-term Memory (recent entries)
  %TRIMMED_MEMORY%

  ## Detected Active Topics
  %TOPICS_SECTION%

  Produce a bulletin with EXACTLY these sections. Be concise and actionable — the agent reads this before every response.

  1. **Current Focus**: What is the user actively working on right now? (1-3 bullets)
  2. **Pending Items**: Any open questions, unfinished tasks, or follow-ups needed? (1-3 bullets)
  3. **Key Decisions**: Recent decisions or preferences that should inform responses (1-3 bullets)
  4. **Patterns**: Notable patterns — recurring topics, workflow habits, communication preferences (1-2 bullets)
  5. **Context**: Important background facts the agent should keep in mind (1-2 bullets)

  Keep each section to 1-3 bullet points maximum. Total bulletin should be under 300 words.
  If a section has no relevant content, write "None detected" for that section.
  Do NOT include the raw data — synthesize it into actionable intelligence.
  """

  defstruct bulletin: nil,
            active_topics: [],
            session_summaries: %{},
            last_refresh: nil,
            refresh_interval: @default_refresh_interval,
            timer_ref: nil

  # ── Public API ───────────────────────────────────────────────────────

  @doc "Returns the current memory bulletin string, or nil if none is available."
  @spec bulletin() :: String.t() | nil
  def bulletin, do: GenServer.call(__MODULE__, :bulletin)

  @doc "Force an immediate bulletin refresh (asynchronous)."
  @spec refresh() :: :ok
  def refresh, do: GenServer.cast(__MODULE__, :refresh)

  @doc "Get active topics detected across recent sessions."
  @spec active_topics() :: [map()]
  def active_topics, do: GenServer.call(__MODULE__, :active_topics)

  @doc "Get a one-line summary of a specific session."
  @spec session_summary(String.t()) :: String.t() | nil
  def session_summary(session_id), do: GenServer.call(__MODULE__, {:session_summary, session_id})

  @doc "Get synthesis statistics."
  @spec synthesis_stats() :: map()
  def synthesis_stats, do: GenServer.call(__MODULE__, :synthesis_stats)

  def start_link(_opts) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @impl true
  def init(_opts) do
    interval =
      Application.get_env(:optimal_engine, :cortex_refresh_interval, @default_refresh_interval)

    ensure_topic_table()
    timer_ref = Process.send_after(self(), :refresh, @boot_delay)

    state = %__MODULE__{refresh_interval: interval, timer_ref: timer_ref}

    Logger.info(
      "[OptimalEngine.Memory.Cortex] started — first synthesis in #{div(@boot_delay, 60_000)}min, interval #{div(interval, 60_000)}min"
    )

    {:ok, state}
  end

  @impl true
  def handle_call(:bulletin, _from, state), do: {:reply, state.bulletin, state}

  @impl true
  def handle_call(:active_topics, _from, state), do: {:reply, state.active_topics, state}

  @impl true
  def handle_call({:session_summary, session_id}, _from, state) do
    {:reply, Map.get(state.session_summaries, session_id), state}
  end

  @impl true
  def handle_call(:synthesis_stats, _from, state) do
    stats = %{
      last_refresh: state.last_refresh,
      bulletin_bytes: if(state.bulletin, do: byte_size(state.bulletin), else: 0),
      active_topic_count: length(state.active_topics),
      session_summaries_count: map_size(state.session_summaries),
      refresh_interval_ms: state.refresh_interval,
      has_bulletin: state.bulletin != nil
    }

    {:reply, stats, state}
  end

  @impl true
  def handle_cast(:refresh, state) do
    state = cancel_timer(state)
    dispatch_synthesis(state)
    timer_ref = Process.send_after(self(), :refresh, state.refresh_interval)
    {:noreply, %{state | timer_ref: timer_ref}}
  end

  @impl true
  def handle_cast({:synthesis_done, updates}, state) do
    {:noreply, Map.merge(state, updates)}
  end

  @impl true
  def handle_info(:refresh, state) do
    state = cancel_timer(state)
    dispatch_synthesis(state)
    timer_ref = Process.send_after(self(), :refresh, state.refresh_interval)
    {:noreply, %{state | timer_ref: timer_ref}}
  end

  # ── Synthesis Engine ─────────────────────────────────────────────────

  defp dispatch_synthesis(state) do
    self_pid = self()
    session_summaries = state.session_summaries

    Task.start(fn ->
      updates = run_synthesis(session_summaries)
      GenServer.cast(self_pid, {:synthesis_done, updates})
    end)
  end

  defp run_synthesis(session_summaries) do
    try do
      memory_content = Store.recall()
      recent_sessions = get_recent_sessions(@max_recent_sessions)
      active_topics = extract_topics(memory_content, recent_sessions)
      update_topic_table(active_topics)
      new_summaries = update_session_summaries(session_summaries, recent_sessions)

      if memory_content == "" and recent_sessions == [] do
        Logger.debug("[OptimalEngine.Memory.Cortex] no material for synthesis, skipping")

        %{
          last_refresh: DateTime.utc_now(),
          active_topics: active_topics,
          session_summaries: new_summaries
        }
      else
        maybe_synthesize_bulletin(memory_content, recent_sessions, active_topics, new_summaries)
      end
    rescue
      e ->
        Logger.error("[OptimalEngine.Memory.Cortex] synthesis crashed — #{inspect(e)}")
        %{last_refresh: DateTime.utc_now()}
    end
  end

  defp maybe_synthesize_bulletin(memory_content, recent_sessions, active_topics, new_summaries) do
    base_updates = %{
      active_topics: active_topics,
      session_summaries: new_summaries,
      last_refresh: DateTime.utc_now()
    }

    case Application.get_env(:optimal_engine, :cortex_provider) do
      nil ->
        base_updates

      provider ->
        messages =
          build_synthesis_messages(memory_content, recent_sessions, active_topics, new_summaries)

        case provider.chat(messages, max_tokens: 500, temperature: 0.2) do
          {:ok, %{content: content}} when is_binary(content) and content != "" ->
            bulletin = String.trim(content)

            Logger.info(
              "[OptimalEngine.Memory.Cortex] bulletin refreshed (#{byte_size(bulletin)} bytes), #{length(active_topics)} active topics"
            )

            Map.put(base_updates, :bulletin, bulletin)

          {:ok, _} ->
            Logger.warning("[OptimalEngine.Memory.Cortex] LLM returned empty content")
            base_updates

          {:error, reason} ->
            Logger.warning("[OptimalEngine.Memory.Cortex] synthesis failed — #{inspect(reason)}")
            base_updates
        end
    end
  end

  # ── Topic Extraction ─────────────────────────────────────────────────

  defp extract_topics(memory_content, recent_sessions) do
    memory_keywords =
      if memory_content != "", do: Index.extract_keywords(memory_content), else: []

    session_keywords =
      Enum.flat_map(recent_sessions, fn session ->
        session[:messages]
        |> Enum.filter(fn msg -> msg["role"] == "user" end)
        |> Enum.flat_map(fn msg -> Index.extract_keywords(msg["content"] || "") end)
      end)

    frequencies =
      Enum.reduce(memory_keywords, %{}, fn kw, acc -> Map.update(acc, kw, 1, &(&1 + 1)) end)

    frequencies =
      Enum.reduce(session_keywords, frequencies, fn kw, acc -> Map.update(acc, kw, 2, &(&1 + 2)) end)

    now = DateTime.utc_now() |> DateTime.to_iso8601()

    frequencies
    |> Enum.filter(fn {_kw, freq} -> freq >= 2 end)
    |> Enum.sort_by(fn {_kw, freq} -> freq end, :desc)
    |> Enum.take(15)
    |> Enum.map(fn {topic, frequency} -> %{topic: topic, frequency: frequency, last_seen: now} end)
  end

  # ── Session Summaries ────────────────────────────────────────────────

  defp update_session_summaries(existing_summaries, recent_sessions) do
    Enum.reduce(recent_sessions, existing_summaries, fn session, acc ->
      session_id = session[:session_id]
      current_summary = Map.get(acc, session_id)

      if current_summary == nil and session[:messages] != [] do
        Map.put(acc, session_id, generate_local_summary(session))
      else
        acc
      end
    end)
  end

  defp generate_local_summary(session) do
    messages = session[:messages] || []
    user_messages = Enum.filter(messages, fn m -> m["role"] == "user" end)

    first_topic =
      case user_messages do
        [first | _] -> first["content"] || "" |> String.slice(0, 100) |> String.trim()
        [] -> "empty session"
      end

    "#{length(user_messages)} exchange(s) — #{first_topic}"
  end

  defp get_recent_sessions(count) do
    try do
      Store.list_sessions()
      |> Enum.take(count)
      |> Enum.map(fn session_meta ->
        messages =
          case Store.load_session(session_meta.session_id) do
            msgs when is_list(msgs) -> msgs
            _ -> []
          end

        Map.put(session_meta, :messages, messages)
      end)
    rescue
      _ -> []
    end
  end

  # ── Synthesis Prompt ─────────────────────────────────────────────────

  defp build_synthesis_messages(memory_content, recent_sessions, active_topics, session_summaries) do
    trimmed_memory = trim_content(memory_content, 3000)

    session_section =
      if recent_sessions != [] do
        recent_sessions
        |> Enum.map(fn session ->
          sid = session[:session_id] || "unknown"
          summary = Map.get(session_summaries, sid, "no summary")
          last = session[:last_active] || "unknown"
          "- **#{sid}** (last active: #{last}): #{summary}"
        end)
        |> Enum.join("\n")
      else
        "No recent sessions."
      end

    topics_section =
      if active_topics != [] do
        active_topics
        |> Enum.map(fn t -> "- #{t.topic} (frequency: #{t.frequency})" end)
        |> Enum.join("\n")
      else
        "No active topics detected yet."
      end

    prompt =
      @synthesis_prompt_template
      |> String.replace("%SESSION_SECTION%", session_section)
      |> String.replace("%TRIMMED_MEMORY%", trimmed_memory)
      |> String.replace("%TOPICS_SECTION%", topics_section)

    [%{role: "user", content: prompt}]
  end

  defp trim_content(content, max_chars) when is_binary(content) do
    if byte_size(content) > max_chars do
      binary_part(
        content,
        max(byte_size(content) - max_chars, 0),
        min(max_chars, byte_size(content))
      )
    else
      content
    end
  end

  defp trim_content(_, _), do: ""

  # ── ETS Topic Table ──────────────────────────────────────────────────

  defp ensure_topic_table do
    if :ets.info(@topic_ets_table) == :undefined do
      :ets.new(@topic_ets_table, [:named_table, :set, :public, read_concurrency: true])
    end
  end

  defp update_topic_table(active_topics) do
    try do
      ensure_topic_table()
      :ets.delete_all_objects(@topic_ets_table)
      Enum.each(active_topics, fn topic -> :ets.insert(@topic_ets_table, {topic.topic, topic}) end)
    rescue
      _ -> :ok
    end
  end

  defp cancel_timer(%__MODULE__{timer_ref: nil} = state), do: state

  defp cancel_timer(%__MODULE__{timer_ref: ref} = state) do
    Process.cancel_timer(ref)
    %{state | timer_ref: nil}
  end
end
