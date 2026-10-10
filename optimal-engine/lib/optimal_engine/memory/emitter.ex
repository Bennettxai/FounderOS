defmodule OptimalEngine.Memory.Emitter do
  @moduledoc "Behaviour for event emission. Implement to bridge to your event bus."
  @callback emit(atom(), map()) :: :ok
end
