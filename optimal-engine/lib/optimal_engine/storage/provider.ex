defmodule OptimalEngine.Storage.Provider do
  @moduledoc """
  Contract for physical storage providers.

  Logical stores describe data ownership and lifecycle. Providers describe the
  replaceable technology used to persist, index, transport, or project that
  data. Provider status must never expose credentials or connection strings.
  """

  @type health :: :healthy | :unreachable | :not_configured | :not_installed | :unknown

  @callback id() :: String.t()
  @callback capabilities() :: [atom()]
  @callback describe() :: map()
  @callback status(keyword()) :: map()
end
