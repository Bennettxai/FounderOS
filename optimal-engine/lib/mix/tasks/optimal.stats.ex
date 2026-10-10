defmodule Mix.Tasks.Optimal.Stats do
  @shortdoc "Show Optimal Context Engine store statistics"
  @moduledoc """
  Displays statistics from the context store.

  Usage:
      mix optimal.stats
  """

  use Mix.Task

  @impl Mix.Task
  def run(_args) do
    Mix.Task.run("app.start")

    IO.puts("\n[optimal.stats] Store Statistics\n")

    case OptimalEngine.Store.stats() do
      {:ok, stats} ->
        IO.puts("  Total contexts: #{stats["total_contexts"]}")
        IO.puts("  ├ Signals:      #{stats["total_signals"]}")
        IO.puts("  ├ Resources:    #{stats["total_resources"]}")
        IO.puts("  ├ Memories:     #{stats["total_memories"]}")
        IO.puts("  └ Skills:       #{stats["total_skills"]}")
        IO.puts("  Entities:       #{stats["total_entities"]}")
        IO.puts("  Edges:          #{stats["total_edges"]}")
        IO.puts("  Decisions:      #{stats["total_decisions"]}")
        IO.puts("  ETS cache:      #{stats["cache_size"]} entries")

        IO.puts("\n[optimal.stats] Node Breakdown\n")
        print_node_breakdown()

        IO.puts("\n[optimal.stats] Type Breakdown\n")
        print_type_breakdown()

        IO.puts("\n[optimal.stats] Genre Breakdown (signals)\n")
        print_genre_breakdown()

        IO.puts("\n[optimal.stats] Phase 1 tables\n")
        print_phase1_tables()

      {:error, reason} ->
        IO.puts("Stats failed: #{inspect(reason)}")
    end
  end

  defp print_node_breakdown do
    sql = """
    SELECT node, COUNT(*) as count
    FROM contexts
    GROUP BY node
    ORDER BY count DESC
    LIMIT 20
    """

    case OptimalEngine.Store.raw_query(sql) do
      {:ok, rows} ->
        Enum.each(rows, fn [node, count] ->
          IO.puts("  #{String.pad_trailing(node, 30)} #{count}")
        end)

      _ ->
        IO.puts("  (unavailable)")
    end
  end

  defp print_type_breakdown do
    sql = """
    SELECT type, COUNT(*) as count
    FROM contexts
    GROUP BY type
    ORDER BY count DESC
    """

    case OptimalEngine.Store.raw_query(sql) do
      {:ok, rows} ->
        Enum.each(rows, fn [type, count] ->
          IO.puts("  #{String.pad_trailing(type, 30)} #{count}")
        end)

      _ ->
        IO.puts("  (unavailable)")
    end
  end

  defp print_genre_breakdown do
    sql = """
    SELECT genre, COUNT(*) as count
    FROM contexts
    WHERE type = 'signal' AND genre IS NOT NULL
    GROUP BY genre
    ORDER BY count DESC
    LIMIT 15
    """

    case OptimalEngine.Store.raw_query(sql) do
      {:ok, rows} ->
        Enum.each(rows, fn [genre, count] ->
          IO.puts("  #{String.pad_trailing(genre, 30)} #{count}")
        end)

      _ ->
        IO.puts("  (unavailable)")
    end
  end

  @phase1_tables ~w(
    tenants principals groups principal_groups roles role_grants acls
    chunks classifications intents assets clusters cluster_members
    wiki_pages citations connectors connector_runs retention_policies
    legal_holds audiences events schema_migrations
  )

  defp print_phase1_tables do
    widest = Enum.max_by(@phase1_tables, &String.length/1) |> String.length()

    Enum.each(@phase1_tables, fn table ->
      count = row_count(table)
      IO.puts("  #{String.pad_trailing(table, widest + 2)} #{count}")
    end)
  end

  defp row_count(table) do
    case OptimalEngine.Store.raw_query("SELECT COUNT(*) FROM #{table}") do
      {:ok, [[n]]} -> n
      _ -> "(unavailable)"
    end
  end
end
