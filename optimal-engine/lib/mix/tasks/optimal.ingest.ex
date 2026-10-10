defmodule Mix.Tasks.Optimal.Ingest do
  @shortdoc "Classify, route, write signal files, and index content"
  @moduledoc """
  Runs raw text through the full intake pipeline:
  classify → route → write signal files to disk → index in SQLite.

  Usage:
      mix optimal.ingest "Customer called about Platform Launch pricing"
      mix optimal.ingest "$(cat path/to/file.md)"
      mix optimal.ingest --file path/to/file.md
      mix optimal.ingest --file docs/notes.md --genre transcript --title "Team Sync"
      mix optimal.ingest --file notes.md --node project-platform-launch
      mix optimal.ingest --file notes.md --workspace sample-q1

  Options:
    --file       Read input from file instead of argument
    --genre      Override auto-detected genre (transcript, brief, spec, plan, note,
                 decision-log, standup, review, report, pitch)
    --title      Explicit title
    --node       Override primary node routing (project-platform-launch, operator, operation-revenue, etc.)
    --type       Force context type: signal, resource, memory, skill (default: auto-detect)
    --workspace  Target workspace ID (default: "default")
    --extract-claims     Force pending-Claim extraction on (overrides config)
    --no-extract-claims  Disable pending-Claim extraction for this run

  By default intake extracts a pending Claim from each accepted Signal
  (config `memory.auto_extract_claims`, default ON), closing the
  Source -> Signal -> Claim break.
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        switches: [
          file: :string,
          genre: :string,
          title: :string,
          node: :string,
          type: :string,
          workspace: :string,
          extract_claims: :boolean
        ],
        aliases: [f: :file, g: :genre, t: :title, n: :node, w: :workspace]
      )

    text = resolve_text(opts, positional)
    intake_opts = build_intake_opts(opts)

    IO.puts("")
    IO.puts("[intake] Classifying...")

    case OptimalEngine.Pipeline.Intake.process(text, intake_opts) do
      {:ok, result} -> print_result(result)
      {:error, reason} -> IO.puts("[intake] Failed: #{inspect(reason)}")
    end
  end

  defp resolve_text(opts, positional) do
    cond do
      file = Keyword.get(opts, :file) -> read_file!(file)
      positional != [] -> Enum.join(positional, " ")
      true -> Mix.raise("Usage: mix optimal.ingest \"text\" OR mix optimal.ingest --file path")
    end
  end

  defp read_file!(file) do
    case File.read(file) do
      {:ok, content} -> content
      {:error, reason} -> Mix.raise("Cannot read file #{file}: #{inspect(reason)}")
    end
  end

  defp build_intake_opts(opts) do
    []
    |> maybe_put(:genre, Keyword.get(opts, :genre))
    |> maybe_put(:title, Keyword.get(opts, :title))
    |> maybe_put(:node, Keyword.get(opts, :node))
    |> maybe_put(:workspace_id, Keyword.get(opts, :workspace))
    |> maybe_put(:extract_claims, Keyword.get(opts, :extract_claims))
    |> maybe_put_type(Keyword.get(opts, :type))
  end

  defp maybe_put(acc, _key, nil), do: acc
  defp maybe_put(acc, key, val), do: Keyword.put(acc, key, val)

  defp maybe_put_type(acc, nil), do: acc

  defp maybe_put_type(acc, t) when t in ~w[signal resource memory skill],
    do: Keyword.put(acc, :type, String.to_atom(t))

  defp maybe_put_type(_, other),
    do: Mix.raise("Unknown type '#{other}'. Use: signal, resource, memory, skill")

  defp print_result(result) do
    sig = result.signal

    IO.puts("  Genre:     #{sig.genre} | Type: #{sig.type} | Mode: #{sig.mode}")
    IO.puts("  Entities:  #{format_list(sig.entities)}")
    IO.puts("  Route:     #{primary_node(result)} (primary)#{cross_ref_note(result)}")
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
    print_claim(Map.get(result, :pending_claim))
    IO.puts("[intake] Done. URI: #{result.uri}")
    IO.puts("")
  end

  defp print_claim(nil), do: :ok

  defp print_claim(claim) do
    IO.puts("[intake] Extracted pending claim: #{claim.id} (#{claim.claim_type})")
    IO.puts("")
  end

  defp primary_node(result), do: result.signal.node || "inbox"

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
