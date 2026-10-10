defmodule OptimalEngine.Spec.Coverage do
  @moduledoc """
  Analyzes which source files are covered by spec contracts and which aren't.

  Coverage is determined by the `surface` field in `spec-meta` — it lists the
  source files that a spec claims to govern.
  """

  alias OptimalEngine.Spec.Parser

  @type report :: %{
          covered: [String.t()],
          uncovered: [String.t()],
          total_source: non_neg_integer(),
          total_covered: non_neg_integer(),
          percentage: float(),
          specs: non_neg_integer()
        }

  @doc """
  Analyzes spec coverage against source files in the engine.

  Walks all `.spec.md` files in `spec_dir`, collects surface paths,
  and compares against actual source files under `source_dir`.
  """
  @spec analyze(String.t(), String.t()) :: {:ok, report()}
  def analyze(spec_dir, source_dir) do
    root = Application.get_env(:optimal_engine, :root_path, "..")
    {:ok, specs} = Parser.parse_all(spec_dir)

    covered_paths =
      specs
      |> Enum.flat_map(fn spec -> spec.meta.surface end)
      |> Enum.map(&resolve_surface_path(&1, root))
      |> MapSet.new()

    source_files =
      source_dir
      |> Path.join("**/*.ex")
      |> Path.wildcard()
      |> Enum.map(&normalize_path/1)
      |> Enum.sort()

    covered = Enum.filter(source_files, &MapSet.member?(covered_paths, &1))
    uncovered = Enum.reject(source_files, &MapSet.member?(covered_paths, &1))

    total = length(source_files)

    percentage =
      if total > 0,
        do: Float.round(length(covered) / total * 100.0, 1),
        else: 0.0

    {:ok,
     %{
       covered: covered,
       uncovered: uncovered,
       total_source: total,
       total_covered: length(covered),
       percentage: percentage,
       specs: length(specs)
     }}
  end

  @doc """
  Returns a per-spec surface map showing which specs cover which files.
  """
  @spec surface_map(String.t()) :: {:ok, %{String.t() => [String.t()]}}
  def surface_map(spec_dir) do
    {:ok, specs} = Parser.parse_all(spec_dir)

    map =
      Enum.reduce(specs, %{}, fn spec, acc ->
        Map.put(acc, spec.meta.id || spec.title, spec.meta.surface)
      end)

    {:ok, map}
  end

  defp normalize_path(path) do
    path
    |> Path.expand()
    |> String.replace(~r|/+|, "/")
  end

  defp resolve_surface_path(path, root) do
    Path.join(root, path)
    |> Path.expand()
    |> String.replace(~r|/+|, "/")
  end
end
