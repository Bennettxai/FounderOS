defmodule Mix.Tasks.Optimal.Read do
  @shortdoc "Read a context by optimal:// URI"
  @moduledoc """
  Reads a context by its `optimal://` URI and prints its metadata + content.

  Usage:
      mix optimal.read "optimal://nodes/project-platform-launch/context.md"
      mix optimal.read "optimal://nodes/operator/signal.md" --tier l0
      mix optimal.read "optimal://nodes/operator/signal.md" --tier l1
      mix optimal.read "optimal://nodes/operator/signal.md" --tier full

  Options:
    --tier   Content tier to display: l0, l1, full (default: l1)
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        switches: [tier: :string],
        aliases: [t: :tier]
      )

    uri =
      case positional do
        [u | _] -> u
        [] -> Mix.raise("Usage: mix optimal.read \"optimal://...\"")
      end

    tier = Keyword.get(opts, :tier, "l1")

    IO.puts("\n[optimal.read] #{uri}\n")

    case OptimalEngine.read(uri) do
      {:ok, ctx} ->
        print_context(ctx, tier)

      {:error, :not_found} ->
        IO.puts("Not found. Run `mix optimal.index` first, or check the URI.")

      {:error, reason} ->
        IO.puts("Error: #{inspect(reason)}")
    end
  end

  defp print_context(ctx, tier) do
    IO.puts("Title:    #{ctx.title}")
    IO.puts("URI:      #{ctx.uri}")
    IO.puts("Type:     #{ctx.type}")
    IO.puts("Node:     #{ctx.node}")

    if ctx.signal do
      IO.puts("Genre:    #{ctx.signal.genre}")
      IO.puts("Mode:     #{ctx.signal.mode}")
      IO.puts("SigType:  #{ctx.signal.type}")
      IO.puts("S/N:      #{ctx.sn_ratio}")
    end

    if ctx.path do
      IO.puts("Path:     #{ctx.path}")
    end

    IO.puts("Modified: #{format_date(ctx.modified_at)}")

    if ctx.entities != [] do
      IO.puts("Entities: #{Enum.join(ctx.entities, ", ")}")
    end

    IO.puts("")

    case tier do
      "l0" ->
        IO.puts("## L0 Abstract\n")
        IO.puts(ctx.l0_abstract)

      "l1" ->
        IO.puts("## L1 Overview\n")
        IO.puts(ctx.l1_overview)

      _ ->
        IO.puts("## Full Content\n")
        IO.puts(ctx.content)
    end
  end

  defp format_date(nil), do: "unknown"
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d %H:%M")
end
