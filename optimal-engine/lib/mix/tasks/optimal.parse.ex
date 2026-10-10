defmodule Mix.Tasks.Optimal.Parse do
  @shortdoc "Parse a file through the Stage 2 pipeline and print the result"

  @moduledoc """
  Parses a single file via `OptimalEngine.Pipeline.Parser` and prints the
  resulting `%ParsedDoc{}` for inspection.

  Useful for verifying format support and debugging the Parser dispatch.

  ## Usage

      mix optimal.parse path/to/file.md
      mix optimal.parse path/to/file.pdf --format raw
      mix optimal.parse path/to/file.csv --format summary

  ## Options

    --format  text | summary | raw (default: summary)
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args, strict: [format: :string], aliases: [f: :format])

    path =
      case positional do
        [p | _] -> p
        [] -> Mix.raise("Usage: mix optimal.parse <path>")
      end

    format = Keyword.get(opts, :format, "summary")

    case OptimalEngine.Pipeline.Parser.parse(path) do
      {:ok, doc} ->
        render(doc, format)

      {:error, reason} ->
        Mix.shell().error("Parse failed: #{inspect(reason)}")
        System.halt(1)
    end
  end

  defp render(doc, "raw"), do: IO.inspect(doc, limit: :infinity, pretty: true)

  defp render(doc, "text") do
    IO.puts(doc.text)
  end

  defp render(doc, _summary) do
    IO.puts("")
    IO.puts("  Path:       #{doc.path || "(inline)"}")
    IO.puts("  Signal id:  #{doc.signal_id}")
    IO.puts("  Modality:   #{doc.modality}")
    IO.puts("  Text bytes: #{byte_size(doc.text)}")
    IO.puts("  Structural: #{length(doc.structure)} elements")
    IO.puts("  Assets:     #{length(doc.assets)}")
    IO.puts("  Warnings:   #{length(doc.warnings)}")

    if doc.structure != [] do
      IO.puts("")
      IO.puts("  First 5 structural elements:")

      Enum.take(doc.structure, 5)
      |> Enum.each(fn e ->
        preview =
          e.text
          |> String.slice(0, 60)
          |> String.replace("\n", "\\n")

        IO.puts("    - #{e.kind}: #{preview}")
      end)
    end

    if doc.metadata != %{} do
      IO.puts("")
      IO.puts("  Metadata:")
      Enum.each(doc.metadata, fn {k, v} -> IO.puts("    #{k}: #{inspect(v)}") end)
    end

    if doc.warnings != [] do
      IO.puts("")
      IO.puts("  Warnings:")
      Enum.each(doc.warnings, fn w -> IO.puts("    ⚠ #{w}") end)
    end

    IO.puts("")
  end
end
