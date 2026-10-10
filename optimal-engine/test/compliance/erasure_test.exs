defmodule OptimalEngine.Compliance.ErasureTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Compliance.{Erasure, LegalHold}
  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.Store

  defp seed_subject do
    suffix = System.unique_integer([:positive])
    id = "user:erase-#{suffix}@test"

    {:ok, _} =
      Principal.upsert(%{id: id, kind: :user, display_name: "Erase Subject #{suffix}"})

    id
  end

  defp seed_context_for(principal_id) do
    ctx_id = "erase-ctx-#{System.unique_integer([:positive])}"

    {:ok, _} =
      Store.raw_query(
        """
        INSERT INTO contexts (id, tenant_id, uri, title, content, created_by, genre, node)
        VALUES (?1, 'default', ?2, 't', 'body', ?3, 'note', 'person-operator')
        """,
        [ctx_id, "optimal://erase/#{ctx_id}", principal_id]
      )

    ctx_id
  end

  test "preview/2 reports counts without touching anything" do
    pid = seed_subject()
    _ = seed_context_for(pid)

    {:ok, preview} = Erasure.preview(pid)
    assert preview.counts.contexts >= 1
  end

  test "erase/2 cascades delete + pseudonymizes events" do
    pid = seed_subject()
    _ = seed_context_for(pid)

    # Seed an event to verify the audit trail gets pseudonymized.
    Store.raw_query(
      """
      INSERT INTO events (tenant_id, principal, kind, target_uri)
      VALUES ('default', ?1, 'test', 'optimal://erase/audit')
      """,
      [pid]
    )

    {:ok, report} = Erasure.erase(pid)
    assert report.principal_id == pid
    refute report.forced?

    # After erasure, the principal row is gone
    {:ok, rows} =
      Store.raw_query("SELECT id FROM principals WHERE id = ?1", [pid])

    assert rows == []

    # Events with the principal literal are replaced with the pseudonym
    {:ok, [[remaining]]} =
      Store.raw_query(
        "SELECT COUNT(*) FROM events WHERE principal = ?1",
        [pid]
      )

    assert remaining == 0
  end

  test "erase/2 refuses while a legal hold is active (unless :force)" do
    pid = seed_subject()
    ctx_id = seed_context_for(pid)

    {:ok, _} = LegalHold.place(ctx_id, "user:legal", "preservation")

    assert {:error, {:legal_hold_active, n}} = Erasure.erase(pid)
    assert n >= 1

    # --force proceeds anyway
    assert {:ok, report} = Erasure.erase(pid, force: true)
    assert report.forced?
  end
end
