defmodule OptimalEngine.SessionCompressor do
  @moduledoc """
  Stateless module that compresses session transcripts while preserving signal.

  Two compression paths:
  - LLM compression (when Ollama is available) — structured summary via language model.
  - Regex fallback — keeps high-signal lines; truncates aggressively if still large.

  All functions are best-effort and always return `{:ok, text}`.
  """

  alias OptimalEngine.Embed.Ollama
  require Logger

  @compression_threshold 10_000
  @message_threshold 20
  @llm_prompt_max_chars 4_000
  @regex_result_max_chars 5_000
  @head_chars 2_000
  @tail_chars 2_000

  # Signal-bearing keywords for assistant message filtering
  @signal_patterns ~w(decided action todo will $ deal close agreed confirmed scheduled deadline)

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Compresses a session transcript string, preserving maximum signal.

  Only compresses when `String.length(text) > 10_000` or `opts[:force] == true`.

  Returns `{:ok, compressed_text}`. Never raises.
  """
  @spec compress(String.t(), keyword()) :: {:ok, String.t()}
  def compress(text, opts \\ []) when is_binary(text) do
    force = Keyword.get(opts, :force, false)

    if force or String.length(text) > @compression_threshold do
      do_compress(text)
    else
      {:ok, text}
    end
  rescue
    err ->
      Logger.warning("[SessionCompressor] compress/2 rescued: #{inspect(err)}")
      {:ok, text}
  end

  @doc """
  Returns `true` when a message list warrants compression before commit.

  Triggers when message count exceeds 20 or total content characters exceed 10,000.
  """
  @spec should_compress?([map()]) :: boolean()
  def should_compress?(messages) when is_list(messages) do
    length(messages) > @message_threshold or
      messages
      |> Enum.map(fn %{content: c} -> String.length(c) end)
      |> Enum.sum() > @compression_threshold
  rescue
    _ -> false
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp do_compress(text) do
    if Ollama.available?() do
      llm_compress(text)
    else
      regex_compress(text)
    end
  end

  # LLM path -------------------------------------------------------------------

  defp llm_compress(text) do
    truncated = String.slice(text, 0, @llm_prompt_max_chars)

    prompt = """
    Compress this conversation transcript to its essential signal. Preserve ALL of:
    - Decisions made (who decided what, when)
    - Action items (who does what, by when)
    - Financial data (amounts, pricing, deals)
    - Proper nouns (people, companies, projects)
    - Dates and deadlines
    - Key facts and commitments

    Remove: pleasantries, repetition, filler, tangents, assistant acknowledgments.

    Transcript:
    #{truncated}

    Compressed summary:
    """

    system =
      "You are a signal compressor. Preserve maximum information density. " <>
        "Output the compressed summary only, in structured bullet format."

    case Ollama.generate(prompt, system: system) do
      {:ok, compressed} ->
        Logger.debug(
          "[SessionCompressor] LLM compression: #{String.length(text)} → #{String.length(compressed)} chars"
        )

        {:ok, String.trim(compressed)}

      {:error, reason} ->
        Logger.info(
          "[SessionCompressor] LLM unavailable (#{inspect(reason)}), using regex fallback"
        )

        regex_compress(text)
    end
  rescue
    err ->
      Logger.warning(
        "[SessionCompressor] llm_compress rescued: #{inspect(err)}, falling back to regex"
      )

      regex_compress(text)
  end

  # Regex path -----------------------------------------------------------------

  defp regex_compress(text) do
    lines = String.split(text, "\n")

    kept =
      Enum.filter(lines, fn line ->
        cond do
          user_line?(line) -> true
          assistant_line?(line) -> high_signal_assistant?(line)
          true -> true
        end
      end)

    result = Enum.join(kept, "\n")

    final =
      if String.length(result) > @regex_result_max_chars do
        head = String.slice(result, 0, @head_chars)
        tail = String.slice(result, -@tail_chars, @tail_chars)
        head <> "\n\n... [compressed] ...\n\n" <> tail
      else
        result
      end

    Logger.debug(
      "[SessionCompressor] Regex compression: #{String.length(text)} → #{String.length(final)} chars"
    )

    {:ok, final}
  rescue
    err ->
      Logger.warning("[SessionCompressor] regex_compress rescued: #{inspect(err)}")
      {:ok, text}
  end

  defp user_line?(line) do
    String.match?(line, ~r/^\*\*USER:\*\*/i) or
      String.match?(line, ~r/^USER:/i) or
      String.match?(line, ~r/^\[user\]/i)
  end

  defp assistant_line?(line) do
    String.match?(line, ~r/^\*\*ASSISTANT:\*\*/i) or
      String.match?(line, ~r/^ASSISTANT:/i) or
      String.match?(line, ~r/^\[assistant\]/i)
  end

  defp high_signal_assistant?(line) do
    lower = String.downcase(line)

    contains_signal_keyword =
      Enum.any?(@signal_patterns, &String.contains?(lower, &1))

    contains_number = String.match?(line, ~r/\d/)

    contains_date =
      String.match?(
        line,
        ~r/\b\d{4}[-\/]\d{2}[-\/]\d{2}\b|\b(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\b/i
      )

    contains_proper_noun = String.match?(line, ~r/\b[A-Z][a-z]+\b/)

    contains_signal_keyword or contains_number or contains_date or contains_proper_noun
  end
end
