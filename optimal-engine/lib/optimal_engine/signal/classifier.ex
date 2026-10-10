defmodule OptimalEngine.Signal.Classifier do
  @moduledoc """
  Signal Theory classifier — analyzes signal content to suggest Mode + Genre + Type + Format + Structure dimensions,
  estimate signal-to-noise ratio, verify the four governing constraints, and detect failure modes.

  This module is the intelligence layer that makes Signal Theory actionable. It provides:

  - **Auto-classification** of signals based on type and data heuristics
  - **S/N ratio estimation** based on data density and structure
  - **Constraint verification** (Shannon, Ashby, Beer, Wiener)
  - **Failure mode detection** across all 11 Signal Theory failure modes

  Implementation is split across two submodules:

  - `OptimalEngine.Signal.Classifier.Analyzer` — inference and scoring helpers
  - `OptimalEngine.Signal.Classifier.FailureModes` — 11 failure mode detectors
  """

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier.Analyzer
  alias OptimalEngine.Signal.Classifier.FailureModes

  @type classification :: %{
          mode: Signal.signal_mode() | nil,
          genre: Signal.signal_genre() | nil,
          type: Signal.signal_type() | nil,
          format: Signal.signal_format() | nil,
          structure: atom() | nil
        }

  @type constraint_result :: :ok | {:violation, String.t()}

  @type failure_mode ::
          :routing_failure
          | :bandwidth_overload
          | :fidelity_failure
          | :genre_mismatch
          | :variety_failure
          | :structure_failure
          | :bridge_failure
          | :herniation_failure
          | :decay_failure
          | :feedback_failure
          | :adversarial_noise

  @doc """
  Analyzes a signal and suggests classification for all five Signal Theory dimensions.

  Uses the signal's `type` string, `data` content, and existing fields to infer
  mode, genre, type, format, and structure.

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.agent.task.completed", source: "/agent/1", data: %{result: "ok"})
      iex> classification = OptimalEngine.Signal.Classifier.classify(signal)
      iex> classification.genre
      :brief
  """
  @spec classify(Signal.t()) :: classification()
  def classify(%Signal{} = signal) do
    %{
      mode: Analyzer.infer_mode(signal),
      genre: Analyzer.infer_genre(signal),
      type: Analyzer.infer_type(signal),
      format: Analyzer.infer_format(signal),
      structure: Analyzer.infer_structure(signal)
    }
  end

  @doc """
  Estimates the signal-to-noise ratio of a signal (0.0 to 1.0).

  The estimation considers:
  - Whether all Signal Theory dimensions are resolved (higher = less noise)
  - Whether the data field is present and structured
  - Whether the type follows reverse-DNS convention
  - Whether agent context is complete

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/t", signal_mode: :code, signal_genre: :spec, signal_type: :inform, signal_format: :json, signal_structure: :openapi, data: %{x: 1})
      iex> ratio = OptimalEngine.Signal.Classifier.sn_ratio(signal)
      iex> ratio > 0.5
      true
  """
  @spec sn_ratio(Signal.t()) :: float()
  def sn_ratio(%Signal{} = signal) do
    scores = [
      Analyzer.dimension_score(signal),
      Analyzer.data_score(signal),
      Analyzer.type_score(signal),
      Analyzer.context_score(signal)
    ]

    total = Enum.sum(scores)
    max_score = length(scores) * 1.0

    Float.round(total / max_score, 2)
  end

  @doc """
  Checks all four governing constraints (Shannon, Ashby, Beer, Wiener).

  Returns a map of constraint name to result. Pass options to configure thresholds.

  ## Options

  - `:max_bytes` — Shannon bandwidth limit (default: 1_000_000)
  - `:acknowledged_ids` — list of acknowledged signal IDs for Wiener check (default: [])

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/t", signal_mode: :code, signal_genre: :spec, signal_type: :inform, signal_format: :json, signal_structure: :default)
      iex> results = OptimalEngine.Signal.Classifier.check_constraints(signal)
      iex> results.ashby
      :ok
  """
  @spec check_constraints(Signal.t(), keyword()) :: %{
          shannon: constraint_result(),
          ashby: constraint_result(),
          beer: constraint_result(),
          wiener: constraint_result()
        }
  def check_constraints(%Signal{} = signal, opts \\ []) do
    max_bytes = Keyword.get(opts, :max_bytes, 1_000_000)
    ack_ids = Keyword.get(opts, :acknowledged_ids, [])

    %{
      shannon: Signal.shannon_check(signal, max_bytes),
      ashby: Signal.ashby_check(signal),
      beer: Signal.beer_check(signal),
      wiener: Signal.wiener_check(signal, ack_ids)
    }
  end

  @doc """
  Detects which of the 11 Signal Theory failure modes apply to a signal.

  Returns a list of `{failure_mode, description}` tuples for all detected failures.
  An empty list means no failures detected.

  ## The 11 Failure Modes

  **Shannon violations:** routing_failure, bandwidth_overload, fidelity_failure
  **Ashby violations:** genre_mismatch, variety_failure, structure_failure
  **Beer violations:** bridge_failure, herniation_failure, decay_failure
  **Wiener violations:** feedback_failure
  **Cross-cutting:** adversarial_noise

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/t")
      iex> failures = OptimalEngine.Signal.Classifier.failure_modes(signal)
      iex> is_list(failures)
      true
  """
  @spec failure_modes(Signal.t()) :: [{failure_mode(), String.t()}]
  def failure_modes(%Signal{} = signal) do
    FailureModes.detect(signal)
  end
end
