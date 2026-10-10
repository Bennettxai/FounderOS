defmodule OptimalEngine.Pipeline.Parser.Audio do
  @moduledoc """
  Audio parser — transcription via a local `whisper.cpp` HTTP server.

  Configuration (`config/config.exs`):

      config :optimal_engine, :whisper,
        url: "http://localhost:8081/inference",
        model: "base.en"

  If the whisper server is unreachable, returns a metadata-only ParsedDoc
  with the audio preserved as an Asset. Phase 5 will also embed the audio
  directly (via CLAP or whisper's encoder) so retrieval still works even
  without transcript.
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{Asset, ParsedDoc, StructuralElement}

  require Logger

  @impl true
  def parse(path, _opts) when is_binary(path) do
    asset =
      case Asset.from_path(path, modality: :audio) do
        {:ok, a} -> a
        _ -> nil
      end

    {text, structure, warnings, extras} = transcribe(path)

    {:ok,
     ParsedDoc.new(
       path: path,
       text: text,
       structure: structure,
       modality: :audio,
       metadata:
         Map.merge(
           %{
             format: Path.extname(path) |> String.trim_leading(".") |> String.downcase(),
             byte_size: byte_size(text),
             has_transcript: text != ""
           },
           extras
         ),
       assets: if(asset, do: [asset], else: []),
       warnings: warnings
     )}
  end

  @impl true
  def parse_text(_text, _opts), do: {:error, :binary_format_requires_path}

  defp transcribe(path) do
    config = Application.get_env(:optimal_engine, :whisper, [])
    url = Keyword.get(config, :url, "http://localhost:8081/inference")

    case request_whisper(url, path) do
      {:ok, body} ->
        case Jason.decode(body) do
          {:ok, %{"text" => text} = json} ->
            segments = Map.get(json, "segments", [])

            {
              String.trim(text),
              segments_to_structure(segments),
              [],
              %{segments: length(segments)}
            }

          {:ok, other} ->
            {"", [], ["whisper returned unexpected json: #{inspect(other)}"], %{}}

          {:error, reason} ->
            {"", [], ["whisper json decode failed: #{inspect(reason)}"], %{}}
        end

      {:error, :no_whisper} ->
        {"", [], ["whisper.cpp server unreachable at #{url} — install + run for transcription"],
         %{}}

      {:error, reason} ->
        {"", [], ["whisper request failed: #{inspect(reason)}"], %{}}
    end
  end

  # Minimal HTTP POST via :httpc (inets is already extra_applications-started).
  defp request_whisper(url, path) do
    with {:ok, bytes} <- File.read(path) do
      boundary = "----optimal-#{System.unique_integer([:positive])}"
      filename = Path.basename(path)
      content_type = audio_mime(filename)

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
      opts = [timeout: 120_000, connect_timeout: 3_000]

      case :httpc.request(:post, request, opts, body_format: :binary) do
        {:ok, {{_, status, _}, _, resp}} when status in 200..299 ->
          {:ok, resp}

        {:ok, {{_, status, _}, _, resp}} ->
          {:error, {:http_status, status, String.slice(resp, 0, 200)}}

        {:error, :econnrefused} ->
          {:error, :no_whisper}

        {:error, reason} ->
          {:error, reason}
      end
    end
  end

  defp audio_mime(filename) do
    case Path.extname(filename) |> String.downcase() do
      ".mp3" -> "audio/mpeg"
      ".wav" -> "audio/wav"
      ".m4a" -> "audio/mp4"
      ".ogg" -> "audio/ogg"
      ".flac" -> "audio/flac"
      _ -> "application/octet-stream"
    end
  end

  defp segments_to_structure(segments) do
    segments
    |> Enum.with_index()
    |> Enum.map(fn {seg, idx} ->
      start = Map.get(seg, "start", idx * 1.0)
      text = Map.get(seg, "text", "") |> String.trim()

      StructuralElement.new(:timestamp,
        text: text,
        offset: idx,
        length: byte_size(text),
        metadata: %{seconds: start, segment_index: idx}
      )
    end)
  end
end
