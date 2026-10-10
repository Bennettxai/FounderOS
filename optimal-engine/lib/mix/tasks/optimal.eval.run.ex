defmodule Mix.Tasks.Optimal.Eval.Run do
  @shortdoc "Run a governed evaluation dataset through Context Package retrieval"

  @moduledoc """
  Run a JSON or JSONL benchmark dataset through the governed evaluation pipeline.

  The task loads benchmark cases from disk, executes each question through
  `OptimalEngine.Evaluation.run_dataset/2`, persists the run/case records, and
  prints aggregate scores.

  ## Dataset formats

      [{"case_id":"case-001","question":"...","expected_answer":"..."}]

      {
        "benchmark_name": "workspace_memory_recall",
        "dataset_name": "local-smoke",
        "cases": [
          {"id":"case-001","query":"...","answer":"..."}
        ]
      }

  JSONL/NDJSON is also supported with one case object per line.

  ## Usage

      mix optimal.eval.run path/to/dataset.json --workspace default
      mix optimal.eval.run path/to/dataset.jsonl --workspace default --top-k 100 --json
  """

  use Mix.Task

  alias OptimalEngine.Evaluation

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, invalid} =
      OptionParser.parse(args,
        strict: [
          workspace: :string,
          tenant: :string,
          actor: :string,
          benchmark: :string,
          dataset: :string,
          dataset_version: :string,
          top_k: :integer,
          limit: :integer,
          format: :string,
          json: :boolean
        ],
        aliases: [w: :workspace, t: :tenant]
      )

    case {rest, invalid} do
      {[path], []} ->
        run_dataset(path, parsed)

      {[], _} ->
        Mix.shell().error("Usage: mix optimal.eval.run path/to/dataset.json[l] --workspace default")
        System.halt(1)

      {_paths, invalid} when invalid != [] ->
        Mix.shell().error("Invalid option(s): #{inspect(invalid)}")
        System.halt(1)

      {paths, _} ->
        Mix.shell().error("Expected one dataset path, got: #{Enum.join(paths, ", ")}")
        System.halt(1)
    end
  end

  defp run_dataset(path, parsed) do
    opts =
      [
        workspace_id: Keyword.get(parsed, :workspace, "default"),
        tenant_id: Keyword.get(parsed, :tenant, "default"),
        actor_id: Keyword.get(parsed, :actor, "system:evaluation"),
        benchmark_name: Keyword.get(parsed, :benchmark),
        dataset_name: Keyword.get(parsed, :dataset),
        dataset_version: Keyword.get(parsed, :dataset_version),
        retrieval_top_k: Keyword.get(parsed, :top_k) || Keyword.get(parsed, :limit),
        limit: Keyword.get(parsed, :limit),
        format: format(Keyword.get(parsed, :format))
      ]
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    case Evaluation.run_dataset(path, opts) do
      {:ok, result} ->
        payload = %{
          run_id: result.run.id,
          status: result.run.status,
          benchmark_name: result.run.benchmark_name,
          dataset_name: result.run.dataset_name,
          case_count: result.summary.case_count,
          passed_count: result.summary.passed_count,
          failed_count: result.summary.failed_count,
          aggregate_scores: result.summary.aggregate_scores
        }

        if Keyword.get(parsed, :json, false) do
          Mix.shell().info(Jason.encode!(payload))
        else
          Mix.shell().info("Evaluation run #{payload.run_id}")
          Mix.shell().info("  benchmark: #{payload.benchmark_name}")
          Mix.shell().info("  dataset: #{payload.dataset_name}")
          Mix.shell().info("  status: #{payload.status}")
          Mix.shell().info("  cases: #{payload.case_count}")
          Mix.shell().info("  passed: #{payload.passed_count}")
          Mix.shell().info("  failed: #{payload.failed_count}")
          Mix.shell().info("  scores: #{Jason.encode!(payload.aggregate_scores)}")
        end

      {:error, reason} ->
        Mix.shell().error("Evaluation run failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  defp format(nil), do: nil
  defp format("json"), do: :json
  defp format("jsonl"), do: :jsonl
  defp format("ndjson"), do: :jsonl
  defp format(other), do: other
end
