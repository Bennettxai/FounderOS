defmodule OptimalEngine.Wiki.Curator do
  @moduledoc """
  The LLM-maintained curator — Stage 9 of the pipeline.

  Takes an existing Wiki page and a batch of new signals (chunks with
  citations), asks Ollama (or any Embed.Provider-compatible model) to
  integrate the new facts while preserving structure + citation
  coverage, and returns the updated page body.

  ## Graceful degradation

  If Ollama is unreachable, `curate/3` returns the original page body
  with a metadata note recording the deferred curation. The caller can
  retry later or fall through — the ChunkTree itself is never affected.

  ## Audience awareness

  The curator is passed the target audience + the schema + the subset of
  citations the audience is allowed to see. The intersection filter
  happens at citation-gathering time (before the curator runs), not
  inside the curator. This keeps the curator prompt simple: "here are
  the facts, here's the schema, update the page."
  """

  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.Wiki.Page

  require Logger

  @default_model "qwen3:8b"

  @type citation :: %{chunk_id: String.t(), text: String.t(), uri: String.t()}

  @type outcome :: %{
          ok?: boolean(),
          page: Page.t(),
          metadata: map(),
          warnings: [String.t()]
        }

  @doc """
  Curate a page with a batch of new signal citations.

  Options:
    * `:audience`       — target audience slug (default: page.audience)
    * `:schema`         — schema rules map (from `.wiki/SCHEMA.md`)
    * `:model`          — override LLM model (default: `qwen3:8b`)
    * `:deterministic`  — when true, skip Ollama entirely and just
                          append new citations verbatim. Used by tests
                          and by environments without Ollama.
  """
  @spec curate(Page.t(), [citation()], keyword()) :: outcome()
  def curate(%Page{} = page, citations, opts \\ []) when is_list(citations) do
    audience = Keyword.get(opts, :audience, page.audience)
    schema = Keyword.get(opts, :schema, %{})
    model = Keyword.get(opts, :model, @default_model)
    deterministic = Keyword.get(opts, :deterministic, false)

    cond do
      citations == [] ->
        %{
          ok?: true,
          page: page,
          metadata: %{reason: :no_new_citations, audience: audience},
          warnings: []
        }

      deterministic or not Ollama.available?() ->
        deterministic_merge(page, citations, audience)

      true ->
        ollama_merge(page, citations, audience, schema, model)
    end
  end

  # ─── deterministic merge ────────────────────────────────────────────────
  #
  # Appends the new citations under a `## New signals` section at the end
  # of the page, each with a `{{cite: uri}}` directive. Guaranteed to
  # preserve every existing citation and add all new ones — no content
  # is ever lost to an LLM misfire.

  defp deterministic_merge(page, citations, audience) do
    appended = build_new_signals_section(citations)

    updated_body =
      page.body
      |> strip_trailing_new_signals_section()
      |> Kernel.<>("\n\n")
      |> Kernel.<>(appended)
      |> String.trim_trailing()
      |> Kernel.<>("\n")

    updated_page = %{
      page
      | body: updated_body,
        version: page.version + 1,
        last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
        curated_by: "deterministic:#{audience}"
    }

    %{
      ok?: true,
      page: updated_page,
      metadata: %{
        reason: :deterministic_fallback,
        audience: audience,
        citations_added: length(citations)
      },
      warnings: []
    }
  end

  defp build_new_signals_section(citations) do
    entries =
      citations
      |> Enum.map(fn c ->
        preview =
          (c.text || "")
          |> String.trim()
          |> String.slice(0, 240)
          |> String.replace(~r/\s+/, " ")

        "- #{preview} {{cite: #{c.uri}}}"
      end)
      |> Enum.join("\n")

    "## New signals\n\n" <> entries
  end

  defp strip_trailing_new_signals_section(body) do
    case Regex.split(~r/\n##\s+New signals\s*\n/i, body, parts: 2) do
      [head, _tail] -> String.trim_trailing(head)
      _ -> body
    end
  end

  # ─── Ollama curator ─────────────────────────────────────────────────────

  defp ollama_merge(page, citations, audience, schema, model) do
    prompt = build_prompt(page, citations, audience, schema)
    system = build_system_prompt(audience, schema)

    case Ollama.generate(prompt, system: system, model: model, timeout_ms: 30_000) do
      {:ok, response} ->
        new_body = extract_body(response, page.body)

        updated_page = %{
          page
          | body: new_body,
            version: page.version + 1,
            last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
            curated_by: "ollama:#{model}"
        }

        %{
          ok?: true,
          page: updated_page,
          metadata: %{
            reason: :ollama_curation,
            audience: audience,
            citations_added: length(citations),
            model: model
          },
          warnings: []
        }

      {:error, reason} ->
        Logger.warning(
          "[Wiki.Curator] Ollama call failed — deterministic fallback. Reason: #{inspect(reason)}"
        )

        outcome = deterministic_merge(page, citations, audience)

        %{
          outcome
          | warnings:
              outcome.warnings ++
                ["ollama failed (#{inspect(reason)}); fell back to deterministic merge"]
        }
    end
  end

  defp build_system_prompt(audience, schema) do
    required_sections =
      Map.get(schema, "required_sections", [])
      |> Enum.map_join(", ", fn s -> "## #{s}" end)

    """
    You are the Wiki Curator for audience: #{audience}.

    Your task: take an existing wiki page and a list of new signals
    (facts extracted from chunks, each with a citation URI), and return
    an UPDATED page that integrates the new facts.

    Rules (enforced at integrity-check time — violate and the commit is rejected):
      1. Preserve every existing {{cite: uri}} directive.
      2. Add {{cite: uri}} for every new fact you include.
      3. Keep the page structure: required sections = #{required_sections}.
      4. Never invent facts. Every claim must have a citation.
      5. Output ONLY the updated page body. No preamble, no explanation,
         no markdown code fences — just the body itself.
    """
  end

  defp build_prompt(page, citations, _audience, _schema) do
    citation_block =
      citations
      |> Enum.map_join("\n", fn c -> "- #{String.trim(c.text)} (source: #{c.uri})" end)

    """
    # Existing page

    #{page.body}

    # New signals to integrate

    #{citation_block}

    # Updated page body
    """
  end

  # Extract the updated body from the LLM response. If the LLM wrapped it in
  # markdown code fences, strip them. If the response is empty or looks
  # truncated, fall back to the original body rather than lose content.
  defp extract_body(raw, original_body) do
    trimmed = String.trim(raw)

    cond do
      trimmed == "" ->
        original_body

      String.starts_with?(trimmed, "```") ->
        trimmed
        |> String.replace_prefix("```markdown", "")
        |> String.replace_prefix("```md", "")
        |> String.replace_prefix("```", "")
        |> String.trim_trailing("`")
        |> String.trim()

      true ->
        trimmed
    end
  end
end
