defmodule OptimalEngine.Pipeline.Intake.Writer do
  @moduledoc """
  Writes classified signal files to disk with YAML frontmatter and genre skeletons.

  ## File format

  Every written file is a valid markdown signal that the indexer can parse:

      ---
      node: project-platform-launch
      title: Requirements Review
      signal:
        mode: linguistic
        genre: transcript
        type: decide
        format: markdown
        sn_ratio: 0.7
      tiers:
        l0: "TRANSCRIPT | project-platform-launch | Requirements Review [S/N: 0.7]"
        l1: "First 300 chars of content..."
      entities:
        - Alice
        - Alice
      routed_to:
        - project-platform-launch
        - operation-revenue
      created_at: "2026-03-18T14:30:00Z"
      ---

      # Requirements Review

      ## Participants
      ...

  ## File naming

  Files are written to `{root}/{workspace_dir}/{node_folder}/signals/{date}-{slug}.md`
  where slug is the title lowercased with spaces replaced by hyphens.

  `workspace_dir` follows `OptimalEngine.Workspace.Filesystem.path/2`: the
  default workspace lives at the root itself (no prefix — single-workspace
  behavior is preserved), every other workspace gets its own directory so
  two workspaces can never read or overwrite each other's signal files.

  Cross-references get the same content written to additional node folders
  with a `cross_ref_from:` note in the frontmatter.
  """

  require Logger

  alias OptimalEngine.Pipeline.Intake.Skeleton
  alias OptimalEngine.Signal
  alias OptimalEngine.Topology.Node, as: TopologyNode
  alias OptimalEngine.URI
  alias OptimalEngine.Workspace
  alias OptimalEngine.Workspace.Filesystem, as: WorkspaceFilesystem

  @doc """
  Writes a classified signal to the primary node's signals/ directory.

  Pass `:tenant_id`/`:workspace_id` in `opts` to resolve the Node folder
  through workspace topology (see `node_to_folder/2`).

  Returns `{:ok, absolute_path}` or `{:error, reason}`.
  """
  @spec write_signal(Signal.t(), keyword()) :: {:ok, String.t()} | {:error, term()}
  def write_signal(%Signal{} = signal, opts \\ []) do
    root = root_path()
    node_folder = node_to_folder(signal.node, opts)
    signals_dir = Path.join([root, node_folder, "signals"])

    filename = build_filename(signal)
    path = Path.join(signals_dir, filename)

    content = build_file_content(signal, opts)

    with :ok <- File.mkdir_p(signals_dir),
         :ok <- File.write(path, content) do
      Logger.debug("[Writer] Wrote signal: #{path}")
      {:ok, path}
    end
  end

  @doc """
  Writes cross-reference copies of a signal to additional node folders.

  Each cross-reference file is identical to the primary but includes a
  `cross_ref_from:` field in the frontmatter indicating the origin node.

  Returns `{:ok, [path]}` — all successfully written paths.
  """
  @spec write_cross_references(Signal.t(), [String.t()], keyword()) :: {:ok, [String.t()]}
  def write_cross_references(%Signal{} = signal, additional_nodes, opts \\ [])
      when is_list(additional_nodes) do
    primary_folder = node_to_folder(signal.node, opts)

    paths =
      additional_nodes
      |> Enum.reject(fn node ->
        # Skip if it resolves to the same folder as the primary
        node_to_folder(node, opts) == primary_folder
      end)
      |> Enum.flat_map(fn node ->
        cross_opts = Keyword.put(opts, :cross_ref_from, signal.node)

        case write_signal_to_node(signal, node, cross_opts) do
          {:ok, path} ->
            [path]

          {:error, reason} ->
            Logger.warning("[Writer] Cross-ref to #{node} failed: #{inspect(reason)}")
            []
        end
      end)

    {:ok, paths}
  end

  @doc """
  Updates a node's context.md with new persistent facts.

  Appends the facts under a dated section if context.md exists, or
  creates a minimal context.md if the file is absent.
  """
  @spec update_context(String.t(), [String.t()], keyword()) :: :ok | {:error, term()}
  def update_context(node, facts, opts \\ []) when is_binary(node) and is_list(facts) do
    root = root_path()
    node_folder = node_to_folder(node, opts)
    context_path = Path.join([root, node_folder, "context.md"])
    date_str = Date.utc_today() |> Date.to_iso8601()

    fact_block = Enum.map_join(facts, "\n", fn f -> "- #{f}" end)

    section = "\n\n## Facts Updated #{date_str}\n\n#{fact_block}\n"

    case File.read(context_path) do
      {:ok, existing} ->
        File.write(context_path, existing <> section)

      {:error, :enoent} ->
        initial = "# #{humanize_node(node)}\n#{section}"

        with :ok <- File.mkdir_p(Path.dirname(context_path)) do
          File.write(context_path, initial)
        end

      {:error, _} = err ->
        err
    end
  end

  @doc """
  Returns the relative path from OptimalOS root that would be written for a signal.
  Useful for display in CLI output without computing absolute paths.
  """
  @spec relative_path(Signal.t(), keyword()) :: String.t()
  def relative_path(%Signal{} = signal, opts \\ []) do
    node_folder = node_to_folder(signal.node, opts)
    filename = build_filename(signal)
    Path.join([node_folder, "signals", filename])
  end

  @doc """
  Returns the optimal:// URI that corresponds to a written signal path.
  """
  @spec signal_uri(Signal.t(), keyword()) :: String.t()
  def signal_uri(%Signal{} = signal, opts \\ []) do
    node_folder = node_to_folder(signal.node, opts)
    filename = build_filename(signal)
    fs_path = Path.join([root_path(), node_folder, "signals", filename])
    URI.from_path(fs_path)
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp write_signal_to_node(%Signal{} = signal, node, opts) do
    root = root_path()
    node_folder = node_to_folder(node, opts)
    signals_dir = Path.join([root, node_folder, "signals"])
    filename = build_filename(signal)
    path = Path.join(signals_dir, filename)

    content = build_file_content(%{signal | node: node}, opts)

    with :ok <- File.mkdir_p(signals_dir),
         :ok <- File.write(path, content) do
      {:ok, path}
    end
  end

  defp build_filename(%Signal{} = signal) do
    date = date_from_signal(signal)
    slug = slugify(signal.title || "untitled")
    "#{date}-#{slug}.md"
  end

  defp date_from_signal(%Signal{created_at: %DateTime{} = dt}) do
    dt |> DateTime.to_date() |> Date.to_iso8601()
  end

  defp date_from_signal(_), do: Date.utc_today() |> Date.to_iso8601()

  defp slugify(title) do
    title
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9\s-]/, "")
    |> String.replace(~r/\s+/, "-")
    |> String.replace(~r/-{2,}/, "-")
    |> String.trim("-")
    |> String.slice(0, 60)
  end

  defp build_file_content(%Signal{} = signal, opts) do
    cross_ref_from = Keyword.get(opts, :cross_ref_from)

    frontmatter = build_frontmatter(signal, cross_ref_from)
    body = build_body(signal)

    "#{frontmatter}\n#{body}"
  end

  defp build_frontmatter(%Signal{} = signal, cross_ref_from) do
    date_str = date_from_signal(signal)
    now_str = DateTime.to_iso8601(signal.created_at || DateTime.utc_now())

    entities_yaml =
      case signal.entities do
        [] -> "  []\n"
        list -> Enum.map_join(list, "", fn e -> "  - #{e}\n" end)
      end

    routed_yaml =
      case signal.routed_to do
        [] -> "  []\n"
        list -> Enum.map_join(list, "", fn r -> "  - #{r}\n" end)
      end

    cross_ref_line =
      if cross_ref_from do
        "cross_ref_from: #{cross_ref_from}\n"
      else
        ""
      end

    l0 = signal.l0_summary || ""

    l1_preview =
      (signal.l1_description || "") |> String.slice(0, 300) |> String.replace("\n", " ")

    """
    ---
    node: #{signal.node || "inbox"}
    title: #{signal.title || "Untitled"}
    date: #{date_str}
    #{cross_ref_line}signal:
      mode: #{signal.mode || :linguistic}
      genre: #{signal.genre || "note"}
      type: #{signal.type || :inform}
      format: #{signal.format || :markdown}
      sn_ratio: #{Float.round(signal.sn_ratio || 0.6, 2)}
    tiers:
      l0: "#{escape_yaml(l0)}"
      l1: "#{escape_yaml(l1_preview)}"
    entities:
    #{entities_yaml}routed_to:
    #{routed_yaml}created_at: "#{now_str}"
    ---
    """
  end

  defp build_body(%Signal{} = signal) do
    title_line = "# #{signal.title || "Untitled"}\n\n"
    genre = signal.genre || "note"
    raw_content = extract_body_content(signal.content || "")

    skeleton_body = Skeleton.apply_skeleton(genre, raw_content)
    title_line <> skeleton_body
  end

  # Extract body from content — strip existing frontmatter if present
  defp extract_body_content(content) do
    case Regex.run(~r/\A---\r?\n.*?\r?\n---\r?\n?(.*)\z/s, content) do
      [_, body] -> String.trim(body)
      _ -> String.trim(content)
    end
  end

  defp escape_yaml(str) do
    str
    |> String.replace("\"", "'")
    |> String.replace("\n", " ")
  end

  # ---------------------------------------------------------------------------
  # Node/folder resolution
  #
  # Workspace topology (`OptimalEngine.Topology.Node`) is the source of truth
  # for Node folders: when a tenant/workspace scope is given and the slug has a
  # topology row, the file lands in that Node's own folder (its `path`). The
  # legacy alias map below is ONLY the final fallback for pre-topology slugs;
  # unknown slugs without a topology row still land in inbox so evidence is
  # never dropped.
  # ---------------------------------------------------------------------------

  # Kept in sync with `Workspace.default_id/0` / `Workspace.Filesystem` —
  # the default workspace lives at the engine root itself.
  @default_workspace_id "default"

  @legacy_node_folder_map %{
    "operator" => "person-operator",
    "entity-company" => "entity-company",
    "person-operator" => "person-operator",
    "product-customer-portal" => "product-customer-portal",
    "project-platform-launch" => "project-platform-launch",
    "operation-weekly-review" => "operation-weekly-review",
    "operation-revenue" => "operation-revenue",
    "operation-delivery" => "operation-delivery",
    "learning-research-library" => "learning-research-library",
    "team" => "team",
    "inbox" => "inbox"
  }

  @doc """
  Resolves a node slug to its folder relative to the engine root.

  Resolution order:

  1. Workspace topology Node lookup (requires `:workspace_id` in `opts`,
     optionally `:tenant_id`) — a topology Node routes to its own `path`
  2. Legacy alias map (e.g. `"operator"` → `"person-operator"`)
  3. `"inbox"` — the routing fallback; evidence is preserved, never dropped

  Every resolved folder (including the legacy-alias and inbox fallbacks) is
  prefixed with the workspace directory for non-default workspaces, so
  on-disk signal storage is segregated per workspace. The default workspace
  keeps the unprefixed single-workspace layout.
  """
  @spec node_to_folder(String.t() | nil, keyword()) :: String.t()
  def node_to_folder(node, opts \\ [])
  def node_to_folder(nil, opts), do: scoped_folder("inbox", opts)

  def node_to_folder(node, opts) when is_binary(node) do
    folder =
      case topology_folder(node, opts) do
        {:ok, folder} -> folder
        :miss -> Map.get(@legacy_node_folder_map, node, "inbox")
      end

    scoped_folder(folder, opts)
  end

  defp topology_folder(_slug, []), do: :miss

  defp topology_folder(slug, opts) do
    workspace_id = Keyword.get(opts, :workspace_id)

    if is_binary(workspace_id) and workspace_id != "" do
      lookup_opts =
        case Keyword.get(opts, :tenant_id) do
          nil -> [workspace_id: workspace_id]
          tenant_id -> [tenant_id: tenant_id, workspace_id: workspace_id]
        end

      case TopologyNode.get_by_slug(slug, lookup_opts) do
        {:ok, %TopologyNode{} = node} -> {:ok, node_folder(node)}
        _ -> :miss
      end
    else
      :miss
    end
  rescue
    _ -> :miss
  catch
    :exit, _ -> :miss
  end

  defp node_folder(%TopologyNode{path: path}) when is_binary(path) and path != "", do: path
  defp node_folder(%TopologyNode{slug: slug}), do: "nodes/#{slug}"

  # Prefixes a node folder with the workspace directory so non-default
  # workspaces never share (or clobber) each other's on-disk signal tree.
  # Mirrors `Workspace.Filesystem.path/2`: the default workspace lives at
  # the root itself, every other workspace under `<root>/<workspace_dir>/`.
  defp scoped_folder(folder, opts) do
    case workspace_dir(opts) do
      nil ->
        folder

      dir ->
        # A topology Node path may already carry the workspace prefix —
        # never double-prefix.
        if folder == dir or String.starts_with?(folder, dir <> "/") do
          folder
        else
          Path.join(dir, folder)
        end
    end
  end

  defp workspace_dir(opts) when is_list(opts) do
    case Keyword.get(opts, :workspace_id) do
      nil -> nil
      "" -> nil
      @default_workspace_id -> nil
      workspace_id when is_binary(workspace_id) -> workspace_prefix(workspace_slug(workspace_id))
      _ -> nil
    end
  end

  # Registered workspaces route through the slug directory provisioned by
  # `Workspace.Filesystem.provision/2`. Unregistered workspace ids (fail
  # closed: still segregated, never the shared root) get a deterministic
  # sanitized directory.
  defp workspace_slug(workspace_id) do
    case Workspace.get(workspace_id) do
      {:ok, %Workspace{slug: slug}} when is_binary(slug) and slug != "" -> slug
      _ -> sanitize_workspace_dir(workspace_id)
    end
  rescue
    _ -> sanitize_workspace_dir(workspace_id)
  catch
    :exit, _ -> sanitize_workspace_dir(workspace_id)
  end

  defp workspace_prefix(slug) do
    root = root_path()

    case WorkspaceFilesystem.path(root, slug) do
      ^root -> nil
      workspace_path -> Path.relative_to(workspace_path, root)
    end
  end

  defp sanitize_workspace_dir(workspace_id) do
    sanitized =
      workspace_id
      |> String.downcase()
      |> String.replace(~r/[^a-z0-9_-]+/, "-")
      |> String.replace(~r/-{2,}/, "-")
      |> String.trim("-")

    if sanitized == "" do
      "ws-" <>
        Base.encode16(binary_part(:crypto.hash(:sha256, workspace_id), 0, 6), case: :lower)
    else
      sanitized
    end
  end

  defp humanize_node(node) do
    node
    |> String.replace("-", " ")
    |> String.split()
    |> Enum.map_join(" ", &String.capitalize/1)
  end

  defp root_path do
    Application.get_env(:optimal_engine, :root_path, File.cwd!())
  end
end
