defmodule OptimalEngine.WorkspaceInitiation do
  @moduledoc """
  First-run initiation workflow for a customizable Optimal Engine workspace.

  Initiation is the bridge between a messy human data dump and a governed
  workspace structure. It preserves the dump as source evidence, extracts an
  unreviewed setup Claim, and conservatively proposes candidate Nodes.

  By default, conservative Node proposals are applied so a first-time user gets
  a usable workspace immediately. The setup Claim remains unreviewed and no
  Claims become Facts. Pass `review_only: true` to leave every candidate as a
  pending topology-change request.
  """

  alias OptimalEngine.MemoryCore
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.WorkspaceTopology

  @candidate_kinds ~w[entity department team project operation learning person product partnership context]

  @type candidate :: %{
          kind: String.t(),
          slug: String.t(),
          name: String.t(),
          reason: String.t(),
          confidence: float(),
          source_line: String.t(),
          metadata: map()
        }

  @type result :: %{
          setup: WorkspaceTopology.setup_result(),
          source_package: SourcePackage.t() | nil,
          setup_claim: map() | nil,
          proposed_nodes: [candidate()],
          proposed_integrations: [map()],
          registered_tool_definitions: [map()],
          topology_change_requests: [WorkspaceTopology.topology_change_request()],
          applied_topology_changes: [map()],
          accepted_nodes: [map()],
          open_questions: [String.t()],
          next_actions: [String.t()]
        }

  @doc """
  Initiate a workspace from a natural-language or markdown context dump.

  Required fields:

    * `:slug`
    * `:name`
    * `:dump_text` or `:dump_path`

  Unless `review_only: true` is set, conservative Node candidates are reviewed
  and applied during initiation. The original setup Claim remains unreviewed.
  """
  @spec initiate(map()) :: {:ok, result()} | {:error, term()}
  def initiate(%{slug: slug, name: name} = attrs) when is_binary(slug) and is_binary(name) do
    actor_id = Map.get(attrs, :actor_id, "workspace_initiation")
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())

    with {:ok, dump_text} <- load_dump_text(attrs),
         {:ok, setup} <- setup_workspace(attrs, actor_id, tenant_id),
         {:ok, source_package, setup_claim} <-
           preserve_setup_source(setup.workspace.id, dump_text, attrs, actor_id, tenant_id),
         proposed_nodes <- infer_node_candidates(dump_text),
         {:ok, topology_requests} <-
           create_topology_requests(setup.workspace.id, proposed_nodes, attrs, actor_id, tenant_id),
         {:ok, applied_topology_changes, accepted_setup} <-
           maybe_apply_topology_requests(setup, topology_requests, proposed_nodes, attrs, actor_id),
         proposed_integrations <- infer_integration_candidates(dump_text),
         {:ok, tool_definitions} <-
           register_integration_placeholders(
             setup.workspace.id,
             proposed_integrations,
             actor_id,
             tenant_id
           ) do
      {:ok,
       %{
         setup: setup,
         source_package: source_package,
         setup_claim: setup_claim,
         proposed_nodes: proposed_nodes,
         proposed_integrations: proposed_integrations,
         registered_tool_definitions: tool_definitions,
         topology_change_requests: topology_requests,
         applied_topology_changes: applied_topology_changes,
         accepted_nodes: accepted_setup.nodes,
         open_questions: open_questions(proposed_nodes, proposed_integrations, dump_text),
         next_actions:
           next_actions(setup.workspace.id, topology_requests, applied_topology_changes, attrs)
       }}
    end
  end

  def initiate(_attrs), do: {:error, :missing_required_fields}

  @doc """
  Infer candidate Nodes from a setup dump without writing anything.

  This parser is deliberately conservative. It favors explicit headings,
  bullets, and labels over speculative extraction.
  """
  @spec infer_node_candidates(String.t()) :: [candidate()]
  def infer_node_candidates(dump_text) when is_binary(dump_text) do
    dump_text
    |> String.split(~r/\R/)
    |> Enum.map(&String.trim/1)
    |> Enum.reject(&(&1 == ""))
    |> scan_lines(nil, [])
    |> Enum.reverse()
    |> unique_candidates()
  end

  @doc """
  Infer external source/tool surfaces mentioned in a setup dump.

  These are not topology Nodes by default. They are integration candidates that
  should become connector rows, MCP tools, API tools, scripts, or model/tool
  definitions only after scope and credentials are confirmed.
  """
  @spec infer_integration_candidates(String.t()) :: [map()]
  def infer_integration_candidates(dump_text) when is_binary(dump_text) do
    lower = String.downcase(dump_text)

    known =
      [
        {"google_workspace", "Google Workspace", "mcp_or_api",
         ["google workspace", "gmail", "google drive", "google calendar", "calendar"]},
        {"fathom", "Fathom", "mcp_or_api", ["fathom"]},
        {"slack", "Slack", "connector_or_api", ["slack"]},
        {"github", "GitHub", "connector_or_api", ["github"]},
        {"notion", "Notion", "connector_or_api", ["notion"]},
        {"linear", "Linear", "connector_or_api", ["linear"]},
        {"jira", "Jira", "connector_or_api", ["jira"]},
        {"hubspot", "HubSpot", "connector_or_api", ["hubspot"]},
        {"zoom", "Zoom", "connector_or_api", ["zoom"]},
        {"custom_api", "Custom API", "api", ["custom api", "api endpoint", "webhook"]},
        {"script", "Script", "script",
         ["script", "shell command", "cron job", "automation script"]},
        {"mcp", "MCP Server", "mcp", ["mcp server", "mcp tool", "mcps"]}
      ]

    known
    |> Enum.filter(fn {_slug, _name, _surface, needles} -> contains_any?(lower, needles) end)
    |> Enum.map(fn {slug, name, surface_type, needles} ->
      %{
        slug: slug,
        name: name,
        surface_type: surface_type,
        status: "needs_scope",
        reason: "Mentioned during workspace initiation.",
        matched_terms: Enum.filter(needles, &String.contains?(lower, &1)),
        required_decisions: [
          "credential_source",
          "allowed_scopes",
          "allowed_nodes_or_partitions",
          "read_write_policy",
          "sync_or_call_cadence"
        ]
      }
    end)
  end

  defp load_dump_text(%{dump_text: text}) when is_binary(text) and byte_size(text) > 0,
    do: {:ok, text}

  defp load_dump_text(%{dump_path: path}) when is_binary(path) do
    case File.read(path) do
      {:ok, text} when byte_size(text) > 0 -> {:ok, text}
      {:ok, _empty} -> {:error, :empty_dump}
      {:error, reason} -> {:error, {:dump_read_failed, reason}}
    end
  end

  defp load_dump_text(_attrs), do: {:error, :dump_required}

  defp setup_workspace(attrs, actor_id, tenant_id) do
    WorkspaceTopology.setup_workspace(%{
      slug: attrs.slug,
      name: attrs.name,
      description: Map.get(attrs, :description),
      tenant_id: tenant_id,
      root: Map.get(attrs, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!())),
      starter_nodes: Map.get(attrs, :starter_nodes, true),
      write_files: Map.get(attrs, :write_files, true),
      write_rhythm: Map.get(attrs, :write_rhythm, true),
      write_agent_sop: Map.get(attrs, :write_agent_sop, true),
      actor_id: actor_id
    })
  end

  defp preserve_setup_source(workspace_id, dump_text, attrs, actor_id, tenant_id) do
    source_package =
      SourcePackage.from_text(dump_text,
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        source_type: "workspace_initiation_dump",
        source_class: "text",
        source_system: "optimal.initiate",
        source_uri: Map.get(attrs, :dump_path),
        trust_label: "user_reported",
        retention_class: Map.get(attrs, :retention_class, "standard"),
        security_labels: Map.get(attrs, :security_labels, ["private"]),
        partition_ids: [workspace_id],
        metadata: %{
          "initiation" => true,
          "workspace_slug" => attrs.slug,
          "ingestion_contract" => "source_to_pending_claim_to_topology_proposals"
        },
        actor_id: actor_id
      )

    with {:ok, claim} <-
           MemoryCore.extract_claim(source_package,
             claim_text: "The user provided a first-run workspace context dump for setup.",
             claim_type: "workspace_setup_context",
             subject_anchor: attrs.slug,
             action_class: "initiates_workspace",
             object_anchor: workspace_id,
             aggregate_confidence: 0.7,
             aggregate_precision: 0.62,
             actor_id: actor_id,
             metadata: %{
               "truth_status" => "unreviewed",
               "setup_role" => "preserve_input_before_structure"
             }
           ) do
      {:ok, source_package, claim}
    end
  end

  defp create_topology_requests(workspace_id, candidates, attrs, actor_id, tenant_id) do
    Enum.reduce_while(candidates, {:ok, []}, fn candidate, {:ok, acc} ->
      request_attrs = %{
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        request_type: "create_node",
        target_object_type: "node",
        proposed_payload: %{
          slug: candidate.slug,
          name: candidate.name,
          kind: candidate.kind,
          description: "Proposed during workspace initiation: #{candidate.reason}",
          metadata:
            Map.merge(candidate.metadata, %{
              "source_line" => candidate.source_line,
              "confidence" => candidate.confidence,
              "initiation_candidate" => true
            })
        },
        reason: candidate.reason,
        requested_by: actor_id,
        audit_event_links: [
          %{
            type: "workspace_initiation",
            actor_id: actor_id,
            workspace_slug: attrs.slug
          }
        ]
      }

      case WorkspaceTopology.propose_change(request_attrs) do
        {:ok, request} -> {:cont, {:ok, [request | acc]}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, requests} -> {:ok, Enum.reverse(requests)}
      other -> other
    end
  end

  defp register_integration_placeholders(_workspace_id, [], _actor_id, _tenant_id), do: {:ok, []}

  defp register_integration_placeholders(workspace_id, integrations, actor_id, tenant_id) do
    Enum.reduce_while(integrations, {:ok, []}, fn integration, {:ok, acc} ->
      protocol_adapter_id = protocol_adapter_id(integration.surface_type)
      tool_name = integration_tool_name(integration)

      case MemoryCore.register_mcp_tool_definition(
             tenant_id: tenant_id,
             workspace_id: workspace_id,
             tool_name: tool_name,
             protocol_adapter_id: protocol_adapter_id,
             implementation_type: integration.surface_type,
             enabled_state: "disabled",
             lifecycle_state: "draft",
             registration_source: "workspace_initiation",
             required_privileges: ["integration:#{integration.slug}:use"],
             allowed_partitions: [],
             input_schema: %{required: ["operation"]},
             output_schema: %{required: ["status"]},
             execution_policy: %{
               review_required: true,
               credentials_required: true,
               side_effects_allowed: false
             },
             routing_policy: %{
               workspace_id: workspace_id,
               integration_slug: integration.slug,
               status: "needs_scope"
             },
             audit_policy: %{record_every_call: true},
             security_labels: ["private"],
             partition_ids: [workspace_id],
             metadata: Map.merge(integration, %{registered_by: actor_id})
           ) do
        {:ok, definition} -> {:cont, {:ok, [definition | acc]}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, definitions} -> {:ok, Enum.reverse(definitions)}
      other -> other
    end
  end

  defp maybe_apply_topology_requests(setup, requests, candidates, attrs, actor_id) do
    if Map.get(attrs, :review_only, false) do
      {:ok, [], %{setup | nodes: []}}
    else
      workspace = setup.workspace

      with {:ok, applied} <-
             apply_topology_requests(requests,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               actor_id: actor_id
             ),
           {:ok, accepted_setup} <-
             write_accepted_node_projections(setup, candidates, attrs, actor_id) do
        {:ok, applied, accepted_setup}
      end
    end
  end

  defp apply_topology_requests(requests, opts) do
    Enum.reduce_while(requests, {:ok, []}, fn request, {:ok, acc} ->
      case WorkspaceTopology.review_change_request(request.id, :approve,
             tenant_id: Keyword.fetch!(opts, :tenant_id),
             workspace_id: Keyword.fetch!(opts, :workspace_id),
             actor_id: Keyword.fetch!(opts, :actor_id),
             apply: true
           ) do
        {:ok, reviewed} -> {:cont, {:ok, [reviewed | acc]}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, applied} -> {:ok, Enum.reverse(applied)}
      other -> other
    end
  end

  defp write_accepted_node_projections(setup, candidates, attrs, actor_id) do
    candidate_nodes =
      Enum.map(candidates, fn candidate ->
        %{
          kind: candidate.kind,
          slug: candidate.slug,
          name: candidate.name,
          description: "Created during workspace initiation: #{candidate.reason}",
          metadata:
            Map.merge(candidate.metadata, %{
              "source_line" => candidate.source_line,
              "confidence" => candidate.confidence,
              "initiation_candidate" => true,
              "accepted_during_initiation" => true
            })
        }
      end)

    if candidate_nodes == [] do
      {:ok, %{setup | nodes: []}}
    else
      WorkspaceTopology.setup_workspace(%{
        slug: setup.workspace.slug,
        name: setup.workspace.name,
        description: setup.workspace.description,
        tenant_id: setup.workspace.tenant_id,
        root: Map.get(attrs, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!())),
        starter_nodes: false,
        write_files: Map.get(attrs, :write_files, true),
        write_rhythm: false,
        write_agent_sop: false,
        actor_id: actor_id,
        nodes: candidate_nodes
      })
    end
  end

  defp scan_lines([], _current_kind, acc), do: acc

  defp scan_lines([line | rest], current_kind, acc) do
    cond do
      section_kind(line) ->
        scan_lines(rest, section_kind(line), acc)

      labeled_entries?(line) ->
        {kind, names} = labeled_entries(line)

        candidates =
          Enum.map(names, &candidate(kind, &1, line, "Explicit #{kind} label in setup dump."))

        scan_lines(rest, current_kind, candidates ++ acc)

      current_kind && list_item?(line) ->
        name = clean_item(line)

        next_acc =
          maybe_add_candidate(
            acc,
            current_kind,
            name,
            line,
            "Listed under #{current_kind} section."
          )

        scan_lines(rest, current_kind, next_acc)

      true ->
        scan_lines(rest, current_kind, acc)
    end
  end

  defp section_kind(line) do
    normalized =
      line
      |> String.downcase()
      |> String.trim_leading("#")
      |> String.trim()

    cond do
      contains_any?(normalized, ["companies", "organizations", "businesses", "entities"]) ->
        "entity"

      contains_any?(normalized, ["departments", "functions"]) ->
        "department"

      contains_any?(normalized, ["teams"]) ->
        "team"

      contains_any?(normalized, ["projects", "initiatives"]) ->
        "project"

      contains_any?(normalized, [
        "operations",
        "processes",
        "rhythm",
        "cadence",
        "workflows",
        "sops"
      ]) ->
        "operation"

      contains_any?(normalized, ["learning", "research", "knowledge"]) ->
        "learning"

      contains_any?(normalized, ["people", "team members", "clients", "contacts"]) ->
        "person"

      contains_any?(normalized, ["products", "platforms", "tools", "apps"]) ->
        "product"

      contains_any?(normalized, ["partnerships", "partners"]) ->
        "partnership"

      contains_any?(normalized, ["context", "references", "integrations", "connectors"]) ->
        "context"

      true ->
        nil
    end
  end

  defp labeled_entries?(line) do
    line
    |> String.downcase()
    |> String.match?(
      ~r/^\s*(companies|organizations|entities|projects|people|clients|products|operations|rhythm|research|integrations|tools)\s*:/
    )
  end

  defp labeled_entries(line) do
    [label, value] = String.split(line, ":", parts: 2)
    kind = label_to_kind(label)

    names =
      value
      |> String.split([",", ";"])
      |> Enum.map(&clean_item/1)
      |> Enum.reject(&reject_candidate_name?/1)

    {kind, names}
  end

  defp label_to_kind(label) do
    label = String.downcase(label)

    cond do
      contains_any?(label, ["companies", "organizations", "entities"]) -> "entity"
      contains_any?(label, ["projects"]) -> "project"
      contains_any?(label, ["people", "clients"]) -> "person"
      contains_any?(label, ["products", "tools"]) -> "product"
      contains_any?(label, ["operations", "rhythm"]) -> "operation"
      contains_any?(label, ["research"]) -> "learning"
      contains_any?(label, ["integrations"]) -> "context"
      true -> "context"
    end
  end

  defp list_item?(line), do: String.match?(line, ~r/^[-*+] |\d+\.\s+/)

  defp clean_item(line) do
    line
    |> String.replace(~r/^[-*+]\s+/, "")
    |> String.replace(~r/^\d+\.\s+/, "")
    |> String.replace(~r/^\[[ xX]\]\s+/, "")
    |> String.split(~r/\s+[-–—]\s+/, parts: 2)
    |> hd()
    |> String.split(":", parts: 2)
    |> hd()
    |> String.trim()
    |> String.trim("\"'")
  end

  defp maybe_add_candidate(acc, kind, name, source_line, reason) do
    if reject_candidate_name?(name) or kind not in @candidate_kinds do
      acc
    else
      [candidate(kind, name, source_line, reason) | acc]
    end
  end

  defp candidate(kind, name, source_line, reason) do
    %{
      kind: kind,
      slug: slugify(name),
      name: name,
      reason: reason,
      confidence: 0.64,
      source_line: source_line,
      metadata: %{
        "candidate_source" => "workspace_initiation_parser",
        "review_required" => true
      }
    }
  end

  defp reject_candidate_name?(name) do
    normalized = String.downcase(String.trim(name))

    normalized == "" or
      String.length(normalized) < 2 or
      word_count(normalized) > 10 or
      normalized in ["none", "n/a", "todo", "unknown", "misc", "etc"] or
      String.ends_with?(normalized, "?")
  end

  defp unique_candidates(candidates) do
    candidates
    |> Enum.uniq_by(fn candidate -> {candidate.kind, candidate.slug} end)
    |> Enum.take(40)
  end

  defp open_questions(candidates, integrations, dump_text) do
    present_kinds = MapSet.new(Enum.map(candidates, & &1.kind))
    lower = String.downcase(dump_text)

    [
      "Which proposed Nodes should become durable workspace structure?",
      missing_kind_question(
        present_kinds,
        "entity",
        "What organization, company, or personal operating entity owns this workspace?"
      ),
      missing_kind_question(
        present_kinds,
        "person",
        "Who are the primary humans, agents, customers, or partners involved?"
      ),
      missing_kind_question(
        present_kinds,
        "project",
        "What are the active projects or initiatives that need tracking?"
      ),
      missing_kind_question(
        present_kinds,
        "operation",
        "What operating rhythm, recurring processes, or review cadence should govern this workspace?"
      ),
      if(
        String.contains?(lower, [
          "calendar",
          "gmail",
          "slack",
          "github",
          "drive",
          "notion",
          "linear",
          "hubspot",
          "integration",
          "connector"
        ]),
        do: "Which integrations should connect first, and what permissions should each one have?",
        else: nil
      ),
      integration_question(integrations),
      if(MapSet.member?(present_kinds, "project") and MapSet.member?(present_kinds, "entity"),
        do: "Which projects belong under which organization or entity Node?",
        else: nil
      )
    ]
    |> Enum.reject(&is_nil/1)
  end

  defp missing_kind_question(present_kinds, kind, question) do
    if MapSet.member?(present_kinds, kind), do: nil, else: question
  end

  defp next_actions(workspace_id, requests, applied_topology_changes, attrs) do
    review_command =
      cond do
        applied_topology_changes != [] ->
          "Inspect accepted Nodes with: mix optimal.topology --workspace #{workspace_id}"

        Map.get(attrs, :review_only, false) and requests != [] ->
          first_request = hd(requests)

          "Review proposals, then approve one with: mix optimal.topology approve #{first_request.id} --workspace #{workspace_id} --apply"

        true ->
          "Add explicit Nodes with: mix optimal.setup #{workspace_id} --node project:example:Example"
      end

    [
      review_command,
      "Confirm integration scopes before enabling any MCP/API/script tools.",
      "Ingest edited markdown back as source evidence instead of treating file edits as automatic truth.",
      "After approval, retrieve context with: mix optimal.rag \"what should I work on next?\" --workspace #{workspace_id} --trace"
    ]
  end

  defp integration_question([]), do: nil

  defp integration_question(integrations) do
    names = integrations |> Enum.map(& &1.name) |> Enum.join(", ")

    "For #{names}, where are credentials stored, which data can be read, and what actions require confirmation?"
  end

  defp protocol_adapter_id("mcp"), do: "mcp"
  defp protocol_adapter_id("mcp_or_api"), do: "mcp"
  defp protocol_adapter_id("api"), do: "api"
  defp protocol_adapter_id("script"), do: "script"
  defp protocol_adapter_id(_surface_type), do: "connector"

  defp integration_tool_name(%{surface_type: "script"} = integration),
    do: "script.#{integration.slug}.run"

  defp integration_tool_name(%{surface_type: "api"} = integration),
    do: "api.#{integration.slug}.call"

  defp integration_tool_name(%{surface_type: "mcp"} = integration),
    do: "mcp.#{integration.slug}.call"

  defp integration_tool_name(integration), do: "integration.#{integration.slug}.call"

  defp contains_any?(text, needles), do: Enum.any?(needles, &String.contains?(text, &1))

  defp slugify(name) do
    name
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9]+/, "-")
    |> String.trim("-")
    |> case do
      "" -> "untitled"
      slug -> slug
    end
  end

  defp word_count(text), do: text |> String.split(~r/\s+/, trim: true) |> length()
end
