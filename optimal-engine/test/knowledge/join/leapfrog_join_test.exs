defmodule OptimalEngine.Knowledge.Join.LeapfrogJoinTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Join.LeapfrogJoin
  alias OptimalEngine.Knowledge.Join.Iterator.ETS, as: ETSIter

  # Convenience: wrap a sorted list into a {module, state} iterator pair
  defp iter(values), do: {ETSIter, ETSIter.new(Enum.sort(values))}

  describe "run/1 with zero iterators" do
    test "returns empty list" do
      assert LeapfrogJoin.run([]) == []
    end
  end

  describe "run/1 with a single iterator" do
    test "returns all values from the iterator" do
      result = LeapfrogJoin.run([iter(["a", "b", "c"])])
      assert result == ["a", "b", "c"]
    end

    test "returns empty list for an empty iterator" do
      result = LeapfrogJoin.run([iter([])])
      assert result == []
    end

    test "returns single-element list" do
      result = LeapfrogJoin.run([iter(["x"])])
      assert result == ["x"]
    end
  end

  describe "run/1 with two iterators" do
    test "intersects two overlapping sets" do
      result = LeapfrogJoin.run([iter(["a", "b", "c", "d"]), iter(["b", "c", "e"])])
      assert result == ["b", "c"]
    end

    test "returns empty when no overlap" do
      result = LeapfrogJoin.run([iter(["a", "b"]), iter(["c", "d"])])
      assert result == []
    end

    test "returns single overlap" do
      result = LeapfrogJoin.run([iter(["a", "b", "c"]), iter(["c", "d", "e"])])
      assert result == ["c"]
    end

    test "handles identical sets" do
      result = LeapfrogJoin.run([iter(["a", "b", "c"]), iter(["a", "b", "c"])])
      assert result == ["a", "b", "c"]
    end

    test "handles one empty iterator" do
      result = LeapfrogJoin.run([iter(["a", "b", "c"]), iter([])])
      assert result == []
    end

    test "handles large numeric overlap" do
      vals_a = Enum.map(1..100, &Integer.to_string/1) |> Enum.sort()
      vals_b = Enum.map(50..150, &Integer.to_string/1) |> Enum.sort()
      result = LeapfrogJoin.run([iter(vals_a), iter(vals_b)])
      expected = MapSet.intersection(MapSet.new(vals_a), MapSet.new(vals_b)) |> Enum.sort()
      assert Enum.sort(result) == expected
    end
  end

  describe "run/1 with three iterators" do
    test "finds three-way intersection" do
      result =
        LeapfrogJoin.run([
          iter(["a", "b", "c", "d"]),
          iter(["b", "c", "d", "e"]),
          iter(["a", "c", "e", "f"])
        ])

      assert result == ["c"]
    end

    test "returns empty when one iterator has no overlap" do
      result =
        LeapfrogJoin.run([
          iter(["a", "b", "c"]),
          iter(["b", "c"]),
          iter(["x", "y", "z"])
        ])

      assert result == []
    end

    test "returns multiple values when all three share them" do
      result =
        LeapfrogJoin.run([
          iter(["alice", "bob", "carol", "dave"]),
          iter(["alice", "bob", "carol"]),
          iter(["alice", "bob"])
        ])

      assert result == ["alice", "bob"]
    end
  end

  describe "run/1 result ordering" do
    test "always returns values in sorted ascending order" do
      result =
        LeapfrogJoin.run([
          iter(["carol", "alice", "bob"]),
          iter(["bob", "carol", "alice"])
        ])

      assert result == Enum.sort(result)
    end
  end
end
