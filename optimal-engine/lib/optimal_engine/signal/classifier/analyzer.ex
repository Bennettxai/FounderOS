defmodule OptimalEngine.Signal.Classifier.Analyzer do
  @moduledoc """
  Inference and scoring helpers for the Signal Theory classifier.

  Handles:
  - Five-dimension inference (mode, genre, type, format, structure) from signal fields
  - S/N ratio component scoring (dimension, data, type, context)
  - Code-heuristic detection
  """

  alias OptimalEngine.Signal.Envelope, as: Signal

  # ── Type → Genre / Structure mapping ──

  @type_genre_map %{
    "error" => {:error, :error_report},
    "alert" => {:alert, :alert_template},
    "progress" => {:progress, :progress_update},
    "task" => {:brief, :task_brief},
    "spec" => {:spec, :specification},
    "report" => {:report, :report_template},
    "review" => {:pr, :review_template},
    "chat" => {:chat, :conversation},
    "decision" => {:adr, :decision_record}
  }

  # ── Type segment → Signal Type mapping ──

  @type_action_map %{
    "completed" => :inform,
    "created" => :inform,
    "started" => :inform,
    "failed" => :inform,
    "request" => :direct,
    "command" => :direct,
    "dispatch" => :direct,
    "decided" => :decide,
    "approved" => :commit,
    "committed" => :commit,
    "merged" => :commit,
    "expressed" => :express,
    "acknowledged" => :express
  }

  @doc "Returns the type-genre map used for genre and structure inference."
  def type_genre_map, do: @type_genre_map

  # ── Inference ──────────────────────────────────────────────────

  @doc "Infers the signal mode from data type and content."
  @spec infer_mode(Signal.t()) :: atom()
  def infer_mode(%Signal{data: data}) when is_binary(data) do
    if code_like?(data), do: :code, else: :linguistic
  end

  def infer_mode(%Signal{data: data}) when is_map(data), do: :code
  def infer_mode(%Signal{data: data}) when is_list(data), do: :code
  def infer_mode(_), do: :linguistic

  @doc "Infers the signal genre from the type string segments."
  @spec infer_genre(Signal.t()) :: atom()
  def infer_genre(%Signal{type: type}) when is_binary(type) do
    segments = String.split(type, ".")

    genre_segment =
      Enum.find(segments, fn seg ->
        Map.has_key?(@type_genre_map, seg)
      end)

    case genre_segment do
      nil -> :chat
      seg -> elem(@type_genre_map[seg], 0)
    end
  end

  def infer_genre(_), do: :chat

  @doc "Infers the signal action type from the last type segment."
  @spec infer_type(Signal.t()) :: atom()
  def infer_type(%Signal{type: type}) when is_binary(type) do
    segments = String.split(type, ".")
    last = List.last(segments)
    Map.get(@type_action_map, last, :inform)
  end

  def infer_type(_), do: :inform

  @doc "Infers the signal format from data content."
  @spec infer_format(Signal.t()) :: atom()
  def infer_format(%Signal{data: data}) when is_map(data), do: :json
  def infer_format(%Signal{data: data}) when is_list(data), do: :json

  def infer_format(%Signal{data: data}) when is_binary(data) do
    cond do
      code_like?(data) -> :code
      String.contains?(data, ["#", "**", "- "]) -> :markdown
      true -> :cli
    end
  end

  def infer_format(_), do: :json

  @doc "Infers the signal structure atom from type segments."
  @spec infer_structure(Signal.t()) :: atom()
  def infer_structure(%Signal{type: type}) when is_binary(type) do
    segments = String.split(type, ".")

    structure_segment =
      Enum.find(segments, fn seg ->
        Map.has_key?(@type_genre_map, seg)
      end)

    case structure_segment do
      nil -> :default
      seg -> elem(@type_genre_map[seg], 1)
    end
  end

  def infer_structure(_), do: :default

  # ── Scoring ─────────────────────────────────────────────────────

  @doc "Scores how many of the five Signal Theory dimensions are resolved (0.0–1.0)."
  @spec dimension_score(Signal.t()) :: float()
  def dimension_score(signal) do
    dims = [
      signal.signal_mode,
      signal.signal_genre,
      signal.signal_type,
      signal.signal_format,
      signal.signal_structure
    ]

    resolved = Enum.count(dims, &(not is_nil(&1)))
    resolved / 5.0
  end

  @doc "Scores data quality contribution to S/N ratio (0.0–1.0)."
  @spec data_score(Signal.t()) :: float()
  def data_score(%Signal{data: nil}), do: 0.2

  def data_score(%Signal{data: data}) when is_map(data) do
    if map_size(data) > 0, do: 1.0, else: 0.3
  end

  def data_score(%Signal{data: data}) when is_list(data) do
    if length(data) > 0, do: 0.9, else: 0.3
  end

  def data_score(%Signal{data: data}) when is_binary(data) do
    size = byte_size(data)

    cond do
      size == 0 -> 0.2
      size < 10 -> 0.5
      size < 1000 -> 0.8
      true -> 0.7
    end
  end

  def data_score(_), do: 0.5

  @doc "Scores type string depth contribution to S/N ratio (0.0–1.0)."
  @spec type_score(Signal.t()) :: float()
  def type_score(%Signal{type: type}) when is_binary(type) do
    segments = String.split(type, ".")

    cond do
      length(segments) >= 3 -> 1.0
      length(segments) == 2 -> 0.7
      true -> 0.4
    end
  end

  def type_score(_), do: 0.2

  @doc "Scores agent context completeness contribution to S/N ratio (0.0–1.0)."
  @spec context_score(Signal.t()) :: float()
  def context_score(signal) do
    fields = [signal.agent_id, signal.session_id, signal.source]
    present = Enum.count(fields, &(not is_nil(&1)))
    present / 3.0
  end

  # ── Heuristics ──────────────────────────────────────────────────

  @doc "Returns true if the string looks like source code."
  @spec code_like?(String.t()) :: boolean()
  def code_like?(str) when is_binary(str) do
    code_indicators = [
      "def ",
      "fn ",
      "->",
      "|>",
      "defmodule",
      "import ",
      "require ",
      "function",
      "const ",
      "let ",
      "var ",
      "class ",
      "{",
      "}",
      "=>"
    ]

    Enum.any?(code_indicators, &String.contains?(str, &1))
  end
end
