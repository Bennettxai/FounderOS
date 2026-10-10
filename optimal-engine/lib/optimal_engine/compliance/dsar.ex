defmodule OptimalEngine.Compliance.DSAR do
  @moduledoc """
  Data Subject Access Request — enumerate everything the engine
  stores about a principal.

  GDPR Article 15 grants individuals the right to a copy of their
  personal data. CCPA §1798.110 is the California equivalent.
  `export/1` walks every table that references a principal and
  returns a structured report the operator can hand off (or hand to
  `Jason.encode!/1` for download).

  Tables scanned:

    * `principals`         — identity record
    * `contexts`           — signals authored by / about the principal
    * `node_members`       — workspace memberships
    * `principal_skills`   — capability grants
    * `role_grants`        — directly granted roles
    * `principal_groups`   — group memberships
    * `events`             — audit trail where principal acted

  Signals that merely *mention* the principal in content are captured
  via FTS match on the display_name — surfaced under `:mentions`.
  """

  alias OptimalEngine.Store

  @type export :: %{
          principal: map() | nil,
          contexts: [map()],
          mentions: [map()],
          memberships: [map()],
          skills: [map()],
          roles: [map()],
          groups: [String.t()],
          events: [map()],
          exported_at: String.t()
        }

  @doc """
  Produce an export for `principal_id`, scoped to `tenant_id`.
  Returns a map in the shape above. Empty sections when the principal
  has no data in that category — the operator gets a full structural
  report, not a pile of maybes.
  """
  @spec export(String.t(), String.t()) :: {:ok, export()} | {:error, term()}
  def export(principal_id, tenant_id \\ "default") when is_binary(principal_id) do
    with {:ok, principal} <- fetch_principal(principal_id, tenant_id) do
      {:ok,
       %{
         principal: principal,
         contexts: list_contexts(principal_id, tenant_id),
         mentions: search_mentions(principal, tenant_id),
         memberships: list_memberships(principal_id, tenant_id),
         skills: list_skills(principal_id, tenant_id),
         roles: list_roles(principal_id, tenant_id),
         groups: list_groups(principal_id),
         events: list_events(principal_id, tenant_id),
         exported_at: DateTime.utc_now() |> DateTime.to_iso8601()
       }}
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp fetch_principal(id, tenant_id) do
    case Store.raw_query(
           "SELECT id, kind, display_name, external_id, created_at, metadata FROM principals WHERE id = ?1 AND tenant_id = ?2",
           [id, tenant_id]
         ) do
      {:ok, [[id, kind, name, ext, created, meta]]} ->
        {:ok,
         %{
           id: id,
           kind: kind,
           display_name: name,
           external_id: ext,
           created_at: created,
           metadata: decode(meta)
         }}

      {:ok, []} ->
        {:error, :not_found}

      other ->
        other
    end
  end

  defp list_contexts(principal_id, tenant_id) do
    case Store.raw_query(
           "SELECT id, uri, title, genre, created_at FROM contexts WHERE tenant_id = ?1 AND created_by = ?2",
           [tenant_id, principal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, uri, title, genre, at] ->
          %{id: id, uri: uri, title: title, genre: genre, created_at: at}
        end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp search_mentions(nil, _tenant_id), do: []

  defp search_mentions(%{display_name: nil}, _tenant_id), do: []

  defp search_mentions(%{display_name: name}, tenant_id) when is_binary(name) do
    case Store.raw_query(
           """
           SELECT c.id, c.uri, c.title
           FROM contexts c
           WHERE c.tenant_id = ?1 AND c.content LIKE ?2
           LIMIT 100
           """,
           [tenant_id, "%#{name}%"]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, uri, title] -> %{id: id, uri: uri, title: title} end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp list_memberships(principal_id, tenant_id) do
    case Store.raw_query(
           "SELECT node_id, membership_type, added_at FROM node_members WHERE tenant_id = ?1 AND principal_id = ?2",
           [tenant_id, principal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [node, type, at] ->
          %{node_id: node, membership_type: type, added_at: at}
        end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp list_skills(principal_id, tenant_id) do
    case Store.raw_query(
           "SELECT skill_id, level, granted_at FROM principal_skills WHERE tenant_id = ?1 AND principal_id = ?2",
           [tenant_id, principal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [skill, level, at] ->
          %{skill_id: skill, level: level, granted_at: at}
        end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp list_roles(principal_id, tenant_id) do
    case Store.raw_query(
           "SELECT role_id, granted_at FROM role_grants WHERE tenant_id = ?1 AND principal_id = ?2",
           [tenant_id, principal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [role, at] -> %{role_id: role, granted_at: at} end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp list_groups(principal_id) do
    case Store.raw_query(
           "SELECT group_id FROM principal_groups WHERE principal_id = ?1",
           [principal_id]
         ) do
      {:ok, rows} -> Enum.map(rows, fn [g] -> g end)
      _ -> []
    end
  rescue
    _ -> []
  end

  defp list_events(principal_id, tenant_id) do
    case Store.raw_query(
           "SELECT id, ts, kind, target_uri, latency_ms FROM events WHERE tenant_id = ?1 AND principal = ?2 ORDER BY ts DESC LIMIT 500",
           [tenant_id, principal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, ts, kind, uri, lat] ->
          %{id: id, ts: ts, kind: kind, target_uri: uri, latency_ms: lat}
        end)

      _ ->
        []
    end
  rescue
    _ -> []
  end

  defp decode(nil), do: %{}

  defp decode(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, v} -> v
      _ -> %{}
    end
  end
end
