defmodule OptimalEngine.Knowledge.StoreTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Store

  setup do
    store_id = "test_store_#{:erlang.unique_integer([:positive])}"

    {:ok, pid} =
      Store.start_link(
        store_id: store_id,
        name: :"store_#{store_id}"
      )

    on_exit(fn ->
      if Process.alive?(pid), do: GenServer.stop(pid)
    end)

    %{store: pid}
  end

  test "assert and query round-trip", %{store: store} do
    :ok = Store.assert(store, "alice", "knows", "bob")
    {:ok, results} = Store.query(store, subject: "alice")
    assert {"alice", "knows", "bob"} in results
  end

  test "assert_many inserts batch", %{store: store} do
    triples = [
      {"a", "p1", "x"},
      {"b", "p2", "y"},
      {"c", "p3", "z"}
    ]

    :ok = Store.assert_many(store, triples)
    {:ok, count} = Store.count(store)
    assert count == 3
  end

  test "retract removes triple", %{store: store} do
    :ok = Store.assert(store, "alice", "knows", "bob")
    :ok = Store.retract(store, "alice", "knows", "bob")
    {:ok, count} = Store.count(store)
    assert count == 0
  end

  test "sparql returns not supported for ETS backend", %{store: store} do
    assert {:error, :sparql_not_supported} = Store.sparql(store, "SELECT ?s WHERE { ?s ?p ?o }")
  end

  test "count returns number of triples", %{store: store} do
    {:ok, count} = Store.count(store)
    assert count == 0

    :ok = Store.assert(store, "a", "b", "c")
    {:ok, count} = Store.count(store)
    assert count == 1
  end
end
