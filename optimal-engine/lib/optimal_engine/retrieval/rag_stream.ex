defmodule OptimalEngine.Retrieval.RagStream do
  @moduledoc """
  Pipeline-stage streaming for the RAG retrieval flow.

  Spawns a `Task` that runs `RAG.ask/2` with an `:on_event` hook and
  relays each stage boundary as a message to `listener_pid`:

      {:rag_stream_event, event_atom, payload_map}
      {:rag_stream_done,  %{source, envelope, trace}}
      {:rag_stream_error, reason}

  ## Events (in order)

      :intent     — `{intent_type, expanded_query, key_entities, node_hints}` — after IntentAnalyzer
      :wiki_hit   — `{slug, audience, version}` — only when wiki resolves
      :chunks     — `[{slug, score, scale}]` — only on hybrid retrieval path
      :composing  — `{format, bandwidth}` — when composer starts

  The `:envelope` and `:done` events are delivered by the router loop
  after receiving `{:rag_stream_done, …}` so the router controls the
  final two SSE writes.

  ## Usage

      {:ok, _task} = RagStream.start_link(query, receiver, self(),
        workspace_id: "default"
      )

  Options accepted: any opts that `RAG.ask/2` accepts, plus the above
  listener coupling. The `:on_event` opt is reserved — do not pass it;
  `RagStream` sets it internally.
  """

  alias OptimalEngine.Retrieval.{RAG, Receiver}

  @doc """
  Spawn a Task that runs the RAG pipeline and sends stream messages to
  `listener_pid`.

  Returns `{:ok, %Task{}}`.
  """
  @spec start_link(String.t(), Receiver.t(), pid(), keyword()) :: {:ok, Task.t()}
  def start_link(query, receiver, listener_pid, opts \\ [])
      when is_binary(query) and is_pid(listener_pid) do
    ask_opts =
      opts
      |> Keyword.put(:receiver, receiver)
      |> Keyword.put(:on_event, make_event_sender(listener_pid))

    task =
      Task.async(fn ->
        result =
          try do
            RAG.ask(query, ask_opts)
          rescue
            err -> {:error, Exception.message(err)}
          catch
            :exit, reason -> {:error, reason}
          end

        case result do
          {:ok, payload} ->
            send(listener_pid, {:rag_stream_done, payload})

          {:error, reason} ->
            send(listener_pid, {:rag_stream_error, reason})
        end
      end)

    {:ok, task}
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp make_event_sender(listener_pid) do
    fn event, payload ->
      send(listener_pid, {:rag_stream_event, event, payload})
    end
  end
end
