defmodule OptimalEngine.Profile do
  @moduledoc """
  4-tier workspace snapshot — "what is this workspace about?" in one call.

  Tiers:

    1. **static**   — concatenated `context.md` per node (persistent ground truth)
    2. **dynamic**  — concatenated `signal.md` per node (rolling weekly status)
    3. **curated**  — most-recently-curated wiki page headline (Tier 3)
    4. **activity** — top-N recent chunks + top entities by reference count

  ## Bandwidth

    * `:l0`   — static headline (≤200 chars) + curated headline (≤200 chars), no activity
    * `:l1`   — all four tiers, each truncated to a short excerpt (default)
    * `:full` — no truncation

  ## Usage

      iex> OptimalEngine.Profile.get("default")
      {:ok, %OptimalEngine.Profile{workspace_id: "default", ...}}

      iex> OptimalEngine.Profile.get("engineering", audience: "sales", bandwidth: :l0)
      {:ok, %OptimalEngine.Profile{...}}
  """

  require Logger

  alias OptimalEngine.{Store, Wiki, Workspace}
  alias OptimalEngine.Workspace.Filesystem

  @type bandwidth :: :l0 | :l1 | :full

  @type t :: %__MODULE__{
          workspace_id: String.t(),
          tenant_id: String.t(),
          audience: String.t(),
          static: String.t(),
          dynamic: String.t(),
          curated: String.t(),
          activity: [map()],
          entities: [map()],
          generated_at: String.t()
        }

  defstruct workspace_id: "default",
            tenant_id: "default",
            audience: "default",
            static: "",
            dynamic: "",
            curated: "",
            activity: [],
            entities: [],
            generated_at: nil

  @doc """
  Build a 4-tier profile for the given workspace.

  Options:
    * `:audience`      — audience tag for wiki variant lookup (default: `"default"`)
    * `:bandwidth`     — `:l0 | :l1 | :full` (default: `:l1`)
    * `:node_filter`   — if given, limit Tier 1/2 to this single node slug
    * `:recent_limit`  — max recent chunks returned in Tier 4 (default: 20)
    * `:entity_limit`  — max entities by ref count returned in Tier 4 (default: 15)
    * `:tenant_id`     — tenant scope (default: `"default"`)
  """
  @spec get(String.t(), keyword()) :: {:ok, t()} | {:error, term()}
  def get(workspace_id, opts \\ []) when is_binary(workspace_id) do
    audience = Keyword.get(opts, :audience, "default")
    bandwidth = Keyword.get(opts, :bandwidth, :l1)
    node_filter = Keyword.get(opts, :node_filter)
    recent_limit = Keyword.get(opts, :recent_limit, 20)
    entity_limit = Keyword.get(opts, :entity_limit, 15)
    tenant_id = Keyword.get(opts, :tenant_id, "default")

    root = workspace_root()

    # Resolve workspace to its slug (needed for Filesystem.path/2).
    with {:ok, workspace} <- resolve_workspace(workspace_id, tenant_id) do
      ws_path = Filesystem.path(root, workspace.slug)

      static = build_static(ws_path, node_filter)
      dynamic = build_dynamic(ws_path, node_filter)
      curated = build_curated(tenant_id, workspace_id, audience)
      {activity, entities} = build_activity(workspace_id, tenant_id, recent_limit, entity_limit)

      profile = %__MODULE__{
        workspace_id: workspace_id,
        tenant_id: tenant_id,
        audience: audience,
        static: apply_bandwidth_text(static, bandwidth, :l1_static),
        dynamic: apply_bandwidth_text(dynamic, bandwidth, :l1_dynamic),
        curated: apply_bandwidth_text(curated, bandwidth, :l1_curated),
        activity: if(bandwidth == :l0, do: [], else: activity),
        entities: if(bandwidth == :l0, do: [], else: entities),
        generated_at: DateTime.utc_now() |> DateTime.to_iso8601()
      }

      {:ok, profile}
    end
  end

  # ── Tier 1: static (context.md per node) ─────────────────────────────────

  defp build_static(ws_path, node_filter) do
    nodes_path = Path.join(ws_path, "nodes")
    read_node_files(nodes_path, "context.md", node_filter)
  end

  # ── Tier 2: dynamic (signal.md per node) ──────────────────────────────────

  defp build_dynamic(ws_path, node_filter) do
    nodes_path = Path.join(ws_path, "nodes")
    read_node_files(nodes_path, "signal.md", node_filter)
  end

  # Walk <workspace>/nodes/*/context.md (or signal.md).
  # Prefix each section with a node slug header for attribution.
  defp read_node_files(nodes_path, filename, node_filter) do
    case File.ls(nodes_path) do
      {:ok, entries} ->
        entries
        |> Enum.filter(&node_filter_matches?(&1, node_filter))
        |> Enum.sort()
        |> Enum.map(fn node_slug ->
          node_dir = Path.join(nodes_path, node_slug)
          file_path = Path.join(node_dir, filename)

          if File.dir?(node_dir) do
            case safe_read(file_path) do
              "" -> nil
              content -> "## #{node_slug}\n\n#{String.trim(content)}"
            end
          else
            nil
          end
        end)
        |> Enum.reject(&is_nil/1)
        |> Enum.join("\n\n")

      {:error, _reason} ->
        ""
    end
  end

  defp node_filter_matches?(_slug, nil), do: true
  defp node_filter_matches?(slug, filter), do: slug == filter

  # Read a file safely — returns "" on missing or unreadable.
  defp safe_read(path) do
    case File.read(path) do
      {:ok, content} -> content
      {:error, _} -> ""
    end
  end

  # ── Tier 3: curated (wiki page headline) ─────────────────────────────────

  defp build_curated(tenant_id, workspace_id, audience) do
    case Wiki.list(tenant_id, workspace_id) do
      {:ok, []} ->
        ""

      {:ok, pages} ->
        # Prefer pages matching the requested audience; fallback to default.
        audience_pages =
          Enum.filter(pages, &(&1.audience == audience))

        pages_to_scan =
          if audience_pages == [] do
            Enum.filter(pages, &(&1.audience == "default"))
          else
            audience_pages
          end

        # Pick the most recently curated page.
        # last_curated is an ISO8601 / SQLite datetime string — lexicographic
        # descending sort works correctly for these formats.
        best =
          pages_to_scan
          |> Enum.sort_by(&(&1.last_curated || ""), :desc)
          |> List.first()

        if best do
          String.slice(best.body || "", 0, 500)
        else
          ""
        end

      {:error, _} ->
        ""
    end
  end

  # ── Tier 4: activity + entities ───────────────────────────────────────────

  defp build_activity(workspace_id, tenant_id, recent_limit, entity_limit) do
    chunks = fetch_recent_chunks(workspace_id, tenant_id, recent_limit)
    entities = fetch_top_entities(workspace_id, tenant_id, entity_limit)
    {chunks, entities}
  end

  defp fetch_recent_chunks(workspace_id, tenant_id, limit) do
    sql = """
    SELECT id, signal_id, scale, text
    FROM chunks
    WHERE workspace_id = ?1 AND tenant_id = ?2
    ORDER BY created_at DESC
    LIMIT ?3
    """

    case Store.raw_query(sql, [workspace_id, tenant_id, limit]) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, signal_id, scale, text] ->
          %{id: id, signal_id: signal_id, scale: scale, text: text}
        end)

      {:error, reason} ->
        Logger.debug("[Profile] Chunk query failed for #{workspace_id}: #{inspect(reason)}")
        []
    end
  end

  defp fetch_top_entities(workspace_id, tenant_id, limit) do
    sql = """
    SELECT name, type, COUNT(*) AS refs
    FROM entities
    WHERE workspace_id = ?1 AND tenant_id = ?2
    GROUP BY name, type
    ORDER BY refs DESC
    LIMIT ?3
    """

    case Store.raw_query(sql, [workspace_id, tenant_id, limit]) do
      {:ok, rows} ->
        Enum.map(rows, fn [name, type, refs] ->
          %{name: name, type: type, refs: refs}
        end)

      {:error, reason} ->
        Logger.debug("[Profile] Entity query failed for #{workspace_id}: #{inspect(reason)}")
        []
    end
  end

  # ── Bandwidth filtering ───────────────────────────────────────────────────

  # l0: static headline ≤200 chars, curated ≤200 chars, no dynamic/activity
  # l1: all tiers, each capped
  # full: no truncation

  @l0_static_limit 200
  @l0_curated_limit 200
  @l1_limit 800

  defp apply_bandwidth_text(text, :l0, :l1_static), do: truncate(text, @l0_static_limit)
  defp apply_bandwidth_text(_text, :l0, :l1_dynamic), do: ""
  defp apply_bandwidth_text(text, :l0, :l1_curated), do: truncate(text, @l0_curated_limit)
  defp apply_bandwidth_text(text, :l1, _kind), do: truncate(text, @l1_limit)
  defp apply_bandwidth_text(text, :full, _kind), do: text

  defp truncate(text, max) when byte_size(text) <= max, do: text

  defp truncate(text, max) do
    # Safe binary slice — respect UTF-8 boundaries by using String.slice/2
    String.slice(text, 0, max)
  end

  # ── Workspace resolution ──────────────────────────────────────────────────

  # Lookup by id first; if the table doesn't have it, try slug lookup via
  # the default tenant. This lets callers pass either the DB id ("default",
  # "default:engineering") or the human slug ("engineering").
  defp resolve_workspace(workspace_id, tenant_id) do
    case Workspace.get(workspace_id) do
      {:ok, _ws} = ok ->
        ok

      {:error, :not_found} ->
        # Try slug lookup — caller may have passed a bare slug like "engineering"
        Workspace.get_by_slug(workspace_id, tenant_id)

      other ->
        other
    end
  end

  # ── Helpers ───────────────────────────────────────────────────────────────

  defp workspace_root do
    Application.get_env(:optimal_engine, :workspace_root, File.cwd!())
  end
end
