defmodule Mix.Tasks.Optimal.Init do
  @shortdoc "Scaffold a new workspace directory from the sample template"

  @moduledoc """
  Copies the repo's `sample-workspace/` layout to a target directory
  so you can start filling it with real signals without hand-typing
  the folder skeleton.

  ## Usage

      mix optimal.init <target-dir>
      mix optimal.init ~/Desktop/my-engine

  ## What gets written

      <target-dir>/
      ├── README.md                (how-to-use, same as the sample)
      ├── nodes/
      │   ├── entity-company/context.md + signal.md
      │   ├── product-customer-portal/context.md + signal.md
      │   └── …
      ├── .wiki/SCHEMA.md
      ├── architectures/customer_requirement.yaml
      └── assets/README.md

  Existing files in `<target-dir>` are left untouched — the task will
  refuse to overwrite anything. Pass `--force` to overwrite.

  Use the example signals under each node as templates for your own
  entries; then ingest the whole workspace with
  `mix optimal.ingest_workspace <target-dir>`.
  """

  use Mix.Task

  @impl Mix.Task
  def run(args) do
    {parsed, rest, _} = OptionParser.parse(args, strict: [force: :boolean])
    force? = Keyword.get(parsed, :force, false)

    target =
      case rest do
        [t | _] -> Path.expand(t)
        [] -> Mix.raise("Usage: mix optimal.init <target-dir> [--force]")
      end

    source = Path.join(File.cwd!(), "sample-workspace")

    unless File.dir?(source) do
      Mix.raise("""
      sample-workspace/ not found at #{source}.
      Run this task from the repo root, or from a checkout that
      contains the sample workspace.
      """)
    end

    File.mkdir_p!(target)
    copied = copy_tree(source, target, force?)

    Mix.shell().info("""

      Scaffolded #{copied} files into:
        #{target}

      Next:
        cd #{target}
        # edit the sample signals, then:
        cd -
        mix optimal.ingest_workspace #{target}
        mix optimal.rag "your question" --trace
    """)
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp copy_tree(source, target, force?) do
    source
    |> File.ls!()
    |> Enum.reduce(0, fn name, acc ->
      src = Path.join(source, name)
      dst = Path.join(target, name)

      cond do
        File.dir?(src) ->
          File.mkdir_p!(dst)
          acc + copy_tree(src, dst, force?)

        File.exists?(dst) and not force? ->
          Mix.shell().info("  skip (exists): #{Path.relative_to_cwd(dst)}")
          acc

        true ->
          File.cp!(src, dst)
          Mix.shell().info("  wrote:        #{Path.relative_to_cwd(dst)}")
          acc + 1
      end
    end)
  end
end
