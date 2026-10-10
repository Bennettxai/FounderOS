defmodule OptimalEngine.Pipeline.Parser.Video do
  @moduledoc """
  Video parser. If `ffmpeg` is available, extracts:

    * scene-aware keyframes via `ffmpeg -vf select='gt(scene,T)'` — each
      frame is handed to the Image backend for OCR + kept as an Asset
    * the audio track — handed to the Audio backend for transcription

  If `ffmpeg` isn't on `PATH`, returns a metadata-only ParsedDoc with the
  video preserved as an Asset.

  Configuration (via `:optimal_engine, :video`):
    * `:max_keyframes` — cap on extracted frames (default 10)
    * `:scene_threshold` — ffmpeg scene-change sensitivity 0.0–1.0 (default 0.3)
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{Asset, Audio, Image, ParsedDoc, StructuralElement}

  @impl true
  def parse(path, opts) when is_binary(path) do
    video_asset =
      case Asset.from_path(path, modality: :video) do
        {:ok, a} -> a
        _ -> nil
      end

    case System.find_executable("ffmpeg") do
      nil ->
        {:ok,
         ParsedDoc.new(
           path: path,
           text: "",
           modality: :video,
           metadata: %{format: format_from_path(path)},
           assets: if(video_asset, do: [video_asset], else: []),
           warnings: ["ffmpeg not on PATH — install for frame + audio extraction"]
         )}

      _ ->
        extract_with_ffmpeg(path, video_asset, opts)
    end
  end

  @impl true
  def parse_text(_text, _opts), do: {:error, :binary_format_requires_path}

  defp extract_with_ffmpeg(path, video_asset, opts) do
    tmp_dir = System.tmp_dir!() |> Path.join("optimal_video_#{System.unique_integer([:positive])}")
    File.mkdir_p!(tmp_dir)

    try do
      audio_path = Path.join(tmp_dir, "audio.wav")

      _ =
        System.cmd(
          "ffmpeg",
          ["-y", "-i", path, "-vn", "-acodec", "pcm_s16le", "-ar", "16000", "-ac", "1", audio_path],
          stderr_to_stdout: true
        )

      {frame_results, frame_warnings} = extract_keyframes(path, tmp_dir, opts)
      {audio_text, audio_assets, audio_warnings} = maybe_parse(Audio, audio_path, opts)

      frame_texts =
        Enum.map(frame_results, fn %{index: idx, timestamp_s: ts, text: text} ->
          if text != "" do
            "### Frame #{idx} (#{ts}s)\n#{text}"
          end
        end)
        |> Enum.reject(&is_nil/1)

      combined_text =
        (frame_texts ++ [audio_text])
        |> Enum.reject(&(&1 == "" or is_nil(&1)))
        |> Enum.join("\n\n")

      frame_assets = Enum.flat_map(frame_results, fn %{assets: a} -> a end)

      structure =
        Enum.map(frame_results, fn %{index: idx, timestamp_s: ts} ->
          StructuralElement.new(
            :frame,
            offset: 0,
            metadata: %{frame_index: idx, timestamp_s: ts}
          )
        end)

      assets =
        [video_asset | frame_assets ++ audio_assets]
        |> Enum.reject(&is_nil/1)

      {:ok,
       ParsedDoc.new(
         path: path,
         text: combined_text,
         structure: structure,
         modality: :video,
         metadata: %{
           format: format_from_path(path),
           frame_count: length(frame_results),
           has_audio: File.exists?(audio_path),
           frame_timestamps: Enum.map(frame_results, & &1.timestamp_s)
         },
         assets: assets,
         warnings: frame_warnings ++ audio_warnings
       )}
    after
      File.rm_rf!(tmp_dir)
    end
  end

  defp extract_keyframes(path, tmp_dir, opts) do
    cfg = Application.get_env(:optimal_engine, :video, [])
    max_frames = Keyword.get(opts, :max_keyframes, cfg[:max_keyframes] || 10)
    threshold = Keyword.get(opts, :scene_threshold, cfg[:scene_threshold] || 0.3)

    frames_dir = Path.join(tmp_dir, "frames")
    File.mkdir_p!(frames_dir)

    {output, _} =
      System.cmd(
        "ffmpeg",
        [
          "-y",
          "-i",
          path,
          "-vf",
          "select='gt(scene,#{threshold})',showinfo",
          "-vsync",
          "vfr",
          "-q:v",
          "3",
          "-frames:v",
          to_string(max_frames),
          Path.join(frames_dir, "frame_%04d.jpg")
        ],
        stderr_to_stdout: true
      )

    frame_files =
      case File.ls(frames_dir) do
        {:ok, files} ->
          files
          |> Enum.filter(&String.ends_with?(&1, ".jpg"))
          |> Enum.sort()

        _ ->
          []
      end

    if frame_files == [] do
      extract_uniform_keyframes(path, frames_dir, max_frames, opts)
    else
      timestamps = parse_showinfo_timestamps(output, length(frame_files))

      results =
        frame_files
        |> Enum.with_index(1)
        |> Enum.map(fn {file, idx} ->
          frame_path = Path.join(frames_dir, file)
          ts = Enum.at(timestamps, idx - 1, 0.0)
          {text, assets, _warnings} = maybe_parse(Image, frame_path, opts)

          assets =
            Enum.map(assets, fn a ->
              %{a | metadata: Map.merge(a.metadata || %{}, %{frame_index: idx, timestamp_s: ts})}
            end)

          %{index: idx, timestamp_s: ts, text: text, assets: assets}
        end)

      {results, []}
    end
  end

  defp extract_uniform_keyframes(path, frames_dir, max_frames, opts) do
    capped = min(max_frames, 5)

    {duration_output, _} =
      System.cmd(
        "ffmpeg",
        ["-i", path, "-f", "null", "-"],
        stderr_to_stdout: true
      )

    duration = parse_duration(duration_output)
    interval = if duration > 0, do: max(duration / (capped + 1), 1.0), else: 1.0

    {_, _} =
      System.cmd(
        "ffmpeg",
        [
          "-y",
          "-i",
          path,
          "-vf",
          "fps=1/#{interval}",
          "-frames:v",
          to_string(capped),
          "-q:v",
          "3",
          Path.join(frames_dir, "frame_%04d.jpg")
        ],
        stderr_to_stdout: true
      )

    frame_files =
      case File.ls(frames_dir) do
        {:ok, files} -> files |> Enum.filter(&String.ends_with?(&1, ".jpg")) |> Enum.sort()
        _ -> []
      end

    results =
      frame_files
      |> Enum.with_index(1)
      |> Enum.map(fn {file, idx} ->
        frame_path = Path.join(frames_dir, file)
        ts = Float.round(interval * idx, 1)
        {text, assets, _warnings} = maybe_parse(Image, frame_path, opts)

        assets =
          Enum.map(assets, fn a ->
            %{a | metadata: Map.merge(a.metadata || %{}, %{frame_index: idx, timestamp_s: ts})}
          end)

        %{index: idx, timestamp_s: ts, text: text, assets: assets}
      end)

    {results,
     if(frame_files == [], do: ["no frames extracted — video may be audio-only"], else: [])}
  end

  defp parse_showinfo_timestamps(output, expected_count) do
    Regex.scan(~r/pts_time:\s*([\d.]+)/, output)
    |> Enum.map(fn [_, ts] ->
      case Float.parse(ts) do
        {f, _} -> Float.round(f, 2)
        :error -> 0.0
      end
    end)
    |> Enum.take(expected_count)
  end

  defp parse_duration(output) do
    case Regex.run(~r/Duration:\s*(\d+):(\d+):(\d+)\.(\d+)/, output) do
      [_, h, m, s, _ms] ->
        String.to_integer(h) * 3600 + String.to_integer(m) * 60 + String.to_integer(s)

      _ ->
        0
    end
  end

  defp format_from_path(path) do
    Path.extname(path) |> String.trim_leading(".") |> String.downcase()
  end

  defp maybe_parse(_backend, path, _opts) when not is_binary(path), do: {"", [], []}

  defp maybe_parse(backend, path, opts) do
    if File.exists?(path) do
      case backend.parse(path, opts) do
        {:ok, doc} -> {doc.text, doc.assets, doc.warnings}
        _ -> {"", [], []}
      end
    else
      {"", [], []}
    end
  end
end
