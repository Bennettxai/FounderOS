defmodule OptimalEngine.Pipeline.Parser.Image do
  @moduledoc """
  Image parser — OCR via the system `tesseract` binary. If `tesseract` isn't
  on `PATH`, returns a metadata-only ParsedDoc with the image preserved as
  an Asset.

  The original image stays addressable as an asset so Phase 5's vision
  embedder can embed it directly (nomic-embed-vision).
  """

  @behaviour OptimalEngine.Pipeline.Parser.Backend

  alias OptimalEngine.Pipeline.Parser.{Asset, ParsedDoc}

  @impl true
  def parse(path, _opts) when is_binary(path) do
    asset =
      case Asset.from_path(path, modality: :image) do
        {:ok, a} -> a
        _ -> nil
      end

    {text, warnings} =
      case System.find_executable("tesseract") do
        nil ->
          {"", ["tesseract not on PATH — install tesseract for OCR text extraction"]}

        _bin ->
          run_tesseract(path)
      end

    {:ok,
     ParsedDoc.new(
       path: path,
       text: text,
       structure: [],
       modality: :image,
       metadata: %{
         format: Path.extname(path) |> String.trim_leading(".") |> String.downcase(),
         byte_size: byte_size(text),
         has_ocr: text != ""
       },
       assets: if(asset, do: [asset], else: []),
       warnings: warnings
     )}
  end

  @impl true
  def parse_text(_text, _opts), do: {:error, :binary_format_requires_path}

  defp run_tesseract(path) do
    # tesseract writes to stdout when output path is "-"
    case System.cmd("tesseract", [path, "-"], stderr_to_stdout: true) do
      {output, 0} ->
        {String.trim(output), []}

      {err, code} ->
        {"", ["tesseract exited #{code}: #{String.slice(err, 0, 200)}"]}
    end
  end
end
