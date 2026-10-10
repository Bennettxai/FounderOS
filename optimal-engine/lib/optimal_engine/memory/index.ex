defmodule OptimalEngine.Memory.Index do
  @moduledoc """
  ETS-backed inverted keyword index for memory entries.

  Maintains two ETS tables:
  - `@index_table` — keyword → [entry_id, ...] (inverted index)
  - `@entry_table` — entry_id → entry_map (full entry store)

  Supports full rebuild from parsed entries and incremental single-entry indexing.
  """

  require Logger

  @index_table :optimal_engine_memory_index
  @entry_table :optimal_engine_memory_entries

  @stop_words MapSet.new(~w(
    the and for are but not you all any can had her was one our out day been have
    from this that with what when will more about which them than been would make
    like time just know take people into year your good some could over such after
    come made find back only first great even give most those down should well
    being work through where much other also life between know years hand high
    because large turn each long next look state want head around move both
    think still might school world kind keep never really need does going right
    used every last very just said same tell call before mean also actually thing
    many then those however these while most only must since well still under
    again too own part here there where help using really trying getting doing
    went got let its use way may new now old see try run put set did get how
    has him his she her its who why yes yet able
  ))

  # ── Public API ───────────────────────────────────────────────────────

  @doc "Ensure ETS tables exist, creating them if needed."
  def ensure_tables do
    if :ets.info(@index_table) == :undefined do
      :ets.new(@index_table, [:named_table, :set, :public, read_concurrency: true])
    end

    if :ets.info(@entry_table) == :undefined do
      :ets.new(@entry_table, [:named_table, :set, :public, read_concurrency: true])
    end

    :ok
  end

  @doc "Full rebuild of the index from a list of `{entry_id, entry}` tuples."
  def rebuild(entries) do
    ensure_tables()
    :ets.delete_all_objects(@index_table)
    :ets.delete_all_objects(@entry_table)

    Enum.each(entries, fn {entry_id, entry} ->
      index_single(entry_id, entry)
    end)

    entry_count = length(entries)
    kw_count = :ets.info(@index_table, :size) || 0

    Logger.debug(
      "[OptimalEngine.Memory.Index] rebuilt: #{entry_count} entries, #{kw_count} keywords"
    )

    :ok
  rescue
    e -> Logger.error("[OptimalEngine.Memory.Index] rebuild failed: #{inspect(e)}")
  end

  @doc "Incrementally index a single entry without full rebuild."
  def add_entry(entry_id, entry) do
    ensure_tables()
    index_single(entry_id, entry)
  rescue
    e -> Logger.warning("[OptimalEngine.Memory.Index] add_entry failed: #{inspect(e)}")
  end

  @doc "Look up entry IDs that match any keyword in the query. Returns a frequency map."
  @spec query_keywords([String.t()]) :: %{String.t() => non_neg_integer()}
  def query_keywords(keywords) do
    keywords
    |> Enum.flat_map(fn keyword ->
      try do
        case :ets.lookup(@index_table, keyword) do
          [{^keyword, ids}] -> ids
          [] -> []
        end
      rescue
        ArgumentError -> []
      end
    end)
    |> Enum.frequencies()
  end

  @doc "Fetch a single entry by ID from the entry table."
  @spec get_entry(String.t()) :: map() | nil
  def get_entry(entry_id) do
    try do
      case :ets.lookup(@entry_table, entry_id) do
        [{^entry_id, entry}] -> entry
        [] -> nil
      end
    rescue
      ArgumentError -> nil
    end
  end

  @doc "Return all entries from the entry table as a list of maps."
  @spec all_entries() :: [map()]
  def all_entries do
    try do
      :ets.tab2list(@entry_table) |> Enum.map(fn {_id, entry} -> entry end)
    rescue
      _ -> []
    end
  end

  @doc "Return the number of indexed keywords."
  @spec keyword_count() :: non_neg_integer()
  def keyword_count do
    try do
      :ets.info(@index_table, :size) || 0
    rescue
      _ -> 0
    end
  end

  @doc "Return the number of indexed entries."
  @spec entry_count() :: non_neg_integer()
  def entry_count do
    try do
      :ets.info(@entry_table, :size) || 0
    rescue
      _ -> 0
    end
  end

  @doc "Extract meaningful keywords from text, excluding stop words."
  @spec extract_keywords(String.t()) :: [String.t()]
  def extract_keywords(message) do
    message
    |> String.replace(~r/([a-z])([A-Z])/, "\\1 \\2")
    |> String.replace(~r/([A-Z]{2,})([A-Z][a-z])/, "\\1 \\2")
    |> String.downcase()
    |> String.replace(~r/[`"'{}()\[\]]/, " ")
    |> String.replace(~r/[_\-]/, " ")
    |> String.split(~r/[\s,.:;!?\/\\|@#$%^&*+=<>~]+/, trim: true)
    |> Enum.reject(fn word -> MapSet.member?(@stop_words, word) end)
    |> Enum.filter(fn word -> String.length(word) > 2 end)
    |> Enum.reject(fn word -> Regex.match?(~r/^\d+$/, word) end)
    |> Enum.uniq()
  end

  # ── Private ──────────────────────────────────────────────────────────

  defp index_single(entry_id, entry) do
    :ets.insert(@entry_table, {entry_id, entry})

    keywords = extract_keywords(entry[:content] || "")
    category_kw = if entry[:category], do: [String.downcase(entry[:category])], else: []

    Enum.each(Enum.uniq(keywords ++ category_kw), fn keyword ->
      existing =
        case :ets.lookup(@index_table, keyword) do
          [{^keyword, ids}] -> ids
          [] -> []
        end

      :ets.insert(@index_table, {keyword, [entry_id | existing]})
    end)
  end
end
