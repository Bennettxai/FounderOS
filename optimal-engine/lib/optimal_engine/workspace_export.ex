defmodule OptimalEngine.WorkspaceExport do
  @moduledoc """
  Lifecycle facade for workspace projections.

  Markdown, HTML, wiki pages, app views, reports, and packages are surfaces.
  This module records those surfaces as generated projections tied to source
  object versions. It is the first step toward making human-visible files and
  app views inspectable, rebuildable, and safe to re-ingest.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.MemoryCore.SourcePackageService
  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.WorkspaceTopology

  @type export_record :: %{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          source_object_type: String.t(),
          source_object_id: String.t(),
          surface_type: String.t(),
          destination_uri: String.t(),
          object_version: String.t() | nil,
          policy_version: String.t() | nil,
          lifecycle_state: String.t(),
          generated_by: String.t() | nil,
          generated_at: String.t() | nil,
          metadata: map()
        }

  @type projection_revision :: %{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          export_record_id: String.t(),
          revision_number: non_neg_integer(),
          content_hash: String.t(),
          projection_uri: String.t(),
          source_object_links: [map()],
          lifecycle_state: String.t(),
          created_by: String.t() | nil,
          created_at: String.t() | nil,
          metadata: map()
        }

  @type link_health_record :: %{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          export_record_id: String.t() | nil,
          projection_revision_id: String.t() | nil,
          status: String.t(),
          broken_links: [String.t()],
          backlinks: [String.t()],
          missing_references: [String.t()],
          stale_references: [String.t()],
          checked_at: String.t() | nil,
          metadata: map()
        }

  @type edit_capture :: %{
          kind: :source_package | :topology_change_request,
          export_record: export_record(),
          source_package: map() | nil,
          intake_result: map() | nil,
          topology_change_request: map() | nil
        }

  @type invalidation_result :: %{
          workspace_id: String.t(),
          source_object_type: String.t(),
          source_object_id: String.t(),
          export_record_ids: [String.t()],
          projection_revision_count: non_neg_integer(),
          link_health_record_count: non_neg_integer()
        }

  @doc """
  Record that a source object was exported to a surface.

  Required attrs:
    * `:source_object_type`
    * `:source_object_id`
    * `:surface_type`
    * `:destination_uri`
  """
  @spec record_export(map()) :: {:ok, export_record()} | {:error, term()}
  def record_export(
        %{
          source_object_type: source_object_type,
          source_object_id: source_object_id,
          surface_type: surface_type,
          destination_uri: destination_uri
        } = attrs
      )
      when is_binary(source_object_type) and is_binary(source_object_id) and
             is_binary(surface_type) and is_binary(destination_uri) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    object_version = Map.get(attrs, :object_version)
    policy_version = Map.get(attrs, :policy_version)
    lifecycle_state = Map.get(attrs, :lifecycle_state, "current")
    generated_by = Map.get(attrs, :generated_by)
    metadata = Map.get(attrs, :metadata, %{})

    id =
      Map.get(attrs, :id) ||
        deterministic_id("export", [
          workspace_id,
          source_object_type,
          source_object_id,
          surface_type,
          destination_uri
        ])

    sql = """
    INSERT INTO export_records (
      id, tenant_id, workspace_id, source_object_type, source_object_id,
      surface_type, destination_uri, object_version, policy_version,
      lifecycle_state, generated_by, metadata
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
    ON CONFLICT(workspace_id, source_object_type, source_object_id, surface_type, destination_uri)
    DO UPDATE SET
      object_version = excluded.object_version,
      policy_version = excluded.policy_version,
      lifecycle_state = excluded.lifecycle_state,
      generated_by = excluded.generated_by,
      generated_at = datetime('now'),
      metadata = excluded.metadata
    """

    with :ok <-
           Store.raw_execute(sql, [
             id,
             tenant_id,
             workspace_id,
             source_object_type,
             source_object_id,
             surface_type,
             destination_uri,
             object_version,
             policy_version,
             lifecycle_state,
             generated_by,
             Jason.encode!(metadata)
           ]) do
      get_export(id, tenant_id: tenant_id, workspace_id: workspace_id)
    end
  end

  @doc "Fetch an export record by id."
  @spec get_export(String.t(), keyword()) :: {:ok, export_record()} | {:error, :not_found}
  def get_export(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             export_columns() <>
             " FROM export_records WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_export(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Record a new projection revision for an export.

  `:content` may be supplied instead of `:content_hash`; the hash will be
  calculated from the content.
  """
  @spec record_projection_revision(String.t(), map()) ::
          {:ok, projection_revision()} | {:error, term()}
  def record_projection_revision(export_record_id, attrs)
      when is_binary(export_record_id) and is_map(attrs) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    projection_uri = Map.fetch!(attrs, :projection_uri)
    content_hash = Map.get(attrs, :content_hash) || content_hash(Map.get(attrs, :content, ""))
    source_object_links = Map.get(attrs, :source_object_links, [])
    lifecycle_state = Map.get(attrs, :lifecycle_state, "current")
    created_by = Map.get(attrs, :created_by)
    metadata = Map.get(attrs, :metadata, %{})
    revision_number = Map.get(attrs, :revision_number) || next_revision_number(export_record_id)

    id =
      Map.get(attrs, :id) ||
        deterministic_id("projection", [export_record_id, Integer.to_string(revision_number)])

    sql = """
    INSERT INTO projection_revisions (
      id, tenant_id, workspace_id, export_record_id, revision_number,
      content_hash, projection_uri, source_object_links, lifecycle_state,
      created_by, metadata
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
    ON CONFLICT(export_record_id, revision_number) DO UPDATE SET
      content_hash = excluded.content_hash,
      projection_uri = excluded.projection_uri,
      source_object_links = excluded.source_object_links,
      lifecycle_state = excluded.lifecycle_state,
      metadata = excluded.metadata
    """

    with :ok <-
           Store.raw_execute(sql, [
             id,
             tenant_id,
             workspace_id,
             export_record_id,
             revision_number,
             content_hash,
             projection_uri,
             Jason.encode!(source_object_links),
             lifecycle_state,
             created_by,
             Jason.encode!(metadata)
           ]) do
      get_projection_revision(id, tenant_id: tenant_id, workspace_id: workspace_id)
    end
  end

  @doc "Fetch a projection revision by id."
  @spec get_projection_revision(String.t(), keyword()) ::
          {:ok, projection_revision()} | {:error, :not_found}
  def get_projection_revision(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             projection_columns() <>
             " FROM projection_revisions WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_projection(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Return the latest projection revision for an export record."
  @spec latest_projection_revision(String.t()) ::
          {:ok, projection_revision()} | {:error, :not_found}
  def latest_projection_revision(export_record_id) when is_binary(export_record_id) do
    case Store.raw_query(
           "SELECT " <>
             projection_columns() <>
             " FROM projection_revisions WHERE export_record_id = ?1 ORDER BY revision_number DESC LIMIT 1",
           [export_record_id]
         ) do
      {:ok, [row]} -> {:ok, row_to_projection(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Record the result of checking links/backlinks/staleness for a projection."
  @spec record_link_health(map()) :: {:ok, link_health_record()} | {:error, term()}
  def record_link_health(attrs) when is_map(attrs) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    export_record_id = Map.get(attrs, :export_record_id)
    projection_revision_id = Map.get(attrs, :projection_revision_id)
    broken_links = Map.get(attrs, :broken_links, [])
    backlinks = Map.get(attrs, :backlinks, [])
    missing_references = Map.get(attrs, :missing_references, [])
    stale_references = Map.get(attrs, :stale_references, [])

    status =
      Map.get(attrs, :status) || health_status(broken_links, missing_references, stale_references)

    metadata = Map.get(attrs, :metadata, %{})

    id =
      Map.get(attrs, :id) ||
        deterministic_id("link_health", [
          export_record_id || "",
          projection_revision_id || "",
          status,
          Integer.to_string(System.unique_integer([:positive]))
        ])

    sql = """
    INSERT INTO link_health_records (
      id, tenant_id, workspace_id, export_record_id, projection_revision_id,
      status, broken_links, backlinks, missing_references, stale_references,
      metadata
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
    """

    with :ok <-
           Store.raw_execute(sql, [
             id,
             tenant_id,
             workspace_id,
             export_record_id,
             projection_revision_id,
             status,
             Jason.encode!(broken_links),
             Jason.encode!(backlinks),
             Jason.encode!(missing_references),
             Jason.encode!(stale_references),
             Jason.encode!(metadata)
           ]) do
      get_link_health(id, tenant_id: tenant_id, workspace_id: workspace_id)
    end
  end

  @doc "Fetch a link health record by id."
  @spec get_link_health(String.t(), keyword()) :: {:ok, link_health_record()} | {:error, :not_found}
  def get_link_health(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             link_health_columns() <>
             " FROM link_health_records WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_link_health(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Compare supplied content against the latest recorded projection hash.

  Returns `:fresh` if the content matches, `:stale` if it changed, and
  `:missing_projection` when no projection revision exists yet.
  """
  @spec detect_drift(String.t(), String.t()) :: {:ok, :fresh | :stale | :missing_projection}
  def detect_drift(export_record_id, current_content)
      when is_binary(export_record_id) and is_binary(current_content) do
    case latest_projection_revision(export_record_id) do
      {:ok, %{content_hash: hash}} ->
        if hash == content_hash(current_content), do: {:ok, :fresh}, else: {:ok, :stale}

      {:error, :not_found} ->
        {:ok, :missing_projection}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Mark projections that depend on a source object as stale.

  This is the first invalidation rule for markdown/HTML/app surfaces. It
  invalidates direct exports of the object and projection revisions whose
  `source_object_links` mention the object ID.
  """
  @spec invalidate_for_object(String.t(), String.t(), keyword()) ::
          {:ok, invalidation_result()} | {:error, term()}
  def invalidate_for_object(source_object_type, source_object_id, opts \\ [])
      when is_binary(source_object_type) and is_binary(source_object_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    reason = Keyword.get(opts, :reason, "source object changed")

    with {:ok, export_ids} <-
           export_ids_for_object(tenant_id, workspace_id, source_object_type, source_object_id),
         {:ok, projection_count} <-
           mark_exports_stale(export_ids, tenant_id, workspace_id),
         {:ok, link_health_count} <-
           record_stale_link_health(
             export_ids,
             tenant_id,
             workspace_id,
             source_object_type,
             source_object_id,
             reason
           ) do
      {:ok,
       %{
         workspace_id: workspace_id,
         source_object_type: source_object_type,
         source_object_id: source_object_id,
         export_record_ids: export_ids,
         projection_revision_count: projection_count,
         link_health_record_count: link_health_count
       }}
    end
  end

  @doc "Convenience invalidation rule for Node-backed projections."
  @spec invalidate_for_node(String.t(), keyword()) ::
          {:ok, invalidation_result()} | {:error, term()}
  def invalidate_for_node(node_id, opts \\ []) when is_binary(node_id) do
    invalidate_for_object("node", node_id, opts)
  end

  @doc """
  Capture a human or agent edit made through a projection surface.

  Normal content edits become Source Packages so they can flow through Signal,
  Claim, Fact, Memory, and retrieval later. Topology edits become pending
  topology change requests so a file or agent cannot silently mutate workspace
  structure.
  """
  @spec capture_edit(String.t(), map()) :: {:ok, edit_capture()} | {:error, term()}
  def capture_edit(export_record_id, %{content: content} = attrs)
      when is_binary(export_record_id) and is_binary(content) do
    with {:ok, export} <-
           get_export(
             export_record_id,
             tenant_id: Map.get(attrs, :tenant_id, Tenant.default_id()),
             workspace_id: Map.get(attrs, :workspace_id)
           ) do
      case edit_kind(attrs) do
        :topology_change_request ->
          capture_topology_edit(export, attrs)

        :source_package ->
          capture_source_edit(export, attrs)
      end
    end
  end

  defp capture_source_edit(export, attrs) do
    content = Map.fetch!(attrs, :content)

    source_opts = [
      tenant_id: export.tenant_id,
      workspace_id: export.workspace_id,
      actor_id: Map.get(attrs, :actor_id),
      created_by: Map.get(attrs, :actor_id),
      source_uri: Map.get(attrs, :source_uri, export.destination_uri),
      source_time: Map.get(attrs, :source_time),
      surface_type: export.surface_type,
      projection_uri: Map.get(attrs, :projection_uri, export.destination_uri),
      export_record_id: export.id,
      projection_revision_id: Map.get(attrs, :projection_revision_id),
      node_id: node_id_for_export(export),
      metadata:
        Map.merge(Map.get(attrs, :metadata, %{}), %{
          edit_capture: true,
          export_record_id: export.id,
          source_object_type: export.source_object_type,
          source_object_id: export.source_object_id,
          surface_type: export.surface_type
        })
    ]

    with {:ok, source_package} <- SourcePackageService.record_workspace_edit(content, source_opts),
         {:ok, intake_result} <- maybe_route_source_edit(content, export, attrs) do
      {:ok,
       %{
         kind: :source_package,
         export_record: export,
         source_package: source_package,
         intake_result: intake_result,
         topology_change_request: nil
       }}
    end
  end

  defp capture_topology_edit(export, attrs) do
    payload =
      Map.get(attrs, :proposed_payload) ||
        %{
          content: Map.fetch!(attrs, :content),
          export_record_id: export.id,
          source_object_type: export.source_object_type,
          source_object_id: export.source_object_id,
          destination_uri: export.destination_uri
        }

    with {:ok, request} <-
           WorkspaceTopology.propose_change(%{
             tenant_id: export.tenant_id,
             workspace_id: export.workspace_id,
             request_type: Map.get(attrs, :request_type, "projection_edit"),
             target_object_type:
               Map.get(attrs, :target_object_type, export.source_object_type || "projection"),
             target_object_id: Map.get(attrs, :target_object_id, export.source_object_id),
             proposed_payload: payload,
             reason: Map.get(attrs, :reason, "Projection edit requested a topology change."),
             requested_by: Map.get(attrs, :actor_id),
             audit_event_links: Map.get(attrs, :audit_event_links, [])
           }) do
      {:ok,
       %{
         kind: :topology_change_request,
         export_record: export,
         source_package: nil,
         intake_result: nil,
         topology_change_request: request
       }}
    end
  end

  defp maybe_route_source_edit(content, export, attrs) do
    if Map.get(attrs, :route, false) do
      Intake.process(content, intake_opts(export, attrs))
    else
      {:ok, nil}
    end
  rescue
    error -> {:error, {:route_failed, error}}
  catch
    :exit, reason -> {:error, {:route_failed, reason}}
  end

  defp intake_opts(export, attrs) do
    [
      workspace_id: export.workspace_id,
      node: Map.get(attrs, :node) || Map.get(attrs, :node_slug) || "inbox",
      title: Map.get(attrs, :title, "Workspace edit"),
      genre: Map.get(attrs, :genre, "workspace-edit"),
      type: :signal,
      actor_id: Map.get(attrs, :actor_id),
      source_type: "workspace_edit",
      source_class: "text",
      source_system: "workspace_export",
      source_uri: Map.get(attrs, :source_uri, export.destination_uri),
      metadata: %{
        export_record_id: export.id,
        projection_revision_id: Map.get(attrs, :projection_revision_id),
        source_object_type: export.source_object_type,
        source_object_id: export.source_object_id,
        routed_from_projection: true
      }
    ]
  end

  defp edit_kind(attrs) do
    case Map.get(attrs, :edit_type) || Map.get(attrs, :kind) do
      value when value in [:topology, :topology_change, :topology_change_request] ->
        :topology_change_request

      value when value in ["topology", "topology_change", "topology_change_request"] ->
        :topology_change_request

      _ ->
        :source_package
    end
  end

  defp node_id_for_export(%{source_object_type: "node", source_object_id: source_object_id}) do
    source_object_id
  end

  defp node_id_for_export(_export), do: nil

  defp next_revision_number(export_record_id) do
    case Store.raw_query(
           "SELECT COALESCE(MAX(revision_number), 0) + 1 FROM projection_revisions WHERE export_record_id = ?1",
           [export_record_id]
         ) do
      {:ok, [[number]]} when is_integer(number) -> number
      _ -> 1
    end
  end

  defp export_ids_for_object(tenant_id, workspace_id, source_object_type, source_object_id) do
    like_pattern = "%#{source_object_id}%"

    sql = """
    SELECT DISTINCT er.id
    FROM export_records er
    LEFT JOIN projection_revisions pr ON pr.export_record_id = er.id
    WHERE er.tenant_id = ?1
      AND er.workspace_id = ?2
      AND (
        (er.source_object_type = ?3 AND er.source_object_id = ?4)
        OR pr.source_object_links LIKE ?5
      )
    ORDER BY er.id
    """

    case Store.raw_query(sql, [
           tenant_id,
           workspace_id,
           source_object_type,
           source_object_id,
           like_pattern
         ]) do
      {:ok, rows} -> {:ok, Enum.map(rows, fn [id] -> id end)}
      other -> other
    end
  end

  defp mark_exports_stale([], _tenant_id, _workspace_id), do: {:ok, 0}

  defp mark_exports_stale(export_ids, tenant_id, workspace_id) do
    placeholders = placeholders(length(export_ids), 3)

    with :ok <-
           Store.raw_execute(
             "UPDATE export_records SET lifecycle_state = 'stale' WHERE tenant_id = ?1 AND workspace_id = ?2 AND id IN (#{placeholders})",
             [tenant_id, workspace_id | export_ids]
           ),
         {:ok, [[projection_count]]} <-
           Store.raw_query(
             "SELECT COUNT(*) FROM projection_revisions WHERE tenant_id = ?1 AND workspace_id = ?2 AND export_record_id IN (#{placeholders})",
             [tenant_id, workspace_id | export_ids]
           ),
         :ok <-
           Store.raw_execute(
             "UPDATE projection_revisions SET lifecycle_state = 'stale' WHERE tenant_id = ?1 AND workspace_id = ?2 AND export_record_id IN (#{placeholders})",
             [tenant_id, workspace_id | export_ids]
           ) do
      {:ok, projection_count}
    end
  end

  defp record_stale_link_health([], _tenant_id, _workspace_id, _type, _id, _reason), do: {:ok, 0}

  defp record_stale_link_health(
         export_ids,
         tenant_id,
         workspace_id,
         source_object_type,
         source_object_id,
         reason
       ) do
    stale_ref = "#{source_object_type}:#{source_object_id}"

    count =
      Enum.reduce(export_ids, 0, fn export_id, acc ->
        projection_id =
          case latest_projection_revision(export_id) do
            {:ok, projection} -> projection.id
            _ -> nil
          end

        case record_link_health(%{
               tenant_id: tenant_id,
               workspace_id: workspace_id,
               export_record_id: export_id,
               projection_revision_id: projection_id,
               status: "stale",
               stale_references: [stale_ref],
               metadata: %{reason: reason}
             }) do
          {:ok, _record} -> acc + 1
          _ -> acc
        end
      end)

    {:ok, count}
  end

  defp placeholders(count, start_index) do
    start_index..(start_index + count - 1)
    |> Enum.map_join(", ", &"?#{&1}")
  end

  defp health_status([], [], []), do: "healthy"
  defp health_status(_broken_links, _missing_references, _stale_references), do: "issues"

  defp content_hash(content) when is_binary(content) do
    :crypto.hash(:sha256, content)
    |> Base.encode16(case: :lower)
  end

  defp deterministic_id(prefix, parts) do
    hash =
      :crypto.hash(:sha256, Enum.join(parts, "|"))
      |> Base.encode16(case: :lower)
      |> binary_part(0, 24)

    "#{prefix}_#{hash}"
  end

  defp export_columns do
    "id, tenant_id, workspace_id, source_object_type, source_object_id, surface_type, destination_uri, object_version, policy_version, lifecycle_state, generated_by, generated_at, metadata"
  end

  defp projection_columns do
    "id, tenant_id, workspace_id, export_record_id, revision_number, content_hash, projection_uri, source_object_links, lifecycle_state, created_by, created_at, metadata"
  end

  defp link_health_columns do
    "id, tenant_id, workspace_id, export_record_id, projection_revision_id, status, broken_links, backlinks, missing_references, stale_references, checked_at, metadata"
  end

  defp row_to_export([
         id,
         tenant_id,
         workspace_id,
         source_object_type,
         source_object_id,
         surface_type,
         destination_uri,
         object_version,
         policy_version,
         lifecycle_state,
         generated_by,
         generated_at,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      source_object_type: source_object_type,
      source_object_id: source_object_id,
      surface_type: surface_type,
      destination_uri: destination_uri,
      object_version: object_version,
      policy_version: policy_version,
      lifecycle_state: lifecycle_state,
      generated_by: generated_by,
      generated_at: generated_at,
      metadata: decode_map(metadata)
    }
  end

  defp row_to_projection([
         id,
         tenant_id,
         workspace_id,
         export_record_id,
         revision_number,
         content_hash,
         projection_uri,
         source_object_links,
         lifecycle_state,
         created_by,
         created_at,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      export_record_id: export_record_id,
      revision_number: revision_number,
      content_hash: content_hash,
      projection_uri: projection_uri,
      source_object_links: decode_list(source_object_links),
      lifecycle_state: lifecycle_state,
      created_by: created_by,
      created_at: created_at,
      metadata: decode_map(metadata)
    }
  end

  defp row_to_link_health([
         id,
         tenant_id,
         workspace_id,
         export_record_id,
         projection_revision_id,
         status,
         broken_links,
         backlinks,
         missing_references,
         stale_references,
         checked_at,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      export_record_id: export_record_id,
      projection_revision_id: projection_revision_id,
      status: status,
      broken_links: decode_list(broken_links),
      backlinks: decode_list(backlinks),
      missing_references: decode_list(missing_references),
      stale_references: decode_list(stale_references),
      checked_at: checked_at,
      metadata: decode_map(metadata)
    }
  end

  defp workspace_clause(nil, _position), do: ""
  defp workspace_clause(_workspace_id, position), do: " AND workspace_id = ?#{position}"

  defp scoped_params(params, nil), do: params
  defp scoped_params(params, workspace_id), do: params ++ [workspace_id]

  defp decode_map(nil), do: %{}
  defp decode_map(""), do: %{}

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_list(decoded) -> decoded
      _ -> []
    end
  end
end
