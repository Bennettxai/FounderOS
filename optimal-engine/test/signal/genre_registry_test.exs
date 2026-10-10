defmodule OptimalEngine.Signal.GenreRegistryTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Signal.GenreRegistry, as: R
  alias OptimalEngine.Pipeline.Classifier
  alias OptimalEngine.Retrieval.Composer

  describe "registry loading" do
    test "loads the full genre vocabulary (>130 genres)" do
      assert R.count() > 130
      assert "spec" in R.genres()
      assert "decision-log" in R.genres()
      assert "brief" in R.genres()
    end

    test "every entry has category, signal_type, mode and a skeleton" do
      for g <- R.all() do
        assert is_binary(g.genre)
        assert is_binary(g.category)
        assert is_binary(g.signal_type)
        assert is_list(g.sections)
      end
    end

    test "fetch + skeleton resolve a known genre" do
      assert {:ok, %{category: "technical"}} = R.fetch("spec")
      skeleton = R.skeleton("spec")
      assert Enum.any?(skeleton, &(&1.name == "Requirements"))
    end

    test "unknown genre fetch returns :error, skeleton returns []" do
      assert R.fetch("not-a-real-genre") == :error
      assert R.skeleton("not-a-real-genre") == []
    end
  end

  describe "detect/1 keyword fallback" do
    test "maps spec-like text to spec" do
      text = "## Requirements\nThe system must X.\n## Acceptance Criteria\nPasses when Y."
      assert R.detect(text) == "spec"
    end

    test "maps decision text to a decision genre, not the default" do
      text = "We decided to adopt the new pricing. Decision: ship Friday."
      refute R.detect(text) == "note"
    end

    test "defaults to note when nothing matches" do
      assert R.detect("zzz qqq") == "note"
    end
  end

  describe "pipeline classifier uses the full vocabulary" do
    test "classifies a clearly decision-log input as decision-log (not chat/note)" do
      content = "# Q3 Pricing\n\nDecision: we decided to raise prices. Key decisions below."
      sig = Classifier.classify(content)
      assert sig.genre == "decision-log"
    end

    test "classifies a clearly spec input as spec" do
      content = "# Build X\n\n## Requirements\n1. Do A\n\n## Acceptance Criteria\nB passes."
      sig = Classifier.classify(content)
      assert sig.genre == "spec"
    end

    test "classifies a clearly brief input as brief" do
      content =
        "# Outreach\n\n## Objective\nClose deal.\n\n## Key Messages\n- a\n\n## Call to Action\nReply by Friday."

      sig = Classifier.classify(content)
      assert sig.genre == "brief"
    end

    test "falls back gracefully to note for unstructured chatter" do
      sig = Classifier.classify("hey just checking in, nothing much happening")
      assert sig.genre == "note"
    end
  end

  describe "composer renders registry skeletons" do
    test "an arbitrary registry genre renders its section skeleton" do
      signal = %OptimalEngine.Signal{
        title: "Incident 42",
        node: "ops",
        genre: "postmortem",
        sn_ratio: 0.8,
        content: "Something broke.",
        l1_description: "Outage summary."
      }

      # Drive the private reformat path through the registry by rendering for a
      # receiver whose primary genre is a registry genre.
      rendered = render(signal, "postmortem")
      assert rendered =~ "Postmortem: Incident 42"
      # postmortem skeleton sections should appear as headers
      for %{name: name} <- R.skeleton("postmortem") do
        assert rendered =~ "## #{name}"
      end
    end
  end

  # Helper that exercises Composer.render_for with a minimal topology stub.
  defp render(signal, genre) do
    topology = %{
      endpoints: %{
        "rx" => %{id: "rx", genre_competence: [genre]}
      }
    }

    {:ok, out} = Composer.render_for(signal, "rx", topology)
    out
  end
end
