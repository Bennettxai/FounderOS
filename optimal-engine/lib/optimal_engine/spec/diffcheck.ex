defmodule OptimalEngine.Spec.Diffcheck do
  @moduledoc """
  Git-based drift detection: detects when source files changed but their
  corresponding spec didn't update.

  This is the governance gate. If code evolves but the spec stays stale,
  the spec is no longer trustworthy. Diffcheck makes drift visible.
  """

  alias OptimalEngine.Spec.Parser

  @type drift :: %{
          source_file: String.t(),
          spec_file: String.t(),
          spec_id: String.t(),
          reason: String.t()
        }

  @doc """
  Checks for spec drift by comparing git changes.

  1. Gets list of changed files from `git diff --name-only` against `base`
  2. For each changed file, finds specs whose `surface` includes that file
  3. If the spec file wasn't also changed → drift

  Options:
  - `:base` — git ref to diff against (default: "HEAD~1")
  """
  @spec check(String.t(), String.t(), keyword()) :: {:ok, [drift()]}
  def check(spec_dir, root_path, opts \\ []) do
    base = Keyword.get(opts, :base, "HEAD~1")

    case git_changed_files(root_path, base) do
      {:ok, changed_files} ->
        {:ok, specs} = Parser.parse_all(spec_dir)
        drifts = find_drifts(specs, changed_files, root_path)
        {:ok, drifts}

      {:error, reason} ->
        {:error, reason}
    end
  end

  # -- Private -----------------------------------------------------------------

  defp git_changed_files(root_path, base) do
    try do
      case System.cmd("git", ["diff", "--name-only", base],
             cd: root_path,
             stderr_to_stdout: true
           ) do
        {output, 0} ->
          files =
            output
            |> String.trim()
            |> String.split("\n", trim: true)

          {:ok, MapSet.new(files)}

        {output, _code} ->
          # Might be initial commit or shallow clone — try against empty tree
          case System.cmd("git", ["diff", "--name-only", "--cached", "HEAD"],
                 cd: root_path,
                 stderr_to_stdout: true
               ) do
            {output2, 0} ->
              files = output2 |> String.trim() |> String.split("\n", trim: true)
              {:ok, MapSet.new(files)}

            _ ->
              {:error, "git diff failed: #{String.trim(output)}"}
          end
      end
    rescue
      e -> {:error, "git command failed: #{inspect(e)}"}
    end
  end

  defp find_drifts(specs, changed_files, root_path) do
    Enum.flat_map(specs, fn spec ->
      spec_relative = relative_path(spec.file, root_path)
      spec_changed? = MapSet.member?(changed_files, spec_relative)

      spec.meta.surface
      |> Enum.filter(fn surface_path ->
        # Source file changed but spec didn't
        MapSet.member?(changed_files, surface_path) and not spec_changed?
      end)
      |> Enum.map(fn source_file ->
        %{
          source_file: source_file,
          spec_file: spec_relative,
          spec_id: spec.meta.id || spec.title,
          reason: "Source file changed but spec was not updated"
        }
      end)
    end)
  end

  defp relative_path(nil, _root), do: ""

  defp relative_path(path, root_path) do
    path
    |> Path.expand()
    |> String.replace_prefix(Path.expand(root_path) <> "/", "")
  end
end
