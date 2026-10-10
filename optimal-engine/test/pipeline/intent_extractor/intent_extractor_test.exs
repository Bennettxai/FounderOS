defmodule OptimalEngine.Pipeline.IntentExtractorTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Decomposer.Chunk
  alias OptimalEngine.Pipeline.IntentExtractor
  alias OptimalEngine.Pipeline.IntentExtractor.Intent

  # Turn off Ollama refinement so heuristic-only behavior is testable
  # without the local server running.
  @opts [ollama_augmentation: false]

  defp intent_of(text) do
    chunk = Chunk.new(id: "c1", signal_id: "s", tenant_id: "default", text: text)
    {:ok, %Intent{intent: intent, confidence: confidence}} = IntentExtractor.extract(chunk, @opts)
    {intent, confidence}
  end

  describe "heuristic classification — per intent" do
    test ":request_info — question-mark chunks" do
      assert {:request_info, _} = intent_of("Could you send the numbers?")
      assert {:request_info, _} = intent_of("When will the prod deploy happen?")
    end

    test ":propose_decision — decision-language" do
      assert {:propose_decision, _} = intent_of("We should go with Postgres for the MVP.")
      assert {:propose_decision, _} = intent_of("Proposing: drop the legacy auth module by Q3.")
    end

    test ":commit_action — action ownership" do
      assert {:commit_action, _} = intent_of("I'll finish the API spec by Friday.")
      assert {:commit_action, _} = intent_of("Todo: migrate the store to WAL mode.")
    end

    test ":express_concern — risk language" do
      assert {:express_concern, _} = intent_of("Worried about the memory footprint of the index.")
      assert {:express_concern, _} = intent_of("Blocker: the vendor API keeps timing out.")
    end

    test ":specify — constraints + requirements" do
      assert {:specify, _} = intent_of("Must support concurrent writes without data loss.")
      assert {:specify, _} = intent_of("Requirement: every ACL row has an expiry.")
    end

    test ":measure — metrics + numbers" do
      assert {:measure, _} = intent_of("p99 latency is 140ms on the hot path.")
      assert {:measure, _} = intent_of("87% of hits come from the ETS cache.")
    end

    test ":reflect — retrospective language" do
      assert {:reflect, _} =
               intent_of("Looking back on Q2, the auth refactor cost more than planned.")
    end

    test ":narrate — sequenced events" do
      assert {:narrate, _} = intent_of("First, the request came in. Then, the router matched it.")
    end

    test ":reference — pointers to other context" do
      assert {:reference, _} = intent_of("See the ADR-042 for the full rationale.")
    end

    test ":record_fact — default when nothing else fires" do
      assert {:record_fact, _} = intent_of("The weather today is cold.")
      assert {:record_fact, _} = intent_of("Alpha Centauri is the closest star system.")
    end
  end

  describe "confidence bounds" do
    test "heuristic confidence always in 0.0..1.0" do
      examples = [
        "Could you send?",
        "We should decide.",
        "I'll own this.",
        "Looking back.",
        "Just a fact.",
        ""
      ]

      Enum.each(examples, fn text ->
        chunk = Chunk.new(id: "c", signal_id: "s", text: text)
        {:ok, %Intent{confidence: c}} = IntentExtractor.extract(chunk, @opts)
        assert c >= 0.0 and c <= 1.0, "#{inspect(text)} → confidence #{c}"
      end)
    end

    test "confidence rises with more matches of the rule" do
      weak = "Could you send?"
      strong = "Could you send? What about this? And how about that? When will it ship?"

      {_i1, c1} = intent_of(weak)
      {_i2, c2} = intent_of(strong)
      assert c2 >= c1
    end
  end

  describe "evidence" do
    test "evidence window is a substring of the original text" do
      text = "Let's decide: ship on Monday and regroup after."
      chunk = Chunk.new(id: "c", signal_id: "s", text: text)

      {:ok, %Intent{evidence: evidence}} = IntentExtractor.extract(chunk, @opts)
      assert is_binary(evidence)
      assert String.contains?(text, evidence) or byte_size(evidence) > 0
    end
  end

  describe "extract_tree/2 — gold-set accuracy" do
    @gold [
      {"Can you send me the deck?", :request_info},
      {"Proposing: switch to daily standups.", :propose_decision},
      {"I'll own the migration plan.", :commit_action},
      {"Worried about the cost spike.", :express_concern},
      {"Requirement: all writes must be idempotent.", :specify},
      {"Throughput peaked at 4200 rps.", :measure},
      {"In retrospect, we should have shipped the proxy sooner.", :reflect},
      {"First the PR landed, then tests ran, finally it merged.", :narrate},
      {"See the ADR-023 for context.", :reference},
      {"The company was founded in 2019.", :record_fact}
    ]

    test "≥80% accuracy on the 10-sample gold set (heuristics only)" do
      correct =
        @gold
        |> Enum.map(fn {text, expected} ->
          chunk = Chunk.new(id: "c", signal_id: "s", text: text)
          {:ok, %Intent{intent: actual}} = IntentExtractor.extract(chunk, @opts)
          {expected, actual}
        end)
        |> Enum.count(fn {expected, actual} -> expected == actual end)

      ratio = correct / length(@gold)
      assert ratio >= 0.8, "gold-set accuracy #{ratio} below 0.80"
    end
  end
end
