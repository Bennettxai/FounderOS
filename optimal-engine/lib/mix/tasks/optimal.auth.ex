defmodule Mix.Tasks.Optimal.Auth do
  @shortdoc "Manage API keys for apps, agents, and remote CLI/API access"

  @moduledoc """
  Manage API keys for Optimal Engine.

  Local CLI commands that run on the same machine use the local store directly.
  They do not need an API key.

  API keys are for HTTP/API clients, MCP servers, remote agents, scripts, and
  external apps that connect to a running Optimal Engine service.

  ## Usage

      mix optimal.auth mint --name "Business OS" --tenant default --workspace default:my-workspace
      mix optimal.auth list --tenant default
      mix optimal.auth revoke <key-id>
      mix optimal.auth delete <key-id>
      mix optimal.auth env --name "Local Agent" --workspace "*"

  ## Options

    --tenant       <id>    Tenant/organization scope. Default: `default`.
    --name         <name>  Human label for a minted key. Required for `mint`.
    --principal    <id>    Optional principal/service-account id.
    --scope        <s>     API scope. Repeatable. Default: `*`.
    --workspace    <id>    Workspace scope. Repeatable. Default: `*`.
    --expires-at   <iso>   Optional ISO-8601 expiry timestamp.
    --json                Print machine-readable JSON.

  The raw token is printed only once when minted. Store it in a secret manager
  or environment variable; do not put it in markdown, Source Packages, package
  manifests, or Context Packages.
  """

  use Mix.Task

  alias OptimalEngine.Auth.ApiKey

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, rest, _} =
      OptionParser.parse(args,
        strict: [
          tenant: :string,
          name: :string,
          principal: :string,
          scope: :keep,
          workspace: :keep,
          expires_at: :string,
          json: :boolean
        ],
        aliases: [t: :tenant, n: :name, p: :principal, s: :scope, w: :workspace]
      )

    case rest do
      ["mint" | _] -> mint(opts)
      ["env" | _] -> mint_env(opts)
      ["list" | _] -> list(opts)
      ["revoke", id | _] -> revoke(id, opts)
      ["delete", id | _] -> delete(id, opts)
      _ -> Mix.raise("Usage: mix optimal.auth mint|env|list|revoke <id>|delete <id>")
    end
  end

  defp mint(opts) do
    attrs = mint_attrs(opts)

    case ApiKey.mint(attrs) do
      {:ok, %{id: id, key: key, secret: _secret}} ->
        if opts[:json] do
          print_json(%{id: id, key: key, tenant_id: attrs.tenant_id})
        else
          IO.puts("API key minted.")
          IO.puts("  id:       #{id}")
          IO.puts("  tenant:   #{attrs.tenant_id}")
          IO.puts("  scopes:   #{Enum.join(attrs.scopes, ",")}")
          IO.puts("  workspace_scope: #{Enum.join(attrs.workspace_scope, ",")}")
          IO.puts("")
          IO.puts("Store this token now; it will not be shown again:")
          IO.puts("")
          IO.puts(key)
        end

      {:error, reason} ->
        Mix.raise("optimal.auth mint failed: #{inspect(reason)}")
    end
  end

  defp mint_env(opts) do
    opts = Keyword.put_new(opts, :name, "local-agent")
    attrs = mint_attrs(opts)

    case ApiKey.mint(attrs) do
      {:ok, %{id: id, key: key, secret: _secret}} ->
        if opts[:json] do
          print_json(%{
            id: id,
            tenant_id: attrs.tenant_id,
            env: %{"OPTIMAL_ENGINE_API_KEY" => key}
          })
        else
          IO.puts("# Add this to the shell/profile/secret manager for the app or agent:")
          IO.puts("export OPTIMAL_ENGINE_API_KEY=#{key}")
          IO.puts("# key id: #{id}")
        end

      {:error, reason} ->
        Mix.raise("optimal.auth env failed: #{inspect(reason)}")
    end
  end

  defp list(opts) do
    tenant_id = Keyword.get(opts, :tenant, "default")

    case ApiKey.list(tenant_id) do
      {:ok, keys} ->
        if opts[:json] do
          print_json(%{tenant_id: tenant_id, keys: Enum.map(keys, &key_to_map/1)})
        else
          IO.puts("API keys for tenant #{tenant_id}:")
          IO.puts("")

          Enum.each(keys, fn key ->
            IO.puts("  #{key.id}  #{key.name}")
            IO.puts("    principal: #{key.principal_id || "-"}")
            IO.puts("    scopes:    #{Enum.join(key.scopes, ",")}")
            IO.puts("    workspaces: #{Enum.join(key.workspace_scope, ",")}")
            IO.puts("    prefix:    #{key.prefix}")
            IO.puts("    created:   #{key.created_at}")
            if key.expires_at, do: IO.puts("    expires:   #{key.expires_at}")
            if key.last_used_at, do: IO.puts("    last used: #{key.last_used_at}")
            IO.puts("")
          end)

          if keys == [], do: IO.puts("  none")
        end

      {:error, reason} ->
        Mix.raise("optimal.auth list failed: #{inspect(reason)}")
    end
  end

  defp revoke(id, opts) do
    case ApiKey.revoke(id) do
      :ok ->
        if opts[:json],
          do: print_json(%{id: id, status: "revoked"}),
          else: IO.puts("Revoked #{id}.")

      {:error, reason} ->
        Mix.raise("optimal.auth revoke failed: #{inspect(reason)}")
    end
  end

  defp delete(id, opts) do
    case ApiKey.delete(id) do
      :ok ->
        if opts[:json],
          do: print_json(%{id: id, status: "deleted"}),
          else: IO.puts("Deleted #{id}.")

      {:error, reason} ->
        Mix.raise("optimal.auth delete failed: #{inspect(reason)}")
    end
  end

  defp mint_attrs(opts) do
    name = Keyword.get(opts, :name) || Mix.raise("--name is required")
    tenant_id = Keyword.get(opts, :tenant, "default")

    %{
      tenant_id: tenant_id,
      name: name,
      principal_id: Keyword.get(opts, :principal),
      scopes: keep_values(opts, :scope, ["*"]),
      workspace_scope: keep_values(opts, :workspace, ["*"]),
      expires_at: Keyword.get(opts, :expires_at)
    }
  end

  defp keep_values(opts, key, default) do
    values =
      opts
      |> Keyword.get_values(key)
      |> List.flatten()
      |> Enum.flat_map(&String.split(&1, ",", trim: true))
      |> Enum.map(&String.trim/1)
      |> Enum.reject(&(&1 == ""))

    if values == [], do: default, else: values
  end

  defp key_to_map(key) do
    %{
      id: key.id,
      tenant_id: key.tenant_id,
      principal_id: key.principal_id,
      prefix: key.prefix,
      name: key.name,
      scopes: key.scopes,
      workspace_scope: key.workspace_scope,
      expires_at: key.expires_at,
      created_at: key.created_at,
      last_used_at: key.last_used_at
    }
  end

  defp print_json(payload) do
    payload
    |> Jason.encode!(pretty: true)
    |> IO.puts()
  end
end
