defmodule OptimalEngine.Pipeline.Parser.BinaryFormatsTest do
  @moduledoc """
  Smoke tests for binary-format parsers (PDF / image / audio / video).

  These tests don't require the external tools (pdftotext / tesseract /
  whisper / ffmpeg) to be installed. They verify the graceful-degradation
  path: when the tool is missing or the file is unavailable, the parser
  still returns a well-formed `%ParsedDoc{}` with warnings set.
  """
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Parser.{Audio, Image, ParsedDoc, Pdf, Video}

  describe "PDF" do
    test "returns ParsedDoc with warning when pdftotext missing or path invalid" do
      tmp = Path.join(System.tmp_dir!(), "nofile_#{System.unique_integer([:positive])}.pdf")
      File.write!(tmp, "not a real pdf")

      try do
        assert {:ok, %ParsedDoc{modality: :mixed} = doc} = Pdf.parse(tmp, [])
        assert is_binary(doc.text)
        assert is_list(doc.warnings)
      after
        File.rm(tmp)
      end
    end

    test "parse_text is not supported for pdf" do
      assert {:error, :binary_format_requires_path} = Pdf.parse_text("", [])
    end
  end

  describe "Image" do
    test "preserves image as asset with warning when tesseract is absent" do
      tmp = Path.join(System.tmp_dir!(), "fake_#{System.unique_integer([:positive])}.png")
      File.write!(tmp, "\x89PNG\r\n\x1a\n")

      try do
        assert {:ok, %ParsedDoc{modality: :image} = doc} = Image.parse(tmp, [])
        assert Enum.any?(doc.assets, &(&1.modality == :image))
      after
        File.rm(tmp)
      end
    end
  end

  describe "Audio" do
    test "reports graceful warning when whisper is unreachable" do
      tmp = Path.join(System.tmp_dir!(), "fake_#{System.unique_integer([:positive])}.wav")
      File.write!(tmp, "RIFF....WAVE")

      try do
        assert {:ok, %ParsedDoc{modality: :audio} = doc} = Audio.parse(tmp, [])
        assert Enum.any?(doc.assets, &(&1.modality == :audio))
        assert is_list(doc.warnings)
      after
        File.rm(tmp)
      end
    end
  end

  describe "Video" do
    test "preserves video as asset when ffmpeg missing" do
      tmp = Path.join(System.tmp_dir!(), "fake_#{System.unique_integer([:positive])}.mp4")
      File.write!(tmp, "\x00\x00\x00\x20ftypmp42")

      try do
        assert {:ok, %ParsedDoc{modality: :video} = doc} = Video.parse(tmp, [])
        assert Enum.any?(doc.assets, &(&1.modality == :video))
      after
        File.rm(tmp)
      end
    end
  end
end
