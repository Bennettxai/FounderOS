defmodule OptimalEngine.Signal.GenreRegistry do
  @moduledoc """
  The single canonical genre vocabulary for the engine.

  Historically the engine carried four divergent genre vocabularies (a 9-atom
  CloudEvents set in the envelope builder/validation, a 9-key type→genre map in
  the classifier analyzer, a 17-entry regex set in the pipeline classifier, and
  a 5-skeleton set in the composer). The 143 genre templates under the OptimalOS
  taxonomy were inert docs, so ~134 genres collapsed to `:chat`/`:note`.

  This module loads the canonical registry generated from those templates
  (`priv/genres.exs`, shipped inside the engine so downloading apps are
  self-contained) and exposes:

    * `all/0`          — every genre entry
    * `genres/0`       — every genre name (string)
    * `fetch/1`        — lookup a genre entry by name
    * `valid?/1`       — membership check
    * `category/1`     — the genre's taxonomy category
    * `skeleton/1`     — the ordered section skeleton for composition
    * `detect/1`       — classify free text to the best-matching genre name

  Each entry is a map:

      %{
        genre: "spec",
        category: "technical",
        signal_type: "inform",
        mode: "mixed",
        purpose: "...",
        sections: [%{name: "Requirements", required: true}, ...]
      }

  The registry is read once at compile time so there is no runtime file IO and
  no dependency on the OptimalOS checkout at runtime.
  """

  @default_genre "note"

  @registry_path Path.join([:code.priv_dir(:optimal_engine), "genres.exs"])
  @external_resource @registry_path

  @genres (case File.read(@registry_path) do
             {:ok, contents} ->
               {term, _binding} = Code.eval_string(contents)
               term

             {:error, _} ->
               []
           end)

  @genre_index Map.new(@genres, fn g -> {g.genre, g} end)
  @genre_names @genres |> Enum.map(& &1.genre) |> Enum.sort()

  # High-signal keyword cues per genre, derived from the genre name and its
  # section headings. Used by `detect/1` as a graceful fallback after the
  # pipeline's high-precision regex set.
  @keyword_index Map.new(@genres, fn g ->
                   name_words =
                     g.genre
                     |> String.split("-", trim: true)
                     |> Enum.filter(&(String.length(&1) > 2))

                   section_words =
                     g.sections
                     |> Enum.flat_map(fn s ->
                       s.name |> String.downcase() |> String.split(~r/[^a-z0-9]+/, trim: true)
                     end)
                     |> Enum.filter(&(String.length(&1) > 3))

                   {g.genre, Enum.uniq(name_words ++ section_words)}
                 end)

  @doc "Returns every genre entry in the canonical registry."
  @spec all() :: [map()]
  def all, do: @genres

  @doc "Returns every genre name (sorted strings)."
  @spec genres() :: [String.t()]
  def genres, do: @genre_names

  @doc "Returns the number of genres in the registry."
  @spec count() :: non_neg_integer()
  def count, do: length(@genres)

  @doc "Returns the default genre name used when nothing matches."
  @spec default_genre() :: String.t()
  def default_genre, do: @default_genre

  @doc "Fetches a genre entry by name. Returns `{:ok, entry}` or `:error`."
  @spec fetch(String.t() | atom()) :: {:ok, map()} | :error
  def fetch(genre) when is_atom(genre), do: fetch(Atom.to_string(genre))

  def fetch(genre) when is_binary(genre) do
    case Map.fetch(@genre_index, genre) do
      {:ok, entry} -> {:ok, entry}
      :error -> :error
    end
  end

  @doc "True when the given genre name exists in the registry."
  @spec valid?(String.t() | atom()) :: boolean()
  def valid?(genre) when is_atom(genre), do: valid?(Atom.to_string(genre))
  def valid?(genre) when is_binary(genre), do: Map.has_key?(@genre_index, genre)
  def valid?(_), do: false

  @doc "Returns the taxonomy category for a genre, or nil."
  @spec category(String.t() | atom()) :: String.t() | nil
  def category(genre) do
    case fetch(genre) do
      {:ok, %{category: c}} -> c
      :error -> nil
    end
  end

  @doc """
  Returns the ordered section skeleton for a genre as a list of
  `%{name: String.t(), required: boolean()}`. Returns `[]` for unknown genres.
  """
  @spec skeleton(String.t() | atom()) :: [%{name: String.t(), required: boolean()}]
  def skeleton(genre) do
    case fetch(genre) do
      {:ok, %{sections: sections}} -> sections
      :error -> []
    end
  end

  @doc """
  Detects the best-matching genre for free text via keyword scoring across the
  full registry. Returns a genre name string, defaulting to `"note"` when no
  genre accumulates a positive score.

  This is intentionally a coarse fallback: the pipeline classifier first tries
  its high-precision regex patterns and only consults `detect/1` when those
  produce the default. Scoring rewards genre-name token hits over section-word
  hits so that, e.g., text mentioning "decision" maps to `decision-log`/`adr`
  rather than any genre that happens to contain a "Decision" section.
  """
  @spec detect(String.t()) :: String.t()
  def detect(text) when is_binary(text) do
    down = String.downcase(text)

    {best, score} =
      Enum.reduce(@keyword_index, {@default_genre, 0}, fn {genre, keywords}, {bg, bs} ->
        s = score_genre(genre, keywords, down)
        if s > bs, do: {genre, s}, else: {bg, bs}
      end)

    if score > 0, do: best, else: @default_genre
  end

  def detect(_), do: @default_genre

  # Name-token hits weigh 3x; section-word hits weigh 1x.
  defp score_genre(genre, keywords, down) do
    name_tokens = String.split(genre, "-", trim: true)

    name_score =
      Enum.count(name_tokens, fn t ->
        String.length(t) > 2 and word_present?(down, t)
      end) * 3

    section_score =
      Enum.count(keywords -- name_tokens, fn w -> word_present?(down, w) end)

    name_score + section_score
  end

  defp word_present?(down, word) do
    String.contains?(down, word)
  end
end
