defmodule OptimalEngine.EvaluationTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Evaluation, MemoryCore, Store}

  test "records benchmark runs and per-case judge scores" do
    workspace_id = "evaluation-test-#{System.unique_integer([:positive])}"

    on_exit(fn ->
      Store.raw_execute("DELETE FROM evaluation_runs WHERE workspace_id = ?1", [workspace_id])
    end)

    assert {:ok, run} =
             Evaluation.record_run(
               workspace_id: workspace_id,
               benchmark_name: "long_memory_recall",
               dataset_name: "synthetic-company-memory",
               dataset_size: 35,
               question_count: 700,
               answer_model: "answer-model",
               judge_model: "judge-model",
               judge_strategy: "3x_majority",
               retrieval_top_k: 100,
               run_config: %{temperature: 0},
               retrieval_config: %{mode: "context_package"},
               judge_config: %{votes: 3}
             )

    assert {:ok, passed} =
             Evaluation.record_case(run.id,
               case_id: "case-001",
               conversation_id: "conv-001",
               question: "What happened in the launch review?",
               expected_answer: "The launch was approved.",
               actual_answer: "The launch was approved.",
               retrieved_object_links: [%{type: "fact", id: "fact_001"}],
               scores: %{accuracy: 1.0, grounding: 0.9},
               judge_output: %{votes: ["pass", "pass", "pass"]},
               status: "passed"
             )

    assert passed.retrieved_object_links == [%{type: "fact", id: "fact_001"}]

    assert {:ok, _failed} =
             Evaluation.record_case(run.id,
               case_id: "case-002",
               question: "What was the budget?",
               actual_answer: "Unknown.",
               scores: %{accuracy: 0.0, grounding: 0.2},
               status: "failed",
               error_reason: "missing_ground_truth"
             )

    assert {:ok, summary} = Evaluation.summarize(run.id)
    assert summary.case_count == 2
    assert summary.passed_count == 1
    assert summary.failed_count == 1
    assert_in_delta summary.aggregate_scores["accuracy"].average, 0.5, 0.001
    assert_in_delta summary.aggregate_scores["grounding"].average, 0.55, 0.001

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM evaluation_runs WHERE workspace_id = ?1 AND benchmark_name = 'long_memory_recall'",
               [workspace_id]
             )

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM evaluation_cases WHERE workspace_id = ?1 AND evaluation_run_id = ?2",
               [workspace_id, run.id]
             )
  end

  test "runs a benchmark through retrieval, deterministic judging, and persisted cases" do
    workspace_id = "evaluation-runner-test-#{System.unique_integer([:positive])}"

    on_exit(fn ->
      Store.raw_execute("DELETE FROM evaluation_runs WHERE workspace_id = ?1", [workspace_id])
    end)

    source =
      MemoryCore.source_package_from_text("The launch was approved after final review.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        security_labels: ["internal"],
        partition_ids: ["ops"]
      )

    assert {:ok, claim} =
             MemoryCore.extract_claim(source,
               claim_text: "The launch was approved after final review.",
               subject_anchor: "launch",
               action_class: "approval",
               object_anchor: "final_review",
               aggregate_confidence: 0.95,
               aggregate_precision: 0.9
             )

    assert {:ok, fact} =
             MemoryCore.promote_claim_to_fact(claim,
               fact_text: "The launch was approved.",
               aggregate_confidence: 0.96,
               aggregate_precision: 0.92
             )

    assert {:ok, _memory} =
             MemoryCore.build_memory_object(fact,
               summary: "The launch approval unblocked the release plan."
             )

    assert {:ok, result} =
             Evaluation.run_benchmark(
               [
                 %{
                   case_id: "launch-001",
                   conversation_id: "conv-launch",
                   question: "approved",
                   expected_answer: "launch was approved"
                 }
               ],
               workspace_id: workspace_id,
               benchmark_name: "governed_recall_smoke",
               dataset_name: "local-memory-smoke",
               retrieval_top_k: 5,
               retrieval_opts: [
                 allowed_partitions: ["ops"],
                 allowed_security_labels: ["internal"]
               ]
             )

    assert result.run.status == "completed"
    assert result.summary.case_count == 1
    assert result.summary.passed_count == 1
    assert [recorded_case] = result.cases
    assert recorded_case.status == "passed"
    assert recorded_case.context_package_id
    assert [%{type: "fact"} | _] = recorded_case.retrieved_object_links
    assert_in_delta result.summary.aggregate_scores["accuracy"].average, 1.0, 0.001

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM evaluation_cases WHERE workspace_id = ?1 AND status = 'passed'",
               [workspace_id]
             )
  end

  test "loads benchmark datasets from JSON envelopes" do
    path =
      Path.join(
        System.tmp_dir!(),
        "optimal-eval-dataset-#{System.unique_integer([:positive])}.json"
      )

    File.write!(
      path,
      Jason.encode!(%{
        benchmark_name: "workspace_memory_recall",
        dataset_name: "local-json-smoke",
        dataset_version: "2026-06-03",
        retrieval_top_k: 7,
        cases: [
          %{
            id: "json-001",
            conversation_id: "conv-json",
            query: "What was approved?",
            answer: "The launch was approved.",
            split: "smoke",
            retrieval_opts: %{subject_anchor: "launch"}
          }
        ]
      })
    )

    on_exit(fn -> File.rm(path) end)

    assert {:ok, dataset} = Evaluation.load_dataset(path)
    assert dataset.metadata["benchmark_name"] == "workspace_memory_recall"
    assert dataset.metadata["dataset_name"] == "local-json-smoke"
    assert dataset.metadata["dataset_size"] == 1
    assert [case_attrs] = dataset.cases
    assert case_attrs.case_id == "json-001"
    assert case_attrs.conversation_id == "conv-json"
    assert case_attrs.question == "What was approved?"
    assert case_attrs.expected_answer == "The launch was approved."
    assert case_attrs.retrieval_opts == %{"subject_anchor" => "launch"}
    assert case_attrs.metadata.dataset_split == "smoke"
  end

  test "runs benchmark datasets from JSONL through the evaluation pipeline" do
    workspace_id = "evaluation-jsonl-test-#{System.unique_integer([:positive])}"

    path =
      Path.join(
        System.tmp_dir!(),
        "optimal-eval-dataset-#{System.unique_integer([:positive])}.jsonl"
      )

    File.write!(path, [
      Jason.encode!(%{
        "question_id" => "jsonl-001",
        "session_id" => "session-jsonl",
        "question" => "approved",
        "expected_answer" => "launch was approved"
      }),
      "\n",
      Jason.encode!(%{
        "question_id" => "jsonl-002",
        "question" => "budget",
        "expected_answer" => "budget was approved"
      })
    ])

    on_exit(fn ->
      File.rm(path)
      Store.raw_execute("DELETE FROM evaluation_runs WHERE workspace_id = ?1", [workspace_id])
    end)

    launch_source =
      MemoryCore.source_package_from_text("The launch was approved after review.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        security_labels: ["internal"],
        partition_ids: ["ops"]
      )

    assert {:ok, launch_claim} =
             MemoryCore.extract_claim(launch_source,
               claim_text: "The launch was approved after review.",
               subject_anchor: "launch",
               action_class: "approval",
               aggregate_confidence: 0.95,
               aggregate_precision: 0.9
             )

    assert {:ok, launch_fact} =
             MemoryCore.promote_claim_to_fact(launch_claim,
               fact_text: "The launch was approved.",
               aggregate_confidence: 0.96,
               aggregate_precision: 0.92
             )

    assert {:ok, _launch_memory} =
             MemoryCore.build_memory_object(launch_fact,
               summary: "The launch approval unblocked the release plan."
             )

    budget_source =
      MemoryCore.source_package_from_text("The budget was approved by finance.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        security_labels: ["internal"],
        partition_ids: ["ops"]
      )

    assert {:ok, budget_claim} =
             MemoryCore.extract_claim(budget_source,
               claim_text: "The budget was approved by finance.",
               subject_anchor: "budget",
               action_class: "approval",
               aggregate_confidence: 0.95,
               aggregate_precision: 0.9
             )

    assert {:ok, budget_fact} =
             MemoryCore.promote_claim_to_fact(budget_claim,
               fact_text: "The budget was approved.",
               aggregate_confidence: 0.96,
               aggregate_precision: 0.92
             )

    assert {:ok, _budget_memory} =
             MemoryCore.build_memory_object(budget_fact,
               summary: "The approved budget cleared finance review."
             )

    assert {:ok, result} =
             Evaluation.run_dataset(path,
               workspace_id: workspace_id,
               benchmark_name: "jsonl_recall_smoke",
               retrieval_opts: [
                 allowed_partitions: ["ops"],
                 allowed_security_labels: ["internal"]
               ]
             )

    assert result.run.status == "completed"
    assert result.run.dataset_name =~ "optimal-eval-dataset"
    assert result.summary.case_count == 2
    assert result.summary.passed_count == 2
    assert Enum.map(result.cases, & &1.case_id) == ["jsonl-001", "jsonl-002"]
    assert_in_delta result.summary.aggregate_scores["accuracy"].average, 1.0, 0.001
  end
end
