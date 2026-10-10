defmodule OptimalEngine.Architecture.Processors.TsFeatureExtractor do
  @moduledoc """
  Time-series feature extractor — classical statistics, zero ML
  dependencies. Extracts `{count, min, max, mean, stddev, trend, last}`
  from a numeric series so the engine has something indexable
  alongside the raw samples.

  This is the pattern for "not every processor is a model". For
  anomaly detection swap in a different processor that wraps a Prophet
  / STL / IsolationForest implementation — the architecture's
  `processor:` hint is the only thing that changes.
  """

  @behaviour OptimalEngine.Architecture.Processor

  @impl true
  def id, do: :ts_feature_extractor

  @impl true
  def modality, do: :time_series

  @impl true
  def emits, do: [:features]

  @impl true
  def init(_config), do: {:ok, %{}}

  @impl true
  def process(_field, series, _state) when is_list(series) and series != [] do
    nums = Enum.filter(series, &is_number/1)

    if nums == [] do
      {:error, :no_numeric_samples}
    else
      n = length(nums)
      sum = Enum.sum(nums)
      mean = sum / n
      sq_dev = Enum.reduce(nums, 0.0, fn x, acc -> acc + (x - mean) * (x - mean) end)
      stddev = if n > 1, do: :math.sqrt(sq_dev / (n - 1)), else: 0.0

      features = %{
        count: n,
        min: Enum.min(nums),
        max: Enum.max(nums),
        mean: mean,
        stddev: stddev,
        last: List.last(nums),
        trend: trend(nums)
      }

      {:ok, %{kind: :features, value: features, metadata: %{method: "classical_stats"}}}
    end
  end

  def process(_field, _value, _state), do: {:error, :not_a_numeric_series}

  # Sign of the linear-regression slope: 1 (rising), -1 (falling), 0 (flat).
  defp trend(nums) when length(nums) < 2, do: 0

  defp trend(nums) do
    n = length(nums)
    xs = Enum.to_list(0..(n - 1))
    xm = (n - 1) / 2
    ym = Enum.sum(nums) / n

    num =
      Enum.zip(xs, nums)
      |> Enum.reduce(0.0, fn {x, y}, acc -> acc + (x - xm) * (y - ym) end)

    cond do
      num > 0 -> 1
      num < 0 -> -1
      true -> 0
    end
  end
end
