defmodule OptimalEngine.Store.TransactionTest do
  @moduledoc """
  Regression tests for `OptimalEngine.Store.transaction/2` — the single
  GenServer-call BEGIN IMMEDIATE..COMMIT/ROLLBACK primitive that FactPromoter
  atomic promotion and ActiveMemoryPool atomic mutations build on.

  Contract under test:
  - fun returns `{:ok, result}`  -> COMMIT, returns `{:ok, result}`
  - fun returns `{:error, rsn}`  -> ROLLBACK, returns `{:error, rsn}`
  - fun raises/throws            -> ROLLBACK, exception propagates to caller,
                                    Store GenServer survives
  - any other return             -> ROLLBACK (fail closed)
  - statements run via txn_query/txn_execute on the bound connection; reads
    inside the txn see the txn's own uncommitted writes
  - the whole fun runs in ONE GenServer call, so concurrent
    read-check-write sequences are serialized (no lost updates)
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Store

  setup do
    table = "txn_test_#{System.unique_integer([:positive])}"

    :ok =
      Store.raw_execute("CREATE TABLE IF NOT EXISTS #{table} (id INTEGER PRIMARY KEY, val TEXT)")

    on_exit(fn -> Store.raw_execute("DROP TABLE IF EXISTS #{table}") end)

    {:ok, table: table}
  end

  defp count_rows(table) do
    {:ok, [[n]]} = Store.raw_query("SELECT COUNT(*) FROM #{table}")
    n
  end

  describe "commit path" do
    test "commits writes and returns the fun's {:ok, result}", %{table: table} do
      result =
        Store.transaction(fn txn ->
          {:ok, 1} =
            Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (?1, ?2)", [1, "a"])

          {:ok, 1} =
            Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (?1, ?2)", [2, "b"])

          {:ok, :wrote_two}
        end)

      assert result == {:ok, :wrote_two}
      assert count_rows(table) == 2
    end

    test "txn_query sees the transaction's own uncommitted writes", %{table: table} do
      {:ok, seen_inside} =
        Store.transaction(fn txn ->
          {:ok, 1} = Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (1, 'x')")
          {:ok, [[n]]} = Store.txn_query(txn, "SELECT COUNT(*) FROM #{table}")
          {:ok, n}
        end)

      assert seen_inside == 1
    end

    test "txn_execute reports affected row count (0 for no-match UPDATE)", %{table: table} do
      {:ok, changes} =
        Store.transaction(fn txn ->
          Store.txn_execute(txn, "UPDATE #{table} SET val = 'y' WHERE id = ?1", [999])
        end)

      assert changes == 0
    end
  end

  describe "rollback paths" do
    test "rolls back on {:error, reason} and propagates the error unchanged", %{table: table} do
      result =
        Store.transaction(fn txn ->
          {:ok, 1} = Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (1, 'a')")
          {:error, :business_rule_violated}
        end)

      assert result == {:error, :business_rule_violated}
      assert count_rows(table) == 0
    end

    test "rolls back and reraises in the caller when the fun raises", %{table: table} do
      store_pid = Process.whereis(Store)

      assert_raise RuntimeError, "boom", fn ->
        Store.transaction(fn txn ->
          {:ok, 1} = Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (1, 'a')")
          raise "boom"
        end)
      end

      assert count_rows(table) == 0
      # Store GenServer must survive caller bugs and stay usable.
      assert Process.whereis(Store) == store_pid
      assert {:ok, _} = Store.raw_query("SELECT 1")
    end

    test "rolls back and propagates a throw from the fun", %{table: table} do
      thrown =
        catch_throw(
          Store.transaction(fn txn ->
            {:ok, 1} = Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (1, 'a')")
            throw(:bail)
          end)
        )

      assert thrown == :bail
      assert count_rows(table) == 0
    end

    test "fails closed (rollback) on a malformed fun return", %{table: table} do
      result =
        Store.transaction(fn txn ->
          {:ok, 1} = Store.txn_execute(txn, "INSERT INTO #{table} (id, val) VALUES (1, 'a')")
          :not_a_tagged_tuple
        end)

      assert result == {:error, {:invalid_transaction_return, :not_a_tagged_tuple}}
      assert count_rows(table) == 0
    end
  end

  describe "atomicity / serialization" do
    test "concurrent read-check-write transactions do not lose updates", %{table: table} do
      :ok = Store.raw_execute("INSERT INTO #{table} (id, val) VALUES (1, '0')")
      workers = 8

      1..workers
      |> Enum.map(fn _ ->
        Task.async(fn ->
          Store.transaction(fn txn ->
            {:ok, [[val]]} = Store.txn_query(txn, "SELECT val FROM #{table} WHERE id = 1")
            next = Integer.to_string(String.to_integer(val) + 1)

            {:ok, 1} =
              Store.txn_execute(txn, "UPDATE #{table} SET val = ?1 WHERE id = 1", [next])

            {:ok, next}
          end)
        end)
      end)
      |> Enum.each(fn task -> assert {:ok, _} = Task.await(task, 30_000) end)

      {:ok, [[final]]} = Store.raw_query("SELECT val FROM #{table} WHERE id = 1")
      assert final == Integer.to_string(workers)
    end
  end
end
