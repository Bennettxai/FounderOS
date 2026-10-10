defmodule OptimalEngine.Session do
  @moduledoc """
  Session lifecycle management — create, interact, commit.

  A Session tracks the messages in a single conversation turn. When committed,
  it generates a summary, extracts any learned facts (as :memory contexts), and
  archives the session to SQLite.

  ## Lifecycle

      # Start a session
      {:ok, session_id} = Session.start_session()

      # Add messages during the conversation
      :ok = Session.add_message(session_id, :user, "Customer called about pricing")
      :ok = Session.add_message(session_id, :assistant, "Logged to project-platform-launch/signal.md")

      # Commit — extracts memories, generates summary, archives
      {:ok, summary} = Session.commit(session_id)

      # Get the context string for loading into agent context
      context_str = Session.get_context(session_id)

  ## Process architecture

  Each session is a lightweight GenServer registered under its ID in the
  `OptimalEngine.SessionRegistry`. The `SessionSupervisor` (DynamicSupervisor)
  manages session lifecycle.

  Sessions auto-expire after `@idle_timeout_ms` of inactivity.
  """

  use GenServer
  require Logger

  alias OptimalEngine.Insight.MemoryExtractor, as: MemoryExtractor
  alias OptimalEngine.SessionCompressor

  @idle_timeout_ms 60 * 60 * 1_000
  @max_messages 200

  defstruct [
    :id,
    :started_at,
    messages: [],
    committed: false,
    summary: "",
    metadata: %{}
  ]

  @type message :: %{role: :user | :assistant | :system, content: String.t(), at: DateTime.t()}

  # ---------------------------------------------------------------------------
  # Client API
  # ---------------------------------------------------------------------------

  @doc """
  Starts a new session. Returns `{:ok, session_id}`.

  Options:
  - `:metadata` — map of extra metadata to attach to the session
  """
  @spec start_session(keyword()) :: {:ok, String.t()} | {:error, term()}
  def start_session(opts \\ []) do
    session_id = generate_id()
    metadata = Keyword.get(opts, :metadata, %{})

    child_spec = {__MODULE__, session_id: session_id, metadata: metadata}

    case DynamicSupervisor.start_child(OptimalEngine.SessionSupervisor, child_spec) do
      {:ok, _pid} -> {:ok, session_id}
      {:error, reason} -> {:error, reason}
    end
  end

  @doc "Adds a message to the session."
  @spec add_message(String.t(), :user | :assistant | :system, String.t()) ::
          :ok | {:error, term()}
  def add_message(session_id, role, content)
      when role in [:user, :assistant, :system] and is_binary(content) do
    case lookup(session_id) do
      {:ok, pid} -> GenServer.call(pid, {:add_message, role, content})
      err -> err
    end
  end

  @doc """
  Commits the session. Generates a summary and persists to SQLite.
  Returns `{:ok, summary_string}`.
  """
  @spec commit(String.t()) :: {:ok, String.t()} | {:error, term()}
  def commit(session_id) do
    case lookup(session_id) do
      {:ok, pid} -> GenServer.call(pid, :commit, 15_000)
      err -> err
    end
  end

  @doc "Returns the current session context as a formatted string for agent loading."
  @spec get_context(String.t()) :: String.t()
  def get_context(session_id) do
    case lookup(session_id) do
      {:ok, pid} -> GenServer.call(pid, :get_context)
      _ -> ""
    end
  end

  @doc "Returns session info (id, message count, started_at, committed)."
  @spec info(String.t()) :: {:ok, map()} | {:error, :not_found}
  def info(session_id) do
    case lookup(session_id) do
      {:ok, pid} -> GenServer.call(pid, :info)
      err -> err
    end
  end

  @doc "Lists all active session IDs."
  @spec list_sessions() :: [String.t()]
  def list_sessions do
    Registry.select(OptimalEngine.SessionRegistry, [{{:"$1", :_, :_}, [], [:"$1"]}])
  end

  # ---------------------------------------------------------------------------
  # GenServer child_spec
  # ---------------------------------------------------------------------------

  def child_spec(opts) do
    session_id = Keyword.fetch!(opts, :session_id)

    %{
      id: {__MODULE__, session_id},
      start: {__MODULE__, :start_link, [opts]},
      restart: :temporary
    }
  end

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    session_id = Keyword.fetch!(opts, :session_id)
    GenServer.start_link(__MODULE__, opts, name: via(session_id))
  end

  # ---------------------------------------------------------------------------
  # GenServer Callbacks
  # ---------------------------------------------------------------------------

  @impl true
  def init(opts) do
    session_id = Keyword.fetch!(opts, :session_id)
    metadata = Keyword.get(opts, :metadata, %{})

    state = %__MODULE__{
      id: session_id,
      started_at: DateTime.utc_now(),
      messages: [],
      committed: false,
      summary: "",
      metadata: metadata
    }

    Logger.debug("[Session] Started #{session_id}")
    {:ok, state, @idle_timeout_ms}
  end

  @impl true
  def handle_call({:add_message, _role, _content}, _from, %{committed: true} = state) do
    {:reply, {:error, :session_already_committed}, state, @idle_timeout_ms}
  end

  @impl true
  def handle_call({:add_message, role, content}, _from, state) do
    if length(state.messages) >= @max_messages do
      {:reply, {:error, :max_messages_reached}, state, @idle_timeout_ms}
    else
      msg = %{role: role, content: content, at: DateTime.utc_now()}
      new_state = %{state | messages: state.messages ++ [msg]}
      {:reply, :ok, new_state, @idle_timeout_ms}
    end
  end

  @impl true
  def handle_call(:commit, _from, %{committed: true} = state) do
    {:reply, {:ok, state.summary}, state, @idle_timeout_ms}
  end

  @impl true
  def handle_call(:commit, _from, state) do
    # Compress transcript if the session is large (best-effort, never blocks)
    transcript = render_context(state)

    compressed =
      if SessionCompressor.should_compress?(state.messages) do
        case SessionCompressor.compress(transcript) do
          {:ok, c} -> c
          _ -> transcript
        end
      else
        transcript
      end

    summary = generate_summary(state, compressed)
    new_state = %{state | committed: true, summary: summary}

    persist_session(new_state)

    # Fire-and-forget: extract memories from transcript after commit
    Task.start(fn -> extract_and_store_memories(transcript, state.id) end)

    Logger.info("[Session] Committed #{state.id} (#{length(state.messages)} messages)")
    {:reply, {:ok, summary}, new_state, @idle_timeout_ms}
  end

  @impl true
  def handle_call(:get_context, _from, state) do
    context = render_context(state)
    {:reply, context, state, @idle_timeout_ms}
  end

  @impl true
  def handle_call(:info, _from, state) do
    info = %{
      id: state.id,
      started_at: state.started_at,
      message_count: length(state.messages),
      committed: state.committed,
      summary: state.summary
    }

    {:reply, {:ok, info}, state, @idle_timeout_ms}
  end

  @impl true
  def handle_info(:timeout, state) do
    Logger.debug("[Session] #{state.id} idle timeout — shutting down")
    {:stop, :normal, state}
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp generate_summary(%{messages: []}, _compressed), do: "(empty session)"

  defp generate_summary(%{messages: messages}, compressed) do
    if OptimalEngine.Embed.Ollama.available?() do
      llm_summary(compressed)
    else
      fallback_summary(messages)
    end
  end

  defp llm_summary(transcript) do
    truncated = String.slice(transcript, 0, 2_000)

    prompt =
      "Summarize this conversation in 2-3 sentences. Focus on decisions, action items, and outcomes.\n\nConversation:\n#{truncated}"

    system = "Be concise. Output only the summary."

    case OptimalEngine.Embed.Ollama.generate(prompt, system: system) do
      {:ok, summary} ->
        summary |> String.trim() |> truncate(500)

      {:error, _} ->
        fallback_summary_from_transcript(transcript)
    end
  rescue
    _ -> fallback_summary_from_transcript(transcript)
  end

  defp fallback_summary(messages) do
    user_msgs =
      messages
      |> Enum.filter(&(&1.role == :user))
      |> Enum.map(& &1.content)

    case user_msgs do
      [] -> "(no user messages)"
      msgs -> msgs |> Enum.take(3) |> Enum.join(" | ") |> truncate(200)
    end
  end

  defp fallback_summary_from_transcript(transcript) do
    transcript
    |> String.split("\n")
    |> Enum.filter(&String.match?(&1, ~r/^\*\*USER:\*\*/i))
    |> Enum.map(&String.replace(&1, ~r/^\*\*USER:\*\*\s*/i, ""))
    |> Enum.take(3)
    |> case do
      [] -> "(no user messages)"
      msgs -> msgs |> Enum.join(" | ") |> truncate(200)
    end
  end

  defp render_context(%{messages: [], id: id}) do
    "# Session #{id}\n\n(No messages yet)"
  end

  defp render_context(%{id: id, started_at: started_at, messages: messages}) do
    header =
      "# Session #{id}\n> Started: #{Calendar.strftime(started_at, "%Y-%m-%d %H:%M")} UTC\n"

    body =
      Enum.map_join(messages, "\n\n", fn %{role: role, content: content} ->
        label = role |> to_string() |> String.upcase()
        "**#{label}:** #{content}"
      end)

    header <> "\n" <> body
  end

  defp persist_session(state) do
    sql = """
    INSERT OR REPLACE INTO sessions (id, started_at, committed_at, summary, message_count, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6)
    """

    params = [
      state.id,
      DateTime.to_iso8601(state.started_at),
      DateTime.to_iso8601(DateTime.utc_now()),
      state.summary,
      length(state.messages),
      Jason.encode!(state.metadata)
    ]

    case OptimalEngine.Store.raw_query(sql, params) do
      {:ok, _} -> :ok
      {:error, reason} -> Logger.warning("[Session] Persist failed: #{inspect(reason)}")
    end
  end

  defp extract_and_store_memories(transcript, session_id) do
    case MemoryExtractor.extract(transcript) do
      {:ok, memories} when memories != [] ->
        Enum.each(memories, fn memory ->
          store_memory(memory, session_id)
        end)

      _ ->
        :ok
    end
  rescue
    _ -> :ok
  end

  defp store_memory(memory, _session_id) do
    id = "mem-" <> (:crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower))

    sql = """
    INSERT OR IGNORE INTO contexts (id, uri, type, title, content, node, sn_ratio, genre, created_at, modified_at)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
    """

    now = DateTime.utc_now() |> DateTime.to_iso8601()

    params = [
      id,
      "optimal://user/memories/#{id}",
      "memory",
      "#{memory.category}: #{String.slice(memory.content, 0, 50)}",
      memory.content,
      "inbox",
      memory.confidence,
      to_string(memory.category),
      now,
      now
    ]

    case OptimalEngine.Store.raw_query(sql, params) do
      {:ok, _} -> :ok
      {:error, reason} -> Logger.debug("[Session] store_memory failed: #{inspect(reason)}")
    end
  rescue
    _ -> :ok
  end

  defp lookup(session_id) do
    case Registry.lookup(OptimalEngine.SessionRegistry, session_id) do
      [{pid, _}] -> {:ok, pid}
      [] -> {:error, :not_found}
    end
  end

  defp via(session_id) do
    {:via, Registry, {OptimalEngine.SessionRegistry, session_id}}
  end

  defp generate_id do
    :crypto.strong_rand_bytes(12) |> Base.encode16(case: :lower)
  end

  defp truncate(str, max) do
    if String.length(str) > max do
      String.slice(str, 0, max) <> "..."
    else
      str
    end
  end
end
