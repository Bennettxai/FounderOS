defmodule Mix.Tasks.Optimal.Remember do
  @shortdoc "Store governed memory signals and mine friction patterns"
  @moduledoc """
  Governed memory capture for the knowledge base.

  The default explicit-memory path writes through `OptimalEngine.Memory.remember/2`.
  That preserves source evidence, creates pending Claims, and keeps durable truth behind review.

  The contextual, mine, list, and escalations modes remain available for the older friction-insight loop.

  Usage:
      mix optimal.remember "always check duplicates before inserting"
      mix optimal.remember "decision made" --workspace default:my-workspace --force
      mix optimal.remember --contextual
      mix optimal.remember --mine
      mix optimal.remember --list
      mix optimal.remember --list --category process
      mix optimal.remember --escalations
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        strict: [
          workspace: :string,
          tenant: :string,
          audience: :string,
          actor: :string,
          force: :boolean,
          gate_threshold: :float,
          salience_floor: :float,
          category: :string,
          contextual: :boolean,
          mine: :boolean,
          list: :boolean,
          escalations: :boolean,
          legacy: :boolean
        ],
        aliases: [w: :workspace, a: :audience]
      )

    cond do
      Keyword.get(opts, :contextual, false) ->
        run_contextual()

      Keyword.get(opts, :mine, false) ->
        run_mine()

      Keyword.get(opts, :list, false) ->
        run_list(opts)

      Keyword.get(opts, :escalations, false) ->
        run_escalations()

      positional != [] ->
        observation = Enum.join(positional, " ")

        if Keyword.get(opts, :legacy, false) do
          run_legacy_explicit(observation)
        else
          run_explicit(observation, opts)
        end

      true ->
        IO.puts("Usage: mix optimal.remember \"observation\"")

        IO.puts(
          "       mix optimal.remember \"observation\" --workspace default:my-workspace --force"
        )

        IO.puts("       mix optimal.remember --contextual")
        IO.puts("       mix optimal.remember --mine")
        IO.puts("       mix optimal.remember --list [--category CAT]")
        IO.puts("       mix optimal.remember --escalations")
    end
  end

  # ---------------------------------------------------------------------------
  # Mode runners
  # ---------------------------------------------------------------------------

  defp run_explicit(observation, opts) do
    IO.puts("\nOptimal Memory - Governed Intake\n")

    attrs =
      %{
        content: observation,
        workspace_id: Keyword.get(opts, :workspace, "default")
      }
      |> maybe_put(:tenant_id, Keyword.get(opts, :tenant))
      |> maybe_put(:audience, Keyword.get(opts, :audience))
      |> maybe_put(:actor_id, Keyword.get(opts, :actor))

    remember_opts =
      [
        force: Keyword.get(opts, :force, false),
        gate_threshold: Keyword.get(opts, :gate_threshold),
        salience_floor: Keyword.get(opts, :salience_floor),
        versioned_projection: true
      ]
      |> reject_nil_keyword()

    case OptimalEngine.Memory.remember(attrs, remember_opts) do
      {:ok, result} ->
        IO.puts("  Action:    #{result.action}")
        IO.puts("  Workspace: #{attrs.workspace_id}")
        IO.puts("  Source:    #{result.source_package.id}")
        IO.puts("  Gate:      #{format_gate(result.gate)}")

        if result.pending_claim do
          IO.puts("  Claim:     #{result.pending_claim.id}")
        end

        if result.memory do
          IO.puts("  Memory:    #{result.memory.id}")
        end

      {:error, reason} ->
        IO.puts("  Failed: #{inspect(reason)}")
    end
  end

  defp run_legacy_explicit(observation) do
    IO.puts("\nRememberLoop - Storing Observation\n")

    case OptimalEngine.Insight.Remember.remember(observation) do
      {:ok, result} ->
        IO.puts("  Stored: [#{result.category}] #{result.content}")
        IO.puts("  Confidence: #{result.confidence}")

        if result.escalation.escalated do
          IO.puts(
            "  Escalation: #{result.escalation.count} observations in this category (total confidence: #{result.escalation.total_confidence})"
          )

          if result.escalation.ready_for_rethink do
            IO.puts("  -> Ready for rethink! Run: mix optimal.rethink \"#{result.category}\"")
          end
        end

      {:error, reason} ->
        IO.puts("  Failed: #{inspect(reason)}")
    end
  end

  defp run_contextual do
    IO.puts("\nRememberLoop - Contextual Scan\n")

    case OptimalEngine.Insight.Remember.contextual_scan() do
      {:ok, []} ->
        IO.puts("  No friction signals found in recent contexts.")

      {:ok, observations} ->
        IO.puts("  Found #{length(observations)} friction signals:\n")

        Enum.each(observations, fn obs ->
          IO.puts("  [#{obs.category}] #{String.slice(obs.content, 0, 80)}")
        end)
    end
  end

  defp run_mine do
    IO.puts("\nRememberLoop - Session Mining\n")

    case OptimalEngine.Insight.Remember.mine_sessions() do
      {:ok, []} ->
        IO.puts("  No patterns extracted from sessions.")

      {:ok, observations} ->
        IO.puts("  Extracted #{length(observations)} patterns:\n")

        Enum.each(observations, fn obs ->
          IO.puts("  [#{obs.category}] #{String.slice(obs.content, 0, 80)}")
        end)
    end
  end

  defp run_list(opts) do
    IO.puts("\nRememberLoop - Observations\n")

    category = Keyword.get(opts, :category)

    opts = if category, do: [category: category], else: []

    case OptimalEngine.Insight.Remember.list(opts) do
      {:ok, []} ->
        IO.puts("  No observations stored yet.")

      {:ok, observations} ->
        IO.puts("  #{length(observations)} observations:\n")

        Enum.each(observations, fn obs ->
          IO.puts("  ##{obs.id} [#{obs.category}] #{String.slice(obs.content, 0, 60)}")
          IO.puts("    confidence: #{obs.confidence} | source: #{obs.source} | #{obs.created_at}")
        end)
    end
  end

  defp run_escalations do
    IO.puts("\nRememberLoop - Escalation Candidates\n")

    case OptimalEngine.Insight.Remember.escalation_candidates() do
      {:ok, []} ->
        IO.puts("  No categories have reached escalation threshold (3+ observations).")

      {:ok, candidates} ->
        Enum.each(candidates, fn c ->
          status = if c.ready_for_rethink, do: "READY FOR RETHINK", else: "accumulating"

          IO.puts(
            "  [#{c.category}] #{c.count} observations, confidence: #{Float.round(c.total_confidence * 1.0, 2)} - #{status}"
          )
        end)
    end
  end

  defp maybe_put(map, _key, nil), do: map
  defp maybe_put(map, key, value), do: Map.put(map, key, value)

  defp reject_nil_keyword(opts) do
    Enum.reject(opts, fn {_key, value} -> is_nil(value) end)
  end

  defp format_gate(gate) do
    "#{gate.reason} score=#{Float.round(gate.score, 3)} encode?=#{gate.should_encode}"
  end
end
