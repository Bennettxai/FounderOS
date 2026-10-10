defmodule OptimalEngine.MemoryCore.GovernedModel do
  @moduledoc """
  Fail-open facade that routes the engine's OWN model/tool calls through the
  Tool/Model governance control plane.

  The engine makes external model calls (Ollama embeddings, LLM generation)
  and tool calls from many call sites. Historically those bypassed governance:
  no `model_call_run`/`tool_call_run` was ever recorded, so the audit/provenance
  layer had zero rows. This module is the seam.

  ## Contract

      GovernedModel.call_model("embed_text", input_payload, fn -> Ollama.embed_text(text) end,
        model_id: "nomic-embed-text",
        model_task_type: "embedding",
        workspace_id: "default"
      )

  It:

    1. ensures a `model_call_operation` exists for the `function_name`
       (idempotent — `content_id` keys it; re-registration is a no-op upsert);
    2. records a `model_call_run` via
       `OptimalEngine.MemoryCore.ToolModelGovernance.record_model_call/3`
       carrying provenance (model id), latency, and status;
    3. returns the WRAPPED CALL's result **unchanged**.

  ## Fail-open

  Governance is observability, not a gate, for these internal calls. If the
  governance write itself fails (DB locked, missing table, etc.) the wrapped
  result is STILL returned — embeddings and retrieval must never break because
  audit logging hiccupped. Governance failures are logged at `warning`.

  ## Config

      config :optimal_engine, :governance, log_model_calls: true   # default true

  When disabled, the wrapped function is invoked directly with zero governance
  overhead.
  """

  alias OptimalEngine.MemoryCore.ToolModelGovernance, as: Governance

  require Logger

  @doc """
  Run `fun` (a 0-arity function returning the model call's result) under
  governance. The result of `fun` is returned verbatim.

  `input_payload` is a map describing the call inputs for provenance/hashing
  (do NOT put large blobs / full image bytes here — a digest or shape is enough).

  Options:
    * `:model_id` — provider model identifier (e.g. "nomic-embed-text"). Recorded
      as provenance on both the operation and the run metadata.
    * `:model_task_type` — "embedding" | "generation" | ... (default "generation")
    * `:workspace_id` — governance scope (default "default")
    * `:actor_id` — requesting actor (default "engine")
    * `:metadata` — extra metadata merged onto the run
  """
  @spec call_model(String.t(), map(), (-> result), keyword()) :: result
        when result: term()
  def call_model(function_name, input_payload, fun, opts \\ [])
      when is_binary(function_name) and is_map(input_payload) and is_function(fun, 0) do
    if enabled?() do
      started_at = System.monotonic_time(:millisecond)
      result = fun.()
      latency_ms = max(System.monotonic_time(:millisecond) - started_at, 0)
      log_model_run(function_name, input_payload, result, latency_ms, opts)
      result
    else
      fun.()
    end
  end

  @doc "Whether internal model/tool calls should be logged through governance."
  @spec enabled?() :: boolean()
  def enabled? do
    :optimal_engine
    |> Application.get_env(:governance, [])
    |> Keyword.get(:log_model_calls, true)
  end

  # ─── internal ──────────────────────────────────────────────────────────────

  defp log_model_run(function_name, input_payload, result, latency_ms, opts) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    model_id = opts[:model_id]
    model_task_type = Keyword.get(opts, :model_task_type, "generation")

    {status, output_payload} = classify_result(result)

    record_opts =
      [
        workspace_id: workspace_id,
        actor_id: Keyword.get(opts, :actor_id, "engine"),
        latency_ms: latency_ms,
        output_payload: output_payload,
        metadata:
          Map.merge(
            %{
              model_id: model_id,
              model_task_type: model_task_type,
              provider: Keyword.get(opts, :provider, "ollama"),
              call_status: status
            },
            Keyword.get(opts, :metadata, %{})
          )
      ]

    try do
      ensure_operation(function_name, workspace_id, model_id, model_task_type)

      case Governance.record_model_call(function_name, input_payload, record_opts) do
        {:ok, _run} ->
          :ok

        {:error, reason} ->
          Logger.warning(
            "[GovernedModel] failed to record model_call_run for #{function_name}: #{inspect(reason)}"
          )
      end
    rescue
      exception ->
        Logger.warning(
          "[GovernedModel] governance logging raised for #{function_name}: " <>
            Exception.message(exception)
        )
    catch
      kind, reason ->
        Logger.warning(
          "[GovernedModel] governance logging threw (#{kind}) for #{function_name}: #{inspect(reason)}"
        )
    end
  end

  # Idempotent registration. content_id makes re-registration a stable upsert,
  # so it is safe to call on every model invocation.
  defp ensure_operation(function_name, workspace_id, model_id, model_task_type) do
    Governance.register_model_call_operation(
      workspace_id: workspace_id,
      function_name: function_name,
      model_task_type: model_task_type,
      model_id: model_id,
      metadata: %{registered_by: "governed_model"}
    )
  end

  # Map the wrapped call's result to a run status + a small output payload.
  # We never store full vectors / generated text verbatim here — only shape and
  # status — to keep run rows bounded.
  defp classify_result({:ok, vector}) when is_list(vector),
    do: {"completed", %{"ok" => true, "result_kind" => "vector", "dim" => length(vector)}}

  defp classify_result({:ok, text}) when is_binary(text),
    do: {"completed", %{"ok" => true, "result_kind" => "text", "length" => byte_size(text)}}

  defp classify_result({:ok, other}),
    do: {"completed", %{"ok" => true, "result_kind" => inspect_kind(other)}}

  defp classify_result(:ok), do: {"completed", %{"ok" => true}}

  defp classify_result({:error, reason}),
    do: {"failed", %{"ok" => false, "error" => inspect(reason)}}

  defp classify_result(other),
    do: {"completed", %{"ok" => true, "result_kind" => inspect_kind(other)}}

  defp inspect_kind(value) when is_map(value), do: "map"
  defp inspect_kind(value) when is_list(value), do: "list"
  defp inspect_kind(_), do: "scalar"
end
