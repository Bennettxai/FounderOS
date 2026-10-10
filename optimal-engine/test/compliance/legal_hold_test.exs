defmodule OptimalEngine.Compliance.LegalHoldTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Compliance.LegalHold
  alias OptimalEngine.Store

  defp seed_context do
    id = "hold-ctx-#{System.unique_integer([:positive])}"

    {:ok, _} =
      Store.raw_query(
        """
        INSERT INTO contexts (id, tenant_id, uri, title, content, genre, node)
        VALUES (?1, 'default', ?2, 't', 'body', 'note', 'person-operator')
        """,
        [id, "optimal://holds/#{id}"]
      )

    id
  end

  test "place/4 then released?/2 round-trip" do
    sig_id = seed_context()

    {:ok, hold_id} = LegalHold.place(sig_id, "user:legal", "litigation hold")
    assert is_integer(hold_id)
    assert {:ok, true} = LegalHold.held?(sig_id)

    :ok = LegalHold.release(hold_id)
    assert {:ok, false} = LegalHold.held?(sig_id)
  end

  test "active/1 lists open holds" do
    sig_id = seed_context()
    {:ok, _} = LegalHold.place(sig_id, "user:legal", "active")

    holds = LegalHold.active("default")
    assert Enum.any?(holds, &(&1.signal_id == sig_id))
  end

  test "count_holds_for_principal/2 returns 0 when none exist" do
    assert {:ok, 0} = LegalHold.count_holds_for_principal("user:nobody-here", "default")
  end
end
