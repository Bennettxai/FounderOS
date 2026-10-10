defmodule OptimalEngine.Pipeline.Decomposer do
  @moduledoc """
  Stage 3 of the ingestion pipeline.

  Takes a `%ParsedDoc{}` (from Stage 2) and breaks it into a hierarchical
  `%ChunkTree{}` at four fixed scales:

      :document  (exactly 1)
        └── :section   (from parser-reported heading / page / slide boundaries)
             └── :paragraph  (from parser-reported paragraph elements,
                              or blank-line split when absent)
                  └── :chunk  (sliding window within each paragraph)

  ## Boundary respect

  The Decomposer NEVER splits across a parser-reported structural boundary.
  Paragraphs don't span sections. `:chunk`-scale windows don't span
  paragraphs. If a paragraph is shorter than the window size, it becomes a
  single chunk.

  ## Sliding window

  Configured via `:window_bytes` (default `2048`, ≈ 512 tokens assuming
  4 bytes/token) and `:overlap_bytes` (default `256`, ≈ 64 tokens).
  Applies inside each `:paragraph`-scale chunk's text. Small paragraphs
  produce a single chunk.

  ## Output shape

  `{:ok, %ChunkTree{root_chunk_id: doc_id, chunks: [...]}}` on success.
  Chunks are ordered by `scale` then `offset_bytes` for deterministic
  iteration.

  ## Persistence

  `decompose_and_store/2` invokes `Store.insert_chunks/1` after building
  the tree. The in-memory `decompose/2` variant is pure — useful for
  tests and for pipelines that want to inspect before persisting.
  """

  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}
  alias OptimalEngine.Pipeline.Parser.ParsedDoc
  alias OptimalEngine.Pipeline.Parser.StructuralElement

  @default_window_bytes 2_048
  @default_overlap_bytes 256
  @default_tenant "default"

  @doc """
  Decomposes a `%ParsedDoc{}` into a `%ChunkTree{}`. Pure — does not touch
  the store.

  Options:

    * `:tenant_id`          — defaults to the default tenant
    * `:window_bytes`       — :chunk scale window size in bytes (default 2048)
    * `:overlap_bytes`      — window overlap in bytes (default 256)
    * `:classification_level` — `"public" | "internal" | "confidential" | "restricted"` (default `"internal"`)
  """
  @spec decompose(ParsedDoc.t(), keyword()) :: {:ok, ChunkTree.t()}
  def decompose(%ParsedDoc{} = doc, opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, @default_tenant)
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    signal_id = doc.signal_id || derive_signal_id(doc.text)
    window = Keyword.get(opts, :window_bytes, @default_window_bytes)
    overlap = Keyword.get(opts, :overlap_bytes, @default_overlap_bytes)
    class_level = Keyword.get(opts, :classification_level, "internal")
    modality = doc.modality || :text

    document = build_document_chunk(doc, tenant_id, signal_id, modality, class_level)
    sections = build_section_chunks(doc, document, tenant_id, signal_id, modality, class_level)

    paragraphs =
      build_paragraph_chunks(doc, sections, document, tenant_id, signal_id, modality, class_level)

    chunks =
      build_chunk_level_chunks(
        doc,
        paragraphs,
        tenant_id,
        signal_id,
        modality,
        class_level,
        window,
        overlap
      )

    all =
      ([document] ++ sections ++ paragraphs ++ chunks)
      |> Enum.map(&%{&1 | workspace_id: workspace_id})

    {:ok, %ChunkTree{root_chunk_id: document.id, chunks: all}}
  end

  @doc """
  Decomposes + persists to the `chunks` table in one call. Idempotent on
  `chunk.id` thanks to the deterministic id scheme.
  """
  @spec decompose_and_store(ParsedDoc.t(), keyword()) ::
          {:ok, ChunkTree.t()} | {:error, term()}
  def decompose_and_store(%ParsedDoc{} = doc, opts \\ []) do
    with {:ok, tree} <- decompose(doc, opts),
         :ok <- OptimalEngine.Store.insert_chunks(tree.chunks) do
      maybe_embed_chunks(tree, opts)
      {:ok, tree}
    end
  end

  # Phase 5 connection: after chunks land in the store, embed them and persist
  # to `chunk_embeddings` so semantic search has vectors. Gated by config flag
  # (`config :optimal_engine, :embed, on_ingest: true`) and the per-call
  # `:skip_embed` opt. Failures (e.g. Ollama down) are non-fatal — chunks are
  # still stored; backfill via `mix optimal.embed_backfill` later.
  defp maybe_embed_chunks(tree, opts) do
    cond do
      Keyword.get(opts, :skip_embed, false) ->
        :ok

      not OptimalEngine.Pipeline.Embedder.embed_on_ingest?() ->
        :ok

      true ->
        case OptimalEngine.Pipeline.Embedder.embed_and_store(tree, opts) do
          {:ok, %{embedded: n, errors: errs}} ->
            require Logger
            Logger.debug("[Decomposer] embedded #{n} chunks, #{length(errs)} skipped")
            :ok

          {:error, reason} ->
            require Logger
            Logger.warning("[Decomposer] embed_and_store failed: #{inspect(reason)}")
            :ok
        end
    end
  end

  # ─── :document scale ─────────────────────────────────────────────────────

  defp build_document_chunk(doc, tenant_id, signal_id, modality, class_level) do
    Chunk.new(
      id: Chunk.build_id(signal_id, :document, 0),
      tenant_id: tenant_id,
      signal_id: signal_id,
      parent_id: nil,
      scale: :document,
      offset_bytes: 0,
      length_bytes: byte_size(doc.text),
      text: doc.text,
      modality: modality,
      asset_ref: first_asset_hash(doc),
      classification_level: class_level
    )
  end

  defp first_asset_hash(%ParsedDoc{assets: [%{hash: h} | _]}) when is_binary(h), do: h
  defp first_asset_hash(_), do: nil

  # ─── :section scale ──────────────────────────────────────────────────────
  #
  # A "section" is a stretch of text between two parser-reported
  # section-level boundaries: a heading, a page, or a slide. If the parser
  # reported none of those, the document itself is the only section — we
  # emit a single section chunk spanning the full text so the tree stays
  # uniform.

  @section_kinds [:heading, :page, :slide, :section]

  defp build_section_chunks(doc, document, tenant_id, signal_id, modality, class_level) do
    boundaries =
      doc.structure
      |> Enum.filter(&(&1.kind in @section_kinds))
      |> Enum.sort_by(& &1.offset)

    case boundaries do
      [] ->
        if byte_size(doc.text) == 0 do
          []
        else
          [
            section_from_span(
              document,
              0,
              byte_size(doc.text),
              doc.text,
              0,
              tenant_id,
              signal_id,
              modality,
              class_level,
              boundary: nil
            )
          ]
        end

      elements ->
        elements
        |> sections_from_boundaries(doc.text)
        |> Enum.with_index()
        |> Enum.map(fn {{offset, length, boundary}, idx} ->
          span_text = slice_bytes(doc.text, offset, length)

          section_from_span(
            document,
            offset,
            length,
            span_text,
            idx,
            tenant_id,
            signal_id,
            modality,
            class_level,
            boundary: boundary
          )
        end)
    end
  end

  # Convert a list of structural boundaries into [{offset, length, boundary_element}]
  # spanning the full text contiguously. Prepends a synthetic prologue span when
  # content exists before the first boundary.
  defp sections_from_boundaries(elements, text) do
    total = byte_size(text)

    {spans, last_end} =
      Enum.reduce(elements, {[], 0}, fn element, {acc, _prev_end} ->
        start = element.offset
        # Close the *previous* span at `start`; we'll emit the next one on the
        # next iteration. We track the current element's start as the begin of
        # its own span; length is computed on the next pass.
        acc_with_current = [{start, nil, element} | maybe_close_prev(acc, start)]
        {acc_with_current, start}
      end)

    # Close the final span at the end of text
    spans = close_trailing_span(spans, last_end, total)

    # Prepend a prologue span if content exists before the first boundary
    first_boundary_offset =
      case elements do
        [%StructuralElement{offset: o} | _] -> o
        _ -> total
      end

    spans =
      if first_boundary_offset > 0 do
        [{0, first_boundary_offset, nil} | Enum.reverse(spans)]
      else
        Enum.reverse(spans)
      end

    spans
  end

  defp maybe_close_prev([], _new_offset), do: []

  defp maybe_close_prev([{off, nil, element} | rest], new_offset) do
    [{off, max(new_offset - off, 0), element} | rest]
  end

  defp maybe_close_prev(spans, _new_offset), do: spans

  defp close_trailing_span([], _end, _total), do: []

  defp close_trailing_span([{off, nil, element} | rest], _end, total) do
    [{off, max(total - off, 0), element} | rest]
  end

  defp close_trailing_span(spans, _end, _total), do: spans

  defp section_from_span(
         document,
         offset,
         length,
         text,
         idx,
         tenant_id,
         signal_id,
         modality,
         class_level,
         boundary: boundary
       ) do
    Chunk.new(
      id: Chunk.build_id(signal_id, :section, idx),
      tenant_id: tenant_id,
      signal_id: signal_id,
      parent_id: document.id,
      scale: :section,
      offset_bytes: offset,
      length_bytes: length,
      text: text,
      modality: modality,
      classification_level: class_level,
      asset_ref: boundary_asset_ref(boundary)
    )
  end

  defp boundary_asset_ref(nil), do: nil

  defp boundary_asset_ref(%StructuralElement{metadata: %{asset_ref: ref}}) when is_binary(ref),
    do: ref

  defp boundary_asset_ref(_), do: nil

  # ─── :paragraph scale ────────────────────────────────────────────────────
  #
  # Use parser-reported `:paragraph` elements when present. Otherwise split
  # each section's text on blank-line boundaries as a fallback — this keeps
  # the paragraph scale populated even for formats (PDF, HTML-stripped text)
  # that didn't surface paragraph structure explicitly.

  defp build_paragraph_chunks(doc, sections, document, tenant_id, signal_id, modality, class_level) do
    paragraph_elements = Enum.filter(doc.structure, &(&1.kind == :paragraph))

    case paragraph_elements do
      [] ->
        fallback_paragraphs(
          sections,
          document,
          doc.text,
          tenant_id,
          signal_id,
          modality,
          class_level
        )

      elements ->
        paragraphs_from_elements(
          elements,
          sections,
          document,
          tenant_id,
          signal_id,
          modality,
          class_level,
          doc.text
        )
    end
  end

  defp paragraphs_from_elements(
         elements,
         sections,
         document,
         tenant_id,
         signal_id,
         modality,
         class_level,
         text
       ) do
    elements
    |> Enum.sort_by(& &1.offset)
    |> Enum.with_index()
    |> Enum.map(fn {element, idx} ->
      length =
        cond do
          element.length > 0 -> element.length
          element.text != "" -> byte_size(element.text)
          true -> 0
        end

      text_span =
        cond do
          element.text != "" -> element.text
          element.offset >= 0 and length > 0 -> slice_bytes(text, element.offset, length)
          true -> ""
        end

      parent = containing_section(sections, document, element.offset)

      Chunk.new(
        id: Chunk.build_id(signal_id, :paragraph, idx),
        tenant_id: tenant_id,
        signal_id: signal_id,
        parent_id: parent.id,
        scale: :paragraph,
        offset_bytes: element.offset,
        length_bytes: byte_size(text_span),
        text: text_span,
        modality: modality,
        classification_level: class_level
      )
    end)
  end

  defp fallback_paragraphs(
         sections,
         document,
         full_text,
         tenant_id,
         signal_id,
         modality,
         class_level
       ) do
    targets =
      case sections do
        [] when byte_size(full_text) > 0 ->
          [{document, 0, byte_size(full_text), full_text}]

        [] ->
          []

        secs ->
          Enum.map(secs, fn sec -> {sec, sec.offset_bytes, sec.length_bytes, sec.text} end)
      end

    {paragraphs, _counter} =
      Enum.reduce(targets, {[], 0}, fn {parent, base_offset, _length, section_text},
                                       {acc, counter} ->
        split_paragraphs(section_text, base_offset, counter)
        |> Enum.reduce({acc, counter}, fn {off, len, txt}, {inner_acc, inner_counter} ->
          chunk =
            Chunk.new(
              id: Chunk.build_id(signal_id, :paragraph, inner_counter),
              tenant_id: tenant_id,
              signal_id: signal_id,
              parent_id: parent.id,
              scale: :paragraph,
              offset_bytes: off,
              length_bytes: len,
              text: txt,
              modality: modality,
              classification_level: class_level
            )

          {[chunk | inner_acc], inner_counter + 1}
        end)
      end)

    Enum.reverse(paragraphs)
  end

  # Splits `text` on blank-line boundaries, returning `[{absolute_offset, length, text}]`
  # with offsets normalized to `base_offset + in_section_offset`.
  defp split_paragraphs(text, base_offset, _start_counter) do
    text
    |> String.split(~r/\n{2,}/, trim: false)
    |> Enum.reduce({[], 0}, fn chunk, {acc, cursor} ->
      trimmed_len = byte_size(chunk)

      if String.trim(chunk) == "" do
        {acc, cursor + trimmed_len + 2}
      else
        {[{base_offset + cursor, trimmed_len, chunk} | acc], cursor + trimmed_len + 2}
      end
    end)
    |> elem(0)
    |> Enum.reverse()
  end

  defp containing_section([], document, _offset), do: document

  defp containing_section(sections, _document, offset) do
    Enum.find(sections, hd(sections), fn s ->
      offset >= s.offset_bytes and offset < s.offset_bytes + s.length_bytes
    end)
  end

  # ─── :chunk scale ────────────────────────────────────────────────────────
  #
  # Sliding window within each paragraph. Never spans a paragraph boundary.
  # Paragraphs shorter than the window size produce exactly one chunk.

  defp build_chunk_level_chunks(
         _doc,
         paragraphs,
         tenant_id,
         signal_id,
         modality,
         class_level,
         window,
         overlap
       ) do
    step = max(window - overlap, 1)

    paragraphs
    |> Enum.flat_map(fn para ->
      para
      |> slide_windows(window, step)
      |> Enum.map(fn {offset_in_para, text} ->
        abs_offset = para.offset_bytes + offset_in_para

        %{parent: para, offset: abs_offset, text: text}
      end)
    end)
    |> Enum.with_index()
    |> Enum.map(fn {%{parent: parent, offset: offset, text: text}, idx} ->
      Chunk.new(
        id: Chunk.build_id(signal_id, :chunk, idx),
        tenant_id: tenant_id,
        signal_id: signal_id,
        parent_id: parent.id,
        scale: :chunk,
        offset_bytes: offset,
        length_bytes: byte_size(text),
        text: text,
        modality: modality,
        classification_level: class_level
      )
    end)
  end

  defp slide_windows(%Chunk{text: text}, window, _step) when byte_size(text) <= window,
    do: [{0, text}]

  defp slide_windows(%Chunk{text: text}, window, step) do
    total = byte_size(text)
    slide_windows_bytes(text, 0, total, window, step, [])
  end

  defp slide_windows_bytes(_text, offset, total, _window, _step, acc) when offset >= total,
    do: Enum.reverse(acc)

  defp slide_windows_bytes(text, offset, total, window, step, acc) do
    take = min(window, total - offset)
    slice = slice_bytes(text, offset, take)
    next_offset = offset + step
    new_acc = [{offset, slice} | acc]

    # Stop once we've reached the end
    if offset + take >= total do
      Enum.reverse(new_acc)
    else
      slide_windows_bytes(text, next_offset, total, window, step, new_acc)
    end
  end

  # ─── helpers ─────────────────────────────────────────────────────────────

  defp derive_signal_id(text) when is_binary(text) do
    "sha256:" <> (:crypto.hash(:sha256, text) |> Base.encode16(case: :lower))
  end

  # Safe binary slice that clamps to the string length. Works around Elixir's
  # `String.slice` being codepoint-aware; we want raw bytes here since offsets
  # are byte offsets from the parser.
  defp slice_bytes(binary, offset, length)
       when is_binary(binary) and is_integer(offset) and is_integer(length) do
    total = byte_size(binary)
    clamped_offset = max(min(offset, total), 0)
    clamped_length = max(min(length, total - clamped_offset), 0)
    :binary.part(binary, clamped_offset, clamped_length)
  end
end
