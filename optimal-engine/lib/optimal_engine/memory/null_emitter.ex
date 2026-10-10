defmodule OptimalEngine.Memory.NullEmitter do
  @moduledoc "No-op emitter for standalone use."
  @behaviour OptimalEngine.Memory.Emitter
  @impl true
  def emit(_topic, _payload), do: :ok
end
