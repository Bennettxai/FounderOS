defmodule OptimalEngine.API.WorkspaceAuthorizationTest do
  @moduledoc """
  Regression tests for workspace AUTHORIZATION (not just data scoping):

  1. `OptimalEngine.API.WorkspaceAuthPlug` — every workspace-parameterized
     endpoint (query/body/path) must enforce tenant ownership (404 on a
     foreign tenant's workspace) and the API key's `workspace_scope` (403 on
     scope mismatch), and set `conn.assigns[:workspace_id]`.
  2. POST /api/rag governed recall — the authorization envelope
     (actor/partitions/security_labels) is derived server-side from the
     authenticated principal; body-supplied values are stripped.
  """

  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router
  alias OptimalEngine.Auth.ApiKey
  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @opts Router.init([])

  defp request(method, path, headers \\ [], body \\ nil) do
    conn =
      case body do
        nil ->
          conn(method, path)

        b ->
          conn(method, path, Jason.encode!(b))
          |> put_req_header("content-type", "application/json")
      end

    conn = Enum.reduce(headers, conn, fn {k, v}, c -> put_req_header(c, k, v) end)
    Router.call(conn, @opts)
  end

  defp json_body(conn) do
    {:ok, body} = Jason.decode(conn.resp_body)
    body
  end

  # Force auth_required: true for the duration of a test.
  defp with_auth_required(fun) do
    original = Application.get_env(:optimal_engine, :auth, [])
    updated = Keyword.merge(original, auth_required: true, bcrypt_cost: 4)
    Application.put_env(:optimal_engine, :auth, updated)

    try do
      fun.()
    after
      Application.put_env(:optimal_engine, :auth, original)
    end
  end

  # Insert a workspace row directly — no filesystem provisioning side effects.
  defp insert_workspace(id, tenant_id, slug) do
    {:ok, _} =
      Store.raw_query(
        """
        INSERT OR IGNORE INTO workspaces (id, tenant_id, slug, name, status, metadata)
        VALUES (?1, ?2, ?3, ?4, 'active', '{}')
        """,
        [id, tenant_id, slug, "WS #{slug}"]
      )

    id
  end

  setup do
    suffix = System.unique_integer([:positive])

    # A foreign tenant with one registered workspace.
    foreign_tenant = "exampleorg-#{suffix}"
    {:ok, _} = Tenant.create(%{id: foreign_tenant, name: "ExampleOrg #{suffix}"})
    foreign_ws = insert_workspace("#{foreign_tenant}:secrets", foreign_tenant, "secrets")

    # Two registered workspaces under the default tenant.
    eng_ws = insert_workspace("default:eng-#{suffix}", "default", "eng-#{suffix}")
    other_ws = insert_workspace("default:other-#{suffix}", "default", "other-#{suffix}")

    {:ok,
     suffix: suffix,
     foreign_tenant: foreign_tenant,
     foreign_ws: foreign_ws,
     eng_ws: eng_ws,
     other_ws: other_ws}
  end

  describe "tenant ownership (anonymous caller, default tenant)" do
    test "another tenant's registered workspace returns 404", ctx do
      conn = request(:get, "/api/graph?workspace=#{ctx.foreign_ws}")
      assert conn.status == 404
      assert %{"error" => "workspace not found"} = json_body(conn)
    end

    test "unregistered workspace id with a foreign tenant prefix returns 404", ctx do
      conn = request(:get, "/api/graph?workspace=ghost-tenant-#{ctx.suffix}:eng")
      assert conn.status == 404
    end

    test "own tenant's registered workspace passes and sets the assign", ctx do
      conn = request(:get, "/api/graph?workspace=#{ctx.eng_ws}")
      assert conn.status == 200
      assert conn.assigns[:workspace_id] == ctx.eng_ws
    end

    test "implicit default workspace still works (single-workspace behavior)" do
      conn = request(:get, "/api/graph")
      assert conn.status == 200
      assert conn.assigns[:workspace_id] == "default"
    end

    test "bare unregistered workspace ids remain usable (compat)", ctx do
      conn = request(:get, "/api/graph?workspace=adhoc-ws-#{ctx.suffix}")
      assert conn.status == 200
    end

    test "body-parameterized endpoints are enforced", ctx do
      conn =
        request(:post, "/api/batch/import/signals", [], %{
          "workspace" => ctx.foreign_ws,
          "signals" => []
        })

      assert conn.status == 404
    end

    test "a body/query workspace mismatch cannot smuggle a foreign workspace", ctx do
      # Query names an allowed workspace; body names the foreign one. Both
      # must be authorized — handlers read either source.
      conn =
        request(:post, "/api/batch/import/signals?workspace=#{ctx.eng_ws}", [], %{
          "workspace" => ctx.foreign_ws,
          "signals" => []
        })

      assert conn.status == 404
    end

    test "path-parameterized /api/workspaces/:id endpoints are enforced", ctx do
      assert request(:get, "/api/workspaces/#{ctx.foreign_ws}").status == 404
      assert request(:get, "/api/workspaces/#{ctx.foreign_ws}/config").status == 404

      assert request(:patch, "/api/workspaces/#{ctx.foreign_ws}", [], %{"name" => "pwned"}).status ==
               404

      assert request(:post, "/api/workspaces/#{ctx.foreign_ws}/archive").status == 404

      # Own-tenant workspace by path still resolves.
      assert request(:get, "/api/workspaces/#{ctx.eng_ws}").status == 200
    end

    test "workspace-parameterized endpoint families all 404 on a foreign workspace", ctx do
      for path <- [
            "/api/graph/hubs?workspace=#{ctx.foreign_ws}",
            "/api/node/some-node?workspace=#{ctx.foreign_ws}",
            "/api/search?q=secret&workspace=#{ctx.foreign_ws}",
            "/api/grep?q=secret&workspace=#{ctx.foreign_ws}",
            "/api/signals/sig123?workspace=#{ctx.foreign_ws}",
            "/api/activity?workspace=#{ctx.foreign_ws}",
            "/api/health?workspace=#{ctx.foreign_ws}",
            "/api/optimal/graph?workspace=#{ctx.foreign_ws}",
            "/api/workspace?workspace=#{ctx.foreign_ws}",
            "/api/wiki?workspace=#{ctx.foreign_ws}",
            "/api/memory?workspace=#{ctx.foreign_ws}",
            "/api/subscriptions?workspace=#{ctx.foreign_ws}",
            "/api/batch/export/signals?workspace=#{ctx.foreign_ws}",
            "/api/batch/export/memories?workspace=#{ctx.foreign_ws}",
            "/api/batch/export/workspace?workspace=#{ctx.foreign_ws}",
            "/api/rag/stream?query=secret+stuff&workspace=#{ctx.foreign_ws}"
          ] do
        conn = request(:get, path)
        assert conn.status == 404, "#{path} returned #{conn.status}, expected 404"
      end
    end

    test "non-workspace endpoints are unaffected" do
      assert request(:get, "/api/status").status == 200
      assert request(:get, "/api/metrics").status == 200
      assert request(:get, "/api/architectures").status == 200
    end
  end

  describe "API key workspace_scope enforcement" do
    test "403 when the requested workspace is outside the key's scope", ctx do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: "default",
            name: "scoped key #{ctx.suffix}",
            workspace_scope: [ctx.eng_ws]
          })

        conn =
          request(:get, "/api/graph?workspace=#{ctx.other_ws}", [
            {"authorization", "Bearer #{token}"}
          ])

        assert conn.status == 403
        assert %{"error" => "workspace_scope_denied"} = json_body(conn)
      end)
    end

    test "200 when the requested workspace is inside the key's scope", ctx do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: "default",
            name: "scoped key ok #{ctx.suffix}",
            workspace_scope: [ctx.eng_ws]
          })

        conn =
          request(:get, "/api/graph?workspace=#{ctx.eng_ws}", [
            {"authorization", "Bearer #{token}"}
          ])

        assert conn.status == 200
        assert conn.assigns[:workspace_id] == ctx.eng_ws
      end)
    end

    test "wildcard scope never crosses the tenant boundary — foreign workspace is 404", ctx do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: ctx.foreign_tenant,
            name: "exampleorg wildcard #{ctx.suffix}",
            workspace_scope: ["*"]
          })

        # The default tenant's "default" workspace is foreign to this key's
        # tenant — wildcard workspace_scope must not grant cross-tenant reads.
        conn = request(:get, "/api/graph?workspace=default", [{"authorization", "Bearer #{token}"}])
        assert conn.status == 404

        # Its own tenant's workspace resolves.
        conn =
          request(:get, "/api/graph?workspace=#{ctx.foreign_ws}", [
            {"authorization", "Bearer #{token}"}
          ])

        assert conn.status == 200
      end)
    end
  end

  describe "POST /api/rag governed recall — server-derived authorization envelope" do
    test "body-supplied partitions/security_labels/actor are stripped" do
      conn =
        request(:post, "/api/rag", [], %{
          "query" => "what does the secret partition contain",
          "context_package" => true,
          "actor" => "intruder-actor",
          "partitions" => ["p-secret"],
          "security_labels" => ["restricted", "topsecret"]
        })

      assert conn.status == 200
      body = json_body(conn)
      assert body["source"] == "context_package"

      authorization =
        get_in(body, ["context_package", "retrieval_plan", "filters", "authorization"])

      # Anonymous caller carries no grants: the body-supplied envelope must
      # NOT appear — empty envelope, which the coordinator fails closed.
      assert authorization["allowed_partitions"] == []
      assert authorization["allowed_security_labels"] == []

      envelope = get_in(body, ["context_package", "authorization_envelope"])
      assert envelope["allowed_partitions"] == []
      assert envelope["allowed_security_labels"] == []
      refute envelope["actor_id"] == "intruder-actor"
      refute get_in(body, ["context_package", "requesting_actor_id"]) == "intruder-actor"
    end

    test "envelope comes from the API key's grants and principal, not the body", ctx do
      with_auth_required(fn ->
        # api_keys.principal_id has an enforced FK — the principal must exist.
        {:ok, _} =
          OptimalEngine.Identity.Principal.upsert(%{
            id: "alice-#{ctx.suffix}",
            tenant_id: "default",
            kind: :user,
            display_name: "Alice #{ctx.suffix}"
          })

        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: "default",
            name: "rag grants #{ctx.suffix}",
            principal_id: "alice-#{ctx.suffix}",
            workspace_scope: ["*"],
            metadata: %{
              "allowed_partitions" => ["team-a"],
              "allowed_security_labels" => ["internal"]
            }
          })

        conn =
          request(:post, "/api/rag", [{"authorization", "Bearer #{token}"}], %{
            "query" => "what does the secret partition contain",
            "context_package" => true,
            "actor" => "intruder-actor",
            "partitions" => ["p-secret"],
            "security_labels" => ["topsecret"]
          })

        assert conn.status == 200
        body = json_body(conn)

        authorization =
          get_in(body, ["context_package", "retrieval_plan", "filters", "authorization"])

        assert authorization["allowed_partitions"] == ["team-a"]
        assert authorization["allowed_security_labels"] == ["internal"]

        envelope = get_in(body, ["context_package", "authorization_envelope"])
        assert envelope["allowed_partitions"] == ["team-a"]
        assert envelope["allowed_security_labels"] == ["internal"]

        assert get_in(body, ["context_package", "requesting_actor_id"]) ==
                 "alice-#{ctx.suffix}"
      end)
    end
  end
end
