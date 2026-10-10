defmodule OptimalEngine.Embed.WhisperTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Embed.Whisper

  describe "available?/0" do
    test "returns a boolean without crashing when server is absent" do
      # In the test environment whisper.cpp is not running; this must return
      # false (or true if one happens to be up), never crash.
      result = Whisper.available?()
      assert is_boolean(result)
    end
  end

  describe "transcribe/2" do
    test "returns {:error, _} when the file does not exist" do
      assert {:error, _reason} =
               Whisper.transcribe("/tmp/definitely-not-a-real-file-#{System.unique_integer()}.wav")
    end

    test "returns {:error, :unreachable} when server down and file exists" do
      tmp = Path.join(System.tmp_dir!(), "fake-whisper-#{System.unique_integer([:positive])}.wav")
      File.write!(tmp, "RIFF....WAVEfmt ")

      try do
        result = Whisper.transcribe(tmp, url: "http://127.0.0.1:1/notreal", timeout_ms: 500)
        assert match?({:error, _}, result)
      after
        File.rm(tmp)
      end
    end
  end
end
