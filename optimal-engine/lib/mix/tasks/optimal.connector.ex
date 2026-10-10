defmodule Mix.Tasks.Optimal.Connector do
  @shortdoc "Register / list / run enterprise connectors"

  @moduledoc """
  Operate on enterprise connectors.

  ## Usage

      mix optimal.connector list                           — show every registered adapter
      mix optimal.connector register <id> --kind slack --config path/to/config.json
      mix optimal.connector run <id>                       — run one sync cycle
      mix optimal.connector run <id> --max-retries 3

  ## Options

    --kind         <atom>    — adapter kind (required for `register`)
    --config       <path>    — JSON file with the adapter's config (required for `register`)
    --tenant       <id>      — tenant scope (default: default)
    --max-retries  <n>       — runner retry cap for transient errors (default 5)
  """

  use Mix.Task

  alias OptimalEngine.Connectors
  alias OptimalEngine.Connectors.Registry

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, _} =
      OptionParser.parse(args,
        strict: [kind: :string, config: :string, tenant: :string, max_retries: :integer]
      )

    case rest do
      ["list" | _] -> do_list()
      ["register", id | _] -> do_register(id, parsed)
      ["run", id | _] -> do_run(id, parsed)
      _ -> Mix.raise("Usage: mix optimal.connector list | register <id> | run <id>")
    end
  end

  defp do_list do
    IO.puts("Available adapters (14 total):\n")

    Registry.summary()
    |> Enum.each(fn {kind, name, auth} ->
      IO.puts("  #{pad(kind)} #{pad(name, 22)} auth=#{auth}")
    end)
  end

  defp do_register(id, parsed) do
    kind = parsed |> Keyword.fetch!(:kind) |> String.to_atom()
    path = Keyword.fetch!(parsed, :config)
    tenant = Keyword.get(parsed, :tenant, "default")

    config =
      path
      |> File.read!()
      |> Jason.decode!()

    {:ok, id} = Connectors.register(%{id: id, kind: kind, config: config, tenant_id: tenant})
    IO.puts("Registered connector #{id} (#{kind}).")
  end

  defp do_run(id, parsed) do
    opts = [max_retries: Keyword.get(parsed, :max_retries, 5)]

    case Connectors.run(id, opts) do
      {:ok, result} ->
        IO.puts("Run complete: #{result.status}")
        IO.puts("  signals:      #{result.signals}")
        IO.puts("  errors:       #{result.errors}")
        IO.puts("  cursor_before: #{inspect(result.cursor_before)}")
        IO.puts("  cursor_after:  #{inspect(result.cursor_after)}")
        if result.reason, do: IO.puts("  reason:       #{inspect(result.reason)}")

      {:error, reason} ->
        Mix.raise("Connector run failed: #{inspect(reason)}")
    end
  end

  defp pad(value, width \\ 14) do
    value
    |> to_string()
    |> String.pad_trailing(width)
  end
end
