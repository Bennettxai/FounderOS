defmodule Mix.Tasks.Optimal.Architectures do
  @shortdoc "List + inspect data architectures and their processor bindings"

  @moduledoc """
  The engine's architecture registry — what kinds of data points it
  can store, and which processors own each field.

  ## Usage

      mix optimal.architectures              — list every architecture
      mix optimal.architectures show <name>  — field-level detail
      mix optimal.architectures processors   — every registered processor

  See `OptimalEngine.Architecture` for the runtime API.
  """

  use Mix.Task

  alias OptimalEngine.Architecture

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    case args do
      [] -> list_all()
      ["show", name | _] -> show(name)
      ["processors" | _] -> list_processors()
      _ -> Mix.raise("Usage: mix optimal.architectures [show <name> | processors]")
    end
  end

  defp list_all do
    IO.puts("Registered data architectures:\n")

    Architecture.list()
    |> Enum.each(fn arch ->
      IO.puts("  #{pad(arch.name, 24)} v#{arch.version}  modality=#{arch.modality_primary}")

      if arch.description do
        IO.puts("    #{arch.description}")
      end
    end)

    IO.puts("")
    IO.puts("mix optimal.architectures show <name>    — field-level detail")
  end

  defp show(name) do
    case Architecture.fetch(name) do
      {:ok, arch} ->
        IO.puts("\n#{arch.name} (v#{arch.version})")
        IO.puts("  id:              #{arch.id}")
        IO.puts("  description:     #{arch.description || "—"}")
        IO.puts("  primary modality: #{arch.modality_primary}")
        IO.puts("  granularity:     #{Enum.join(arch.granularity, " → ")}")
        IO.puts("\n  Fields:")

        Enum.each(arch.fields, fn f ->
          required = if f.required, do: " (required)", else: ""
          processor = if f.processor, do: " ⇢ #{f.processor}", else: ""
          dims = if f.dims != [], do: " [#{Enum.map_join(f.dims, "×", &to_string/1)}]", else: ""

          IO.puts("    #{pad(f.name, 14)} #{f.modality}#{dims}#{required}#{processor}")

          if f.description do
            IO.puts("      #{f.description}")
          end
        end)

      {:error, :not_found} ->
        Mix.raise("No architecture registered as #{inspect(name)}")
    end
  end

  defp list_processors do
    IO.puts("Registered processors:\n")

    Architecture.processor_summary()
    |> Enum.each(fn {id, modality, emits} ->
      IO.puts("  #{pad(id, 24)} modality=#{pad(modality, 12)} emits=#{inspect(emits)}")
    end)
  end

  defp pad(v, w), do: v |> to_string() |> String.pad_trailing(w)
end
