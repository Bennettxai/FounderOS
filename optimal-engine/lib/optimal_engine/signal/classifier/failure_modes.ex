defmodule OptimalEngine.Signal.Classifier.FailureModes do
  @moduledoc """
  Detection logic for all 11 Signal Theory failure modes.

  Failure modes are grouped by governing constraint:

  **Shannon violations** (channel capacity)
  - `routing_failure` — no source, cannot route signal to origin
  - `bandwidth_overload` — data payload too large for channel
  - `fidelity_failure` — S/N ratio below threshold, meaning lost in noise

  **Ashby violations** (variety / repertoire)
  - `genre_mismatch` — declared genre contradicts type-inferred genre
  - `variety_failure` — no Signal Theory dimensions resolved at all
  - `structure_failure` — partial classification (some dimensions unresolved)

  **Beer violations** (viable structure)
  - `bridge_failure` — excessive extensions create context overhead
  - `herniation_failure` — parent_id set without correlation_id (broken causality)
  - `decay_failure` — signal is older than 24 hours (potentially stale)

  **Wiener violations** (feedback loop)
  - `feedback_failure` — direct-type signal with no correlation established

  **Cross-cutting**
  - `adversarial_noise` — extreme extension count suggests noise injection
  """

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Classifier.Analyzer

  @doc """
  Runs all 11 failure-mode detectors against `signal`.

  Returns a list of `{failure_mode, description}` tuples. An empty list means
  no failures were detected.
  """
  @spec detect(Signal.t()) :: [{atom(), String.t()}]
  def detect(%Signal{} = signal) do
    []
    |> detect_shannon_failures(signal)
    |> detect_ashby_failures(signal)
    |> detect_beer_failures(signal)
    |> detect_wiener_failures(signal)
    |> detect_adversarial(signal)
    |> Enum.reverse()
  end

  # ── Shannon violations ───────────────────────────────────────────

  defp detect_shannon_failures(failures, signal) do
    data_size =
      signal.data
      |> inspect()
      |> byte_size()

    failures =
      if is_nil(signal.source) do
        [{:routing_failure, "no source — cannot route signal to origin"} | failures]
      else
        failures
      end

    failures =
      if data_size > 100_000 do
        [
          {:bandwidth_overload, "data payload is #{data_size} bytes — consider chunking"}
          | failures
        ]
      else
        failures
      end

    if signal.signal_sn_ratio != nil and signal.signal_sn_ratio < 0.3 do
      [
        {:fidelity_failure, "S/N ratio #{signal.signal_sn_ratio} — meaning likely lost in noise"}
        | failures
      ]
    else
      failures
    end
  end

  # ── Ashby violations ────────────────────────────────────────────

  defp detect_ashby_failures(failures, signal) do
    dims = [
      signal.signal_mode,
      signal.signal_genre,
      signal.signal_type,
      signal.signal_format,
      signal.signal_structure
    ]

    nil_count = Enum.count(dims, &is_nil/1)

    failures =
      if nil_count == 5 do
        [
          {:variety_failure, "no Signal Theory dimensions resolved — raw unclassified event"}
          | failures
        ]
      else
        failures
      end

    failures =
      if nil_count > 0 and nil_count < 5 do
        [
          {:structure_failure, "#{nil_count} of 5 dimensions unresolved — partial classification"}
          | failures
        ]
      else
        failures
      end

    if signal.signal_genre != nil do
      inferred = Analyzer.infer_genre(signal)

      if inferred != signal.signal_genre and inferred != :chat do
        [
          {:genre_mismatch,
           "signal_genre :#{signal.signal_genre} doesn't match type-inferred :#{inferred}"}
          | failures
        ]
      else
        failures
      end
    else
      failures
    end
  end

  # ── Beer violations ─────────────────────────────────────────────

  defp detect_beer_failures(failures, signal) do
    failures =
      if signal.parent_id != nil and signal.correlation_id == nil do
        [
          {:herniation_failure, "parent_id set without correlation_id — broken causality chain"}
          | failures
        ]
      else
        failures
      end

    failures =
      if signal.extensions != nil and map_size(signal.extensions) > 20 do
        [
          {:bridge_failure,
           "#{map_size(signal.extensions)} extensions — excessive context overhead"}
          | failures
        ]
      else
        failures
      end

    if signal.time != nil do
      age = DateTime.diff(DateTime.utc_now(), signal.time, :second)

      if age > 86_400 do
        [{:decay_failure, "signal is #{div(age, 3600)} hours old — may be stale"} | failures]
      else
        failures
      end
    else
      failures
    end
  end

  # ── Wiener violations ───────────────────────────────────────────

  defp detect_wiener_failures(failures, signal) do
    if signal.signal_type == :direct and is_nil(signal.correlation_id) and
         is_nil(signal.parent_id) do
      [
        {:feedback_failure,
         "direct-type signal with no correlation_id — no feedback loop established"}
        | failures
      ]
    else
      failures
    end
  end

  # ── Adversarial noise (cross-cutting) ───────────────────────────

  defp detect_adversarial(failures, signal) do
    if signal.extensions != nil and map_size(signal.extensions) > 50 do
      [
        {:adversarial_noise, "#{map_size(signal.extensions)} extensions — possible noise injection"}
        | failures
      ]
    else
      failures
    end
  end
end
