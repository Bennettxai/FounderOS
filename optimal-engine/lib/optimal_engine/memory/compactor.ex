defmodule OptimalEngine.Memory.Compactor do
  @moduledoc """
  Context compaction for OptimalEngine.Memory.

  Reduces context size while preserving meaning through summarization
  and pruning strategies.

  ## Configuration

      config :optimal_engine, OptimalEngine.Memory.Compactor,
        warn_at: 0.85,
        compact_at: 0.90,
        hard_stop: 0.95
  """

  @type message :: %{role: atom(), content: String.t()}

  @default_warn_at 0.85
  @default_compact_at 0.90
  @default_hard_stop 0.95

  @doc """
  Compact a list of messages to reduce context size.

  Keeps system messages and recent messages, summarizes the middle.

  ## Options
    - `:max_tokens` - token budget (default: 100_000)
    - `:keep_recent` - number of recent messages to keep verbatim (default: 20)
    - `:keep_system` - whether to preserve system messages (default: true)
  """
  @spec compact([message()], keyword()) :: [message()]
  def compact(messages, opts \\ []) do
    max_tokens = Keyword.get(opts, :max_tokens, 100_000)
    keep_recent = Keyword.get(opts, :keep_recent, 20)
    keep_system = Keyword.get(opts, :keep_system, true)

    current_tokens = estimate_tokens(messages)

    if current_tokens <= max_tokens do
      messages
    else
      do_compact(messages, keep_recent, keep_system)
    end
  end

  @doc """
  Summarize a list of messages into a single summary string.

  Extracts key content from each message and concatenates into
  a condensed form.
  """
  @spec summarize_messages([message()]) :: String.t()
  def summarize_messages([]), do: ""

  def summarize_messages(messages) do
    messages
    |> Enum.map(fn msg ->
      role = msg.role
      content = truncate(msg.content, 100)
      "[#{role}] #{content}"
    end)
    |> Enum.join(" | ")
  end

  @doc """
  Remove messages older than the threshold, keeping system messages
  and the most recent messages.

  ## Options
    - `:keep_recent` - number of recent messages to keep (default: 20)
    - `:keep_system` - keep all system messages (default: true)
  """
  @spec prune_old([message()], keyword()) :: [message()]
  def prune_old(messages, opts \\ []) do
    keep_recent = Keyword.get(opts, :keep_recent, 20)
    keep_system = Keyword.get(opts, :keep_system, true)

    total = length(messages)

    if total <= keep_recent do
      messages
    else
      recent = Enum.take(messages, -keep_recent)

      if keep_system do
        old = Enum.take(messages, total - keep_recent)
        system_msgs = Enum.filter(old, &(&1.role == :system))
        system_msgs ++ recent
      else
        recent
      end
    end
  end

  @doc """
  Estimate token savings from compaction.

  Returns `{before_tokens, after_tokens, savings_pct}`.
  """
  @spec estimate_savings([message()], keyword()) ::
          {non_neg_integer(), non_neg_integer(), float()}
  def estimate_savings(messages, opts \\ []) do
    before = estimate_tokens(messages)
    compacted = compact(messages, opts)
    after_tokens = estimate_tokens(compacted)
    savings = if before > 0, do: (before - after_tokens) / before, else: 0.0
    {before, after_tokens, Float.round(savings, 4)}
  end

  @doc """
  Check if compaction is needed based on current token count and budget.

  Returns `:ok`, `:warn`, `:compact`, or `:hard_stop`.
  """
  @spec check_threshold(non_neg_integer(), non_neg_integer()) ::
          :ok | :warn | :compact | :hard_stop
  def check_threshold(current_tokens, max_tokens) do
    ratio = current_tokens / max(max_tokens, 1)

    cond do
      ratio >= hard_stop() -> :hard_stop
      ratio >= compact_at() -> :compact
      ratio >= warn_at() -> :warn
      true -> :ok
    end
  end

  # --- Private ---

  defp do_compact(messages, keep_recent, keep_system) do
    total = length(messages)
    recent = Enum.take(messages, -keep_recent)
    old = Enum.take(messages, max(0, total - keep_recent))

    system_msgs =
      if keep_system do
        Enum.filter(old, &(&1.role == :system))
      else
        []
      end

    non_system_old = Enum.reject(old, &(&1.role == :system))
    summary_text = summarize_messages(non_system_old)

    summary_msg = %{
      role: :system,
      content: "[compacted: #{length(non_system_old)} messages] #{summary_text}"
    }

    system_msgs ++ [summary_msg] ++ recent
  end

  defp estimate_tokens(messages) do
    messages
    |> Enum.map(fn %{content: c} -> String.length(c) end)
    |> Enum.sum()
    |> div(4)
  end

  defp truncate(text, max_len) when byte_size(text) <= max_len, do: text
  defp truncate(text, max_len), do: String.slice(text, 0, max_len) <> "..."

  defp config, do: Application.get_env(:optimal_engine, __MODULE__, [])
  defp warn_at, do: config()[:warn_at] || @default_warn_at
  defp compact_at, do: config()[:compact_at] || @default_compact_at
  defp hard_stop, do: config()[:hard_stop] || @default_hard_stop
end
