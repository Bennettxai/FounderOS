defmodule Mix.Tasks.Optimal.Setup do
  @shortdoc "Create a workspace with starter Nodes, rhythm, projections, and agent SOP"

  @moduledoc """
  Creates or refreshes a workspace operating topology.

  This is the first command a human or agent CLI should run when starting a new
  workspace. It creates the workspace if needed, seeds standard Node Types,
  creates starter Nodes, writes markdown/rhythm/agent files, and records the
  generated files as projections.

  ## Usage

      mix optimal.setup my-workspace --name "My Workspace"
      mix optimal.setup client-os --node project:launch-plan:"Launch Plan"
      mix optimal.setup research --no-starter-nodes --node learning:papers:"Papers"

  ## Node format

      --node kind:slug:name

  Examples:

      --node project:launch-plan:"Launch Plan"
      --node person:founder:"Founder"
      --node product:portal:"Customer Portal"
  """

  use Mix.Task

  alias OptimalEngine.WorkspaceTopology

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _invalid} =
      OptionParser.parse(args,
        strict: [
          name: :string,
          description: :string,
          tenant: :string,
          root: :string,
          node: :keep,
          no_starter_nodes: :boolean,
          no_files: :boolean,
          no_rhythm: :boolean,
          no_agent_sop: :boolean
        ]
      )

    slug =
      case rest do
        [slug | _] -> slug
        [] -> Mix.raise("Usage: mix optimal.setup <workspace-slug> [--name NAME]")
      end

    attrs = %{
      slug: slug,
      name: Keyword.get(opts, :name, titleize(slug)),
      description: Keyword.get(opts, :description),
      tenant_id: Keyword.get(opts, :tenant, OptimalEngine.Tenancy.Tenant.default_id()),
      root: Keyword.get(opts, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!())),
      starter_nodes: not Keyword.get(opts, :no_starter_nodes, false),
      write_files: not Keyword.get(opts, :no_files, false),
      write_rhythm: not Keyword.get(opts, :no_rhythm, false),
      write_agent_sop: not Keyword.get(opts, :no_agent_sop, false),
      actor_id: "mix optimal.setup",
      nodes: parse_nodes(Keyword.get_values(opts, :node))
    }

    case WorkspaceTopology.setup_workspace(attrs) do
      {:ok, result} ->
        print_result(result)

      {:error, reason} ->
        Mix.raise("optimal.setup failed: #{inspect(reason)}")
    end
  end

  defp parse_nodes(values) do
    Enum.map(values, fn value ->
      case String.split(value, ":", parts: 3) do
        [kind, slug, name] ->
          %{kind: kind, slug: slug, name: name, description: "Custom setup Node."}

        _ ->
          Mix.raise("Invalid --node #{inspect(value)}. Expected kind:slug:name")
      end
    end)
  end

  defp print_result(result) do
    Mix.shell().info("""

    Optimal workspace ready
    #{String.duplicate("-", 60)}
    Workspace: #{result.workspace.id}
    Path:      #{result.path}

    Node Types: #{length(result.node_types)}
    Nodes:      #{length(result.nodes)}
    Relations:  #{length(result.relationships)}
    Exports:    #{length(result.exports)}

    Starter Nodes:
    #{format_nodes(result.nodes)}

    Agent CLI SOP:
      #{result.agent_sop_path || "(not written)"}

    Next:
      mix optimal.topology --workspace #{result.workspace.id}
      mix optimal.ingest_workspace #{result.path}
      mix optimal.rag "what changed this week?" --workspace #{result.workspace.id} --trace
    """)
  end

  defp format_nodes([]), do: "  (none)"

  defp format_nodes(nodes) do
    nodes
    |> Enum.map(fn node -> "  - #{node.kind}:#{node.slug}  #{node.name}" end)
    |> Enum.join("\n")
  end

  defp titleize(slug) do
    slug
    |> String.replace(["-", "_"], " ")
    |> String.split()
    |> Enum.map_join(" ", &String.capitalize/1)
  end
end
