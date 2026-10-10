defmodule Mix.Tasks.Optimal.Decompose do
  @shortdoc "Parse + decompose a file and print the chunk tree"

  @moduledoc """
  Runs Stages 2 + 3 on a file and prints the resulting `%ChunkTree{}`.

  Useful for verifying the decomposition shape for any format the Parser
  supports. Does NOT write to the store — use `mix optimal.ingest` once
  end-to-end ingestion is wired up.

  ## Usage

      mix optimal.decompose path/to/file.md
      mix optimal.decompose path/to/file.pdf --window 4096 --overlap 512
      mix optimal.decompose path/to/file.docx --format raw

  ## Options

    --window   :chunk scale window size in bytes (default 2048, ≈ 512 tokens)
    --overlap  window overlap in bytes (default 256, ≈ 64 tokens)
    --format   `summary` | `raw` | `counts` (default `summary`)
  """

  use Mix.Task

  alias OptimalEngine.Pipeline.Decomposer
  alias OptimalEngine.Pipeline.Decomposer.ChunkTree
  alias OptimalEngine.Pipeline.Enricher.VlmEnricher
  alias OptimalEngine.Pipeline.Parser

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        strict: [window: :integer, overlap: :integer, format: :string],
        aliases: [w: :window, o: :overlap, f: :format]
      )

    path =
      case positional do
        [p | _] -> p
        [] -> Mix.raise("Usage: mix optimal.decompose <path>")
      end

    format = Keyword.get(opts, :format, "summary")

    decompose_opts =
      [
        window_bytes: Keyword.get(opts, :window, 2048),
        overlap_bytes: Keyword.get(opts, :overlap, 256)
      ]

    with {:ok, parsed} <- Parser.parse(path),
         {:ok, enriched} <- VlmEnricher.enrich(parsed),
         {:ok, tree} <- Decomposer.decompose(enriched, decompose_opts) do
      render(tree, format)
    else
      {:error, reason} ->
        Mix.shell().error("Decompose failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  defp render(tree, "raw"), do: IO.inspect(tree, limit: :infinity, pretty: true)

  defp render(tree, "counts") do
    tree
    |> ChunkTree.counts()
    |> Enum.each(fn {scale, n} -> IO.puts("  #{scale}: #{n}") end)
  end

  defp render(tree, _summary) do
    counts = ChunkTree.counts(tree)

    IO.puts("")
    IO.puts("  Root id:       #{tree.root_chunk_id}")
    IO.puts("  :document      #{counts.document}")
    IO.puts("  :section       #{counts.section}")
    IO.puts("  :paragraph     #{counts.paragraph}")
    IO.puts("  :chunk         #{counts.chunk}")
    IO.puts("  Total chunks:  #{length(tree.chunks)}")
    IO.puts("")

    IO.puts("  First 3 :chunk-scale windows:")

    tree
    |> ChunkTree.at_scale(:chunk)
    |> Enum.take(3)
    |> Enum.each(fn chunk ->
      preview =
        chunk.text
        |> String.slice(0, 80)
        |> String.replace("\n", "\\n")

      IO.puts(
        "    [#{chunk.id}] offset=#{chunk.offset_bytes} len=#{chunk.length_bytes}  #{preview}"
      )
    end)

    IO.puts("")
  end
end
