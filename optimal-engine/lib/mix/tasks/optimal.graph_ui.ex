defmodule Mix.Tasks.Optimal.GraphUi do
  use Mix.Task
  @shortdoc "Start the API and open the knowledge graph visualizer"

  @moduledoc """
  Starts the OptimalEngine API server and opens the knowledge graph
  visualizer (graph.html) in the default browser.

      $ mix optimal.graph_ui

  The API must be reachable at http://localhost:4200. The visualizer
  is served as a static file opened directly from the filesystem.
  """

  def run(_args) do
    Mix.Task.run("app.start")

    priv_dir = :code.priv_dir(:optimal_engine) |> to_string()
    static_dir = Path.join(priv_dir, "static")
    graph_path = Path.join(static_dir, "graph.html")

    unless File.exists?(graph_path) do
      Mix.raise("""
      graph.html not found at #{graph_path}.
      Expected it at priv/static/graph.html in the :optimal_engine application.
      """)
    end

    IO.puts("[OptimalEngine] API running on http://localhost:4200")
    IO.puts("[OptimalEngine] Graph visualizer: #{graph_path}")
    IO.puts("[OptimalEngine] Opening in default browser…")

    open_browser(graph_path)

    IO.puts("[OptimalEngine] Press Ctrl+C to stop.")
    Process.sleep(:infinity)
  end

  defp open_browser(path) do
    case :os.type() do
      {:unix, :darwin} ->
        System.cmd("open", [path], stderr_to_stdout: true)

      {:unix, _} ->
        System.cmd("xdg-open", [path], stderr_to_stdout: true)

      {:win32, _} ->
        System.cmd("cmd", ["/c", "start", path], stderr_to_stdout: true)

      other ->
        IO.puts("[OptimalEngine] Unsupported OS #{inspect(other)}. Open manually: #{path}")
    end
  end
end
