defmodule OptimalEngine.Pipeline.Parser.Backend do
  @moduledoc """
  Behaviour every parser backend implements.

  Two entry points: `parse/2` for file paths (the primary case) and
  `parse_text/2` for inline text with a format hint (used by the Intake stage
  for pasted content that never hits the filesystem).

  Both must return `{:ok, %ParsedDoc{}}` or `{:error, reason}`. Missing
  external tools are reported via `ParsedDoc.warnings`, never via `{:error, …}`.
  """

  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  @callback parse(path :: String.t(), opts :: keyword()) ::
              {:ok, ParsedDoc.t()} | {:error, term()}

  @callback parse_text(text :: String.t(), opts :: keyword()) ::
              {:ok, ParsedDoc.t()} | {:error, term()}

  @optional_callbacks parse_text: 2
end
