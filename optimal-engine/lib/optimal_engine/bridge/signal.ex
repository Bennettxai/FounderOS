defmodule OptimalEngine.Bridge.Signal do
  @moduledoc """
  Bridge to OptimalEngine.Signal.Core — CloudEvents envelopes, classification, and quality auditing.

  OptimalEngine.Signal.Core is stateless (no GenServers). All functions are pure wrappers.

  ## Integration points

  - `classify/1` — Augments OptimalEngine.Pipeline.Classifier with OptimalEngine.Signal.Classifier's
    inference (mode, genre, type, format, structure) for improved accuracy
  - `audit/1` — Runs OptimalEngine.Signal.FailureModes.detect/1 against a signal to produce
    a list of Signal Theory constraint violations (Shannon, Ashby, Beer, Wiener)
  - `measure_sn/1` — Computes S/N ratio from resolved signal dimensions
  - `wrap/1` — Wraps an Optimal signal into a CloudEvents v1.0.2 envelope
  """

  alias OptimalEngine.Signal, as: OptSignal
  alias OptimalEngine.Signal.Envelope

  @doc """
  Augments an OptimalEngine.Signal with OptimalEngine.Signal.Core classification.

  Takes an already-classified OptSignal and fills any nil dimensions using
  OptimalEngine.Signal.Classifier inference. Returns the enriched signal.
  """
  @spec enhance_classification(OptSignal.t(), String.t()) :: OptSignal.t()
  def enhance_classification(%OptSignal{} = signal, raw_text) do
    classification =
      signal
      |> to_envelope(raw_text)
      |> OptimalEngine.Signal.Classifier.classify()

    # Only fill dimensions that are nil or default
    signal
    |> maybe_fill(:mode, classification[:mode], :linguistic)
    |> maybe_fill_genre(classification[:genre])
    |> maybe_fill(:type, classification[:type], :inform)
  end

  @doc """
  Audits a signal for Signal Theory constraint violations.

  Returns a list of `{failure_mode, description}` tuples. Empty list = clean signal.
  """
  @spec audit(OptSignal.t()) :: [{atom(), String.t()}]
  def audit(%OptSignal{} = signal) do
    signal
    |> to_envelope()
    |> OptimalEngine.Signal.Classifier.failure_modes()
  rescue
    _ -> []
  end

  @doc """
  Computes the S/N ratio for a signal based on its resolved dimensions.
  """
  @spec measure_sn(OptSignal.t()) :: float()
  def measure_sn(%OptSignal{} = signal) do
    case signal |> to_envelope() |> OptimalEngine.Signal.Classifier.sn_ratio() do
      ratio when is_float(ratio) -> ratio
      _ -> signal.sn_ratio || 0.5
    end
  rescue
    _ -> signal.sn_ratio || 0.5
  end

  @doc """
  Wraps an Optimal signal as a CloudEvents v1.0.2 envelope map.
  """
  @spec to_cloud_event(OptSignal.t()) :: map()
  def to_cloud_event(%OptSignal{} = signal) do
    case signal |> to_envelope() |> Envelope.to_cloud_event() do
      %{} = event -> event
      _ -> %{}
    end
  rescue
    _ -> %{}
  end

  defp to_envelope(signal, raw_text \\ nil)

  defp to_envelope(%OptSignal{} = signal, raw_text) do
    Envelope.new!(
      signal_type_string(signal),
      source: signal.node || "optimal-engine",
      data: %{"text" => raw_text || signal.content || ""},
      signal_mode: normalize_mode(signal.mode),
      signal_genre: normalize_genre(signal.genre),
      signal_type: normalize_type(signal.type),
      signal_format: normalize_format(signal.format),
      signal_structure:
        if(is_binary(signal.structure) and signal.structure != "",
          do: String.to_atom(signal.structure),
          else: nil
        ),
      signal_sn_ratio: signal.sn_ratio
    )
  end

  defp normalize_genre(nil), do: nil
  defp normalize_genre(g) when is_atom(g), do: g
  defp normalize_genre("spec"), do: :spec
  defp normalize_genre("report"), do: :report
  defp normalize_genre("brief"), do: :brief
  defp normalize_genre("pr"), do: :pr
  defp normalize_genre("adr"), do: :adr
  defp normalize_genre("chat"), do: :chat
  defp normalize_genre("error"), do: :error
  defp normalize_genre("command"), do: :command
  defp normalize_genre("event"), do: :event
  defp normalize_genre("metric"), do: :metric
  defp normalize_genre(_), do: nil

  defp normalize_format(nil), do: nil
  defp normalize_format(:markdown), do: :markdown
  defp normalize_format(:code), do: :code
  defp normalize_format(:json), do: :json
  defp normalize_format(_), do: :markdown

  defp signal_type_string(%OptSignal{genre: g, type: t}) do
    "optimal.#{g || "note"}.#{t || "inform"}"
  end

  defp maybe_fill(signal, :mode, inferred, default) do
    if signal.mode == nil or signal.mode == default do
      case inferred do
        nil -> signal
        val -> %{signal | mode: normalize_mode(val)}
      end
    else
      signal
    end
  end

  defp maybe_fill(signal, :type, inferred, default) do
    if signal.type == nil or signal.type == default do
      case inferred do
        nil -> signal
        val -> %{signal | type: normalize_type(val)}
      end
    else
      signal
    end
  end

  defp maybe_fill_genre(signal, nil), do: signal
  defp maybe_fill_genre(%{genre: g} = signal, _) when g != nil and g != "note", do: signal
  defp maybe_fill_genre(signal, inferred), do: %{signal | genre: to_string(inferred)}

  defp normalize_mode(:code), do: :code
  defp normalize_mode(:visual), do: :visual
  defp normalize_mode(:data), do: :data
  defp normalize_mode(:mixed), do: :mixed
  defp normalize_mode(_), do: :linguistic

  defp normalize_type(:direct), do: :direct
  defp normalize_type(:commit), do: :commit
  defp normalize_type(:decide), do: :decide
  defp normalize_type(:express), do: :express
  defp normalize_type(_), do: :inform
end
