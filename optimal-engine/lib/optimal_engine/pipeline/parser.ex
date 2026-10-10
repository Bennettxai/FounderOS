defmodule OptimalEngine.Pipeline.Parser do
  @moduledoc """
  Stage 2 of the ingestion pipeline.

  Dispatches a filesystem path (or raw text with a format hint) to the right
  backend and returns a canonical `%ParsedDoc{}`. Every backend implements
  the `Parser.Backend` behaviour:

      @callback parse(path_or_binary, opts) :: {:ok, ParsedDoc.t()} | {:error, term()}

  ## Adding a new format

  1. Create `lib/optimal_engine/pipeline/parser/<format>.ex`.
  2. Implement `@behaviour OptimalEngine.Pipeline.Parser.Backend`.
  3. Register the extension(s) in `@backends` below.
  4. Add a fixture + test at `test/pipeline/parser/<format>_test.exs`.

  ## Graceful degradation

  Backends that depend on external executables (pdftotext, tesseract,
  whisper.cpp, ffmpeg) MUST NOT crash when the tool is missing. Instead they
  return a `%ParsedDoc{}` with an empty `text`, a `warnings` entry explaining
  the missing dependency, and as much metadata as they can glean without
  running the tool (file size, detected content type, etc.).

  Phase 3 (Decomposer) will still produce a `:document`-scale chunk from the
  warning-only doc, which keeps the rest of the pipeline running end-to-end
  in environments without the shell tools installed.
  """

  alias OptimalEngine.Pipeline.Parser.ParsedDoc

  # Extension → backend module. Ordered roughly by frequency of use.
  @backends [
    # Text-based
    {~w(.md), OptimalEngine.Pipeline.Parser.Markdown},
    {~w(.txt .rst .adoc .log), OptimalEngine.Pipeline.Parser.Text},
    {~w(.yaml .yml .toml), OptimalEngine.Pipeline.Parser.Yaml},
    {~w(.json), OptimalEngine.Pipeline.Parser.Json},
    {~w(.csv .tsv), OptimalEngine.Pipeline.Parser.Csv},
    {~w(.html .htm .xhtml), OptimalEngine.Pipeline.Parser.Html},
    # Source code
    {~w(.ex .exs .py .js .ts .jsx .tsx .go .rs .rb .java .c .cpp .cc .hpp .h
        .cs .kt .swift .php .sh .bash .zsh .lua .pl .sql .scala .clj .hs .ml
        .fs .dart .nim .zig .vim .el), OptimalEngine.Pipeline.Parser.Code},
    # Office
    {~w(.docx .pptx .xlsx), OptimalEngine.Pipeline.Parser.Office},
    # Binary — shell-tool-dependent with graceful degradation
    {~w(.pdf), OptimalEngine.Pipeline.Parser.Pdf},
    {~w(.png .jpg .jpeg .gif .webp .svg), OptimalEngine.Pipeline.Parser.Image},
    {~w(.mp3 .wav .m4a .ogg .flac), OptimalEngine.Pipeline.Parser.Audio},
    {~w(.mp4 .mov .webm .mkv), OptimalEngine.Pipeline.Parser.Video}
  ]

  @doc """
  Parse a file at `path`, dispatching to the right backend.

  Returns `{:ok, %ParsedDoc{}}` on success, `{:error, reason}` on unrecoverable
  failure. Missing shell tools do NOT produce errors — they produce a
  `%ParsedDoc{}` with warnings.
  """
  @spec parse(String.t(), keyword()) :: {:ok, ParsedDoc.t()} | {:error, term()}
  def parse(path, opts \\ []) when is_binary(path) do
    case dispatch(path) do
      {:ok, backend} ->
        backend.parse(path, opts)

      :unknown ->
        # Unknown extension → fall back to plain text reader. If the file
        # reads as UTF-8, great; otherwise the Text backend will note the
        # issue in warnings.
        OptimalEngine.Pipeline.Parser.Text.parse(path, opts)
    end
  end

  @doc """
  Parse inline text with an explicit format hint (useful for piped signals
  that never touch the filesystem).

      Parser.parse_text("# Hello", format: :markdown)
  """
  @spec parse_text(String.t(), keyword()) :: {:ok, ParsedDoc.t()} | {:error, term()}
  def parse_text(text, opts \\ []) when is_binary(text) do
    format = Keyword.get(opts, :format, :text)

    backend =
      case format do
        :markdown -> OptimalEngine.Pipeline.Parser.Markdown
        :md -> OptimalEngine.Pipeline.Parser.Markdown
        :yaml -> OptimalEngine.Pipeline.Parser.Yaml
        :yml -> OptimalEngine.Pipeline.Parser.Yaml
        :json -> OptimalEngine.Pipeline.Parser.Json
        :csv -> OptimalEngine.Pipeline.Parser.Csv
        :html -> OptimalEngine.Pipeline.Parser.Html
        :code -> OptimalEngine.Pipeline.Parser.Code
        _ -> OptimalEngine.Pipeline.Parser.Text
      end

    backend.parse_text(text, opts)
  end

  @doc "Returns the backend module that will handle `path`, or `:unknown`."
  @spec dispatch(String.t()) :: {:ok, module()} | :unknown
  def dispatch(path) when is_binary(path) do
    ext = path |> Path.extname() |> String.downcase()

    Enum.find_value(@backends, :unknown, fn {exts, mod} ->
      if ext in exts, do: {:ok, mod}, else: nil
    end)
  end

  @doc "Returns the full registry of `{extensions, backend_module}` tuples."
  @spec backends() :: [{[String.t()], module()}]
  def backends, do: @backends
end
