defmodule Mix.Tasks.Optimal.Topology do
  @shortdoc "Inspect workspace topology (nodes + members + skills)"

  @moduledoc """
  Prints a summary of the current workspace topology: nodes (by kind), top
  members, skill registry, and recent skill grants.

  ## Usage

      mix optimal.topology
      mix optimal.topology --tenant example-org
      mix optimal.topology --workspace default
      mix optimal.topology --nodes-only
      mix optimal.topology --skills-only
      mix optimal.topology approve <request-id> --workspace default:my-workspace --apply
      mix optimal.topology reject <request-id> --workspace default:my-workspace
  """

  use Mix.Task

  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Topology
  alias OptimalEngine.WorkspaceTopology

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _} =
      OptionParser.parse(args,
        strict: [
          tenant: :string,
          workspace: :string,
          nodes_only: :boolean,
          skills_only: :boolean,
          apply: :boolean,
          actor: :string
        ]
      )

    tenant_id = Keyword.get(opts, :tenant, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace)
    nodes_only = Keyword.get(opts, :nodes_only, false)
    skills_only = Keyword.get(opts, :skills_only, false)

    case rest do
      ["approve", request_id | _] ->
        review_request(request_id, :approve, tenant_id, workspace_id, opts)

      ["reject", request_id | _] ->
        review_request(request_id, :reject, tenant_id, workspace_id, opts)

      [] ->
        print_summary(tenant_id, workspace_id, nodes_only, skills_only)

      _ ->
        Mix.raise("Usage: mix optimal.topology [approve|reject <request-id>] [--workspace ID]")
    end
  end

  defp print_summary(tenant_id, workspace_id, nodes_only, skills_only) do
    IO.puts("")
    IO.puts("  Workspace topology")
    IO.puts("  tenant:    #{tenant_id}")
    IO.puts("  workspace: #{workspace_id || "(all)"}")
    IO.puts("  " <> String.duplicate("-", 60))

    if skills_only do
      print_skills(tenant_id)
    else
      print_nodes(tenant_id, workspace_id)
      unless nodes_only, do: print_skills(tenant_id)
    end

    IO.puts("")
  end

  defp review_request(request_id, decision, tenant_id, workspace_id, opts) do
    apply? = Keyword.get(opts, :apply, false)
    actor_id = Keyword.get(opts, :actor, "mix optimal.topology")

    case WorkspaceTopology.review_change_request(request_id, decision,
           tenant_id: tenant_id,
           workspace_id: workspace_id,
           actor_id: actor_id,
           apply: apply?
         ) do
      {:ok, %{request: request, applied: applied}} ->
        IO.puts("")
        IO.puts("  Topology change #{request.review_status}")
        IO.puts("  " <> String.duplicate("-", 60))
        IO.puts("  Request:   #{request.id}")
        IO.puts("  Workspace: #{request.workspace_id}")
        IO.puts("  Type:      #{request.request_type}")
        IO.puts("  Applied:   #{format_applied(applied)}")
        IO.puts("")

      {:error, reason} ->
        Mix.raise("optimal.topology #{decision} failed: #{inspect(reason)}")
    end
  end

  defp format_applied(nil), do: "no"
  defp format_applied(%{id: id}), do: id
  defp format_applied(other), do: inspect(other)

  defp print_nodes(tenant_id, workspace_id) do
    opts =
      [tenant_id: tenant_id]
      |> maybe_put(:workspace_id, workspace_id)

    case Topology.list_nodes(opts) do
      {:ok, []} ->
        IO.puts("  No nodes.")
        IO.puts("")

      {:ok, nodes} ->
        grouped = Enum.group_by(nodes, & &1.kind)

        IO.puts("  Nodes (#{length(nodes)} total)")
        IO.puts("")

        Enum.each(grouped, fn {kind, ns} ->
          IO.puts("    #{kind} (#{length(ns)}):")

          Enum.each(ns, fn n ->
            status_mark = if n.status == :active, do: "*", else: "-"
            parent = if n.parent_id, do: " <- #{n.parent_id}", else: ""
            IO.puts("      #{status_mark} #{n.slug}  [#{n.workspace_id} / #{n.style}]#{parent}")
          end)

          IO.puts("")
        end)

      _ ->
        IO.puts("  (nodes unavailable)")
    end
  end

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, value), do: Keyword.put(opts, key, value)

  defp print_skills(tenant_id) do
    case Topology.list_skills(tenant_id: tenant_id) do
      {:ok, []} ->
        IO.puts("  No skills registered.")

      {:ok, skills} ->
        IO.puts("  Skills (#{length(skills)} total)")
        IO.puts("")

        Enum.each(skills, fn s ->
          kind_str = if s.kind, do: " [#{s.kind}]", else: ""
          IO.puts("    - #{s.name}#{kind_str}")
        end)

      _ ->
        IO.puts("  (skills unavailable)")
    end
  end
end
