defmodule OptimalEngine.Signal.Classifier.AnalyzerTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier.Analyzer

  # --- Helpers ---

  defp signal(type, overrides \\ []) do
    defaults = [source: "/test"]
    Signal.new!(type, Keyword.merge(defaults, overrides))
  end

  # --- infer_mode/1 ---

  describe "infer_mode/1" do
    test "returns :code for map data" do
      s = signal("miosa.test", data: %{key: "val"})
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :code for list data" do
      s = signal("miosa.test", data: [1, 2, 3])
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :linguistic for plain text data" do
      s = signal("miosa.test", data: "hello world")
      assert Analyzer.infer_mode(s) == :linguistic
    end

    test "returns :code for code-like text with def" do
      s = signal("miosa.test", data: "def foo(x), do: x + 1")
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :code for text with pipeline operator" do
      s = signal("miosa.test", data: "data |> Enum.map(&(&1 + 1))")
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :code for text with defmodule" do
      s = signal("miosa.test", data: "defmodule Foo do\nend")
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :code for JavaScript-like text with function keyword" do
      s = signal("miosa.test", data: "function greet(name) { return name; }")
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :code for const declaration" do
      s = signal("miosa.test", data: "const x = 42;")
      assert Analyzer.infer_mode(s) == :code
    end

    test "returns :linguistic for nil data" do
      s = signal("miosa.test")
      assert Analyzer.infer_mode(s) == :linguistic
    end

    test "returns :linguistic for empty string data" do
      s = signal("miosa.test", data: "")
      assert Analyzer.infer_mode(s) == :linguistic
    end
  end

  # --- infer_genre/1 ---

  describe "infer_genre/1" do
    test "infers :error for error segment in type" do
      s = signal("miosa.agent.error.occurred")
      assert Analyzer.infer_genre(s) == :error
    end

    test "infers :alert for alert segment" do
      s = signal("miosa.system.alert.fired")
      assert Analyzer.infer_genre(s) == :alert
    end

    test "infers :progress for progress segment" do
      s = signal("miosa.agent.progress.update")
      assert Analyzer.infer_genre(s) == :progress
    end

    test "infers :brief for task segment" do
      s = signal("miosa.agent.task.completed")
      assert Analyzer.infer_genre(s) == :brief
    end

    test "infers :spec for spec segment" do
      s = signal("miosa.spec.defined")
      assert Analyzer.infer_genre(s) == :spec
    end

    test "infers :report for report segment" do
      s = signal("miosa.report.generated")
      assert Analyzer.infer_genre(s) == :report
    end

    test "infers :pr for review segment" do
      s = signal("miosa.code.review.submitted")
      assert Analyzer.infer_genre(s) == :pr
    end

    test "infers :chat for chat segment" do
      s = signal("miosa.chat.message.sent")
      assert Analyzer.infer_genre(s) == :chat
    end

    test "infers :adr for decision segment" do
      s = signal("miosa.decision.made")
      assert Analyzer.infer_genre(s) == :adr
    end

    test "defaults to :chat when no recognized segment" do
      s = signal("miosa.unknown.something")
      assert Analyzer.infer_genre(s) == :chat
    end

    test "defaults to :chat for single segment type" do
      s = signal("miosa")
      assert Analyzer.infer_genre(s) == :chat
    end
  end

  # --- infer_type/1 ---

  describe "infer_type/1" do
    test "infers :inform for completed" do
      s = signal("miosa.task.completed")
      assert Analyzer.infer_type(s) == :inform
    end

    test "infers :inform for created" do
      s = signal("miosa.user.created")
      assert Analyzer.infer_type(s) == :inform
    end

    test "infers :inform for started" do
      s = signal("miosa.job.started")
      assert Analyzer.infer_type(s) == :inform
    end

    test "infers :inform for failed" do
      s = signal("miosa.agent.failed")
      assert Analyzer.infer_type(s) == :inform
    end

    test "infers :direct for request" do
      s = signal("miosa.agent.request")
      assert Analyzer.infer_type(s) == :direct
    end

    test "infers :direct for command" do
      s = signal("miosa.system.command")
      assert Analyzer.infer_type(s) == :direct
    end

    test "infers :direct for dispatch" do
      s = signal("miosa.agent.dispatch")
      assert Analyzer.infer_type(s) == :direct
    end

    test "infers :decide for decided" do
      s = signal("miosa.team.decided")
      assert Analyzer.infer_type(s) == :decide
    end

    test "infers :commit for approved" do
      s = signal("miosa.pr.approved")
      assert Analyzer.infer_type(s) == :commit
    end

    test "infers :commit for committed" do
      s = signal("miosa.code.committed")
      assert Analyzer.infer_type(s) == :commit
    end

    test "infers :commit for merged" do
      s = signal("miosa.branch.merged")
      assert Analyzer.infer_type(s) == :commit
    end

    test "infers :express for expressed" do
      s = signal("miosa.agent.expressed")
      assert Analyzer.infer_type(s) == :express
    end

    test "infers :express for acknowledged" do
      s = signal("miosa.feedback.acknowledged")
      assert Analyzer.infer_type(s) == :express
    end

    test "defaults to :inform for unknown last segment" do
      s = signal("miosa.unknown.xyz")
      assert Analyzer.infer_type(s) == :inform
    end
  end

  # --- infer_format/1 ---

  describe "infer_format/1" do
    test "returns :json for map data" do
      s = signal("miosa.test", data: %{a: 1})
      assert Analyzer.infer_format(s) == :json
    end

    test "returns :json for list data" do
      s = signal("miosa.test", data: [1, 2, 3])
      assert Analyzer.infer_format(s) == :json
    end

    test "returns :code for code-like binary data" do
      s = signal("miosa.test", data: "def foo, do: :ok")
      assert Analyzer.infer_format(s) == :code
    end

    test "returns :markdown for markdown indicators" do
      s = signal("miosa.test", data: "# Title\n- item 1\n**bold**")
      assert Analyzer.infer_format(s) == :markdown
    end

    test "returns :cli for plain text without code or markdown" do
      s = signal("miosa.test", data: "Server started successfully")
      assert Analyzer.infer_format(s) == :cli
    end

    test "returns :json for nil data" do
      s = signal("miosa.test")
      assert Analyzer.infer_format(s) == :json
    end
  end

  # --- infer_structure/1 ---

  describe "infer_structure/1" do
    test "returns :error_report for error type segment" do
      s = signal("miosa.agent.error.occurred")
      assert Analyzer.infer_structure(s) == :error_report
    end

    test "returns :task_brief for task segment" do
      s = signal("miosa.agent.task.completed")
      assert Analyzer.infer_structure(s) == :task_brief
    end

    test "returns :alert_template for alert segment" do
      s = signal("miosa.system.alert.fired")
      assert Analyzer.infer_structure(s) == :alert_template
    end

    test "returns :progress_update for progress segment" do
      s = signal("miosa.agent.progress.update")
      assert Analyzer.infer_structure(s) == :progress_update
    end

    test "returns :specification for spec segment" do
      s = signal("miosa.spec.created")
      assert Analyzer.infer_structure(s) == :specification
    end

    test "returns :decision_record for decision segment" do
      s = signal("miosa.decision.made")
      assert Analyzer.infer_structure(s) == :decision_record
    end

    test "returns :default for unrecognized type" do
      s = signal("miosa.unknown.event")
      assert Analyzer.infer_structure(s) == :default
    end
  end

  # --- dimension_score/1 ---

  describe "dimension_score/1" do
    test "returns 0.0 when no dimensions are set" do
      s = signal("miosa.test")
      assert Analyzer.dimension_score(s) == 0.0
    end

    test "returns 1.0 when all five dimensions are set" do
      s =
        signal("miosa.test",
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform,
          signal_format: :json,
          signal_structure: :task_brief
        )

      assert Analyzer.dimension_score(s) == 1.0
    end

    test "returns 0.2 when one dimension is set" do
      s = signal("miosa.test", signal_mode: :code)
      assert Analyzer.dimension_score(s) == 0.2
    end

    test "returns 0.6 when three dimensions are set" do
      s =
        signal("miosa.test",
          signal_mode: :code,
          signal_genre: :brief,
          signal_type: :inform
        )

      assert Analyzer.dimension_score(s) == 0.6
    end
  end

  # --- data_score/1 ---

  describe "data_score/1" do
    test "returns 0.2 for nil data" do
      s = signal("miosa.test")
      assert Analyzer.data_score(s) == 0.2
    end

    test "returns 1.0 for non-empty map data" do
      s = signal("miosa.test", data: %{key: "value"})
      assert Analyzer.data_score(s) == 1.0
    end

    test "returns 0.3 for empty map data" do
      s = signal("miosa.test", data: %{})
      assert Analyzer.data_score(s) == 0.3
    end

    test "returns 0.9 for non-empty list data" do
      s = signal("miosa.test", data: [1, 2, 3])
      assert Analyzer.data_score(s) == 0.9
    end

    test "returns 0.3 for empty list data" do
      s = signal("miosa.test", data: [])
      assert Analyzer.data_score(s) == 0.3
    end

    test "returns 0.2 for empty string data" do
      s = signal("miosa.test", data: "")
      assert Analyzer.data_score(s) == 0.2
    end

    test "returns 0.5 for short string data (< 10 bytes)" do
      s = signal("miosa.test", data: "hello")
      assert Analyzer.data_score(s) == 0.5
    end

    test "returns 0.8 for medium string data (10-999 bytes)" do
      s = signal("miosa.test", data: "this is a medium-length string content")
      assert Analyzer.data_score(s) == 0.8
    end

    test "returns 0.7 for large string data (>= 1000 bytes)" do
      s = signal("miosa.test", data: String.duplicate("x", 1000))
      assert Analyzer.data_score(s) == 0.7
    end
  end

  # --- type_score/1 ---

  describe "type_score/1" do
    test "returns 1.0 for three or more segments" do
      s = signal("miosa.agent.task.completed")
      assert Analyzer.type_score(s) == 1.0
    end

    test "returns 0.7 for exactly two segments" do
      s = signal("miosa.task")
      assert Analyzer.type_score(s) == 0.7
    end

    test "returns 0.4 for single segment" do
      s = signal("miosa")
      assert Analyzer.type_score(s) == 0.4
    end
  end

  # --- context_score/1 ---

  describe "context_score/1" do
    test "returns 0.0 when no context fields are set" do
      s = Signal.new!("miosa.test")
      # source defaults to nil
      assert Analyzer.context_score(s) == 0.0
    end

    test "returns 1.0 when all three context fields are set" do
      s =
        signal("miosa.test",
          agent_id: "agent-1",
          session_id: "sess-1"
        )

      # source is set via signal helper as "/test"
      assert Analyzer.context_score(s) == 1.0
    end

    test "returns 0.333... when only source is set" do
      s = signal("miosa.test")
      # agent_id and session_id are nil, source is "/test"
      score = Analyzer.context_score(s)
      assert_in_delta score, 1 / 3.0, 0.01
    end

    test "returns 0.667 when two context fields are set" do
      s = signal("miosa.test", agent_id: "agent-1")
      score = Analyzer.context_score(s)
      assert_in_delta score, 2 / 3.0, 0.01
    end
  end

  # --- code_like?/1 ---

  describe "code_like?/1" do
    test "returns true for def keyword" do
      assert Analyzer.code_like?("def my_func(x), do: x")
    end

    test "returns true for fn keyword" do
      assert Analyzer.code_like?("fn x -> x + 1 end")
    end

    test "returns true for pipeline operator" do
      assert Analyzer.code_like?("data |> transform()")
    end

    test "returns true for arrow operator" do
      assert Analyzer.code_like?("%{key => value}")
    end

    test "returns true for defmodule" do
      assert Analyzer.code_like?("defmodule MyApp do")
    end

    test "returns true for import" do
      assert Analyzer.code_like?("import Ecto.Query")
    end

    test "returns true for require" do
      assert Analyzer.code_like?("require Logger")
    end

    test "returns true for function keyword" do
      assert Analyzer.code_like?("function greet(name) {}")
    end

    test "returns true for const keyword" do
      assert Analyzer.code_like?("const PI = 3.14;")
    end

    test "returns true for let keyword" do
      assert Analyzer.code_like?("let count = 0;")
    end

    test "returns true for var keyword" do
      assert Analyzer.code_like?("var x = 10;")
    end

    test "returns true for class keyword" do
      assert Analyzer.code_like?("class Animal {}")
    end

    test "returns true for curly brace" do
      assert Analyzer.code_like?("{key: value}")
    end

    test "returns false for plain English text" do
      refute Analyzer.code_like?("This is a plain English sentence about nothing.")
    end

    test "returns false for empty string" do
      refute Analyzer.code_like?("")
    end

    test "type_genre_map/0 returns the expected mapping" do
      map = Analyzer.type_genre_map()
      assert is_map(map)
      assert Map.has_key?(map, "error")
      assert Map.has_key?(map, "alert")
      assert Map.has_key?(map, "task")
      assert {genre, structure} = map["error"]
      assert genre == :error
      assert structure == :error_report
    end
  end
end
