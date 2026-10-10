defmodule OptimalEngine.Connectors.PullScheduler do
  @moduledoc """
  Periodic connector pull loop — the engine's intake FEED.

  On an interval the scheduler runs each enabled connector once and drives
  the produced signals through `Pipeline.Intake.process/2`, so external
  evidence (local sources folder today; SaaS connectors as they come
  online) flows into the lifecycle: Source → Signal → Claim → Fact →
  Memory.

  The Runner already owns per-connector state (cursor advance on success,
  retry/backoff, disable-on-fatal). This scheduler is the *driver*: it
  decides WHICH connectors run and WHEN, and supplies the `signal_sink`
  that turns each emitted `%Signal{}` into an intake call.

  ## Config (`config :optimal_engine, :pull_scheduler`)

    * `:enabled`       — start the loop? (default `true`)
    * `:boot_delay_ms` — first run delay (default 2 min)
    * `:interval_ms`   — cadence (default 24 h — a daily pull)
    * `:tenant_id`     — tenant scope (default `"default"`)

  ## Config (`config :optimal_engine, :connectors`)

    * `:enabled` — list of connector specs to run each tick. Each spec is
      either a bare `kind` atom (a row is auto-provisioned with default
      config) or a map:

          %{kind: :sources_folder, workspace_id: "default", config: %{...}}

      Default: `[:sources_folder]`.

  Mirrors `OptimalEngine.MemoryCore.PromotionScheduler` in shape:
  `run_once/1` is the unit-testable entry point that needs no live process.
  """

  use GenServer
  require Logger

  alias OptimalEngine.Connectors
  alias OptimalEngine.Connectors.Registry
  alias OptimalEngine.Pipeline.Intake

  @default_boot_delay_ms 2 * 60 * 1_000
  @default_interval_ms 24 * 60 * 60 * 1_000
  @default_tenant_id "default"
  @default_connectors [:sources_folder]

  defstruct timer_ref: nil,
            last_run_at: nil,
            last_result: nil,
            enabled: true,
            boot_delay_ms: @default_boot_delay_ms,
            interval_ms: @default_interval_ms,
            tenant_id: @default_tenant_id

  @type connector_result :: %{
          connector_id: String.t(),
          kind: atom(),
          status: :ok | :error,
          signals: non_neg_integer(),
          reason: term() | nil
        }

  @type summary :: %{
          tenant_id: String.t(),
          connector_results: [connector_result()],
          ingested_count: non_neg_integer(),
          error_count: non_neg_integer()
        }

  # ── client API ─────────────────────────────────────────────────────────

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    name = Keyword.get(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  @doc "Current scheduler state summary."
  @spec status(GenServer.server()) :: map()
  def status(server \\ __MODULE__), do: GenServer.call(server, :status)

  @doc "Run a pull cycle immediately through the scheduler process."
  @spec force_run(keyword()) :: {:ok, summary()} | {:error, term()}
  def force_run(opts \\ []) do
    server = Keyword.get(opts, :server, __MODULE__)
    GenServer.call(server, {:force_run, opts}, 120_000)
  end

  @doc """
  Run one pull cycle without a live process — the unit-testable entry point.

  Options:
    * `:connectors`  — override the configured connector spec list
    * `:tenant_id`   — tenant scope (default "default")
    * `:adapter`     — adapter module override (test seam, passed to Runner)
    * `:signal_sink` — override the default intake sink (test seam)
    * `:max_retries` — passed through to the Runner
  """
  @spec run_once(keyword()) :: {:ok, summary()}
  def run_once(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, @default_tenant_id)
    specs = opts |> Keyword.get(:connectors, configured_connectors()) |> Enum.map(&normalize_spec/1)

    results =
      Enum.map(specs, fn spec -> run_connector(spec, tenant_id, opts) end)

    summary = %{
      tenant_id: tenant_id,
      connector_results: results,
      ingested_count: Enum.sum(Enum.map(results, & &1.signals)),
      error_count: Enum.count(results, &(&1.status == :error))
    }

    {:ok, summary}
  end

  # ── per-connector ────────────────────────────────────────────────────────

  defp run_connector(%{kind: kind} = spec, tenant_id, opts) do
    connector_id = spec[:id] || connector_id(kind, tenant_id)

    with {:ok, _kind} <- ensure_known(kind),
         {:ok, _id} <- ensure_row(spec, connector_id, tenant_id),
         {:ok, run_result} <- Connectors.run(connector_id, runner_opts(spec, opts)) do
      %{
        connector_id: connector_id,
        kind: kind,
        status: status_of(run_result),
        signals: Map.get(run_result, :signals, 0),
        reason: Map.get(run_result, :reason)
      }
    else
      {:error, reason} ->
        Logger.warning("[Connectors.PullScheduler] #{kind} failed: #{inspect(reason)}")

        %{
          connector_id: connector_id,
          kind: kind,
          status: :error,
          signals: 0,
          reason: reason
        }
    end
  end

  defp status_of(%{status: :success}), do: :ok
  defp status_of(%{status: :disabled}), do: :ok
  defp status_of(_), do: :error

  defp ensure_known(kind) do
    case Registry.fetch(kind) do
      {:ok, _mod} -> {:ok, kind}
      {:error, _} -> {:error, {:unknown_connector, kind}}
    end
  end

  # Idempotent provisioning: upsert keeps config/enabled fresh but never
  # resets the cursor (upsert_row only sets cursor on first insert), so the
  # dedupe ledger survives across ticks.
  defp ensure_row(spec, connector_id, tenant_id) do
    Connectors.register(%{
      id: connector_id,
      kind: spec.kind,
      tenant_id: tenant_id,
      workspace_id: spec[:workspace_id] || "default",
      config: spec[:config] || %{},
      enabled: true
    })
  end

  defp runner_opts(spec, opts) do
    sink = Keyword.get(opts, :signal_sink, intake_sink())

    [signal_sink: sink]
    |> maybe_put(:adapter, Keyword.get(opts, :adapter))
    |> maybe_put(:max_retries, Keyword.get(opts, :max_retries))
    |> maybe_put(:workspace_id, spec[:workspace_id])
  end

  # The sink the Runner invokes with the connector's emitted signals + a
  # scope map carrying the workspace. Each signal's body is re-driven
  # through the full intake pipeline, attributed to the connector workspace.
  defp intake_sink do
    fn signals, scope ->
      Enum.each(signals, fn signal ->
        ingest_signal(signal, scope)
      end)

      :ok
    end
  end

  defp ingest_signal(signal, scope) do
    content = signal.content || ""

    if String.trim(content) == "" do
      :ok
    else
      intake_opts =
        [workspace_id: Map.get(scope, :workspace_id, "default")]
        |> maybe_put(:genre, signal.genre)
        |> maybe_put(:title, signal.title)

      case Intake.process(content, intake_opts) do
        {:ok, _result} ->
          :ok

        {:error, reason} ->
          Logger.warning("[Connectors.PullScheduler] intake failed: #{inspect(reason)}")
      end
    end
  end

  # ── spec + config plumbing ───────────────────────────────────────────────

  defp configured_connectors do
    :optimal_engine
    |> Application.get_env(:connectors, [])
    |> Keyword.get(:enabled, @default_connectors)
  end

  defp normalize_spec(kind) when is_atom(kind), do: %{kind: kind}
  defp normalize_spec(%{kind: kind} = spec) when is_atom(kind), do: spec
  defp normalize_spec(%{"kind" => kind} = spec), do: Map.put(spec, :kind, String.to_atom(kind))

  defp connector_id(kind, tenant_id), do: "pull:#{tenant_id}:#{kind}"

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, value), do: Keyword.put(opts, key, value)

  # ── GenServer ────────────────────────────────────────────────────────────

  @impl true
  def init(opts) do
    state = opts |> scheduler_config() |> struct_state()

    state =
      if state.enabled do
        timer_ref = Process.send_after(self(), :tick, state.boot_delay_ms)

        Logger.info(
          "[Connectors.PullScheduler] started — first pull in #{state.boot_delay_ms}ms, interval #{state.interval_ms}ms"
        )

        %{state | timer_ref: timer_ref}
      else
        Logger.info("[Connectors.PullScheduler] disabled")
        state
      end

    {:ok, state}
  end

  @impl true
  def handle_call(:status, _from, state) do
    {:reply,
     %{
       enabled: state.enabled,
       tenant_id: state.tenant_id,
       interval_ms: state.interval_ms,
       boot_delay_ms: state.boot_delay_ms,
       last_run_at: state.last_run_at,
       last_result: state.last_result
     }, state}
  end

  @impl true
  def handle_call({:force_run, opts}, _from, state) do
    {reply, state} = run_and_store(opts, state)
    {:reply, reply, state}
  end

  @impl true
  def handle_info(:tick, state) do
    state = cancel_timer(state)
    {_reply, state} = run_and_store([tenant_id: state.tenant_id], state)
    timer_ref = Process.send_after(self(), :tick, state.interval_ms)
    {:noreply, %{state | timer_ref: timer_ref}}
  end

  defp run_and_store(opts, state) do
    {:ok, summary} = reply = run_once(Keyword.put_new(opts, :tenant_id, state.tenant_id))

    Logger.info(
      "[Connectors.PullScheduler] ingested=#{summary.ingested_count} errors=#{summary.error_count}"
    )

    {reply, %{state | last_run_at: DateTime.utc_now(), last_result: summary}}
  end

  defp cancel_timer(%{timer_ref: nil} = state), do: state

  defp cancel_timer(%{timer_ref: ref} = state) do
    Process.cancel_timer(ref)
    %{state | timer_ref: nil}
  end

  defp scheduler_config(opts) do
    config = Application.get_env(:optimal_engine, :pull_scheduler, [])

    %{
      enabled: get(opts, config, :enabled, true),
      boot_delay_ms: get(opts, config, :boot_delay_ms, @default_boot_delay_ms),
      interval_ms: get(opts, config, :interval_ms, @default_interval_ms),
      tenant_id: get(opts, config, :tenant_id, @default_tenant_id)
    }
  end

  defp get(opts, config, key, default) do
    Keyword.get(opts, key, Keyword.get(config, key, default))
  end

  defp struct_state(map), do: struct(__MODULE__, map)
end
