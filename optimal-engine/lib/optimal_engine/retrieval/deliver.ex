defmodule OptimalEngine.Retrieval.Deliver do
  @moduledoc """
  Final encode step of the RAG flow.

  Given a composed payload (either a wiki page or a packed chunk list)
  and a `Receiver`, produce the exact bytes the receiver expects:

    * `:plain`    — naked text with numeric `[n]` citations + Sources footer
    * `:markdown` — footnote-style `[^n]` citations + horizontal-rule block
    * `:claude`   — XML `<context>`…`<document index=… source=…>` wrappers
    * `:openai`   — JSON message array `[%{role: "system", content: …}, …]`

  The heavy directive resolution lives in `OptimalEngine.Wiki.Directives`.
  This module reuses it for the wiki path so a curated page renders
  consistently whether the consumer is a CLI, an agent, or a tool call.

  For the chunk path (no directives), we synthesize a pseudo-wiki body
  with `{{cite: uri}}` markers per chunk and pipe it through the same
  renderer — one code path, one output shape.
  """

  alias OptimalEngine.Retrieval.Receiver
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Directives
  alias OptimalEngine.Wiki.Page

  @type source :: %{required(:uri) => String.t(), optional(any()) => any()}

  @type envelope :: %{
          body: String.t(),
          format: Receiver.format(),
          sources: [String.t()],
          warnings: [String.t()]
        }

  @doc """
  Render a curated wiki `page` for `receiver`.

  The supplied `resolver` is called for every non-cite directive; the
  caller must provide one (usually `OptimalEngine.Retrieval.RAG` wires
  a resolver that hits the store).
  """
  @spec render_wiki(Page.t(), Receiver.t(), Directives.resolver()) :: envelope()
  def render_wiki(%Page{} = page, %Receiver{format: format}, resolver) do
    {body, warnings} = Wiki.render(page, resolver, format: format)
    sources = collect_cited_uris(page.body)

    %{body: body, format: format, sources: sources, warnings: warnings}
  end

  @doc """
  Render a list of packed chunks as a cited envelope.

  Each chunk becomes a labelled section with a `{{cite: uri}}`
  directive appended; the wiki directive renderer then produces
  citations consistent with the wiki path.

  Chunks without a `:uri` are emitted inline without citation.
  """
  @spec render_chunks([map()], Receiver.t()) :: envelope()
  def render_chunks(chunks, %Receiver{format: format}) when is_list(chunks) do
    body = compose_chunk_body(chunks)
    resolver = fn _d, _opts -> {:ok, "", %{}} end

    {rendered, warnings} = Directives.render(body, resolver, format: format)

    sources =
      chunks
      |> Enum.map(&Map.get(&1, :uri))
      |> Enum.reject(&is_nil/1)
      |> Enum.uniq()

    %{body: rendered, format: format, sources: sources, warnings: warnings}
  end

  @doc """
  Convenience: render a plain error/empty envelope in the receiver's
  format. Used when a retrieval yields no hits.
  """
  @spec empty(Receiver.t(), String.t()) :: envelope()
  def empty(%Receiver{format: format}, message \\ "No results.") do
    body =
      case format do
        :plain -> message
        :markdown -> "_#{message}_"
        :claude -> "<context>\n#{message}\n</context>"
        :openai -> Jason.encode!([%{role: "system", content: message}])
      end

    %{body: body, format: format, sources: [], warnings: []}
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp compose_chunk_body(chunks) do
    chunks
    |> Enum.with_index(1)
    |> Enum.map_join("\n\n", fn {chunk, idx} ->
      title = Map.get(chunk, :title, "Chunk #{idx}")
      content = Map.get(chunk, :content, "") |> to_string() |> String.trim()
      uri = Map.get(chunk, :uri)

      header = "### #{title}"

      citation =
        case uri do
          nil -> ""
          "" -> ""
          u -> " {{cite: #{u}}}"
        end

      "#{header}#{citation}\n\n#{content}"
    end)
  end

  # Pull `{{cite: …}}` URIs straight out of a wiki body so the Deliver
  # contract (`%{sources: […]}`) is populated even when the format
  # wrapper hides the inline markers (claude/openai).
  defp collect_cited_uris(body) when is_binary(body) do
    Regex.scan(~r/\{\{\s*cite\s*:\s*([^\}\s]+)/, body, capture: :all_but_first)
    |> Enum.map(fn [uri] -> uri end)
    |> Enum.uniq()
  end
end
