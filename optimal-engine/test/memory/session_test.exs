defmodule OptimalEngine.Memory.SessionTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Session

  setup do
    session_id = "test_session_#{System.unique_integer([:positive])}"
    {:ok, _pid} = Session.start_session(session_id)

    on_exit(fn ->
      try do
        Session.stop(session_id)
      catch
        :exit, _ -> :ok
      end
    end)

    %{session_id: session_id}
  end

  describe "start_session/1" do
    test "starts a new session" do
      id = "new_session_#{System.unique_integer([:positive])}"
      assert {:ok, pid} = Session.start_session(id)
      assert is_pid(pid)
      Session.stop(id)
    end

    test "returns error if session already started" do
      id = "dup_#{System.unique_integer([:positive])}"
      {:ok, _} = Session.start_session(id)
      assert {:error, {:already_started, _}} = Session.start_session(id)
      Session.stop(id)
    end
  end

  describe "add_message/3 and messages/1" do
    test "adds and retrieves messages", %{session_id: id} do
      :ok = Session.add_message(id, :user, "Hello")
      :ok = Session.add_message(id, :assistant, "Hi there")

      msgs = Session.messages(id)
      assert length(msgs) == 2
      assert hd(msgs).role == :user
      assert hd(msgs).content == "Hello"
    end

    test "preserves message order", %{session_id: id} do
      for i <- 1..5 do
        Session.add_message(id, :user, "msg #{i}")
      end

      msgs = Session.messages(id)
      contents = Enum.map(msgs, & &1.content)
      assert contents == ["msg 1", "msg 2", "msg 3", "msg 4", "msg 5"]
    end
  end

  describe "messages/2" do
    test "returns last N messages", %{session_id: id} do
      for i <- 1..10 do
        Session.add_message(id, :user, "msg #{i}")
      end

      msgs = Session.messages(id, 3)
      assert length(msgs) == 3
      contents = Enum.map(msgs, & &1.content)
      assert contents == ["msg 8", "msg 9", "msg 10"]
    end

    test "returns all if N > total", %{session_id: id} do
      Session.add_message(id, :user, "only one")
      msgs = Session.messages(id, 100)
      assert length(msgs) == 1
    end
  end

  describe "summarize/1" do
    test "returns summary with recent messages", %{session_id: id} do
      Session.add_message(id, :system, "You are helpful")

      for i <- 1..10 do
        Session.add_message(id, :user, "message #{i}")
      end

      summary = Session.summarize(id)
      assert is_binary(summary)
      assert String.contains?(summary, "[system]")
      assert String.contains?(summary, "message 10")
    end

    test "handles empty session", %{session_id: id} do
      summary = Session.summarize(id)
      assert summary == ""
    end
  end

  describe "persist/1 and load/1" do
    @tag :tmp_dir
    test "persists and loads session from disk", %{tmp_dir: tmp_dir} do
      Application.put_env(:optimal_engine, OptimalEngine.Memory.Session, session_path: tmp_dir)

      id = "persist_test_#{System.unique_integer([:positive])}"
      {:ok, _} = Session.start_session(id)

      Session.add_message(id, :system, "System prompt")
      Session.add_message(id, :user, "Hello")
      Session.add_message(id, :assistant, "Hi")

      assert :ok = Session.persist(id)
      Session.stop(id)

      # Load into new session
      {:ok, _} = Session.load(id)
      msgs = Session.messages(id)
      assert length(msgs) == 3
      assert hd(msgs).role == :system
      assert hd(msgs).content == "System prompt"

      Session.stop(id)
      Application.delete_env(:optimal_engine, OptimalEngine.Memory.Session)
    end
  end
end
