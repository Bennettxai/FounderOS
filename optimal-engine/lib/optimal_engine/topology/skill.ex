defmodule OptimalEngine.Topology.Skill do
  @moduledoc """
  A named, tenant-scoped capability: `"elixir"`, `"enterprise-sales"`,
  `"phoenix-arbiter-model"`, `"sql-optimization"`.

  Skills apply uniformly to humans and agents — both are principals; both
  can hold skills at any `level`. This lets capability lookups ("who can
  optimize this SQL?") return a mix without the caller needing to know
  which holders are people and which are agents.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type kind :: :technical | :communication | :strategic | :domain | :tool

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          name: String.t(),
          kind: kind() | nil,
          description: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            name: nil,
            kind: nil,
            description: nil

  @allowed_kinds [:technical, :communication, :strategic, :domain, :tool]

  @doc """
  Upsert a skill. `id` defaults to `"{tenant_id}:{slug(name)}"` for a
  readable, deterministic key.
  """
  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{name: name} = attrs) when is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    id = Map.get(attrs, :id) || "#{tenant_id}:#{slugify(name)}"
    kind = Map.get(attrs, :kind)
    description = Map.get(attrs, :description)

    cond do
      not is_nil(kind) and kind not in @allowed_kinds ->
        {:error, {:invalid_kind, kind}}

      true ->
        sql = """
        INSERT INTO skills (id, tenant_id, name, kind, description)
        VALUES (?1, ?2, ?3, ?4, ?5)
        ON CONFLICT(id) DO UPDATE SET
          name        = excluded.name,
          kind        = excluded.kind,
          description = excluded.description
        """

        params = [id, tenant_id, name, kind && Atom.to_string(kind), description]

        case Store.raw_query(sql, params) do
          {:ok, _} ->
            {:ok,
             %__MODULE__{
               id: id,
               tenant_id: tenant_id,
               name: name,
               kind: kind,
               description: description
             }}

          other ->
            other
        end
    end
  end

  @doc "Lists skills, optionally filtered by kind."
  @spec list(keyword()) :: {:ok, [t()]}
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    kind = Keyword.get(opts, :kind)

    {sql, params} =
      case kind do
        nil ->
          {"SELECT id, tenant_id, name, kind, description FROM skills WHERE tenant_id = ?1 ORDER BY name",
           [tenant_id]}

        k when k in @allowed_kinds ->
          {"SELECT id, tenant_id, name, kind, description FROM skills WHERE tenant_id = ?1 AND kind = ?2 ORDER BY name",
           [tenant_id, Atom.to_string(k)]}
      end

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        {:ok,
         Enum.map(rows, fn [id, tid, name, kind_str, description] ->
           %__MODULE__{
             id: id,
             tenant_id: tid,
             name: name,
             kind: safe_atom(kind_str, nil),
             description: description
           }
         end)}

      other ->
        other
    end
  end

  @doc "Fetch by id, tenant-scoped."
  @spec get(String.t(), String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id, tenant_id \\ Tenant.default_id()) when is_binary(id) do
    case Store.raw_query(
           "SELECT id, tenant_id, name, kind, description FROM skills WHERE id = ?1 AND tenant_id = ?2",
           [id, tenant_id]
         ) do
      {:ok, [[id, tid, name, kind, description]]} ->
        {:ok,
         %__MODULE__{
           id: id,
           tenant_id: tid,
           name: name,
           kind: safe_atom(kind, nil),
           description: description
         }}

      {:ok, []} ->
        {:error, :not_found}

      other ->
        other
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp slugify(name) do
    name
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9]+/, "-")
    |> String.trim("-")
  end

  defp safe_atom(nil, fallback), do: fallback

  defp safe_atom(str, fallback) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      ArgumentError -> fallback
    end
  end
end
