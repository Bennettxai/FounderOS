defmodule Mix.Tasks.Optimal.Classify do
  @shortdoc "Parse + decompose + classify + intent-extract a file; print summary"

  @moduledoc """
  Runs Stages 2 + 3 + 4 on a file:

    1. `OptimalEngine.Pipeline.Parser`         (Stage 2)
    2. `OptimalEngine.Pipeline.Decomposer`     (Stage 3)
    3. `OptimalEngine.Pipeline.Classifier`     (Stage 4a — per chunk)
    4. `OptimalEngine.Pipeline.IntentExtractor` (Stage 4b — per chunk)

  Prints counts + the intent distribution across chunks. Does NOT write to
  the store — use `mix optimal.ingest` once end-to-end persistence is wired.

  ## Usage

      mix optimal.classify path/to/file.md
      mix optimal.classify path/to/file.md --format json
  """

  use Mix.Task

  alias OptimalEngine.Pipeline.{Classifier, Decomposer, IntentExtractor, Parser}
  alias OptimalEngine.Pipeline.Enricher.VlmEnricher

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args, strict: [format: :string], aliases: [f: :format])

    path =
      case positional do
        [p | _] -> p
        [] -> Mix.raise("Usage: mix optimal.classify <path>")
      end

    format = Keyword.get(opts, :format, "summary")

    with {:ok, parsed} <- Parser.parse(path),
         {:ok, enriched} <- VlmEnricher.enrich(parsed),
         {:ok, tree} <- Decomposer.decompose(enriched),
         {:ok, classifications} <- Classifier.classify_tree(tree),
         {:ok, intents} <- IntentExtractor.extract_tree(tree) do
      render(tree, classifications, intents, format)
    else
      {:error, reason} ->
        Mix.shell().error("Classify failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  defp render(_tree, classifications, intents, "json") do
    payload = %{
      classifications:
        Enum.map(classifications, fn c ->
          %{
            chunk_id: c.chunk_id,
            mode: c.mode,
            genre: c.genre,
            signal_type: c.signal_type,
            format: c.format,
            structure: c.structure,
            sn_ratio: c.sn_ratio,
            confidence: c.confidence
          }
        end),
      intents:
        Enum.map(intents, fn i ->
          %{
            chunk_id: i.chunk_id,
            intent: i.intent,
            confidence: i.confidence,
            evidence: i.evidence
          }
        end)
    }

    IO.puts(Jason.encode!(payload, pretty: true))
  end

  defp render(tree, classifications, intents, _summary) do
    intent_histogram =
      intents
      |> Enum.group_by(& &1.intent)
      |> Enum.map(fn {intent, rows} -> {intent, length(rows)} end)
      |> Enum.sort_by(fn {_, n} -> -n end)

    avg_conf_intent =
      if intents == [], do: 0.0, else: avg(Enum.map(intents, & &1.confidence))

    avg_conf_class =
      if classifications == [],
        do: 0.0,
        else: avg(Enum.map(classifications, &(&1.confidence || 0.0)))

    IO.puts("")
    IO.puts("  Root:                #{tree.root_chunk_id}")
    IO.puts("  Total chunks:        #{length(tree.chunks)}")
    IO.puts("  Classifications:     #{length(classifications)}")
    IO.puts("  Intents:             #{length(intents)}")
    IO.puts("  Avg class confidence: #{Float.round(avg_conf_class, 2)}")
    IO.puts("  Avg intent confidence: #{Float.round(avg_conf_intent, 2)}")
    IO.puts("")
    IO.puts("  Intent distribution:")

    Enum.each(intent_histogram, fn {intent, n} ->
      IO.puts("    #{String.pad_trailing(to_string(intent), 20)} #{n}")
    end)

    IO.puts("")
  end

  defp avg(list) do
    Enum.sum(list) / length(list)
  end
end
