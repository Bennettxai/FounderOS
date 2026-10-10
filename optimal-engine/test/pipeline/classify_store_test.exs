defmodule OptimalEngine.Pipeline.ClassifyStoreTest do
  @moduledoc """
  Round-trip: classify + intent-extract every chunk in a ChunkTree,
  persist via Store.insert_classifications + Store.insert_intents,
  verify the rows land correctly.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.{Classifier, Decomposer, IntentExtractor}
  alias OptimalEngine.Pipeline.Parser.ParsedDoc
  alias OptimalEngine.Store

  @opts [ollama_augmentation: false]

  test "classifications persist one row per chunk keyed by chunk_id" do
    unique = System.unique_integer([:positive])
    text = "Decision: use Phoenix.\n\nRequirement #{unique}: must auto-scale."
    signal_id = "sha256:clstest-#{unique}"
    doc = ParsedDoc.new(text: text, signal_id: signal_id)

    {:ok, tree} = Decomposer.decompose_and_store(doc)
    {:ok, classifications} = Classifier.classify_tree(tree, @opts)
    assert :ok = Store.insert_classifications(classifications)

    {:ok, rows} =
      Store.raw_query(
        """
        SELECT c.chunk_id, c.mode, c.genre, c.signal_type, c.format, c.structure,
               c.sn_ratio, c.confidence
        FROM classifications c
        JOIN chunks ch ON ch.id = c.chunk_id
        WHERE ch.signal_id = ?1
        """,
        [signal_id]
      )

    assert length(rows) == length(classifications)

    # Re-insert: verify idempotency (ON CONFLICT on chunk_id)
    assert :ok = Store.insert_classifications(classifications)

    {:ok, [[count]]} =
      Store.raw_query(
        """
        SELECT COUNT(*) FROM classifications
        WHERE chunk_id IN (SELECT id FROM chunks WHERE signal_id = ?1)
        """,
        [signal_id]
      )

    assert count == length(classifications)
  end

  test "intents persist one row per chunk keyed by chunk_id" do
    unique = System.unique_integer([:positive])
    text = "We should decide by Monday. I'll own the spec. Blocker: the vendor API."
    signal_id = "sha256:inttest-#{unique}"
    doc = ParsedDoc.new(text: text, signal_id: signal_id)

    {:ok, tree} = Decomposer.decompose_and_store(doc)
    {:ok, intents} = IntentExtractor.extract_tree(tree, @opts)
    assert :ok = Store.insert_intents(intents)

    {:ok, rows} =
      Store.raw_query(
        """
        SELECT i.chunk_id, i.intent, i.confidence, i.evidence
        FROM intents i
        JOIN chunks ch ON ch.id = i.chunk_id
        WHERE ch.signal_id = ?1
        """,
        [signal_id]
      )

    assert length(rows) == length(intents)

    # Every row must have a valid intent enum value.
    rows
    |> Enum.map(fn [_, intent, _, _] -> intent end)
    |> Enum.each(fn intent ->
      assert intent in ~w(request_info propose_decision record_fact express_concern
                          commit_action reference narrate reflect specify measure)
    end)
  end
end
