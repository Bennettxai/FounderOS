defmodule OptimalEngine.API.PaginationTest do
  @moduledoc """
  Tests for pagination on every list endpoint.

  Coverage:
    - Default offset=0, limit=50
    - Custom offset + limit via query params
    - has_more=true when more items exist beyond the page
    - has_more=false at the last page
    - limit capped at 200
    - offset past total returns empty data with correct total
    - Pagination envelope always present alongside existing response keys
  """
  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Pagination
  alias OptimalEngine.API.Router

  @opts Router.init([])

  # ---------------------------------------------------------------------------
  # Helper: fire an HTTP request through the router
  # ---------------------------------------------------------------------------

  defp request(method, path, body \\ nil) do
    conn =
      case body do
        nil ->
          conn(method, path)

        b ->
          conn(method, path, Jason.encode!(b))
          |> put_req_header("content-type", "application/json")
      end

    conn =
      put_peer_data(conn, %{
        address: unique_loopback_address(),
        port: 0,
        ssl_cert: nil
      })

    Router.call(conn, @opts)
  end

  defp unique_loopback_address do
    n = System.unique_integer([:positive])
    {127, rem(div(n, 65_536), 256), rem(div(n, 256), 256), rem(n, 254) + 1}
  end

  defp decode(conn), do: Jason.decode!(conn.resp_body)

  # ---------------------------------------------------------------------------
  # Pagination helper unit tests
  # ---------------------------------------------------------------------------

  describe "Pagination.parse/1" do
    test "defaults to offset=0, limit=50" do
      conn = conn(:get, "/api/memory")
      assert {0, 50} == Pagination.parse(conn)
    end

    test "parses custom offset and limit" do
      conn = conn(:get, "/api/memory?offset=10&limit=25")
      assert {10, 25} == Pagination.parse(conn)
    end

    test "clamps negative offset to 0" do
      conn = conn(:get, "/api/memory?offset=-5&limit=10")
      assert {0, 10} == Pagination.parse(conn)
    end

    test "clamps limit > 200 to 200" do
      conn = conn(:get, "/api/memory?offset=0&limit=999")
      assert {0, 200} == Pagination.parse(conn)
    end

    test "clamps limit < 1 to 1" do
      conn = conn(:get, "/api/memory?offset=0&limit=0")
      assert {0, 1} == Pagination.parse(conn)
    end

    test "ignores non-integer values and falls back to defaults" do
      conn = conn(:get, "/api/memory?offset=abc&limit=xyz")
      assert {0, 50} == Pagination.parse(conn)
    end
  end

  describe "Pagination.wrap/4" do
    test "has_more=true when more items remain" do
      result = Pagination.wrap([1, 2, 3], 10, 0, 3)
      assert result.pagination.has_more == true
      assert result.pagination.total == 10
      assert result.pagination.offset == 0
      assert result.pagination.limit == 3
    end

    test "has_more=false when all items fit on the page" do
      result = Pagination.wrap([1, 2, 3], 3, 0, 50)
      assert result.pagination.has_more == false
    end

    test "has_more=false on the last page" do
      result = Pagination.wrap([3], 3, 2, 50)
      assert result.pagination.has_more == false
    end

    test "has_more=true on a middle page" do
      result = Pagination.wrap([2], 5, 2, 1)
      assert result.pagination.has_more == true
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/memory
  # ---------------------------------------------------------------------------

  describe "GET /api/memory pagination" do
    setup do
      # Seed a handful of memories so we can test pagination
      workspace_id = "pagination-test-mem-#{System.unique_integer([:positive])}"

      for i <- 1..5 do
        {:ok, _} =
          OptimalEngine.Memory.create(%{
            content: "Memory entry #{i} for pagination test",
            workspace_id: workspace_id
          })
      end

      {:ok, workspace_id: workspace_id}
    end

    test "returns pagination object alongside memories key", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "memories")
      assert Map.has_key?(body, "pagination")
      assert Map.has_key?(body["pagination"], "offset")
      assert Map.has_key?(body["pagination"], "limit")
      assert Map.has_key?(body["pagination"], "total")
      assert Map.has_key?(body["pagination"], "has_more")
    end

    test "default offset=0 limit=50 returns all 5 entries", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}")
      body = decode(conn)
      assert body["pagination"]["offset"] == 0
      assert body["pagination"]["limit"] == 50
      assert body["pagination"]["total"] == 5
      assert body["pagination"]["has_more"] == false
    end

    test "limit=2 returns 2 items with has_more=true", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}&limit=2")
      body = decode(conn)
      assert length(body["memories"]) == 2
      assert body["pagination"]["has_more"] == true
      assert body["pagination"]["total"] == 5
    end

    test "offset=4 limit=10 returns last item", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}&offset=4&limit=10")
      body = decode(conn)
      assert length(body["memories"]) == 1
      assert body["pagination"]["has_more"] == false
      assert body["pagination"]["total"] == 5
    end

    test "offset past total returns empty memories with correct total", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}&offset=100")
      body = decode(conn)
      assert body["memories"] == []
      assert body["pagination"]["total"] == 5
      assert body["pagination"]["has_more"] == false
    end

    test "limit capped at 200", %{workspace_id: ws} do
      conn = request(:get, "/api/memory?workspace=#{ws}&limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/wiki
  # ---------------------------------------------------------------------------

  describe "GET /api/wiki pagination" do
    setup do
      tenant = "default"
      workspace = "pagination-test-wiki-#{System.unique_integer([:positive])}"

      for i <- 1..3 do
        :ok =
          OptimalEngine.Wiki.Store.put(%OptimalEngine.Wiki.Page{
            tenant_id: tenant,
            workspace_id: workspace,
            slug: "wiki-page-#{i}-#{System.unique_integer([:positive])}",
            audience: "default",
            version: 1,
            frontmatter: %{},
            body: "Content #{i}"
          })
      end

      {:ok, tenant: tenant, workspace: workspace}
    end

    test "returns pagination alongside pages key", %{tenant: t, workspace: ws} do
      conn = request(:get, "/api/wiki?tenant=#{t}&workspace=#{ws}")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "pages")
      assert Map.has_key?(body, "pagination")
    end

    test "default returns all 3 pages", %{tenant: t, workspace: ws} do
      conn = request(:get, "/api/wiki?tenant=#{t}&workspace=#{ws}")
      body = decode(conn)
      assert body["pagination"]["total"] == 3
      assert body["pagination"]["has_more"] == false
    end

    test "limit=1 has_more=true", %{tenant: t, workspace: ws} do
      conn = request(:get, "/api/wiki?tenant=#{t}&workspace=#{ws}&limit=1")
      body = decode(conn)
      assert length(body["pages"]) == 1
      assert body["pagination"]["has_more"] == true
    end

    test "offset past total returns empty pages", %{tenant: t, workspace: ws} do
      conn = request(:get, "/api/wiki?tenant=#{t}&workspace=#{ws}&offset=100")
      body = decode(conn)
      assert body["pages"] == []
      assert body["pagination"]["total"] == 3
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/workspaces
  # ---------------------------------------------------------------------------

  describe "GET /api/workspaces pagination" do
    test "returns pagination alongside workspaces key" do
      conn = request(:get, "/api/workspaces")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "workspaces")
      assert Map.has_key?(body, "pagination")
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/workspaces?limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end

    test "offset=0 default present in pagination" do
      conn = request(:get, "/api/workspaces")
      body = decode(conn)
      assert body["pagination"]["offset"] == 0
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/subscriptions
  # ---------------------------------------------------------------------------

  describe "GET /api/subscriptions pagination" do
    test "returns pagination alongside subscriptions key" do
      conn = request(:get, "/api/subscriptions")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "subscriptions")
      assert Map.has_key?(body, "pagination")
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/subscriptions?limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/activity
  # ---------------------------------------------------------------------------

  describe "GET /api/activity pagination" do
    test "returns pagination alongside events key" do
      conn = request(:get, "/api/activity")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "events")
      assert Map.has_key?(body, "pagination")
    end

    test "pagination fields present" do
      conn = request(:get, "/api/activity?limit=5")
      body = decode(conn)
      pagination = body["pagination"]
      assert Map.has_key?(pagination, "offset")
      assert Map.has_key?(pagination, "limit")
      assert Map.has_key?(pagination, "total")
      assert Map.has_key?(pagination, "has_more")
      assert pagination["limit"] == 5
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/activity?limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/search
  # ---------------------------------------------------------------------------

  describe "GET /api/search pagination" do
    test "returns pagination alongside results key" do
      conn = request(:get, "/api/search?q=nonexistent-term-xyz-#{System.unique_integer()}")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "results")
      assert Map.has_key?(body, "pagination")
    end

    test "empty query returns empty results with pagination" do
      conn = request(:get, "/api/search")
      body = decode(conn)
      assert body["results"] == []
      assert body["pagination"]["total"] == 0
      assert body["pagination"]["has_more"] == false
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/search?q=test&limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/grep
  # ---------------------------------------------------------------------------

  describe "GET /api/grep pagination" do
    test "returns pagination alongside results key" do
      conn = request(:get, "/api/grep?q=test")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "results")
      assert Map.has_key?(body, "pagination")
    end

    test "pagination has all required fields" do
      conn = request(:get, "/api/grep?q=test&limit=5")
      body = decode(conn)
      pagination = body["pagination"]
      assert Map.has_key?(pagination, "offset")
      assert Map.has_key?(pagination, "limit")
      assert Map.has_key?(pagination, "total")
      assert Map.has_key?(pagination, "has_more")
      assert pagination["limit"] == 5
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/grep?q=test&limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/wiki/contradictions
  # ---------------------------------------------------------------------------

  describe "GET /api/wiki/contradictions pagination" do
    test "returns pagination alongside contradictions key" do
      conn = request(:get, "/api/wiki/contradictions")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "contradictions")
      assert Map.has_key?(body, "pagination")
    end

    test "count field still present (backward compat)" do
      conn = request(:get, "/api/wiki/contradictions")
      body = decode(conn)
      assert Map.has_key?(body, "count")
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/wiki/contradictions?limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end
  end

  # ---------------------------------------------------------------------------
  # GET /api/auth/keys
  # ---------------------------------------------------------------------------

  describe "GET /api/auth/keys pagination" do
    test "returns pagination alongside keys key" do
      conn = request(:get, "/api/auth/keys")
      assert conn.status == 200
      body = decode(conn)
      assert Map.has_key?(body, "keys")
      assert Map.has_key?(body, "pagination")
    end

    test "limit capped at 200" do
      conn = request(:get, "/api/auth/keys?limit=9999")
      body = decode(conn)
      assert body["pagination"]["limit"] == 200
    end

    test "offset=0 default" do
      conn = request(:get, "/api/auth/keys")
      body = decode(conn)
      assert body["pagination"]["offset"] == 0
    end
  end
end
