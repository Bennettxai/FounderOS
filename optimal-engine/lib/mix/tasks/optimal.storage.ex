defmodule Mix.Tasks.Optimal.Storage do
  use Mix.Task

  @shortdoc "Inspect providers and plan storage for workspace use cases"

  @moduledoc """
  Inspect physical storage providers or generate a use-case plan.

      mix optimal.storage list
      mix optimal.storage list --probe
      mix optimal.storage use-cases
      mix optimal.storage plan desktop_local,analytics

  Planning is read-only. It never enables a provider or moves canonical data.
  """

  alias OptimalEngine.Storage.{Planner, Providers}

  @impl true
  def run(args) do
    {opts, positional, _invalid} = OptionParser.parse(args, strict: [probe: :boolean])
    Mix.Task.run("app.start")

    case positional do
      ["list"] -> print(Providers.list(probe: Keyword.get(opts, :probe, false)))
      ["use-cases"] -> print(Planner.use_cases())
      ["plan", use_cases] -> plan(String.split(use_cases, ",", trim: true), opts)
      _ -> Mix.raise("usage: mix optimal.storage list|use-cases|plan <case,case> [--probe]")
    end
  end

  defp plan(use_cases, opts) do
    case Planner.plan(use_cases, probe: Keyword.get(opts, :probe, false)) do
      {:ok, plan} -> print(plan)
      {:error, reason} -> Mix.raise("storage plan failed: #{inspect(reason)}")
    end
  end

  defp print(value), do: IO.puts(Jason.encode!(value, pretty: true))
end
