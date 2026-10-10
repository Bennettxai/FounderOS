defmodule OptimalEngine.Memory.Cortex.Provider do
  @moduledoc """
  Behaviour for Cortex LLM providers.

  Implement `chat/2` to supply an LLM backend for memory synthesis.

  ## Example

      defmodule MyApp.LLMBridge do
        @behaviour OptimalEngine.Memory.Cortex.Provider

        @impl true
        def chat(messages, opts) do
          # call your LLM here
          {:ok, %{content: "synthesized bulletin"}}
        end
      end

  Then configure:

      config :optimal_engine, cortex_provider: MyApp.LLMBridge
  """

  @type message :: %{role: String.t(), content: String.t()}
  @type opt :: {:max_tokens, pos_integer()} | {:temperature, float()}

  @callback chat([message()], [opt()]) :: {:ok, %{content: String.t()}} | {:error, term()}
end
