defmodule OptimalEngine.Memory.Surfacer do
  @moduledoc """
  Proactive memory surfacing — Phase 15.

  When something interesting happens in a workspace (a wiki page is
  curated, a new signal lands, a cluster shifts), the Surfacer matches
  the event against active subscriptions and pushes envelopes to any
  connected stream listeners (SSE clients in the desktop UI).

  Architecture:

  - Holds a process map of `subscription_id => MapSet<listener_pid>`.
  - Public API: `subscribe/2` (a stream connects), `unsubscribe/2`,
    `notify_wiki_updated/3`, `notify_chunk_indexed/3`.
  - On event, iterates active subscriptions, computes a relevance score,
    and (if above threshold + outside cooldown) pushes a structured
    surface message to listeners + records the event row.

  Listener message shape:

      {:surface, %{
         subscription_id: "sub:default:anon:workspace:*",
         trigger:         :wiki_updated | :chunk_indexed,
         envelope:        %{slug: …, kind: :wiki_page | :signal, audience: …},
         category:        :recent_actions | …,
         score:           0.85,
         pushed_at:       "2026-04-29T20:30:00Z"
       }}
  """

  use GenServer
  require Logger

  alias OptimalEngine.Memory.Subscription
  alias OptimalEngine.Memory.Webhook
  alias OptimalEngine.Store

  @threshold 0.5
  # Avoid pushing the same (subscription, slug) pair within this window.
  @cooldown_seconds 3600

  # ── Client API ──────────────────────────────────────────────────────────

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Register a listener PID for a subscription's push stream."
  @spec subscribe(String.t(), pid()) :: :ok
  def subscribe(subscription_id, pid) when is_binary(subscription_id) and is_pid(pid) do
    GenServer.cast(__MODULE__, {:subscribe, subscription_id, pid})
  end

  @spec unsubscribe(String.t(), pid()) :: :ok
  def unsubscribe(subscription_id, pid) when is_binary(subscription_id) and is_pid(pid) do
    GenServer.cast(__MODULE__, {:unsubscribe, subscription_id, pid})
  end

  @doc """
  Triggered by the wiki curator (or by the API for testing) when a
  wiki page lands. Drives the surfacing fan-out.
  """
  @spec notify_wiki_updated(String.t(), String.t(), keyword()) :: :ok
  def notify_wiki_updated(workspace_id, slug, opts \\ []) do
    GenServer.cast(
      __MODULE__,
      {:event, :wiki_updated, workspace_id, slug,
       %{
         audience: Keyword.get(opts, :audience, "default"),
         body_preview: Keyword.get(opts, :body_preview, ""),
         entities: Keyword.get(opts, :entities, [])
       }}
    )
  end

  @doc """
  Records a contradiction-surfacing event and fans it out to active subscribers.

  `workspace_id` — the workspace where the contradiction was detected.
  `page_slug`    — the wiki page slug where the contradiction was found.
  `contradictions` — list of `%{type: :entity_attr_clash, entity:, attr:, claims: [...]}`
                     maps as returned by `Integrity.check_contradictions/2`.

  Records a `surfacing_events` row with `category = 'contradictions'` and
  pushes a `{:surface, ...}` message to every connected listener for active
  subscriptions in this workspace.
  """
  @spec notify_contradiction(String.t(), String.t(), list()) :: :ok
  def notify_contradiction(workspace_id, page_slug, contradictions)
      when is_binary(workspace_id) and is_binary(page_slug) and is_list(contradictions) do
    GenServer.cast(
      __MODULE__,
      {:event, :contradiction_detected, workspace_id, page_slug,
       %{
         audience: "default",
         body_preview: "contradict",
         entities: Enum.map(contradictions, & &1.entity),
         contradictions: contradictions
       }}
    )
  end

  @doc """
  Triggered when a new versioned memory is created via
  `OptimalEngine.Memory.Versioned.create/1`. Fans out to active
  subscriptions in `workspace_id`.

  `memory_id` becomes the envelope slug so subscribers can correlate the
  push back to the specific memory row.
  """
  @spec notify_memory_added(String.t(), String.t(), map()) :: :ok
  def notify_memory_added(workspace_id, memory_id, meta \\ %{})
      when is_binary(workspace_id) and is_binary(memory_id) do
    GenServer.cast(
      __MODULE__,
      {:event, :memory_added, workspace_id, memory_id,
       %{
         audience: Map.get(meta, :audience, "default"),
         body_preview: "",
         entities: []
       }}
    )
  end

  @doc "Trigger a synthetic surface event by subscription_id (test helper)."
  @spec test_push(String.t(), String.t()) :: :ok
  def test_push(subscription_id, slug) do
    GenServer.cast(__MODULE__, {:test_push, subscription_id, slug})
  end

  @doc "Returns current listener counts per subscription."
  @spec listener_stats() :: %{optional(String.t()) => non_neg_integer()}
  def listener_stats do
    GenServer.call(__MODULE__, :listener_stats)
  end

  # ── Server callbacks ────────────────────────────────────────────────────

  @impl true
  def init(_opts) do
    Logger.info("[Surfacer] online — proactive memory push")
    {:ok, %{listeners: %{}, last_push: %{}}}
  end

  @impl true
  def handle_cast({:subscribe, sub_id, pid}, state) do
    Process.monitor(pid)
    listeners = Map.update(state.listeners, sub_id, MapSet.new([pid]), &MapSet.put(&1, pid))
    {:noreply, %{state | listeners: listeners}}
  end

  @impl true
  def handle_cast({:unsubscribe, sub_id, pid}, state) do
    listeners =
      case Map.get(state.listeners, sub_id) do
        nil -> state.listeners
        set -> Map.put(state.listeners, sub_id, MapSet.delete(set, pid))
      end

    {:noreply, %{state | listeners: listeners}}
  end

  @impl true
  def handle_cast({:event, trigger, workspace_id, slug, meta}, state) do
    new_state =
      Subscription.list_all_active()
      |> Enum.filter(&(&1.workspace_id == workspace_id))
      |> Enum.reduce(state, fn sub, acc ->
        score = score(sub, slug, meta)

        cond do
          score < @threshold ->
            acc

          on_cooldown?(acc, sub.id, slug) ->
            acc

          true ->
            push(sub, trigger, slug, meta, score, acc.listeners)
            record_event(sub, trigger, slug, meta, score)
            %{acc | last_push: Map.put(acc.last_push, {sub.id, slug}, now())}
        end
      end)

    {:noreply, new_state}
  end

  @impl true
  def handle_cast({:test_push, sub_id, slug}, state) do
    case Subscription.list(status: :active) |> elem(1) |> Enum.find(&(&1.id == sub_id)) do
      nil ->
        {:noreply, state}

      sub ->
        push(
          sub,
          :test_push,
          slug,
          %{audience: "default", body_preview: "", entities: []},
          1.0,
          state.listeners
        )

        record_event(sub, :test_push, slug, %{}, 1.0)
        {:noreply, state}
    end
  end

  @impl true
  def handle_call(:listener_stats, _from, state) do
    stats = state.listeners |> Map.new(fn {k, v} -> {k, MapSet.size(v)} end)
    {:reply, stats, state}
  end

  @impl true
  def handle_info({:DOWN, _ref, :process, pid, _reason}, state) do
    listeners =
      Map.new(state.listeners, fn {sub_id, set} -> {sub_id, MapSet.delete(set, pid)} end)

    {:noreply, %{state | listeners: listeners}}
  end

  def handle_info(_msg, state), do: {:noreply, state}

  # ── Scoring ─────────────────────────────────────────────────────────────

  defp score(%Subscription{scope: :workspace}, _slug, _meta), do: 1.0

  defp score(%Subscription{scope: :node, scope_value: node_slug}, slug, meta) do
    cond do
      node_slug == nil -> 0.0
      String.contains?(slug, node_slug) -> 1.0
      Enum.any?(meta[:entities] || [], &String.contains?(slug <> &1, node_slug)) -> 0.7
      true -> 0.0
    end
  end

  defp score(%Subscription{scope: :topic, scope_value: topic}, slug, meta) do
    topic = topic || ""
    body = meta[:body_preview] || ""

    cond do
      topic == "" ->
        0.0

      String.contains?(slug, String.downcase(topic)) ->
        1.0

      String.contains?(String.downcase(body), String.downcase(topic)) ->
        0.7

      Enum.any?(
        meta[:entities] || [],
        &String.contains?(String.downcase(&1), String.downcase(topic))
      ) ->
        0.6

      true ->
        0.0
    end
  end

  defp score(%Subscription{scope: :audience, scope_value: aud}, _slug, meta) do
    if (meta[:audience] || "default") == aud, do: 1.0, else: 0.0
  end

  # ── Push + record ───────────────────────────────────────────────────────

  defp push(sub, trigger, slug, meta, score, listeners) do
    payload = %{
      subscription_id: sub.id,
      workspace_id: sub.workspace_id,
      trigger: trigger,
      envelope: %{
        slug: slug,
        kind: kind_of(trigger),
        audience: meta[:audience] || "default"
      },
      category: detect_category(slug, meta),
      score: score,
      pushed_at: now()
    }

    msg = {:surface, payload}

    # SSE push to connected listener PIDs (existing behaviour — unchanged)
    case Map.get(listeners, sub.id) do
      nil -> :ok
      set -> Enum.each(set, &send(&1, msg))
    end

    # Webhook delivery — fire-and-forget; must never block or crash the Surfacer
    with webhook_url when is_binary(webhook_url) and webhook_url != "" <-
           get_in(sub.metadata, ["webhook_url"]) do
      Task.start(fn -> Webhook.deliver(payload, sub.metadata) end)
    end

    :ok
  end

  defp record_event(sub, trigger, slug, meta, score) do
    Store.raw_query(
      """
      INSERT INTO surfacing_events
        (tenant_id, workspace_id, subscription_id, trigger, envelope_slug,
         envelope_kind, category, score, metadata)
      VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
      """,
      [
        sub.tenant_id,
        sub.workspace_id,
        sub.id,
        Atom.to_string(trigger),
        slug,
        Atom.to_string(kind_of(trigger)),
        Atom.to_string(detect_category(slug, meta)),
        score,
        Jason.encode!(meta)
      ]
    )
  end

  defp kind_of(:wiki_updated), do: :wiki_page
  defp kind_of(:chunk_indexed), do: :signal
  defp kind_of(:memory_added), do: :memory
  defp kind_of(:test_push), do: :wiki_page
  defp kind_of(:contradiction_detected), do: :wiki_page
  defp kind_of(_), do: :other

  # Tiny heuristic — categorize the surface based on slug + metadata.
  # Real classifier comes later; this maps the slug to one of the 14
  # enterprise-relevant categories from Engramme's taxonomy.
  defp detect_category(slug, meta) do
    s = String.downcase(slug || "")
    body = String.downcase(meta[:body_preview] || "")

    cond do
      # Contradiction events take priority — set explicitly by notify_contradiction/3.
      String.contains?(body, "contradict") or Map.has_key?(meta, :contradictions) ->
        :contradictions

      String.contains?(s, "decision") or String.contains?(body, "decision") ->
        :recent_actions

      String.contains?(s, "pricing") ->
        :recent_actions

      String.contains?(s, "schedule") or String.contains?(s, "deadline") ->
        :schedules

      String.contains?(s, "owner") or String.contains?(s, "responsibility") ->
        :ownership

      String.contains?(s, "todo") or String.contains?(s, "action-items") ->
        :open_tasks

      String.contains?(s, "spec") or String.contains?(s, "rfc") ->
        :professional_knowledge

      String.contains?(s, "playbook") or String.contains?(s, "how-to") ->
        :procedures

      String.contains?(s, "contact") or String.contains?(s, "people") ->
        :contacts

      true ->
        :unassigned
    end
  end

  defp on_cooldown?(state, sub_id, slug) do
    case Map.get(state.last_push, {sub_id, slug}) do
      nil ->
        false

      iso ->
        case DateTime.from_iso8601(iso) do
          {:ok, dt, _} -> DateTime.diff(DateTime.utc_now(), dt, :second) < @cooldown_seconds
          _ -> false
        end
    end
  end

  defp now, do: DateTime.utc_now() |> DateTime.to_iso8601()
end
