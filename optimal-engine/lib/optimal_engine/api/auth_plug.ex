defmodule OptimalEngine.API.AuthPlug do
  @moduledoc """
  Plug-based API key authentication for OptimalEngine.

  Pipeline (per request):
  1. Extract token from `Authorization: Bearer <token>` or `X-API-Key: <token>`.
  2. If no token present:
     - `auth_required: false` (default in dev) → assigns anonymous defaults, continues.
     - `auth_required: true` → halts with 401 `{"error":"missing_api_key"}`.
  3. If token present → `ApiKey.verify/1`:
     - Success → assigns `current_tenant`, `current_principal`, `current_api_key`.
       Fires async `ApiKey.record_usage/1`.
     - Failure → halts with 401 and the relevant error atom as a string.
  4. Workspace scope check (when `conn.assigns[:workspace_id]` is set):
     - Key's `workspace_scope` must include the workspace id or contain `"*"`.
     - Mismatch → halts with 403 `{"error":"workspace_scope_denied"}`.

  ## Assigns set on success

  - `:current_tenant`    — tenant_id string (e.g. `"default"`)
  - `:current_principal` — principal_id string or `:anonymous`
  - `:current_api_key`   — `%OptimalEngine.Auth.ApiKey{}` or `nil` (anonymous)

  ## Configuration

      config :optimal_engine, :auth,
        auth_required: false,   # true in production
        bcrypt_cost: 12         # lower in test (set to 4)

  Read `auth_required` via `Application.get_env(:optimal_engine, :auth, [])`.
  """

  @behaviour Plug

  import Plug.Conn

  alias OptimalEngine.Auth.ApiKey

  @impl true
  def init(opts), do: opts

  @impl true
  def call(conn, _opts) do
    auth_required = auth_required?()

    case extract_token(conn) do
      nil ->
        handle_missing(conn, auth_required)

      token ->
        handle_token(conn, token)
    end
  end

  # ---------------------------------------------------------------------------
  # Token extraction
  # ---------------------------------------------------------------------------

  defp extract_token(conn) do
    bearer_token(conn) || api_key_header(conn)
  end

  defp bearer_token(conn) do
    case get_req_header(conn, "authorization") do
      ["Bearer " <> token | _] -> token
      _ -> nil
    end
  end

  defp api_key_header(conn) do
    case get_req_header(conn, "x-api-key") do
      [token | _] when token != "" -> token
      _ -> nil
    end
  end

  # ---------------------------------------------------------------------------
  # Missing token
  # ---------------------------------------------------------------------------

  defp handle_missing(conn, false) do
    # Anonymous mode — dev default. Continue with default tenant.
    conn
    |> assign(:current_tenant, "default")
    |> assign(:current_principal, :anonymous)
    |> assign(:current_api_key, nil)
  end

  defp handle_missing(conn, true) do
    halt_401(conn, "missing_api_key")
  end

  # ---------------------------------------------------------------------------
  # Token verification
  # ---------------------------------------------------------------------------

  defp handle_token(conn, token) do
    case ApiKey.verify(token) do
      {:ok, key} ->
        ApiKey.record_usage(key.id)

        conn
        |> assign(:current_tenant, key.tenant_id)
        |> assign(:current_principal, key.principal_id || :anonymous)
        |> assign(:current_api_key, key)
        |> check_workspace_scope()

      {:error, :revoked} ->
        halt_401(conn, "api_key_revoked")

      {:error, :expired} ->
        halt_401(conn, "api_key_expired")

      {:error, _} ->
        halt_401(conn, "invalid_api_key")
    end
  end

  # ---------------------------------------------------------------------------
  # Workspace scope check
  # ---------------------------------------------------------------------------

  defp check_workspace_scope(%{halted: true} = conn), do: conn

  defp check_workspace_scope(conn) do
    workspace_id = conn.assigns[:workspace_id]
    key = conn.assigns[:current_api_key]

    cond do
      # Anonymous or no workspace constraint — skip
      is_nil(key) -> conn
      is_nil(workspace_id) -> conn
      # Key has wildcard scope — always allowed
      "*" in key.workspace_scope -> conn
      # Key explicitly includes this workspace
      workspace_id in key.workspace_scope -> conn
      # Denied
      true -> halt_403(conn, "workspace_scope_denied")
    end
  end

  # ---------------------------------------------------------------------------
  # Auth required config
  # ---------------------------------------------------------------------------

  defp auth_required? do
    :optimal_engine
    |> Application.get_env(:auth, [])
    |> Keyword.get(:auth_required, false)
  end

  # ---------------------------------------------------------------------------
  # Error responses
  # ---------------------------------------------------------------------------

  defp halt_401(conn, error) do
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(401, Jason.encode!(%{error: error}))
    |> halt()
  end

  defp halt_403(conn, error) do
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(403, Jason.encode!(%{error: error}))
    |> halt()
  end
end
