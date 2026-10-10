defmodule OptimalEngine.Pipeline.SemanticProcessor do
  @moduledoc """
  LLM-powered L0/L1 summary and embedding generation for Context structs.

  Wraps `Ollama` to produce richer, semantically meaningful tiered summaries
  than the regex-based `Classifier` can provide. When Ollama is unavailable the
  module is a silent no-op — the caller already holds regex-generated L0/L1 from
  `Classifier` and the pipeline continues unchanged.

  ## Processing pipeline (when Ollama is reachable)

  1. Generate a one-sentence L0 abstract via LLM.
  2. Generate a 2-3 paragraph L1 overview via LLM.
  3. Build a combined embedding input: `title + l0_abstract + content`.
  4. Embed via `Ollama.embed/2` and persist via `VectorStore.store/2`.
  5. Update the Context struct with the new L0/L1 strings.
  6. Write `.abstract.md` and `.overview.md` sidecar files alongside the source.

  ## Failure philosophy

  Every LLM call is wrapped in `rescue`. Any individual failure is logged and
  gracefully degraded — the pipeline always returns `{:ok, context}`.
  """

  require Logger

  alias OptimalEngine.Context
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store.Vectors, as: VectorStore

  @l0_max 200
  @l1_max 1_500
  @l0_content_limit 500
  @l1_content_limit 2_000

  @system_l0 "You are a signal classifier for a cognitive operating system. Be extremely concise. Output only the summary, nothing else."
  @system_l1 "You are a signal classifier for a cognitive operating system. Be concise and structured. Use bullet points for lists. Output only the overview, nothing else."

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Enriches a Context with LLM-generated L0/L1 summaries and a stored embedding.

  Always returns `{:ok, context}`. When Ollama is unavailable the context is
  returned unchanged. Partial failures (e.g. embedding succeeds but sidecar
  write fails) are logged and do not abort the call.
  """
  @spec process(Context.t()) :: {:ok, Context.t()}
  def process(%Context{} = context) do
    cond do
      not enabled?() ->
        Logger.debug("[SemanticProcessor] disabled, skipping semantic processing")
        {:ok, context}

      Ollama.available?() ->
        run_semantic_pipeline(context)

      true ->
        Logger.debug("[SemanticProcessor] Ollama unavailable, skipping semantic processing")
        {:ok, context}
    end
  end

  @doc "Whether asynchronous LLM enrichment is enabled for intake."
  @spec enabled?() :: boolean()
  def enabled? do
    :optimal_engine
    |> Application.get_env(:semantic_processing, [])
    |> Keyword.get(:enabled, true)
  end

  @doc """
  Generates a one-sentence L0 abstract via LLM.

  Returns `{:ok, String.t()}` (max #{@l0_max} chars) or `{:error, reason}`.
  """
  @spec generate_l0(String.t(), map()) :: {:ok, String.t()} | {:error, term()}
  def generate_l0(content, metadata) when is_binary(content) and is_map(metadata) do
    title = Map.get(metadata, :title, "")
    genre = Map.get(metadata, :genre, "")
    node = Map.get(metadata, :node, "")
    snippet = truncate(content, @l0_content_limit)

    prompt = """
    Summarize in exactly one sentence for instant recall. Include the most important fact, decision, or action item.

    Title: #{title}
    Genre: #{genre}
    Node: #{node}
    Content: #{snippet}

    One-sentence summary:
    """

    case Ollama.generate(prompt, system: @system_l0) do
      {:ok, text} ->
        {:ok, text |> String.trim() |> truncate(@l0_max)}

      {:error, reason} ->
        Logger.warning("[SemanticProcessor] generate_l0 failed: #{inspect(reason)}")
        {:error, reason}
    end
  rescue
    e ->
      Logger.warning("[SemanticProcessor] generate_l0 exception: #{inspect(e)}")
      {:error, :exception}
  end

  @doc """
  Generates a 2-3 paragraph L1 overview via LLM.

  Returns `{:ok, String.t()}` (max #{@l1_max} chars) or `{:error, reason}`.
  """
  @spec generate_l1(String.t(), map()) :: {:ok, String.t()} | {:error, term()}
  def generate_l1(content, metadata) when is_binary(content) and is_map(metadata) do
    title = Map.get(metadata, :title, "")
    genre = Map.get(metadata, :genre, "")
    node = Map.get(metadata, :node, "")
    snippet = truncate(content, @l1_content_limit)

    prompt = """
    Write a 2-3 paragraph overview of this content. Capture: key points, decisions made, action items, people mentioned, and financial data if any.

    Title: #{title}
    Genre: #{genre}
    Node: #{node}
    Content: #{snippet}

    Overview:
    """

    case Ollama.generate(prompt, system: @system_l1) do
      {:ok, text} ->
        {:ok, text |> String.trim() |> truncate(@l1_max)}

      {:error, reason} ->
        Logger.warning("[SemanticProcessor] generate_l1 failed: #{inspect(reason)}")
        {:error, reason}
    end
  rescue
    e ->
      Logger.warning("[SemanticProcessor] generate_l1 exception: #{inspect(e)}")
      {:error, :exception}
  end

  @doc """
  Writes `.abstract.md` and `.overview.md` sidecars alongside the source file.

  Only writes when `context.path` is non-nil and the parent directory exists.
  Returns `:ok` or `{:error, reason}`.
  """
  @spec write_sidecar_files(Context.t()) :: :ok | {:error, term()}
  def write_sidecar_files(%Context{path: nil}), do: :ok

  def write_sidecar_files(%Context{path: path, l0_abstract: l0, l1_overview: l1} = _ctx)
      when is_binary(path) do
    dir = Path.dirname(path)

    if File.dir?(dir) do
      base = Path.rootname(path)
      write_sidecar(base <> ".abstract.md", l0 || "")
      write_sidecar(base <> ".overview.md", l1 || "")
    else
      Logger.debug("[SemanticProcessor] Sidecar skipped — directory not found: #{dir}")
      :ok
    end
  end

  @doc """
  Returns `true` if the `.abstract.md` sidecar exists and is at least as new as
  `source_path`. Used by the indexer to skip already-processed files.
  """
  @spec sidecar_fresh?(String.t()) :: boolean()
  def sidecar_fresh?(source_path) when is_binary(source_path) do
    abstract_path = Path.rootname(source_path) <> ".abstract.md"

    with {:ok, %{mtime: sidecar_mtime}} <- File.stat(abstract_path),
         {:ok, %{mtime: source_mtime}} <- File.stat(source_path) do
      sidecar_mtime >= source_mtime
    else
      _ -> false
    end
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp run_semantic_pipeline(%Context{} = context) do
    metadata = metadata_from_context(context)
    content = context.content || ""

    {l0, l1} = generate_summaries(content, metadata, context)
    updated = %Context{context | l0_abstract: l0, l1_overview: l1}

    generate_and_store_embedding(updated)
    write_sidecar_files(updated)

    {:ok, updated}
  rescue
    e ->
      Logger.warning("[SemanticProcessor] Pipeline exception: #{inspect(e)}")
      {:ok, context}
  end

  defp generate_summaries(content, metadata, context) do
    l0 =
      case generate_l0(content, metadata) do
        {:ok, text} -> text
        {:error, _} -> context.l0_abstract || ""
      end

    l1 =
      case generate_l1(content, metadata) do
        {:ok, text} -> text
        {:error, _} -> context.l1_overview || ""
      end

    {l0, l1}
  end

  defp generate_and_store_embedding(%Context{id: nil}), do: :skip

  defp generate_and_store_embedding(%Context{id: id} = context) do
    embed_input = build_embed_input(context)

    case Ollama.embed(embed_input) do
      {:ok, vector} ->
        case VectorStore.store(id, vector) do
          :ok ->
            Logger.debug("[SemanticProcessor] Stored embedding for context #{id}")

          {:error, reason} ->
            Logger.warning(
              "[SemanticProcessor] VectorStore.store failed for #{id}: #{inspect(reason)}"
            )
        end

      {:error, reason} ->
        Logger.warning("[SemanticProcessor] Ollama.embed failed for #{id}: #{inspect(reason)}")
    end
  rescue
    e ->
      Logger.warning("[SemanticProcessor] Embedding exception: #{inspect(e)}")
  end

  defp build_embed_input(%Context{title: title, l0_abstract: l0, content: content}) do
    [title || "", l0 || "", content || ""]
    |> Enum.reject(&(&1 == ""))
    |> Enum.join("\n")
  end

  defp write_sidecar(path, content) do
    case File.write(path, content) do
      :ok ->
        Logger.debug("[SemanticProcessor] Wrote sidecar: #{path}")
        :ok

      {:error, reason} = err ->
        Logger.warning("[SemanticProcessor] Failed to write sidecar #{path}: #{inspect(reason)}")
        err
    end
  end

  defp metadata_from_context(%Context{} = ctx) do
    genre =
      if ctx.signal do
        ctx.signal.genre || "note"
      else
        "note"
      end

    %{
      title: ctx.title || "",
      genre: genre,
      node: ctx.node || "inbox"
    }
  end

  defp truncate(str, max_len) when is_binary(str) do
    if String.length(str) > max_len do
      str
      |> String.slice(0, max_len)
      |> String.replace(~r/\s+\S+$/, "")
      |> Kernel.<>("...")
    else
      str
    end
  end
end
