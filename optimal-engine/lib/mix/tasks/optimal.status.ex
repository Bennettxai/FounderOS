defmodule Mix.Tasks.Optimal.Status do
  @shortdoc "Print engine liveness + readiness (supervisor, store, migrations, embedder)"

  @moduledoc """
  Engine runtime status — for runbooks, CI deploys, status pages.

  Distinct from `mix optimal.health`, which runs 10 knowledge-base
  diagnostics. This one checks the engine *process* is healthy:
  supervisor up, store writable, migrations current, embedder
  reachable, credential key present if connectors are configured.

  ## Usage

      mix optimal.status               — full report
      mix optimal.status --json        — machine-readable
      mix optimal.status --quick       — skip slow checks (embedder)
      mix optimal.status --metrics     — include telemetry snapshot

  Exit codes:
    0  — up (every check :ok, no warnings)
    2  — degraded (at least one :warn, no :error)
    1  — down (at least one :error, or supervisor not running)
  """

  use Mix.Task

  alias OptimalEngine.Health
  alias OptimalEngine.Telemetry

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _, _} =
      OptionParser.parse(args, strict: [json: :boolean, quick: :boolean, metrics: :boolean])

    skip = if Keyword.get(parsed, :quick, false), do: [:embedder], else: []
    ready = Health.ready(skip: skip)
    status = Health.status()

    if Keyword.get(parsed, :json, false) do
      emit_json(ready, status, parsed)
    else
      emit_text(ready, status, parsed)
    end

    System.halt(exit_code(status))
  end

  defp emit_text(ready, status, parsed) do
    IO.puts("Engine status: #{status}")
    IO.puts("")
    IO.puts("Checks:")

    Enum.each(ready.checks, fn {name, result} ->
      IO.puts("  #{pad(name)} #{format_result(result)}")
    end)

    if ready.degraded != [] do
      IO.puts("")
      IO.puts("Degraded: #{Enum.join(ready.degraded, ", ")}")
    end

    if Keyword.get(parsed, :metrics, false) do
      IO.puts("")
      IO.puts("Metrics:")
      snap = safe_snapshot()
      IO.puts("  uptime_ms: #{snap.uptime_ms}")

      Enum.each(snap.counters, fn {k, v} ->
        IO.puts("  #{pad(k, 36)} #{v}")
      end)
    end
  end

  defp emit_json(ready, status, parsed) do
    payload = %{
      status: status,
      ok?: ready.ok?,
      checks: Map.new(ready.checks, fn {k, v} -> {k, format_result_json(v)} end),
      degraded: ready.degraded
    }

    payload =
      if Keyword.get(parsed, :metrics, false) do
        Map.put(payload, :metrics, safe_snapshot())
      else
        payload
      end

    IO.puts(Jason.encode!(payload))
  end

  defp format_result(:ok), do: "OK"
  defp format_result({:warn, reason}), do: "WARN — #{inspect(reason)}"
  defp format_result({:error, reason}), do: "ERROR — #{inspect(reason)}"

  defp format_result_json(:ok), do: %{status: "ok"}
  defp format_result_json({:warn, reason}), do: %{status: "warn", reason: inspect(reason)}
  defp format_result_json({:error, reason}), do: %{status: "error", reason: inspect(reason)}

  defp pad(value, width \\ 20), do: value |> to_string() |> String.pad_trailing(width)

  defp exit_code(:up), do: 0
  defp exit_code(:degraded), do: 2
  defp exit_code(:down), do: 1

  defp safe_snapshot do
    Telemetry.snapshot()
  rescue
    _ -> %{uptime_ms: 0, counters: %{}, histograms: %{}}
  catch
    :exit, _ -> %{uptime_ms: 0, counters: %{}, histograms: %{}}
  end
end
