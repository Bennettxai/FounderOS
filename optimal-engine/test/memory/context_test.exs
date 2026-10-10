defmodule OptimalEngine.Memory.ContextTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.{Context, Session}

  setup do
    :ets.delete_all_objects(:optimal_engine_memory_store)
    :ets.delete_all_objects(:optimal_engine_memory_collections)

    session_id = "ctx_test_#{System.unique_integer([:positive])}"
    {:ok, _} = Session.start_session(session_id)

    on_exit(fn ->
      try do
        Session.stop(session_id)
      catch
        :exit, _ -> :ok
      end
    end)

    %{session_id: session_id}
  end

  describe "build_context/2 with :recent strategy" do
    test "returns recent messages", %{session_id: id} do
      for i <- 1..5, do: Session.add_message(id, :user, "msg #{i}")

      {:ok, ctx} = Context.build_context(id, strategy: :recent, recent_count: 3)
      assert length(ctx) == 3
      contents = Enum.map(ctx, & &1.content)
      assert "msg 5" in contents
    end
  end

  describe "build_context/2 with :relevant strategy" do
    test "injects matching memories", %{session_id: id} do
      Session.add_message(id, :user, "Tell me about Elixir")

      OptimalEngine.Memory.store("knowledge", "elixir-101", "Elixir runs on BEAM", tags: ["elixir"])

      {:ok, ctx} =
        Context.build_context(id,
          strategy: :relevant,
          collections: ["knowledge"],
          query: "elixir",
          recent_count: 10
        )

      assert Enum.any?(ctx, &String.contains?(&1.content, "BEAM"))
    end
  end

  describe "build_context/2 with :summary strategy" do
    test "summarizes old messages", %{session_id: id} do
      Session.add_message(id, :system, "System prompt")
      for i <- 1..30, do: Session.add_message(id, :user, "Message #{i}")

      {:ok, ctx} = Context.build_context(id, strategy: :summary, recent_count: 5)

      # Should have summary + 5 recent
      assert length(ctx) <= 7
      assert Enum.any?(ctx, &String.contains?(&1.content, "[context summary]"))
    end

    test "returns all if under recent_count", %{session_id: id} do
      Session.add_message(id, :user, "short")
      {:ok, ctx} = Context.build_context(id, strategy: :summary, recent_count: 50)
      assert length(ctx) == 1
    end
  end

  describe "inject/2" do
    test "prepends memory entries to messages" do
      OptimalEngine.Memory.store("tips", "t1", "Use GenServer", tags: ["elixir"])

      messages = [%{role: :user, content: "How do I manage state?"}]

      injected =
        Context.inject(messages,
          collections: ["tips"],
          query: "elixir",
          max_injections: 5
        )

      assert length(injected) == 2
      assert hd(injected).role == :system
      assert String.contains?(hd(injected).content, "GenServer")
    end

    test "returns unchanged when no query" do
      messages = [%{role: :user, content: "hi"}]
      assert Context.inject(messages, collections: ["x"]) == messages
    end

    test "returns unchanged when no collections" do
      messages = [%{role: :user, content: "hi"}]
      assert Context.inject(messages, query: "test") == messages
    end
  end

  describe "estimate_tokens/1" do
    test "estimates tokens from string" do
      text = String.duplicate("x", 400)
      assert Context.estimate_tokens(text) == 100
    end

    test "estimates tokens from message list" do
      msgs = [
        %{role: :user, content: String.duplicate("x", 400)},
        %{role: :assistant, content: String.duplicate("y", 400)}
      ]

      assert Context.estimate_tokens(msgs) == 200
    end

    test "returns 0 for empty" do
      assert Context.estimate_tokens("") == 0
      assert Context.estimate_tokens([]) == 0
    end
  end

  describe "compact/2" do
    test "delegates to Compactor" do
      msgs = for i <- 1..50, do: %{role: :user, content: "Message number #{i} with content"}
      compacted = Context.compact(msgs, max_tokens: 30, keep_recent: 3)
      assert length(compacted) < length(msgs)
    end
  end
end
