defmodule OptimalEngine.MemoryCore.ID do
  @moduledoc """
  Stable ID helpers for Memory Core objects.

  Content-derived IDs are used for source objects so repeated ingestion is
  idempotent inside a workspace. Ledger and execution records use random IDs
  because each activity is an auditable event.
  """

  @spec content_id(String.t(), iodata()) :: String.t()
  def content_id(prefix, content) when is_binary(prefix) do
    hash =
      :crypto.hash(:sha256, IO.iodata_to_binary(content))
      |> Base.encode16(case: :lower)
      |> String.slice(0, 32)

    "#{prefix}_#{hash}"
  end

  @spec random_id(String.t()) :: String.t()
  def random_id(prefix) when is_binary(prefix) do
    suffix =
      :crypto.strong_rand_bytes(16)
      |> Base.encode16(case: :lower)

    "#{prefix}_#{suffix}"
  end

  @spec sha256(String.t()) :: String.t()
  def sha256(value) when is_binary(value) do
    :crypto.hash(:sha256, value)
    |> Base.encode16(case: :lower)
  end
end
