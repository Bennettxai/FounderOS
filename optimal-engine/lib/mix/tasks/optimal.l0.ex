defmodule Mix.Tasks.Optimal.L0 do
  @shortdoc "Generate and print the L0 always-loaded context"
  @moduledoc """
  Generates and prints the L0 context — the always-loaded minimal context
  (~2000 tokens) that every agent interaction should begin with.

  Usage:
      mix optimal.l0
      mix optimal.l0 --refresh    # force rebuild before printing
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, _, _} = OptionParser.parse(args, switches: [refresh: :boolean])

    if Keyword.get(opts, :refresh, false) do
      OptimalEngine.Retrieval.L0Cache.refresh()
      Process.sleep(200)
    end

    content = OptimalEngine.Retrieval.L0Cache.get()

    if content == "" do
      IO.puts("[optimal.l0] Cache empty — run `mix optimal.index` first")
    else
      IO.puts(content)
    end
  end
end
