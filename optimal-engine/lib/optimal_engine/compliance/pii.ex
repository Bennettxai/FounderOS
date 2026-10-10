defmodule OptimalEngine.Compliance.PII do
  @moduledoc """
  Personally Identifiable Information detection.

  Finds common PII patterns in free-form text and reports each match
  with a type, offset, and the substring matched. The detector is
  deliberately regex-driven: fast, no ML dependency, good enough to
  back a redactor + a screening alert before anything richer ships
  in a later phase.

  ## Detected categories

      :email         — RFC-ish `local@host.tld`
      :phone         — North-American 10-digit + common international forms
      :ssn           — US 9-digit `NNN-NN-NNNN`
      :credit_card   — 13–19 digit sequences that pass Luhn
      :ipv4          — `a.b.c.d` with each octet in [0,255]
      :url           — http(s)://host(:port)(/path)
      :ip_address    — alias for `:ipv4` (kept for schema stability)

  HIPAA adds `:mrn` (medical-record-number) and `:dob` (date of birth)
  but those are dialect-specific — wire them per-tenant rather than in
  the global regex set. See `OptimalEngine.Compliance.Redact.configure/1`.
  """

  @type kind :: :email | :phone | :ssn | :credit_card | :ipv4 | :url | :ip_address
  @type match :: %{
          kind: kind(),
          value: String.t(),
          offset: non_neg_integer(),
          length: non_neg_integer()
        }

  @patterns %{
    email: ~r/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/,
    phone: ~r/(?:\+?1[\s.-]?)?(?:\(?\d{3}\)?[\s.-]?)\d{3}[\s.-]?\d{4}\b|\+[1-9]\d{1,14}\b/,
    ssn: ~r/\b(?!000|666|9\d{2})\d{3}-(?!00)\d{2}-(?!0000)\d{4}\b/,
    ipv4: ~r/\b(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\b/,
    url: ~r/https?:\/\/[^\s<>"']+/,
    credit_card_candidate: ~r/\b(?:\d[ -]?){13,19}\b/
  }

  @doc """
  Scan `text` and return every PII match in document order.
  """
  @spec scan(String.t()) :: [match()]
  def scan(text) when is_binary(text) do
    (scan_basic(text) ++ scan_credit_cards(text))
    |> Enum.sort_by(& &1.offset)
  end

  @doc "`true` when `scan/1` returns at least one match."
  @spec any?(String.t()) :: boolean()
  def any?(text) when is_binary(text), do: scan(text) != []

  @doc "Return the distinct set of PII `kind` atoms present in `text`."
  @spec kinds_present(String.t()) :: [kind()]
  def kinds_present(text) when is_binary(text) do
    text |> scan() |> Enum.map(& &1.kind) |> Enum.uniq()
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp scan_basic(text) do
    Enum.flat_map(@patterns, fn
      {:credit_card_candidate, _} ->
        []

      {kind, re} ->
        Regex.scan(re, text, return: :index)
        |> Enum.map(fn [{off, len}] ->
          %{
            kind: kind,
            value: :binary.part(text, off, len),
            offset: off,
            length: len
          }
        end)
    end)
  end

  defp scan_credit_cards(text) do
    Regex.scan(@patterns.credit_card_candidate, text, return: :index)
    |> Enum.flat_map(fn [{off, len}] ->
      candidate = :binary.part(text, off, len)
      digits = String.replace(candidate, ~r/[^0-9]/, "")

      if byte_size(digits) >= 13 and luhn_valid?(digits) do
        [%{kind: :credit_card, value: candidate, offset: off, length: len}]
      else
        []
      end
    end)
  end

  # Luhn check: duplicate every second digit from the right; sum mod 10 = 0.
  defp luhn_valid?(digits) when is_binary(digits) do
    digits
    |> String.graphemes()
    |> Enum.reverse()
    |> Enum.with_index()
    |> Enum.reduce(0, fn {d, idx}, acc ->
      n = String.to_integer(d)

      cond do
        rem(idx, 2) == 0 -> acc + n
        n * 2 > 9 -> acc + n * 2 - 9
        true -> acc + n * 2
      end
    end)
    |> rem(10)
    |> Kernel.==(0)
  end
end
