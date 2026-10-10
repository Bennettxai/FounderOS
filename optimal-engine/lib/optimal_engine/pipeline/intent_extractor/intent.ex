defmodule OptimalEngine.Pipeline.IntentExtractor.Intent do
  @moduledoc """
  Per-chunk extracted intent.

  Mirrors the `intents` table (Phase 1 migration 005):

      %Intent{chunk_id, tenant_id, intent, confidence, evidence}

  `intent` is one of ten fixed values — the enum is the engine's answer to
  "what is this signal *for*?" Not the speech act (that's
  `Classification.signal_type`), not the shape (that's genre + structure) —
  the goal the signal is trying to accomplish.

  The ten values are the canonical Signal Theory intent set; see
  `docs/architecture/ARCHITECTURE.md` §Stage 4.

      :request_info       asking for something
      :propose_decision   putting a decision on the table
      :record_fact        stating something as ground truth
      :express_concern    flagging risk / blocker
      :commit_action      taking on a task
      :reference          pointing at other context
      :narrate            describing a sequence of events
      :reflect            analyzing past signals
      :specify            defining a contract or requirement
      :measure            reporting a metric or quantity

  `confidence` is `0.0..1.0`. `evidence` is the snippet of text that drove the
  inference, so auditors can spot-check.
  """

  @intents ~w(
    request_info propose_decision record_fact express_concern commit_action
    reference narrate reflect specify measure
  )a

  @type kind ::
          :request_info
          | :propose_decision
          | :record_fact
          | :express_concern
          | :commit_action
          | :reference
          | :narrate
          | :reflect
          | :specify
          | :measure

  @type t :: %__MODULE__{
          chunk_id: String.t(),
          tenant_id: String.t(),
          intent: kind(),
          confidence: float(),
          evidence: String.t() | nil
        }

  defstruct chunk_id: nil,
            tenant_id: "default",
            intent: :record_fact,
            confidence: 0.5,
            evidence: nil

  @doc "The full ordered list of canonical intent values."
  @spec all() :: [kind()]
  def all, do: @intents

  @doc "Build an Intent with sensible defaults."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields), do: struct(__MODULE__, fields)

  @doc "Validates that an atom is a canonical intent value."
  @spec valid?(atom()) :: boolean()
  def valid?(kind) when is_atom(kind), do: kind in @intents
end
