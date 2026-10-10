defmodule OptimalEngine.MemoryCore.JSON do
  @moduledoc false

  @spec encode(term()) :: String.t()
  def encode(nil), do: "null"
  def encode(value), do: Jason.encode!(value)

  @spec list(term()) :: String.t()
  def list(nil), do: "[]"
  def list(value) when is_list(value), do: Jason.encode!(value)
  def list(value), do: Jason.encode!([value])

  @spec map(term()) :: String.t()
  def map(nil), do: "{}"
  def map(value) when is_map(value), do: Jason.encode!(value)
  def map(value), do: Jason.encode!(%{value: value})
end
