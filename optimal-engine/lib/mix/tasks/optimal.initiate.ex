defmodule Mix.Tasks.Optimal.Initiate do
  @shortdoc "Start a workspace from a natural-language setup dump"

  @moduledoc """
  Creates a workspace operating surface and turns a messy setup dump into
  reviewable structure proposals.

  The dump is preserved as a Source Package, converted into an unreviewed setup
  Claim, and scanned for candidate Nodes. By default, conservative Node
  proposals are applied immediately so the workspace is usable. Use
  `--review-only` to leave proposals pending instead.

  ## Usage

      mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
      mix optimal.initiate founder-os --text "Projects: Launch, Hiring"

  ## Options

    * `--name` - workspace display name
    * `--dump` - path to a markdown/text setup dump
    * `--text` - inline setup dump text
    * `--tenant` - tenant id, defaults to `default`
    * `--root` - workspace filesystem root
    * `--no-starter-nodes` - skip starter nodes
    * `--no-files` - skip markdown projection files
    * `--no-rhythm` - skip rhythm folders
    * `--no-agent-sop` - skip AGENTS.md projection
    * `--review-only` - preserve proposals without applying them
  """

  use Mix.Task

  alias OptimalEngine.WorkspaceInitiation

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _invalid} =
      OptionParser.parse(args,
        strict: [
          name: :string,
          dump: :string,
          text: :string,
          description: :string,
          tenant: :string,
          root: :string,
          no_starter_nodes: :boolean,
          no_files: :boolean,
          no_rhythm: :boolean,
          no_agent_sop: :boolean,
          review_only: :boolean
        ]
      )

    slug =
      case rest do
        [slug | _] -> slug
        [] -> Mix.raise("Usage: mix optimal.initiate <workspace-slug> --dump setup.md")
      end

    attrs =
      %{
        slug: slug,
        name: Keyword.get(opts, :name, titleize(slug)),
        description: Keyword.get(opts, :description),
        tenant_id: Keyword.get(opts, :tenant, OptimalEngine.Tenancy.Tenant.default_id()),
        root:
          Keyword.get(opts, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!())),
        starter_nodes: not Keyword.get(opts, :no_starter_nodes, false),
        write_files: not Keyword.get(opts, :no_files, false),
        write_rhythm: not Keyword.get(opts, :no_rhythm, false),
        write_agent_sop: not Keyword.get(opts, :no_agent_sop, false),
        review_only: Keyword.get(opts, :review_only, false),
        actor_id: "mix optimal.initiate"
      }
      |> maybe_put(:dump_path, Keyword.get(opts, :dump))
      |> maybe_put(:dump_text, Keyword.get(opts, :text))

    case WorkspaceInitiation.initiate(attrs) do
      {:ok, result} ->
        print_result(result)

      {:error, reason} ->
        Mix.raise("optimal.initiate failed: #{inspect(reason)}")
    end
  end

  defp print_result(result) do
    Mix.shell().info("""

    Optimal workspace initiated
    #{String.duplicate("-", 60)}
    Workspace:      #{result.setup.workspace.id}
    Path:           #{result.setup.path}
    Source Package: #{result.source_package && result.source_package.id}
    Setup Claim:    #{result.setup_claim && result.setup_claim.id}

    Starter Nodes:  #{length(result.setup.nodes)}
    Proposals:      #{length(result.topology_change_requests)}
    Applied:        #{length(result.applied_topology_changes)}
    Accepted Nodes: #{length(result.accepted_nodes)}
    Integrations:   #{length(result.proposed_integrations)}

    Proposed Nodes:
    #{format_proposals(result.proposed_nodes, result.topology_change_requests)}

    Integration Surfaces:
    #{format_integrations(result.proposed_integrations, result.registered_tool_definitions)}

    Accepted Nodes:
    #{format_nodes(result.accepted_nodes)}

    Open Questions:
    #{format_list(result.open_questions)}

    Next:
    #{format_list(result.next_actions)}
    """)
  end

  defp format_proposals([], _requests),
    do: "  (none detected; add explicit nodes or improve the setup dump)"

  defp format_proposals(candidates, requests) do
    requests_by_slug =
      Map.new(requests, fn request -> {request.proposed_payload["slug"], request.id} end)

    candidates
    |> Enum.map(fn candidate ->
      request_id = Map.get(requests_by_slug, candidate.slug, "(request unavailable)")
      "  - #{candidate.kind}:#{candidate.slug}  #{candidate.name}  [#{request_id}]"
    end)
    |> Enum.join("\n")
  end

  defp format_integrations([], _definitions), do: "  (none detected)"

  defp format_integrations(integrations, definitions) do
    definitions_by_slug =
      Map.new(definitions, fn definition ->
        slug = definition.metadata[:slug] || definition.metadata["slug"]
        {slug, definition}
      end)

    integrations
    |> Enum.map(fn integration ->
      definition = Map.get(definitions_by_slug, integration.slug)
      tool = if definition, do: definition.tool_name, else: "(tool placeholder unavailable)"

      "  - #{integration.surface_type}:#{integration.slug}  #{integration.name}  [#{tool}, disabled]"
    end)
    |> Enum.join("\n")
  end

  defp format_nodes([]),
    do: "  (none applied; run without --review-only to apply conservative candidates)"

  defp format_nodes(nodes) do
    nodes
    |> Enum.map(fn node -> "  - #{node.kind}:#{node.slug}  #{node.name}" end)
    |> Enum.join("\n")
  end

  defp format_list([]), do: "  (none)"

  defp format_list(items) do
    items
    |> Enum.map(&"  - #{&1}")
    |> Enum.join("\n")
  end

  defp maybe_put(map, _key, nil), do: map
  defp maybe_put(map, _key, ""), do: map
  defp maybe_put(map, key, value), do: Map.put(map, key, value)

  defp titleize(slug) do
    slug
    |> String.replace(["-", "_"], " ")
    |> String.split()
    |> Enum.map_join(" ", &String.capitalize/1)
  end
end
