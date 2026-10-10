defmodule Mix.Tasks.Optimal.Embed do
  @shortdoc "Parse → decompose → embed a file; print per-chunk vector summary"

  @moduledoc """
  Runs Stages 2 + 3 + 5 on a file:

    1. `OptimalEngine.Pipeline.Parser`      — extract text + structure
    2. `OptimalEngine.Pipeline.Decomposer`  — hierarchical chunking
    3. `OptimalEngine.Pipeline.Embedder`    — per-chunk 768-dim vectors

  Does NOT persist — use `mix optimal.ingest` for the full pipeline. This
  task is for verifying dispatch + provider health.

  ## Usage

      mix optimal.embed path/to/file.md
      mix optimal.embed path/to/file.pdf --format json
  """

  use Mix.Task

  alias OptimalEngine.Pipeline.{Decomposer, Embedder, Parser}
  alias OptimalEngine.Pipeline.Enricher.VlmEnricher

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args, strict: [format: :string], aliases: [f: :format])

    path =
      case positional do
        [p | _] -> p
        [] -> Mix.raise("Usage: mix optimal.embed <path>")
      end

    format = Keyword.get(opts, :format, "summary")

    with {:ok, parsed} <- Parser.parse(path),
         {:ok, enriched} <- VlmEnricher.enrich(parsed),
         {:ok, tree} <- Decomposer.decompose(enriched),
         {:ok, embeddings, %{errors: errors}} <- Embedder.embed_tree(tree) do
      render(tree, embeddings, errors, format)
    else
      {:error, reason} ->
        Mix.shell().error("Embed failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  defp render(_tree, embeddings, errors, "json") do
    payload = %{
      embeddings:
        Enum.map(embeddings, fn e ->
          %{
            chunk_id: e.chunk_id,
            model: e.model,
            modality: e.modality,
            dim: e.dim,
            vector_preview: Enum.take(e.vector, 5)
          }
        end),
      errors: errors
    }

    IO.puts(Jason.encode!(payload, pretty: true))
  end

  defp render(tree, embeddings, errors, _summary) do
    by_modality = Enum.group_by(embeddings, & &1.modality)

    IO.puts("")
    IO.puts("  Total chunks:        #{length(tree.chunks)}")
    IO.puts("  Embedded:            #{length(embeddings)}")
    IO.puts("  Skipped (errors):    #{length(errors)}")

    if embeddings != [] do
      dim = embeddings |> hd() |> Map.get(:dim)
      IO.puts("  Embedding dim:       #{dim} (nomic-aligned)")
    end

    IO.puts("")
    IO.puts("  Embeddings by modality:")

    Enum.each(by_modality, fn {modality, list} ->
      IO.puts("    #{String.pad_trailing(to_string(modality), 10)} #{length(list)}")
    end)

    if errors != [] do
      IO.puts("")
      IO.puts("  Errors (first 5):")

      errors
      |> Enum.take(5)
      |> Enum.each(fn {id, reason} ->
        IO.puts("    [#{id}] → #{inspect(reason)}")
      end)
    end

    IO.puts("")
  end
end
