defmodule OptimalEngine.Memory do
  @moduledoc """
  Unified memory API for OptimalEngine.

  This module provides two complementary memory APIs:

  ## 1. Versioned memory (primary — SQL-backed)

  First-class, workspace-scoped, versioned memory with typed relations and
  soft forgetting. Backed by the `memories` and `memory_relations` tables
  (migration 028). Delegates to `OptimalEngine.Memory.Versioned`.

  Key operations: `create/1`, `get/1`, `list/1`, `update/2`, `extend/2`,
  `derive/2`, `forget/2`, `versions/1`, `relations/1`, `delete/1`.

  ## 2. Key-value memory (legacy — ETS-backed)

  Simple collection/key/value store used by the agent memory system,
  episodic memory, and session management. Backed by ETS via
  `OptimalEngine.Memory.Store.ETS`.

  Key operations: `store/4`, `recall/2`, `search/2`, `forget/3`,
  `collections/0`, `export/2`, `import_collection/2`.

  The two APIs coexist without conflict. `forget/2` dispatches based on
  argument types: `forget(id, keyword_opts)` → versioned; the legacy
  path uses the explicit `forget_key/2` alias.
  """

  alias OptimalEngine.Memory.{EncodingGate, SemanticDedup, Versioned}

  alias OptimalEngine.MemoryCore.{
    Claim,
    ClaimExtractor,
    DerivationLedgerEntry,
    ScoringPolicy,
    SourcePackage,
    Store
  }

  require Logger

  # ===========================================================================
  # Versioned memory API (primary)
  # ===========================================================================

  @type versioned :: OptimalEngine.Memory.Versioned.t()

  @doc """
  Creates a new versioned memory (v1, is_latest=true).

  Required: `:content` (non-empty string)
  Optional: `:workspace_id`, `:tenant_id`, `:is_static`, `:audience`,
            `:citation_uri`, `:source_chunk_id`, `:metadata`

  When `is_static: true` and the workspace config has
  `memory.auto_promote_to_wiki: true`, the memory is asynchronously
  promoted to a wiki page (fire-and-forget via `Task.start`).
  """
  @spec create(map()) :: {:ok, versioned()} | {:error, term()}
  def create(attrs) do
    case Versioned.create(attrs) do
      {:ok, mem} = ok ->
        maybe_create_pending_claim(attrs, mem)
        maybe_promote_to_wiki(mem)
        ok

      other ->
        other
    end
  end

  defp maybe_create_pending_claim(%{content: content} = attrs, mem)
       when is_binary(content) and content != "" do
    gate = %EncodingGate{
      should_encode: true,
      score: 1.0,
      novelty: 1.0,
      salience: 1.0,
      prediction_error: 0.0,
      reason: "memory_create_bridge"
    }

    source_package =
      memory_source_package(content, attrs, [], gate, true,
        source_type: "versioned_memory",
        source_uri: "memory://#{mem.id}",
        metadata: %{"versioned_memory_id" => mem.id}
      )

    claim_opts = [
      actor_id: "system:memory-create-bridge",
      extracted_by: "system:memory-create-bridge",
      claim_text: content,
      metadata: governed_memory_metadata(attrs, gate, true, %{"versioned_memory_id" => mem.id})
    ]

    case ClaimExtractor.extract_from_source(source_package, claim_opts) do
      {:ok, _claim} ->
        :ok

      {:error, reason} ->
        Logger.warning("Memory.create pending claim bridge failed: #{inspect(reason)}")
    end
  end

  defp maybe_create_pending_claim(_attrs, _mem), do: :ok

  @doc """
  Governed memory intake for agent-facing writes.

  `create/1` is the low-level legacy versioned-memory primitive. `remember/2`
  is the higher-level governed path: it preserves the input as a Source Package
  first, then either records a rejected intake decision or creates a pending
  Claim. It never promotes text directly into durable truth.

  Returns `{:ok, %{action:, memory:, source_package:, pending_claim:, gate:}}`.

  Options:
    - `:force` — bypass the encoding gate and still create a pending Claim
    - `:gate_threshold` — default `0.30`
    - `:salience_floor` — default `0.10`
  """
  @spec remember(map(), keyword()) ::
          {:ok,
           %{
             action: :pending_claim | :rejected,
             memory: versioned() | nil,
             gate: EncodingGate.t(),
             source_package: SourcePackage.t(),
             pending_claim: Claim.t() | nil
           }}
          | {:error, term()}
  @spec remember(String.t(), keyword()) :: :ok | {:error, term()}
  def remember(input, opts \\ [])

  def remember(%{content: content} = attrs, opts) when is_binary(content) and content != "" do
    workspace_id = Map.get(attrs, :workspace_id, "default")
    audience = Map.get(attrs, :audience, "default")

    candidates = list(workspace_id: workspace_id, audience: audience, limit: 200)
    dedup = SemanticDedup.decide(content, candidates, opts)

    gate =
      EncodingGate.evaluate(content,
        candidates: candidates,
        threshold: Keyword.get(opts, :gate_threshold, 0.30),
        salience_floor: Keyword.get(opts, :salience_floor, 0.10)
      )

    encode? = gate.should_encode or Keyword.get(opts, :force, false)
    source_package = memory_source_package(content, attrs, opts, gate, encode?)

    if encode? do
      claim_opts =
        opts
        |> Keyword.put_new(:actor_id, Map.get(attrs, :actor_id) || Map.get(attrs, :created_by))
        |> Keyword.put_new(:claim_text, content)
        |> Keyword.put_new(:metadata, governed_memory_metadata(attrs, gate, encode?))

      case ClaimExtractor.extract_from_source(source_package, claim_opts) do
        {:ok, claim} ->
          {action, memory} = maybe_versioned_projection(attrs, source_package, claim, dedup, opts)

          {:ok,
           %{
             action: action,
             memory: memory,
             source_package: source_package,
             pending_claim: claim,
             gate: gate,
             dedup: dedup
           }}

        other ->
          other
      end
    else
      with :ok <- Store.insert_source_package(source_package),
           :ok <- record_remember_rejection(source_package, gate, opts) do
        {:ok,
         %{
           action: :rejected,
           memory: nil,
           source_package: source_package,
           pending_claim: nil,
           gate: gate,
           dedup: dedup
         }}
      end
    end
  end

  def remember(insight, opts) when is_binary(insight) do
    tags = Keyword.get(opts, :tags, [])
    key = Keyword.get(opts, :key, "insight_#{:erlang.unique_integer([:positive])}")
    store("memory", key, insight, tags: tags)
  end

  def remember(_attrs, _opts), do: {:error, :missing_required_fields}

  defp maybe_versioned_projection(attrs, source_package, claim, dedup, opts) do
    if Keyword.get(opts, :versioned_projection, false) do
      do_versioned_projection(attrs, source_package, claim, dedup)
    else
      {:pending_claim, nil}
    end
  end

  defp do_versioned_projection(_attrs, _source_package, _claim, %{action: :skip, existing: existing})
       when not is_nil(existing) do
    {:skip, Map.put(existing, :was_existing, true)}
  end

  defp do_versioned_projection(attrs, source_package, claim, _dedup) do
    metadata =
      attrs
      |> Map.get(:metadata, %{})
      |> normalize_metadata()
      |> Map.merge(%{
        "memory_core_source_package_id" => source_package.id,
        "memory_core_pending_claim_id" => claim.id,
        "projection_kind" => "versioned_memory_staging"
      })

    projection_attrs =
      attrs
      |> Map.put(:metadata, metadata)

    case Versioned.create(projection_attrs) do
      {:ok, memory} -> {:add, memory}
      {:error, {:duplicate, memory}} -> {:skip, Map.put(memory, :was_existing, true)}
      {:error, _reason} -> {:pending_claim, nil}
    end
  end

  @doc "Fetches a versioned memory by id."
  @spec get(String.t()) :: {:ok, versioned()} | {:error, :not_found}
  defdelegate get(id), to: Versioned

  @doc """
  Lists versioned memories.

  Returns a plain list (not `{:ok, list}`) for ergonomic router use.

  Options:
    - `:workspace_id` — defaults to "default"
    - `:audience` — string filter
    - `:include_forgotten` — default false
    - `:include_old_versions` — default false
    - `:limit` — default 50
  """
  @spec list(keyword()) :: [versioned()]
  def list(opts \\ []) do
    case Versioned.list(opts) do
      {:ok, mems} -> mems
      _ -> []
    end
  end

  @doc """
  Counts versioned memories matching the same filter opts as `list/1`.
  Returns `{:ok, integer()}` so callers can distinguish 0 from error.
  """
  @spec count(keyword()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(opts \\ []), do: Versioned.count(opts)

  @doc """
  Creates a new version of a memory (bumps version, demotes old to
  is_latest=false, adds `:updates` relation).
  """
  @spec update(String.t(), map()) :: {:ok, versioned()} | {:error, term()}
  defdelegate update(id, attrs), to: Versioned

  @doc """
  Creates a child memory with `:extends` relation. Source keeps is_latest.
  """
  @spec extend(String.t(), map()) :: {:ok, versioned()} | {:error, term()}
  defdelegate extend(id, attrs), to: Versioned

  @doc """
  Creates a derived memory with `:derives` relation. Source keeps is_latest.
  """
  @spec derive(String.t(), map()) :: {:ok, versioned()} | {:error, term()}
  defdelegate derive(id, attrs), to: Versioned

  @doc """
  Soft-forgets a versioned memory (sets is_forgotten=1).

  Options:
    - `:reason` — stored in `forget_reason`
    - `:forget_after` — ISO8601 timestamp

  Dispatches on the second argument type:
    - `forget(id, keyword_list)` — versioned memory soft-forget
    - `forget(id, binary)` — alias for legacy `forget_key/2`
  """
  @spec forget(String.t(), keyword() | String.t()) :: :ok | {:error, term()}
  def forget(id, opts \\ [])

  # Versioned soft-forget: forget(memory_id, keyword_opts)
  def forget(id, opts) when is_binary(id) and is_list(opts) do
    Versioned.forget(id, opts)
  end

  # Legacy ETS delete: forget(collection, key) — both strings
  def forget(collection, key) when is_binary(collection) and is_binary(key) do
    backend().delete(collection, key)
  end

  @doc "Returns the full version chain ordered v1 → ... → latest."
  @spec versions(String.t()) :: {:ok, [versioned()]} | {:error, term()}
  defdelegate versions(id), to: Versioned

  @doc """
  Returns all typed relations touching `memory_id` (inbound + outbound).

  Each relation map includes:
    - `:id`, `:source_memory_id`, `:target_memory_id`, `:target_id` (alias),
      `:relation`, `:direction`, `:created_at`
  """
  @spec relations(String.t()) :: {:ok, [map()]} | {:error, term()}
  def relations(id) when is_binary(id) do
    case Versioned.relations(id) do
      {:ok, rels} ->
        # Add :target_id alias so the router's `r.target_id` resolves correctly.
        enriched = Enum.map(rels, &Map.put(&1, :target_id, &1.target_memory_id))
        {:ok, enriched}

      other ->
        other
    end
  end

  @doc "Hard deletes a versioned memory (cascades relations)."
  @spec delete(String.t()) :: :ok | {:error, term()}
  defdelegate delete(id), to: Versioned

  # ===========================================================================
  # Legacy key-value memory API (ETS-backed)
  # ===========================================================================

  @type collection :: String.t()
  @type key :: String.t()
  @type value :: term()

  @doc """
  Store a memory entry in a collection.

  ## Parameters
    - `collection` - the memory collection name (e.g., "decisions", "patterns")
    - `key` - unique key within the collection
    - `value` - any term to store

  ## Options
    - `:tags` - list of string tags for search (default: `[]`)
  """
  @spec store(collection(), key(), value(), keyword()) :: :ok | {:error, term()}
  def store(collection, key, value, opts \\ []) do
    tags = Keyword.get(opts, :tags, [])
    metadata = %{tags: tags}
    backend().put(collection, key, value, metadata)
  end

  @doc """
  Recall a memory entry by collection and key.

  Returns `{:ok, entry}` or `{:error, :not_found}`.
  """
  @spec recall(collection(), key()) ::
          {:ok, OptimalEngine.Memory.Store.entry()} | {:error, :not_found}
  def recall(collection, key) do
    backend().get(collection, key)
  end

  @doc """
  Search memories in a collection by query string (keyword match).

  Returns `{:ok, matches}` with entries whose key, value, or tags match
  any query term.
  """
  @spec search(collection(), String.t()) :: {:ok, [OptimalEngine.Memory.Store.entry()]}
  def search(collection, query) do
    backend().search(collection, query)
  end

  @doc """
  Delete a legacy key-value memory entry.

  Two-arg form: `forget(collection, key)` — legacy ETS delete.
  Single-arg / keyword form: `forget(id)` / `forget(id, opts)` — versioned.
  """
  @spec forget_key(collection(), key()) :: :ok
  def forget_key(collection, key) do
    backend().delete(collection, key)
  end

  @doc """
  List all memory collections.
  """
  @spec collections() :: {:ok, [collection()]}
  def collections do
    backend().collections()
  end

  @doc """
  Recalls all long-term memories as a single concatenated string.
  Used by the Bridge and Cortex for synthesis.
  """
  @spec recall() :: String.t()
  def recall do
    OptimalEngine.Memory.Store.recall()
  end

  @doc """
  Export a collection to a JSON file at the given path.
  """
  @spec export(collection(), String.t()) :: :ok | {:error, term()}
  def export(collection, path) do
    {:ok, entries} = backend().list(collection, [])

    data =
      Enum.map(entries, fn entry ->
        entry
        |> put_in([:metadata, :created_at], DateTime.to_iso8601(entry.metadata.created_at))
        |> put_in([:metadata, :updated_at], DateTime.to_iso8601(entry.metadata.updated_at))
      end)

    dir = Path.dirname(path)
    File.mkdir_p!(dir)
    File.write!(path, Jason.encode!(data, pretty: true))
    :ok
  end

  @doc """
  Import a collection from a JSON file at the given path.
  Entries are merged into the specified collection.
  """
  @spec import_collection(collection(), String.t()) :: :ok | {:error, term()}
  def import_collection(collection, path) do
    case File.read(path) do
      {:ok, content} ->
        entries = Jason.decode!(content, keys: :atoms)

        Enum.each(entries, fn entry ->
          metadata =
            entry.metadata
            |> Map.update!(:created_at, &parse_dt/1)
            |> Map.update!(:updated_at, &parse_dt/1)

          backend().put(collection, entry.key, entry.value, metadata)
        end)

        :ok

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp parse_dt(dt) when is_binary(dt) do
    {:ok, parsed, _} = DateTime.from_iso8601(dt)
    parsed
  end

  defp parse_dt(%DateTime{} = dt), do: dt

  defp memory_source_package(content, attrs, opts, gate, encode?, overrides \\ []) do
    override_metadata = Keyword.get(overrides, :metadata, %{})

    SourcePackage.from_text(content,
      tenant_id: Map.get(attrs, :tenant_id, "default"),
      workspace_id: Map.get(attrs, :workspace_id, "default"),
      source_type: Keyword.get(overrides, :source_type, "memory_remember"),
      source_class: "text",
      source_system: "optimal_engine.memory",
      source_uri: Keyword.get(overrides, :source_uri),
      trust_label: "unreviewed",
      quarantine_state: if(encode?, do: "clear", else: "rejected"),
      metadata: governed_memory_metadata(attrs, gate, encode?, override_metadata),
      created_by:
        Map.get(attrs, :created_by) || Map.get(attrs, :actor_id) || Keyword.get(opts, :actor_id),
      access_policy_id: Map.get(attrs, :access_policy_id),
      security_labels: Map.get(attrs, :security_labels, []),
      partition_ids: Map.get(attrs, :partition_ids, [])
    )
  end

  defp governed_memory_metadata(attrs, gate, encode?, extra \\ %{}) do
    path = if encode?, do: "memory_core_pending_claim", else: "rejected_source_package"

    attrs
    |> Map.get(:metadata, %{})
    |> normalize_metadata()
    |> Map.merge(normalize_metadata(extra))
    |> Map.put("memory_intake", %{
      "path" => path,
      "gate" => gate_summary(gate)
    })
  end

  defp record_remember_rejection(%SourcePackage{} = source_package, gate, opts) do
    source_ref = DerivationLedgerEntry.object_ref("source_package", source_package.id)

    ledger =
      DerivationLedgerEntry.new(
        "memory.remember",
        "source_rejected",
        [source_ref],
        [source_ref],
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_links: [source_ref],
        evidence_links: [source_ref],
        actor_id: Keyword.get(opts, :actor_id) || source_package.created_by,
        parser_id: "optimal_engine.memory.remember",
        scoring_policy_version: ScoringPolicy.version(),
        access_policy_id: source_package.access_policy_id,
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        metadata: %{
          quarantine_state: "rejected",
          reason: "encoding_gate_rejected",
          gate: gate_summary(gate)
        }
      )

    Store.insert_derivation_entry(ledger)
  end

  defp gate_summary(gate) do
    %{
      "should_encode" => gate.should_encode,
      "score" => gate.score,
      "novelty" => gate.novelty,
      "salience" => gate.salience,
      "prediction_error" => gate.prediction_error,
      "reason" => gate.reason,
      "similar_memory_id" => gate.similar_memory_id
    }
  end

  defp normalize_metadata(nil), do: %{}
  defp normalize_metadata(metadata) when is_map(metadata), do: metadata
  defp normalize_metadata(_metadata), do: %{}

  defp backend, do: OptimalEngine.Memory.Store.backend()

  # ---------------------------------------------------------------------------
  # WikiBridge auto-promotion (Phase 17.1/7 bridge)
  # ---------------------------------------------------------------------------

  # Fire-and-forget: if the memory is static and the workspace config enables
  # auto_promote_to_wiki, promote it to the "memory-promotions" wiki page.
  # Any failure is logged and swallowed — memory create must never be blocked.
  defp maybe_promote_to_wiki(%Versioned{is_static: true} = mem) do
    alias OptimalEngine.Workspace.Config
    mem_cfg = Config.get_section(mem.workspace_id, :memory, %{auto_promote_to_wiki: false})

    if Map.get(mem_cfg, :auto_promote_to_wiki, false) do
      Task.start(fn ->
        opts = [
          workspace_id: mem.workspace_id,
          tenant_id: mem.tenant_id,
          audience: mem.audience
        ]

        slug = "memory-promotions"

        case OptimalEngine.Memory.WikiBridge.promote_memory_to_wiki(mem.id, slug, opts) do
          {:ok, _page} ->
            Logger.info("[Memory] auto-promoted #{mem.id} to wiki/#{slug}")

          {:error, reason} ->
            Logger.warning(
              "[Memory] WikiBridge.promote_memory_to_wiki failed for #{mem.id}: #{inspect(reason)}"
            )
        end
      end)
    end

    :ok
  rescue
    e ->
      Logger.warning("[Memory] maybe_promote_to_wiki error: #{inspect(e)}")
      :ok
  end

  defp maybe_promote_to_wiki(_mem), do: :ok
end
