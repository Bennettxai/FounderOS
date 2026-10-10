defmodule OptimalEngine.Wiki.Scheduler do
  @moduledoc """
  Periodic wiki maintenance. Scans for stale pages and queues re-curation.

  A page is "stale" when:
  - `last_curated` is older than the workspace's configured `max_stale_days` (default 7)
  - AND new signals have been ingested into cited nodes since `last_curated`

  Both conditions must hold. A page that is old but has no new supporting
  signals is not worth re-curating — the LLM would have nothing new to
  integrate.

  Runs every `interval_minutes` (default 60, configurable per workspace via
  the `:curation` config section). Each tick processes at most
  `max_pages_per_cycle` pages per workspace to avoid overwhelming Ollama.

  ## Scheduling

  The first tick fires after a 5-minute boot delay so the rest of the
  supervision tree (Ollama, SQLite) can come up before any work begins.
  Subsequent ticks are governed by the shortest `interval_minutes` across
  all active workspaces, re-evaluated on each tick.

  ## Manual control

      OptimalEngine.Wiki.Scheduler.force_run("default")
  """

  use GenServer
  require Logger

  alias OptimalEngine.Store, as: RawStore
  alias OptimalEngine.Wiki.{Curator, Page}
  alias OptimalEngine.Wiki.Store, as: WikiStore
  alias OptimalEngine.Workspace
  alias OptimalEngine.Workspace.Config

  @boot_delay_ms 5 * 60 * 1_000
  @default_interval_minutes 60
  @default_max_stale_days 7
  @default_max_pages_per_cycle 5

  defstruct timer_ref: nil,
            last_run: nil

  # ── Public API ───────────────────────────────────────────────────────────────

  @doc "Start the scheduler under the supervision tree."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Returns stale pages for a workspace — those older than `max_stale_days`
  that also have new signals in their cited nodes since `last_curated`.

  Options:
    * `:max_stale_days` — override days threshold (default from workspace config)
    * `:tenant_id`      — tenant to scope (default: `"default"`)
  """
  @spec stale_pages(String.t(), keyword()) :: [Page.t()]
  def stale_pages(workspace_id, opts \\ []) do
    GenServer.call(__MODULE__, {:stale_pages, workspace_id, opts})
  end

  @doc "Trigger an immediate curation scan for a workspace without waiting for the tick."
  @spec force_run(String.t()) :: :ok
  def force_run(workspace_id) do
    GenServer.cast(__MODULE__, {:force_run, workspace_id})
  end

  # ── GenServer callbacks ──────────────────────────────────────────────────────

  @impl true
  def init(_opts) do
    timer_ref = Process.send_after(self(), :tick, @boot_delay_ms)

    Logger.info("[Wiki.Scheduler] started — first scan in #{div(@boot_delay_ms, 60_000)}min")

    {:ok, %__MODULE__{timer_ref: timer_ref}}
  end

  @impl true
  def handle_call({:stale_pages, workspace_id, opts}, _from, state) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    cur_cfg = curation_config(workspace_id)

    max_stale_days =
      Keyword.get(opts, :max_stale_days, Map.get(cur_cfg, :max_stale_days, @default_max_stale_days))

    pages = find_stale_pages(tenant_id, workspace_id, max_stale_days)
    {:reply, pages, state}
  end

  @impl true
  def handle_cast({:force_run, workspace_id}, state) do
    Logger.info("[Wiki.Scheduler] force_run requested for workspace=#{workspace_id}")
    run_workspace_scan(workspace_id)
    {:noreply, state}
  end

  @impl true
  def handle_info(:tick, state) do
    state = cancel_timer(state)

    Logger.debug("[Wiki.Scheduler] tick — scanning all active workspaces")
    run_all_workspaces()

    next_ms = next_interval_ms()
    timer_ref = Process.send_after(self(), :tick, next_ms)

    {:noreply, %{state | timer_ref: timer_ref, last_run: DateTime.utc_now()}}
  end

  # ── Core scanning logic ──────────────────────────────────────────────────────

  defp run_all_workspaces do
    case Workspace.list(status: :active) do
      {:ok, workspaces} ->
        Enum.each(workspaces, fn ws -> run_workspace_scan(ws.id) end)

      {:error, reason} ->
        Logger.warning("[Wiki.Scheduler] failed to list workspaces: #{inspect(reason)}")
    end
  end

  defp run_workspace_scan(workspace_id) do
    cur_cfg = curation_config(workspace_id)

    unless Map.get(cur_cfg, :enabled, true) do
      Logger.debug("[Wiki.Scheduler] curation disabled for workspace=#{workspace_id}, skipping")
      return_ok()
    else
      max_stale_days = Map.get(cur_cfg, :max_stale_days, @default_max_stale_days)
      max_pages = Map.get(cur_cfg, :max_pages_per_cycle, @default_max_pages_per_cycle)

      # We use the workspace id as both workspace_id and tenant_id for now.
      # The default workspace has tenant "default"; multi-tenant support
      # can layer in a workspace→tenant lookup here later.
      tenant_id = "default"

      stale = find_stale_pages(tenant_id, workspace_id, max_stale_days)
      batch = Enum.take(stale, max_pages)

      if batch == [] do
        Logger.debug("[Wiki.Scheduler] no stale pages for workspace=#{workspace_id}")
      else
        Logger.info(
          "[Wiki.Scheduler] workspace=#{workspace_id} — #{length(batch)} stale pages to re-curate (#{length(stale)} total stale)"
        )

        Enum.each(batch, fn page -> queue_curation(page, workspace_id) end)
      end
    end
  end

  # Return value placeholder to satisfy cond/unless branches that don't carry state.
  defp return_ok, do: :ok

  # ── Stale page detection ─────────────────────────────────────────────────────

  # A page is stale when:
  #   1. last_curated < now - max_stale_days
  #   2. at least one context in a cited node has been ingested since last_curated
  #
  # Cited nodes are derived from the citations table:
  #   citation → chunk → signal (context) → node
  #
  # We do this in two queries to keep each query readable:
  #   Query 1 — pages older than max_stale_days
  #   Query 2 — for each, check if new signals exist in cited nodes
  defp find_stale_pages(tenant_id, workspace_id, max_stale_days) do
    age_sql = """
    SELECT w.tenant_id, w.workspace_id, w.slug, w.audience, w.version,
           w.frontmatter, w.body, w.last_curated, w.curated_by
    FROM wiki_pages w
    INNER JOIN (
      SELECT tenant_id, workspace_id, slug, audience, MAX(version) AS max_version
      FROM wiki_pages
      WHERE tenant_id = ?1 AND workspace_id = ?2
      GROUP BY tenant_id, workspace_id, slug, audience
    ) latest
      ON w.tenant_id    = latest.tenant_id
     AND w.workspace_id = latest.workspace_id
     AND w.slug         = latest.slug
     AND w.audience     = latest.audience
     AND w.version      = latest.max_version
    WHERE datetime(w.last_curated) < datetime('now', ?3)
    ORDER BY w.last_curated ASC
    """

    age_param = "-#{max_stale_days} days"

    case RawStore.raw_query(age_sql, [tenant_id, workspace_id, age_param]) do
      {:ok, rows} ->
        rows
        |> Enum.map(&row_to_page/1)
        |> Enum.filter(fn page -> has_new_signals?(page) end)

      {:error, reason} ->
        Logger.warning("[Wiki.Scheduler] stale query failed: #{inspect(reason)}")
        []
    end
  end

  # Check if any new context in the cited nodes has been ingested since last_curated.
  defp has_new_signals?(%Page{last_curated: nil}), do: true

  defp has_new_signals?(%Page{} = page) do
    sql = """
    SELECT COUNT(*)
    FROM contexts ctx
    WHERE ctx.node IN (
      SELECT DISTINCT ctx2.node
      FROM contexts ctx2
      JOIN chunks ch  ON ch.signal_id = ctx2.id
      JOIN citations  cit ON cit.chunk_id = ch.id
      WHERE cit.wiki_slug     = ?1
        AND cit.wiki_audience = ?2
        AND cit.tenant_id     = ?3
    )
    AND datetime(ctx.created_at) > datetime(?4)
    AND ctx.workspace_id = ?5
    LIMIT 1
    """

    params = [
      page.slug,
      page.audience,
      page.tenant_id,
      page.last_curated,
      page.workspace_id
    ]

    case RawStore.raw_query(sql, params) do
      {:ok, [[count]]} when count > 0 -> true
      _ -> false
    end
  end

  # ── Async curation dispatch ──────────────────────────────────────────────────

  defp queue_curation(%Page{} = page, workspace_id) do
    Logger.info(
      "[Wiki.Scheduler] curating page=#{page.slug} audience=#{page.audience} workspace=#{workspace_id} last_curated=#{page.last_curated}"
    )

    Task.start(fn ->
      citations = gather_new_citations(page)

      outcome = Curator.curate(page, citations)

      if outcome.ok? do
        case WikiStore.put(outcome.page) do
          :ok ->
            Logger.info(
              "[Wiki.Scheduler] re-curation committed: page=#{page.slug} v#{outcome.page.version}"
            )

          {:error, reason} ->
            Logger.warning(
              "[Wiki.Scheduler] store.put failed after curation of page=#{page.slug}: #{inspect(reason)}"
            )
        end
      else
        Logger.warning(
          "[Wiki.Scheduler] curation returned ok?=false for page=#{page.slug}: #{inspect(outcome.metadata)}"
        )
      end
    end)
  end

  # Gather chunks from cited nodes that are newer than last_curated.
  # Returns a list of citation maps compatible with Curator.curate/3.
  defp gather_new_citations(%Page{last_curated: nil}), do: []

  defp gather_new_citations(%Page{} = page) do
    sql = """
    SELECT DISTINCT ch.id, ch.text, ctx.node
    FROM contexts ctx
    JOIN chunks ch ON ch.signal_id = ctx.id
    WHERE ctx.node IN (
      SELECT DISTINCT ctx2.node
      FROM contexts ctx2
      JOIN chunks ch2  ON ch2.signal_id = ctx2.id
      JOIN citations   cit ON cit.chunk_id = ch2.id
      WHERE cit.wiki_slug     = ?1
        AND cit.wiki_audience = ?2
        AND cit.tenant_id     = ?3
    )
    AND datetime(ctx.created_at) > datetime(?4)
    AND ctx.workspace_id = ?5
    AND ch.scale = 'paragraph'
    ORDER BY ctx.created_at ASC
    LIMIT 50
    """

    params = [
      page.slug,
      page.audience,
      page.tenant_id,
      page.last_curated,
      page.workspace_id
    ]

    case RawStore.raw_query(sql, params) do
      {:ok, rows} ->
        Enum.map(rows, fn [chunk_id, text, node] ->
          %{
            chunk_id: chunk_id,
            text: text || "",
            uri: "optimal://#{node}##{chunk_id}"
          }
        end)

      _ ->
        []
    end
  end

  # ── Scheduling helpers ───────────────────────────────────────────────────────

  # Use the shortest configured interval across all active workspaces, so the
  # scheduler honours any workspace that wants more frequent scans.
  defp next_interval_ms do
    case Workspace.list(status: :active) do
      {:ok, workspaces} ->
        min_minutes =
          workspaces
          |> Enum.map(fn ws ->
            ws.id
            |> curation_config()
            |> Map.get(:interval_minutes, @default_interval_minutes)
          end)
          |> Enum.min(fn -> @default_interval_minutes end)

        min_minutes * 60 * 1_000

      _ ->
        @default_interval_minutes * 60 * 1_000
    end
  end

  defp curation_config(workspace_id) do
    Config.get_section(workspace_id, :curation, default_curation_config())
  end

  defp default_curation_config do
    %{
      enabled: true,
      interval_minutes: @default_interval_minutes,
      max_stale_days: @default_max_stale_days,
      max_pages_per_cycle: @default_max_pages_per_cycle
    }
  end

  defp cancel_timer(%__MODULE__{timer_ref: nil} = state), do: state

  defp cancel_timer(%__MODULE__{timer_ref: ref} = state) do
    Process.cancel_timer(ref)
    %{state | timer_ref: nil}
  end

  # ── Row conversion ───────────────────────────────────────────────────────────

  defp row_to_page([
         tenant_id,
         workspace_id,
         slug,
         audience,
         version,
         fm_json,
         body,
         last_curated,
         curated_by
       ]) do
    frontmatter =
      case Jason.decode(fm_json || "{}") do
        {:ok, m} -> m
        _ -> %{}
      end

    %Page{
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      slug: slug,
      audience: audience,
      version: version,
      frontmatter: frontmatter,
      body: body,
      last_curated: last_curated,
      curated_by: curated_by
    }
  end
end
