defmodule Mix.Tasks.Optimal.Index do
  @shortdoc "Full reindex of Optimal Engine workspace projection files"
  @moduledoc """
  Crawls the configured Optimal Engine workspace projection directory,
  classifies supported file types, and persists searchable context to the
  configured store.

  Supported types:
  - Markdown (.md)         → :signal (org folders) or :resource (docs/)
  - Code files (.ex, .py, .js, .ts, .go, etc.) → :resource
  - Data files (.json, .yaml, .toml, .csv)      → :resource
  - Binary files (.pdf, .docx, etc.)            → :resource (metadata only)
  - _memories/ subtree                          → :memory
  - _skills/ subtree                            → :skill

  Usage:
      mix optimal.index

  The task starts the full application (Store, Indexer, etc.),
  triggers a full index, then waits for completion.
  """

  use Mix.Task
  require Logger

  @poll_interval_ms 500
  @timeout_ms 600_000

  @impl Mix.Task
  def run(_args) do
    Mix.Task.run("app.start")

    IO.puts("\n[optimal.index] Starting full reindex...")

    case OptimalEngine.Pipeline.Indexer.full_index() do
      {:ok, :started} ->
        wait_for_completion(@timeout_ms)

      {:error, :already_running} ->
        IO.puts("[optimal.index] Index already running. Waiting for completion...")
        wait_for_completion(@timeout_ms)
    end
  end

  defp wait_for_completion(timeout) do
    deadline = System.monotonic_time(:millisecond) + timeout
    poll(deadline)
  end

  defp poll(deadline) do
    now = System.monotonic_time(:millisecond)

    if now > deadline do
      IO.puts("\n[optimal.index] Timeout waiting for index completion.")
      Mix.raise("Index timed out after #{@timeout_ms}ms")
    end

    status = OptimalEngine.Pipeline.Indexer.status()

    case status.status do
      :idle when not is_nil(status.last_run) ->
        IO.puts("\n[optimal.index] Contexts indexed. Building knowledge graph...")
        {:ok, edge_count} = OptimalEngine.Graph.rebuild()
        {:ok, stats} = OptimalEngine.Store.stats()
        IO.puts("[optimal.index] Complete!")
        IO.puts("  Contexts indexed: #{status.indexed_count}")
        IO.puts("  Total contexts:   #{stats["total_contexts"]}")
        IO.puts("  ├ Signals:        #{stats["total_signals"]}")
        IO.puts("  ├ Resources:      #{stats["total_resources"]}")
        IO.puts("  ├ Memories:       #{stats["total_memories"]}")
        IO.puts("  └ Skills:         #{stats["total_skills"]}")
        IO.puts("  Entities:         #{stats["total_entities"]}")
        IO.puts("  Edges:            #{edge_count}")
        IO.puts("  Cache entries:    #{stats["cache_size"]}")

      :running ->
        IO.write(".")
        Process.sleep(@poll_interval_ms)
        poll(deadline)

      :error ->
        IO.puts("\n[optimal.index] Indexer encountered an error. Check logs.")
        Mix.raise("Index failed")

      _ ->
        Process.sleep(@poll_interval_ms)
        poll(deadline)
    end
  end
end
