defmodule Mix.Tasks.Optimal.Skills do
  @shortdoc "List and promote workflow traces into skill packages"
  @moduledoc """
  Inspect and promote procedural knowledge in Memory Core.

  Repeatable procedures are captured as Workflow Traces (via
  `OptimalEngine.MemoryCore.WorkflowSkill.record_trace/2`) and, once a pattern
  repeats past the promotion threshold, folded into a Generalized Workflow,
  Procedural Memory Object, and a draft (disabled) Skill Package.

  Usage:
      mix optimal.skills list                       # list skill packages
      mix optimal.skills list --traces              # list workflow traces
      mix optimal.skills list --workflows           # list generalized workflows
      mix optimal.skills list --procedures          # list procedural memory objects
      mix optimal.skills promote <workflow_family>  # promote a repeating pattern
      mix optimal.skills promote <family> --threshold 3 --subject <anchor>

  Options:
      --workspace <id>   workspace (default: "default")
      --threshold <n>    traces required before promotion (default: 2)
      --subject <anchor> restrict to one subject anchor
  """

  use Mix.Task

  alias OptimalEngine.MemoryCore.{Store, WorkflowSkill}

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        strict: [
          workspace: :string,
          threshold: :integer,
          subject: :string,
          traces: :boolean,
          workflows: :boolean,
          procedures: :boolean
        ]
      )

    workspace = Keyword.get(opts, :workspace, "default")

    case positional do
      ["promote", family | _] -> promote(family, workspace, opts)
      ["promote"] -> IO.puts("Usage: mix optimal.skills promote <workflow_family>")
      ["list" | _] -> list(workspace, opts)
      [] -> list(workspace, opts)
      _ -> IO.puts(@moduledoc)
    end
  end

  defp list(workspace, opts) do
    cond do
      Keyword.get(opts, :traces) ->
        {:ok, rows} = Store.list_workflow_traces(workspace)

        print("Workflow Traces", rows, fn t ->
          "#{t.id}  family=#{t.workflow_family}  subject=#{t.subject_anchor || "-"}  state=#{t.lifecycle_state}"
        end)

      Keyword.get(opts, :workflows) ->
        {:ok, rows} = Store.list_generalized_workflows(workspace)

        print("Generalized Workflows", rows, fn w ->
          "#{w.id}  family=#{w.workflow_family}  traces=#{length(w.workflow_trace_links)}  state=#{w.lifecycle_state}"
        end)

      Keyword.get(opts, :procedures) ->
        {:ok, rows} = Store.list_procedural_memory_objects(workspace)

        print("Procedural Memory Objects", rows, fn p ->
          "#{p.id}  capability=#{p.capability_name}  risk=#{p.risk_class}  state=#{p.lifecycle_state}"
        end)

      true ->
        {:ok, rows} = Store.list_skill_packages(workspace)

        print("Skill Packages", rows, fn s ->
          "#{s.id}  name=#{s.skill_package_name}  v#{s.version}  review=#{s.review_status}  enabled=#{s.enabled_state}"
        end)
    end
  end

  defp promote(family, workspace, opts) do
    promote_opts =
      [workspace_id: workspace]
      |> maybe_put(:threshold, Keyword.get(opts, :threshold))
      |> maybe_put(:subject_anchor, Keyword.get(opts, :subject))
      # promote_repeated is a direct call, not via record_trace's auto path
      |> Keyword.put(:auto_promote, false)

    case WorkflowSkill.promote_repeated(family, promote_opts) do
      {:ok, :below_threshold} ->
        IO.puts("Below threshold: not enough repeating traces for \"#{family}\" yet.")

      {:ok, %{skill_package: pkg, traces: traces}} ->
        IO.puts("Promoted \"#{family}\" from #{length(traces)} traces:")
        IO.puts("  skill_package: #{pkg.id} (#{pkg.review_status}/#{pkg.enabled_state})")

      {:error, reason} ->
        IO.puts("Promotion failed: #{inspect(reason)}")
    end
  end

  defp print(title, rows, fmt) do
    IO.puts("#{title} (#{length(rows)})")

    if rows == [] do
      IO.puts("  (none)")
    else
      Enum.each(rows, fn row -> IO.puts("  " <> fmt.(row)) end)
    end
  end

  defp maybe_put(kw, _key, nil), do: kw
  defp maybe_put(kw, key, value), do: Keyword.put(kw, key, value)
end
