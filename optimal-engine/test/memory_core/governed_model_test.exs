defmodule OptimalEngine.MemoryCore.GovernedModelTest do
  @moduledoc """
  Proves the engine's own model calls are routed through governance and land a
  `model_call_run` row with the right model id + status.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.GovernedModel
  alias OptimalEngine.Store

  setup do
    # Default config is on; make sure no leftover override from another test.
    Application.put_env(:optimal_engine, :governance, log_model_calls: true)
    :ok
  end

  defp runs_for(function_name) do
    {:ok, rows} =
      Store.raw_query(
        """
        SELECT function_name, run_status, decision_state, latency_ms, metadata
        FROM model_call_runs
        WHERE function_name = ?1
        ORDER BY rowid DESC
        """,
        [function_name]
      )

    rows
  end

  test "a successful (vector) model call records a completed model_call_run with the model id" do
    fn_name = "test.embed_text_#{System.unique_integer([:positive])}"
    vector = Enum.map(1..8, fn _ -> :rand.uniform() end)

    result =
      GovernedModel.call_model(
        fn_name,
        %{"chars" => 5},
        fn -> {:ok, vector} end,
        model_id: "nomic-embed-text",
        model_task_type: "embedding"
      )

    # Wrapped result returned verbatim.
    assert {:ok, ^vector} = result

    rows = runs_for(fn_name)
    assert [[^fn_name, run_status, decision_state, latency_ms, metadata_json] | _] = rows
    assert run_status == "completed"
    assert decision_state == "allowed"
    assert is_integer(latency_ms)

    metadata = Jason.decode!(metadata_json)
    assert metadata["model_id"] == "nomic-embed-text"
    assert metadata["model_task_type"] == "embedding"
    assert metadata["call_status"] == "completed"
  end

  test "a failed provider call records a failed model_call_run but still returns the error" do
    fn_name = "test.embed_text_fail_#{System.unique_integer([:positive])}"

    result =
      GovernedModel.call_model(
        fn_name,
        %{"chars" => 5},
        fn -> {:error, :ollama_unavailable} end,
        model_id: "nomic-embed-text",
        model_task_type: "embedding"
      )

    assert {:error, :ollama_unavailable} = result

    rows = runs_for(fn_name)
    assert [[^fn_name, _run_status, _decision_state, _latency, metadata_json] | _] = rows

    # Provider-level failure is captured in run provenance metadata (the
    # governance lifecycle run_status reflects the control-plane decision,
    # not the upstream provider outcome).
    metadata = Jason.decode!(metadata_json)
    assert metadata["call_status"] == "failed"
    assert metadata["model_id"] == "nomic-embed-text"
  end

  test "disabling the flag bypasses governance entirely (no run row, result passthrough)" do
    Application.put_env(:optimal_engine, :governance, log_model_calls: false)
    on_exit(fn -> Application.put_env(:optimal_engine, :governance, log_model_calls: true) end)

    fn_name = "test.disabled_#{System.unique_integer([:positive])}"

    assert {:ok, :payload} =
             GovernedModel.call_model(fn_name, %{}, fn -> {:ok, :payload} end,
               model_id: "nomic-embed-text"
             )

    assert runs_for(fn_name) == []
  end
end
