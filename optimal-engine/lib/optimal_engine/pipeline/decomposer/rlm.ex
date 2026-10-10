defmodule OptimalEngine.Pipeline.Decomposer.RLM do
  @moduledoc """
  Optional Recursive Language Model decomposition adapter.

  The adapter delegates recursive execution to the bundled DSPy sidecar and
  returns atomic units with source provenance. It never replaces the normal
  deterministic decomposer: callers must fall back when this module returns
  an error.
  """

  @default_timeout 120_000

  @spec available?(keyword()) :: boolean()
  def available?(opts \\ []) do
    {executable, _args} = command(opts)
    not is_nil(System.find_executable(executable)) and File.regular?(script(opts))
  end

  @spec decompose(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def decompose(content, opts \\ []) when is_binary(content) do
    if available?(opts) do
      payload = %{
        title: Keyword.get(opts, :title, "Untitled source"),
        content: content,
        related_titles: Keyword.get(opts, :related_titles, []),
        model: Keyword.get(opts, :model) || System.get_env("OPTIMAL_RLM_MODEL"),
        sub_model: Keyword.get(opts, :sub_model) || System.get_env("OPTIMAL_RLM_SUB_MODEL"),
        max_iterations: Keyword.get(opts, :max_iterations, 15),
        max_llm_calls: Keyword.get(opts, :max_llm_calls, 30),
        max_output_chars: Keyword.get(opts, :max_output_chars, 15_000)
      }

      invoke(payload, opts)
    else
      {:error, :rlm_unavailable}
    end
  end

  @spec health(keyword()) :: {:ok, map()} | {:error, term()}
  def health(opts \\ []) do
    {executable, prefix_args} = command(opts)

    case System.cmd(executable, prefix_args ++ [script(opts), "--check"], stderr_to_stdout: true) do
      {output, 0} ->
        case Jason.decode(output) do
          {:ok, %{"available" => true} = status} -> {:ok, status}
          {:ok, status} -> {:error, {:rlm_unavailable, status}}
          {:error, reason} -> {:error, {:invalid_rlm_health, reason}}
        end

      {output, status} ->
        {:error, {:rlm_health_exit, status, String.trim(output)}}
    end
  end

  defp invoke(payload, opts) do
    {executable, prefix_args} = command(opts)
    args = prefix_args ++ [script(opts)]
    timeout = Keyword.get(opts, :timeout, @default_timeout)
    runner = Keyword.get(opts, :runner, &System.cmd/3)

    task =
      Task.async(fn ->
        runner.(executable, args,
          input: Jason.encode!(payload),
          stderr_to_stdout: true
        )
      end)

    case Task.yield(task, timeout) || Task.shutdown(task, :brutal_kill) do
      {:ok, {output, 0}} -> decode(output)
      {:ok, {output, status}} -> {:error, {:rlm_exit, status, String.trim(output)}}
      nil -> {:error, :rlm_timeout}
    end
  end

  defp decode(output) do
    with {:ok, result} <- Jason.decode(output),
         atoms when is_list(atoms) and atoms != [] <- result["atoms"] do
      {:ok, result}
    else
      {:error, reason} -> {:error, {:invalid_rlm_json, reason}}
      _ -> {:error, :empty_rlm_output}
    end
  end

  defp command(opts) do
    case Keyword.get(opts, :command) || System.get_env("OPTIMAL_RLM_COMMAND") do
      nil -> {default_python(), []}
      command -> {"sh", ["-c", command <> " \"$@\"", "optimal-rlm"]}
    end
  end

  defp default_python do
    configured = System.get_env("OPTIMAL_RLM_PYTHON")
    local = Path.expand("../../../../.optimal/rlm-venv/bin/python", __DIR__)

    cond do
      is_binary(configured) and configured != "" -> configured
      File.regular?(local) -> local
      true -> "python3"
    end
  end

  defp script(opts) do
    Keyword.get(opts, :script) ||
      Application.app_dir(:optimal_engine, "priv/rlm/decompose.py")
  end
end
