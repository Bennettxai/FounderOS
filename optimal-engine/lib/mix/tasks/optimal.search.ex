defmodule Mix.Tasks.Optimal.Search do
  @shortdoc "Search indexed contexts with hybrid BM25 + temporal scoring"
  @moduledoc """
  Searches the context store using hybrid BM25 + temporal decay + S/N scoring.

  Usage:
      mix optimal.search "query"
      mix optimal.search "query" --node operator --limit 5
      mix optimal.search "query" --type signal --genre spec
      mix optimal.search "query" --type resource
      mix optimal.search "query" --uri "optimal://nodes/project-platform-launch/"
      mix optimal.search "query" --workspace default:my-workspace

  Options:
    --workspace  Filter by workspace ID (default: default)
    --node       Filter by node ID (e.g. operator, product-customer-portal)
    --type       Filter by context type: signal, resource, memory, skill (default: all)
    --genre      Filter by genre (signals only, e.g. spec, brief, decision-log)
    --uri        Scope to a URI prefix (e.g. optimal://nodes/project-platform-launch/)
    --limit      Max results (default 10)
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args,
        switches: [
          workspace: :string,
          node: :string,
          type: :string,
          genre: :string,
          uri: :string,
          limit: :integer
        ],
        aliases: [w: :workspace, n: :node, t: :type, g: :genre, l: :limit]
      )

    query =
      case positional do
        [q | _] -> q
        [] -> Mix.raise("Usage: mix optimal.search \"query\"")
      end

    search_opts =
      opts
      |> Keyword.take([:node, :genre, :uri, :limit])
      |> Keyword.put_new(:limit, 10)
      |> maybe_put_workspace(Keyword.get(opts, :workspace))
      |> add_type_filter(Keyword.get(opts, :type))

    IO.puts("\n[optimal.search] Query: \"#{query}\"")

    if filter = Keyword.get(search_opts, :node) do
      IO.puts("  Node filter: #{filter}")
    end

    if type_filter = Keyword.get(search_opts, :type) do
      IO.puts("  Type filter: #{type_filter}")
    end

    if uri = Keyword.get(search_opts, :uri) do
      IO.puts("  URI scope:   #{uri}")
    end

    if workspace = Keyword.get(search_opts, :workspace_id) do
      IO.puts("  Workspace:   #{workspace}")
    end

    IO.puts("")

    case OptimalEngine.Retrieval.Search.search(query, search_opts) do
      {:ok, []} ->
        IO.puts("No results found.")

      {:ok, results} ->
        Enum.each(results, &print_result/1)
        IO.puts("\n#{length(results)} result(s)")

      {:error, reason} ->
        IO.puts("Search failed: #{inspect(reason)}")
    end
  end

  defp add_type_filter(opts, nil), do: opts

  defp add_type_filter(opts, type_str) do
    case type_str do
      t when t in ~w[signal resource memory skill] ->
        Keyword.put(opts, :type, String.to_atom(t))

      other ->
        IO.puts("[optimal.search] Warning: unknown type '#{other}' — ignoring")
        opts
    end
  end

  defp maybe_put_workspace(opts, nil), do: opts
  defp maybe_put_workspace(opts, workspace_id), do: Keyword.put(opts, :workspace_id, workspace_id)

  defp print_result(ctx) do
    date = format_date(ctx.modified_at)
    score = Float.round(ctx.score || 0.0, 3)
    type_tag = "[#{ctx.type}]"

    IO.puts("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    IO.puts("  #{ctx.title}")

    genre_str = if ctx.signal, do: " | #{ctx.signal.genre}", else: ""
    IO.puts("  #{type_tag}#{genre_str} | #{ctx.node} | #{date} | score: #{score}")
    IO.puts("  #{ctx.l0_abstract}")

    if ctx.l1_overview && ctx.l1_overview != "" do
      excerpt = String.slice(ctx.l1_overview, 0, 200)
      IO.puts("\n  #{excerpt}")
    end

    IO.puts("  URI:  #{ctx.uri}")

    if ctx.path && ctx.path != "" do
      IO.puts("  Path: #{ctx.path}")
    end
  end

  defp format_date(nil), do: "unknown"
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d")
end
