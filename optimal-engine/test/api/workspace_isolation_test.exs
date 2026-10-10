defmodule OptimalEngine.API.WorkspaceIsolationTest do
  @moduledoc """
  T3 workspace isolation: the graph/signal/activity endpoint families must
  never return data from a workspace other than the one requested
  (default "default"). Two workspaces with disjoint data — requesting A must
  never surface B's contexts, edges, entities, or audit events.
  """

  use ExUnit.Case, async: false

  import Plug.Test

  alias OptimalEngine.API.Router
  alias OptimalEngine.API.RateLimiter
  alias OptimalEngine.Context
  alias OptimalEngine.Store

  @opts Router.init([])

  defp request(method, path) do
    conn(method, path) |> Router.call(@opts)
  end

  defp json_body(conn) do
    {:ok, body} = Jason.decode(conn.resp_body)
    body
  end

  defp insert_context(id, workspace_id, attrs) do
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    ctx = %Context{
      id: id,
      uri: "optimal://nodes/#{attrs[:node]}/#{id}.md",
      type: :signal,
      path: "/tmp/#{id}.md",
      title: attrs[:title],
      content: attrs[:content] || "content for #{id}",
      l0_abstract: "SIGNAL | note | #{attrs[:title]}",
      l1_overview: "overview for #{id}",
      signal: nil,
      node: attrs[:node],
      sn_ratio: 0.7,
      entities: attrs[:entities] || [],
      created_at: now,
      modified_at: now,
      routed_to: [],
      workspace_id: workspace_id,
      metadata: %{}
    }

    :ok = Store.insert_context(ctx)
    ctx
  end

  setup do
    RateLimiter.reset()

    suffix = System.unique_integer([:positive])
    ws_a = "iso-ws-a-#{suffix}"
    ws_b = "iso-ws-b-#{suffix}"

    # Same node slug in both workspaces — slugs are reused across workspaces,
    # so node-keyed endpoints must ALSO filter by workspace.
    node = "iso-node-#{suffix}"

    # Two contexts per workspace sharing an entity, so shared-entity edges
    # exist inside each workspace.
    a1 =
      insert_context("isoa1#{suffix}", ws_a, %{
        title: "A1 brief",
        node: node,
        entities: ["EntityA#{suffix}"]
      })

    a2 =
      insert_context("isoa2#{suffix}", ws_a, %{
        title: "A2 brief",
        node: node,
        entities: ["EntityA#{suffix}"]
      })

    b1 =
      insert_context("isob1#{suffix}", ws_b, %{
        title: "B1 secret",
        node: node,
        entities: ["EntityB#{suffix}"]
      })

    b2 =
      insert_context("isob2#{suffix}", ws_b, %{
        title: "B2 secret",
        node: node,
        entities: ["EntityB#{suffix}"]
      })

    # Orphan in B (no node, no entities → no edges) — visible to the health
    # orphaned-contexts check only when scoped to ws_b.
    b3 =
      insert_context("isob3#{suffix}", ws_b, %{
        title: "B3 orphan secret",
        node: nil,
        entities: []
      })

    {:ok,
     ws_a: ws_a, ws_b: ws_b, node: node, a1: a1, a2: a2, b1: b1, b2: b2, b3: b3, suffix: suffix}
  end

  describe "GET /api/graph" do
    test "workspace A never returns workspace B contexts, edges, or entities", ctx do
      conn = request(:get, "/api/graph?workspace=#{ctx.ws_a}")
      assert conn.status == 200
      body = json_body(conn)

      assert body["workspace_id"] == ctx.ws_a

      context_ids = Enum.map(body["contexts"], & &1["id"])
      assert ctx.a1.id in context_ids
      assert ctx.a2.id in context_ids
      refute ctx.b1.id in context_ids
      refute ctx.b2.id in context_ids

      edge_endpoints =
        Enum.flat_map(body["edges"], fn e -> [e["source"], e["target"]] end)

      refute ctx.b1.id in edge_endpoints
      refute ctx.b2.id in edge_endpoints

      entity_names = Enum.map(body["entities"], & &1["name"])
      assert "EntityA#{ctx.suffix}" in entity_names
      refute "EntityB#{ctx.suffix}" in entity_names
    end

    test "workspace B sees its own contexts but none of A's", ctx do
      body = request(:get, "/api/graph?workspace=#{ctx.ws_b}") |> json_body()

      context_ids = Enum.map(body["contexts"], & &1["id"])
      assert ctx.b1.id in context_ids
      refute ctx.a1.id in context_ids

      titles = Enum.map(body["contexts"], & &1["title"])
      refute "A1 brief" in titles
    end

    test "default workspace (no param) does not leak either workspace", ctx do
      conn = request(:get, "/api/graph")
      assert conn.status == 200
      body = json_body(conn)

      context_ids = Enum.map(body["contexts"], & &1["id"])
      refute ctx.a1.id in context_ids
      refute ctx.b1.id in context_ids
    end
  end

  describe "GET /api/node/:node_id" do
    test "same node slug across workspaces only returns the requested workspace", ctx do
      body = request(:get, "/api/node/#{ctx.node}?workspace=#{ctx.ws_a}") |> json_body()

      ids = Enum.map(body["contexts"], & &1["id"])
      assert ctx.a1.id in ids
      refute ctx.b1.id in ids
      refute ctx.b2.id in ids
    end
  end

  describe "GET /api/signals/:id" do
    test "signal from another workspace returns 404 (IDOR fix)", ctx do
      conn = request(:get, "/api/signals/#{ctx.b1.id}?workspace=#{ctx.ws_a}")
      assert conn.status == 404
    end

    test "signal resolves in its own workspace", ctx do
      conn = request(:get, "/api/signals/#{ctx.b1.id}?workspace=#{ctx.ws_b}")
      assert conn.status == 200
      body = json_body(conn)
      assert body["id"] == ctx.b1.id
      assert body["title"] == "B1 secret"
    end

    test "without a workspace param, non-default signals are not exposed", ctx do
      conn = request(:get, "/api/signals/#{ctx.b1.id}")
      assert conn.status == 404
    end
  end

  describe "GET /api/activity" do
    test "events are filtered to the requested workspace", ctx do
      kind = "iso-test-#{ctx.suffix}"

      {:ok, _} =
        Store.raw_query(
          "INSERT INTO events (tenant_id, principal, kind, target_uri, workspace_id) VALUES ('default', 'tester', ?1, 'optimal://a', ?2)",
          [kind, ctx.ws_a]
        )

      {:ok, _} =
        Store.raw_query(
          "INSERT INTO events (tenant_id, principal, kind, target_uri, workspace_id) VALUES ('default', 'tester', ?1, 'optimal://b', ?2)",
          [kind, ctx.ws_b]
        )

      body = request(:get, "/api/activity?workspace=#{ctx.ws_a}&kind=#{kind}") |> json_body()

      assert body["workspace_id"] == ctx.ws_a
      assert length(body["events"]) == 1
      assert hd(body["events"])["target_uri"] == "optimal://a"
    end
  end

  describe "GET /api/optimal/*" do
    test "/api/optimal/graph entities are workspace-scoped", ctx do
      body = request(:get, "/api/optimal/graph?workspace=#{ctx.ws_a}") |> json_body()

      names = Enum.map(body["entities"], & &1["name"])
      assert "EntityA#{ctx.suffix}" in names
      refute "EntityB#{ctx.suffix}" in names
    end

    test "/api/optimal/nodes falls back to workspace-scoped contexts", ctx do
      body = request(:get, "/api/optimal/nodes?workspace=#{ctx.ws_a}") |> json_body()

      node_entry = Enum.find(body["nodes"], &(&1["slug"] == ctx.node))

      # The nodes table has no row for this slug in ws_a (fallback path), so
      # when the entry exists its count must only reflect ws_a signals.
      if node_entry do
        assert node_entry["signal_count"] == 2
      end
    end

    test "/api/optimal/nodes/:slug/files only lists the requested workspace's signals", ctx do
      conn = request(:get, "/api/optimal/nodes/#{ctx.node}/files?workspace=#{ctx.ws_b}")
      assert conn.status == 200
      body = json_body(conn)

      names = Enum.map(body["files"], & &1["name"])
      assert "B1 secret" in names
      refute "A1 brief" in names
    end
  end

  describe "GET /api/graph analysis endpoints" do
    test "hubs/triangles/clusters/reflect accept workspace and stay scoped", ctx do
      for path <- [
            "/api/graph/hubs?workspace=#{ctx.ws_a}",
            "/api/graph/triangles?workspace=#{ctx.ws_a}&limit=5",
            "/api/graph/clusters?workspace=#{ctx.ws_a}",
            "/api/graph/reflect?workspace=#{ctx.ws_a}&min=1"
          ] do
        conn = request(:get, path)
        assert conn.status == 200, "#{path} returned #{conn.status}"
        refute conn.resp_body =~ "EntityB#{ctx.suffix}"
        refute conn.resp_body =~ ctx.b1.id
      end
    end
  end

  describe "GET /api/health" do
    test "diagnostics never include another workspace's context ids or titles", ctx do
      conn = request(:get, "/api/health?full=true&workspace=#{ctx.ws_a}")
      assert conn.status == 200
      refute conn.resp_body =~ "B1 secret"
      refute conn.resp_body =~ ctx.b1.id
      # b3 is an orphaned context in ws_b — the unscoped orphan check used to
      # list it tenant-wide; the scoped check must not.
      refute conn.resp_body =~ ctx.b3.id
      refute conn.resp_body =~ "B3 orphan secret"
    end

    test "scoped diagnostics still see their own workspace's orphans", ctx do
      conn = request(:get, "/api/health?full=true&workspace=#{ctx.ws_b}")
      assert conn.status == 200
      assert conn.resp_body =~ ctx.b3.id
    end
  end
end
