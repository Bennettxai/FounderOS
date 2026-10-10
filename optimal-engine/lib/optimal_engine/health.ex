defmodule OptimalEngine.Health do
  @moduledoc """
  Liveness + readiness checks.

  Liveness is a boolean: "is the supervision tree up?". Readiness is
  richer: "can the engine actually serve traffic right now?" — SQLite
  writable, embedder reachable, credential key present if connectors
  are configured, migration state current.

  ## Usage

      iex> OptimalEngine.Health.live?()
      true

      iex> OptimalEngine.Health.ready()
      %{
        ok?: true,
        checks: %{
          store: :ok,
          migrations: :ok,
          credential_key: :ok,
          embedder: :ok
        },
        degraded: []
      }

  Each check returns `:ok | {:warn, reason} | {:error, reason}`.
  `ok?` is `true` when no check returned `:error`. Warnings indicate
  degraded mode — the engine still serves, but operators should know.
  """

  alias OptimalEngine.Connectors.Credential
  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.Store

  @type check_result :: :ok | {:warn, term()} | {:error, term()}
  @type readiness :: %{
          ok?: boolean(),
          checks: %{atom() => check_result()},
          degraded: [atom()]
        }

  @doc "True when the top-level supervisor is up."
  @spec live?() :: boolean()
  def live? do
    case Process.whereis(OptimalEngine.Supervisor) do
      nil -> false
      _ -> true
    end
  end

  @doc """
  Run every readiness check and return a combined report.

  Callers that want to skip slow checks (e.g. Ollama) can pass
  `skip: [:embedder]`.
  """
  @spec ready(keyword()) :: readiness()
  def ready(opts \\ []) do
    skip = Keyword.get(opts, :skip, [])

    checks =
      [:store, :migrations, :credential_key, :embedder]
      |> Enum.reject(&(&1 in skip))
      |> Enum.map(fn name -> {name, run_check(name)} end)
      |> Enum.into(%{})

    errors = Enum.filter(checks, fn {_, v} -> match?({:error, _}, v) end)
    warns = Enum.filter(checks, fn {_, v} -> match?({:warn, _}, v) end)

    %{
      ok?: errors == [],
      checks: checks,
      degraded: Enum.map(warns, fn {name, _} -> name end)
    }
  end

  @doc "Terse summary for CLIs: `:up | :degraded | :down`."
  @spec status() :: :up | :degraded | :down
  def status do
    if live?() do
      r = ready()

      cond do
        r.ok? and r.degraded == [] -> :up
        r.ok? -> :degraded
        true -> :down
      end
    else
      :down
    end
  end

  # ─── checks ──────────────────────────────────────────────────────────────

  defp run_check(:store) do
    case Store.raw_query("SELECT 1", []) do
      {:ok, [[1]]} -> :ok
      other -> {:error, {:store_unreachable, other}}
    end
  rescue
    e -> {:error, {:store_raised, Exception.message(e)}}
  end

  defp run_check(:migrations) do
    case Store.raw_query(
           "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='wiki_pages'",
           []
         ) do
      {:ok, [[1]]} -> :ok
      {:ok, [[0]]} -> {:error, :migrations_pending}
      other -> {:error, {:migrations_unknown, other}}
    end
  rescue
    e -> {:error, {:migrations_raised, Exception.message(e)}}
  end

  defp run_check(:credential_key) do
    cond do
      Credential.ready?() ->
        :ok

      connectors_configured?() ->
        {:error, :connector_key_missing}

      true ->
        # No connectors registered yet + no key set — that's the default
        # dev state, not a degraded one. Credential encryption only
        # matters once a connector is wired up.
        :ok
    end
  end

  defp run_check(:embedder) do
    case safe_ollama_ping() do
      :ok -> :ok
      {:error, reason} -> {:warn, {:embedder_unreachable, reason}}
    end
  end

  defp safe_ollama_ping do
    try do
      case Ollama.embed("health", model: "nomic-embed-text") do
        {:ok, _} -> :ok
        {:error, reason} -> {:error, reason}
      end
    rescue
      e -> {:error, Exception.message(e)}
    catch
      :exit, reason -> {:error, reason}
    end
  end

  defp connectors_configured? do
    case Store.raw_query("SELECT COUNT(*) FROM connectors WHERE enabled = 1", []) do
      {:ok, [[n]]} when is_integer(n) -> n > 0
      _ -> false
    end
  rescue
    _ -> false
  end
end
