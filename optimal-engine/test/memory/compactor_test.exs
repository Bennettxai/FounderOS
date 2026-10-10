defmodule OptimalEngine.Memory.CompactorTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Memory.Compactor

  defp make_msg(role, content), do: %{role: role, content: content}

  describe "compact/2" do
    test "returns messages unchanged if under budget" do
      msgs = [make_msg(:user, "hello"), make_msg(:assistant, "hi")]
      assert Compactor.compact(msgs, max_tokens: 100_000) == msgs
    end

    test "compacts when over budget" do
      # Each message ~25 chars = ~6 tokens; 50 messages = ~300 tokens
      msgs = for i <- 1..50, do: make_msg(:user, "This is message number #{i}")

      compacted = Compactor.compact(msgs, max_tokens: 50, keep_recent: 5)
      assert length(compacted) < length(msgs)

      # Last 5 should be preserved verbatim
      last_5 = Enum.take(compacted, -5)
      original_last_5 = Enum.take(msgs, -5)
      assert last_5 == original_last_5
    end

    test "preserves system messages during compaction" do
      msgs =
        [make_msg(:system, "You are helpful")] ++
          for i <- 1..30, do: make_msg(:user, "Message #{i}")

      compacted = Compactor.compact(msgs, max_tokens: 50, keep_recent: 5, keep_system: true)

      system_msgs = Enum.filter(compacted, &(&1.role == :system))
      assert length(system_msgs) >= 1
      assert Enum.any?(system_msgs, &(&1.content == "You are helpful"))
    end
  end

  describe "summarize_messages/1" do
    test "returns empty string for empty list" do
      assert Compactor.summarize_messages([]) == ""
    end

    test "summarizes messages into condensed form" do
      msgs = [
        make_msg(:user, "What is Elixir?"),
        make_msg(:assistant, "A functional language on the BEAM VM")
      ]

      summary = Compactor.summarize_messages(msgs)
      assert is_binary(summary)
      assert String.contains?(summary, "[user]")
      assert String.contains?(summary, "[assistant]")
    end

    test "truncates long messages" do
      long = String.duplicate("x", 500)
      msgs = [make_msg(:user, long)]
      summary = Compactor.summarize_messages(msgs)
      assert String.length(summary) < 500
    end
  end

  describe "prune_old/2" do
    test "keeps all if under threshold" do
      msgs = [make_msg(:user, "a"), make_msg(:user, "b")]
      assert Compactor.prune_old(msgs, keep_recent: 10) == msgs
    end

    test "prunes old messages, keeps recent" do
      msgs = for i <- 1..20, do: make_msg(:user, "msg #{i}")
      pruned = Compactor.prune_old(msgs, keep_recent: 5)
      assert length(pruned) == 5
      contents = Enum.map(pruned, & &1.content)
      assert "msg 20" in contents
      refute "msg 1" in contents
    end

    test "preserves system messages from old section" do
      msgs =
        [make_msg(:system, "system prompt")] ++
          for i <- 1..20, do: make_msg(:user, "msg #{i}")

      pruned = Compactor.prune_old(msgs, keep_recent: 5, keep_system: true)
      assert Enum.any?(pruned, &(&1.content == "system prompt"))
    end

    test "discards system when keep_system is false" do
      msgs =
        [make_msg(:system, "system prompt")] ++
          for i <- 1..20, do: make_msg(:user, "msg #{i}")

      pruned = Compactor.prune_old(msgs, keep_recent: 5, keep_system: false)
      refute Enum.any?(pruned, &(&1.content == "system prompt"))
      assert length(pruned) == 5
    end
  end

  describe "estimate_savings/2" do
    test "returns zero savings when under budget" do
      msgs = [make_msg(:user, "hi")]
      {before, after_t, savings} = Compactor.estimate_savings(msgs, max_tokens: 100_000)
      assert before == after_t
      assert savings == 0.0
    end

    test "returns positive savings when compaction occurs" do
      long_content = String.duplicate("This is a verbose message with lots of content. ", 20)
      msgs = for i <- 1..100, do: make_msg(:user, "#{long_content} Message #{i}")
      {before, after_t, savings} = Compactor.estimate_savings(msgs, max_tokens: 50, keep_recent: 5)
      assert after_t < before
      assert savings > 0.0
    end
  end

  describe "check_threshold/2" do
    test "returns :ok when under warn threshold" do
      assert Compactor.check_threshold(50_000, 100_000) == :ok
    end

    test "returns :warn at 85%" do
      assert Compactor.check_threshold(86_000, 100_000) == :warn
    end

    test "returns :compact at 90%" do
      assert Compactor.check_threshold(91_000, 100_000) == :compact
    end

    test "returns :hard_stop at 95%" do
      assert Compactor.check_threshold(96_000, 100_000) == :hard_stop
    end
  end
end
