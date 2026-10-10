defmodule OptimalEngine.Governance.Audience do
  @moduledoc """
  Receiver/genre audiences — named groups of roles a Signal can be encoded for.

  An audience binds a human-meaningful name (e.g. "developers", "executives")
  to a set of `role_ids`. Profile + wiki surfacing use the audience name to
  decide *how* to encode an output (which genre, which bandwidth).

  Backed by the `audiences` table (migration 011):

      id, tenant_id, name, role_ids (JSON array), description
  """

  alias OptimalEngine.MemoryCore.ID
  alias OptimalEngine.Store

  @type audience :: %{
          id: String.t(),
          tenant_id: String.t(),
          name: String.t(),
          role_ids: [String.t()],
          description: String.t() | nil
        }

  @doc """
  Create an audience. Returns `{:ok, audience}`.

  Required: `:name`. Optional: `:role_ids` (list), `:description`,
  `:tenant_id` (default `"default"`).
  """
  @spec create(map() | keyword()) :: {:ok, audience()} | {:error, term()}
  def create(attrs) when is_list(attrs), do: attrs |> Map.new() |> create()

  def create(attrs) when is_map(attrs) do
    tenant_id = to_string(Map.get(attrs, :tenant_id) || Map.get(attrs, "tenant_id") || "default")
    name = Map.get(attrs, :name) || Map.get(attrs, "name")
    role_ids = Map.get(attrs, :role_ids) || Map.get(attrs, "role_ids") || []
    description = Map.get(attrs, :description) || Map.get(attrs, "description")

    cond do
      is_nil(name) ->
        {:error, :name_required}

      true ->
        id = Map.get(attrs, :id) || ID.content_id("audience", [tenant_id, ":", to_string(name)])

        with {:ok, _} <-
               Store.raw_query(
                 """
                 INSERT INTO audiences (id, tenant_id, name, role_ids, description)
                 VALUES (?1, ?2, ?3, ?4, ?5)
                 """,
                 [id, tenant_id, to_string(name), Jason.encode!(role_ids), description]
               ) do
          {:ok,
           %{
             id: id,
             tenant_id: tenant_id,
             name: to_string(name),
             role_ids: role_ids,
             description: description
           }}
        end
    end
  end

  @doc "Fetch an audience by id."
  @spec get(String.t(), keyword()) :: {:ok, audience()} | {:error, :not_found}
  def get(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")

    case Store.raw_query(
           "SELECT id, tenant_id, name, role_ids, description FROM audiences WHERE id = ?1 AND tenant_id = ?2",
           [id, tenant_id]
         ) do
      {:ok, [[aid, tid, name, role_ids, desc]]} ->
        {:ok,
         %{
           id: aid,
           tenant_id: tid,
           name: name,
           role_ids: decode_roles(role_ids),
           description: desc
         }}

      _ ->
        {:error, :not_found}
    end
  end

  @doc "List audiences for a tenant."
  @spec list(keyword()) :: [audience()]
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")

    case Store.raw_query(
           "SELECT id, tenant_id, name, role_ids, description FROM audiences WHERE tenant_id = ?1 ORDER BY name",
           [tenant_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [aid, tid, name, role_ids, desc] ->
          %{
            id: aid,
            tenant_id: tid,
            name: name,
            role_ids: decode_roles(role_ids),
            description: desc
          }
        end)

      _ ->
        []
    end
  end

  defp decode_roles(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, list} when is_list(list) -> list
      _ -> []
    end
  end

  defp decode_roles(_), do: []
end
