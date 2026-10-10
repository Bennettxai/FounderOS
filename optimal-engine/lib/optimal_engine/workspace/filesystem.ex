defmodule OptimalEngine.Workspace.Filesystem do
  @moduledoc """
  Provisions the on-disk directory tree for a workspace.

  Layout (relative to the engine root):

      <root>/<workspace_slug>/
      ├── nodes/                  Tier 1 — raw signals, append-only
      ├── .wiki/                  Tier 3 — curated pages
      │   └── SCHEMA.md           governance rules the curator honors
      ├── assets/                 binary attachments, hash-addressed
      └── architectures/          user-defined data-point schemas

  The default workspace is special — it lives at the root itself
  (no `default/` prefix) for backwards-compat with the existing
  `nodes/`, `.wiki/`, `assets/` layout.

  This module is idempotent: calling `provision/2` on an existing
  workspace skips already-created paths. Safe to invoke on every
  workspace upsert.
  """

  require Logger

  @default_slug "default"

  @doc """
  Returns the absolute on-disk path for a workspace, given the engine root.
  Default workspace returns the root itself; others return `<root>/<slug>`.
  """
  @spec path(String.t(), String.t()) :: String.t()
  def path(root, slug) when is_binary(root) and is_binary(slug) do
    if slug == @default_slug do
      root
    else
      Path.join(root, slug)
    end
  end

  @doc """
  Provisions the directory tree for a workspace. Returns `{:ok, path}` with
  the absolute workspace path on success; `{:error, reason}` on failure.

  Idempotent — re-running on an existing workspace is a no-op.

  Options:
    * `:root` — engine root path. Defaults to `File.cwd!/0`.
    * `:write_schema` — whether to drop a starter `.wiki/SCHEMA.md`.
      Defaults to `true` for new workspaces, ignored for the default
      workspace (which has its own schema).
  """
  @spec provision(String.t(), keyword()) :: {:ok, String.t()} | {:error, term()}
  def provision(slug, opts \\ []) when is_binary(slug) do
    root = Keyword.get(opts, :root, File.cwd!())
    write_schema = Keyword.get(opts, :write_schema, true)
    workspace_path = path(root, slug)

    with :ok <- File.mkdir_p(workspace_path),
         :ok <- File.mkdir_p(Path.join(workspace_path, "nodes")),
         :ok <- File.mkdir_p(Path.join(workspace_path, ".wiki")),
         :ok <- File.mkdir_p(Path.join(workspace_path, "assets")),
         :ok <- File.mkdir_p(Path.join(workspace_path, "architectures")) do
      if write_schema and slug != @default_slug do
        maybe_write_schema(workspace_path, slug)
      end

      Logger.info("[Workspace.Filesystem] provisioned #{slug} at #{workspace_path}")
      {:ok, workspace_path}
    end
  end

  @doc """
  Returns the on-disk path for a wiki slug inside a workspace.
  e.g. `wiki_path(root, "engineering", "core-platform-architecture")`
  → `<root>/engineering/.wiki/core-platform-architecture.md`
  """
  @spec wiki_path(String.t(), String.t(), String.t()) :: String.t()
  def wiki_path(root, workspace_slug, slug) do
    Path.join([path(root, workspace_slug), ".wiki", "#{slug}.md"])
  end

  @doc """
  Returns the signals directory for a node inside a workspace.
  """
  @spec node_signals_path(String.t(), String.t(), String.t()) :: String.t()
  def node_signals_path(root, workspace_slug, node_slug) do
    Path.join([path(root, workspace_slug), "nodes", node_slug, "signals"])
  end

  # ── Helpers ─────────────────────────────────────────────────────────────

  defp maybe_write_schema(workspace_path, slug) do
    schema_path = Path.join([workspace_path, ".wiki", "SCHEMA.md"])

    if File.exists?(schema_path) do
      :ok
    else
      File.write(schema_path, default_schema(slug))
    end
  end

  defp default_schema(slug) do
    """
    # Wiki governance — workspace `#{slug}`

    The Optimal Engine curator reads this file and applies its rules
    when generating pages for this workspace. Edit freely.

    ## Required sections

    Every curated page must contain:

    - `## Summary` — one paragraph, ≤ 100 tokens.
    - `## Key points` — bullets, atomic claims, ≤ 2K tokens total.
    - `## Detail` — full prose; no length cap.

    ## Citation requirements

    - Every factual claim carries a `{{cite: optimal://...}}` directive.
    - Citations must resolve to a Tier-1 signal in this workspace.
    - Cross-workspace citations are rejected by the integrity gate.

    ## Audience policy

    Audiences derive from the workspace's audience registry. Default is
    `default`. Add audiences by writing them to the wiki page frontmatter
    `audience:` field; the curator will produce one variant per audience.

    ## Naming

    - Wiki slugs are lowercase-kebab-case and unique per audience.
    - Slugs should match the dominant entity or theme of the page.
    """
  end
end
