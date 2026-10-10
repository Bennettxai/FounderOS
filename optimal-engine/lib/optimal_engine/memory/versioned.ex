defmodule OptimalEngine.Memory.Versioned do
  @moduledoc """
  First-class versioned memory with typed relations and soft forgetting.

  Every memory is workspace-scoped, audience-aware, and participates in a
  version chain. Business operations:

  - `create/1`  — insert a new memory (v1, is_latest=true)
  - `get/1`     — fetch by id
  - `list/1`    — list with filters (workspace, audience, forgotten, old versions)
  - `update/2`  — bump version: creates v(n+1) linked via `:updates` relation,
                  demotes old to is_latest=false
  - `extend/2`  — creates a child linked via `:extends`; both keep is_latest
  - `derive/2`  — creates a derived memory linked via `:derives`; both keep is_latest
  - `forget/2`  — soft delete: sets is_forgotten=1 + optional reason/forget_after
  - `versions/1` — returns the full version chain (root → ... → latest)
  - `relations/1` — returns all typed edges (inbound + outbound)
  - `delete/1`  — hard delete; cascades memory_relations

  ## Relation types

      :updates      — new version supersedes old (old is_latest=0)
      :extends      — child adds to parent (both is_latest=1)
      :derives      — derived/synthesized from source (both is_latest=1)
      :contradicts  — records a contradiction between two memories
      :cites        — this memory cites another as evidence

  ## Storage

  Backed by the `memories` and `memory_relations` tables created in
  migration 028. All SQL is encapsulated in `OptimalEngine.Memory.Versioned.Store`.

  ## Surfacer integration

  After a successful `create/1`, fires
  `OptimalEngine.Memory.Surfacer.notify_memory_added/3` so proactive
  surfacing subscribers receive the new memory.

  ## ID format

  UUIDs via the `uuid` library (`UUID.uuid4()`). Prefixed `mem_` for
  readability in logs.
  """

  alias OptimalEngine.Memory.Versioned.Store, as: VStore
  alias OptimalEngine.Memory.Surfacer
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Workspace

  @valid_relations ~w(updates extends derives contradicts cites)

  @type relation_type :: :updates | :extends | :derives | :contradicts | :cites

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          content: String.t(),
          content_hash: String.t() | nil,
          was_existing: boolean(),
          is_static: boolean(),
          is_forgotten: boolean(),
          forget_after: String.t() | nil,
          forget_reason: String.t() | nil,
          version: pos_integer(),
          parent_memory_id: String.t() | nil,
          root_memory_id: String.t() | nil,
          is_latest: boolean(),
          citation_uri: String.t() | nil,
          source_chunk_id: String.t() | nil,
          audience: String.t(),
          metadata: map(),
          created_at: String.t() | nil,
          updated_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            workspace_id: Workspace.default_id(),
            content: nil,
            content_hash: nil,
            was_existing: false,
            is_static: false,
            is_forgotten: false,
            forget_after: nil,
            forget_reason: nil,
            version: 1,
            parent_memory_id: nil,
            root_memory_id: nil,
            is_latest: true,
            citation_uri: nil,
            source_chunk_id: nil,
            audience: "default",
            metadata: %{},
            created_at: nil,
            updated_at: nil

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Creates a new memory (version 1, is_latest=true).

  Required: `:content`
  Optional: `:workspace_id`, `:tenant_id`, `:is_static`, `:audience`,
            `:citation_uri`, `:source_chunk_id`, `:metadata`

  ## Deduplication

  Before inserting, computes a SHA-256 content hash over the trimmed and
  lowercased content. The behaviour on collision is controlled by the
  `:dedup` option (or the workspace `memory.dedup_policy` config key):

    - `"return_existing"` *(default)* — returns `{:ok, existing}` with
      `was_existing: true`. No new row is written.
    - `"bump_version"` — calls `update/2` on the existing memory, promoting
      it to v(n+1) with the caller's attrs. Returns the new version.
    - `"always_insert"` — skips the dedup check entirely (escape hatch for
      tests and bulk-import pipelines).

  The dedup check only considers *live* memories (`is_forgotten = 0`,
  `is_latest = 1`). A forgotten memory with the same content will not block
  a fresh insert.
  """
  @spec create(map()) :: {:ok, t()} | {:error, term()}
  def create(%{content: content} = attrs) when is_binary(content) and content != "" do
    workspace_id = Map.get(attrs, :workspace_id, Workspace.default_id())
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    audience = Map.get(attrs, :audience, "default")
    dedup_policy = resolve_dedup_policy(attrs)
    content_hash = compute_content_hash(content)

    case dedup_policy do
      "always_insert" ->
        do_insert(attrs, workspace_id, tenant_id, audience, content, content_hash)

      policy when policy in ["return_existing", "bump_version"] ->
        case VStore.find_by_content_hash(workspace_id, audience, content_hash) do
          {:ok, existing_row} ->
            existing = row_to_struct(existing_row)

            if policy == "bump_version" do
              update(existing.id, attrs)
            else
              {:ok, %{existing | was_existing: true}}
            end

          {:error, :not_found} ->
            do_insert(attrs, workspace_id, tenant_id, audience, content, content_hash)

          other ->
            other
        end

      _ ->
        do_insert(attrs, workspace_id, tenant_id, audience, content, content_hash)
    end
  end

  def create(_), do: {:error, :missing_required_fields}

  @doc "Fetches a memory by id."
  @spec get(String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    case VStore.get(id) do
      {:ok, row} -> {:ok, row_to_struct(row)}
      other -> other
    end
  end

  @doc """
  Lists memories.

  Options:
    - `:workspace_id` — defaults to "default"
    - `:audience` — string filter; omit to list all audiences
    - `:include_forgotten` — default false
    - `:include_old_versions` — default false (only is_latest=1 rows)
    - `:limit` — default 50
    - `:offset` — default 0 (for pagination)
  """
  @spec list(keyword()) :: {:ok, [t()]} | {:error, term()}
  def list(opts \\ []) do
    case VStore.list(opts) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  @doc """
  Counts memories matching the same filter opts as `list/1` (minus limit/offset).
  """
  @spec count(keyword()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(opts \\ []) do
    VStore.count(opts)
  end

  @doc """
  Creates a new version of a memory. The new version:
    - increments `version`
    - sets `parent_memory_id` to the old id
    - inherits `root_memory_id` from the old memory
    - gets `is_latest = true`

  The old memory gets `is_latest = false`. A `:updates` relation is added.

  Any mutable field from `attrs` is applied to the new version.
  """
  @spec update(String.t(), map()) :: {:ok, t()} | {:error, term()}
  def update(id, attrs) when is_binary(id) and is_map(attrs) do
    with {:ok, old} <- get(id) do
      new_id = generate_id()

      row_attrs = %{
        id: new_id,
        tenant_id: old.tenant_id,
        workspace_id: old.workspace_id,
        content: Map.get(attrs, :content, old.content),
        is_static: Map.get(attrs, :is_static, old.is_static),
        version: old.version + 1,
        parent_memory_id: old.id,
        root_memory_id: old.root_memory_id || old.id,
        is_latest: true,
        citation_uri: Map.get(attrs, :citation_uri, old.citation_uri),
        source_chunk_id: Map.get(attrs, :source_chunk_id, old.source_chunk_id),
        audience: Map.get(attrs, :audience, old.audience),
        metadata: Map.get(attrs, :metadata, old.metadata)
      }

      with {:ok, _} <- VStore.insert(row_attrs),
           {:ok, _} <- VStore.demote_latest(old.id),
           {:ok, _} <-
             VStore.add_relation(new_id, old.id, "updates", old.workspace_id, old.tenant_id) do
        {:ok, struct_from_attrs(row_attrs)}
      end
    end
  end

  @doc """
  Creates a child memory with relation `:extends`. The source memory
  retains `is_latest=true`. Use this to annotate or augment a memory
  without replacing it.
  """
  @spec extend(String.t(), map()) :: {:ok, t()} | {:error, term()}
  def extend(id, attrs) when is_binary(id) and is_map(attrs) do
    create_derived(id, attrs, "extends")
  end

  @doc """
  Creates a derived memory with relation `:derives`. Source retains
  `is_latest=true`. Use this for synthesized or inferred memories.
  """
  @spec derive(String.t(), map()) :: {:ok, t()} | {:error, term()}
  def derive(id, attrs) when is_binary(id) and is_map(attrs) do
    create_derived(id, attrs, "derives")
  end

  @doc """
  Soft-forgets a memory. Sets `is_forgotten=1`. The row is never touched
  otherwise — hard delete via `delete/1` if needed.

  Options:
    - `:reason` — string explanation stored in `forget_reason`
    - `:forget_after` — ISO8601 timestamp; UI/retention sweeps may use this
  """
  @spec forget(String.t(), keyword()) :: :ok | {:error, term()}
  def forget(id, opts \\ []) when is_binary(id) do
    reason = Keyword.get(opts, :reason)
    forget_after = Keyword.get(opts, :forget_after)

    case VStore.mark_forgotten(id, reason, forget_after) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc """
  Returns the full version chain for the given memory id.
  Finds the root and returns all memories sharing that root, ordered
  by version ascending (oldest first).
  """
  @spec versions(String.t()) :: {:ok, [t()]} | {:error, term()}
  def versions(id) when is_binary(id) do
    with {:ok, mem} <- get(id) do
      root_id = mem.root_memory_id || mem.id

      case VStore.get_version_chain(root_id, mem.workspace_id) do
        {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
        other -> other
      end
    end
  end

  @doc """
  Returns all typed relations touching the given memory id (inbound +
  outbound). Each relation map has:
    - `:id` — relation row id
    - `:source_memory_id`
    - `:target_memory_id`
    - `:relation` — atom (:updates | :extends | :derives | :contradicts | :cites)
    - `:direction` — :outbound | :inbound
    - `:created_at`
  """
  @spec relations(String.t()) :: {:ok, [map()]} | {:error, term()}
  def relations(id) when is_binary(id) do
    case VStore.get_relations(id) do
      {:ok, rows} ->
        result =
          Enum.map(rows, fn [rel_id, source, target, relation, created_at] ->
            %{
              id: rel_id,
              source_memory_id: source,
              target_memory_id: target,
              relation: safe_atom(relation, @valid_relations),
              direction: if(source == id, do: :outbound, else: :inbound),
              created_at: created_at
            }
          end)

        {:ok, result}

      other ->
        other
    end
  end

  @doc "Hard deletes a memory row. Cascades to memory_relations."
  @spec delete(String.t()) :: :ok | {:error, term()}
  def delete(id) when is_binary(id) do
    case VStore.delete(id) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  # Actual insert path — shared by create/1 (new row) after dedup check.
  defp do_insert(attrs, workspace_id, tenant_id, audience, content, content_hash) do
    id = generate_id()

    row_attrs = %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      content: content,
      is_static: Map.get(attrs, :is_static, false),
      version: 1,
      parent_memory_id: nil,
      root_memory_id: id,
      is_latest: true,
      citation_uri: Map.get(attrs, :citation_uri),
      source_chunk_id: Map.get(attrs, :source_chunk_id),
      audience: audience,
      metadata: Map.get(attrs, :metadata, %{}),
      content_hash: content_hash
    }

    case VStore.insert(row_attrs) do
      {:ok, _} ->
        mem = struct_from_attrs(row_attrs)
        notify_surfacer(mem)
        {:ok, mem}

      other ->
        other
    end
  end

  # SHA-256 of content_normalized = content |> String.trim() |> String.downcase()
  # Returns a 64-char lowercase hex string.
  defp compute_content_hash(content) when is_binary(content) do
    normalized = content |> String.trim() |> String.downcase()
    :crypto.hash(:sha256, normalized) |> Base.encode16(case: :lower)
  end

  # Resolves the dedup policy. Priority: explicit `:dedup` opt in attrs >
  # workspace config `memory.dedup_policy` > hardcoded default.
  defp resolve_dedup_policy(attrs) do
    case Map.get(attrs, :dedup) do
      policy when is_binary(policy) ->
        policy

      policy when is_atom(policy) and not is_nil(policy) ->
        Atom.to_string(policy)

      nil ->
        "return_existing"
    end
  end

  defp create_derived(source_id, attrs, relation) do
    with {:ok, source} <- get(source_id),
         {:ok, child} <-
           create(
             Map.merge(%{workspace_id: source.workspace_id, tenant_id: source.tenant_id}, attrs)
           ) do
      case VStore.add_relation(
             child.id,
             source.id,
             relation,
             source.workspace_id,
             source.tenant_id
           ) do
        {:ok, _} -> {:ok, child}
        err -> err
      end
    end
  end

  defp notify_surfacer(%__MODULE__{workspace_id: ws_id, id: mem_id, metadata: meta}) do
    if function_exported?(Surfacer, :notify_memory_added, 3) do
      Surfacer.notify_memory_added(ws_id, mem_id, meta)
    end

    :ok
  end

  defp generate_id do
    "mem_" <> UUID.uuid4()
  end

  defp struct_from_attrs(attrs) do
    %__MODULE__{
      id: attrs.id,
      tenant_id: attrs.tenant_id,
      workspace_id: attrs.workspace_id,
      content: attrs.content,
      content_hash: attrs[:content_hash],
      was_existing: false,
      is_static: normalize_bool(attrs[:is_static] || false),
      is_forgotten: false,
      forget_after: nil,
      forget_reason: nil,
      version: attrs[:version] || 1,
      parent_memory_id: attrs[:parent_memory_id],
      root_memory_id: attrs[:root_memory_id],
      is_latest: normalize_bool(attrs[:is_latest] != false),
      citation_uri: attrs[:citation_uri],
      source_chunk_id: attrs[:source_chunk_id],
      audience: attrs[:audience] || "default",
      metadata: attrs[:metadata] || %{},
      created_at: nil,
      updated_at: nil
    }
  end

  defp row_to_struct([
         id,
         tenant_id,
         workspace_id,
         content,
         is_static,
         is_forgotten,
         forget_after,
         forget_reason,
         version,
         parent_memory_id,
         root_memory_id,
         is_latest,
         citation_uri,
         source_chunk_id,
         audience,
         metadata_json,
         created_at,
         updated_at
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      content: content,
      content_hash: nil,
      was_existing: false,
      is_static: normalize_bool(is_static),
      is_forgotten: normalize_bool(is_forgotten),
      forget_after: forget_after,
      forget_reason: forget_reason,
      version: version,
      parent_memory_id: parent_memory_id,
      root_memory_id: root_memory_id,
      is_latest: normalize_bool(is_latest),
      citation_uri: citation_uri,
      source_chunk_id: source_chunk_id,
      audience: audience,
      metadata: decode_json(metadata_json),
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp normalize_bool(1), do: true
  defp normalize_bool(0), do: false
  defp normalize_bool(true), do: true
  defp normalize_bool(false), do: false
  defp normalize_bool(_), do: false

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(s) when is_binary(s) do
    case Jason.decode(s) do
      {:ok, m} when is_map(m) -> m
      _ -> %{}
    end
  end

  defp safe_atom(str, valid) when is_binary(str) do
    atom = String.to_atom(str)
    if Atom.to_string(atom) in valid, do: atom, else: :unknown
  end

  defp safe_atom(_, _), do: :unknown
end
