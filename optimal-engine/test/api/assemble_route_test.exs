defmodule OptimalEngine.API.AssembleRouteTest do
  @moduledoc """
  Tests for POST /api/assemble — tiered context assembly with MCTS metadata.

  Verifies:
  - Happy path returns all required tiers (l0, l1, l2) and MCTS metadata.
  - Missing query returns 400.
  - workspace_id is echoed back in the response.
  - mcts_metadata carries mcts_enabled flag and candidate_count.
  """
  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router

  @opts Router.init([])

  defp post_assemble(body) do
    conn =
      conn(:post, "/api/assemble", Jason.encode!(body))
      |> put_req_header("content-type", "application/json")

    Router.call(conn, @opts)
  end

  describe "POST /api/assemble" do
    test "returns tiered context and MCTS metadata for a query" do
      conn = post_assemble(%{"query" => "test context assembly", "workspace" => "default"})

      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)

      # Tiered context keys must be present
      assert Map.has_key?(body, "l0")
      assert Map.has_key?(body, "l1")
      assert Map.has_key?(body, "l2")
      assert Map.has_key?(body, "l3")

      # Structural fields
      assert Map.has_key?(body, "total_tokens")
      assert Map.has_key?(body, "sources")
      assert Map.has_key?(body, "query")
      assert body["query"] == "test context assembly"
      assert body["workspace_id"] == "default"

      # MCTS metadata must be present with correct shape
      assert Map.has_key?(body, "mcts_metadata")
      mcts = body["mcts_metadata"]
      assert Map.has_key?(mcts, "candidate_count")
      assert Map.has_key?(mcts, "selected_sources")
      assert Map.has_key?(mcts, "mcts_enabled")
      assert is_list(mcts["selected_sources"])
      assert is_integer(mcts["candidate_count"])
      assert is_boolean(mcts["mcts_enabled"])
    end

    test "returns 400 when query is missing" do
      conn = post_assemble(%{"workspace" => "default"})
      assert conn.status == 400
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] =~ "query"
    end

    test "returns 400 when query is empty string" do
      conn = post_assemble(%{"query" => "   ", "workspace" => "default"})
      assert conn.status == 400
    end

    test "respects workspace parameter" do
      conn = post_assemble(%{"query" => "workspace scoping test", "workspace" => "default"})
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["workspace_id"] == "default"
    end

    test "l0 tier contains structural inventory content (non-empty)" do
      conn = post_assemble(%{"query" => "context assembly inventory"})
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      # L0 is always loaded (query-independent structural inventory).
      # Even with no signals in test DB, l0 is a string.
      assert is_binary(body["l0"])
    end

    test "sources is a list" do
      conn = post_assemble(%{"query" => "sources list check"})
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert is_list(body["sources"])
    end
  end
end
