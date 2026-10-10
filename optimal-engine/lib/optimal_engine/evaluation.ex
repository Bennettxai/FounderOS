defmodule OptimalEngine.Evaluation do
  @moduledoc """
  Durable benchmark and evaluation records.

  This module stores benchmark runs and per-case results as governed engine data.
  It can also execute a deterministic local evaluation pipeline: each case is
  retrieved through a Context Package, answered from the returned objects, judged,
  persisted, and summarized. External answerers or judges can be passed as
  callbacks without changing the storage contract.
  """

  alias OptimalEngine.MemoryCore.{ID, JSON}
  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Store

  @type run :: map()
  @type evaluation_case :: map()
  @type benchmark_case :: map() | keyword()
  @type dataset :: %{cases: [map()], metadata: map()}

  @spec run_benchmark([benchmark_case()], keyword() | map()) :: {:ok, map()} | {:error, term()}
  def run_benchmark(cases, opts \\ []) when is_list(cases) do
    opts = opts_to_keyword(opts)
    now = timestamp()
    tenant_id = string(Keyword.get(opts, :tenant_id, "default"))
    workspace_id = string(Keyword.get(opts, :workspace_id, "default"))
    cases = Enum.map(cases, &case_to_map/1)

    run_attrs =
      opts
      |> Keyword.take([
        :id,
        :benchmark_name,
        :dataset_name,
        :dataset_version,
        :answer_model,
        :judge_model,
        :judge_strategy,
        :retrieval_top_k,
        :run_config,
        :retrieval_config,
        :judge_config,
        :created_by,
        :metadata
      ])
      |> Keyword.merge(
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        benchmark_name: Keyword.get(opts, :benchmark_name, "memory_recall"),
        dataset_size: Keyword.get(opts, :dataset_size, length(cases)),
        question_count: Keyword.get(opts, :question_count, length(cases)),
        answer_model: Keyword.get(opts, :answer_model, "context-package-extractive"),
        judge_model: Keyword.get(opts, :judge_model, "deterministic-rule"),
        judge_strategy: Keyword.get(opts, :judge_strategy, "expected-answer-contains"),
        retrieval_top_k: Keyword.get(opts, :retrieval_top_k, Keyword.get(opts, :limit, 10)),
        status: "running",
        started_at: now
      )

    with {:ok, run} <- record_run(run_attrs),
         {:ok, recorded_cases} <- run_cases(run, cases, opts),
         {:ok, summary} <- summarize(run.id),
         {:ok, completed_run} <-
           complete_run(run.id,
             status: "completed",
             completed_at: timestamp(),
             aggregate_scores: summary.aggregate_scores
           ) do
      {:ok,
       %{
         run: completed_run,
         cases: recorded_cases,
         summary: Map.put(summary, :run_status, completed_run.status)
       }}
    end
  end

  @spec run_dataset(Path.t(), keyword() | map()) :: {:ok, map()} | {:error, term()}
  def run_dataset(path, opts \\ []) when is_binary(path) do
    opts = opts_to_keyword(opts)

    with {:ok, dataset} <- load_dataset(path, opts) do
      run_benchmark(dataset.cases, dataset_run_opts(dataset, opts))
    end
  end

  @spec load_dataset(Path.t(), keyword() | map()) :: {:ok, dataset()} | {:error, term()}
  def load_dataset(path, opts \\ []) when is_binary(path) do
    opts = opts_to_keyword(opts)
    format = Keyword.get(opts, :format) || infer_dataset_format(path)

    with {:ok, content} <- File.read(path),
         {:ok, raw_dataset} <- decode_dataset(content, format) do
      normalize_dataset(raw_dataset, path)
    end
  end

  @spec record_run(keyword() | map()) :: {:ok, run()} | {:error, term()}
  def record_run(attrs) when is_list(attrs), do: attrs |> Map.new() |> record_run()

  def record_run(attrs) when is_map(attrs) do
    now = timestamp()
    tenant_id = string(Map.get(attrs, :tenant_id) || Map.get(attrs, "tenant_id") || "default")

    workspace_id =
      string(Map.get(attrs, :workspace_id) || Map.get(attrs, "workspace_id") || "default")

    benchmark_name = string(Map.get(attrs, :benchmark_name) || Map.get(attrs, "benchmark_name"))

    run = %{
      id:
        string(
          Map.get(attrs, :id) || Map.get(attrs, "id") ||
            ID.content_id("evalrun", [tenant_id, ":", workspace_id, ":", benchmark_name, ":", now])
        ),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      benchmark_name: benchmark_name,
      dataset_name: string_or_nil(Map.get(attrs, :dataset_name) || Map.get(attrs, "dataset_name")),
      dataset_version:
        string_or_nil(Map.get(attrs, :dataset_version) || Map.get(attrs, "dataset_version")),
      dataset_size: int_or_nil(Map.get(attrs, :dataset_size) || Map.get(attrs, "dataset_size")),
      question_count:
        int_or_nil(Map.get(attrs, :question_count) || Map.get(attrs, "question_count")),
      answer_model: string_or_nil(Map.get(attrs, :answer_model) || Map.get(attrs, "answer_model")),
      judge_model: string_or_nil(Map.get(attrs, :judge_model) || Map.get(attrs, "judge_model")),
      judge_strategy:
        string_or_nil(Map.get(attrs, :judge_strategy) || Map.get(attrs, "judge_strategy")),
      retrieval_top_k:
        int_or_nil(Map.get(attrs, :retrieval_top_k) || Map.get(attrs, "retrieval_top_k")),
      run_config: Map.get(attrs, :run_config) || Map.get(attrs, "run_config") || %{},
      retrieval_config:
        Map.get(attrs, :retrieval_config) || Map.get(attrs, "retrieval_config") || %{},
      judge_config: Map.get(attrs, :judge_config) || Map.get(attrs, "judge_config") || %{},
      aggregate_scores:
        Map.get(attrs, :aggregate_scores) || Map.get(attrs, "aggregate_scores") || %{},
      status: string(Map.get(attrs, :status) || Map.get(attrs, "status") || "recorded"),
      started_at: string_or_nil(Map.get(attrs, :started_at) || Map.get(attrs, "started_at")),
      completed_at: string_or_nil(Map.get(attrs, :completed_at) || Map.get(attrs, "completed_at")),
      created_by: string_or_nil(Map.get(attrs, :created_by) || Map.get(attrs, "created_by")),
      metadata: Map.get(attrs, :metadata) || Map.get(attrs, "metadata") || %{}
    }

    sql = """
    INSERT INTO evaluation_runs (
      id, tenant_id, workspace_id, benchmark_name, dataset_name, dataset_version,
      dataset_size, question_count, answer_model, judge_model, judge_strategy,
      retrieval_top_k, run_config, retrieval_config, judge_config, aggregate_scores,
      status, started_at, completed_at, created_by, metadata
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11,
      ?12, ?13, ?14, ?15, ?16,
      ?17, ?18, ?19, ?20, ?21
    )
    """

    params = [
      run.id,
      run.tenant_id,
      run.workspace_id,
      run.benchmark_name,
      run.dataset_name,
      run.dataset_version,
      run.dataset_size,
      run.question_count,
      run.answer_model,
      run.judge_model,
      run.judge_strategy,
      run.retrieval_top_k,
      JSON.map(run.run_config),
      JSON.map(run.retrieval_config),
      JSON.map(run.judge_config),
      JSON.map(run.aggregate_scores),
      run.status,
      run.started_at,
      run.completed_at,
      run.created_by,
      JSON.map(run.metadata)
    ]

    with :ok <- Store.raw_execute(sql, params) do
      {:ok, run}
    end
  end

  @spec complete_run(String.t(), keyword() | map()) :: {:ok, run()} | {:error, term()}
  def complete_run(run_id, attrs) when is_binary(run_id) and is_list(attrs),
    do: complete_run(run_id, Map.new(attrs))

  def complete_run(run_id, attrs) when is_binary(run_id) and is_map(attrs) do
    with {:ok, _run} <- get_run(run_id) do
      status = string(Map.get(attrs, :status) || Map.get(attrs, "status") || "completed")

      completed_at =
        string_or_nil(Map.get(attrs, :completed_at) || Map.get(attrs, "completed_at")) ||
          timestamp()

      aggregate_scores =
        Map.get(attrs, :aggregate_scores) || Map.get(attrs, "aggregate_scores") || %{}

      sql = """
      UPDATE evaluation_runs
      SET status = ?1,
          completed_at = ?2,
          aggregate_scores = ?3
      WHERE id = ?4
      """

      with :ok <- Store.raw_execute(sql, [status, completed_at, JSON.map(aggregate_scores), run_id]) do
        get_run(run_id)
      end
    end
  end

  @spec record_case(String.t(), keyword() | map()) :: {:ok, evaluation_case()} | {:error, term()}
  def record_case(run_id, attrs) when is_binary(run_id) and is_list(attrs),
    do: record_case(run_id, Map.new(attrs))

  def record_case(run_id, attrs) when is_binary(run_id) and is_map(attrs) do
    with {:ok, run} <- get_run(run_id) do
      now = timestamp()
      case_id = string(Map.get(attrs, :case_id) || Map.get(attrs, "case_id"))

      evaluation_case = %{
        id:
          string(
            Map.get(attrs, :id) || Map.get(attrs, "id") ||
              ID.content_id("evalcase", [run_id, ":", case_id, ":", now])
          ),
        tenant_id: run.tenant_id,
        workspace_id: run.workspace_id,
        evaluation_run_id: run_id,
        case_id: case_id,
        conversation_id:
          string_or_nil(Map.get(attrs, :conversation_id) || Map.get(attrs, "conversation_id")),
        question: string(Map.get(attrs, :question) || Map.get(attrs, "question")),
        expected_answer:
          string_or_nil(Map.get(attrs, :expected_answer) || Map.get(attrs, "expected_answer")),
        actual_answer:
          string_or_nil(Map.get(attrs, :actual_answer) || Map.get(attrs, "actual_answer")),
        context_package_id:
          string_or_nil(Map.get(attrs, :context_package_id) || Map.get(attrs, "context_package_id")),
        retrieved_object_links:
          Map.get(attrs, :retrieved_object_links) || Map.get(attrs, "retrieved_object_links") || [],
        scores: Map.get(attrs, :scores) || Map.get(attrs, "scores") || %{},
        judge_output: Map.get(attrs, :judge_output) || Map.get(attrs, "judge_output") || %{},
        status: string(Map.get(attrs, :status) || Map.get(attrs, "status") || "recorded"),
        error_reason:
          string_or_nil(Map.get(attrs, :error_reason) || Map.get(attrs, "error_reason")),
        metadata: Map.get(attrs, :metadata) || Map.get(attrs, "metadata") || %{}
      }

      sql = """
      INSERT INTO evaluation_cases (
        id, tenant_id, workspace_id, evaluation_run_id, case_id, conversation_id,
        question, expected_answer, actual_answer, context_package_id,
        retrieved_object_links, scores, judge_output, status, error_reason, metadata
      ) VALUES (
        ?1, ?2, ?3, ?4, ?5, ?6,
        ?7, ?8, ?9, ?10,
        ?11, ?12, ?13, ?14, ?15, ?16
      )
      """

      params = [
        evaluation_case.id,
        evaluation_case.tenant_id,
        evaluation_case.workspace_id,
        evaluation_case.evaluation_run_id,
        evaluation_case.case_id,
        evaluation_case.conversation_id,
        evaluation_case.question,
        evaluation_case.expected_answer,
        evaluation_case.actual_answer,
        evaluation_case.context_package_id,
        JSON.list(evaluation_case.retrieved_object_links),
        JSON.map(evaluation_case.scores),
        JSON.map(evaluation_case.judge_output),
        evaluation_case.status,
        evaluation_case.error_reason,
        JSON.map(evaluation_case.metadata)
      ]

      with :ok <- Store.raw_execute(sql, params) do
        {:ok, evaluation_case}
      end
    end
  end

  @spec get_run(String.t()) :: {:ok, run()} | {:error, :not_found | term()}
  def get_run(run_id) when is_binary(run_id) do
    sql = """
    SELECT id, tenant_id, workspace_id, benchmark_name, dataset_name, dataset_version,
           dataset_size, question_count, answer_model, judge_model, judge_strategy,
           retrieval_top_k, run_config, retrieval_config, judge_config, aggregate_scores,
           status, started_at, completed_at, created_by, metadata
    FROM evaluation_runs
    WHERE id = ?1
    """

    case Store.raw_query(sql, [run_id]) do
      {:ok, [row]} -> {:ok, run_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @spec summarize(String.t()) :: {:ok, map()} | {:error, term()}
  def summarize(run_id) when is_binary(run_id) do
    with {:ok, run} <- get_run(run_id),
         {:ok, rows} <-
           Store.raw_query(
             "SELECT status, scores FROM evaluation_cases WHERE evaluation_run_id = ?1",
             [run_id]
           ) do
      scores = Enum.map(rows, fn [_status, scores] -> decode_map(scores) end)

      {:ok,
       %{
         run_id: run.id,
         benchmark_name: run.benchmark_name,
         dataset_name: run.dataset_name,
         case_count: length(rows),
         passed_count: Enum.count(rows, fn [status, _scores] -> status == "passed" end),
         failed_count: Enum.count(rows, fn [status, _scores] -> status == "failed" end),
         aggregate_scores: aggregate_scores(scores)
       }}
    end
  end

  defp run_from_row([
         id,
         tenant_id,
         workspace_id,
         benchmark_name,
         dataset_name,
         dataset_version,
         dataset_size,
         question_count,
         answer_model,
         judge_model,
         judge_strategy,
         retrieval_top_k,
         run_config,
         retrieval_config,
         judge_config,
         aggregate_scores,
         status,
         started_at,
         completed_at,
         created_by,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      benchmark_name: benchmark_name,
      dataset_name: dataset_name,
      dataset_version: dataset_version,
      dataset_size: dataset_size,
      question_count: question_count,
      answer_model: answer_model,
      judge_model: judge_model,
      judge_strategy: judge_strategy,
      retrieval_top_k: retrieval_top_k,
      run_config: decode_map(run_config),
      retrieval_config: decode_map(retrieval_config),
      judge_config: decode_map(judge_config),
      aggregate_scores: decode_map(aggregate_scores),
      status: status,
      started_at: started_at,
      completed_at: completed_at,
      created_by: created_by,
      metadata: decode_map(metadata)
    }
  end

  defp aggregate_scores([]), do: %{}

  defp aggregate_scores(scores) do
    scores
    |> Enum.flat_map(&Map.keys/1)
    |> Enum.uniq()
    |> Enum.reduce(%{}, fn key, acc ->
      values =
        scores
        |> Enum.map(&Map.get(&1, key))
        |> Enum.filter(&is_number/1)

      if values == [] do
        acc
      else
        Map.put(acc, key, %{
          count: length(values),
          average: Enum.sum(values) / length(values),
          min: Enum.min(values),
          max: Enum.max(values)
        })
      end
    end)
  end

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end

  defp decode_map(value) when is_map(value), do: value
  defp decode_map(_), do: %{}

  defp infer_dataset_format(path) do
    case path |> Path.extname() |> String.downcase() do
      ".jsonl" -> :jsonl
      ".ndjson" -> :jsonl
      _ -> :json
    end
  end

  defp decode_dataset(content, :json) do
    case Jason.decode(content) do
      {:ok, decoded} -> {:ok, decoded}
      {:error, reason} -> {:error, {:invalid_dataset_json, reason}}
    end
  end

  defp decode_dataset(content, :jsonl) do
    content
    |> String.split(~r/\R/, trim: true)
    |> Enum.with_index(1)
    |> Enum.reject(fn {line, _line_number} ->
      trimmed = String.trim(line)
      trimmed == "" or String.starts_with?(trimmed, "#")
    end)
    |> Enum.reduce_while({:ok, []}, fn {line, line_number}, {:ok, acc} ->
      case Jason.decode(line) do
        {:ok, decoded} when is_map(decoded) ->
          {:cont, {:ok, acc ++ [decoded]}}

        {:ok, decoded} ->
          {:halt, {:error, {:invalid_jsonl_case, line_number, decoded}}}

        {:error, reason} ->
          {:halt, {:error, {:invalid_jsonl, line_number, reason}}}
      end
    end)
  end

  defp decode_dataset(_content, format), do: {:error, {:unsupported_dataset_format, format}}

  defp normalize_dataset(raw_dataset, path) when is_list(raw_dataset) do
    normalize_case_list(raw_dataset, %{}, path)
  end

  defp normalize_dataset(raw_dataset, path) when is_map(raw_dataset) do
    cases =
      map_get(raw_dataset, :cases) ||
        map_get(raw_dataset, :questions) ||
        map_get(raw_dataset, :examples) ||
        map_get(raw_dataset, :items)

    if is_list(cases) do
      metadata =
        raw_dataset
        |> drop_keys(["cases", "questions", "examples", "items"])
        |> Map.put_new("dataset_path", path)

      normalize_case_list(cases, metadata, path)
    else
      normalize_case_list([raw_dataset], %{"dataset_path" => path}, path)
    end
  end

  defp normalize_dataset(raw_dataset, _path),
    do: {:error, {:invalid_dataset_shape, raw_dataset}}

  defp normalize_case_list(cases, metadata, path) do
    normalized =
      cases
      |> Enum.with_index(1)
      |> Enum.map(fn {case_attrs, index} -> normalize_dataset_case(case_attrs, index) end)

    metadata =
      metadata
      |> Map.put_new("dataset_path", path)
      |> Map.put("dataset_size", length(normalized))
      |> Map.put("question_count", length(normalized))

    {:ok, %{cases: normalized, metadata: metadata}}
  end

  defp normalize_dataset_case(case_attrs, index) when is_map(case_attrs) do
    question = first_present(case_attrs, [:question, :query, :input, :prompt])

    expected_answer =
      first_present(case_attrs, [
        :expected_answer,
        :answer,
        :target,
        :reference,
        :gold,
        :ground_truth
      ])

    %{
      case_id:
        string(
          first_present(case_attrs, [:case_id, :id, :question_id, :qid]) ||
            "case-#{index}"
        ),
      conversation_id:
        string_or_nil(
          first_present(case_attrs, [
            :conversation_id,
            :conversation,
            :conv_id,
            :session_id,
            :dialogue_id
          ])
        ),
      question: string(question),
      expected_answer: string_or_nil(expected_answer),
      retrieval_opts: map_get(case_attrs, :retrieval_opts) || %{},
      metadata:
        %{
          source_case: case_attrs,
          source_case_index: index
        }
        |> put_if_present(:dataset_split, first_present(case_attrs, [:split, :subset]))
        |> put_if_present(
          :question_type,
          first_present(case_attrs, [:question_type, :type, :category])
        )
    }
  end

  defp normalize_dataset_case(case_attrs, index) do
    %{
      case_id: "case-#{index}",
      question: string(case_attrs),
      expected_answer: nil,
      retrieval_opts: %{},
      metadata: %{source_case: case_attrs, source_case_index: index}
    }
  end

  defp dataset_run_opts(dataset, opts) do
    metadata = dataset.metadata

    defaults =
      [
        benchmark_name: map_get(metadata, :benchmark_name) || "memory_recall",
        dataset_name:
          map_get(metadata, :dataset_name) ||
            dataset_name_from_path(map_get(metadata, :dataset_path)),
        dataset_version: map_get(metadata, :dataset_version),
        dataset_size: map_get(metadata, :dataset_size),
        question_count: map_get(metadata, :question_count),
        answer_model: map_get(metadata, :answer_model),
        judge_model: map_get(metadata, :judge_model),
        judge_strategy: map_get(metadata, :judge_strategy),
        retrieval_top_k: map_get(metadata, :retrieval_top_k),
        run_config:
          Map.merge(map_get(metadata, :run_config) || %{}, %{
            "dataset_path" => map_get(metadata, :dataset_path)
          }),
        retrieval_config: map_get(metadata, :retrieval_config) || %{},
        judge_config: map_get(metadata, :judge_config) || %{},
        metadata: Map.drop(metadata, ["run_config", "retrieval_config", "judge_config"])
      ]
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    Keyword.merge(defaults, opts)
  end

  defp dataset_name_from_path(nil), do: nil

  defp dataset_name_from_path(path) do
    path
    |> Path.basename()
    |> Path.rootname()
  end

  defp drop_keys(map, keys) do
    Enum.reduce(keys, map, fn key, acc -> Map.delete(acc, key) end)
  end

  defp first_present(map, keys) do
    Enum.find_value(keys, &map_get(map, &1))
  end

  defp put_if_present(map, _key, nil), do: map
  defp put_if_present(map, key, value), do: Map.put(map, key, value)

  defp run_cases(run, cases, opts) do
    cases
    |> Enum.with_index(1)
    |> Enum.reduce_while({:ok, []}, fn {case_attrs, index}, {:ok, acc} ->
      case run_case(run, case_attrs, index, opts) do
        {:ok, evaluation_case} -> {:cont, {:ok, acc ++ [evaluation_case]}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
  end

  defp run_case(run, case_attrs, index, opts) do
    case_id = string(map_get(case_attrs, :case_id) || "case-#{index}")
    question = string(map_get(case_attrs, :question))
    retrieval_opts = retrieval_opts(run, case_attrs, opts)
    retriever = Keyword.get(opts, :retriever, &MemoryCore.retrieve/2)

    case retriever.(question, retrieval_opts) do
      {:ok, context_package} ->
        answerer = Keyword.get(opts, :answerer, &default_answer/2)
        judge = Keyword.get(opts, :judge, &default_judge/3)
        actual_answer = string(answerer.(case_attrs, context_package))
        judge_result = normalize_judge_result(judge.(case_attrs, actual_answer, context_package))

        record_case(run.id,
          case_id: case_id,
          conversation_id: string_or_nil(map_get(case_attrs, :conversation_id)),
          question: question,
          expected_answer: string_or_nil(map_get(case_attrs, :expected_answer)),
          actual_answer: actual_answer,
          context_package_id: context_package.id,
          retrieved_object_links: Map.get(context_package, :returned_object_links, []),
          scores: Map.get(judge_result, :scores, %{}),
          judge_output: Map.get(judge_result, :judge_output, %{}),
          status: Map.get(judge_result, :status, "recorded"),
          error_reason: Map.get(judge_result, :error_reason),
          metadata:
            Map.merge(map_get(case_attrs, :metadata) || %{}, %{
              retrieval_plan: Map.get(context_package, :retrieval_plan, %{}),
              filtered_object_summary: Map.get(context_package, :filtered_object_summary, %{})
            })
        )

      {:error, reason} ->
        record_case(run.id,
          case_id: case_id,
          conversation_id: string_or_nil(map_get(case_attrs, :conversation_id)),
          question: question,
          expected_answer: string_or_nil(map_get(case_attrs, :expected_answer)),
          actual_answer: nil,
          scores: %{accuracy: 0.0, retrieval_success: 0.0},
          judge_output: %{retrieval_error: inspect(reason)},
          status: "failed",
          error_reason: inspect(reason),
          metadata: map_get(case_attrs, :metadata) || %{}
        )
    end
  end

  defp retrieval_opts(run, case_attrs, opts) do
    base =
      opts
      |> Keyword.get(:retrieval_opts, [])
      |> opts_to_keyword()

    case_specific =
      case_attrs
      |> map_get(:retrieval_opts)
      |> opts_to_keyword()

    Keyword.merge(
      [
        tenant_id: run.tenant_id,
        workspace_id: run.workspace_id,
        actor_id: Keyword.get(opts, :actor_id) || Keyword.get(opts, :created_by),
        limit: Keyword.get(opts, :retrieval_top_k, Keyword.get(opts, :limit, 10))
      ],
      Keyword.merge(base, case_specific)
    )
  end

  defp default_answer(_case_attrs, context_package) do
    context_package
    |> answer_parts()
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.join("\n")
  end

  defp answer_parts(context_package) do
    fact_parts = Enum.map(Map.get(context_package, :facts, []), &text_field(&1, :fact_text))

    memory_parts =
      Enum.map(Map.get(context_package, :memory_objects, []), &text_field(&1, :summary))

    asset_parts =
      Enum.map(Map.get(context_package, :asset_extractions, []), &text_field(&1, :content_text))

    fact_parts ++ memory_parts ++ asset_parts
  end

  defp text_field(map, preferred_key) do
    Map.get(map, preferred_key) || Map.get(map, :text) || Map.get(map, "text")
  end

  defp default_judge(case_attrs, actual_answer, context_package) do
    expected_answer = string_or_nil(map_get(case_attrs, :expected_answer))
    retrieved_count = length(Map.get(context_package, :returned_object_links, []))
    retrieval_success = if retrieved_count > 0, do: 1.0, else: 0.0

    cond do
      expected_answer == nil ->
        %{
          status: "recorded",
          scores: %{answered: answered_score(actual_answer), retrieval_success: retrieval_success},
          judge_output: %{strategy: "no_expected_answer", retrieved_count: retrieved_count}
        }

      contains_normalized?(actual_answer, expected_answer) ->
        %{
          status: "passed",
          scores: %{accuracy: 1.0, grounding: retrieval_success},
          judge_output: %{
            strategy: "expected-answer-contains",
            expected_answer: expected_answer,
            retrieved_count: retrieved_count
          }
        }

      true ->
        %{
          status: "failed",
          scores: %{accuracy: 0.0, grounding: retrieval_success},
          judge_output: %{
            strategy: "expected-answer-contains",
            expected_answer: expected_answer,
            retrieved_count: retrieved_count
          },
          error_reason: "expected_answer_not_found"
        }
    end
  end

  defp normalize_judge_result(result) when is_map(result) do
    %{
      status: string(Map.get(result, :status) || Map.get(result, "status") || "recorded"),
      scores: Map.get(result, :scores) || Map.get(result, "scores") || %{},
      judge_output: Map.get(result, :judge_output) || Map.get(result, "judge_output") || %{},
      error_reason: string_or_nil(Map.get(result, :error_reason) || Map.get(result, "error_reason"))
    }
  end

  defp normalize_judge_result(:passed), do: %{status: "passed", scores: %{accuracy: 1.0}}
  defp normalize_judge_result(:failed), do: %{status: "failed", scores: %{accuracy: 0.0}}

  defp normalize_judge_result(other) do
    %{status: "recorded", judge_output: %{result: inspect(other)}}
  end

  defp answered_score(""), do: 0.0
  defp answered_score(_), do: 1.0

  defp contains_normalized?(actual, expected) do
    actual |> normalize_text() |> String.contains?(normalize_text(expected))
  end

  defp normalize_text(value) do
    value
    |> string()
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9]+/, " ")
    |> String.trim()
  end

  defp case_to_map(value) when is_list(value), do: Map.new(value)
  defp case_to_map(value) when is_map(value), do: value

  defp opts_to_keyword(nil), do: []
  defp opts_to_keyword(value) when is_list(value), do: value

  defp opts_to_keyword(value) when is_map(value) do
    Enum.map(value, fn {key, option_value} -> {option_key(key), option_value} end)
  end

  defp opts_to_keyword(_), do: []

  defp option_key(key) when is_atom(key), do: key
  defp option_key(key) when is_binary(key), do: String.to_atom(key)
  defp option_key(key), do: key

  defp map_get(map, key) when is_map(map) do
    Map.get(map, key) || Map.get(map, to_string(key))
  end

  defp map_get(_, _), do: nil

  defp int_or_nil(nil), do: nil
  defp int_or_nil(value) when is_integer(value), do: value

  defp int_or_nil(value) when is_binary(value) do
    case Integer.parse(value) do
      {int, ""} -> int
      _ -> nil
    end
  end

  defp int_or_nil(_), do: nil

  defp string(nil), do: ""
  defp string(value) when is_binary(value), do: value
  defp string(value), do: to_string(value)

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: string(value)

  defp timestamp, do: DateTime.utc_now() |> DateTime.truncate(:second) |> DateTime.to_iso8601()
end
