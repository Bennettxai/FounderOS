defmodule OptimalEngine.Pipeline.Parser.Asset do
  @moduledoc """
  Represents a binary asset kept alongside extracted text in a `%ParsedDoc{}`.

  Images, audio files, video files, and PDFs all live on disk (or in the
  engine's `assets/` store) and this struct is the reference the Decomposer
  + Store use to link chunks back to their source binary.

  `hash` is content-addressed (sha256 of the raw bytes) so the same binary
  ingested twice produces a single Asset row.
  """

  @type modality :: :image | :audio | :video | :binary

  @type t :: %__MODULE__{
          hash: String.t(),
          content_type: String.t(),
          size: non_neg_integer(),
          path: String.t() | nil,
          modality: modality(),
          metadata: map()
        }

  defstruct hash: nil,
            content_type: "application/octet-stream",
            size: 0,
            path: nil,
            modality: :binary,
            metadata: %{}

  @doc "Build an Asset from a file path, hashing its contents."
  @spec from_path(String.t(), keyword()) :: {:ok, t()} | {:error, term()}
  def from_path(path, opts \\ []) when is_binary(path) do
    with {:ok, bytes} <- File.read(path) do
      {:ok,
       %__MODULE__{
         hash: "sha256:" <> (:crypto.hash(:sha256, bytes) |> Base.encode16(case: :lower)),
         content_type: Keyword.get(opts, :content_type, guess_content_type(path)),
         size: byte_size(bytes),
         path: path,
         modality: Keyword.get(opts, :modality, guess_modality(path)),
         metadata: Keyword.get(opts, :metadata, %{})
       }}
    end
  end

  defp guess_content_type(path) do
    case Path.extname(path) |> String.downcase() do
      ".png" -> "image/png"
      ".jpg" -> "image/jpeg"
      ".jpeg" -> "image/jpeg"
      ".gif" -> "image/gif"
      ".webp" -> "image/webp"
      ".svg" -> "image/svg+xml"
      ".mp3" -> "audio/mpeg"
      ".wav" -> "audio/wav"
      ".m4a" -> "audio/mp4"
      ".ogg" -> "audio/ogg"
      ".flac" -> "audio/flac"
      ".mp4" -> "video/mp4"
      ".mov" -> "video/quicktime"
      ".webm" -> "video/webm"
      ".pdf" -> "application/pdf"
      _ -> "application/octet-stream"
    end
  end

  defp guess_modality(path) do
    case Path.extname(path) |> String.downcase() do
      ext when ext in ~w(.png .jpg .jpeg .gif .webp .svg) -> :image
      ext when ext in ~w(.mp3 .wav .m4a .ogg .flac) -> :audio
      ext when ext in ~w(.mp4 .mov .webm) -> :video
      _ -> :binary
    end
  end
end
