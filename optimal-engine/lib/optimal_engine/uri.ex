defmodule OptimalEngine.URI do
  @moduledoc """
  `optimal://` URI system for addressable context.

  Every piece of context in OptimalEngine is addressable via a URI in the form:

      optimal://{namespace}/{path}

  ## Namespaces

  | Namespace              | Maps to                          | Context type |
  |------------------------|----------------------------------|--------------|
  | `resources/`           | `docs/` folder                   | :resource    |
  | `user/memories/`       | `_memories/user/`                | :memory      |
  | `agent/memories/`      | `_memories/agent/`               | :memory      |
  | `agent/skills/`        | `_skills/`                       | :skill       |
  | `nodes/{id}/`          | `nodes/{id}/` workspace projection | :signal      |
  | `sessions/{id}/`       | `.system/sessions/{id}/`         | :memory      |
  | `inbox/`               | `inbox/`                         | :signal      |

  ## Operations

  - `parse/1`        — Parse an `optimal://` URI string into a structured map
  - `from_path/1`    — Build a URI from a filesystem path
  - `resolve/1`      — Convert URI → absolute filesystem path
  - `namespace/1`    — Return just the namespace atom from a URI
  - `ls/1`           — List contexts under a URI prefix
  """

  require Logger

  @type parsed :: %{
          namespace: atom(),
          segments: [String.t()],
          raw: String.t()
        }

  # Node folder -> node ID mapping. Current public examples use semantic node
  # IDs instead of old numbered folders.
  @node_folders %{
    "entity-company" => "entity-company",
    "person-operator" => "person-operator",
    "product-customer-portal" => "product-customer-portal",
    "project-platform-launch" => "project-platform-launch",
    "operation-weekly-review" => "operation-weekly-review",
    "operation-revenue" => "operation-revenue",
    "operation-delivery" => "operation-delivery",
    "learning-research-library" => "learning-research-library",
    "inbox" => "inbox",
    "team" => "team",
    "docs" => "resources"
  }

  # Node ID → folder name (reverse)
  @node_ids Map.new(@node_folders, fn {folder, id} -> {id, folder} end)

  @doc """
  Parses an `optimal://` URI string.

  Returns `{:ok, parsed}` on success, `{:error, reason}` if the URI is malformed.

  ## Examples

      iex> OptimalEngine.URI.parse("optimal://nodes/project-platform-launch/context.md")
      {:ok, %{namespace: :nodes, segments: ["project-platform-launch", "context.md"], raw: "optimal://nodes/project-platform-launch/context.md"}}

      iex> OptimalEngine.URI.parse("optimal://resources/api-docs.md")
      {:ok, %{namespace: :resources, segments: ["api-docs.md"], raw: "optimal://resources/api-docs.md"}}
  """
  @spec parse(String.t()) :: {:ok, parsed()} | {:error, term()}
  def parse("optimal://" <> rest) do
    parts = String.split(rest, "/", trim: true)

    case parts do
      [] ->
        {:error, :empty_uri}

      # Two-segment namespaces: user/memories, agent/memories, agent/skills
      ["user", "memories" | segments] ->
        {:ok, %{namespace: :user_memories, segments: segments, raw: "optimal://" <> rest}}

      ["agent", "memories" | segments] ->
        {:ok, %{namespace: :agent_memories, segments: segments, raw: "optimal://" <> rest}}

      ["agent", "skills" | segments] ->
        {:ok, %{namespace: :agent_skills, segments: segments, raw: "optimal://" <> rest}}

      [ns_str | segments] ->
        namespace = namespace_atom(ns_str)
        {:ok, %{namespace: namespace, segments: segments, raw: "optimal://" <> rest}}
    end
  end

  def parse(other), do: {:error, {:invalid_scheme, other}}

  @doc """
  Builds an `optimal://` URI from a filesystem path.

  Inspects the path relative to the root and determines the appropriate namespace.
  Returns a URI string.
  """
  @spec from_path(String.t()) :: String.t()
  def from_path(path) when is_binary(path) do
    root = Application.get_env(:optimal_engine, :root_path, "")
    relative = String.replace_prefix(path, root <> "/", "")
    parts = String.split(relative, "/", trim: true)

    case parts do
      [] ->
        "optimal://inbox/"

      [top | rest] ->
        build_uri(top, rest)
    end
  end

  @doc """
  Resolves an `optimal://` URI to an absolute filesystem path.

  Returns `{:ok, path}` if the URI maps to a valid location,
  `{:error, reason}` if the namespace is unknown or path cannot be resolved.
  """
  @spec resolve(String.t() | parsed()) :: {:ok, String.t()} | {:error, term()}
  def resolve(uri) when is_binary(uri) do
    case parse(uri) do
      {:ok, parsed} -> resolve(parsed)
      err -> err
    end
  end

  def resolve(%{namespace: ns, segments: segments}) do
    root = Application.get_env(:optimal_engine, :root_path, "")
    base = namespace_to_path(ns, segments, root)

    case base do
      nil -> {:error, {:unresolvable_namespace, ns}}
      path -> {:ok, path}
    end
  end

  @doc "Returns the namespace atom from a parsed or raw URI."
  @spec namespace(String.t() | parsed()) :: atom() | nil
  def namespace(%{namespace: ns}), do: ns

  def namespace(uri) when is_binary(uri) do
    case parse(uri) do
      {:ok, parsed} -> parsed.namespace
      _ -> nil
    end
  end

  @doc """
  Returns the context type implied by a URI namespace.
  """
  @spec context_type(String.t() | parsed()) :: OptimalEngine.Context.context_type()
  def context_type(%{namespace: :resources}), do: :resource
  def context_type(%{namespace: :user_memories}), do: :memory
  def context_type(%{namespace: :agent_memories}), do: :memory
  def context_type(%{namespace: :agent_skills}), do: :skill
  def context_type(%{namespace: :sessions}), do: :memory
  def context_type(%{namespace: :nodes}), do: :signal
  def context_type(%{namespace: :inbox}), do: :signal
  def context_type(%{namespace: _}), do: :resource

  def context_type(uri) when is_binary(uri) do
    case parse(uri) do
      {:ok, parsed} -> context_type(parsed)
      _ -> :resource
    end
  end

  @doc """
  Returns the node ID implied by a `optimal://nodes/{id}/` URI, or nil.
  """
  @spec node_id(String.t() | parsed()) :: String.t() | nil
  def node_id(%{namespace: :nodes, segments: [node_id | _]}), do: node_id
  def node_id(%{namespace: :inbox}), do: "inbox"
  def node_id(%{namespace: _}), do: nil

  def node_id(uri) when is_binary(uri) do
    case parse(uri) do
      {:ok, parsed} -> node_id(parsed)
      _ -> nil
    end
  end

  @doc """
  Returns all known node URIs as `optimal://nodes/{id}/` strings.
  """
  @spec all_node_uris() :: [String.t()]
  def all_node_uris do
    @node_ids
    |> Map.keys()
    |> Enum.map(fn node_id -> "optimal://nodes/#{node_id}/" end)
    |> Enum.sort()
  end

  @doc """
  Returns the folder name on disk for a node ID.

  ## Examples

      iex> OptimalEngine.URI.node_folder("project-platform-launch")
      "project-platform-launch"
  """
  @spec node_folder(String.t()) :: String.t() | nil
  def node_folder(node_id) when is_binary(node_id) do
    Map.get(@node_ids, node_id)
  end

  # --- Private ---

  defp build_uri("docs", rest) do
    path = Enum.join(rest, "/")
    "optimal://resources/#{path}"
  end

  defp build_uri(top, rest) when is_map_key(@node_folders, top) do
    node_id = Map.fetch!(@node_folders, top)
    path = Enum.join(rest, "/")

    if node_id == "inbox" do
      "optimal://inbox/#{path}"
    else
      "optimal://nodes/#{node_id}/#{path}"
    end
  end

  defp build_uri("_memories", ["user" | rest]) do
    path = Enum.join(rest, "/")
    "optimal://user/memories/#{path}"
  end

  defp build_uri("_memories", ["agent" | rest]) do
    path = Enum.join(rest, "/")
    "optimal://agent/memories/#{path}"
  end

  defp build_uri("_skills", rest) do
    path = Enum.join(rest, "/")
    "optimal://agent/skills/#{path}"
  end

  defp build_uri(".system", ["sessions" | rest]) do
    case rest do
      [session_id | file_rest] ->
        path = Enum.join(file_rest, "/")
        "optimal://sessions/#{session_id}/#{path}"

      [] ->
        "optimal://sessions/"
    end
  end

  defp build_uri(other, rest) do
    path = Enum.join([other | rest], "/")
    "optimal://inbox/#{path}"
  end

  defp namespace_atom("resources"), do: :resources
  defp namespace_atom("user"), do: :user_memories
  defp namespace_atom("agent"), do: :agent_memories
  defp namespace_atom("nodes"), do: :nodes
  defp namespace_atom("sessions"), do: :sessions
  defp namespace_atom("inbox"), do: :inbox
  defp namespace_atom(other), do: String.to_atom(other)

  defp namespace_to_path(:resources, segments, root) do
    Path.join([root, "docs"] ++ segments)
  end

  defp namespace_to_path(:user_memories, segments, root) do
    Path.join([root, "_memories", "user"] ++ segments)
  end

  defp namespace_to_path(:agent_memories, segments, root) do
    Path.join([root, "_memories", "agent"] ++ segments)
  end

  defp namespace_to_path(:agent_skills, segments, root) do
    Path.join([root, "_skills"] ++ segments)
  end

  defp namespace_to_path(:sessions, [session_id | rest], root) do
    Path.join([root, ".system", "sessions", session_id] ++ rest)
  end

  defp namespace_to_path(:sessions, [], root) do
    Path.join([root, ".system", "sessions"])
  end

  defp namespace_to_path(:nodes, [node_id | rest], root) do
    case Map.get(@node_ids, node_id) do
      nil -> nil
      folder -> Path.join([root, folder] ++ rest)
    end
  end

  defp namespace_to_path(:inbox, segments, root) do
    Path.join([root, "inbox"] ++ segments)
  end

  defp namespace_to_path(_, _segments, _root), do: nil
end
