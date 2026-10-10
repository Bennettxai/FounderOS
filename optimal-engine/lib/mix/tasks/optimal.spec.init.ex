defmodule Mix.Tasks.Optimal.Spec.Init do
  @shortdoc "Scaffold the .spec/ directory with templates and a starter spec"
  @moduledoc """
  Creates the `.spec/` directory structure at the Optimal Engine root with:
  - `.spec/specs/` — where spec files live
  - `.spec/templates/` — reusable spec templates
  - A starter spec for the intake pipeline

  Usage:
      mix optimal.spec.init
  """

  use Mix.Task

  @impl Mix.Task
  def run(_args) do
    root = Application.get_env(:optimal_engine, :root_path, "..")
    spec_dir = Path.join(root, ".spec")
    specs_dir = Path.join(spec_dir, "specs")
    templates_dir = Path.join(spec_dir, "templates")

    IO.puts("\n[spec.init] Initializing spec-led development...\n")

    File.mkdir_p!(specs_dir)
    File.mkdir_p!(templates_dir)
    IO.puts("  + #{specs_dir}/")
    IO.puts("  + #{templates_dir}/")

    # Write template
    template_path = Path.join(templates_dir, "module.spec.md")

    unless File.exists?(template_path) do
      File.write!(template_path, module_template())
      IO.puts("  + #{template_path}")
    end

    # Write starter spec
    starter_path = Path.join(specs_dir, "intake-pipeline.spec.md")

    unless File.exists?(starter_path) do
      File.write!(starter_path, starter_spec())
      IO.puts("  + #{starter_path}")
    end

    IO.puts("\n[spec.init] Done. Run `mix optimal.spec.check` to verify.\n")
  end

  defp module_template do
    """
    # {Module Name}

    ```spec-meta
    id: {module-id}
    kind: module
    status: draft
    surface: []
    verification_minimum_strength: claimed
    ```

    ```spec-requirements
    - id: {req-id}
      statement: {The module shall...}
      priority: must
      stability: evolving
    ```

    ```spec-scenarios
    - id: {scenario-id}
      covers: [{req-id}]
      given: [{precondition}]
      when: [{action}]
      then: [{expected outcome}]
    ```

    ```spec-verification
    - kind: source_file
      target: {path/to/source.ex}
      covers: [{req-id}]
    ```
    """
  end

  defp starter_spec do
    """
    # Intake Pipeline

    ```spec-meta
    id: intake-pipeline
    kind: module
    status: active
    node: entity-company
    surface:
      - engine/lib/optimal_engine/intake.ex
      - engine/lib/optimal_engine/intake/writer.ex
      - engine/lib/optimal_engine/intake/skeleton.ex
    verification_minimum_strength: linked
    ```

    ```spec-requirements
    - id: classify_signal
      statement: Intake shall classify raw text into Mode + Genre + Type + Format + Structure dimensions via Classifier
      priority: must
      stability: stable

    - id: route_to_node
      statement: Intake shall route classified signals to a primary node with optional cross-references
      priority: must
      stability: stable

    - id: write_signal_files
      statement: Intake shall write structured markdown files with YAML frontmatter to the routed node folder
      priority: must
      stability: stable

    - id: apply_genre_skeleton
      statement: Writer shall apply the genre-specific skeleton template to signal content
      priority: must
      stability: stable

    - id: index_after_write
      statement: Intake shall index the written context in the SQLite store after writing to disk
      priority: must
      stability: stable

    - id: reject_low_sn
      statement: Intake shall reject signals with S/N ratio below 0.3
      priority: must
      stability: stable
    ```

    ```spec-scenarios
    - id: ingest_text_end_to_end
      covers: [classify_signal, route_to_node, write_signal_files, index_after_write]
      given: [raw text input with entity mentions]
      when: [OptimalEngine.Pipeline.Intake.process/2 is called]
      then: [signal is classified, routed, written to disk, and indexed in SQLite]

    - id: genre_skeleton_applied
      covers: [apply_genre_skeleton]
      given: [a classified signal with genre "transcript"]
      when: [Writer renders the signal to markdown]
      then: [output contains genre-specific sections from Skeleton]

    - id: low_sn_rejected
      covers: [reject_low_sn]
      given: [input text with S/N ratio below 0.3]
      when: [Intake processes the text]
      then: [signal is rejected with quality_action :rejected]
    ```

    ```spec-verification
    - kind: source_file
      target: engine/lib/optimal_engine/intake.ex
      covers: [classify_signal, route_to_node, index_after_write, reject_low_sn]

    - kind: source_file
      target: engine/lib/optimal_engine/intake/writer.ex
      covers: [write_signal_files, apply_genre_skeleton]

    - kind: source_file
      target: engine/lib/optimal_engine/intake/skeleton.ex
      covers: [apply_genre_skeleton]

    - kind: test_file
      target: engine/test/intake_test.exs
      covers: [classify_signal, route_to_node, write_signal_files, index_after_write]
    ```
    """
  end
end
