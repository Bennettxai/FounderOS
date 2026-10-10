defmodule Mix.Tasks.Optimal.Spec.Drift do
  @shortdoc "Detect code changes without corresponding spec updates"
  @moduledoc """
  Compares git changes against spec surface declarations. If a source file
  changed but its governing spec didn't, that's drift — the spec may be stale.

  Exits with code 1 if drift is detected.

  Usage:
      mix optimal.spec.drift
      mix optimal.spec.drift --base HEAD~3
      mix optimal.spec.drift --base main
  """

  use Mix.Task

  alias OptimalEngine.Spec.Diffcheck

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _, _} = OptionParser.parse(args, strict: [base: :string, spec_dir: :string])

    root = Application.get_env(:optimal_engine, :root_path, "..")
    spec_dir = parsed[:spec_dir] || Path.join(root, ".spec/specs")
    opts = if base = parsed[:base], do: [base: base], else: []

    IO.puts("\n[spec.drift] Checking for spec drift...\n")

    unless File.dir?(spec_dir) do
      IO.puts("  No .spec/specs/ directory found. Run `mix optimal.spec.init` first.")
      System.halt(1)
    end

    case Diffcheck.check(spec_dir, root, opts) do
      {:ok, []} ->
        IO.puts("  #{IO.ANSI.green()}No drift detected. Specs are current.#{IO.ANSI.reset()}\n")

      {:ok, drifts} ->
        IO.puts("  #{IO.ANSI.yellow()}#{length(drifts)} drift(s) detected:#{IO.ANSI.reset()}\n")

        Enum.each(drifts, fn d ->
          IO.puts("    #{IO.ANSI.red()}x#{IO.ANSI.reset()} #{d.source_file}")
          IO.puts("      spec: #{d.spec_file} (#{d.spec_id})")
          IO.puts("      #{d.reason}")
          IO.puts("")
        end)

        IO.puts("  Update the spec files above to match code changes.\n")
        System.halt(1)

      {:error, reason} ->
        IO.puts("  [spec.drift] Error: #{inspect(reason)}\n")
        IO.puts("  This can happen on initial commits or shallow clones.\n")
    end
  end
end
