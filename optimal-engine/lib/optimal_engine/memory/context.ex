defmodule OptimalEngine.Memory.Context do
  @moduledoc """
  MemoryOS context management.

  Builds, injects, and compacts context windows for agent conversations.
  Combines session history with relevant memories to produce an optimal
  context payload within token budget.

  ## Strategies
    - `:recent` - last N messages from session
    - `:relevant` - keyword-matched memories injected into context
    - `:summary` - summarize older messages, keep recent verbatim
  """

  alias OptimalEngine.Memory.{Session, Compactor}

  @type strategy :: :recent | :relevant | :summary
  @type context_entry :: %{role: atom(), content: String.t()}

  @doc """
  Build a context window from session messages and optional memory injections.

  ## Options
    - `:strategy` - one of `:recent`, `:relevant`, `:summary` (default: `:recent`)
    - `:max_tokens` - maximum token budget (default: 100_000)
    - `:recent_count` - number of recent messages for `:recent` strategy (default: 50)
    - `:collections` - list of memory collections to search for `:relevant` strategy
    - `:query` - search query for `:relevant` strategy
  """
  @spec build_context(String.t(), keyword()) :: {:ok, [context_entry()]}
  def build_context(session_id, opts \\ []) do
    strategy = Keyword.get(opts, :strategy, :recent)
    max_tokens = Keyword.get(opts, :max_tokens, 100_000)

    context = do_build(strategy, session_id, opts)

    # Trim to budget
    trimmed = trim_to_budget(context, max_tokens)

    {:ok, trimmed}
  end

  @doc """
  Inject relevant memories into a list of context messages.

  Searches the given collections for the query and prepends matching
  memories as system-role context entries.

  ## Options
    - `:collections` - list of collection names to search
    - `:query` - search query string
    - `:max_injections` - max number of memories to inject (default: 5)
  """
  @spec inject([context_entry()], keyword()) :: [context_entry()]
  def inject(messages, opts \\ []) do
    collections = Keyword.get(opts, :collections, [])
    query = Keyword.get(opts, :query, "")
    max_injections = Keyword.get(opts, :max_injections, 5)

    if query == "" or collections == [] do
      messages
    else
      memories = search_collections(collections, query, max_injections)

      injected =
        Enum.map(memories, fn entry ->
          %{
            role: :system,
            content: "[memory:#{entry.key}] #{format_value(entry.value)}"
          }
        end)

      injected ++ messages
    end
  end

  @doc """
  Compact context when approaching token limit.

  Delegates to `OptimalEngine.Memory.Compactor.compact/2`.
  """
  @spec compact([context_entry()], keyword()) :: [context_entry()]
  def compact(messages, opts \\ []) do
    Compactor.compact(messages, opts)
  end

  @doc """
  Rough token count estimation.

  Uses the heuristic of ~4 characters per token.
  """
  @spec estimate_tokens([context_entry()] | String.t()) :: non_neg_integer()
  def estimate_tokens(text) when is_binary(text) do
    div(String.length(text), 4)
  end

  def estimate_tokens(messages) when is_list(messages) do
    messages
    |> Enum.map(fn
      %{content: content} -> String.length(content)
      text when is_binary(text) -> String.length(text)
      _ -> 0
    end)
    |> Enum.sum()
    |> div(4)
  end

  # --- Private ---

  defp do_build(:recent, session_id, opts) do
    count = Keyword.get(opts, :recent_count, 50)
    msgs = Session.messages(session_id, count)
    Enum.map(msgs, &to_context_entry/1)
  end

  defp do_build(:relevant, session_id, opts) do
    base = do_build(:recent, session_id, opts)
    inject(base, opts)
  end

  defp do_build(:summary, session_id, opts) do
    all = Session.messages(session_id)
    recent_count = Keyword.get(opts, :recent_count, 20)

    if length(all) <= recent_count do
      Enum.map(all, &to_context_entry/1)
    else
      old = Enum.take(all, length(all) - recent_count)
      recent = Enum.take(all, -recent_count)

      summary_text = Compactor.summarize_messages(Enum.map(old, &to_context_entry/1))

      [%{role: :system, content: "[context summary] #{summary_text}"}] ++
        Enum.map(recent, &to_context_entry/1)
    end
  end

  defp to_context_entry(%{role: role, content: content}) do
    %{role: role, content: content}
  end

  defp search_collections(collections, query, max) do
    collections
    |> Enum.flat_map(fn col ->
      case OptimalEngine.Memory.search(col, query) do
        {:ok, entries} -> entries
        _ -> []
      end
    end)
    |> Enum.take(max)
  end

  defp trim_to_budget(messages, max_tokens) do
    {kept, _} =
      Enum.reduce(messages, {[], 0}, fn msg, {acc, tokens} ->
        msg_tokens = estimate_tokens(msg.content)

        if tokens + msg_tokens <= max_tokens do
          {acc ++ [msg], tokens + msg_tokens}
        else
          {acc, tokens}
        end
      end)

    kept
  end

  defp format_value(value) when is_binary(value), do: value
  defp format_value(value) when is_map(value), do: Jason.encode!(value)
  defp format_value(value), do: inspect(value)
end
