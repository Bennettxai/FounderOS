defmodule Mix.Tasks.Optimal.Intake do
  @shortdoc "Interactive intake — reads multi-line content from stdin"
  @moduledoc """
  Interactive intake pipeline. Reads content from stdin (Ctrl+D to finish),
  then runs the full intake pipeline: classify → route → write files → index.

  Usage:
      mix optimal.intake
      mix optimal.intake --genre transcript --title "Requirements Review Call"
      mix optimal.intake --node project-platform-launch --title "Ed Call"
      echo "Customer called about pricing" | mix optimal.intake --genre note
      mix optimal.intake --workspace sample-q1 --genre note

  Options:
    --genre      Override auto-detected genre (transcript, brief, spec, plan, note,
                 decision-log, standup, review, report, pitch)
    --title      Explicit title
    --node       Override primary node routing (project-platform-launch, operator, operation-revenue, etc.)
    --workspace  Target workspace ID (default: "default")

  Examples:
      $ mix optimal.intake --genre transcript --title "Requirements Review Call"
      Enter content (Ctrl+D to finish):
      > Customer called about Platform Launch pricing...
      > He wants $2K per seat...
      > [Ctrl+D]

      [intake] Classifying...
        Genre: transcript | Type: decide | Mode: linguistic
        ...
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, _positional, _} =
      OptionParser.parse(args,
        switches: [
          genre: :string,
          title: :string,
          node: :string,
          workspace: :string
        ],
        aliases: [g: :genre, t: :title, n: :node, w: :workspace]
      )

    text = read_stdin(opts)

    if String.trim(text) == "" do
      Mix.raise("No content provided. Pipe input or enter text followed by Ctrl+D.")
    end

    intake_opts = build_intake_opts(opts)

    IO.puts("")
    IO.puts("[intake] Classifying...")

    case OptimalEngine.Pipeline.Intake.process(text, intake_opts) do
      {:ok, result} -> print_result(result)
      {:error, reason} -> IO.puts("[intake] Failed: #{inspect(reason)}")
    end
  end

  defp read_stdin(opts) do
    # If stdin is a TTY (interactive), show a prompt
    if :io.columns() != :enotsup do
      genre_hint = if g = Keyword.get(opts, :genre), do: " [#{g}]", else: ""
      IO.puts("Enter content#{genre_hint} (Ctrl+D to finish):")
    end

    IO.read(:stdio, :eof)
    |> case do
      :eof -> ""
      {:error, _} -> ""
      data -> data
    end
  end

  defp build_intake_opts(opts) do
    []
    |> maybe_put(:genre, Keyword.get(opts, :genre))
    |> maybe_put(:title, Keyword.get(opts, :title))
    |> maybe_put(:node, Keyword.get(opts, :node))
    |> maybe_put(:workspace_id, Keyword.get(opts, :workspace))
  end

  defp maybe_put(acc, _key, nil), do: acc
  defp maybe_put(acc, key, val), do: Keyword.put(acc, key, val)

  defp print_result(result) do
    sig = result.signal

    IO.puts("  Genre:     #{sig.genre} | Type: #{sig.type} | Mode: #{sig.mode}")
    IO.puts("  Entities:  #{format_list(sig.entities)}")
    IO.puts("  Route:     #{sig.node || "inbox"} (primary)#{cross_ref_note(result)}")
    IO.puts("")
    IO.puts("[intake] Writing signals...")

    Enum.each(result.files_written, fn f ->
      IO.puts("  + #{f}")
    end)

    Enum.each(result.cross_references, fn f ->
      IO.puts("  + #{f} (cross-ref)")
    end)

    IO.puts("")
    IO.puts("[intake] Updating index...")
    IO.puts("  + #{1 + length(result.cross_references)} context(s) indexed")
    IO.puts("")
    IO.puts("[intake] Done. URI: #{result.uri}")
    IO.puts("")
  end

  defp cross_ref_note(%{cross_references: []}), do: ""

  defp cross_ref_note(%{cross_references: refs}) do
    nodes =
      refs
      |> Enum.map(fn path ->
        path |> String.split("/") |> List.first("")
      end)
      |> Enum.uniq()
      |> Enum.join(", ")

    ", cross-ref: #{nodes}"
  end

  defp format_list([]), do: "(none)"
  defp format_list(list), do: Enum.join(list, ", ")
end
