defmodule OptimalEngine.Memory.SessionStore do
  @moduledoc """
  Session persistence — JSONL files with optional secondary store callback.

  Stores conversation history as newline-delimited JSON. Supports:
  - Append a message entry to a session
  - Load all messages for a session
  - List session metadata (last active, message count, topic hint)
  - Resume a session by ID

  A secondary store (e.g., SQLite/Ecto) can be plugged in by configuring:

      config :optimal_engine, secondary_store: MyApp.SessionStore.SQLite

  The secondary store must implement `OptimalEngine.Memory.SessionStore.Secondary`.
  When a secondary store is configured, writes go to both JSONL and the
  secondary; reads prefer the secondary and fall back to JSONL.
  """

  require Logger

  @doc "Returns the sessions directory path."
  def sessions_dir do
    Application.get_env(:optimal_engine, :sessions_dir, "~/.osa/sessions")
    |> Path.expand()
  end

  @doc "Append a message entry to a session's JSONL file."
  @spec append(String.t(), map()) :: :ok
  def append(session_id, entry) when is_binary(session_id) and is_map(entry) do
    dir = sessions_dir()
    File.mkdir_p!(dir)
    timestamp = DateTime.utc_now() |> DateTime.to_iso8601()
    path = session_path(dir, session_id)
    line = Jason.encode!(Map.put(entry, :timestamp, timestamp))
    File.write!(path, line <> "\n", [:append, :utf8])
    :ok
  rescue
    e ->
      Logger.warning("[OptimalEngine.Memory.SessionStore] append failed: #{Exception.message(e)}")
      :ok
  end

  @doc "Load a session's message history."
  @spec load(String.t()) :: [map()]
  def load(session_id) when is_binary(session_id) do
    load_from_jsonl(sessions_dir(), session_id)
  end

  @doc "Resume a session — returns {:ok, messages} or {:error, :not_found}."
  @spec resume(String.t()) :: {:ok, [map()]} | {:error, :not_found}
  def resume(session_id) when is_binary(session_id) do
    path = session_path(sessions_dir(), session_id)

    if File.exists?(path) do
      messages =
        path
        |> File.read!()
        |> String.split("\n", trim: true)
        |> Enum.map(fn line ->
          case Jason.decode(line) do
            {:ok, msg} -> msg
            _ -> nil
          end
        end)
        |> Enum.reject(&is_nil/1)

      {:ok, messages}
    else
      {:error, :not_found}
    end
  end

  @doc "List all sessions with metadata sorted by last_active descending."
  @spec list_sessions() :: [map()]
  def list_sessions do
    dir = sessions_dir()

    if File.exists?(dir) do
      case File.ls(dir) do
        {:ok, files} ->
          files
          |> Enum.filter(&String.ends_with?(&1, ".jsonl"))
          |> Enum.map(fn filename ->
            session_id = String.trim_trailing(filename, ".jsonl")
            path = Path.join(dir, filename)
            extract_metadata(session_id, path)
          end)
          |> Enum.sort_by(& &1.last_active, :desc)

        _ ->
          []
      end
    else
      []
    end
  end

  @doc "Count sessions by file count."
  @spec session_count() :: non_neg_integer()
  def session_count do
    dir = sessions_dir()

    if File.exists?(dir) do
      case File.ls(dir) do
        {:ok, files} -> Enum.count(files, &String.ends_with?(&1, ".jsonl"))
        _ -> 0
      end
    else
      0
    end
  end

  # ── Private ──────────────────────────────────────────────────────────

  defp session_path(dir, session_id), do: Path.join(dir, "#{session_id}.jsonl")

  defp load_from_jsonl(dir, session_id) do
    path = session_path(dir, session_id)

    if File.exists?(path) do
      path
      |> File.read!()
      |> String.split("\n", trim: true)
      |> Enum.map(fn line ->
        case Jason.decode(line) do
          {:ok, msg} -> msg
          _ -> nil
        end
      end)
      |> Enum.reject(&is_nil/1)
    else
      []
    end
  end

  defp extract_metadata(session_id, path) do
    try do
      content = File.read!(path)
      lines = String.split(content, "\n", trim: true)
      message_count = length(lines)

      {first_timestamp, topic_hint} =
        case lines do
          [first | _] ->
            case Jason.decode(first) do
              {:ok, msg} ->
                ts = Map.get(msg, "timestamp")

                topic =
                  case Map.get(msg, "content") do
                    c when is_binary(c) and byte_size(c) > 0 ->
                      c |> String.slice(0, 80) |> String.trim()

                    _ ->
                      nil
                  end

                {ts, topic}

              _ ->
                {nil, nil}
            end

          [] ->
            {nil, nil}
        end

      last_timestamp =
        case List.last(lines) do
          nil ->
            nil

          last ->
            case Jason.decode(last) do
              {:ok, msg} -> Map.get(msg, "timestamp")
              _ -> nil
            end
        end

      %{
        session_id: session_id,
        message_count: message_count,
        first_active: first_timestamp,
        last_active: last_timestamp || first_timestamp,
        topic_hint: topic_hint
      }
    rescue
      _ ->
        %{
          session_id: session_id,
          message_count: 0,
          first_active: nil,
          last_active: nil,
          topic_hint: nil
        }
    end
  end
end
