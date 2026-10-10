defmodule OptimalEngine.Wiki.Directives do
  @moduledoc """
  The executable-directive layer of the Wiki.

  Wiki pages carry literal `{{verb: argument [key=value …]}}` tokens in their
  body. This module lexes those tokens into `%Directive{}` structs, renders
  them into one of four output formats, and verifies the argument against
  a whitelist so no arbitrary-code execution is possible.

  ## Whitelist

      cite     — URI pointer rendered as a footnote
      include  — inline the referenced content (tier optional)
      expand   — sub-query against the wiki for a slug
      search   — invoke hybrid retrieval
      table    — fetch a structured row (CSV/sheet)
      trace    — walk the knowledge graph from an entity
      recent   — inject recent signals from a node
      wikilink — `[[slug]]` (non-directive syntax, treated here for parity)

  Any verb not in the whitelist is rendered verbatim with a warning so
  humans can spot typos without pages silently losing content.

  ## Output formats

  - `:plain`   — inline-resolved, no markup (for `optimal rag --format text`)
  - `:markdown`— footnote-style citations + markdown content inclusions
  - `:claude`  — XML `<document index="N" source="URI">…</document>` blocks
  - `:openai`  — JSON message array `[%{role, content}]`
  """

  @directive_re ~r/\{\{\s*([a-z_]+)\s*:\s*([^}]*?)\s*\}\}/
  @wikilink_re ~r/\[\[([^\]]+)\]\]/

  @whitelist ~w(cite include expand search table trace recent)a

  defmodule Directive do
    @moduledoc false
    @enforce_keys [:verb, :argument]
    defstruct [:verb, :argument, options: %{}, raw: nil, offset: 0, length: 0]
  end

  @type resolver :: (Directive.t(), keyword() -> {:ok, String.t(), map()} | {:error, term()})

  @doc """
  Parse a wiki body into a list of `%Directive{}` tokens + `[[wikilink]]`
  references, in document order. Positions preserved for re-splicing.
  """
  @spec parse(String.t()) :: {:ok, [Directive.t()]}
  def parse(body) when is_binary(body) do
    directives =
      Regex.scan(@directive_re, body, return: :index)
      |> Enum.map(fn [{full_off, full_len}, {verb_off, verb_len}, {arg_off, arg_len}] ->
        verb_str = :binary.part(body, verb_off, verb_len)
        arg_str = :binary.part(body, arg_off, arg_len)
        raw = :binary.part(body, full_off, full_len)

        {verb, options} = parse_argument(arg_str)

        %Directive{
          verb: String.to_atom(verb_str),
          argument: verb,
          options: options,
          raw: raw,
          offset: full_off,
          length: full_len
        }
      end)

    wikilinks =
      Regex.scan(@wikilink_re, body, return: :index)
      |> Enum.map(fn [{full_off, full_len}, {slug_off, slug_len}] ->
        slug = :binary.part(body, slug_off, slug_len)
        raw = :binary.part(body, full_off, full_len)

        %Directive{
          verb: :wikilink,
          argument: slug,
          options: %{},
          raw: raw,
          offset: full_off,
          length: full_len
        }
      end)

    all = (directives ++ wikilinks) |> Enum.sort_by(& &1.offset)
    {:ok, all}
  end

  @doc "Returns `true` if every directive in `body` uses a whitelisted verb."
  @spec all_verbs_whitelisted?(String.t()) :: boolean()
  def all_verbs_whitelisted?(body) do
    {:ok, directives} = parse(body)

    Enum.all?(directives, fn d ->
      d.verb == :wikilink or d.verb in @whitelist
    end)
  end

  @doc "The list of whitelisted verbs (for docs / schema enforcement)."
  @spec whitelist() :: [atom()]
  def whitelist, do: @whitelist

  @doc """
  Render a wiki body into one of the four output formats.

  Directives are resolved via the supplied `resolver` callback. If the
  resolver returns `{:error, _}` for a directive, the directive is
  rendered verbatim (with a footnote-style warning in markdown/plain
  formats) and the page-wide error list accumulates.

  Returns `{rendered, warnings}`.
  """
  @spec render(String.t(), resolver(), keyword()) :: {String.t(), [String.t()]}
  def render(body, resolver, opts \\ []) do
    format = Keyword.get(opts, :format, :markdown)
    {:ok, directives} = parse(body)

    {parts, citations, warnings, cursor} =
      Enum.reduce(directives, {[], [], [], 0}, fn d, {acc, cites, warns, cur} ->
        before = slice_bytes(body, cur, d.offset - cur)

        case resolver.(d, opts) do
          {:ok, rendered, meta} ->
            {cite_tag, new_cites} = maybe_record_citation(d, meta, cites, format)
            {[before, rendered, cite_tag | acc], new_cites, warns, d.offset + d.length}

          {:error, reason} ->
            fallback = render_fallback(d, reason, format)
            warn = "Directive #{inspect(d.raw)} failed: #{inspect(reason)}"
            {[before, fallback | acc], cites, [warn | warns], d.offset + d.length}
        end
      end)

    tail = slice_bytes(body, cursor, byte_size(body) - cursor)
    # `parts` was built with prepends; reverse once to get doc order.
    body_rendered = [Enum.reverse(parts), tail] |> IO.iodata_to_binary()

    rendered = apply_format(body_rendered, citations, format)
    {rendered, Enum.reverse(warnings)}
  end

  # ─── private ─────────────────────────────────────────────────────────────

  # Parse the argument part of a directive, e.g.
  #   "optimal://foo tier=l1 limit=5"
  # into {primary, %{"tier" => "l1", "limit" => "5"}}
  defp parse_argument(arg_str) do
    tokens = String.split(arg_str, ~r/\s+/, trim: true)

    {primary_tokens, kv_tokens} =
      Enum.split_with(tokens, fn t -> not String.contains?(t, "=") end)

    primary = Enum.join(primary_tokens, " ")

    options =
      Enum.reduce(kv_tokens, %{}, fn token, acc ->
        case String.split(token, "=", parts: 2) do
          [k, v] -> Map.put(acc, String.trim(k), unquote_string(String.trim(v)))
          _ -> acc
        end
      end)

    {unquote_string(primary), options}
  end

  defp unquote_string(v) do
    v
    |> String.trim_leading("\"")
    |> String.trim_trailing("\"")
    |> String.trim_leading("'")
    |> String.trim_trailing("'")
  end

  # Named `slice_bytes` so we don't shadow Kernel.binary_slice/3 (Elixir 1.14+).
  defp slice_bytes(binary, offset, length)
       when is_binary(binary) and is_integer(offset) and is_integer(length) do
    total = byte_size(binary)
    clamped_offset = max(min(offset, total), 0)
    clamped_length = max(min(length, total - clamped_offset), 0)
    :binary.part(binary, clamped_offset, clamped_length)
  end

  defp maybe_record_citation(%Directive{verb: :cite, argument: uri}, _meta, cites, :plain) do
    idx = length(cites) + 1
    {" [#{idx}]", cites ++ [{idx, uri}]}
  end

  defp maybe_record_citation(%Directive{verb: :cite, argument: uri}, _meta, cites, :markdown) do
    idx = length(cites) + 1
    {" [^#{idx}]", cites ++ [{idx, uri}]}
  end

  # Claude + openai don't inject inline markers but still need the citation
  # list so the per-format wrapper can emit <document>/role=system blocks.
  defp maybe_record_citation(%Directive{verb: :cite, argument: uri}, _meta, cites, :claude) do
    idx = length(cites) + 1
    {"", cites ++ [{idx, uri}]}
  end

  defp maybe_record_citation(%Directive{verb: :cite, argument: uri}, _meta, cites, :openai) do
    idx = length(cites) + 1
    {"", cites ++ [{idx, uri}]}
  end

  defp maybe_record_citation(_, _, cites, _format), do: {"", cites}

  defp render_fallback(%Directive{raw: raw}, _reason, :plain), do: "⟨#{raw}⟩"
  defp render_fallback(%Directive{raw: raw}, _reason, :markdown), do: "`#{raw}`"
  defp render_fallback(%Directive{raw: raw}, _reason, _other), do: raw

  # Plain / markdown have nothing to append when there are no citations.
  # Claude + openai still want their wrappers so downstream consumers get
  # predictable shapes.
  defp apply_format(body, [], :plain), do: body
  defp apply_format(body, [], :markdown), do: body

  defp apply_format(body, [], :claude), do: "<context>\n" <> body <> "\n</context>"

  defp apply_format(body, [], :openai) do
    Jason.encode!([%{role: "system", content: body}])
  end

  defp apply_format(body, [], _format), do: body

  defp apply_format(body, citations, :markdown) do
    footnotes =
      Enum.map_join(citations, "\n", fn {idx, uri} -> "[^#{idx}]: #{uri}" end)

    body <> "\n\n---\n\n" <> footnotes
  end

  defp apply_format(body, citations, :plain) do
    footnotes =
      Enum.map_join(citations, "\n", fn {idx, uri} -> "[#{idx}] #{uri}" end)

    body <> "\n\nSources:\n" <> footnotes
  end

  defp apply_format(body, citations, :claude) do
    docs =
      Enum.map_join(citations, "\n", fn {idx, uri} ->
        ~s(<document index="#{idx}" source="#{uri}"></document>)
      end)

    "<context>\n" <> body <> "\n\n" <> docs <> "\n</context>"
  end

  defp apply_format(body, citations, :openai) do
    sources =
      Enum.map(citations, fn {idx, uri} ->
        %{role: "system", content: "Source #{idx}: #{uri}"}
      end)

    messages = [%{role: "system", content: body} | sources]
    Jason.encode!(messages)
  end

  defp apply_format(body, _citations, _format), do: body
end
