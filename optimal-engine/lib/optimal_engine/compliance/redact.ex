defmodule OptimalEngine.Compliance.Redact do
  @moduledoc """
  Replace PII matches with deterministic placeholders.

  The default policy substitutes `<REDACTED:kind>` for each match. A
  `:strategy` option supports other modes without changing the call
  site:

      :placeholder   → `<REDACTED:email>` (default)
      :mask          → keep shape, replace each character with `*`
      :hash          → `<REDACTED:email:7a3c9f>` (first 6 hex chars of sha256)
      :remove        → drop the matched bytes entirely

  Redaction runs on a byte-offset basis so Unicode content doesn't
  shift — a UTF-8-safe implementation costs more CPU and isn't needed
  for the ASCII-heavy PII patterns we match today. Revisit when the
  kind list grows past ASCII (DOB written in Cyrillic, say).
  """

  alias OptimalEngine.Compliance.PII

  @type strategy :: :placeholder | :mask | :hash | :remove

  @type redact_opts :: [strategy: strategy(), only: [PII.kind()], except: [PII.kind()]]

  @type report :: %{
          redacted: String.t(),
          matches: [PII.match()],
          strategy: strategy()
        }

  @doc """
  Produce a redacted copy of `text` plus a report of every match.
  """
  @spec redact(String.t(), redact_opts()) :: report()
  def redact(text, opts \\ []) when is_binary(text) do
    strategy = Keyword.get(opts, :strategy, :placeholder)
    only = Keyword.get(opts, :only, :all)
    except = Keyword.get(opts, :except, [])

    matches =
      text
      |> PII.scan()
      |> Enum.filter(fn m -> include?(m.kind, only, except) end)

    redacted = apply_redactions(text, matches, strategy)

    %{redacted: redacted, matches: matches, strategy: strategy}
  end

  @doc "Same as `redact/2` but returns only the redacted string."
  @spec redact!(String.t(), redact_opts()) :: String.t()
  def redact!(text, opts \\ []), do: redact(text, opts).redacted

  # ─── private ─────────────────────────────────────────────────────────────

  defp include?(_kind, :all, []), do: true
  defp include?(kind, :all, except), do: kind not in except
  defp include?(kind, only, except) when is_list(only), do: kind in only and kind not in except

  defp apply_redactions(text, [], _strategy), do: text

  defp apply_redactions(text, matches, strategy) do
    # Walk matches from right to left so offsets remain valid after each splice.
    matches
    |> Enum.sort_by(& &1.offset, :desc)
    |> Enum.reduce(text, fn match, acc -> splice(acc, match, strategy) end)
  end

  defp splice(text, match, strategy) do
    before = binary_part(text, 0, match.offset)

    after_ =
      binary_part(text, match.offset + match.length, byte_size(text) - match.offset - match.length)

    replacement = replacement(match, strategy)
    before <> replacement <> after_
  end

  defp replacement(match, :placeholder), do: "<REDACTED:#{match.kind}>"

  defp replacement(match, :mask), do: String.duplicate("*", match.length)

  defp replacement(match, :hash) do
    digest =
      :crypto.hash(:sha256, match.value)
      |> Base.encode16(case: :lower)
      |> String.slice(0, 6)

    "<REDACTED:#{match.kind}:#{digest}>"
  end

  defp replacement(_match, :remove), do: ""
end
