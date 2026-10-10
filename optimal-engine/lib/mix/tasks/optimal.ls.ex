defmodule Mix.Tasks.Optimal.Ls do
  @shortdoc "List contexts under an optimal:// URI"
  @moduledoc """
  Lists all indexed contexts reachable under an `optimal://` URI prefix.

  Usage:
      mix optimal.ls "optimal://nodes/"
      mix optimal.ls "optimal://nodes/project-platform-launch/"
      mix optimal.ls "optimal://resources/"
      mix optimal.ls "optimal://nodes/operator/" --limit 20

  Options:
    --limit   Max results to display (default 50)
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        switches: [limit: :integer],
        aliases: [l: :limit]
      )

    uri_prefix =
      case positional do
        [u | _] -> u
        [] -> "optimal://nodes/"
      end

    limit = Keyword.get(opts, :limit, 50)

    IO.puts("\n[optimal.ls] #{uri_prefix}\n")

    case OptimalEngine.ls(uri_prefix, limit: limit) do
      {:ok, []} ->
        IO.puts("  (no contexts found)")

      {:ok, contexts} ->
        Enum.each(contexts, &print_context_line/1)
        IO.puts("\n#{length(contexts)} context(s)")

      {:error, reason} ->
        IO.puts("Error: #{inspect(reason)}")
    end
  end

  defp print_context_line(ctx) do
    date = format_date(ctx.modified_at)
    type_tag = "[#{ctx.type}]"
    genre_str = if ctx.signal, do: " #{ctx.signal.genre}", else: ""
    IO.puts("  #{type_tag}#{genre_str} #{ctx.title} (#{ctx.node} | #{date})")
    IO.puts("  #{ctx.uri}")
  end

  defp format_date(nil), do: "unknown"
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d")
end
