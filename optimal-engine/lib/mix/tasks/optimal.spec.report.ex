defmodule Mix.Tasks.Optimal.Spec.Report do
  @shortdoc "Coverage and verification summary for all specs"
  @moduledoc """
  Generates a human-readable report of spec coverage and verification status.

  Shows:
  - Which source files are covered by specs
  - Which source files have NO spec coverage
  - Overall coverage percentage
  - Verification strength distribution

  Usage:
      mix optimal.spec.report
      mix optimal.spec.report --source-dir ../engine/lib
  """

  use Mix.Task

  alias OptimalEngine.Spec.{Coverage, State}

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _, _} = OptionParser.parse(args, strict: [spec_dir: :string, source_dir: :string])

    root = Application.get_env(:optimal_engine, :root_path, "..")
    spec_dir = parsed[:spec_dir] || Path.join(root, ".spec/specs")
    source_dir = parsed[:source_dir] || Path.join(root, "engine/lib")

    IO.puts("\n[spec.report] Spec Coverage & Verification Report\n")

    unless File.dir?(spec_dir) do
      IO.puts("  No .spec/specs/ directory found. Run `mix optimal.spec.init` first.\n")
      return_or_halt()
    end

    # Coverage analysis (Coverage.analyze spec returns {:ok, report} only;
    # parse failures propagate as raises from the underlying Parser.parse_all).
    {:ok, report} = Coverage.analyze(spec_dir, source_dir)
    print_coverage(report)

    # State summary (if exists)
    case State.read() do
      {:ok, state} ->
        print_verification(state)

      {:error, :not_found} ->
        IO.puts("  No spec state found. Run `mix optimal.spec.check` first.\n")

      {:error, reason} ->
        IO.puts("  State read error: #{inspect(reason)}\n")
    end
  end

  defp print_coverage(report) do
    color =
      cond do
        report.percentage >= 80.0 -> IO.ANSI.green()
        report.percentage >= 50.0 -> IO.ANSI.yellow()
        true -> IO.ANSI.red()
      end

    IO.puts("  #{IO.ANSI.cyan()}COVERAGE#{IO.ANSI.reset()}")
    IO.puts("  " <> String.duplicate("-", 50))
    IO.puts("  Specs:            #{report.specs}")
    IO.puts("  Source files:     #{report.total_source}")
    IO.puts("  Covered:          #{report.total_covered}")
    IO.puts("  Uncovered:        #{report.total_source - report.total_covered}")
    IO.puts("  Coverage:         #{color}#{report.percentage}%#{IO.ANSI.reset()}")
    IO.puts("")

    if report.uncovered != [] do
      IO.puts("  #{IO.ANSI.yellow()}Uncovered files:#{IO.ANSI.reset()}")

      report.uncovered
      |> Enum.take(20)
      |> Enum.each(fn f ->
        IO.puts("    - #{Path.basename(f)}")
      end)

      remaining = length(report.uncovered) - 20
      if remaining > 0, do: IO.puts("    ... and #{remaining} more")
      IO.puts("")
    end
  end

  defp print_verification(state) do
    summary = state["summary"] || %{}
    coverage = get_in(state, ["verification", "coverage"]) || %{}

    IO.puts("  #{IO.ANSI.cyan()}VERIFICATION#{IO.ANSI.reset()}")
    IO.puts("  " <> String.duplicate("-", 50))
    IO.puts("  Subjects:         #{summary["subjects"] || 0}")
    IO.puts("  Requirements:     #{summary["requirements"] || 0}")
    IO.puts("  Passing:          #{IO.ANSI.green()}#{summary["passing"] || 0}#{IO.ANSI.reset()}")
    IO.puts("  Failing:          #{IO.ANSI.red()}#{summary["failing"] || 0}#{IO.ANSI.reset()}")
    IO.puts("")
    IO.puts("  Strength distribution:")
    IO.puts("    Claimed:   #{coverage["claimed"] || 0}")
    IO.puts("    Linked:    #{coverage["linked"] || 0}")
    IO.puts("    Executed:  #{coverage["executed"] || 0}")
    IO.puts("")

    if generated_at = state["generated_at"] do
      IO.puts("  Last verified: #{generated_at}\n")
    end
  end

  defp return_or_halt do
    if Mix.env() == :test, do: :ok, else: System.halt(1)
  end
end
