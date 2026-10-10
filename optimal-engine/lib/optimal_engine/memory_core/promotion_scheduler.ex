defmodule OptimalEngine.MemoryCore.PromotionScheduler do
  @moduledoc """
  Autonomous Claim → Fact → Memory Object promotion loop.

  This closes the PROMOTION break in the lifecycle
  (Source → Signal → Claim → Fact → Memory → Context Package → Action): pending
  Claims would otherwise sit unreviewed forever. On an interval the scheduler
  drains pending Claims per workspace and, in `:auto_threshold` mode, promotes
  each Claim whose `aggregate_confidence` meets the per-workspace
  `ScoringPolicy.auto_promote_threshold/1` through the governed
  `ClaimReview.promote/2` path (which itself emits a Fact and a Memory Object).

  The scheduler never manufactures truth. It only offers high-confidence Claims
  to the same governed review boundary the API and Mix tasks use, with
  `policy: :auto` and `actor_id: "system:promotion-scheduler"`.

  ## Modes (`config :optimal_engine, :memory, claim_promotion_mode:`)

    * `:auto_threshold` (default) — promote Claims at/above the per-workspace
      auto-promotion threshold.
    * `:manual` — never auto-promote; the loop is a no-op (humans drive
      `ClaimReview.promote/2`).
    * `:agent_review` — like `:manual` for now; reserved for an agent reviewer
      that approves Claims out of band. The scheduler does not auto-promote.

  Mirrors `OptimalEngine.MemoryCore.ContextRefreshScheduler` in shape.
  """

  use GenServer
  require Logger

  alias OptimalEngine.MemoryCore.{ClaimReview, ScoringPolicy}
  alias OptimalEngine.Workspace

  @default_boot_delay_ms 5 * 60 * 1_000
  @default_interval_ms 30 * 60 * 1_000
  @default_batch_limit 50
  @default_workspace_limit 100
  @default_mode :auto_threshold
  @actor_id "system:promotion-scheduler"

  defstruct timer_ref: nil,
            last_run_at: nil,
            last_result: nil,
            enabled: true,
            mode: @default_mode,
            boot_delay_ms: @default_boot_delay_ms,
            interval_ms: @default_interval_ms,
            tenant_id: "default",
            actor_id: @actor_id,
            batch_limit: @default_batch_limit,
            workspace_limit: @default_workspace_limit,
            workspace_ids: :active,
            continue_on_error: true

  @type summary :: %{
          tenant_id: String.t(),
          mode: atom(),
          workspace_results: [map()],
          promoted_count: non_neg_integer(),
          skipped_count: non_neg_integer(),
          error_count: non_neg_integer(),
          promoted_fact_ids: [String.t()],
          promoted_memory_object_ids: [String.t()]
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

  @doc "Runs a promotion cycle immediately through the scheduler process."
  @spec force_run(String.t() | nil, keyword()) :: {:ok, summary()} | {:error, term()}
  def force_run(workspace_id \\ nil, opts \\ []) do
    server = Keyword.get(opts, :server, __MODULE__)
    GenServer.call(server, {:force_run, workspace_id, opts}, 60_000)
  end

  @doc """
  Runs one promotion cycle without requiring a scheduler process.

  This is the unit-testable entry point: seed pending Claims, call `run_once/1`,
  and assert on the returned summary (and the resulting Facts/Memory Objects).
  """
  @spec run_once(keyword()) :: {:ok, summary()} | {:error, term()}
  def run_once(opts \\ []) do
    tenant_id = string_opt(opts, :tenant_id, "default")
    mode = resolve_mode(opts)

    with {:ok, workspace_ids} <- workspace_ids(opts, tenant_id) do
      workspace_ids
      |> Enum.reduce_while(empty_summary(tenant_id, mode), fn workspace_id, {:ok, acc} ->
        case promote_workspace(workspace_id, tenant_id, mode, opts) do
          {:ok, workspace_result} ->
            {:cont, merge_workspace_result(acc, workspace_result)}

          {:error, reason} ->
            if Keyword.get(opts, :continue_on_error, true) do
              {:cont, merge_workspace_result(acc, error_workspace_result(workspace_id, reason))}
            else
              {:halt, {:error, {workspace_id, reason}}}
            end
        end
      end)
    end
  end

  # ── promotion per workspace ────────────────────────────────────────────

  defp promote_workspace(_workspace_id, _tenant_id, mode, _opts)
       when mode in [:manual, :agent_review] do
    {:ok,
     %{
       workspace_id: nil,
       promoted_fact_ids: [],
       promoted_memory_object_ids: [],
       promoted_count: 0,
       skipped_count: 0,
       error_count: 0,
       errors: []
     }}
  end

  defp promote_workspace(workspace_id, tenant_id, :auto_threshold, opts) do
    threshold = ScoringPolicy.auto_promote_threshold(workspace_id)
    batch_limit = Keyword.get(opts, :batch_limit, @default_batch_limit)
    actor_id = string_opt(opts, :actor_id, @actor_id)

    with {:ok, claims} <-
           ClaimReview.pending(
             tenant_id: tenant_id,
             workspace_id: workspace_id,
             limit: batch_limit
           ) do
      result =
        Enum.reduce(claims, blank_workspace_result(workspace_id), fn claim, acc ->
          if (claim.aggregate_confidence || 0.0) >= threshold do
            promote_claim(claim, tenant_id, workspace_id, actor_id, acc)
          else
            %{acc | skipped_count: acc.skipped_count + 1}
          end
        end)

      {:ok, result}
    end
  end

  defp promote_claim(claim, tenant_id, workspace_id, actor_id, acc) do
    case ClaimReview.promote(claim.id,
           tenant_id: tenant_id,
           workspace_id: workspace_id,
           policy: :auto,
           actor_id: actor_id
         ) do
      {:ok, %{fact: fact, memory_object: memory}} ->
        %{
          acc
          | promoted_count: acc.promoted_count + 1,
            promoted_fact_ids: acc.promoted_fact_ids ++ [fact.id],
            promoted_memory_object_ids: acc.promoted_memory_object_ids ++ [memory.id]
        }

      {:error, reason} ->
        %{
          acc
          | error_count: acc.error_count + 1,
            errors: acc.errors ++ [%{claim_id: claim.id, reason: reason}]
        }
    end
  end

  # ── GenServer ──────────────────────────────────────────────────────────

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
          "[MemoryCore.PromotionScheduler] started — mode=#{state.mode}, first promotion in #{state.boot_delay_ms}ms"
        )

        %{state | timer_ref: timer_ref}
      else
        Logger.info("[MemoryCore.PromotionScheduler] disabled")
        state
      end

    {:ok, state}
  end

  @impl true
  def handle_call(:status, _from, state) do
    {:reply,
     %{
       enabled: state.enabled,
       mode: state.mode,
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
          "[MemoryCore.PromotionScheduler] promoted=#{summary.promoted_count} skipped=#{summary.skipped_count} errors=#{summary.error_count}"
        )

        {{:ok, summary}, %{state | last_run_at: timestamp(), last_result: summary}}

      {:error, reason} = error ->
        Logger.warning("[MemoryCore.PromotionScheduler] promotion failed: #{inspect(reason)}")

        {error, %{state | last_run_at: timestamp(), last_result: %{error: reason}}}
    end
  end

  # ── config / state ───────────────────────────────────────────────────────

  defp scheduler_config(opts) do
    app_config = Application.get_env(:optimal_engine, :promotion_scheduler, [])
    Keyword.merge(app_config, opts)
  end

  defp struct_state(config) do
    %__MODULE__{
      enabled: Keyword.get(config, :enabled, true),
      mode: resolve_mode(config),
      boot_delay_ms: Keyword.get(config, :boot_delay_ms, @default_boot_delay_ms),
      interval_ms: Keyword.get(config, :interval_ms, @default_interval_ms),
      tenant_id: string_opt(config, :tenant_id, "default"),
      actor_id: string_opt(config, :actor_id, @actor_id),
      batch_limit: Keyword.get(config, :batch_limit, @default_batch_limit),
      workspace_limit: Keyword.get(config, :workspace_limit, @default_workspace_limit),
      workspace_ids: Keyword.get(config, :workspace_ids, :active),
      continue_on_error: Keyword.get(config, :continue_on_error, true)
    }
  end

  defp run_opts(state) do
    [
      tenant_id: state.tenant_id,
      mode: state.mode,
      actor_id: state.actor_id,
      batch_limit: state.batch_limit,
      workspace_limit: state.workspace_limit,
      workspace_ids: state.workspace_ids,
      continue_on_error: state.continue_on_error
    ]
  end

  # Explicit opt wins; otherwise the configured memory mode; otherwise default.
  defp resolve_mode(opts) do
    case Keyword.get(opts, :mode) do
      mode when mode in [:auto_threshold, :manual, :agent_review] ->
        mode

      _ ->
        :optimal_engine
        |> Application.get_env(:memory, [])
        |> Keyword.get(:claim_promotion_mode, @default_mode)
        |> normalize_mode()
    end
  end

  defp normalize_mode(mode) when mode in [:auto_threshold, :manual, :agent_review], do: mode
  defp normalize_mode(_), do: @default_mode

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

  # ── summaries ────────────────────────────────────────────────────────────

  defp empty_summary(tenant_id, mode) do
    {:ok,
     %{
       tenant_id: tenant_id,
       mode: mode,
       workspace_results: [],
       promoted_count: 0,
       skipped_count: 0,
       error_count: 0,
       promoted_fact_ids: [],
       promoted_memory_object_ids: []
     }}
  end

  defp blank_workspace_result(workspace_id) do
    %{
      workspace_id: workspace_id,
      promoted_fact_ids: [],
      promoted_memory_object_ids: [],
      promoted_count: 0,
      skipped_count: 0,
      error_count: 0,
      errors: []
    }
  end

  defp error_workspace_result(workspace_id, reason) do
    %{
      blank_workspace_result(workspace_id)
      | error_count: 1,
        errors: [%{workspace_id: workspace_id, reason: reason}]
    }
  end

  defp merge_workspace_result(acc, workspace_result) do
    {:ok,
     %{
       acc
       | workspace_results: acc.workspace_results ++ [workspace_result],
         promoted_count: acc.promoted_count + workspace_result.promoted_count,
         skipped_count: acc.skipped_count + workspace_result.skipped_count,
         error_count: acc.error_count + workspace_result.error_count,
         promoted_fact_ids: acc.promoted_fact_ids ++ workspace_result.promoted_fact_ids,
         promoted_memory_object_ids:
           acc.promoted_memory_object_ids ++ workspace_result.promoted_memory_object_ids
     }}
  end

  # ── helpers ────────────────────────────────────────────────────────────

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
