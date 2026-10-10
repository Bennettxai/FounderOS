defmodule OptimalEngine.Architecture.Architectures.TimeSeriesWindow do
  @moduledoc """
  A window of a numeric time series — IoT telemetry, vitals, market
  ticks, app metrics, any sampled signal.

  A classical retrieval stack can't do anything useful with a raw
  time series. This architecture lets downstream processors emit
  **features** (FFT coefficients, rolling stats, anomaly scores) that
  the engine stores alongside the raw samples, so queries like
  "spikes in latency above 3σ" have something to match against.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "time_series_window",
      version: 1,
      description: "Sampled numeric signal over a time window",
      modality_primary: :time_series,
      granularity: [:window, :sample],
      fields: [
        %Field{
          name: :series,
          modality: :time_series,
          required: true,
          dims: [:any],
          processor: :ts_feature_extractor,
          description: "The numeric samples in acquisition order"
        },
        %Field{
          name: :timestamps,
          modality: :time_series,
          required: true,
          dims: [:any],
          description: "Parallel array of epoch-ms timestamps"
        },
        %Field{
          name: :sample_rate_hz,
          modality: :structured,
          required: false,
          description: "Nominal sample rate (Hz)"
        },
        %Field{
          name: :source,
          modality: :structured,
          required: false,
          description: "Device id / metric name / probe origin"
        },
        %Field{
          name: :features,
          modality: :tensor,
          required: false,
          dims: [:any],
          description: "Derived feature vector (FFT / stats / anomaly)"
        },
        %Field{
          name: :label,
          modality: :text,
          required: false,
          description: "Optional human annotation for this window"
        }
      ]
    )
  end
end
