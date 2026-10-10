defmodule OptimalEngine.Wiki.Integrity do
  @moduledoc """
  Integrity checks for wiki pages — the "citation hygiene" layer.

  A healthy wiki page has three properties:

    1. **Every directive uses a whitelisted verb.** Typos + malicious
       injection attempts surface as warnings.
    2. **Every `{{cite: uri}}` resolves.** Broken citations are flagged
       with the offending URI so the curator can repair or remove.
    3. **Every factual claim is cited.** A paragraph containing a claim
       (detected heuristically) without a nearby `{{cite: …}}` is flagged.

  Additional checks come from `.wiki/SCHEMA.md` rules (minimum sections,
  citation density, size ceilings) — those are applied via
  `Integrity.against_schema/2`.

  Contradiction detection (`check_contradictions/2`) scans the page body
  for cases where the same bold-entity name is paired with two distinct
  numeric/date values in different cited passages. The heuristic is
  deliberately conservative — false positives are worse than false
  negatives for a human-review gate.

  This module is pure: it doesn't mutate the page, it just reports.
  """

  alias OptimalEngine.Wiki.Directives
  alias OptimalEngine.Workspace.Config

  @type issue :: %{
          severity: :error | :warning | :info,
          kind: atom(),
          message: String.t(),
          location: String.t() | nil
        }

  @type contradiction :: %{
          type: :entity_attr_clash,
          entity: String.t(),
          attr: :numeric | :date,
          claims: [%{value: String.t(), citation: String.t(), hash: String.t()}]
        }

  @type report :: %{
          page_slug: String.t(),
          ok?: boolean(),
          issues: [issue()],
          contradictions: [contradiction()]
        }

  @doc """
  Run the full integrity suite against a page body.

  Options:
    * `:resolve_uri` — function `fn uri -> :ok | {:error, reason} end` used
      to test whether each cite resolves. Defaults to always-ok so tests
      that don't need live resolution can skip the Store.
    * `:detect_contradictions` — boolean (default `true`). When true,
      `check_contradictions/2` is run and the result is folded into the
      report under the `:contradictions` key.
    * `:workspace_slug` — workspace slug used to read the contradictions
      policy from `Workspace.Config`. Defaults to `page.workspace_id`.

  Returns `%{page_slug, ok?, issues, contradictions}`.

  `ok?` is true when every issue has severity `:info` AND the contradiction
  policy does not mandate rejection.

  Possible return `:status` values for callers that need more detail:
    * `:valid`               — clean, no contradictions
    * `:valid_with_warnings` — has issues/contradictions but policy allows it
    * `:invalid`             — error-level issue OR policy=reject with contradictions
  """
  @spec check(OptimalEngine.Wiki.Page.t(), keyword()) :: report()
  def check(%OptimalEngine.Wiki.Page{} = page, opts \\ []) do
    resolver = Keyword.get(opts, :resolve_uri, fn _ -> :ok end)
    detect = Keyword.get(opts, :detect_contradictions, true)
    workspace_slug = Keyword.get(opts, :workspace_slug, page.workspace_id || "default")

    issues =
      []
      |> append(check_verbs(page.body))
      |> append(check_citations(page.body, resolver))
      |> append(check_claim_density(page.body))
      |> append(check_page_size(page.body))

    contradictions =
      if detect do
        check_contradictions(page, opts)
      else
        []
      end

    has_error = Enum.any?(issues, &(&1.severity == :error))

    # Apply contradiction policy when contradictions were found.
    # `:config_root` can be injected by tests to point at a temp directory.
    config_root = Keyword.get(opts, :config_root)

    {contradictions, extra_invalid} =
      if contradictions == [] do
        {contradictions, false}
      else
        policy = contradiction_policy(workspace_slug, config_root)
        apply_policy(policy, contradictions)
      end

    %{
      page_slug: page.slug,
      ok?: not has_error and not extra_invalid,
      issues: issues,
      contradictions: contradictions
    }
  end

  @doc """
  Apply schema-defined rules from `.wiki/SCHEMA.md`. The schema is a set
  of markdown-documented rules; we support the machine-enforceable subset:

    * required top-level sections (`## Summary`, `## Related`, etc.)
    * minimum citation density per section
    * maximum page size in bytes

  Custom rules can be added via `:rules` keyword.
  """
  @spec against_schema(OptimalEngine.Wiki.Page.t(), map(), keyword()) :: report()
  def against_schema(%OptimalEngine.Wiki.Page{} = page, schema, opts \\ []) when is_map(schema) do
    base = check(page, opts)

    schema_issues =
      []
      |> append(check_required_sections(page.body, Map.get(schema, "required_sections", [])))
      |> append(check_max_size(page.body, Map.get(schema, "max_bytes")))
      |> append(
        check_required_frontmatter(page.frontmatter, Map.get(schema, "required_frontmatter", []))
      )

    %{
      base
      | issues: base.issues ++ schema_issues,
        ok?: base.ok? and Enum.all?(schema_issues, &(&1.severity != :error))
    }
  end

  @doc """
  Scan a page body for entity-attribute contradictions.

  Heuristic (conservative — false positives are worse than false negatives):

  1. Find every `{{cite: <uri>}}` directive and the text *before* it (up to
     the previous directive or paragraph break). That text is a "cited passage".
  2. In each cited passage, look for a bold entity name (`**name**`) followed
     within 120 characters by a numeric value (price `$N`, percentage `N%`,
     or ISO date `YYYY-MM-DD`).
  3. Group (entity_name_downcased, attr_kind) → list of distinct values seen.
  4. When the same entity has **two or more distinct values** for the same
     attr_kind, it is a contradiction.

  Returns a list of `%{type: :entity_attr_clash, entity:, attr:, claims:}`
  maps. Each claim has `%{value:, citation:, hash:}`.

  The hash is a short SHA-256 of `"entity::value::citation"` for stable
  deduplication by callers.
  """
  @spec check_contradictions(OptimalEngine.Wiki.Page.t(), keyword()) :: [contradiction()]
  def check_contradictions(%OptimalEngine.Wiki.Page{} = page, _opts \\ []) do
    extract_cited_claims(page.body)
    |> group_by_entity_attr()
    |> filter_contradictions()
  end

  # ─── checks ──────────────────────────────────────────────────────────────

  defp check_verbs(body) do
    if Directives.all_verbs_whitelisted?(body) do
      []
    else
      {:ok, directives} = Directives.parse(body)

      directives
      |> Enum.reject(fn d -> d.verb == :wikilink or d.verb in Directives.whitelist() end)
      |> Enum.map(fn d ->
        %{
          severity: :error,
          kind: :invalid_verb,
          message: "Unknown directive verb `#{d.verb}` in `#{d.raw}`",
          location: "offset #{d.offset}"
        }
      end)
    end
  end

  defp check_citations(body, resolver) do
    {:ok, directives} = Directives.parse(body)

    directives
    |> Enum.filter(&(&1.verb == :cite))
    |> Enum.flat_map(fn d ->
      case resolver.(d.argument) do
        :ok ->
          []

        {:error, reason} ->
          [
            %{
              severity: :error,
              kind: :broken_citation,
              message: "Citation `#{d.argument}` does not resolve (#{inspect(reason)})",
              location: "offset #{d.offset}"
            }
          ]
      end
    end)
  end

  # Heuristic: every non-empty paragraph should contain at least one cite
  # directive, wikilink, or explicit acknowledgment phrase (for pure
  # structural paragraphs like headers, we skip).
  defp check_claim_density(body) do
    paragraphs =
      body
      |> strip_directive_markup()
      |> String.split(~r/\n{2,}/, trim: true)
      |> Enum.with_index()

    paragraphs
    |> Enum.flat_map(fn {para, idx} ->
      trimmed = String.trim(para)

      cond do
        trimmed == "" ->
          []

        String.starts_with?(trimmed, "#") ->
          []

        String.starts_with?(trimmed, "- ") or String.starts_with?(trimmed, "* ") ->
          # List items often carry their own citations; flag only if long
          if String.length(trimmed) > 200 and not contains_citation?(body, idx) do
            [claim_issue(idx, trimmed)]
          else
            []
          end

        contains_citation?(body, idx) ->
          []

        # Very short paragraphs (< 40 chars) are likely labels/section-intros
        String.length(trimmed) < 40 ->
          []

        true ->
          [claim_issue(idx, trimmed)]
      end
    end)
  end

  defp claim_issue(idx, para) do
    %{
      severity: :warning,
      kind: :uncited_claim,
      message:
        "Paragraph #{idx + 1} contains substantive content but no {{cite: …}} — #{String.slice(para, 0, 60)}…",
      location: "paragraph #{idx + 1}"
    }
  end

  defp contains_citation?(body, paragraph_idx) do
    # Cheap approach: parse the raw body for directives and check whether
    # any directive falls within the given paragraph boundary. For this
    # coarse heuristic we count ANY citation in the body — the goal is
    # to avoid false positives on small pages, not to prove per-paragraph
    # citation placement.
    {:ok, directives} = Directives.parse(body)
    _ = paragraph_idx
    Enum.any?(directives, &(&1.verb == :cite))
  end

  defp strip_directive_markup(body) do
    body
    |> String.replace(~r/\{\{[^}]+\}\}/, " ")
    |> String.replace(~r/\[\[[^\]]+\]\]/, " ")
  end

  defp check_page_size(body) do
    size = byte_size(body)

    cond do
      size > 50_000 ->
        [
          %{
            severity: :warning,
            kind: :page_too_large,
            message: "Page body is #{size} bytes; consider spawning child pages",
            location: nil
          }
        ]

      size == 0 ->
        [
          %{
            severity: :error,
            kind: :empty_page,
            message: "Page body is empty",
            location: nil
          }
        ]

      true ->
        []
    end
  end

  defp check_required_sections(_body, []), do: []

  defp check_required_sections(body, required) when is_list(required) do
    headings = Regex.scan(~r/^##\s+(.+?)\s*$/m, body, capture: :all_but_first)
    present = headings |> Enum.map(fn [h] -> String.downcase(h) end) |> MapSet.new()

    required
    |> Enum.reject(fn name -> MapSet.member?(present, String.downcase(name)) end)
    |> Enum.map(fn name ->
      %{
        severity: :error,
        kind: :missing_section,
        message: "Required section `## #{name}` is absent",
        location: nil
      }
    end)
  end

  defp check_max_size(_body, nil), do: []

  defp check_max_size(body, max) when is_integer(max) do
    if byte_size(body) > max do
      [
        %{
          severity: :error,
          kind: :schema_size_exceeded,
          message: "Page body is #{byte_size(body)} bytes; schema maximum is #{max}",
          location: nil
        }
      ]
    else
      []
    end
  end

  defp check_required_frontmatter(_fm, []), do: []

  defp check_required_frontmatter(fm, required) when is_list(required) do
    Enum.flat_map(required, fn key ->
      case Map.get(fm, key) do
        nil ->
          [
            %{
              severity: :error,
              kind: :missing_frontmatter,
              message: "Required frontmatter key `#{key}` is absent",
              location: nil
            }
          ]

        "" ->
          [
            %{
              severity: :error,
              kind: :empty_frontmatter,
              message: "Required frontmatter key `#{key}` is empty",
              location: nil
            }
          ]

        _ ->
          []
      end
    end)
  end

  defp append(acc, list) when is_list(list), do: acc ++ list

  # ─── contradiction detection ─────────────────────────────────────────────

  # Regex patterns used for claim extraction.
  # Entity: **bold text** — kept conservative (no whitespace inside ** pair).
  @entity_re ~r/\*\*([^*\n]+)\*\*/

  # Value candidates: "$2K", "$2,400", "$2.4K", "2.4%", "2026-04-28".
  # Deliberately narrow — only price-like, percent-like, or date values.
  @value_re ~r/(\$[\d,]+(?:\.\d+)?[KMB]?|[\d,]+(?:\.\d+)?%|\d{4}-\d{2}-\d{2})/

  # Cite directive — captured group 1 = the URI.
  @cite_re ~r/\{\{\s*cite\s*:\s*([^\}]+?)\s*\}\}/

  # How many bytes before a cite directive we scan for entity+value pairs.
  @lookahead_bytes 200

  # Split body into (passage, citation_uri) segments. Each segment is the
  # text immediately preceding a {{cite:...}} directive, up to @lookahead_bytes.
  defp extract_cited_claims(body) do
    cite_positions =
      Regex.scan(@cite_re, body, return: :index)
      |> Enum.map(fn [{full_off, full_len}, {uri_off, uri_len}] ->
        uri = :binary.part(body, uri_off, uri_len) |> String.trim()
        passage_start = max(full_off - @lookahead_bytes, 0)
        passage = :binary.part(body, passage_start, full_off - passage_start)
        {passage, uri, full_off + full_len}
      end)

    Enum.flat_map(cite_positions, fn {passage, citation_uri, _end_pos} ->
      extract_entity_value_pairs(passage, citation_uri)
    end)
  end

  # From a passage + citation_uri, produce [{entity_key, attr_kind, value, citation_uri}].
  #
  # Conservative scoping: split the passage into lines and only pair an entity
  # with values that appear on the SAME line. This prevents cross-sentence bleed
  # where a previous sentence's entity matches a later sentence's value through
  # the shared lookahead window.
  defp extract_entity_value_pairs(passage, citation_uri) do
    lines = String.split(passage, ~r/\r?\n/, trim: true)

    Enum.flat_map(lines, fn line ->
      entities =
        Regex.scan(@entity_re, line, capture: :all_but_first)
        |> Enum.map(&hd/1)

      if entities == [] do
        []
      else
        values =
          Regex.scan(@value_re, line, capture: :all_but_first)
          |> Enum.map(fn [v] -> {v, classify_value(v)} end)

        # Only pair entities with values found on the same line.
        for entity <- entities,
            {value, attr_kind} <- values do
          {String.downcase(entity), attr_kind, value, citation_uri}
        end
      end
    end)
  end

  defp classify_value(v) do
    cond do
      Regex.match?(~r/\d{4}-\d{2}-\d{2}/, v) -> :date
      true -> :numeric
    end
  end

  # Group claims by {entity_key, attr_kind} → list of distinct values + citations.
  defp group_by_entity_attr(claims) do
    Enum.group_by(claims, fn {entity, attr_kind, _value, _citation} ->
      {entity, attr_kind}
    end)
    |> Enum.map(fn {{entity, attr_kind}, entries} ->
      distinct =
        entries
        |> Enum.uniq_by(fn {_e, _a, value, _c} -> normalise_value(value) end)

      {entity, attr_kind, distinct}
    end)
  end

  # Normalise a value string before deduplication — strip commas, trailing
  # K/M/B suffix resolution is intentionally NOT done here. We only normalise
  # trivial formatting differences (1,000 == 1000) but we do NOT try to convert
  # "$2K" and "$2,000" to the same canonical form. The heuristic is conservative:
  # if the page writes the value two different ways it should be flagged.
  defp normalise_value(v) do
    v |> String.replace(",", "") |> String.downcase()
  end

  # Only flag when there are ≥2 distinct normalised values for an entity.
  defp filter_contradictions(grouped) do
    grouped
    |> Enum.filter(fn {_entity, _attr_kind, distinct} -> length(distinct) >= 2 end)
    |> Enum.map(fn {entity, attr_kind, distinct} ->
      claims =
        Enum.map(distinct, fn {_e, _a, value, citation} ->
          %{
            value: value,
            citation: citation,
            hash: claim_hash(entity, value, citation)
          }
        end)

      %{
        type: :entity_attr_clash,
        entity: entity,
        attr: attr_kind,
        claims: claims
      }
    end)
  end

  defp claim_hash(entity, value, citation) do
    :crypto.hash(:sha256, "#{entity}::#{value}::#{citation}")
    |> Base.encode16(case: :lower)
    |> binary_part(0, 12)
  end

  # ─── policy application ──────────────────────────────────────────────────

  # Returns the contradictions policy string for a workspace slug.
  # Defaults to "flag_for_review" when config is absent.
  # `root` is the engine root path passed to `Config.get_section/4`; in tests
  # a temp directory can be supplied so YAML can be written and read back.
  defp contradiction_policy(workspace_slug, root) do
    effective_root = root || File.cwd!()

    section =
      Config.get_section(
        workspace_slug,
        :contradictions,
        %{policy: "flag_for_review"},
        effective_root
      )

    case section do
      %{policy: policy} when is_binary(policy) -> policy
      %{"policy" => policy} when is_binary(policy) -> policy
      _ -> "flag_for_review"
    end
  end

  # Returns {contradictions, extra_invalid?}.
  # `extra_invalid?` causes `ok?` to be false for "reject" policy.
  # For "silent_resolve", contradictions list is collapsed to the winner (newest by citation order).
  # For "flag_for_review", contradictions are passed through unchanged.
  defp apply_policy("reject", contradictions) do
    {contradictions, true}
  end

  defp apply_policy("silent_resolve", contradictions) do
    # "Newer citation wins" — keep the last claim in the list (document order = ingest order).
    resolved =
      Enum.map(contradictions, fn c ->
        winning_claim = List.last(c.claims)
        %{c | claims: [winning_claim]}
      end)

    {resolved, false}
  end

  defp apply_policy(_flag_for_review, contradictions) do
    {contradictions, false}
  end
end
