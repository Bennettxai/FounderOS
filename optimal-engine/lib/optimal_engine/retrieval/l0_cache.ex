defmodule OptimalEngine.Retrieval.L0Cache do
  @moduledoc """
  Maintains the always-loaded L0 context — the structural inventory of the library.

  L0 answers: "What exists? What's available?" — NOT "What's happening now?"

  Following tiered context systems's tiered loading model:
  - **L0** = Directory/inventory — nodes, skills, resources, memory counts
  - **L1** = Per-file summaries/abstracts — one-liner each
  - **L2** = Full content — loaded on demand

  The L0 inventory includes:
  1. **Node Map** — all 12 org nodes with context counts
  2. **Available Skills** — Mix commands / engine capabilities
  3. **Resource Index** — what docs/specs/references exist
  4. **Memory Summary** — how many memories, by category
  5. **System State** — active ops, recent decisions (the "live" portion)

  The cache auto-refreshes every 30 minutes, or when explicitly invalidated.
  """

  use GenServer
  require Logger

  alias OptimalEngine.Store
  alias OptimalEngine.Bridge.Memory, as: BridgeMemory

  @refresh_interval_ms 30 * 60 * 1_000
  @max_l0_chars 12_000

  # --- Client API ---

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Returns the current L0 context string (structural inventory)."
  @spec get() :: String.t()
  def get do
    GenServer.call(__MODULE__, :get)
  end

  @doc "Forces a cache refresh."
  @spec refresh() :: :ok
  def refresh do
    GenServer.cast(__MODULE__, :refresh)
  end

  # --- GenServer Callbacks ---

  @impl true
  def init(_opts) do
    schedule_refresh()
    send(self(), :build)
    {:ok, %{content: "", built_at: nil}}
  end

  @impl true
  def handle_call(:get, _from, state) do
    {:reply, state.content, state}
  end

  @impl true
  def handle_cast(:refresh, state) do
    new_content = build_l0()
    {:noreply, %{state | content: new_content, built_at: DateTime.utc_now()}}
  end

  @impl true
  def handle_info(:build, state) do
    new_content = build_l0()
    schedule_refresh()
    {:noreply, %{state | content: new_content, built_at: DateTime.utc_now()}}
  end

  @impl true
  def handle_info(:refresh_timer, state) do
    new_content = build_l0()
    schedule_refresh()
    Logger.debug("[L0Cache] Refreshed at #{DateTime.utc_now()}")
    {:noreply, %{state | content: new_content, built_at: DateTime.utc_now()}}
  end

  # --- Private: L0 Builder ---

  defp build_l0 do
    sections = [
      build_header(),
      build_node_inventory(),
      build_available_skills(),
      build_resource_index(),
      build_memory_summary(),
      build_system_state()
    ]

    content =
      sections
      |> Enum.reject(&is_nil/1)
      |> Enum.join("\n\n")

    if String.length(content) > @max_l0_chars do
      String.slice(content, 0, @max_l0_chars) <> "\n\n[...L0 truncated at budget]"
    else
      content
    end
  rescue
    e ->
      Logger.warning("[L0Cache] Build failed: #{inspect(e)}")
      "# L0 — Library Inventory\n\n_Inventory unavailable — run `mix optimal.index` first._"
  end

  defp build_header do
    now = DateTime.utc_now()

    """
    # L0 — Library Inventory
    > Generated: #{Calendar.strftime(now, "%Y-%m-%d %H:%M")} UTC
    > This is your structural inventory. What exists in the library.
    """
    |> String.trim()
  end

  # Section 1: Node Map — what org folders exist and how much is in each
  defp build_node_inventory do
    sql = """
    SELECT node, type, COUNT(*) as cnt
    FROM contexts
    GROUP BY node, type
    ORDER BY node, type
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        "## Nodes\n\n_Empty library — run `mix optimal.index`_"

      {:ok, rows} ->
        # Group by node
        by_node =
          Enum.group_by(rows, fn [node, _type, _cnt] -> node end)

        lines =
          by_node
          |> Enum.sort_by(fn {node, _} -> node end)
          |> Enum.map(fn {node, type_rows} ->
            counts =
              Enum.map_join(type_rows, ", ", fn [_node, type, cnt] ->
                "#{cnt} #{type}s"
              end)

            total = Enum.reduce(type_rows, 0, fn [_, _, c], acc -> acc + c end)
            "- **#{node}** (#{total}) — #{counts}"
          end)

        "## Nodes\n\n" <> Enum.join(lines, "\n")

      _ ->
        nil
    end
  end

  # Section 2: Available skills/commands
  defp build_available_skills do
    skills = [
      "`mix optimal.search \"query\"` — Search the library (FTS5 + vector + temporal)",
      "`mix optimal.ingest \"text\"` — Classify + route + store a signal",
      "`mix optimal.intake \"text\"` — Full intake pipeline",
      "`mix optimal.read \"optimal://...\" --tier l0|l1|full` — Tiered read",
      "`mix optimal.assemble \"topic\"` — Build tiered context bundle",
      "`mix optimal.ls \"optimal://...\"` — List contexts under URI",
      "`mix optimal.l0` — Print this inventory",
      "`mix optimal.index` — Full reindex from filesystem",
      "`mix optimal.stats` — Store statistics",
      "`mix optimal.graph query|sync|materialize` — Knowledge graph ops",
      "`mix optimal.knowledge metrics` — SICA learning patterns"
    ]

    "## Available Skills\n\n" <> Enum.map_join(skills, "\n", &"- #{&1}")
  end

  # Section 3: Resource index — what docs/specs/references exist
  defp build_resource_index do
    sql = """
    SELECT title, node, path, l0_abstract
    FROM contexts
    WHERE type = 'resource'
    ORDER BY node, title
    LIMIT 30
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        nil

      {:ok, rows} ->
        by_node = Enum.group_by(rows, fn [_title, node, _path, _l0] -> node end)

        sections =
          by_node
          |> Enum.sort_by(fn {node, _} -> node end)
          |> Enum.map(fn {node, resources} ->
            items =
              Enum.map_join(resources, "\n", fn [title, _node, _path, _l0] ->
                "  - #{title}"
              end)

            "**#{node}:**\n#{items}"
          end)

        "## Resources\n\n" <> Enum.join(sections, "\n")

      _ ->
        nil
    end
  end

  # Section 4: Memory summary — how many memories by category
  defp build_memory_summary do
    sql = """
    SELECT genre, COUNT(*) as cnt
    FROM contexts
    WHERE type = 'memory'
    GROUP BY genre
    ORDER BY cnt DESC
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        "## Memories\n\n_No extracted memories yet._"

      {:ok, rows} ->
        total = Enum.reduce(rows, 0, fn [_, c], acc -> acc + c end)

        cats =
          Enum.map_join(rows, ", ", fn [genre, cnt] ->
            "#{genre || "uncategorized"}: #{cnt}"
          end)

        "## Memories (#{total} total)\n\n#{cats}"

      _ ->
        nil
    end
  end

  # Section 5: System state — the "live" portion (active ops, decisions, people)
  defp build_system_state do
    parts = [
      build_cortex_bulletin(),
      build_active_operations(),
      build_recent_decisions(),
      build_key_people()
    ]

    live =
      parts
      |> Enum.reject(&is_nil/1)
      |> Enum.join("\n\n")

    if live == "" do
      nil
    else
      "## System State (Live)\n\n" <> live
    end
  end

  defp build_cortex_bulletin do
    bulletin = BridgeMemory.bulletin()
    if bulletin != "", do: "**Cortex:** #{bulletin}", else: nil
  end

  defp build_active_operations do
    sql = """
    SELECT title, node, COALESCE(genre, 'note'), modified_at
    FROM contexts
    WHERE type = 'signal'
      AND (signal_type IN ('direct', 'decide') OR genre IN ('plan', 'standup'))
      AND modified_at >= datetime('now', '-7 days')
    ORDER BY modified_at DESC
    LIMIT 5
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        nil

      {:ok, rows} ->
        lines =
          Enum.map_join(rows, "\n", fn [title, node, genre, modified_at] ->
            date = format_date_str(modified_at)
            "- #{title} (#{genre} | #{node} | #{date})"
          end)

        "**Active Ops:**\n#{lines}"

      _ ->
        nil
    end
  end

  defp build_recent_decisions do
    sql = """
    SELECT title, node, modified_at
    FROM contexts
    WHERE type = 'signal'
      AND genre IN ('decision-log', 'adr', 'decide')
      AND modified_at >= datetime('now', '-30 days')
    ORDER BY modified_at DESC
    LIMIT 5
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        nil

      {:ok, rows} ->
        lines =
          Enum.map_join(rows, "\n", fn [title, node, modified_at] ->
            date = format_date_str(modified_at)
            "- #{title} (#{node} | #{date})"
          end)

        "**Recent Decisions:**\n#{lines}"

      _ ->
        nil
    end
  end

  defp build_key_people do
    sql = """
    SELECT e.name, COUNT(*) as mention_count
    FROM entities e
    JOIN contexts c ON c.id = e.context_id
    WHERE c.modified_at >= datetime('now', '-30 days')
    GROUP BY e.name
    ORDER BY mention_count DESC
    LIMIT 8
    """

    case Store.raw_query(sql) do
      {:ok, []} ->
        nil

      {:ok, rows} ->
        people = Enum.map_join(rows, ", ", fn [name, count] -> "#{name} (×#{count})" end)
        "**Key People:** #{people}"

      _ ->
        nil
    end
  end

  defp schedule_refresh do
    Process.send_after(self(), :refresh_timer, @refresh_interval_ms)
  end

  defp format_date_str(nil), do: "unknown"
  defp format_date_str(str) when is_binary(str), do: String.slice(str, 0, 10)
end
