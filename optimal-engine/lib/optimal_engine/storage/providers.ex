defmodule OptimalEngine.Storage.Providers do
  @moduledoc """
  Registry and safe health inspection for physical storage providers.

  Optional providers are configured only through environment references. The
  returned inventory reports which variables are present but never returns
  their values.
  """

  @providers [
    %{
      id: "sqlite",
      name: "SQLite",
      kind: "embedded",
      role: "canonical",
      capabilities: ~w(relational temporal jobs metrics backups model_registry)a,
      activation: "default_local",
      probe: :store,
      required_env: [],
      optional_env: []
    },
    %{
      id: "fts5",
      name: "SQLite FTS5",
      kind: "embedded",
      role: "projection",
      capabilities: ~w(full_text)a,
      activation: "default_local",
      probe: :store,
      required_env: [],
      optional_env: []
    },
    %{
      id: "embedded_vectors",
      name: "Embedded vectors",
      kind: "embedded",
      role: "projection",
      capabilities: ~w(vector)a,
      activation: "default_local",
      probe: :store,
      required_env: [],
      optional_env: []
    },
    %{
      id: "rocksdb",
      name: "RocksDB",
      kind: "embedded",
      role: "projection",
      capabilities: ~w(graph)a,
      activation: "default_local",
      probe: {:module, :rocksdb},
      required_env: [],
      optional_env: []
    },
    %{
      id: "filesystem",
      name: "Local filesystem",
      kind: "embedded",
      role: "canonical_artifact",
      capabilities: ~w(objects assets)a,
      activation: "default_local",
      probe: :root_path,
      required_env: [],
      optional_env: []
    },
    %{
      id: "ets",
      name: "Erlang ETS",
      kind: "embedded",
      role: "ephemeral",
      capabilities: ~w(cache coordination)a,
      activation: "default_local",
      probe: :runtime,
      required_env: [],
      optional_env: []
    },
    %{
      id: "postgres",
      name: "PostgreSQL",
      kind: "service",
      role: "canonical",
      capabilities: ~w(relational temporal document geospatial time_series)a,
      activation: "cloud_team",
      probe: {:tcp_url, "OPTIMAL_POSTGRES_URL", 5432},
      required_env: ["OPTIMAL_POSTGRES_URL"],
      optional_env: []
    },
    %{
      id: "s3",
      name: "S3-compatible object storage",
      kind: "service",
      role: "canonical_artifact",
      capabilities: ~w(objects assets backups)a,
      activation: "cloud_team",
      probe: {:http, "OPTIMAL_S3_ENDPOINT", "/"},
      required_env:
        ~w(OPTIMAL_S3_ENDPOINT OPTIMAL_S3_BUCKET OPTIMAL_S3_ACCESS_KEY_ID OPTIMAL_S3_SECRET_ACCESS_KEY),
      optional_env: []
    },
    %{
      id: "nats_jetstream",
      name: "NATS JetStream",
      kind: "service",
      role: "transport",
      capabilities: ~w(event_stream jobs replication coordination)a,
      activation: "distributed_runtime",
      probe: {:tcp_url, "OPTIMAL_NATS_URL", 4222},
      required_env: ["OPTIMAL_NATS_URL"],
      optional_env: []
    },
    %{
      id: "valkey",
      name: "Valkey",
      kind: "service",
      role: "ephemeral",
      capabilities: ~w(cache coordination rate_limits)a,
      activation: "distributed_runtime",
      probe: {:tcp_url, "OPTIMAL_VALKEY_URL", 6379},
      required_env: ["OPTIMAL_VALKEY_URL"],
      optional_env: []
    },
    %{
      id: "qdrant",
      name: "Qdrant",
      kind: "service",
      role: "projection",
      capabilities: ~w(vector)a,
      activation: "high_scale_semantic",
      probe: {:http, "OPTIMAL_QDRANT_URL", "/healthz"},
      required_env: ["OPTIMAL_QDRANT_URL"],
      optional_env: ["OPTIMAL_QDRANT_API_KEY"]
    },
    %{
      id: "duckdb",
      name: "DuckDB and Parquet",
      kind: "embedded",
      role: "projection",
      capabilities: ~w(analytics columnar lakehouse)a,
      activation: "analytics",
      probe: {:executable, "duckdb"},
      required_env: ["OPTIMAL_ANALYTICS_PATH"],
      optional_env: []
    },
    %{
      id: "openbao",
      name: "OpenBao",
      kind: "service",
      role: "secrets",
      capabilities: ~w(secrets kms pki)a,
      activation: "cloud_team",
      probe: {:http, "OPTIMAL_OPENBAO_ADDR", "/v1/sys/health"},
      required_env: ["OPTIMAL_OPENBAO_ADDR", "OPTIMAL_OPENBAO_TOKEN"],
      optional_env: []
    },
    %{
      id: "fractal",
      name: "Fractal Computing",
      kind: "enterprise_partner",
      role: "enterprise_substrate",
      capabilities: ~w(digital_twin distributed_analytics ai_safe_structured_data)a,
      activation: "enterprise_ai",
      probe: {:http, "OPTIMAL_FRACTAL_ENDPOINT", "/"},
      required_env: ["OPTIMAL_FRACTAL_ENDPOINT", "OPTIMAL_FRACTAL_CREDENTIAL_REF"],
      optional_env: []
    }
  ]

  @spec list(keyword()) :: [map()]
  def list(opts \\ []) do
    Enum.map(@providers, &inspect_provider(&1, opts))
  end

  @spec get(String.t(), keyword()) :: {:ok, map()} | {:error, :not_found}
  def get(id, opts \\ []) do
    case Enum.find(@providers, &(&1.id == id)) do
      nil -> {:error, :not_found}
      provider -> {:ok, inspect_provider(provider, opts)}
    end
  end

  @spec provider_ids() :: [String.t()]
  def provider_ids, do: Enum.map(@providers, & &1.id)

  defp inspect_provider(provider, opts) do
    configured = Enum.all?(provider.required_env, &present_env?/1)
    missing = Enum.reject(provider.required_env, &present_env?/1)
    probe? = Keyword.get(opts, :probe, false)

    health =
      cond do
        provider.required_env != [] and not configured -> :not_configured
        probe? -> run_probe(provider.probe)
        provider.activation == "default_local" -> :healthy
        true -> :configured
      end

    lifecycle_state = lifecycle_state(provider, configured, health)

    provider
    |> Map.drop([:probe, :required_env, :optional_env])
    |> Map.put(:configured, configured)
    |> Map.put(:health, Atom.to_string(health))
    |> Map.put(:lifecycle_state, lifecycle_state)
    |> Map.put(:required_environment, provider.required_env)
    |> Map.put(:optional_environment, provider.optional_env)
    |> Map.put(:missing_environment, missing)
  end

  defp lifecycle_state(%{activation: "default_local"}, true, :healthy), do: "active"
  defp lifecycle_state(_provider, false, _health), do: "not_configured"
  defp lifecycle_state(_provider, true, :healthy), do: "available"
  defp lifecycle_state(_provider, true, :configured), do: "configured_unverified"
  defp lifecycle_state(_provider, true, _health), do: "unavailable"

  defp run_probe(:runtime), do: :healthy

  defp run_probe(:store) do
    case safe_call(fn -> OptimalEngine.Store.raw_query("SELECT 1", []) end) do
      {:ok, [[1]]} -> :healthy
      _ -> :unreachable
    end
  end

  defp run_probe(:root_path) do
    path = Application.get_env(:optimal_engine, :root_path, ".")
    if File.dir?(path), do: :healthy, else: :unreachable
  end

  defp run_probe({:module, module}) do
    if Code.ensure_loaded?(module), do: :healthy, else: :not_installed
  end

  defp run_probe({:executable, executable}) do
    if System.find_executable(executable), do: :healthy, else: :not_installed
  end

  defp run_probe({:tcp_url, env, default_port}) do
    with value when is_binary(value) <- System.get_env(env),
         {:ok, host, port} <- endpoint(value, default_port),
         {:ok, socket} <-
           :gen_tcp.connect(String.to_charlist(host), port, [:binary, active: false], 750) do
      :gen_tcp.close(socket)
      :healthy
    else
      _ -> :unreachable
    end
  end

  defp run_probe({:http, env, path}) do
    with base when is_binary(base) <- System.get_env(env),
         url <- String.trim_trailing(base, "/") <> path,
         {:ok, {{_, status, _}, _headers, _body}} when status in 200..499 <-
           safe_call(fn ->
             :httpc.request(:get, {String.to_charlist(url), []}, [timeout: 1_000], [])
           end) do
      :healthy
    else
      _ -> :unreachable
    end
  end

  defp endpoint(value, default_port) do
    uri = URI.parse(value)

    cond do
      is_binary(uri.host) -> {:ok, uri.host, uri.port || default_port}
      String.contains?(value, ":") -> parse_host_port(value, default_port)
      value != "" -> {:ok, value, default_port}
      true -> {:error, :invalid_endpoint}
    end
  end

  defp parse_host_port(value, default_port) do
    case String.split(value, ":", parts: 2) do
      [host, port] ->
        case Integer.parse(port) do
          {parsed, ""} -> {:ok, host, parsed}
          _ -> {:ok, host, default_port}
        end

      [host] ->
        {:ok, host, default_port}
    end
  end

  defp present_env?(name) do
    case System.get_env(name) do
      value when is_binary(value) -> String.trim(value) != ""
      _ -> false
    end
  end

  defp safe_call(fun) do
    fun.()
  rescue
    _ -> {:error, :probe_failed}
  catch
    :exit, _ -> {:error, :probe_failed}
  end
end
