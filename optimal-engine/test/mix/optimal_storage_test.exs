defmodule Mix.Tasks.Optimal.StorageTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  test "lists providers as JSON" do
    output = capture_io(fn -> Mix.Tasks.Optimal.Storage.run(["list"]) end)
    assert {:ok, providers} = Jason.decode(output)
    assert Enum.any?(providers, &(&1["id"] == "sqlite"))
    assert Enum.any?(providers, &(&1["id"] == "fractal"))
  end

  test "plans multiple use cases" do
    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Storage.run(["plan", "desktop_local,analytics"])
      end)

    assert {:ok, plan} = Jason.decode(output)
    assert Enum.any?(plan["providers"], &(&1["id"] == "duckdb"))
  end
end
