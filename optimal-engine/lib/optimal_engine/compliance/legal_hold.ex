defmodule OptimalEngine.Compliance.LegalHold do
  @moduledoc """
  Place + release legal holds on signals.

  A legal hold freezes retention — a held signal survives retention
  sweeps until the hold is explicitly released. Erasure refuses to
  run against held content unless an operator passes `:force`.

  Schema (`legal_holds` table, migration 010):

      id, tenant_id, signal_id, held_by, reason, placed_at, released_at
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type hold :: %{
          id: integer(),
          signal_id: String.t(),
          held_by: String.t(),
          reason: String.t(),
          placed_at: String.t(),
          released_at: String.t() | nil
        }

  @doc """
  Place a hold. Returns `{:ok, hold_id}`.
  """
  @spec place(String.t(), String.t(), String.t(), keyword()) ::
          {:ok, integer()} | {:error, term()}
  def place(signal_id, held_by, reason, opts \\ [])
      when is_binary(signal_id) and is_binary(held_by) and is_binary(reason) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())

    with {:ok, _} <-
           Store.raw_query(
             """
             INSERT INTO legal_holds (tenant_id, signal_id, held_by, reason)
             VALUES (?1, ?2, ?3, ?4)
             """,
             [tenant_id, signal_id, held_by, reason]
           ),
         {:ok, [[id]]} <- Store.raw_query("SELECT last_insert_rowid()", []) do
      {:ok, id}
    end
  end

  @doc "Release a hold by its numeric id."
  @spec release(integer()) :: :ok | {:error, term()}
  def release(hold_id) when is_integer(hold_id) do
    case Store.raw_query(
           "UPDATE legal_holds SET released_at = datetime('now') WHERE id = ?1 AND released_at IS NULL",
           [hold_id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc "List every active (un-released) hold for a tenant."
  @spec active(String.t()) :: [hold()]
  def active(tenant_id \\ Tenant.default_id()) do
    case Store.raw_query(
           "SELECT id, signal_id, held_by, reason, placed_at, released_at FROM legal_holds WHERE tenant_id = ?1 AND released_at IS NULL ORDER BY placed_at DESC",
           [tenant_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, sid, by, reason, at, rel] ->
          %{id: id, signal_id: sid, held_by: by, reason: reason, placed_at: at, released_at: rel}
        end)

      _ ->
        []
    end
  end

  @doc """
  Count active holds covering anything authored by `principal_id`.

  **Critical path for erasure + retention.** On DB error we return
  `{:error, :hold_check_failed}` rather than `{:ok, 0}` — a transient
  failure must never be reported as "no holds" because callers use the
  answer to gate destructive actions.
  """
  @spec count_holds_for_principal(String.t(), String.t()) ::
          {:ok, non_neg_integer()} | {:error, :hold_check_failed}
  def count_holds_for_principal(principal_id, tenant_id) do
    sql = """
    SELECT COUNT(*)
    FROM legal_holds h
    JOIN contexts c ON c.id = h.signal_id
    WHERE h.tenant_id = ?1
      AND h.released_at IS NULL
      AND c.tenant_id = ?1
      AND c.created_by = ?2
    """

    case Store.raw_query(sql, [tenant_id, principal_id]) do
      {:ok, [[n]]} when is_integer(n) -> {:ok, n}
      _ -> {:error, :hold_check_failed}
    end
  rescue
    _ -> {:error, :hold_check_failed}
  end

  @doc """
  `{:ok, true}` when the signal is currently held, `{:ok, false}` when
  it isn't, `{:error, :hold_check_failed}` when the DB query can't run.
  Retention + erasure must distinguish "not held" from "we don't know";
  previously both collapsed to `false`, letting transient failures
  allow deletion of held content.
  """
  @spec held?(String.t(), String.t()) :: {:ok, boolean()} | {:error, :hold_check_failed}
  def held?(signal_id, tenant_id \\ Tenant.default_id()) do
    case Store.raw_query(
           "SELECT COUNT(*) FROM legal_holds WHERE tenant_id = ?1 AND signal_id = ?2 AND released_at IS NULL",
           [tenant_id, signal_id]
         ) do
      {:ok, [[n]]} when is_integer(n) -> {:ok, n > 0}
      _ -> {:error, :hold_check_failed}
    end
  rescue
    _ -> {:error, :hold_check_failed}
  end
end
