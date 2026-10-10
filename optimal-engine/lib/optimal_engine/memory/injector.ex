defmodule OptimalEngine.Memory.Injector do
  @moduledoc """
  Conditional memory injection — determines which memories to surface
  based on current context (files, task, session, scope).

  ## Injection Rules

  | Category           | Scope       | Rule                                       |
  |--------------------|-------------|---------------------------------------------|
  | `:project_info`    | `:workspace`| ALWAYS inject                               |
  | `:user_preference` | `:global`   | ALWAYS inject                               |
  | `:lesson`          | any         | Inject when file pattern matches            |
  | `:pattern`         | any         | Inject when task type matches               |
  | `:solution`        | any         | Inject when similar error/task detected      |
  | `:context`         | `:session`  | Inject only for current session             |
  | `:project_spec`    | `:workspace`| Inject when working on related feature      |

  Entries are returned sorted by relevance score (descending), with an
  optional token budget cap to prevent context overflow.
  """

  alias OptimalEngine.Memory.Taxonomy

  @type injection_context :: %{
          optional(:files) => [String.t()],
          optional(:task) => String.t(),
          optional(:task_type) => String.t(),
          optional(:error) => String.t(),
          optional(:session_id) => String.t(),
          optional(:max_entries) => pos_integer(),
          optional(:max_tokens) => pos_integer()
        }

  @extension_keywords %{
    ".ex" => ~w(elixir otp genserver phoenix ecto),
    ".exs" => ~w(elixir test exunit mix config),
    ".go" => ~w(golang goroutine channel interface struct),
    ".rs" => ~w(rust cargo borrow lifetime ownership),
    ".ts" => ~w(typescript node react svelte frontend),
    ".tsx" => ~w(typescript react jsx component frontend),
    ".svelte" => ~w(svelte component store frontend),
    ".py" => ~w(python pip django flask),
    ".sql" => ~w(sql database query migration schema),
    ".yaml" => ~w(kubernetes docker devops deployment config),
    ".yml" => ~w(kubernetes docker devops deployment config),
    ".json" => ~w(config json schema),
    ".md" => ~w(documentation markdown)
  }

  @chars_per_token 4

  @doc """
  Given a pool of taxonomy entries and a context map, returns the subset
  of memories that should be injected into the current system prompt.

  The context map may contain:
    - `:files` — list of file paths being worked on
    - `:task` — current task description
    - `:task_type` — type of task (e.g., "debug", "feature", "refactor")
    - `:error` — current error message, if any
    - `:session_id` — current session identifier
    - `:max_entries` — cap on number of returned entries (default: 20)
    - `:max_tokens` — rough token budget for injected memories

  Returns a list of `Taxonomy.t()` entries sorted by relevance, each with
  an updated `relevance_score` reflecting contextual fit.
  """
  @spec inject_relevant([Taxonomy.t()], injection_context()) :: [Taxonomy.t()]
  def inject_relevant(entries, context) when is_list(entries) and is_map(context) do
    max_entries = Map.get(context, :max_entries, 20)
    max_tokens = Map.get(context, :max_tokens)

    entries
    |> Enum.map(fn entry -> {score_entry(entry, context), entry} end)
    |> Enum.filter(fn {score, _entry} -> score > 0.0 end)
    |> Enum.sort_by(fn {score, _entry} -> score end, :desc)
    |> Enum.take(max_entries)
    |> maybe_trim_to_budget(max_tokens)
    |> Enum.map(fn {score, entry} ->
      %{entry | relevance_score: Float.round(score, 4)}
    end)
  end

  @doc """
  Format injected memories as a context block suitable for system prompt injection.
  """
  @spec format_for_prompt([Taxonomy.t()]) :: String.t()
  def format_for_prompt([]), do: ""

  def format_for_prompt(entries) do
    entries
    |> Enum.map(fn entry ->
      "[memory [#{entry.category}] [#{entry.scope}]] #{entry.content}"
    end)
    |> Enum.join("\n")
  end

  defp score_entry(entry, context) do
    base = base_score(entry)
    contextual = contextual_score(entry, context)
    recency = recency_score(entry)
    base * 0.3 + contextual * 0.5 + recency * 0.2
  end

  defp base_score(%Taxonomy{category: :project_info, scope: :workspace}), do: 1.0
  defp base_score(%Taxonomy{category: :user_preference, scope: :global}), do: 1.0
  defp base_score(%Taxonomy{category: :project_info}), do: 0.7
  defp base_score(%Taxonomy{category: :user_preference}), do: 0.7
  defp base_score(%Taxonomy{category: :project_spec, scope: :workspace}), do: 0.6
  defp base_score(%Taxonomy{category: :lesson}), do: 0.3
  defp base_score(%Taxonomy{category: :pattern}), do: 0.3
  defp base_score(%Taxonomy{category: :solution}), do: 0.2
  defp base_score(%Taxonomy{category: :context, scope: :session}), do: 0.4
  defp base_score(%Taxonomy{category: :context}), do: 0.1
  defp base_score(_), do: 0.1

  defp contextual_score(entry, context) do
    scores = [
      file_match_score(entry, context),
      task_match_score(entry, context),
      error_match_score(entry, context),
      session_match_score(entry, context)
    ]

    Enum.max(scores)
  end

  defp file_match_score(%Taxonomy{category: cat} = entry, %{files: files})
       when cat in [:lesson, :pattern, :solution, :project_spec] and is_list(files) and files != [] do
    content_down = String.downcase(entry.content)

    file_keywords =
      files
      |> Enum.flat_map(fn path ->
        ext = Path.extname(path)
        Map.get(@extension_keywords, ext, [])
      end)
      |> Enum.uniq()

    hits = Enum.count(file_keywords, &String.contains?(content_down, &1))

    filename_hits =
      Enum.count(files, fn path ->
        basename = Path.basename(path) |> String.downcase()
        String.contains?(content_down, basename)
      end)

    cond do
      filename_hits > 0 -> 1.0
      hits >= 3 -> 0.9
      hits >= 1 -> 0.6
      true -> 0.0
    end
  end

  defp file_match_score(_entry, _context), do: 0.0

  defp task_match_score(%Taxonomy{category: :pattern} = entry, %{task_type: task_type})
       when is_binary(task_type) and task_type != "" do
    content_down = String.downcase(entry.content)
    if String.contains?(content_down, String.downcase(task_type)), do: 0.8, else: 0.0
  end

  defp task_match_score(%Taxonomy{} = entry, %{task: task})
       when is_binary(task) and task != "" do
    content_down = String.downcase(entry.content)
    task_words = task |> String.downcase() |> String.split(~r/\s+/, trim: true)
    significant = Enum.filter(task_words, &(String.length(&1) > 3))
    hits = Enum.count(significant, &String.contains?(content_down, &1))
    total = max(length(significant), 1)
    ratio = hits / total

    cond do
      ratio >= 0.5 -> 0.8
      ratio >= 0.2 -> 0.5
      hits >= 1 -> 0.3
      true -> 0.0
    end
  end

  defp task_match_score(_entry, _context), do: 0.0

  defp error_match_score(%Taxonomy{category: :solution} = entry, %{error: error})
       when is_binary(error) and error != "" do
    content_down = String.downcase(entry.content)

    error_words =
      error
      |> String.downcase()
      |> String.split(~r/[\s:,.()\[\]{}]+/, trim: true)
      |> Enum.filter(&(String.length(&1) > 3))
      |> Enum.take(10)

    hits = Enum.count(error_words, &String.contains?(content_down, &1))

    cond do
      hits >= 5 -> 1.0
      hits >= 3 -> 0.8
      hits >= 1 -> 0.4
      true -> 0.0
    end
  end

  defp error_match_score(%Taxonomy{category: :lesson} = entry, %{error: error})
       when is_binary(error) and error != "" do
    content_down = String.downcase(entry.content)

    error_words =
      error
      |> String.downcase()
      |> String.split(~r/[\s:,.()\[\]{}]+/, trim: true)
      |> Enum.filter(&(String.length(&1) > 3))
      |> Enum.take(10)

    hits = Enum.count(error_words, &String.contains?(content_down, &1))

    cond do
      hits >= 3 -> 0.6
      hits >= 1 -> 0.3
      true -> 0.0
    end
  end

  defp error_match_score(_entry, _context), do: 0.0

  defp session_match_score(%Taxonomy{category: :context, scope: :session}, %{session_id: sid})
       when is_binary(sid) and sid != "",
       do: 1.0

  defp session_match_score(%Taxonomy{category: :context, scope: :session}, _context), do: 0.0

  defp session_match_score(_entry, _context), do: 0.0

  defp recency_score(%Taxonomy{accessed_at: nil, created_at: nil}), do: 0.3

  defp recency_score(%Taxonomy{accessed_at: accessed_at}) when not is_nil(accessed_at) do
    age_hours = DateTime.diff(DateTime.utc_now(), accessed_at, :second) / 3600.0
    :math.exp(-0.693 * age_hours / 48.0)
  end

  defp recency_score(%Taxonomy{created_at: created_at}) when not is_nil(created_at) do
    age_hours = DateTime.diff(DateTime.utc_now(), created_at, :second) / 3600.0
    :math.exp(-0.693 * age_hours / 48.0)
  end

  defp recency_score(_), do: 0.3

  defp maybe_trim_to_budget(scored_entries, nil), do: scored_entries

  defp maybe_trim_to_budget(scored_entries, max_tokens) do
    max_chars = max_tokens * @chars_per_token

    {kept, _} =
      Enum.reduce_while(scored_entries, {[], max_chars}, fn {score, entry}, {acc, budget} ->
        size = byte_size(entry.content)

        if size <= budget do
          {:cont, {acc ++ [{score, entry}], budget - size}}
        else
          {:halt, {acc, budget}}
        end
      end)

    kept
  end
end
