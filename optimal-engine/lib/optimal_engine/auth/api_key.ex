defmodule OptimalEngine.Auth.ApiKey do
  @moduledoc """
  API key lifecycle management for OptimalEngine.

  Token format: `oe_<id>_<secret>`
  - `id`     — key id, stored in plaintext, used for lookup
  - `secret` — 32-byte url-safe base64 random; bcrypt-hashed before storage
  - `prefix` — first 8 chars of the raw secret; stored for display (not auth)

  ## Security notes
  - Secrets are returned exactly once from `mint/1` and never stored in plaintext.
  - `verify/1` does a constant-time bcrypt check after id lookup.
  - Never log the full token or the secret portion.
  - `record_usage/1` is async (fire-and-forget Task) to avoid adding latency
    to the hot request path.
  """

  require Logger

  alias OptimalEngine.Store

  @token_prefix "oe_"
  @secret_bytes 32

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          principal_id: String.t() | nil,
          prefix: String.t(),
          name: String.t(),
          scopes: [String.t()],
          workspace_scope: [String.t()],
          expires_at: String.t() | nil,
          created_at: String.t(),
          last_used_at: String.t() | nil,
          revoked_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            tenant_id: nil,
            principal_id: nil,
            prefix: nil,
            name: nil,
            scopes: ["*"],
            workspace_scope: ["*"],
            expires_at: nil,
            created_at: nil,
            last_used_at: nil,
            revoked_at: nil,
            metadata: %{}

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Mint a new API key for a tenant.

  Accepted attrs (map with string or atom keys):
  - `tenant_id`       (required) — tenant this key belongs to
  - `name`            (required) — human label ("production server", "ci/cd")
  - `principal_id`    (optional) — link to a principal
  - `scopes`          (optional) — list of scope strings, default `["*"]`
  - `workspace_scope` (optional) — list of workspace ids, default `["*"]`
  - `expires_at`      (optional) — ISO-8601 string or nil
  - `metadata`        (optional) — arbitrary map

  Returns `{:ok, %{id: id, secret: secret, key: "oe_<id>_<secret>"}}` on success.
  The `secret` value is returned **only here** and must be shown to the user immediately.
  """
  @spec mint(map()) ::
          {:ok, %{id: String.t(), secret: String.t(), key: String.t()}} | {:error, term()}
  def mint(attrs) do
    tenant_id = fetch!(attrs, :tenant_id)
    name = fetch!(attrs, :name)
    id = generate_id()
    secret = generate_secret()
    prefix = String.slice(secret, 0, 8)
    hashed = Bcrypt.hash_pwd_salt(secret, log_rounds: bcrypt_cost())

    principal_id = get_attr(attrs, :principal_id)
    scopes = get_attr(attrs, :scopes, ["*"])
    workspace_scope = get_attr(attrs, :workspace_scope, ["*"])
    expires_at = get_attr(attrs, :expires_at)
    metadata = get_attr(attrs, :metadata, %{})

    sql = """
    INSERT INTO api_keys
      (id, tenant_id, principal_id, hashed_secret, prefix, name, scopes, workspace_scope, expires_at, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
    """

    case Store.raw_query(sql, [
           id,
           tenant_id,
           principal_id,
           hashed,
           prefix,
           name,
           Jason.encode!(scopes),
           Jason.encode!(workspace_scope),
           expires_at,
           Jason.encode!(metadata)
         ]) do
      {:ok, _} ->
        key = build_token(id, secret)
        {:ok, %{id: id, secret: secret, key: key}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc """
  Verify a token string of the form `oe_<id>_<secret>`.

  Returns:
  - `{:ok, %ApiKey{}}` on success
  - `{:error, :invalid}` when the token cannot be parsed or bcrypt check fails
  - `{:error, :revoked}` when the key has been soft-revoked
  - `{:error, :expired}` when `expires_at` is in the past
  """
  @spec verify(String.t()) :: {:ok, t()} | {:error, :invalid | :revoked | :expired}
  def verify(token) when is_binary(token) do
    with {:parse, {:ok, id, secret}} <- {:parse, parse_token(token)},
         {:lookup, {:ok, key}} <- {:lookup, get_by_id(id)},
         {:revoked, false} <- {:revoked, revoked?(key)},
         {:expired, false} <- {:expired, expired?(key)},
         {:bcrypt, true} <- {:bcrypt, Bcrypt.verify_pass(secret, key.hashed_secret)} do
      {:ok, key}
    else
      {:parse, _} -> {:error, :invalid}
      {:lookup, _} -> {:error, :invalid}
      {:revoked, _} -> {:error, :revoked}
      {:expired, _} -> {:error, :expired}
      {:bcrypt, _} -> {:error, :invalid}
    end
  end

  def verify(_), do: {:error, :invalid}

  @doc """
  List non-revoked API keys for a tenant. Secrets are never returned.

  Options:
    - `:limit`  — max rows (default 50)
    - `:offset` — row offset for pagination (default 0)
  """
  @spec list(String.t(), keyword()) :: {:ok, [t()]}
  def list(tenant_id, opts \\ []) when is_binary(tenant_id) do
    limit = Keyword.get(opts, :limit, 50)
    offset = Keyword.get(opts, :offset, 0)

    sql = """
    SELECT id, tenant_id, principal_id, prefix, name, scopes, workspace_scope,
           expires_at, created_at, last_used_at, revoked_at, metadata
    FROM api_keys
    WHERE tenant_id = ?1 AND revoked_at IS NULL
    ORDER BY created_at DESC
    LIMIT ?2 OFFSET ?3
    """

    case Store.raw_query(sql, [tenant_id, limit, offset]) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      {:error, reason} -> {:error, reason}
    end
  end

  @doc "Count non-revoked API keys for a tenant."
  @spec count(String.t()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(tenant_id) when is_binary(tenant_id) do
    case Store.raw_query(
           "SELECT COUNT(*) FROM api_keys WHERE tenant_id = ?1 AND revoked_at IS NULL",
           [tenant_id]
         ) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, []} -> {:ok, 0}
      {:error, reason} -> {:error, reason}
    end
  end

  @doc "Soft-revoke a key by id. Sets `revoked_at` to the current UTC timestamp."
  @spec revoke(String.t()) :: :ok | {:error, term()}
  def revoke(id) when is_binary(id) do
    case Store.raw_query(
           "UPDATE api_keys SET revoked_at = datetime('now') WHERE id = ?1 AND revoked_at IS NULL",
           [id]
         ) do
      {:ok, _} -> :ok
      {:error, reason} -> {:error, reason}
    end
  end

  @doc "Hard-delete a key by id. Permanent."
  @spec delete(String.t()) :: :ok | {:error, term()}
  def delete(id) when is_binary(id) do
    case Store.raw_query("DELETE FROM api_keys WHERE id = ?1", [id]) do
      {:ok, _} -> :ok
      {:error, reason} -> {:error, reason}
    end
  end

  @doc """
  Update `last_used_at` asynchronously. Fire-and-forget — never blocks the caller.
  Errors are logged at debug level and swallowed.
  """
  @spec record_usage(String.t()) :: :ok
  def record_usage(id) when is_binary(id) do
    Task.start(fn ->
      case Store.raw_query(
             "UPDATE api_keys SET last_used_at = datetime('now') WHERE id = ?1",
             [id]
           ) do
        {:ok, _} ->
          :ok

        {:error, reason} ->
          Logger.debug("[ApiKey] record_usage failed for #{id}: #{inspect(reason)}")
      end
    end)

    :ok
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp generate_id do
    :crypto.strong_rand_bytes(12) |> Base.encode16(case: :lower)
  end

  defp generate_secret do
    :crypto.strong_rand_bytes(@secret_bytes) |> Base.url_encode64(padding: false)
  end

  defp build_token(id, secret), do: "#{@token_prefix}#{id}_#{secret}"

  # Parses "oe_<id>_<secret>" — id has no underscores (hex), secret may have dashes/underscores.
  defp parse_token(@token_prefix <> rest) do
    # id is 24 hex chars (12 bytes * 2), separator is "_", rest is the secret
    case rest do
      <<id::binary-size(24), "_", secret::binary>> when byte_size(secret) > 0 ->
        {:ok, id, secret}

      _ ->
        :error
    end
  end

  defp parse_token(_), do: :error

  defp get_by_id(id) do
    sql = """
    SELECT id, tenant_id, principal_id, hashed_secret, prefix, name, scopes, workspace_scope,
           expires_at, created_at, last_used_at, revoked_at, metadata
    FROM api_keys
    WHERE id = ?1
    """

    case Store.raw_query(sql, [id]) do
      {:ok, [row]} -> {:ok, row_to_struct_with_hash(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  defp revoked?(%__MODULE__{revoked_at: nil}), do: false
  defp revoked?(%__MODULE__{}), do: true

  defp expired?(%__MODULE__{expires_at: nil}), do: false

  defp expired?(%__MODULE__{expires_at: ts}) do
    case DateTime.from_iso8601(ts) do
      {:ok, dt, _} -> DateTime.compare(dt, DateTime.utc_now()) == :lt
      _ -> false
    end
  end

  # Used for list/1 — no hashed_secret column
  defp row_to_struct([
         id,
         tenant_id,
         principal_id,
         prefix,
         name,
         scopes_json,
         ws_scope_json,
         expires_at,
         created_at,
         last_used_at,
         revoked_at,
         meta_json
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      principal_id: principal_id,
      prefix: prefix,
      name: name,
      scopes: decode_json_list(scopes_json, ["*"]),
      workspace_scope: decode_json_list(ws_scope_json, ["*"]),
      expires_at: expires_at,
      created_at: created_at,
      last_used_at: last_used_at,
      revoked_at: revoked_at,
      metadata: decode_json_map(meta_json)
    }
  end

  # Used for verify — includes hashed_secret column (not stored in struct)
  defp row_to_struct_with_hash([
         id,
         tenant_id,
         principal_id,
         hashed_secret,
         prefix,
         name,
         scopes_json,
         ws_scope_json,
         expires_at,
         created_at,
         last_used_at,
         revoked_at,
         meta_json
       ]) do
    struct = %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      principal_id: principal_id,
      prefix: prefix,
      name: name,
      scopes: decode_json_list(scopes_json, ["*"]),
      workspace_scope: decode_json_list(ws_scope_json, ["*"]),
      expires_at: expires_at,
      created_at: created_at,
      last_used_at: last_used_at,
      revoked_at: revoked_at,
      metadata: decode_json_map(meta_json)
    }

    # Attach hashed_secret outside the struct for verify — not a struct field
    Map.put(struct, :hashed_secret, hashed_secret)
  end

  defp decode_json_list(nil, default), do: default
  defp decode_json_list("", default), do: default

  defp decode_json_list(json, default) do
    case Jason.decode(json) do
      {:ok, list} when is_list(list) -> list
      _ -> default
    end
  end

  defp decode_json_map(nil), do: %{}
  defp decode_json_map(""), do: %{}

  defp decode_json_map(json) do
    case Jason.decode(json) do
      {:ok, map} when is_map(map) -> map
      _ -> %{}
    end
  end

  defp fetch!(attrs, key) when is_map(attrs) do
    Map.get(attrs, key) || Map.get(attrs, to_string(key)) ||
      raise ArgumentError, "ApiKey.mint/1 requires #{inspect(key)}"
  end

  defp get_attr(attrs, key, default \\ nil) do
    Map.get(attrs, key, Map.get(attrs, to_string(key), default))
  end

  defp bcrypt_cost do
    Application.get_env(:optimal_engine, :auth, [])
    |> Keyword.get(:bcrypt_cost, 12)
  end
end
