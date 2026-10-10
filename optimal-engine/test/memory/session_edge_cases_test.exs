defmodule OptimalEngine.Memory.Session.EdgeCasesTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Session

  setup do
    session_id = "edge_test_#{System.unique_integer([:positive])}"
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

  # --- Session isolation ---

  describe "session isolation" do
    test "two sessions do not share messages" do
      id1 = "iso_1_#{System.unique_integer([:positive])}"
      id2 = "iso_2_#{System.unique_integer([:positive])}"

      {:ok, _} = Session.start_session(id1)
      {:ok, _} = Session.start_session(id2)

      Session.add_message(id1, :user, "Session 1 only")
      Session.add_message(id2, :user, "Session 2 only")

      msgs1 = Session.messages(id1)
      msgs2 = Session.messages(id2)

      assert length(msgs1) == 1
      assert length(msgs2) == 1
      assert hd(msgs1).content == "Session 1 only"
      assert hd(msgs2).content == "Session 2 only"

      Session.stop(id1)
      Session.stop(id2)
    end

    test "stopping one session does not affect another" do
      id1 = "stop_1_#{System.unique_integer([:positive])}"
      id2 = "stop_2_#{System.unique_integer([:positive])}"

      {:ok, _} = Session.start_session(id1)
      {:ok, _} = Session.start_session(id2)

      Session.add_message(id2, :user, "persist this")
      Session.stop(id1)

      # id2 should still be accessible
      msgs = Session.messages(id2)
      assert length(msgs) == 1
      Session.stop(id2)
    end
  end

  # --- Message roles ---

  describe "all message roles" do
    test "system role is stored and retrieved" do
      session_id = "roles_#{System.unique_integer([:positive])}"
      {:ok, _} = Session.start_session(session_id)
      Session.add_message(session_id, :system, "You are an expert.")
      msgs = Session.messages(session_id)
      assert hd(msgs).role == :system
      Session.stop(session_id)
    end

    test "tool role is stored and retrieved", %{session_id: id} do
      Session.add_message(id, :tool, "Tool output: success")
      msgs = Session.messages(id)
      assert hd(msgs).role == :tool
      assert hd(msgs).content == "Tool output: success"
    end

    test "messages preserve all four roles in order", %{session_id: id} do
      Session.add_message(id, :system, "System context")
      Session.add_message(id, :user, "User input")
      Session.add_message(id, :assistant, "AI response")
      Session.add_message(id, :tool, "Tool result")

      msgs = Session.messages(id)
      roles = Enum.map(msgs, & &1.role)
      assert roles == [:system, :user, :assistant, :tool]
    end
  end

  # --- Message timestamps ---

  describe "message timestamps" do
    test "each message has a DateTime timestamp", %{session_id: id} do
      Session.add_message(id, :user, "hello")
      msgs = Session.messages(id)
      assert %DateTime{} = hd(msgs).timestamp
    end

    test "timestamps are in chronological order", %{session_id: id} do
      Session.add_message(id, :user, "first")
      Process.sleep(2)
      Session.add_message(id, :user, "second")

      msgs = Session.messages(id)
      [first, second] = msgs
      assert DateTime.compare(first.timestamp, second.timestamp) in [:lt, :eq]
    end
  end

  # --- summarize/1 edge cases ---

  describe "summarize/1 edge cases" do
    test "summary with only system messages", %{session_id: id} do
      Session.add_message(id, :system, "You are an assistant")
      summary = Session.summarize(id)
      assert String.contains?(summary, "[system]")
      assert String.contains?(summary, "You are an assistant")
    end

    test "summary with no system message shows recent only", %{session_id: id} do
      for i <- 1..3, do: Session.add_message(id, :user, "msg #{i}")
      summary = Session.summarize(id)
      refute String.contains?(summary, "[system]")
      assert String.contains?(summary, "[user]")
    end

    test "summary truncates long message content", %{session_id: id} do
      long_content = String.duplicate("x", 300)
      Session.add_message(id, :user, long_content)
      summary = Session.summarize(id)
      # The content is truncated at 200 chars + "..."
      assert String.contains?(summary, "...")
    end

    test "summary shows omission count when many messages", %{session_id: id} do
      # system + 10 regular messages, recent only shows last 5
      Session.add_message(id, :system, "System prompt")
      for i <- 1..10, do: Session.add_message(id, :user, "msg #{i}")

      summary = Session.summarize(id)
      # 10 total, 5 recent, 1 system -> 4 omitted
      assert String.contains?(summary, "omitted")
    end
  end

  # --- messages/2 edge cases ---

  describe "messages/2 edge cases" do
    test "requesting 0 messages returns empty list", %{session_id: id} do
      Session.add_message(id, :user, "hello")
      msgs = Session.messages(id, 0)
      assert msgs == []
    end

    test "requesting N from empty session returns empty list", %{session_id: id} do
      assert Session.messages(id, 5) == []
    end

    test "last N messages are correct slice", %{session_id: id} do
      for i <- 1..7, do: Session.add_message(id, :user, "msg #{i}")
      msgs = Session.messages(id, 3)
      assert length(msgs) == 3
      contents = Enum.map(msgs, & &1.content)
      assert contents == ["msg 5", "msg 6", "msg 7"]
    end
  end

  # --- Duplicate start_session ---

  describe "start_session/1 idempotency" do
    test "second start returns already_started error with same pid" do
      id = "dup_edge_#{System.unique_integer([:positive])}"
      {:ok, pid1} = Session.start_session(id)
      {:error, {:already_started, pid2}} = Session.start_session(id)
      assert pid1 == pid2
      Session.stop(id)
    end
  end

  # --- load/1 when session doesn't exist on disk ---

  describe "load/1 without prior persist" do
    test "starts session even when no disk file exists", %{session_id: id} do
      new_id = "load_no_file_#{System.unique_integer([:positive])}"
      {:ok, _} = Session.load(new_id)
      # Should start with empty messages
      assert Session.messages(new_id) == []
      Session.stop(new_id)
    end
  end

  # --- Auto-persist disabled ---

  describe "auto-persist control" do
    test "auto-persist can be disabled via config", %{session_id: id} do
      Application.put_env(:optimal_engine, OptimalEngine.Memory.Session, auto_persist: false)

      for _ <- 1..15, do: Session.add_message(id, :user, "no auto persist")

      # Should not crash even without persisting
      assert length(Session.messages(id)) == 15

      Application.delete_env(:optimal_engine, OptimalEngine.Memory.Session)
    end
  end
end
