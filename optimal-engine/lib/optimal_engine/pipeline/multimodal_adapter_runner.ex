defmodule OptimalEngine.Pipeline.MultimodalAdapterRunner do
  @moduledoc """
  Executes configured multimodal adapter commands against governed assets.

  The runner keeps the evidence lifecycle intact:

      asset -> adapter command -> asset_adapter_run -> optional asset_extraction
            -> derivation ledger

  Missing tools and failed commands are recorded as adapter runs instead of
  becoming invisible failures.
  """

  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Pipeline.MultimodalExtractionParser
  alias OptimalEngine.Pipeline.MultimodalToolRegistry

  @type run_result :: {:ok, map()} | {:error, term()}

  @spec run(String.t(), atom() | String.t(), keyword()) :: run_result()
  def run(asset_id, adapter_id, opts \\ []) when is_binary(asset_id) do
    opts = Keyword.put(opts, :adapter_id, adapter_id)

    with {:ok, asset} <- MemoryCore.get_asset(asset_id, scope_opts(opts)),
         {:ok, tool} <- get_tool(adapter_id) do
      execute_or_record(asset, tool, opts)
    end
  end

  @spec run_recommended(String.t(), atom(), keyword()) :: run_result()
  def run_recommended(asset_id, modality, opts \\ [])
      when is_binary(asset_id) and is_atom(modality) do
    case MultimodalToolRegistry.recommended_tool_ids(modality) do
      [adapter_id | _] -> run(asset_id, adapter_id, Keyword.put(opts, :modality, modality))
      [] -> {:error, {:no_recommended_adapter, modality}}
    end
  end

  defp execute_or_record(asset, tool, opts) do
    command = Keyword.get(opts, :command) || tool.command

    cond do
      is_nil(command) ->
        record(asset, tool, opts,
          status: "unavailable",
          error_reason: "adapter has no local command configured"
        )

      System.find_executable(command) == nil ->
        record(asset, tool, opts,
          status: "unavailable",
          error_reason: "command not found: #{command}"
        )

      true ->
        execute_command(asset, tool, command, opts)
    end
  end

  defp execute_command(asset, tool, command, opts) do
    args =
      Keyword.get(opts, :args) || MultimodalToolRegistry.default_args(tool.id, asset.storage_path)

    case System.cmd(command, args, stderr_to_stdout: true) do
      {output, 0} ->
        record(asset, tool, opts,
          status: "completed",
          output_text: normalize_output(output),
          metadata: command_metadata(command, args, opts)
        )

      {output, exit_status} ->
        record(asset, tool, opts,
          status: "failed",
          output_text: normalize_output(output),
          error_reason: "command exited with status #{exit_status}",
          metadata: command_metadata(command, args, opts)
        )
    end
  rescue
    error ->
      record(asset, tool, opts,
        status: "failed",
        error_reason: Exception.message(error),
        metadata: command_metadata(command, Keyword.get(opts, :args, []), opts)
      )
  end

  defp record(asset, tool, opts, attrs) do
    metadata =
      opts
      |> Keyword.get(:metadata, %{})
      |> Map.merge(Keyword.get(attrs, :metadata, %{}))
      |> stringify_keys()

    run_opts =
      Keyword.merge(scope_opts(opts),
        actor_id: Keyword.get(opts, :actor_id),
        adapter_id: tool.id,
        adapter_role: adapter_role(tool, opts),
        modality: Keyword.get(opts, :modality, asset.modality),
        status: Keyword.fetch!(attrs, :status),
        output_text: Keyword.get(attrs, :output_text, ""),
        output_ref: Keyword.get(opts, :output_ref),
        model_id: Keyword.get(opts, :model_id, Atom.to_string(tool.id)),
        model_version: Keyword.get(opts, :model_version),
        confidence: Keyword.get(opts, :confidence),
        precision: Keyword.get(opts, :precision),
        error_reason: Keyword.get(attrs, :error_reason),
        metadata: metadata
      )

    with {:ok, run} <- MemoryCore.record_asset_adapter_run(asset.id, run_opts),
         :ok <- maybe_record_extraction_projection(run, tool, opts) do
      {:ok, run}
    end
  end

  defp maybe_record_extraction_projection(%{status: "completed"} = run, tool, opts) do
    if Keyword.get(opts, :auto_extract, true) do
      run
      |> MultimodalExtractionParser.parse(tool, opts)
      |> record_extraction_projections(run)
    else
      :ok
    end
  end

  defp maybe_record_extraction_projection(_run, _tool, _opts), do: :ok

  defp record_extraction_projections([], _run), do: :ok

  defp record_extraction_projections(projection_opts_list, run) do
    Enum.reduce_while(projection_opts_list, :ok, fn extraction_opts, :ok ->
      case MemoryCore.record_asset_extraction(run.id, extraction_opts) do
        {:ok, _result} -> {:cont, :ok}
        {:error, reason} -> {:halt, {:error, {:asset_extraction_projection_failed, reason}}}
      end
    end)
  end

  defp adapter_role(tool, opts) do
    Keyword.get(opts, :adapter_role) || tool.profile.primary_role || List.first(tool.roles)
  end

  defp command_metadata(command, args, opts) do
    %{
      "command" => command,
      "args" => args,
      "adapter_profile" => stringify_keys(Keyword.get(opts, :adapter_profile, %{})),
      "configured_by" => Keyword.get(opts, :configured_by, "runtime")
    }
  end

  defp normalize_output(output) when is_binary(output) do
    output
    |> String.trim()
    |> String.slice(0, 200_000)
  end

  defp stringify_keys(map) when is_map(map) do
    Map.new(map, fn {key, value} -> {to_string(key), value} end)
  end

  defp get_tool(adapter_id) when is_atom(adapter_id) do
    case MultimodalToolRegistry.get(adapter_id) do
      nil -> {:error, {:unknown_adapter, adapter_id}}
      tool -> {:ok, tool}
    end
  end

  defp get_tool(adapter_id) when is_binary(adapter_id) do
    case MultimodalToolRegistry.get(adapter_id) do
      nil -> {:error, {:unknown_adapter, adapter_id}}
      tool -> {:ok, tool}
    end
  end

  defp scope_opts(opts) do
    opts
    |> Keyword.take([:tenant_id, :workspace_id])
    |> Keyword.put_new(:tenant_id, "default")
    |> Keyword.put_new(:workspace_id, "default")
  end
end
