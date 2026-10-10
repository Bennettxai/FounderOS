defmodule OptimalEngine.Topology.PrincipalSkill do
  @moduledoc """
  Ties a principal (human OR agent) to a skill at a level, with optional
  evidence (a link, an id of a signal, free text).

  Levels:

    * `:novice`       — familiar; can ask good questions
    * `:intermediate` — gets work done; needs review on edge cases
    * `:expert`       — self-directed; can train juniors
    * `:lead`         — sets direction for the skill

  Same shape for humans and agents — the engine doesn't care which kind
  a skill-holder is. Capability lookups return a mix; the caller decides
  whether to filter by principal.kind.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type level :: :novice | :intermediate | :expert | :lead

  @allowed_levels [:novice, :intermediate, :expert, :lead]

  @doc """
  Grant a skill to a principal at a level. Idempotent on
  `(principal_id, skill_id)`: re-granting updates `level` + `evidence`.
  """
  @spec grant(String.t(), String.t(), keyword()) :: :ok | {:error, term()}
  def grant(principal_id, skill_id, opts \\ [])
      when is_binary(principal_id) and is_binary(skill_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    level = Keyword.get(opts, :level, :intermediate)
    evidence = Keyword.get(opts, :evidence)

    if level not in @allowed_levels do
      {:error, {:invalid_level, level}}
    else
      sql = """
      INSERT INTO principal_skills (tenant_id, principal_id, skill_id, level, evidence)
      VALUES (?1, ?2, ?3, ?4, ?5)
      ON CONFLICT(principal_id, skill_id) DO UPDATE SET
        level    = excluded.level,
        evidence = excluded.evidence
      """

      case Store.raw_query(sql, [tenant_id, principal_id, skill_id, Atom.to_string(level), evidence]) do
        {:ok, _} -> :ok
        other -> other
      end
    end
  end

  @doc "Revoke a skill from a principal."
  @spec revoke(String.t(), String.t(), keyword()) :: :ok | {:error, term()}
  def revoke(principal_id, skill_id, opts \\ [])
      when is_binary(principal_id) and is_binary(skill_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())

    case Store.raw_query(
           "DELETE FROM principal_skills WHERE principal_id = ?1 AND skill_id = ?2 AND tenant_id = ?3",
           [principal_id, skill_id, tenant_id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc """
  Returns every skill granted to a principal, optionally filtered by
  minimum level.
  """
  @spec skills_of(String.t(), keyword()) :: {:ok, [map()]}
  def skills_of(principal_id, opts \\ []) when is_binary(principal_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    min_level = Keyword.get(opts, :min_level)

    sql = """
    SELECT ps.skill_id, s.name, s.kind, ps.level, ps.evidence, ps.acquired_at
    FROM principal_skills ps
    JOIN skills s ON s.id = ps.skill_id
    WHERE ps.principal_id = ?1 AND ps.tenant_id = ?2
    ORDER BY s.name
    """

    case Store.raw_query(sql, [principal_id, tenant_id]) do
      {:ok, rows} ->
        rows =
          rows
          |> Enum.map(fn [sid, name, skind, level, evidence, acquired_at] ->
            %{
              skill_id: sid,
              name: name,
              kind: safe_atom(skind, nil),
              level: safe_atom(level, :intermediate),
              evidence: evidence,
              acquired_at: acquired_at
            }
          end)
          |> filter_by_min_level(min_level)

        {:ok, rows}

      other ->
        other
    end
  end

  @doc "Returns every principal who holds the skill at or above a minimum level."
  @spec principals_with_skill(String.t(), keyword()) :: {:ok, [map()]}
  def principals_with_skill(skill_id, opts \\ []) when is_binary(skill_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    min_level = Keyword.get(opts, :min_level)

    sql = """
    SELECT ps.principal_id, p.kind, p.display_name, ps.level, ps.evidence
    FROM principal_skills ps
    JOIN principals p ON p.id = ps.principal_id
    WHERE ps.skill_id = ?1 AND ps.tenant_id = ?2
    ORDER BY p.display_name
    """

    case Store.raw_query(sql, [skill_id, tenant_id]) do
      {:ok, rows} ->
        rows =
          rows
          |> Enum.map(fn [pid, pkind, pname, level, evidence] ->
            %{
              principal_id: pid,
              principal_kind: safe_atom(pkind, :user),
              display_name: pname,
              level: safe_atom(level, :intermediate),
              evidence: evidence
            }
          end)
          |> filter_by_min_level(min_level)

        {:ok, rows}

      other ->
        other
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp filter_by_min_level(rows, nil), do: rows

  defp filter_by_min_level(rows, min_level) when min_level in @allowed_levels do
    rank = fn level -> Enum.find_index(@allowed_levels, &(&1 == level)) || 0 end
    min_rank = rank.(min_level)
    Enum.filter(rows, fn %{level: l} -> rank.(l) >= min_rank end)
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
