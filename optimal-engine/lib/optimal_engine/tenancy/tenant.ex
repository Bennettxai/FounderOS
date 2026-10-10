defmodule OptimalEngine.Tenancy.Tenant do
  @moduledoc """
  A tenant is the hard security and account boundary for isolated data and ACLs.
  One tenant can contain many organizations, and each organization can own many
  workspaces.
  For v0.1, single-tenant deployments use the `"default"` tenant seeded
  automatically by migration 015.

  ## Invariants

  - Every primary table row carries a `tenant_id`.
  - No cross-tenant reads, ever. Queries must pass a tenant scope.
  - The `default` tenant is reserved and cannot be deleted.
  """

  alias OptimalEngine.Store

  @default_id "default"

  @type t :: %__MODULE__{
          id: String.t(),
          name: String.t(),
          plan: String.t(),
          region: String.t() | nil,
          created_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            name: nil,
            plan: "default",
            region: nil,
            created_at: nil,
            metadata: %{}

  @doc "Returns the reserved default-tenant id."
  @spec default_id() :: String.t()
  def default_id, do: @default_id

  @doc "Returns the tenant with the given id, or `{:error, :not_found}`."
  @spec get(String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    case Store.raw_query(
           "SELECT id, name, plan, region, created_at, metadata FROM tenants WHERE id = ?1",
           [id]
         ) do
      {:ok, [[tid, name, plan, region, created_at, meta_json]]} ->
        {:ok,
         %__MODULE__{
           id: tid,
           name: name,
           plan: plan,
           region: region,
           created_at: created_at,
           metadata: decode_json(meta_json)
         }}

      {:ok, []} ->
        {:error, :not_found}

      other ->
        other
    end
  end

  @doc "Creates a new tenant. Returns `{:ok, tenant}` or `{:error, reason}`."
  @spec create(map()) :: {:ok, t()} | {:error, term()}
  def create(%{id: id, name: name} = attrs) when is_binary(id) and is_binary(name) do
    plan = Map.get(attrs, :plan, "default")
    region = Map.get(attrs, :region)
    metadata = Map.get(attrs, :metadata, %{})

    case Store.raw_query(
           """
           INSERT INTO tenants (id, name, plan, region, metadata)
           VALUES (?1, ?2, ?3, ?4, ?5)
           """,
           [id, name, plan, region, Jason.encode!(metadata)]
         ) do
      {:ok, _} ->
        {:ok, %__MODULE__{id: id, name: name, plan: plan, region: region, metadata: metadata}}

      other ->
        other
    end
  end

  @doc "Lists all tenants."
  @spec list() :: {:ok, [t()]}
  def list do
    case Store.raw_query(
           "SELECT id, name, plan, region, created_at, metadata FROM tenants ORDER BY id",
           []
         ) do
      {:ok, rows} ->
        tenants =
          Enum.map(rows, fn [id, name, plan, region, created_at, meta_json] ->
            %__MODULE__{
              id: id,
              name: name,
              plan: plan,
              region: region,
              created_at: created_at,
              metadata: decode_json(meta_json)
            }
          end)

        {:ok, tenants}

      other ->
        other
    end
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(str) when is_binary(str) do
    case Jason.decode(str) do
      {:ok, m} -> m
      _ -> %{}
    end
  end
end
