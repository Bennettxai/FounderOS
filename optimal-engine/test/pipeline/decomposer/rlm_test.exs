defmodule OptimalEngine.Pipeline.Decomposer.RLMTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Pipeline.Decomposer.RLM

  test "returns structured atoms from the sidecar" do
    script = temporary_script()

    runner = fn _executable, _args, opts ->
      payload = Jason.decode!(opts[:input])
      assert payload["content"] == "A long source"

      {Jason.encode!(%{"strategy" => "rlm", "atoms" => [%{"title" => "A", "body" => "B"}]}), 0}
    end

    assert {:ok, %{"atoms" => [%{"title" => "A"}]}} =
             RLM.decompose("A long source", script: script, runner: runner)
  end

  test "rejects empty sidecar output" do
    script = temporary_script()
    runner = fn _, _, _ -> {Jason.encode!(%{"atoms" => []}), 0} end

    assert {:error, :empty_rlm_output} =
             RLM.decompose("source", script: script, runner: runner)
  end

  defp temporary_script do
    path = Path.join(System.tmp_dir!(), "optimal-rlm-#{System.unique_integer([:positive])}.py")
    File.write!(path, "# test")
    on_exit(fn -> File.rm(path) end)
    path
  end
end
