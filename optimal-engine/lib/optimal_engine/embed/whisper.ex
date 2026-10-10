defmodule OptimalEngine.Embed.Whisper do
  @moduledoc """
  Whisper.cpp local-HTTP client.

  Transcribes audio to text via a `whisper.cpp` server (the typical
  `/inference` endpoint). The resulting text is then routed through
  `OptimalEngine.Embed.Ollama.embed_text/2` so audio chunks land in the
  same 768-dim aligned space as text + image, making cross-modal retrieval
  possible (text query → audio segment).

  Whisper itself does not produce 768-dim vectors — it produces text. This
  module is the "audio → text" bridge; the text→vector step happens in
  `OptimalEngine.Pipeline.Embedder`.

  ## Configuration (`config/config.exs`)

      config :optimal_engine, :whisper,
        url: "http://localhost:8081/inference",
        model: "base.en",
        timeout_ms: 120_000

  ## Graceful degradation

  If the whisper server is unreachable or returns a non-2xx status, this
  module returns `{:error, reason}` — callers treat it as "skip this audio
  chunk for now" and move on. The raw audio file stays in the
  assets/ dir so Phase 10 re-runs can pick it up.
  """

  require Logger

  @type transcript :: %{
          text: String.t(),
          segments: [%{start: float(), end: float() | nil, text: String.t()}],
          language: String.t() | nil,
          duration_ms: non_neg_integer() | nil
        }

  @availability_ttl_ms 60_000
  @availability_cache_key :whisper_availability_cache

  @doc """
  Returns `true` if the whisper server responds to a probe within the
  configured timeout. Result is cached in the process dictionary for 60s
  so we don't hammer the health endpoint on every transcribe call.
  """
  @spec available?() :: boolean()
  def available? do
    now = System.monotonic_time(:millisecond)

    case Process.get(@availability_cache_key) do
      {result, cached_at} when now - cached_at < @availability_ttl_ms ->
        result

      _ ->
        result = probe()
        Process.put(@availability_cache_key, {result, now})
        result
    end
  end

  @doc """
  Transcribe an audio file (filesystem path) via the whisper.cpp HTTP
  server. Returns `{:ok, %{text, segments, language, duration_ms}}` on
  success.
  """
  @spec transcribe(String.t(), keyword()) :: {:ok, transcript()} | {:error, atom()}
  def transcribe(path, opts \\ []) when is_binary(path) do
    cfg = config()
    url = Keyword.get(opts, :url, cfg[:url])
    timeout_ms = Keyword.get(opts, :timeout_ms, cfg[:timeout_ms] || 120_000)

    with {:ok, bytes} <- File.read(path) do
      send_multipart(url, Path.basename(path), bytes, timeout_ms)
    end
  end

  # ── private ──────────────────────────────────────────────────────────────

  defp config do
    Application.get_env(:optimal_engine, :whisper,
      url: "http://localhost:8081/inference",
      model: "base.en",
      timeout_ms: 120_000
    )
  end

  defp probe do
    cfg = config()
    url = cfg[:url] || "http://localhost:8081/inference"

    headers = [{~c"connection", ~c"close"}]
    opts = [timeout: 1_500, connect_timeout: 1_500]

    case :httpc.request(:head, {String.to_charlist(url), headers}, opts, []) do
      {:ok, _} -> true
      _ -> false
    end
  rescue
    _ -> false
  end

  defp send_multipart(url, filename, bytes, timeout_ms) do
    boundary = "----optimal-whisper-#{System.unique_integer([:positive])}"
    content_type = guess_audio_mime(filename)

    body =
      [
        "--",
        boundary,
        "\r\n",
        ~s(Content-Disposition: form-data; name="file"; filename=") <>
          filename <> ~s("\r\n),
        "Content-Type: ",
        content_type,
        "\r\n\r\n",
        bytes,
        "\r\n--",
        boundary,
        "--\r\n"
      ]
      |> IO.iodata_to_binary()

    headers = [{~c"connection", ~c"close"}]
    ct = ~c"multipart/form-data; boundary=#{boundary}"
    request = {String.to_charlist(url), headers, ct, body}
    opts = [timeout: timeout_ms, connect_timeout: 3_000]

    case :httpc.request(:post, request, opts, body_format: :binary) do
      {:ok, {{_, status, _}, _, resp}} when status in 200..299 ->
        decode_response(resp)

      {:ok, {{_, status, _}, _, resp}} ->
        Logger.warning("Whisper #{status}: #{String.slice(resp, 0, 200)}")
        {:error, :http_status}

      {:error, :econnrefused} ->
        {:error, :unreachable}

      {:error, reason} ->
        Logger.warning("Whisper request failed: #{inspect(reason)}")
        {:error, :request_failed}
    end
  end

  defp decode_response(body) do
    case Jason.decode(body) do
      {:ok, %{"text" => text} = json} ->
        {:ok,
         %{
           text: String.trim(text),
           segments: normalize_segments(Map.get(json, "segments", [])),
           language: Map.get(json, "language"),
           duration_ms: Map.get(json, "duration_ms")
         }}

      {:ok, other} ->
        Logger.warning("Whisper unexpected JSON: #{inspect(other)}")
        {:error, :unexpected_response}

      {:error, _reason} ->
        {:error, :json_decode}
    end
  end

  defp normalize_segments(segments) when is_list(segments) do
    Enum.map(segments, fn seg ->
      %{
        start: Map.get(seg, "start", 0.0) * 1.0,
        end: Map.get(seg, "end"),
        text: Map.get(seg, "text", "") |> String.trim()
      }
    end)
  end

  defp normalize_segments(_), do: []

  defp guess_audio_mime(filename) do
    case Path.extname(filename) |> String.downcase() do
      ".mp3" -> "audio/mpeg"
      ".wav" -> "audio/wav"
      ".m4a" -> "audio/mp4"
      ".ogg" -> "audio/ogg"
      ".flac" -> "audio/flac"
      _ -> "application/octet-stream"
    end
  end
end
