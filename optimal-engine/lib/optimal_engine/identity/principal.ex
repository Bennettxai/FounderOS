defmodule OptimalEngine.Identity.Principal do
  @moduledoc """
  A principal is an identity the engine recognizes: a human user, an AI agent,
  or a service account. Principals are scoped to a tenant. Every read/write
  into the engine carries a principal so ACLs + audit can be enforced.

  ## Kinds

  - `:user` — a human. Typically authenticated via SAML/OIDC.
  - `:agent` — an AI agent acting on behalf of a user or autonomously.
  - `:service` — a service account (e.g., connector worker).

  ## Identity string conventions

  - `"user:ada@sample.com"`
  - `"agent:ada-bot"`
  - `"service:slack-connector"`

  Phase 1 exposes CRUD only. SAML / OIDC / SCIM adapters land in Phase 9–11.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type kind :: :user | :agent | :service

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          kind: kind(),
          display_name: String.t(),
          external_id: String.t() | nil,
          created_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            kind: :user,
            display_name: nil,
            external_id: nil,
            created_at: nil,
            metadata: %{}

  @doc "Upsert a principal. Idempotent on `id`."
  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{id: id, kind: kind, display_name: display_name} = attrs)
      when is_binary(id) and kind in [:user, :agent, :service] and is_binary(display_name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    external_id = Map.get(attrs, :external_id)
    metadata = Map.get(attrs, :metadata, %{})

    sql = """
    INSERT INTO principals (id, tenant_id, kind, display_name, external_id, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6)
    ON CONFLICT(id) DO UPDATE SET
      display_name = excluded.display_name,
      external_id  = excluded.external_id,
      metadata     = excluded.metadata
    """

    case Store.raw_query(sql, [
           id,
           tenant_id,
           Atom.to_string(kind),
           display_name,
           external_id,
           Jason.encode!(metadata)
         ]) do
      {:ok, _} ->
        {:ok,
         %__MODULE__{
           id: id,
           tenant_id: tenant_id,
           kind: kind,
           display_name: display_name,
           external_id: external_id,
           metadata: metadata
         }}

      other ->
        other
    end
  end

  @doc "Fetch a principal by id, tenant-scoped."
  @spec get(String.t(), String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id, tenant_id \\ Tenant.default_id()) when is_binary(id) do
    case Store.raw_query(
           """
           SELECT id, tenant_id, kind, display_name, external_id, created_at, metadata
           FROM principals WHERE id = ?1 AND tenant_id = ?2
           """,
           [id, tenant_id]
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Returns every group id the principal belongs to."
  @spec groups(String.t()) :: {:ok, [String.t()]}
  def groups(principal_id) when is_binary(principal_id) do
    case Store.raw_query(
           "SELECT group_id FROM principal_groups WHERE principal_id = ?1",
           [principal_id]
         ) do
      {:ok, rows} -> {:ok, Enum.map(rows, fn [g] -> g end)}
      other -> other
    end
  end

  @doc """
  Returns every role id granted to the principal (directly or via a group)
  within the given tenant.
  """
  @spec roles(String.t(), String.t()) :: {:ok, [String.t()]}
  def roles(principal_id, tenant_id \\ Tenant.default_id()) when is_binary(principal_id) do
    sql = """
    SELECT DISTINCT role_id FROM role_grants
    WHERE tenant_id = ?1 AND (
      principal_id = ?2
      OR group_id IN (SELECT group_id FROM principal_groups WHERE principal_id = ?3)
    )
    """

    case Store.raw_query(sql, [tenant_id, principal_id, principal_id]) do
      {:ok, rows} -> {:ok, Enum.map(rows, fn [r] -> r end)}
      other -> other
    end
  end

  defp row_to_struct([id, tenant_id, kind, display_name, external_id, created_at, meta_json]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      kind: String.to_existing_atom(kind),
      display_name: display_name,
      external_id: external_id,
      created_at: created_at,
      metadata: decode_json(meta_json)
    }
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
