defmodule Mix.Tasks.Optimal.Knowledge do
  @moduledoc """
  Knowledge graph operations via OptimalEngine.Knowledge SPARQL engine.

  ## Usage

      mix optimal.knowledge sync       # Sync SQLite edges into SPARQL store
      mix optimal.knowledge count      # Count triples
      mix optimal.knowledge materialize # Run OWL 2 RL reasoning
      mix optimal.knowledge query "Alice"  # Query triples for entity
      mix optimal.knowledge metrics    # Show SICA learning metrics
  """

  use Mix.Task
  require Logger

  @shortdoc "Knowledge graph + SICA learning operations"

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    case args do
      ["sync" | _] ->
        Mix.shell().info("Syncing SQLite edges → knowledge graph + OWL reasoning...")

        case OptimalEngine.Bridge.Knowledge.sync_and_materialize() do
          {:ok, count} -> Mix.shell().info("Synced and materialized — #{inspect(count)} result.")
          other -> Mix.shell().error("Failed: #{inspect(other)}")
        end

      ["count" | _] ->
        case OptimalEngine.Bridge.Knowledge.count() do
          {:ok, count} -> Mix.shell().info("Knowledge store: #{count} triples")
          err -> Mix.shell().error("Failed: #{inspect(err)}")
        end

      ["materialize" | _] ->
        Mix.shell().info("Running OWL 2 RL materialization...")

        case OptimalEngine.Bridge.Knowledge.materialize() do
          {:ok, count} -> Mix.shell().info("Materialized #{count} inferred triples.")
          err -> Mix.shell().error("Failed: #{inspect(err)}")
        end

      ["query" | rest] ->
        entity = Enum.join(rest, " ")

        case OptimalEngine.Bridge.Knowledge.context_for(entity) do
          {:ok, context} -> Mix.shell().info(context)
          {:error, reason} -> Mix.shell().error("Failed: #{inspect(reason)}")
        end

      ["metrics" | _] ->
        metrics = OptimalEngine.Bridge.Memory.learning_metrics()

        Mix.shell().info("""
        === SICA Learning Metrics ===
        Interactions: #{Map.get(metrics, :interactions, 0)}
        Patterns:     #{Map.get(metrics, :patterns, 0)}
        Skills:       #{Map.get(metrics, :skills, 0)}
        Errors:       #{Map.get(metrics, :errors, 0)}
        """)

        patterns = OptimalEngine.Bridge.Memory.patterns()

        if map_size(patterns) > 0 do
          Mix.shell().info("Detected patterns:")

          Enum.each(patterns, fn {name, data} ->
            Mix.shell().info("  #{name}: #{inspect(data)}")
          end)
        end

      _ ->
        Mix.shell().info("""
        Usage:
          mix optimal.knowledge sync        — Sync SQLite edges → SPARQL store
          mix optimal.knowledge count       — Count triples in knowledge store
          mix optimal.knowledge materialize — Run OWL 2 RL reasoning
          mix optimal.knowledge query "Ed"  — Query context for entity
          mix optimal.knowledge metrics     — Show SICA learning stats
        """)
    end
  end
end
