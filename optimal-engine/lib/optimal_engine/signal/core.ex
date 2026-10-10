defmodule OptimalEngine.Signal.Core do
  @moduledoc """
  CloudEvents v1.0.2 + Signal Theory foundation for the MIOSA ecosystem.

  Every event in MIOSA is a **Signal** — a standardized envelope (CloudEvents)
  annotated with quality dimensions from Signal Theory: Mode + Genre + Type + Format + Structure.

  ## Quick Start

      # Create a signal
      {:ok, signal} = OptimalEngine.Signal.new("miosa.agent.task.completed",
        source: "/agent/backend-1",
        data: %{task_id: "t-123", result: :success}
      )

      # Classify it with Signal Theory dimensions
      {:ok, classified} = OptimalEngine.Signal.classify(signal,
        mode: :code,
        genre: :brief,
        type: :inform,
        format: :json,
        structure: :task_brief
      )

      # Check S/N ratio
      ratio = OptimalEngine.Signal.sn_ratio(classified)

      # Serialize to CloudEvents JSON
      json = OptimalEngine.Signal.to_json(classified)

  ## Architecture

  - `OptimalEngine.Signal.Envelope` — core struct (CloudEvents + Signal Theory)
  - `OptimalEngine.Signal.Classifier` — auto-classification and constraint checking
  - `OptimalEngine.Signal.Router` — trie-based pattern matching for signal routing
  - `OptimalEngine.Signal.Dispatcher` — delivery engine with pluggable adapters
  - `OptimalEngine.Signal.Journal` — ETS-backed history and causality tracking
  """

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier

  @doc """
  Creates a new signal with the given type and options.

  Delegates to `OptimalEngine.Signal.Envelope.new/2`.

  ## Examples

      iex> {:ok, signal} = OptimalEngine.Signal.new("miosa.test.event", source: "/test")
      iex> signal.type
      "miosa.test.event"
  """
  @spec new(String.t(), keyword()) :: {:ok, Signal.t()} | {:error, term()}
  defdelegate new(type, opts \\ []), to: Signal

  @doc """
  Creates a new signal, raising on validation failure.

  Delegates to `OptimalEngine.Signal.Envelope.new!/2`.

  ## Examples

      iex> signal = OptimalEngine.Signal.new!("miosa.test.event", source: "/test")
      iex> signal.specversion
      "1.0.2"
  """
  @spec new!(String.t(), keyword()) :: Signal.t()
  defdelegate new!(type, opts \\ []), to: Signal

  @doc """
  Classifies a signal with Signal Theory dimensions.

  Delegates to `OptimalEngine.Signal.Envelope.classify/2`.

  ## Examples

      iex> signal = OptimalEngine.Signal.new!("miosa.test", source: "/test")
      iex> {:ok, classified} = OptimalEngine.Signal.classify(signal, mode: :code, genre: :spec)
      iex> classified.signal_mode
      :code
  """
  @spec classify(Signal.t(), keyword()) :: {:ok, Signal.t()} | {:error, term()}
  defdelegate classify(signal, opts), to: Signal

  @doc """
  Auto-classifies a signal using heuristics from the Classifier.

  Returns the signal with all five Signal Theory dimensions populated.

  ## Examples

      iex> signal = OptimalEngine.Signal.new!("miosa.agent.task.completed", source: "/agent/1", data: %{result: "ok"})
      iex> {:ok, classified} = OptimalEngine.Signal.auto_classify(signal)
      iex> classified.signal_mode != nil
      true
  """
  @spec auto_classify(Signal.t()) :: {:ok, Signal.t()} | {:error, term()}
  def auto_classify(%Signal{} = signal) do
    classification = Classifier.classify(signal)
    sn = Classifier.sn_ratio(signal)

    Signal.classify(signal,
      mode: classification.mode,
      genre: classification.genre,
      type: classification.type,
      format: classification.format,
      structure: classification.structure,
      sn_ratio: sn
    )
  end

  @doc """
  Validates a signal against CloudEvents and Signal Theory constraints.

  Delegates to `OptimalEngine.Signal.Envelope.validate/1`.
  """
  @spec validate(Signal.t()) :: :ok | {:error, [String.t()]}
  defdelegate validate(signal), to: Signal

  @doc """
  Creates a child signal linked to the parent via causality chain.

  Delegates to `OptimalEngine.Signal.Envelope.chain/3`.
  """
  @spec chain(Signal.t(), String.t(), keyword()) :: {:ok, Signal.t()} | {:error, term()}
  defdelegate chain(parent, child_type, opts \\ []), to: Signal

  @doc """
  Estimates the signal-to-noise ratio.

  Delegates to `OptimalEngine.Signal.Classifier.sn_ratio/1`.
  """
  @spec sn_ratio(Signal.t()) :: float()
  defdelegate sn_ratio(signal), to: Classifier

  @doc """
  Serializes a signal to CloudEvents JSON string.

  ## Examples

      iex> signal = OptimalEngine.Signal.new!("miosa.test", source: "/test", data: %{x: 1})
      iex> json = OptimalEngine.Signal.to_json(signal)
      iex> is_binary(json)
      true
  """
  @spec to_json(Signal.t()) :: String.t()
  def to_json(%Signal{} = signal) do
    signal
    |> Signal.to_cloud_event()
    |> Jason.encode!()
  end

  @doc """
  Deserializes a CloudEvents JSON string into a Signal.

  ## Examples

      iex> signal = OptimalEngine.Signal.new!("miosa.test", source: "/test", data: %{"x" => 1})
      iex> json = OptimalEngine.Signal.to_json(signal)
      iex> {:ok, restored} = OptimalEngine.Signal.from_json(json)
      iex> restored.type
      "miosa.test"
  """
  @spec from_json(String.t()) :: {:ok, Signal.t()} | {:error, term()}
  def from_json(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, map} -> Signal.from_cloud_event(map)
      {:error, _} = err -> err
    end
  end

  @doc """
  Checks all four governing constraints for a signal.

  Delegates to `OptimalEngine.Signal.Classifier.check_constraints/2`.
  """
  @spec check_constraints(Signal.t(), keyword()) :: map()
  defdelegate check_constraints(signal, opts \\ []), to: Classifier

  @doc """
  Detects Signal Theory failure modes.

  Delegates to `OptimalEngine.Signal.Classifier.failure_modes/1`.
  """
  @spec failure_modes(Signal.t()) :: [{Classifier.failure_mode(), String.t()}]
  defdelegate failure_modes(signal), to: Classifier
end
