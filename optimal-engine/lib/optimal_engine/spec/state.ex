defmodule OptimalEngine.Spec.State do
  @moduledoc """
  Reads and writes `.system/spec_state.json` — the canonical spec verification state.

  Deterministic serialization: maps are sorted by key before encoding so that
  git diffs remain clean and predictable. Write is skipped if content hasn't changed.
  """

  @state_file "spec_state.json"

  @doc """
  Returns the default state file path.
  """
  @spec default_path() :: String.t()
  def default_path do
    root = Application.get_env(:optimal_engine, :root_path, ".")
    Path.join([root, ".system", @state_file])
  end

  @doc """
  Reads the current spec state from disk.
  """
  @spec read(String.t()) :: {:ok, map()} | {:error, :not_found | term()}
  def read(path \\ default_path()) do
    case File.read(path) do
      {:ok, json} ->
        case Jason.decode(json) do
          {:ok, state} -> {:ok, state}
          {:error, reason} -> {:error, {:json_decode, reason}}
        end

      {:error, :enoent} ->
        {:error, :not_found}

      {:error, reason} ->
        {:error, {:file_read, reason}}
    end
  end

  @doc """
  Writes spec state to disk with deterministic serialization.

  Skips the write if the file already contains identical content.
  """
  @spec write(map(), String.t()) :: :ok | {:error, term()}
  def write(state, path \\ default_path()) do
    sorted = sort_deep(state)
    stamped = Map.put(sorted, "generated_at", DateTime.utc_now() |> DateTime.to_iso8601())

    json = Jason.encode!(stamped, pretty: true)

    case File.read(path) do
      {:ok, ^json} ->
        :ok

      _ ->
        File.mkdir_p!(Path.dirname(path))
        File.write(path, json)
    end
  end

  @doc """
  Builds a state map from parsed specs and verification findings.
  """
  @spec build(list(), list()) :: map()
  def build(specs, findings) do
    subjects =
      Enum.map(specs, fn spec ->
        %{
          "id" => spec.meta.id,
          "kind" => spec.meta.kind,
          "file" => spec.file,
          "status" => spec.meta.status,
          "requirements" => length(spec.requirements),
          "scenarios" => length(spec.scenarios),
          "verifications" => length(spec.verifications)
        }
      end)

    {claimed, linked, executed} = count_strengths(findings)

    %{
      "summary" => %{
        "subjects" => length(specs),
        "requirements" => Enum.sum(Enum.map(specs, &length(&1.requirements))),
        "scenarios" => Enum.sum(Enum.map(specs, &length(&1.scenarios))),
        "verifications" => Enum.sum(Enum.map(specs, &length(&1.verifications))),
        "findings" => length(findings),
        "passing" => Enum.count(findings, & &1.meets_minimum),
        "failing" => Enum.count(findings, &(not &1.meets_minimum))
      },
      "subjects" => subjects,
      "verification" => %{
        "claims" => Enum.map(findings, &finding_to_map/1),
        "coverage" => %{
          "claimed" => claimed,
          "linked" => linked,
          "executed" => executed
        }
      }
    }
  end

  # -- Private -----------------------------------------------------------------

  defp count_strengths(findings) do
    Enum.reduce(findings, {0, 0, 0}, fn finding, {c, l, e} ->
      case finding.strength do
        :executed -> {c, l, e + 1}
        :linked -> {c, l + 1, e}
        _ -> {c + 1, l, e}
      end
    end)
  end

  defp finding_to_map(finding) do
    %{
      "subject" => finding.subject,
      "requirement" => finding.requirement,
      "strength" => to_string(finding.strength),
      "min_strength" => to_string(finding.min_strength),
      "meets_minimum" => finding.meets_minimum,
      "target" => finding.target
    }
  end

  defp sort_deep(map) when is_map(map) do
    map
    |> Enum.sort_by(fn {k, _} -> k end)
    |> Enum.map(fn {k, v} -> {k, sort_deep(v)} end)
    |> Map.new()
  end

  defp sort_deep(list) when is_list(list), do: Enum.map(list, &sort_deep/1)
  defp sort_deep(other), do: other
end
