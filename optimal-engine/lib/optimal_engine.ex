defmodule OptimalEngine do
  @moduledoc """
  Optimal Context Engine — universal context ingestion + Signal Theory classification.

  Every piece of context is a `Context` classified on type (resource/memory/skill/signal)
  with optional full Signal Theory dimensions: `S = (Mode, Genre, Type, Format, Structure)`.

  ## URI Scheme

      optimal://resources/{path}       — static knowledge
      optimal://user/memories/{path}   — user-learned facts
      optimal://agent/skills/{path}    — callable tools
      optimal://agent/memories/{path}  — agent-learned patterns
      optimal://nodes/{node-id}/       — organizational nodes (12 folders)
      optimal://sessions/{id}/         — conversation history
      optimal://inbox/                 — unrouted content

  ## Public API

  ### Indexing
      OptimalEngine.index()                         # full reindex (all file types)
      OptimalEngine.index_file(path)                # single file

  ### Search
      OptimalEngine.search("query")
      OptimalEngine.search("query", type: :signal, node: "operator", limit: 5)
      OptimalEngine.search("query", type: :resource)
      OptimalEngine.search("query", uri: "optimal://nodes/project-platform-launch/")

  ### Context Tiers
      OptimalEngine.l0()                            # always-loaded ~2K token context
      OptimalEngine.stats()                         # store statistics

  ### Ingestion (classify + route + store)
      OptimalEngine.ingest(text)
      OptimalEngine.ingest(text, type: :memory, path: "optimal://user/memories/note.md")

  ### Composition (signal only)
      OptimalEngine.render_for(context_or_signal, "sales-lead")

  ### Sessions
      {:ok, session_id} = OptimalEngine.start_session()
      OptimalEngine.add_message(session_id, :user, "Customer called about pricing")
      {:ok, summary} = OptimalEngine.commit_session(session_id)

  ### URI Operations
      OptimalEngine.ls("optimal://nodes/project-platform-launch/")
      OptimalEngine.uri_for(path)
      OptimalEngine.resolve_uri("optimal://nodes/operator/signal.md")
  """

  alias OptimalEngine.Context
  alias OptimalEngine.Session
  alias OptimalEngine.Signal
  alias OptimalEngine.Store
  alias OptimalEngine.Routing
  alias OptimalEngine.URI
  alias OptimalEngine.Pipeline.Classifier
  alias OptimalEngine.Pipeline.Indexer
  alias OptimalEngine.Retrieval.Composer
  alias OptimalEngine.Retrieval.L0Cache
  alias OptimalEngine.Retrieval.Search, as: SearchEngine

  # ---------------------------------------------------------------------------
  # Indexing
  # ---------------------------------------------------------------------------

  @doc "Triggers a full reindex of the OptimalOS directory (all file types)."
  @spec index() :: {:ok, :started} | {:error, :already_running}
  def index, do: Indexer.full_index()

  @doc "Indexes a single file (any supported type)."
  @spec index_file(String.t()) :: {:ok, Context.t()} | {:error, term()}
  def index_file(path), do: Indexer.index_file(path)

  # ---------------------------------------------------------------------------
  # Search
  # ---------------------------------------------------------------------------

  @doc """
  Searches the context store with hybrid BM25 + temporal + S/N scoring.

  Returns `{:ok, [%Context{}]}`.

  ## Options
  - `:type`      — `:signal`, `:resource`, `:memory`, or `:skill`
  - `:node`      — filter by node ID
  - `:genre`     — filter by genre (signals only)
  - `:uri`       — scope to a URI prefix
  - `:limit`     — max results (default 10)
  - `:min_score` — minimum score threshold
  """
  @spec search(String.t(), keyword()) :: {:ok, [Context.t()]} | {:error, term()}
  def search(query, opts \\ []), do: SearchEngine.search(query, opts)

  # ---------------------------------------------------------------------------
  # Context Tiers
  # ---------------------------------------------------------------------------

  @doc "Returns the current L0 context string (~2K tokens, always loaded)."
  @spec l0() :: String.t()
  def l0, do: L0Cache.get()

  @doc "Returns store statistics."
  @spec stats() :: {:ok, map()} | {:error, term()}
  def stats, do: Store.stats()

  # ---------------------------------------------------------------------------
  # Ingestion
  # ---------------------------------------------------------------------------

  @doc """
  Classifies, routes, and stores raw content as a Context.

  Supports explicit type override:
      OptimalEngine.ingest("Customer called about pricing")           # auto-detected
      OptimalEngine.ingest("api docs text", type: :resource)   # force resource
      OptimalEngine.ingest("I learned X", type: :memory)       # force memory

  Returns the stored Context struct.
  """
  @spec ingest(String.t(), keyword()) :: {:ok, Context.t()} | {:error, term()}
  def ingest(text, opts \\ []) when is_binary(text) do
    path = Keyword.get(opts, :path, "_ingest/#{generate_id()}.md")
    forced_type = Keyword.get(opts, :type)

    topology = load_topology()
    known_entities = topology_entities(topology)

    classify_opts =
      [path: path, known_entities: known_entities]
      |> then(fn o -> if forced_type, do: Keyword.put(o, :type, forced_type), else: o end)

    ctx = Classifier.classify_context(text, classify_opts)
    now = DateTime.utc_now()

    id =
      :crypto.hash(:sha256, path <> DateTime.to_iso8601(now))
      |> Base.encode16(case: :lower)
      |> String.slice(0, 32)

    uri = URI.from_path(path)

    routed_to =
      if ctx.type == :signal && ctx.signal do
        route_signal(%{ctx.signal | path: path, node: ctx.node})
      else
        [ctx.node || "inbox"]
      end

    updated_signal =
      if ctx.signal do
        %{
          ctx.signal
          | id: id,
            path: path,
            created_at: now,
            modified_at: now,
            routed_to: routed_to
        }
      else
        nil
      end

    complete_ctx = %{
      ctx
      | id: id,
        uri: uri,
        path: path,
        created_at: now,
        modified_at: now,
        routed_to: routed_to,
        signal: updated_signal
    }

    case Store.insert_context(complete_ctx) do
      :ok -> {:ok, complete_ctx}
      {:error, _} = err -> err
    end
  end

  # ---------------------------------------------------------------------------
  # Composition
  # ---------------------------------------------------------------------------

  @doc """
  Renders a context (or signal) for a specific receiver.
  Receiver ID must match a key in topology.yaml endpoints.
  """
  @spec render_for(Context.t() | Signal.t(), String.t()) :: {:ok, String.t()} | {:error, term()}
  def render_for(context_or_signal, receiver_id) do
    signal =
      case context_or_signal do
        %Context{} = ctx -> Context.to_signal(ctx)
        %Signal{} = sig -> sig
      end

    case load_topology() do
      %{} = topology -> Composer.render_for(signal, receiver_id, topology)
    end
  end

  # ---------------------------------------------------------------------------
  # Sessions
  # ---------------------------------------------------------------------------

  @doc "Starts a new session. Returns `{:ok, session_id}`."
  @spec start_session(keyword()) :: {:ok, String.t()} | {:error, term()}
  def start_session(opts \\ []), do: Session.start_session(opts)

  @doc "Adds a message to a session."
  @spec add_message(String.t(), :user | :assistant | :system, String.t()) ::
          :ok | {:error, term()}
  def add_message(session_id, role, content),
    do: Session.add_message(session_id, role, content)

  @doc "Commits a session. Returns `{:ok, summary}`."
  @spec commit_session(String.t()) :: {:ok, String.t()} | {:error, term()}
  def commit_session(session_id), do: Session.commit(session_id)

  @doc "Returns the context string for a session."
  @spec session_context(String.t()) :: String.t()
  def session_context(session_id), do: Session.get_context(session_id)

  # ---------------------------------------------------------------------------
  # URI Operations
  # ---------------------------------------------------------------------------

  @doc "Builds an `optimal://` URI from a filesystem path."
  @spec uri_for(String.t()) :: String.t()
  def uri_for(path), do: URI.from_path(path)

  @doc "Resolves an `optimal://` URI to a filesystem path."
  @spec resolve_uri(String.t()) :: {:ok, String.t()} | {:error, term()}
  def resolve_uri(uri), do: URI.resolve(uri)

  @doc """
  Lists contexts under a URI prefix. Returns `{:ok, [%Context{}]}`.

  ## Example
      OptimalEngine.ls("optimal://nodes/project-platform-launch/")
  """
  @spec ls(String.t(), keyword()) :: {:ok, [Context.t()]} | {:error, term()}
  def ls(uri_prefix, opts \\ []) do
    with {:ok, parsed} <- URI.parse(uri_prefix) do
      node = URI.node_id(parsed)
      type = URI.context_type(parsed)
      limit = Keyword.get(opts, :limit, 50)
      ls_by_node(node, type, limit)
    end
  end

  defp ls_by_node(nil, _type, _limit), do: {:ok, []}

  defp ls_by_node(node, type, limit) do
    case Store.get_by_node(node, type: type) do
      {:ok, contexts} -> {:ok, Enum.take(contexts, limit)}
      err -> err
    end
  end

  @doc """
  Reads a context by its `optimal://` URI.
  Returns `{:ok, %Context{}}` or `{:error, reason}`.
  """
  @spec read(String.t()) :: {:ok, Context.t()} | {:error, term()}
  def read(uri) when is_binary(uri) do
    with {:ok, fs_path} <- URI.resolve(uri) do
      read_by_path(fs_path)
    end
  end

  defp read_by_path(fs_path) do
    id =
      :crypto.hash(:sha256, fs_path)
      |> Base.encode16(case: :lower)
      |> String.slice(0, 32)

    case Store.get_context(id) do
      {:ok, _} = result -> result
      {:error, :not_found} -> Indexer.index_file(fs_path)
    end
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp load_topology do
    case Routing.load() do
      {:ok, t} -> t
      {:error, _} -> %{endpoints: %{}}
    end
  end

  defp topology_entities(topology) do
    topology
    |> Map.get(:endpoints, %{})
    |> Map.values()
    |> Enum.flat_map(fn ep ->
      [ep.name, ep.name |> String.split(" ") |> List.first()]
    end)
    |> Enum.reject(&(String.length(&1) < 3))
  end

  defp route_signal(%Signal{} = signal) do
    case OptimalEngine.Pipeline.Router.route(signal) do
      {:ok, destinations} -> destinations
      {:error, _} -> [signal.node || "inbox"]
    end
  end

  defp generate_id do
    :crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower)
  end
end
