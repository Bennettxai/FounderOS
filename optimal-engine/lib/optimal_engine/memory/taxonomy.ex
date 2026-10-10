defmodule OptimalEngine.Memory.Taxonomy do
  @moduledoc """
  Memory taxonomy — categorization and scoping for structured memory entries.

  Every memory entry has a category (what kind of knowledge) and a scope
  (how widely it applies). This module provides:

  - Struct definition for typed memory entries
  - Auto-classification of raw content into categories
  - Filtering by category, scope, or arbitrary predicates

  ## Categories

  | Category        | Description                                  |
  |-----------------|----------------------------------------------|
  | `:user_preference` | User-stated preferences, rules, conventions |
  | `:project_info`    | Project metadata, stack, architecture       |
  | `:project_spec`    | Specifications, requirements, acceptance criteria |
  | `:lesson`          | Mistakes made, corrections, debugging lessons |
  | `:pattern`         | Recurring patterns observed across interactions |
  | `:solution`        | Proven solutions to known problems           |
  | `:context`         | Ephemeral session context, working notes     |

  ## Scopes

  | Scope        | Lifetime                                     |
  |--------------|----------------------------------------------|
  | `:global`    | Persists across all workspaces and sessions   |
  | `:workspace` | Persists within the current project/workspace |
  | `:session`   | Lives only for the current session            |
  """

  @type category ::
          :user_preference
          | :project_info
          | :project_spec
          | :lesson
          | :pattern
          | :solution
          | :context

  @type scope :: :global | :workspace | :session

  @type t :: %__MODULE__{
          id: String.t(),
          category: category(),
          scope: scope(),
          content: String.t(),
          metadata: map(),
          created_at: DateTime.t(),
          accessed_at: DateTime.t(),
          access_count: non_neg_integer(),
          relevance_score: float()
        }

  @enforce_keys [:id, :category, :scope, :content]
  defstruct id: nil,
            category: :context,
            scope: :session,
            content: "",
            metadata: %{},
            created_at: nil,
            accessed_at: nil,
            access_count: 0,
            relevance_score: 0.0

  @categories ~w(user_preference project_info project_spec lesson pattern solution context)a
  @scopes ~w(global workspace session)a

  @category_signals %{
    user_preference: ~w(prefer always never convention style rule habit want like dislike),
    project_info: ~w(stack architecture repo project codebase framework dependency),
    project_spec: ~w(requirement spec acceptance criteria must shall feature story),
    lesson: ~w(lesson learned mistake bug fix correction wrong error regression),
    pattern: ~w(pattern recurring repeatedly noticed common typical),
    solution: ~w(solution solved resolved workaround fix approach),
    context: ~w(currently working session today now)
  }

  @doc """
  Create a new taxonomy entry with auto-generated ID and timestamps.

  ## Options
    - `:category` — override auto-classification
    - `:scope` — defaults to `:workspace`
    - `:metadata` — arbitrary metadata map
  """
  @spec new(String.t(), keyword()) :: t()
  def new(content, opts \\ []) do
    now = DateTime.utc_now()
    category = Keyword.get(opts, :category) || categorize(content)
    scope = Keyword.get(opts, :scope, :workspace)
    metadata = Keyword.get(opts, :metadata, %{})

    %__MODULE__{
      id: generate_id(content),
      category: category,
      scope: scope,
      content: content,
      metadata: metadata,
      created_at: now,
      accessed_at: now,
      access_count: 0,
      relevance_score: 0.0
    }
  end

  @doc """
  Auto-classify content into a category based on keyword signals.

  Falls back to `:context` when no strong signal is detected.
  """
  @spec categorize(String.t()) :: category()
  def categorize(content) when is_binary(content) do
    downcased = String.downcase(content)

    scores =
      Enum.map(@category_signals, fn {category, signals} ->
        hits = Enum.count(signals, &String.contains?(downcased, &1))
        {category, hits}
      end)

    {best_category, best_score} = Enum.max_by(scores, fn {_cat, score} -> score end)

    if best_score > 0, do: best_category, else: :context
  end

  def categorize(_), do: :context

  @doc """
  Filter a list of taxonomy entries by category, scope, or custom predicate.

  ## Filter options
    - `:category` — atom or list of atoms
    - `:scope` — atom or list of atoms
    - `:min_relevance` — minimum relevance score (float)
    - `:min_access_count` — minimum access count
    - `:since` — DateTime, only entries created after this time
    - `:predicate` — arbitrary `(t() -> boolean())` function
  """
  @spec filter_by([t()], keyword()) :: [t()]
  def filter_by(entries, filters) when is_list(entries) do
    Enum.filter(entries, fn entry -> matches_all_filters?(entry, filters) end)
  end

  @doc "Returns the list of valid categories."
  @spec categories() :: [category()]
  def categories, do: @categories

  @doc "Returns the list of valid scopes."
  @spec scopes() :: [scope()]
  def scopes, do: @scopes

  @doc "Record an access — increments access_count and updates accessed_at."
  @spec touch(t()) :: t()
  def touch(%__MODULE__{} = entry) do
    %{entry | access_count: entry.access_count + 1, accessed_at: DateTime.utc_now()}
  end

  @doc "Returns true if the given atom is a valid category."
  @spec valid_category?(atom()) :: boolean()
  def valid_category?(cat), do: cat in @categories

  @doc "Returns true if the given atom is a valid scope."
  @spec valid_scope?(atom()) :: boolean()
  def valid_scope?(scope), do: scope in @scopes

  defp matches_all_filters?(entry, filters) do
    Enum.all?(filters, fn filter -> matches_filter?(entry, filter) end)
  end

  defp matches_filter?(entry, {:category, cats}) when is_list(cats), do: entry.category in cats
  defp matches_filter?(entry, {:category, cat}) when is_atom(cat), do: entry.category == cat
  defp matches_filter?(entry, {:scope, scopes}) when is_list(scopes), do: entry.scope in scopes
  defp matches_filter?(entry, {:scope, scope}) when is_atom(scope), do: entry.scope == scope
  defp matches_filter?(entry, {:min_relevance, min}), do: entry.relevance_score >= min
  defp matches_filter?(entry, {:min_access_count, min}), do: entry.access_count >= min

  defp matches_filter?(entry, {:since, %DateTime{} = dt}) do
    DateTime.compare(entry.created_at, dt) in [:gt, :eq]
  end

  defp matches_filter?(entry, {:predicate, fun}) when is_function(fun, 1), do: fun.(entry)
  defp matches_filter?(_entry, _unknown_filter), do: true

  defp generate_id(content) do
    data = "#{System.monotonic_time(:microsecond)}:#{content}"
    :crypto.hash(:sha256, data) |> Base.encode16(case: :lower) |> String.slice(0, 16)
  end
end
