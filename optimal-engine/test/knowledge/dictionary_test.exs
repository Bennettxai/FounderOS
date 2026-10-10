defmodule OptimalEngine.Knowledge.DictionaryTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Dictionary

  import Bitwise

  @tag_shift 60

  setup do
    name = :"dict_#{:erlang.unique_integer([:positive])}"
    {:ok, pid} = Dictionary.start_link(name: name)
    on_exit(fn -> if Process.alive?(pid), do: GenServer.stop(pid) end)
    %{dict: name, pid: pid}
  end

  # -----------------------------------------------------------------
  # encode/decode round-trip
  # -----------------------------------------------------------------

  describe "encode/decode round-trip" do
    test "URI term survives round-trip", %{dict: dict} do
      uri = "http://example.org/alice"
      {:ok, id} = Dictionary.encode(dict, uri)
      {:ok, ^uri} = Dictionary.decode(dict, id)
    end

    test "literal term survives round-trip", %{dict: dict} do
      lit = "hello world"
      {:ok, id} = Dictionary.encode(dict, lit)
      {:ok, ^lit} = Dictionary.decode(dict, id)
    end

    test "blank node term survives round-trip", %{dict: dict} do
      bnode = "_:b0"
      {:ok, id} = Dictionary.encode(dict, bnode)
      {:ok, ^bnode} = Dictionary.decode(dict, id)
    end

    test "inline integer round-trip (positive)", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, {:integer, 42})
      {:ok, {:integer, 42}} = Dictionary.decode(dict, id)
    end

    test "inline integer round-trip (negative)", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, {:integer, -1})
      {:ok, {:integer, -1}} = Dictionary.decode(dict, id)
    end

    test "inline integer round-trip (zero)", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, {:integer, 0})
      {:ok, {:integer, 0}} = Dictionary.decode(dict, id)
    end

    test "inline integer at max positive boundary", %{dict: _dict} do
      max = (1 <<< (@tag_shift - 1)) - 1
      {:ok, id} = Dictionary.encode(:unused, {:integer, max})
      {:ok, {:integer, ^max}} = Dictionary.decode(:unused, id)
    end

    test "inline integer at min negative boundary", %{dict: _dict} do
      min = -(1 <<< (@tag_shift - 1))
      {:ok, id} = Dictionary.encode(:unused, {:integer, min})
      {:ok, {:integer, ^min}} = Dictionary.decode(:unused, id)
    end

    test "integer overflow returns error", %{dict: _dict} do
      too_big = 1 <<< (@tag_shift - 1)
      {:error, {:integer_overflow, ^too_big}} = Dictionary.encode(:unused, {:integer, too_big})
    end
  end

  # -----------------------------------------------------------------
  # deduplication
  # -----------------------------------------------------------------

  describe "deduplication" do
    test "same string always yields the same ID", %{dict: dict} do
      uri = "http://example.org/bob"
      {:ok, id1} = Dictionary.encode(dict, uri)
      {:ok, id2} = Dictionary.encode(dict, uri)
      assert id1 == id2
    end

    test "different strings yield different IDs", %{dict: dict} do
      {:ok, id1} = Dictionary.encode(dict, "http://example.org/a")
      {:ok, id2} = Dictionary.encode(dict, "http://example.org/b")
      assert id1 != id2
    end

    test "size reflects unique terms only", %{dict: dict} do
      Dictionary.encode(dict, "http://example.org/x")
      Dictionary.encode(dict, "http://example.org/x")
      Dictionary.encode(dict, "http://example.org/y")
      assert Dictionary.size(dict) == 2
    end
  end

  # -----------------------------------------------------------------
  # type tagging
  # -----------------------------------------------------------------

  describe "type tagging" do
    test "URI gets tag 0x1", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, "http://example.org/thing")
      assert Dictionary.id_type(id) == :uri
      assert id >>> @tag_shift == 0x1
    end

    test "compact URI (prefix:local) gets tag 0x1", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, "foaf:name")
      assert Dictionary.id_type(id) == :uri
    end

    test "blank node gets tag 0x2", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, "_:genid42")
      assert Dictionary.id_type(id) == :bnode
      assert id >>> @tag_shift == 0x2
    end

    test "literal gets tag 0x3", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, "just a string")
      assert Dictionary.id_type(id) == :literal
      assert id >>> @tag_shift == 0x3
    end

    test "inline integer gets tag 0x4", %{dict: _dict} do
      {:ok, id} = Dictionary.encode(:unused, {:integer, 99})
      assert Dictionary.id_type(id) == :integer
      assert id >>> @tag_shift == 0x4
    end

    test "plain number-like string is a literal, not inline integer", %{dict: dict} do
      {:ok, id} = Dictionary.encode(dict, "42")
      assert Dictionary.id_type(id) == :literal
    end
  end

  # -----------------------------------------------------------------
  # inline integer encoding
  # -----------------------------------------------------------------

  describe "inline integer encoding" do
    test "positive integers have correct payload bits", %{dict: _dict} do
      {:ok, id} = Dictionary.encode(:unused, {:integer, 255})
      payload = id &&& (1 <<< @tag_shift) - 1
      assert payload == 255
    end

    test "negative integers use two's complement in 60 bits", %{dict: _dict} do
      {:ok, id} = Dictionary.encode(:unused, {:integer, -1})
      payload = id &&& (1 <<< @tag_shift) - 1
      # -1 in 60-bit two's complement = all 1s = (1 <<< 60) - 1
      assert payload == (1 <<< @tag_shift) - 1
    end

    test "inline integers don't consume dictionary slots", %{dict: dict} do
      Dictionary.encode(dict, {:integer, 1})
      Dictionary.encode(dict, {:integer, 2})
      Dictionary.encode(dict, {:integer, 1000})
      assert Dictionary.size(dict) == 0
    end
  end

  # -----------------------------------------------------------------
  # encode_many
  # -----------------------------------------------------------------

  describe "encode_many/2" do
    test "encodes a batch of mixed terms", %{dict: dict} do
      terms = [
        "http://example.org/s",
        "_:b1",
        "hello",
        {:integer, 7}
      ]

      {:ok, ids} = Dictionary.encode_many(dict, terms)
      assert length(ids) == 4

      assert Dictionary.id_type(Enum.at(ids, 0)) == :uri
      assert Dictionary.id_type(Enum.at(ids, 1)) == :bnode
      assert Dictionary.id_type(Enum.at(ids, 2)) == :literal
      assert Dictionary.id_type(Enum.at(ids, 3)) == :integer
    end

    test "batch results match individual encodes", %{dict: dict} do
      terms = ["http://a.com/1", "http://a.com/2", "some literal"]
      {:ok, batch_ids} = Dictionary.encode_many(dict, terms)

      individual_ids =
        Enum.map(terms, fn t ->
          {:ok, id} = Dictionary.encode(dict, t)
          id
        end)

      assert batch_ids == individual_ids
    end
  end

  # -----------------------------------------------------------------
  # decode edge cases
  # -----------------------------------------------------------------

  describe "decode edge cases" do
    test "decoding an unknown ID returns error", %{dict: dict} do
      fake_id = 0x1 <<< @tag_shift ||| 999_999
      assert {:error, :not_found} = Dictionary.decode(dict, fake_id)
    end
  end

  # -----------------------------------------------------------------
  # concurrent access safety
  # -----------------------------------------------------------------

  describe "concurrent access" do
    test "parallel encodes of the same term converge to one ID", %{dict: dict} do
      term = "http://example.org/concurrent-target"
      n = 100

      ids =
        1..n
        |> Task.async_stream(fn _ -> Dictionary.encode(dict, term) end,
          max_concurrency: 20,
          ordered: false
        )
        |> Enum.map(fn {:ok, {:ok, id}} -> id end)

      assert Enum.uniq(ids) |> length() == 1
      assert Dictionary.size(dict) >= 1
    end

    test "parallel encodes of distinct terms yield distinct IDs", %{dict: dict} do
      n = 200

      ids =
        1..n
        |> Task.async_stream(
          fn i -> Dictionary.encode(dict, "http://example.org/term/#{i}") end,
          max_concurrency: 20,
          ordered: false
        )
        |> Enum.map(fn {:ok, {:ok, id}} -> id end)

      assert Enum.uniq(ids) |> length() == n
    end

    test "parallel encode and decode don't race", %{dict: dict} do
      terms = for i <- 1..50, do: "http://example.org/race/#{i}"

      # Pre-encode all terms.
      pairs =
        Enum.map(terms, fn t ->
          {:ok, id} = Dictionary.encode(dict, t)
          {t, id}
        end)

      # Now hammer decode from many processes.
      results =
        pairs
        |> List.duplicate(10)
        |> List.flatten()
        |> Task.async_stream(
          fn {expected_term, id} ->
            {:ok, decoded} = Dictionary.decode(dict, id)
            {expected_term, decoded}
          end,
          max_concurrency: 30,
          ordered: false
        )
        |> Enum.map(fn {:ok, {expected, actual}} ->
          assert expected == actual
          :ok
        end)

      assert length(results) == 500
    end
  end

  # -----------------------------------------------------------------
  # classify_term (internal, but critical for correctness)
  # -----------------------------------------------------------------

  describe "classify_term/1" do
    test "absolute URI" do
      assert Dictionary.classify_term("http://example.org/x") == 0x1
      assert Dictionary.classify_term("https://x.com") == 0x1
      assert Dictionary.classify_term("urn:isbn:123") == 0x1
    end

    test "compact/prefixed URI" do
      assert Dictionary.classify_term("foaf:knows") == 0x1
      assert Dictionary.classify_term("rdf:type") == 0x1
    end

    test "blank node" do
      assert Dictionary.classify_term("_:b0") == 0x2
      assert Dictionary.classify_term("_:genid123") == 0x2
    end

    test "literal" do
      assert Dictionary.classify_term("hello") == 0x3
      assert Dictionary.classify_term("42") == 0x3
      assert Dictionary.classify_term("") == 0x3
    end
  end
end
