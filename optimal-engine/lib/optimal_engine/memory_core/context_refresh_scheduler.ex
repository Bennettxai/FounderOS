defmodule OptimalEngine.MemoryCore.ContextRefreshScheduler do
  @moduledoc """
  Periodically refreshes stale Memory Core Context Packages.

  The scheduler does not implement its own retrieval logic. It calls
  `OptimalEngine.MemoryCore.refresh_stale_context_packages/1`, the same governed
  batch refresh path used by the API and Mix task.
  """

  use GenServer
  require Logger

  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Workspace

  @default_boot_delay_ms 5 * 60 * 1_000
  @default_interval_ms 60 * 60 * 1_000
  @default_batch_limit 50
  @default_workspace_limit 100

  defstruct timer_ref: nil,
            last_run_at: nil,
            last_result: nil,
            enabled: true,
            boot_delay_ms: @default_boot_delay_ms,
            interval_ms: @default_interval_ms,
            tenant_id: "default",
            actor_id: "system:context-refresh-scheduler",
            batch_limit: @default_batch_limit,
            workspace_limit: @default_workspace_limit,
            workspace_ids: :active,
            continue_on_error: true

  @type summary :: %{
          tenant_id: String.t(),
          workspace_results: [map()],
          refreshed_count: non_neg_integer(),
          error_count: non_neg_integer(),
          stale_context_package_ids: [String.t()],
          refreshed_context_package_ids: [String.t()]
        }

  @doc "Starts the scheduler under the supervision tree."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    name = Keyword.get(opts, :name, __MODULE__)
    GenServer.start_link(__MODULE__, opts, name: name)
  end

  @doc "Returns the current scheduler state summary."
  @spec status(GenServer.server()) :: map()
  def status(server \\ __MODULE__) do
    GenServer.call(server, :status)
  end

  @doc "Runs a refresh cycle immediately through the scheduler process."
  @spec force_run(String.t() | nil, keyword()) :: {:ok, summary()} | {:error, term()}
  def force_run(workspace_id \\ nil, opts \\ []) do
    server = Keyword.get(opts, :server, __MODULE__)
    GenServer.call(server, {:force_run, workspace_id, opts}, 60_000)
  end

  @doc "Runs one refresh cycle without requiring a scheduler process."
  @spec run_once(keyword()) :: {:ok, summary()} | {:error, term()}
  def run_once(opts \\ []) do
    tenant_id = string_opt(opts, :tenant_id, "default")

    with {:ok, workspace_ids} <- workspace_ids(opts, tenant_id) do
      workspace_ids
      |> Enum.reduce_while(empty_summary(tenant_id), fn workspace_id, {:ok, acc} ->
        refresh_opts =
          opts
          |> Keyword.drop([:workspace_ids, :workspace_limit])
          |> Keyword.put(:tenant_id, tenant_id)
          |> Keyword.put(:workspace_id, workspace_id)
          |> Keyword.put_new(:actor_id, "system:context-refresh-scheduler")
          |> Keyword.put_new(:batch_limit, @default_batch_limit)
          |> Keyword.put_new(:continue_on_error, true)

        case MemoryCore.refresh_stale_context_packages(refresh_opts) do
          {:ok, result} ->
            workspace_result = summarize_workspace(workspace_id, result)
            {:cont, merge_workspace_result(acc, workspace_result)}

          {:error, reason} ->
            if Keyword.get(opts, :continue_on_error, true) do
              workspace_result = %{
                workspace_id: workspace_id,
                stale_context_package_ids: [],
                refreshed_context_package_ids: [],
                refreshed_count: 0,
                error_count: 1,
                errors: [%{workspace_id: workspace_id, reason: reason}]
              }

              {:cont, merge_workspace_result(acc, workspace_result)}
            else
              {:halt, {:error, {workspace_id, reason}}}
            end
        end
      end)
    end
  end

  @impl true
  def init(opts) do
    state =
      opts
      |> scheduler_config()
      |> struct_state()

    state =
      if state.enabled do
        timer_ref = Process.send_after(self(), :tick, state.boot_delay_ms)

        Logger.info(
          "[MemoryCore.ContextRefreshScheduler] started — first refresh in #{state.boot_delay_ms}ms"
        )

        %{state | timer_ref: timer_ref}
      else
        Logger.info("[MemoryCore.ContextRefreshScheduler] disabled")
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
       workspace_ids: state.workspace_ids,
       batch_limit: state.batch_limit,
       interval_ms: state.interval_ms,
       last_run_at: state.last_run_at,
       last_result: state.last_result
     }, state}
  end

  @impl true
  def handle_call({:force_run, workspace_id, opts}, _from, state) do
    opts =
      state
      |> run_opts()
      |> Keyword.merge(Keyword.drop(opts, [:server]))
      |> maybe_put_workspace(workspace_id)

    {reply, state} = run_and_store(opts, state)
    {:reply, reply, state}
  end

  @impl true
  def handle_info(:tick, state) do
    state = cancel_timer(state)

    {_reply, state} =
      state
      |> run_opts()
      |> run_and_store(state)

    timer_ref = Process.send_after(self(), :tick, state.interval_ms)
    {:noreply, %{state | timer_ref: timer_ref}}
  end

  defp run_and_store(opts, state) do
    case run_once(opts) do
      {:ok, summary} ->
        Logger.info(
          "[MemoryCore.ContextRefreshScheduler] refreshed=#{summary.refreshed_count} errors=#{summary.error_count}"
        )

        {{:ok, summary}, %{state | last_run_at: timestamp(), last_result: summary}}

      {:error, reason} = error ->
        Logger.warning("[MemoryCore.ContextRefreshScheduler] refresh failed: #{inspect(reason)}")

        {error, %{state | last_run_at: timestamp(), last_result: %{error: reason}}}
    end
  end

  defp scheduler_config(opts) do
    app_config = Application.get_env(:optimal_engine, :context_refresh_scheduler, [])

    Keyword.merge(app_config, opts)
  end

  defp struct_state(config) do
    %__MODULE__{
      enabled: Keyword.get(config, :enabled, true),
      boot_delay_ms: Keyword.get(config, :boot_delay_ms, @default_boot_delay_ms),
      interval_ms: Keyword.get(config, :interval_ms, @default_interval_ms),
      tenant_id: string_opt(config, :tenant_id, "default"),
      actor_id: string_opt(config, :actor_id, "system:context-refresh-scheduler"),
      batch_limit: Keyword.get(config, :batch_limit, @default_batch_limit),
      workspace_limit: Keyword.get(config, :workspace_limit, @default_workspace_limit),
      workspace_ids: Keyword.get(config, :workspace_ids, :active),
      continue_on_error: Keyword.get(config, :continue_on_error, true)
    }
  end

  defp run_opts(state) do
    [
      tenant_id: state.tenant_id,
      actor_id: state.actor_id,
      batch_limit: state.batch_limit,
      workspace_limit: state.workspace_limit,
      workspace_ids: state.workspace_ids,
      continue_on_error: state.continue_on_error
    ]
  end

  defp workspace_ids(opts, tenant_id) do
    cond do
      workspace_id = Keyword.get(opts, :workspace_id) ->
        {:ok, [workspace_id]}

      workspace_ids = Keyword.get(opts, :workspace_ids) ->
        normalize_workspace_ids(workspace_ids, opts, tenant_id)

      true ->
        normalize_workspace_ids(:active, opts, tenant_id)
    end
  end

  defp normalize_workspace_ids(:active, opts, tenant_id) do
    case Workspace.list(
           tenant_id: tenant_id,
           status: :active,
           limit: Keyword.get(opts, :workspace_limit, @default_workspace_limit)
         ) do
      {:ok, workspaces} -> {:ok, Enum.map(workspaces, & &1.id)}
      other -> other
    end
  end

  defp normalize_workspace_ids(ids, _opts, _tenant_id) when is_list(ids) do
    {:ok, Enum.map(ids, &to_string/1)}
  end

  defp normalize_workspace_ids(id, _opts, _tenant_id) when is_binary(id), do: {:ok, [id]}

  defp empty_summary(tenant_id) do
    {:ok,
     %{
       tenant_id: tenant_id,
       workspace_results: [],
       refreshed_count: 0,
       error_count: 0,
       stale_context_package_ids: [],
       refreshed_context_package_ids: []
     }}
  end

  defp summarize_workspace(workspace_id, result) do
    refreshed_context_package_ids =
      Enum.map(result.refreshed_context_packages, & &1.id)

    %{
      workspace_id: workspace_id,
      stale_context_package_ids: result.stale_context_package_ids,
      refreshed_context_package_ids: refreshed_context_package_ids,
      refreshed_count: length(refreshed_context_package_ids),
      error_count: length(result.errors),
      errors: result.errors
    }
  end

  defp merge_workspace_result(acc, workspace_result) do
    {:ok,
     %{
       acc
       | workspace_results: acc.workspace_results ++ [workspace_result],
         refreshed_count: acc.refreshed_count + workspace_result.refreshed_count,
         error_count: acc.error_count + workspace_result.error_count,
         stale_context_package_ids:
           acc.stale_context_package_ids ++ workspace_result.stale_context_package_ids,
         refreshed_context_package_ids:
           acc.refreshed_context_package_ids ++
             workspace_result.refreshed_context_package_ids
     }}
  end

  defp maybe_put_workspace(opts, nil), do: opts
  defp maybe_put_workspace(opts, workspace_id), do: Keyword.put(opts, :workspace_id, workspace_id)

  defp cancel_timer(%{timer_ref: nil} = state), do: state

  defp cancel_timer(%{timer_ref: timer_ref} = state) do
    Process.cancel_timer(timer_ref)
    %{state | timer_ref: nil}
  end

  defp string_opt(opts, key, default) do
    opts
    |> Keyword.get(key, default)
    |> to_string()
  end

  defp timestamp, do: DateTime.utc_now() |> DateTime.truncate(:second) |> DateTime.to_iso8601()
end
