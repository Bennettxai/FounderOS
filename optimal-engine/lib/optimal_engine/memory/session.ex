defmodule OptimalEngine.Memory.Session do
  @moduledoc """
  Session memory management for conversation history.

  Each session is a GenServer that tracks messages and supports
  persistence to disk in JSONL format.

  ## Configuration

      config :optimal_engine, OptimalEngine.Memory.Session,
        session_path: "~/.miosa/sessions",
        auto_persist_interval: 10
  """

  use GenServer

  @type session_id :: String.t()
  @type role :: :system | :user | :assistant | :tool
  @type message :: %{
          role: role(),
          content: String.t(),
          timestamp: DateTime.t()
        }

  defstruct id: nil,
            messages: [],
            message_count: 0,
            created_at: nil

  # --- Public API ---

  @doc """
  Start a new session with the given ID.

  The session is started under `OptimalEngine.Memory.SessionSupervisor`.
  """
  @spec start_session(session_id()) :: {:ok, pid()} | {:error, term()}
  def start_session(session_id) do
    DynamicSupervisor.start_child(
      OptimalEngine.Memory.SessionSupervisor,
      {__MODULE__, session_id}
    )
  end

  @doc """
  Add a message to the session.

  ## Parameters
    - `session_id` - the session identifier
    - `role` - one of `:system`, `:user`, `:assistant`, `:tool`
    - `content` - the message content string
  """
  @spec add_message(session_id(), role(), String.t()) :: :ok
  def add_message(session_id, role, content) do
    GenServer.call(via(session_id), {:add_message, role, content})
  end

  @doc """
  Get all messages in the session.
  """
  @spec messages(session_id()) :: [message()]
  def messages(session_id) do
    GenServer.call(via(session_id), :messages)
  end

  @doc """
  Get the last N messages in the session.
  """
  @spec messages(session_id(), pos_integer()) :: [message()]
  def messages(session_id, n) do
    GenServer.call(via(session_id), {:messages, n})
  end

  @doc """
  Get a truncated summary of the session.

  Returns the first system message (if any) plus the last 5 messages,
  with a marker indicating how many messages were omitted.
  """
  @spec summarize(session_id()) :: String.t()
  def summarize(session_id) do
    GenServer.call(via(session_id), :summarize)
  end

  @doc """
  Persist the session to disk in JSONL format.
  """
  @spec persist(session_id()) :: :ok | {:error, term()}
  def persist(session_id) do
    GenServer.call(via(session_id), :persist)
  end

  @doc """
  Load a session from disk. Starts the session GenServer if not already running.
  """
  @spec load(session_id()) :: {:ok, pid()} | {:error, term()}
  def load(session_id) do
    case start_session(session_id) do
      {:ok, pid} ->
        GenServer.call(via(session_id), :load_from_disk)
        {:ok, pid}

      {:error, {:already_started, pid}} ->
        GenServer.call(via(session_id), :load_from_disk)
        {:ok, pid}

      error ->
        error
    end
  end

  @doc """
  Stop a session GenServer.
  """
  @spec stop(session_id()) :: :ok
  def stop(session_id) do
    GenServer.stop(via(session_id))
  end

  # --- GenServer ---

  def start_link(session_id) do
    GenServer.start_link(__MODULE__, session_id, name: via(session_id))
  end

  def child_spec(session_id) do
    %{
      id: {__MODULE__, session_id},
      start: {__MODULE__, :start_link, [session_id]},
      restart: :temporary
    }
  end

  @impl GenServer
  def init(session_id) do
    state = %__MODULE__{
      id: session_id,
      messages: [],
      message_count: 0,
      created_at: DateTime.utc_now()
    }

    {:ok, state}
  end

  @impl GenServer
  def handle_call({:add_message, role, content}, _from, state) do
    message = %{
      role: role,
      content: content,
      timestamp: DateTime.utc_now()
    }

    new_count = state.message_count + 1

    new_state = %{
      state
      | messages: state.messages ++ [message],
        message_count: new_count
    }

    # Auto-persist every N messages
    if auto_persist?() and rem(new_count, auto_persist_interval()) == 0 do
      do_persist(new_state)
    end

    {:reply, :ok, new_state}
  end

  @impl GenServer
  def handle_call(:messages, _from, state) do
    {:reply, state.messages, state}
  end

  @impl GenServer
  def handle_call({:messages, n}, _from, state) do
    {:reply, Enum.take(state.messages, -n), state}
  end

  @impl GenServer
  def handle_call(:summarize, _from, state) do
    summary = build_summary(state.messages)
    {:reply, summary, state}
  end

  @impl GenServer
  def handle_call(:persist, _from, state) do
    result = do_persist(state)
    {:reply, result, state}
  end

  @impl GenServer
  def handle_call(:load_from_disk, _from, state) do
    case do_load(state.id) do
      {:ok, messages} ->
        new_state = %{
          state
          | messages: messages,
            message_count: length(messages)
        }

        {:reply, :ok, new_state}

      {:error, _} ->
        {:reply, :ok, state}
    end
  end

  # --- Private ---

  defp via(session_id) do
    {:via, Registry, {OptimalEngine.Memory.SessionRegistry, session_id}}
  end

  defp build_summary(messages) do
    system_msgs = Enum.filter(messages, &(&1.role == :system))
    recent = Enum.take(messages, -5)
    total = length(messages)
    omitted = max(0, total - length(system_msgs) - 5)

    parts =
      Enum.map(system_msgs, fn m -> "[system] #{m.content}" end) ++
        if(omitted > 0, do: ["... (#{omitted} messages omitted) ..."], else: []) ++
        Enum.map(recent, fn m -> "[#{m.role}] #{truncate(m.content, 200)}" end)

    Enum.join(parts, "\n")
  end

  defp truncate(text, max_len) when byte_size(text) <= max_len, do: text

  defp truncate(text, max_len) do
    String.slice(text, 0, max_len) <> "..."
  end

  defp do_persist(state) do
    dir = session_dir(state.id)
    File.mkdir_p!(dir)
    path = Path.join(dir, "messages.jsonl")

    lines =
      Enum.map(state.messages, fn msg ->
        Jason.encode!(%{
          role: msg.role,
          content: msg.content,
          timestamp: DateTime.to_iso8601(msg.timestamp)
        })
      end)

    File.write!(path, Enum.join(lines, "\n") <> "\n")
    :ok
  end

  defp do_load(session_id) do
    path = Path.join(session_dir(session_id), "messages.jsonl")

    case File.read(path) do
      {:ok, content} ->
        messages =
          content
          |> String.split("\n", trim: true)
          |> Enum.map(fn line ->
            data = Jason.decode!(line, keys: :atoms)
            {:ok, ts, _} = DateTime.from_iso8601(data.timestamp)

            %{
              role: String.to_existing_atom(to_string(data.role)),
              content: data.content,
              timestamp: ts
            }
          end)

        {:ok, messages}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp session_dir(session_id) do
    base =
      config()[:session_path]
      |> Kernel.||("~/.miosa/sessions")
      |> Path.expand()

    Path.join(base, session_id)
  end

  defp auto_persist? do
    config()[:auto_persist] != false
  end

  defp auto_persist_interval do
    config()[:auto_persist_interval] || 10
  end

  defp config do
    Application.get_env(:optimal_engine, __MODULE__, [])
  end
end
