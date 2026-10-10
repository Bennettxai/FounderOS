defmodule OptimalEngine.MemoryCore.ScopeEnvelope do
  @moduledoc """
  Best-known actor/workspace scope for an Evidence Intake operation.

  The Evidence Intake gate runs `command/raw input -> Scope Envelope ->
  Source Package -> Signal -> route to Node`. The Scope Envelope names who is
  acting (`actor`), where the evidence belongs (`tenant_id`, `workspace_id`),
  what kind of operation is happening (`operation_class`), which Nodes the
  caller hinted at (`node_hints`), and which permissions were presented
  (`permissions`).

  Unknown scope never blocks intake. Missing fields fall back to defaults and
  are listed in `unresolved`, so the engine still stores the Source Package
  and routes the derived Signal to inbox/quarantine. Missing Node scope is a
  routing problem, not a reason to lose evidence.

  The default workspace is tenant-scoped: when `workspace_id` is missing, it
  derives from the resolved tenant (`"default"` for the default tenant —
  preserving single-workspace behavior — and `"<tenant_id>:default"` for every
  other tenant, mirroring `OptimalEngine.Workspace`'s id derivation). Tenants
  therefore never share the literal `"default"` workspace.
  """

  alias OptimalEngine.Tenancy.Tenant

  @default_workspace_id "default"
  @default_operation_class "intake.process"

  @scope_keys ~w[actor actor_id created_by tenant_id workspace_id operation_class node node_hints permissions]

  @type t :: %__MODULE__{
          actor: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          operation_class: String.t(),
          node_hints: [String.t()],
          permissions: [String.t()],
          unresolved: [atom()]
        }

  defstruct actor: nil,
            tenant_id: "default",
            workspace_id: @default_workspace_id,
            operation_class: @default_operation_class,
            node_hints: [],
            permissions: [],
            unresolved: []

  @doc """
  Resolves the best-known scope from caller options.

  Accepts a keyword list or map. `overrides` win over `attrs`. Actor is read
  from `:actor`, `:actor_id`, or `:created_by` (first present). Fields that
  fall back to defaults are listed in `unresolved`. A missing `workspace_id`
  falls back to the resolved tenant's own default workspace (see moduledoc),
  never to a workspace shared across tenants.
  """
  @spec resolve(keyword() | map(), keyword() | map()) :: t()
  def resolve(attrs, overrides \\ []) do
    merged = Keyword.merge(normalize(attrs), normalize(overrides))

    actor = first_present(merged, [:actor, :actor_id, :created_by])
    tenant_id = first_present(merged, [:tenant_id])
    workspace_id = first_present(merged, [:workspace_id])
    operation_class = first_present(merged, [:operation_class])

    unresolved =
      []
      |> mark_unresolved(:actor, actor)
      |> mark_unresolved(:tenant_id, tenant_id)
      |> mark_unresolved(:workspace_id, workspace_id)
      |> Enum.reverse()

    resolved_tenant_id = to_string(tenant_id || Tenant.default_id())

    %__MODULE__{
      actor: actor && to_string(actor),
      tenant_id: resolved_tenant_id,
      workspace_id: to_string(workspace_id || default_workspace_id(resolved_tenant_id)),
      operation_class: to_string(operation_class || @default_operation_class),
      node_hints: resolve_node_hints(merged),
      permissions: string_list(Keyword.get(merged, :permissions, [])),
      unresolved: unresolved
    }
  end

  @doc """
  The default workspace id for a tenant.

  Mirrors `OptimalEngine.Workspace`'s id derivation: the default tenant's
  default workspace keeps the bare id `"default"` (single-workspace behavior
  preserved); every other tenant gets `"<tenant_id>:default"` so unscoped
  evidence from one tenant never commingles with another tenant's rows in a
  shared literal `"default"` workspace.
  """
  @spec default_workspace_id(String.t()) :: String.t()
  def default_workspace_id(tenant_id) when is_binary(tenant_id) do
    if tenant_id == Tenant.default_id(),
      do: @default_workspace_id,
      else: "#{tenant_id}:#{@default_workspace_id}"
  end

  @doc "True when no scope field had to fall back to a default."
  @spec resolved?(t()) :: boolean()
  def resolved?(%__MODULE__{unresolved: unresolved}), do: unresolved == []

  @doc """
  Options for `SourcePackage.from_text/2`, merging the resolved scope over the
  caller's intake options. Scope details (operation class, permissions, node
  hints, unresolved fields) are preserved in the Source Package metadata.
  """
  @spec source_package_opts(t(), keyword()) :: keyword()
  def source_package_opts(%__MODULE__{} = scope, opts \\ []) do
    metadata =
      opts
      |> Keyword.get(:metadata, %{})
      |> Map.put(:scope, %{
        operation_class: scope.operation_class,
        permissions: scope.permissions,
        node_hints: scope.node_hints,
        unresolved: Enum.map(scope.unresolved, &to_string/1)
      })

    Keyword.merge(opts,
      tenant_id: scope.tenant_id,
      workspace_id: scope.workspace_id,
      created_by: scope.actor,
      metadata: metadata
    )
  end

  @doc "Options for Derivation Ledger writes: actor plus tenant/workspace scope."
  @spec ledger_opts(t(), keyword()) :: keyword()
  def ledger_opts(%__MODULE__{} = scope, opts \\ []) do
    Keyword.merge(opts,
      actor_id: scope.actor,
      tenant_id: scope.tenant_id,
      workspace_id: scope.workspace_id
    )
  end

  @doc "Tenant/workspace scope for topology-aware Node folder resolution."
  @spec folder_scope(t()) :: keyword()
  def folder_scope(%__MODULE__{} = scope) do
    [tenant_id: scope.tenant_id, workspace_id: scope.workspace_id]
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp normalize(attrs) when is_list(attrs), do: attrs

  defp normalize(%__MODULE__{} = scope) do
    [
      actor: scope.actor,
      tenant_id: scope.tenant_id,
      workspace_id: scope.workspace_id,
      operation_class: scope.operation_class,
      node_hints: scope.node_hints,
      permissions: scope.permissions
    ]
  end

  defp normalize(%{} = attrs) do
    Enum.flat_map(attrs, fn
      {key, value} when is_atom(key) -> [{key, value}]
      {key, value} when is_binary(key) and key in @scope_keys -> [{atom_key(key), value}]
      {_key, _value} -> []
    end)
  end

  defp atom_key(key), do: String.to_existing_atom(key)

  defp first_present(attrs, keys) do
    Enum.find_value(keys, fn key ->
      case Keyword.get(attrs, key) do
        nil -> nil
        "" -> nil
        value -> value
      end
    end)
  end

  defp mark_unresolved(unresolved, field, nil), do: [field | unresolved]
  defp mark_unresolved(unresolved, _field, _value), do: unresolved

  defp resolve_node_hints(attrs) do
    (List.wrap(Keyword.get(attrs, :node)) ++ List.wrap(Keyword.get(attrs, :node_hints)))
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.map(&to_string/1)
    |> Enum.uniq()
  end

  defp string_list(value) do
    value
    |> List.wrap()
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.map(&to_string/1)
  end
end
