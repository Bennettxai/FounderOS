defmodule OptimalEngine.Pipeline.Intake.Skeleton do
  @moduledoc """
  Genre skeleton templates for structured signal files.

  Each genre maps to an ordered list of sections with metadata about
  whether the section is required and whether the engine should attempt
  to auto-populate it from the raw content.

  Supported genres:
  - transcript      — Meeting/call records
  - brief           — Action-oriented summaries for non-technical receivers
  - spec            — Technical requirements and acceptance criteria
  - plan            — Structured execution plans
  - note            — Quick captures for later routing
  - decision-log    — Formal decision records
  - standup         — Weekly status signals
  - review          — Single/double-loop reviews
  - report          — Findings with analysis
  - pitch           — Value proposition structures
  """

  @type section :: %{
          name: String.t(),
          required: boolean(),
          auto: boolean(),
          hint: String.t()
        }

  @type t :: %{
          genre: String.t(),
          sections: [section()]
        }

  @skeletons %{
    "transcript" => [
      %{name: "Participants", required: true, auto: false, hint: "List who was present"},
      %{
        name: "Key Points",
        required: true,
        auto: false,
        hint: "Bullet summary of what was discussed"
      },
      %{name: "Decisions Made", required: false, auto: false, hint: "What was decided, by whom"},
      %{name: "Action Items", required: true, auto: false, hint: "Person + task + deadline"},
      %{name: "Open Questions", required: false, auto: false, hint: "Unresolved items"}
    ],
    "brief" => [
      %{name: "Objective", required: true, auto: false, hint: "One sentence: what outcome"},
      %{
        name: "Key Messages",
        required: true,
        auto: false,
        hint: "3-5 bullets, each an atomic claim"
      },
      %{
        name: "Call to Action",
        required: true,
        auto: false,
        hint: "Single unambiguous ask + deadline"
      },
      %{
        name: "Supporting Materials",
        required: false,
        auto: false,
        hint: "Links, attachments if needed"
      }
    ],
    "spec" => [
      %{name: "Goal", required: true, auto: false, hint: "What and why"},
      %{name: "Requirements", required: true, auto: false, hint: "Numbered list"},
      %{name: "Constraints", required: true, auto: false, hint: "What is off the table"},
      %{name: "Architecture", required: false, auto: false, hint: "How it fits"},
      %{name: "Acceptance Criteria", required: true, auto: false, hint: "How we know it is done"}
    ],
    "plan" => [
      %{name: "Objective", required: true, auto: false, hint: "What this plan achieves"},
      %{name: "Non-Negotiables", required: true, auto: false, hint: "Top 3, no exceptions"},
      %{name: "Time Blocks", required: false, auto: false, hint: "Day-by-day allocation"},
      %{name: "Dependencies", required: false, auto: false, hint: "Who/what we are waiting on"},
      %{name: "Success Criteria", required: true, auto: false, hint: "How we measure the week"}
    ],
    "note" => [
      %{name: "Context", required: true, auto: false, hint: "One line — what triggered this"},
      %{name: "Content", required: true, auto: false, hint: "The information"},
      %{name: "Route", required: false, auto: false, hint: "Where this should live permanently"}
    ],
    "decision-log" => [
      %{name: "Decision", required: true, auto: false, hint: "The decision in one sentence"},
      %{name: "Context", required: true, auto: false, hint: "Why this decision was needed"},
      %{name: "Options Considered", required: false, auto: false, hint: "Alternatives evaluated"},
      %{name: "Rationale", required: true, auto: false, hint: "Why this option over others"},
      %{name: "Implications", required: false, auto: false, hint: "What changes downstream"}
    ],
    "standup" => [
      %{name: "Status", required: true, auto: false, hint: "Current state of active work"},
      %{name: "Priorities This Week", required: true, auto: false, hint: "Top 3 this week"},
      %{name: "Blockers", required: false, auto: false, hint: "What is in the way"},
      %{
        name: "Fidelity Check",
        required: false,
        auto: false,
        hint: "Did delegated signals return correctly"
      }
    ],
    "review" => [
      %{
        name: "Single-Loop",
        required: true,
        auto: false,
        hint: "Did the non-negotiables happen?"
      },
      %{
        name: "Double-Loop",
        required: true,
        auto: false,
        hint: "Were they the RIGHT non-negotiables?"
      },
      %{
        name: "Drift Scores",
        required: false,
        auto: false,
        hint: "Alignment scores across 4 dimensions"
      },
      %{name: "Next Week", required: true, auto: false, hint: "Priorities and adjustments"}
    ],
    "report" => [
      %{
        name: "Executive Summary",
        required: true,
        auto: false,
        hint: "1-3 sentences, action-oriented"
      },
      %{name: "Findings", required: true, auto: false, hint: "What was observed"},
      %{name: "Analysis", required: true, auto: false, hint: "What it means"},
      %{name: "Recommendations", required: true, auto: false, hint: "What to do next"}
    ],
    "pitch" => [
      %{
        name: "Hook",
        required: true,
        auto: false,
        hint: "Opening statement that compels attention"
      },
      %{name: "Problem", required: true, auto: false, hint: "What pain are we solving"},
      %{name: "Solution", required: true, auto: false, hint: "Our specific answer"},
      %{
        name: "Proof",
        required: false,
        auto: false,
        hint: "Evidence: results, case studies, data"
      },
      %{name: "Ask", required: true, auto: false, hint: "Specific next step with deadline"}
    ]
  }

  @doc """
  Returns the ordered section list for a genre.
  Falls back to the `note` skeleton for unknown genres.

  ## Examples

      iex> sections = OptimalEngine.Pipeline.Intake.Skeleton.sections_for("transcript")
      iex> Enum.map(sections, & &1.name)
      ["Participants", "Key Points", "Decisions Made", "Action Items", "Open Questions"]
  """
  @spec sections_for(String.t()) :: [section()]
  def sections_for(genre) when is_binary(genre) do
    Map.get(@skeletons, genre, Map.fetch!(@skeletons, "note"))
  end

  @doc """
  Returns all supported genre names.
  """
  @spec supported_genres() :: [String.t()]
  def supported_genres, do: Map.keys(@skeletons)

  @doc """
  Applies a genre skeleton to raw content, producing section-structured markdown.

  The raw content is placed under the first required section as the starting body.
  All remaining sections are rendered as empty headers for the user to fill in.

  ## Example

      apply_skeleton("note", "Operator said requirements are approved")
      # => "## Context\\n\\nOperator said requirements are approved\\n\\n## Content\\n\\n## Route\\n\\n"
  """
  @spec apply_skeleton(String.t(), String.t()) :: String.t()
  def apply_skeleton(genre, raw_content) when is_binary(genre) and is_binary(raw_content) do
    sections = sections_for(genre)
    content = String.trim(raw_content)

    case sections do
      [] ->
        content

      [first | rest] ->
        first_block = "## #{first.name}\n\n#{content}\n\n"
        rest_blocks = Enum.map_join(rest, "", &section_block/1)
        first_block <> rest_blocks
    end
  end

  @doc """
  Returns true if a genre is supported (has a defined skeleton).
  """
  @spec supported?(String.t()) :: boolean()
  def supported?(genre), do: Map.has_key?(@skeletons, genre)

  # Private: render a single non-first section block
  defp section_block(%{name: name, hint: ""}), do: "## #{name}\n\n"
  defp section_block(%{name: name, hint: hint}), do: "## #{name}\n\n<!-- #{hint} -->\n\n"
end
