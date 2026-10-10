defmodule OptimalEngine.API.AuthPlugTest do
  @moduledoc """
  Integration tests for the AuthPlug behaviour within the Router pipeline.

  Strategy:
  - Tests that need auth_required: true temporarily override the config.
  - Tests that need auth_required: false use the default (test.exs sets false).
  - We call the Router (which now includes AuthPlug in its pipeline) via Plug.Test.
  - GET /api/status is the simplest route to probe — it always returns 200 when auth passes.
  """

  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router
  alias OptimalEngine.Auth.ApiKey

  @opts Router.init([])
  @tenant_id "default"

  # Helper: call the router and return the conn.
  defp call(method, path, headers \\ [], body \\ nil) do
    conn =
      case body do
        nil ->
          conn(method, path)

        b ->
          conn(method, path, Jason.encode!(b))
          |> put_req_header("content-type", "application/json")
      end
      |> put_peer_data(%{
        address: unique_loopback_address(),
        port: 0,
        ssl_cert: nil
      })

    conn = Enum.reduce(headers, conn, fn {k, v}, c -> put_req_header(c, k, v) end)
    Router.call(conn, @opts)
  end

  defp unique_loopback_address do
    n = System.unique_integer([:positive])
    {127, rem(div(n, 65_536), 256), rem(div(n, 256), 256), rem(n, 254) + 1}
  end

  # Temporarily force auth_required: true for the duration of a test.
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

  describe "auth_required: false (dev default)" do
    test "GET /api/status without a token returns 200" do
      conn = call(:get, "/api/status")
      assert conn.status == 200
    end

    test "assigns current_tenant = 'default' in anonymous mode" do
      conn = call(:get, "/api/status")
      assert conn.assigns[:current_tenant] == "default"
    end

    test "assigns current_principal = :anonymous in anonymous mode" do
      conn = call(:get, "/api/status")
      assert conn.assigns[:current_principal] == :anonymous
    end

    test "assigns current_api_key = nil in anonymous mode" do
      conn = call(:get, "/api/status")
      assert conn.assigns[:current_api_key] == nil
    end

    test "a valid token is still accepted when auth_required is false" do
      {:ok, %{key: token, id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "plug ok test"})
      conn = call(:get, "/api/status", [{"authorization", "Bearer #{token}"}])
      assert conn.status == 200
      assert conn.assigns[:current_api_key].id == id
    end
  end

  describe "auth_required: true" do
    test "returns 401 with missing_api_key when no token is sent" do
      with_auth_required(fn ->
        conn = call(:get, "/api/status")
        assert conn.status == 401
        assert {:ok, %{"error" => "missing_api_key"}} = Jason.decode(conn.resp_body)
      end)
    end

    test "returns 401 with invalid_api_key for a garbage token" do
      with_auth_required(fn ->
        conn = call(:get, "/api/status", [{"authorization", "Bearer not-a-real-key"}])
        assert conn.status == 401
        assert {:ok, %{"error" => "invalid_api_key"}} = Jason.decode(conn.resp_body)
      end)
    end

    test "returns 401 with api_key_revoked for a revoked key" do
      with_auth_required(fn ->
        {:ok, %{id: id, key: token}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "plug revoke"})
        :ok = ApiKey.revoke(id)
        conn = call(:get, "/api/status", [{"authorization", "Bearer #{token}"}])
        assert conn.status == 401
        assert {:ok, %{"error" => "api_key_revoked"}} = Jason.decode(conn.resp_body)
      end)
    end

    test "returns 401 with api_key_expired for an expired key" do
      with_auth_required(fn ->
        past = "2000-01-01T00:00:00Z"

        {:ok, %{key: token}} =
          ApiKey.mint(%{tenant_id: @tenant_id, name: "plug expired", expires_at: past})

        conn = call(:get, "/api/status", [{"authorization", "Bearer #{token}"}])
        assert conn.status == 401
        assert {:ok, %{"error" => "api_key_expired"}} = Jason.decode(conn.resp_body)
      end)
    end

    test "accepts a valid token in Authorization: Bearer header" do
      with_auth_required(fn ->
        {:ok, %{key: token, id: id}} =
          ApiKey.mint(%{tenant_id: @tenant_id, name: "bearer ok"})

        conn = call(:get, "/api/status", [{"authorization", "Bearer #{token}"}])
        assert conn.status == 200
        assert conn.assigns[:current_api_key].id == id
      end)
    end

    test "accepts a valid token in X-API-Key header" do
      with_auth_required(fn ->
        {:ok, %{key: token, id: id}} =
          ApiKey.mint(%{tenant_id: @tenant_id, name: "x-api-key ok"})

        conn = call(:get, "/api/status", [{"x-api-key", token}])
        assert conn.status == 200
        assert conn.assigns[:current_api_key].id == id
      end)
    end

    test "Bearer header takes precedence over X-API-Key" do
      with_auth_required(fn ->
        {:ok, %{key: token1, id: id1}} =
          ApiKey.mint(%{tenant_id: @tenant_id, name: "bearer wins"})

        {:ok, %{key: token2}} =
          ApiKey.mint(%{tenant_id: @tenant_id, name: "x-api-key loses"})

        conn =
          call(:get, "/api/status", [
            {"authorization", "Bearer #{token1}"},
            {"x-api-key", token2}
          ])

        assert conn.status == 200
        assert conn.assigns[:current_api_key].id == id1
      end)
    end
  end

  describe "workspace scope enforcement" do
    test "returns 403 when key workspace_scope does not match conn.assigns[:workspace_id]" do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: @tenant_id,
            name: "scoped key",
            workspace_scope: ["default:engineering"]
          })

        # We need to inject workspace_id into the conn BEFORE the plug runs.
        # Do that by calling the plug directly, not via the router.
        base_conn =
          conn(:get, "/api/status")
          |> put_req_header("authorization", "Bearer #{token}")
          |> assign(:workspace_id, "default:marketing")

        plug_conn = OptimalEngine.API.AuthPlug.call(base_conn, [])
        assert plug_conn.status == 403
        assert {:ok, %{"error" => "workspace_scope_denied"}} = Jason.decode(plug_conn.resp_body)
      end)
    end

    test "allows when key has wildcard workspace_scope '*'" do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: @tenant_id,
            name: "wildcard key",
            workspace_scope: ["*"]
          })

        base_conn =
          conn(:get, "/api/status")
          |> put_req_header("authorization", "Bearer #{token}")
          |> assign(:workspace_id, "default:marketing")

        plug_conn = OptimalEngine.API.AuthPlug.call(base_conn, [])
        refute plug_conn.halted
      end)
    end

    test "allows when key workspace_scope includes the requested workspace" do
      with_auth_required(fn ->
        {:ok, %{key: token}} =
          ApiKey.mint(%{
            tenant_id: @tenant_id,
            name: "explicit ws key",
            workspace_scope: ["default:engineering", "default:marketing"]
          })

        base_conn =
          conn(:get, "/api/status")
          |> put_req_header("authorization", "Bearer #{token}")
          |> assign(:workspace_id, "default:marketing")

        plug_conn = OptimalEngine.API.AuthPlug.call(base_conn, [])
        refute plug_conn.halted
      end)
    end
  end

  describe "POST /api/auth/keys" do
    test "mints a key and returns 201 with id, key, prefix" do
      conn = call(:post, "/api/auth/keys", [], %{"name" => "my ci key"})
      assert conn.status == 201
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert is_binary(body["id"])
      assert String.starts_with?(body["key"], "oe_")
      assert is_binary(body["prefix"])
    end

    test "returns 400 when name is missing" do
      conn = call(:post, "/api/auth/keys", [], %{})
      assert conn.status == 400
    end
  end

  describe "GET /api/auth/keys" do
    test "returns 200 with keys list for the tenant" do
      conn = call(:get, "/api/auth/keys")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert is_list(body["keys"])
      assert body["tenant_id"] == "default"
    end
  end

  describe "POST /api/auth/keys/:id/revoke" do
    test "returns 204 on success" do
      {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "to revoke via API"})
      conn = call(:post, "/api/auth/keys/#{id}/revoke")
      assert conn.status == 204
    end
  end

  describe "DELETE /api/auth/keys/:id" do
    test "returns 204 on success" do
      {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "to delete via API"})
      conn = call(:delete, "/api/auth/keys/#{id}")
      assert conn.status == 204
    end
  end
end
